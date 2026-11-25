// Synthetic browser fixture only. Imports use the target Vite root so the same fixture can inspect older revisions.
import { createElement } from "react";
import { createRoot } from "react-dom/client";
import { initI18n } from "/src/i18n";
import "/src/styles.css";

await initI18n();
const [{ ViewSections, INERT }, { readView }] = await Promise.all([
  import("/src/views/extensions/ViewBody"), import("/src/lib/extension-ui"),
]);
const title = "Synthetic long row title with sample content ".repeat(10);
const subtitle = "Synthetic detail with several sources and a sample date ".repeat(6);
const view = readView({ sections: [
  { title: "Long rows", items: [
    { type: "row", key: "long", title, subtitle, badge: { text: "Stable", tone: "positive" }, action: { id: "open", push: { state: "sample-detail" } } },
    { type: "row", key: "short", title: "Short sample row", badge: { text: "Ready", tone: "positive" }, action: { id: "short" } },
    { type: "row", key: "control", title, subtitle, badge: { text: "On", tone: "positive" }, action: { id: "control" }, control: { kind: "switch", name: "sample", on: true, action: { id: "toggle" } } },
    { type: "row", key: "static", title, subtitle, badge: { text: "Info" } },
  ] },
  { title: "Wide Markdown table", items: [{ type: "markdown", text:
    "| Sample category | Active sample items | Pending sample items | Other sample items | Final sample column |\n| --- | ---: | ---: | --- | --- |\n| Example alpha | 12 | 345 | Synthetic information | Rightmost sample value |\n| Example beta | 67 | 890 | Additional sample text | Final visible value |\n\nA normal paragraph below the table." }] },
] });
createRoot(document.getElementById("root")!).render(createElement("main", { className: "min-h-dvh bg-page p-4 text-ink" },
  createElement("div", { className: "flex min-w-0 flex-col gap-4" }, createElement(ViewSections, { view, ...INERT }))));
