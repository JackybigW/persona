// node web/e2e/extension-narrow-layout.mjs <absolute-output-dir> [--source <checkout>] [--observe]
// Runs real ViewSections/CSS on loopback with synthetic data only; --observe records historical failures without failing.
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";
import { chromium } from "playwright-core";

const here = dirname(fileURLToPath(import.meta.url));
const output = process.argv[2];
const option = (name) => { const at = process.argv.indexOf(name); return at < 0 ? undefined : process.argv[at + 1]; };
const web = resolve(option("--source") ?? resolve(here, "../.."), "web");
assert.ok(output && isAbsolute(output) && !resolve(output).startsWith(resolve(web, "..") + "/"), "Artifacts must be outside the checkout");
const observe = process.argv.includes("--observe");
let server, browser;
const results = [], errors = [];
const deadline = setTimeout(() => {
  console.error("EXTENSION_NARROW_LAYOUT TIMEOUT");
  void cleanup().finally(() => process.exit(124));
}, 180_000);
async function cleanup() {
  await browser?.close(); browser = undefined;
  await server?.close(); server = undefined;
}
try {
  await mkdir(output, { recursive: true });
  server = await createServer({ root: web, server: { host: "127.0.0.1", port: 0, strictPort: true, fs: { allow: [web, here] } },
    plugins: [{ name: "extension-narrow-layout-fixture", configureServer(vite) {
      vite.middlewares.use(async (req, res, next) => {
        if (req.url?.split("?")[0] !== "/__extension_narrow_layout.html") return next();
        const html = await vite.transformIndexHtml(req.url, `<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width, initial-scale=1"></head><body><div id="root"></div><script type="module" src="/@fs/${join(here, "fixtures/extension-narrow-layout.tsx")}"></script></body></html>`);
        res.setHeader("Content-Type", "text/html"); res.end(html);
      });
    } }] });
  await server.listen();
  const origin = `http://127.0.0.1:${server.httpServer.address().port}`;
  browser = await chromium.launch({ ...(process.env.PERSONA_E2E_CHROMIUM ? { executablePath: process.env.PERSONA_E2E_CHROMIUM } : { channel: "chrome" }), headless: true, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
  for (const width of [390, 820]) for (const theme of ["light", "dark"]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, colorScheme: theme, locale: "en-US", reducedMotion: "reduce", serviceWorkers: "block" });
    await context.addInitScript((theme) => {
      localStorage.setItem("persona.language", "en");
      document.documentElement.classList.toggle("dark", theme === "dark");
    }, theme);
    await context.route("**/*", (route) => new URL(route.request().url()).origin === origin && !new URL(route.request().url()).pathname.startsWith("/api/") ? route.continue() : route.abort());
    const page = await context.newPage();
    page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(`${origin}/__extension_narrow_layout.html`);
    await page.locator("table tbody tr").last().waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.waitForTimeout(400); // Let the existing presence/section entry animation finish before measuring.
    const geometry = await page.evaluate(() => {
      const box = (el) => { const r = el.getBoundingClientRect(); return { x: r.x, right: r.right, width: r.width }; };
      const rows = [...document.querySelectorAll("section:first-child button")].filter((el) => el.textContent.includes("Stable") || el.textContent.includes("Ready") || el.textContent.includes("Synthetic long"));
      const rowGeometry = rows.map((row) => {
        const trailing = [...row.children].slice(-2);
        const rb = box(row), parent = box(row.parentElement);
        return { row: rb, parent, trailing: trailing.map(box), contained: rb.x >= parent.x - 1 && rb.right <= parent.right + 1 && trailing.every((el) => box(el).right <= parent.right + 1 && box(el).width > 0) };
      });
      const table = document.querySelector("table");
      // Older revisions scroll the table itself. The fixed renderer assigns scrolling to a separate wrapper.
      const scroll = table.closest(".extension-md-table") ?? table;
      const before = scroll.scrollLeft;
      scroll.scrollLeft = scroll.scrollWidth;
      const after = scroll.scrollLeft;
      return { viewport: innerWidth, documentWidth: document.documentElement.scrollWidth, rows: rowGeometry,
        table: { box: box(scroll), clientWidth: scroll.clientWidth, scrollWidth: scroll.scrollWidth, before, after, overflowX: getComputedStyle(scroll).overflowX, ownContainer: scroll !== table,
          scrolls: after > before && scroll.scrollWidth > scroll.clientWidth, lastCell: box(table.querySelector("tbody tr:last-child td:last-child")) } };
    });
    assert.ok(geometry.rows.length >= 3, "Fixture rows rendered");
    assert.equal(geometry.viewport, width);
    const pass = geometry.documentWidth <= width && geometry.rows.every((row) => row.contained) && geometry.table.scrolls && geometry.table.box.right <= width && geometry.table.lastCell.right <= geometry.table.box.right + 1;
    results.push({ width, theme, pass, ...geometry });
    await page.locator("table").evaluate((table) => { (table.closest(".extension-md-table") ?? table).scrollLeft = 0; });
    await page.screenshot({ path: join(output, `${width}-${theme}.png`), fullPage: true });
    await page.locator("table").evaluate((table) => { const scroll = table.closest(".extension-md-table") ?? table; scroll.scrollLeft = scroll.scrollWidth; });
    await page.screenshot({ path: join(output, `${width}-${theme}-scrolled.png`), fullPage: true });
    await context.close();
  }
  assert.deepEqual(errors, [], "No browser page errors");
  if (!observe) assert.ok(results.every((result) => result.pass), JSON.stringify(results));
} finally {
  clearTimeout(deadline);
  await cleanup();
  await writeFile(join(output, "results.json"), JSON.stringify({ observe, results, errors, browserClosed: !browser, serverClosed: !server }, null, 2));
}
console.log(`EXTENSION_NARROW_LAYOUT ${results.every((result) => result.pass) ? "PASS" : "OBSERVED_FAILURE"} (${results.length} cases); browser and server closed`);
