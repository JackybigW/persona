// Synthetic Chinese data follows the reported wire shape, never a captured user view.
// i18n-ignore: authored extension content, not application UI copy.
export const rowTitle = "一段超过四十个汉字的条目正文用于复现窄屏行组件横向溢出问题的合成示例文本";
export const rowSubtitle = "具体 · 示例类型 · 14 次 · 4 种来源 · 最近 2026-11-23";
export const sectionTitle = "示例手写记忆（2）";
export const documentTable = "| 项目 | 有效 | 候选 / 偶现 | 其他 |\n| --- | --- | --- | --- |\n| 事实 | 0 | 380 | 过时 640 |";
export const stressTable = "| 项目 | 有效 | 候选 / 偶现 | 其他 |\n| --- | --- | --- | --- |\n| 合成示例项目 | 123 | 456 | 这是一段连续中文的合成表格内容用于检查最右侧列可以通过表格自己的滚动区域完整阅读 |";
export const extension = "layout-fixture";
export const ui = "mobile";
export const listState = JSON.stringify({ tab: "facets", facet: "judgement" });
export const detailState = JSON.stringify({ tab: "facets", open: "e:sample" });
export const place = { extension, id: ui, placement: "nav_page", title: "合成扩展", extension_name: "合成扩展", icon: "list" };
export const listView = {
  title: place.title, state: listState,
  tabs: { name: "tab", value: "facets", action: { id: "tab" }, items: [{ value: "start", title: "起步" }, { value: "facets", title: "六面" }, { value: "people", title: "人物" }, { value: "check", title: "核对" }, { value: "review", title: "待确认" }] },
  sections: [
    { items: [{ type: "segmented", name: "facet", value: "judgement", choices: [{ value: "judgement", title: "③ 默认判断" }], action: { id: "facet" } }] },
    { title: "③ 默认判断", items: [{ type: "row", title: "管理示例类型", subtitle: "2 种", action: { id: "manage", push: { state: detailState } } }] },
    { title: sectionTitle, items: [
      { type: "row", key: "long-cjk", title: rowTitle, subtitle: rowSubtitle, badge: { text: "稳定", tone: "positive" }, action: { id: "open:sample", push: { state: detailState } } },
      { type: "row", key: "second-cjk", title: rowTitle + "附加的无空格中文文本", subtitle: rowSubtitle, badge: { text: "候选", tone: "caution" }, action: { id: "open:second", push: { state: detailState } } },
    ] },
  ],
};
export const detailView = { title: "合成报告", state: detailState, sections: [
  { title: "文档中的四列表格", items: [{ type: "markdown", text: documentTable }] },
  { title: "四列连续中文压力样例", items: [{ type: "markdown", text: stressTable }] },
] };
