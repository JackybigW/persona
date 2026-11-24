// The sidebar's slot for the one quiet line about the hub (#2897, the app's HubNoticeRow). Most of the time the line draws
// nothing (no update under way, none offered), so its code (HubNoticeBody.tsx), lib/hub-update.ts and their words are
// off the first screen and load with the slot's first render. Drawn only once the chunk is in, never suspending: a
// Suspense here would hold the sidebar's other updates (a navigation) while it loads. A chunk that will not load is
// reported, and the next mount asks again.
import { memo, useEffect, useState } from "react";
import { reportBoundary } from "../../lib/error-reports";
import { lazyPage } from "../../lib/lazy-page";
import { isChunkError } from "../PageBoundary";

const HubNoticeBody = lazyPage(() => import("./HubNoticeBody").then((m) => m.HubNoticeLine));

export const HubNoticeLine = memo(function HubNoticeLine() {
  const [ready, setReady] = useState(HubNoticeBody.ready);
  useEffect(() => {
    if (ready) return;
    let live = true;
    HubNoticeBody.preload().then(
      () => live && setReady(true),
      (error: unknown) => reportBoundary(error, "hub_notice", isChunkError(error)),
    );
    return () => {
      live = false;
    };
  }, [ready]);
  return ready ? <HubNoticeBody /> : null;
});
