// @vitest-environment jsdom
/// <reference types="node" />
// The sidebar's hub line draws nothing unless an update is under way or offered, so its code, lib/hub-update.ts and
// their words are not on the first screen: the slot (HubNoticeRow.tsx) loads them with its first render.
import { readFileSync } from "node:fs";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const hub = vi.hoisted(() => ({ reads: 0 }));
vi.mock("../../lib/hub-update", async (original) => ({
  ...(await original<typeof import("../../lib/hub-update")>()),
  hubUpdate: async () => {
    hub.reads++;
    return { version: "1.0.0", build: 1, channel: "release", can_update: true, busy: [], auto: false, auto_notice: false, confirmed: true,
      state: "idle", latest: { version: "1.1.0", build: 2, newer: true, bad: false } };
  },
}));
vi.hoisted(() => {
  const g = globalThis as Record<string, unknown>;
  g.IS_REACT_ACT_ENVIRONMENT = true;
  g.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} });
});
import startup from "../../i18n/startup-namespaces.json";
import { startupMatcher } from "../../i18n/startup-keys";
import { HubNoticeLine } from "./HubNoticeRow";

const read = (path: string) => readFileSync(new URL(path, import.meta.url), "utf8");
// Lets the chunk and the status read land inside act, a few ticks at a time, until `done`.
async function settle(done: () => boolean) {
  for (let i = 0; i < 100 && !done(); i++) await act(async () => new Promise((r) => setTimeout(r, 5)));
}
let host: HTMLDivElement;
let root: Root;

beforeEach(() => {
  hub.reads = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
});

describe("the sidebar's hub notice slot", () => {
  it("loads the line from its own chunk, with its words", () => {
    const staticImport = (from: string) => new RegExp(`^import[^;]*from\\s+["']${from}["']`, "m");
    const slot = read("./HubNoticeRow.tsx");
    expect(slot).not.toMatch(staticImport("\\./HubNoticeBody"));
    expect(slot).not.toMatch(staticImport("[./]*lib/hub-update"));
    expect(slot).toMatch(/lazyPage\(\(\) => import\("\.\/HubNoticeBody"\)\.then\(\(m\) => m\.HubNoticeLine\)\)/);
    const sidebar = read("../Sidebar.tsx");
    expect(sidebar).not.toMatch(staticImport("\\./agents/HubNoticeBody"));
    expect(sidebar).not.toMatch(staticImport("[./]*lib/hub-update"));
    const inStartup = startupMatcher(startup);
    const words = [...new Set([read("./HubNoticeBody.tsx"), read("../../lib/hub-update.ts")].join("").match(/lib-hub-update\.[a-z_]+/g) ?? [])];
    expect(words.length).toBeGreaterThan(10);
    for (const k of words) expect(inStartup(k), k).toBe(false);
  });

  it("reads the hub's update status once its chunk is in and offers the update", async () => {
    await act(async () => root.render(<HubNoticeLine />));
    await settle(() => host.querySelector('[data-testid="hub-notice-update"]') !== null);
    expect(host.querySelector('[data-testid="hub-notice"]')?.getAttribute("data-kind")).toBe("available");
    expect(host.querySelector('[data-testid="hub-notice-update"]')?.textContent).toBeTruthy();
    expect(hub.reads).toBe(1);
  });
});
