package archguard

import (
	"bufio"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Where a panic may be recovered (docs/server/architecture.md "Validated and sealed"). A programmer error is a
// defect (domainkit/defect): it stops the operation, and only a listed site recovers it, through defect.Catch. Both
// lists are permission lists: a production file that is not listed fails, and so does a line whose file no longer
// does what the line permits, since a stale line would be a standing permission.
//
//   - testdata/recover_sites.txt: every production .go file under server/ that calls the builtin recover(), with its
//     reason. defect.Catch's own file is one of them.
//   - testdata/catch_sites.txt: every production .go file that refers to defect.Catch, which recovers only when it
//     is deferred, with its reason.
//
// Tests (_test.go) and testdata, vendor, node_modules, hidden and "_" directories are not read. The checks read syntax:
// a call of recover that the file does not declare itself, and a selector Catch on the name the file imports
// domainkit/defect under.

const (
	defectSitesDoc = `see docs/server/architecture.md "Validated and sealed"`
	defectImport   = "github.com/nwverse/persona/server/internal/domainkit/defect"
)

func TestRecoverOnlyAtListedSites(t *testing.T) {
	found := scanSites(t, serverDir, []byte("recover"), callsRecover)
	for _, p := range judgeSites("recover_sites.txt", found, readSites(t, "recover_sites.txt"),
		"calls recover(): raise a defect (defect.Raise) and let a site in testdata/catch_sites.txt catch it, or return "+
			"an error; a recover() that is truly needed gets a line in testdata/recover_sites.txt with its reason") {
		t.Error(p)
	}
}

func TestCatchOnlyAtListedSites(t *testing.T) {
	found := scanSites(t, serverDir, []byte("domainkit/defect"), refersToCatch)
	for _, p := range judgeSites("catch_sites.txt", found, readSites(t, "catch_sites.txt"),
		"refers to defect.Catch: a defect is recovered only at the listed sites, the kernel's dispatch.go for a "+
			"module's routes; register through the kernel (kernel/seam), or add a line to testdata/catch_sites.txt "+
			"with the reason this is a boundary of its own") {
		t.Error(p)
	}
}

// readSites reads a site list: "<file under server/> <reason>" per line, "#" starting a comment line.
func readSites(t *testing.T, name string) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sites := map[string]string{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		file, reason, _ := strings.Cut(line, " ")
		switch reason = strings.TrimSpace(reason); {
		case reason == "":
			t.Errorf("testdata/%s:%d: %s has no reason: write why this file is a site after its path", name, n, file)
		case sites[file] != "":
			t.Errorf("testdata/%s:%d: %s is listed twice", name, n, file)
		}
		sites[file] = reason
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return sites
}

// judgeSites is what is wrong between the files found and a list's lines: a file found but not listed, and a line
// whose file was not found.
func judgeSites(list string, found []string, listed map[string]string, does string) []string {
	var problems []string
	for _, f := range found {
		if _, ok := listed[f]; !ok {
			problems = append(problems, "server/"+f+" "+does+"; "+defectSitesDoc)
		}
	}
	var stale []string
	for f := range listed {
		if !slices.Contains(found, f) {
			stale = append(stale, f)
		}
	}
	slices.Sort(stale)
	for _, f := range stale {
		problems = append(problems, "testdata/"+list+" lists "+f+", which no longer does: delete its line; "+defectSitesDoc)
	}
	return problems
}

// scanSites is the production .go files under root (paths relative to it, sorted) for which match holds. Only files
// whose bytes contain prefilter are parsed.
func scanSites(t *testing.T, root string, prefilter []byte, match func(*ast.File) bool) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (name == "testdata" || name == "vendor" || name == "node_modules" ||
				strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil || !bytes.Contains(src, prefilter) {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return err
		}
		if match(f) {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(found)
	return found
}

// callsRecover says whether f calls the builtin recover: a call of the name recover that f does not declare itself
// (a method call such as m.recover() is a selector, not this name).
func callsRecover(f *ast.File) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "recover" && id.Obj == nil {
				found = true
			}
		}
		return !found
	})
	return found
}

// refersToCatch says whether f refers to defect.Catch, under whatever name f imports domainkit/defect.
func refersToCatch(f *ast.File) bool {
	name := ""
	for _, imp := range f.Imports {
		if p, err := strconv.Unquote(imp.Path.Value); err == nil && p == defectImport {
			name = "defect"
			if imp.Name != nil {
				name = imp.Name.Name
			}
		}
	}
	if name == "" || name == "_" {
		return false
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "Catch" {
			if id, ok := s.X.(*ast.Ident); ok && id.Name == name && id.Obj == nil {
				found = true
			}
		}
		if id, ok := n.(*ast.Ident); ok && name == "." && id.Name == "Catch" && id.Obj == nil {
			found = true
		}
		return !found
	})
	return found
}

// The site checks on planted sources: what counts as a site, and that an unlisted site and a stale line each fail.
func TestDefectSiteChecks(t *testing.T) {
	parse := func(src string) *ast.File {
		t.Helper()
		f, err := parser.ParseFile(token.NewFileSet(), "planted.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	for _, c := range []struct {
		src            string
		recover, catch bool
	}{
		{"package p\nfunc f() { defer func() { _ = recover() }() }\n", true, false},
		{"package p\nfunc f() { if recover() != nil {} }\n", true, false},
		{"package p\ntype m struct{}\nfunc (m) recover() error { return nil }\nfunc f(x m) { _ = x.recover() }\n", false, false},
		{"package p\nfunc f() { recover := func() any { return nil }; _ = recover() }\n", false, false},
		{"package p\nimport \"" + defectImport + "\"\nfunc f() { defer defect.Catch(nil) }\n", false, true},
		{"package p\nimport d \"" + defectImport + "\"\nfunc f() { catch := d.Catch; defer catch(nil) }\n", false, true},
		{"package p\nimport . \"" + defectImport + "\"\nfunc f() { defer Catch(nil) }\n", false, true},
		{"package p\nimport \"" + defectImport + "\"\nfunc f() { defect.Raise(\"op\", \"cause\") }\n", false, false},
		{"package p\nimport \"other/defect\"\nfunc f() { defer defect.Catch(nil) }\n", false, false},
	} {
		f := parse(c.src)
		if got := callsRecover(f); got != c.recover {
			t.Errorf("callsRecover = %v, want %v:\n%s", got, c.recover, c.src)
		}
		if got := refersToCatch(f); got != c.catch {
			t.Errorf("refersToCatch = %v, want %v:\n%s", got, c.catch, c.src)
		}
	}
	got := judgeSites("catch_sites.txt", []string{"internal/a/a.go", "internal/b/b.go"},
		map[string]string{"internal/a/a.go": "a reason", "internal/c/c.go": "gone"}, "refers to defect.Catch")
	want := []string{
		"server/internal/b/b.go refers to defect.Catch; " + defectSitesDoc,
		"testdata/catch_sites.txt lists internal/c/c.go, which no longer does: delete its line; " + defectSitesDoc,
	}
	if !slices.Equal(got, want) {
		t.Errorf("judgeSites:\n got %q\nwant %q", got, want)
	}
}
