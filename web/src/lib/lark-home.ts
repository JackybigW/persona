// Settings › Connectors › Lark › "Show messages on home": whether this browser shows the "Messages" page (the Feishu chat
// list, lark-chats.ts), off by default and kept per browser. Its own file because the sidebar reads it on every first
// screen, and importing it from lark-chats.ts put that list's store and its words there too.
import { useSyncExternalStore } from "react";

const HOME_KEY = "lark.homeMessages";
const homeListeners = new Set<() => void>();

export function getLarkHome(): boolean {
  try {
    return localStorage.getItem(HOME_KEY) === "1";
  } catch {
    return false;
  }
}

export function setLarkHome(on: boolean) {
  if (on) localStorage.setItem(HOME_KEY, "1");
  else localStorage.removeItem(HOME_KEY);
  homeListeners.forEach((l) => l());
}

if (typeof window !== "undefined") {
  window.addEventListener("storage", (e) => {
    if (e.key === HOME_KEY || e.key === null) homeListeners.forEach((l) => l());
  });
}

const subscribeHome = (l: () => void) => {
  homeListeners.add(l);
  return () => {
    homeListeners.delete(l);
  };
};

/** Whether this browser shows the "Messages" page (off unless turned on). */
export function useLarkHome(): boolean {
  return useSyncExternalStore(subscribeHome, getLarkHome, () => false);
}
