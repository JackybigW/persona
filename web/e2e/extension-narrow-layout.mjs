// Real /extension route, App shell, ExtensionNavPage, ExtensionUI and synthetic HTTP/WS.
// node web/e2e/extension-narrow-layout.mjs <absolute-output-dir> [--source <checkout>] [--observe] [--compare-ref <sha>]
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { access, mkdir, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";
import { chromium } from "playwright-core";
import { extension, ui, listState, detailState, rowTitle, sectionTitle, place, listView, detailView } from "./fixtures/extension-narrow-layout.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const productionPaths = ["web/src/views/extensions/ViewBody.tsx", "web/src/styles.css"];

// Only this PR's two production files differ from its merge base. The browser still runs the real app and route.
// A comparison refuses unrelated source changes rather than calling a partial overlay an exact baseline.
function baselineOverlay(root, ref) {
  const sha = execFileSync("git", ["rev-parse", "--verify", `${ref}^{commit}`], { cwd: root, encoding: "utf8" }).trim();
  const changed = execFileSync("git", ["diff", "--name-only", `${sha}...HEAD`], { cwd: root, encoding: "utf8" }).trim().split("\n").filter(Boolean);
  const other = changed.filter((p) => !productionPaths.includes(p) && !p.startsWith("web/e2e/") && !/\.test\.[cm]?[jt]sx?$/.test(p));
  assert.deepEqual(other, [], "Exact baseline overlay must include every changed production file");
  const files = new Map(productionPaths.map((path) => [resolve(root, path), execFileSync("git", ["show", `${sha}:${path}`], { cwd: root, encoding: "utf8" })]));
  return { sha, plugin: { name: "narrow-layout-exact-baseline", enforce: "pre", load(id) { return files.get(id.split("?")[0]); } } };
}

export async function verifyNarrowLayout({ sourceRoot = resolve(here, "../.."), output, compareRef, observe = false, screenshots = true, widths = [390, 820], themes = ["light", "dark"] } = {}) {
  const root = resolve(sourceRoot), web = join(root, "web");
  if (output) assert.ok(isAbsolute(output) && !resolve(output).startsWith(root + "/"), "Artifacts must be outside the checkout");
  if (screenshots) assert.ok(output, "Screenshots require an external artifact directory");
  const overlay = compareRef ? baselineOverlay(root, compareRef) : undefined;
  const report = { route: `/extension/${extension}/${ui}`, sourceRoot: root, baselineRef: overlay?.sha, productionOverlay: overlay ? productionPaths : [], results: [], errors: [], apiRequests: [], browserClosed: false, serverClosed: false };
  let server, browser;
  try {
    if (output) await mkdir(output, { recursive: true });
    server = await createServer({ root: web, plugins: overlay ? [overlay.plugin] : [], logLevel: "error", server: { host: "127.0.0.1", port: 0, strictPort: true, watch: null, hmr: false } });
    await server.listen();
    const origin = `http://127.0.0.1:${server.httpServer.address().port}`;
    const candidates = [process.env.PERSONA_E2E_CHROMIUM, process.env.CHROME_BIN, "/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/chromium", chromium.executablePath()].filter(Boolean);
    let executablePath;
    for (const path of candidates) { try { await access(path); executablePath = path; break; } catch { /* Try another installed browser. */ } }
    assert.ok(executablePath, "An installed Chromium is required on the Linux verifier");
    browser = await chromium.launch({ executablePath, headless: true, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
    for (const width of widths) for (const theme of themes) {
      const context = await browser.newContext({ viewport: { width, height: 844 }, deviceScaleFactor: 2, isMobile: width === 390, hasTouch: width === 390, colorScheme: theme, locale: "zh-CN", reducedMotion: "reduce", serviceWorkers: "block" });
      await context.addInitScript((theme) => {
        localStorage.setItem("persona.token", "synthetic-layout-token");
        localStorage.setItem("persona.language", "zh-Hans"); localStorage.setItem("persona.theme", theme);
      }, theme);
      await context.route("**/*", async (route) => {
        const url = new URL(route.request().url());
        if (url.origin !== origin) return route.abort();
        if (!url.pathname.startsWith("/api/")) return route.continue();
        const path = url.pathname.replace(/^\/api\/v1/, "");
        let body = { machines: [], sessions: [], agents: [], groups: [], items: [], interactions: [], todos: [], workflows: [], extensions: [], conversations: [], providers: [], channels: [], personas: [], count: 0, total: 0 };
        if (path === "/managers") body = { default: "assistant", managers: [{ id: "assistant", name: "Synthetic manager", online: true, builtin: true }] };
        else if (/^\/(extensions|plugins)\/-\/ui$/.test(path)) body = { ui: [place] };
        else if (new RegExp(`^/(extensions|plugins)/${extension}/-/ui/${ui}$`).test(path)) {
          const request = route.request().postDataJSON();
          report.apiRequests.push({ width, theme, path, state: request.state, action: request.action });
          assert.ok(!request.action, "Push must navigate without executing an extension action");
          body = { view: request.state === detailState ? detailView : listView };
        }
        await route.fulfill({ contentType: "application/json", body: JSON.stringify(body) });
      });
      await context.routeWebSocket("**/api/v1/ws**", (ws) => {
        const hello = () => ws.send(JSON.stringify({ type: "hello", server_version: "synthetic-layout", hub_name: "Synthetic hub", features: ["extensions", "managers", "extension_ui_slots"], resumed: false }));
        hello();
        ws.onMessage((raw) => { const message = JSON.parse(String(raw)); if (message.type === "client_hello") hello(); if (message.type === "app_ping") ws.send(JSON.stringify({ type: "pong" })); });
      });
      const page = await context.newPage();
      page.setDefaultTimeout(25_000);
      page.on("pageerror", (error) => report.errors.push(error.message));
      await page.goto(`${origin}${report.route}?state=${encodeURIComponent(listState)}`);
      await page.locator("[data-app-shell]").waitFor();
      const section = page.locator("section").filter({ has: page.getByRole("heading", { name: sectionTitle, exact: true }) });
      const rows = section.locator("button");
      await rows.first().waitFor();
      assert.equal(await rows.count(), 2, "Titled Chinese section renders through the real extension route");
      await page.evaluate(async () => { await document.fonts.ready; await Promise.all(document.getAnimations().filter((a) => Number.isFinite(a.effect?.getComputedTiming().iterations)).map((a) => a.finished.catch(() => {}))); });
      const rowGeometry = await rows.evaluateAll((rows) => {
        const box = (el) => { const r = el.getBoundingClientRect(); return { left: r.left, right: r.right, width: r.width }; };
        const visible = (el) => {
          let left = 0, right = innerWidth;
          for (let parent = el.parentElement; parent; parent = parent.parentElement) {
            if (/(hidden|clip|auto|scroll)/.test(getComputedStyle(parent).overflowX)) { const r = parent.getBoundingClientRect(); left = Math.max(left, r.left); right = Math.min(right, r.right); }
          }
          const r = box(el); return { ...r, clipLeft: left, clipRight: right, visible: r.left >= left - 1 && r.right <= right + 1 && r.width > 0 };
        };
        return rows.map((row) => {
          const title = row.querySelector(".flex-1").firstElementChild;
          const badge = row.children[row.children.length - 2], arrow = row.lastElementChild;
          const style = getComputedStyle(title);
          return { row: box(row), title: { text: title.textContent, ...box(title), clientWidth: title.clientWidth, scrollWidth: title.scrollWidth, whiteSpace: style.whiteSpace, overflow: style.overflow, textOverflow: style.textOverflow }, badge: visible(badge), arrow: visible(arrow) };
        });
      });
      const documentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
      if (screenshots) await page.screenshot({ path: join(output, `${width}-${theme}-rows.png`), fullPage: true });
      // #3535 changed exactly this class in Row: removing it restores the pre-fix constraint, not the merge base.
      const control = await rows.first().evaluate((row) => {
        const hadWidth = row.classList.contains("w-full"); row.classList.remove("w-full");
        const title = row.querySelector(".flex-1").firstElementChild;
        const text = title.getBoundingClientRect();
        const arrow = row.lastElementChild.getBoundingClientRect();
        const card = row.closest("section").getBoundingClientRect();
        const clipped = arrow.right > Math.min(card.right, innerWidth);
        if (hadWidth) row.classList.add("w-full");
        return { label: "pre-3535 row class control, not merge base", sourceFix: "c6fabd7cdc4884f01d362741c2c87fc28644bfe7", removedClass: "w-full", titleWidth: text.width, titleScrollWidth: title.scrollWidth, arrowRight: arrow.right, clipRight: Math.min(card.right, innerWidth), clipped };
      });
      await rows.first().click();
      await page.getByRole("heading", { name: detailView.title, exact: true }).waitFor();
      await page.locator("table").last().waitFor();
      const tableGeometry = await page.locator("table").evaluateAll((tables) => tables.map((table) => {
        const scroll = table.closest(".extension-md-table") ?? table;
        scroll.scrollLeft = 0;
        const start = table.querySelector("tbody td:last-child").getBoundingClientRect();
        const bounds = scroll.getBoundingClientRect();
        const overflow = scroll.scrollWidth > scroll.clientWidth + 1;
        scroll.scrollLeft = scroll.scrollWidth;
        const end = table.querySelector("tbody td:last-child").getBoundingClientRect();
        return { ownContainer: scroll !== table, clientWidth: scroll.clientWidth, scrollWidth: scroll.scrollWidth, scrollLeft: scroll.scrollLeft, overflowX: getComputedStyle(scroll).overflowX, left: bounds.left, right: bounds.right, lastCellBefore: { left: start.left, right: start.right }, lastCellAfter: { left: end.left, right: end.right }, accessible: bounds.right <= innerWidth + 1 && (!overflow || scroll.scrollLeft > 0) && end.right <= bounds.right + 1 };
      }));
      const detailDocumentWidth = await page.evaluate(() => document.documentElement.scrollWidth);
      if (screenshots) {
        await page.screenshot({ path: join(output, `${width}-${theme}-tables-scrolled.png`), fullPage: true });
        await page.locator("table").evaluateAll((tables) => tables.forEach((table) => { (table.closest(".extension-md-table") ?? table).scrollLeft = 0; }));
        await page.screenshot({ path: join(output, `${width}-${theme}-tables.png`), fullPage: true });
      }
      assert.equal(tableGeometry.length, 2, "Exact four-column sample and a separate four-column stress case");
      const pass = documentWidth <= width && detailDocumentWidth <= width && rowGeometry.every((r) => r.badge.visible && r.arrow.visible) && tableGeometry.every((t) => t.accessible);
      report.results.push({ width, theme, deviceScaleFactor: 2, isMobile: width === 390, pass, documentWidth, detailDocumentWidth, rowGeometry, tableGeometry, historicalClassControl: control });
      assert.ok(rowGeometry[0].title.text.includes(rowTitle), "Unbroken Chinese fixture is intact");
      assert.ok(control.clipped, "Geometry check must detect a clipped trailing arrow");
      await context.close();
    }
    assert.deepEqual(report.errors, [], "No browser page errors");
    if (!observe) assert.ok(report.results.every((result) => result.pass), JSON.stringify(report.results));
    return report;
  } finally {
    // Release both owned resources even if one close rejects; preserve failures instead of claiming cleanup succeeded.
    const closed = await Promise.allSettled([browser?.close(), server?.close()]);
    report.browserClosed = closed[0].status === "fulfilled";
    report.serverClosed = closed[1].status === "fulfilled";
    if (output) await writeFile(join(output, "results.json"), JSON.stringify(report, null, 2));
    const rejected = closed.find((result) => result.status === "rejected");
    if (rejected) throw rejected.reason;
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const option = (name) => { const at = process.argv.indexOf(name); return at < 0 ? undefined : process.argv[at + 1]; };
  const deadline = setTimeout(() => { console.error("EXTENSION_NARROW_LAYOUT TIMEOUT"); process.exit(124); }, 180_000);
  try {
    const report = await verifyNarrowLayout({ output: process.argv[2], sourceRoot: option("--source"), compareRef: option("--compare-ref"), observe: process.argv.includes("--observe") });
    console.log(`EXTENSION_NARROW_LAYOUT ${report.results.every((r) => r.pass) ? "PASS" : "OBSERVED_FAILURE"}; ${JSON.stringify(report)}`);
  } finally { clearTimeout(deadline); }
}
