# Persona Health on HarmonyOS and iPhone

[Open the offline gallery](index.html) · [Original frames and source hashes](manifest.json) · [Native measurements](measurements.json)

The HarmonyOS app at `a4aa58a91eaf7f8ddbab13a63c0d8feff909587b` and the iPhone app at `698a1f789dff5e22a7930169483b3716fbfada21` were captured at 402 logical units. Both used isolated synthetic Health sessions. Each displayed suggestion and existing history title was verified against its own original accessibility tree and screen image. The ordinary HarmonyOS visits recorded zero Hub writes.

| Appearance | iPhone row height (pt) | HarmonyOS row height (vp) | Original views |
|---|---:|---:|---|
| Light | 70.3–70.3 | 70.33–70.33 | [Home](home-light.png) · [Eight rows](rows-light.png) |
| Dark | 70.3–70.3 | 70.33–70.33 | [Home](home-dark.png) · [Eight rows](rows-dark.png) |
| Largest text | 85.0–85.0 | 85.00–85.00 | [Home](home-large.png) · [Eight rows](rows-large.png) |

All four HarmonyOS history buttons were fully visible during each sequence of continuous scrolls. The light and dark iPhone frames show parts of the report conversation. Its full iPhone hit target comes from the accessibility tree. At the largest text size, the last iPhone suggestion appears in separate visible strips. Each crop retains its original pixel bounds and frame hash.

The dedicated sample opened “体检报告：血脂” once and used Back to return. A short left swipe exposed Archive and a reverse swipe closed it with zero writes. The native menu was observed. One full-swipe archived the selected existing conversation. The Hub received one archive request and no message or new-session writes. The normal list returned the same three other IDs, titles, previews and timestamps. A separate inclusive GET confirmed the original target was archived. The iPhone visit was read-only and its published product code attaches archive directly to the native button, full swipe and contextual menu. [Open and return](history-open.png) · [Two noncommitting gestures](history-short-swipe.png) · [Actual archive](history-direct-archive.png)

The independent intake run recorded one server-sized cup, a fractional protein amount and a custom water amount on the synthetic Hub. Each accepted entry was undone. The manifest distinguishes a lost reply and explicit idempotent retry from a second entry. The final totals and original entries were restored. The panel compares the printed amount, native screen-reader description, circular pixels and an independent GET after each action. [Native intake steps](intake-readback.png)

普通首页、进入已有会话和独立摄入记录各有单独回执。原生读屏动作由应用定义，本次设备操作使用触屏。收据只公开合成数据、界面、控件几何和哈希。

[SHA-256 inventory](SHA256SUMS)

The physical artwork review passed in all three appearances. Each control uses original PNG and native accessibility bounds from this release. Text endings, the actual time as shown, and the visible dots are recorded for each comparable row. [Light detail](artwork-light.png) · [Dark detail](artwork-dark.png) · [Largest text detail](artwork-large.png) · [Geometry and captured text](artwork-measurements.json)
