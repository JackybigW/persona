/// <reference types="node" />
// The sidebar, a first-screen file, shows "Messages" only when this browser turned the Lark switch on. It needs only
// useLarkHome: importing it from lark-chats.ts put the Feishu chat list's store and its words on every first screen.
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import startup from "../i18n/startup-namespaces.json";
import * as chats from "./lark-chats";
import { getLarkHome, setLarkHome, useLarkHome } from "./lark-home";

const read = (file: string) => readFileSync(new URL(file, import.meta.url), "utf8");
const runtimeImport = (from: string) => new RegExp(`^import\\s+(?!type\\b)[^;]*from\\s+["']${from}["']`, "m");

describe("lark-home", () => {
  it("keeps lark-chats.ts and its words off the first screen", () => {
    expect(read("../views/SidebarParts.tsx")).not.toMatch(runtimeImport("\\.\\./lib/lark-chats"));
    expect(read("./lark-home.ts")).not.toMatch(/from\s+["']\.\/lark-chats["']/);
    expect(startup.filter((key) => key === "lib-lark-chats" || key.startsWith("lib-lark-chats."))).toEqual([]);
  });

  it("is the switch lark-chats.ts offers the Messages page and Settings", () => {
    expect(chats.getLarkHome).toBe(getLarkHome);
    expect(chats.setLarkHome).toBe(setLarkHome);
    expect(chats.useLarkHome).toBe(useLarkHome);
  });
});
