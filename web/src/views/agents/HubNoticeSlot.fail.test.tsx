// @vitest-environment jsdom
// A hub notice chunk that will not load (offline, or the hub was redeployed under the page) must not take the sidebar
// down: the slot draws nothing and reports it.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
vi.hoisted(() => {
  const g = globalThis as Record<string, unknown>;
  g.IS_REACT_ACT_ENVIRONMENT = true;
  g.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} });
});
const failed = vi.hoisted(() => ({ loads: 0 }));
vi.mock("./HubNoticeBody", () => {
  failed.loads++;
  throw new Error("Failed to fetch dynamically imported module: /assets/HubNoticeBody-x.js");
});
const reported = vi.hoisted(() => [] as string[]);
vi.mock("../../lib/error-reports", async (load) => ({ ...(await load<typeof import("../../lib/error-reports")>()), reportBoundary: (_e: unknown, where: string) => reported.push(where) }));
import { HubNoticeLine } from "./HubNoticeRow";

it("draws nothing and keeps its siblings when the chunk cannot load", async () => {
  vi.spyOn(console, "error").mockImplementation(() => {});
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  await act(async () => root.render(<div><span data-testid="sibling">Agents</span><HubNoticeLine /></div>));
  for (let i = 0; i < 100 && !reported.length; i++) await act(async () => new Promise((r) => setTimeout(r, 5)));
  expect(reported).toEqual(["hub_notice"]);
  expect(failed.loads).toBeGreaterThan(0);
  expect(host.querySelector('[data-testid="sibling"]')?.textContent).toBe("Agents");
  expect(host.querySelector('[data-testid="hub-notice"]')).toBeNull();
  act(() => root.unmount());
  host.remove();
});
