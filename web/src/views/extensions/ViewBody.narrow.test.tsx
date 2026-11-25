// @vitest-environment jsdom
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { readView } from "../../lib/extension-ui";
import { INERT, ViewSections } from "./ViewBody";

vi.hoisted(() => {
  globalThis.matchMedia = vi.fn().mockReturnValue({ matches: false, addEventListener() {}, removeEventListener() {} });
});

function draw(items: unknown[]) {
  const view = readView({ sections: [{ title: "Synthetic titled section", items }] });
  return new DOMParser().parseFromString(renderToStaticMarkup(<ViewSections view={view} {...INERT} />), "text/html");
}

// Browser geometry and actual scrolling are covered by e2e/extension-narrow-layout.mjs.
describe("extension narrow layout", () => {
  it("constrains standalone row actions while retaining their badge and chevron", () => {
    // i18n-ignore: synthetic extension-authored Chinese wire content without spaces.
    const title = "一段超过四十个汉字的条目正文用于复现窄屏行组件横向溢出问题的合成示例文本";
    const subtitle = "具体 · 示例类型 · 14 次 · 4 种来源 · 最近 2026-11-23";
    const doc = draw([{ type: "row", title, subtitle, badge: { text: "Stable", tone: "positive" }, action: { id: "open", push: { state: '{"tab":"facets","open":"e:sample"}' } } }]);
    expect(doc.querySelector("section h3")?.textContent).toBe("Synthetic titled section");
    const row = doc.querySelector("button")!;
    expect(row.classList.contains("w-full")).toBe(true);
    expect(row.classList.contains("min-w-0")).toBe(true);
    const text = row.querySelector(".flex-1")!;
    expect(text.classList.contains("min-w-0")).toBe(true);
    expect(text.children[0].classList.contains("truncate")).toBe(true);
    expect(text.children[0].textContent).toBe(title);
    expect(text.children[1].classList.contains("line-clamp-2")).toBe(true);
    expect(text.children[1].textContent).toBe(subtitle);
    expect([...row.querySelectorAll(".shrink-0")].some((el) => el.textContent === "Stable")).toBe(true);
    expect(row.lastElementChild?.tagName.toLowerCase()).toBe("svg");
    expect(row.lastElementChild?.classList.contains("shrink-0")).toBe(true);
  });

  it("gives each Markdown table its own bounded scroll container and preserves alignment", () => {
    const table = "| Item | Active | Pending | Other |\n| :--- | ---: | :---: | --- |\n| Sample | 12 | 34 | Synthetic note |";
    const doc = draw([{ type: "markdown", text: `${table}\n\nA normal paragraph.\n\n${table}` }]);
    const tables = [...doc.querySelectorAll("table")];
    expect(tables).toHaveLength(2);
    for (const table of tables) {
      const scroll = table.parentElement!;
      expect(scroll.classList.contains("extension-md-table")).toBe(true);
      expect(scroll.classList.contains("max-w-full")).toBe(true);
      expect(scroll.classList.contains("overflow-x-auto")).toBe(true);
      expect(scroll.parentElement?.classList.contains("min-w-0")).toBe(true);
      expect(table.querySelectorAll("th")).toHaveLength(4);
      expect(table.querySelectorAll("td")[1].getAttribute("style")).toContain("text-align:right");
      expect(table.querySelectorAll("td")[2].getAttribute("style")).toContain("text-align:center");
    }
    expect(doc.querySelector("p")?.textContent).toBe("A normal paragraph.");
  });
});
