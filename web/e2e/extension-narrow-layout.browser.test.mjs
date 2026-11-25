// @vitest-environment node
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { expect, it } from "vitest";
import { verifyNarrowLayout } from "./extension-narrow-layout.mjs";

// Linux runs the real route and browser; screenshot capture remains a separate acceptance step.
it.skipIf(process.platform !== "linux")("contains Chinese rows and four-column Markdown through the actual extension route", async () => {
  const root = fileURLToPath(new URL("../../", import.meta.url));
  let base;
  try { base = execFileSync("git", ["merge-base", "HEAD", "origin/main"], { cwd: root, encoding: "utf8" }).trim(); }
  catch { /* A source archive can still run the head regression without a Git baseline. */ }
  if (base && base !== execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim()) {
    const before = await verifyNarrowLayout({ sourceRoot: root, compareRef: base, screenshots: false, observe: true, widths: [390], themes: ["light"] });
    console.log(`NARROW_BASELINE_GEOMETRY=${JSON.stringify(before)}`);
  }
  const after = await verifyNarrowLayout({ sourceRoot: root, screenshots: false, widths: [390], themes: ["light", "dark"] });
  console.log(`NARROW_HEAD_GEOMETRY=${JSON.stringify(after)}`);
  expect(after.results).toHaveLength(2);
  expect(after.results.every((r) => r.pass)).toBe(true);
  expect(after.browserClosed && after.serverClosed).toBe(true);
}, 150_000);
