# Persona Health on HarmonyOS and iPhone

[Open the offline gallery](index.html) · [Original frames and source hashes](manifest.json) · [Native measurements](measurements.json)

The HarmonyOS app at `4914aa5b571504042efd7b6b095ad5b33321bf6e` and the iPhone app at `698a1f789dff5e22a7930169483b3716fbfada21` were captured at 402 logical units. Both used isolated synthetic Health sessions. Each displayed suggestion and existing history title was verified against its own original accessibility tree and screen image. The ordinary HarmonyOS visits recorded zero Hub writes.

| Appearance | iPhone row height (pt) | HarmonyOS row height (vp) | Original views |
|---|---:|---:|---|
| Light | 70.3–70.3 | 70.33–70.33 | [Home](home-light.png) · [Eight rows](rows-light.png) |
| Dark | 70.3–70.3 | 70.33–70.33 | [Home](home-dark.png) · [Eight rows](rows-dark.png) |
| Largest text | 85.0–85.0 | 85.00–85.00 | [Home](home-large.png) · [Eight rows](rows-large.png) |

All four HarmonyOS history buttons were fully visible during each sequence of continuous scrolls. The light and dark iPhone frames show parts of the report conversation. Its full iPhone hit target comes from the accessibility tree. At the largest text size, the last iPhone suggestion appears in separate visible strips. Each crop retains its original pixel bounds and frame hash.

The complete interaction sample opened the HarmonyOS session “体检报告：血脂” once. The native transcript and its matching session-items request were observed before a normal Back. A long press and a separate short left swipe exposed Archive. Each route opened the HarmonyOS confirmation for that title. Cancel preserved all four IDs, titles, timestamps and previews. The Hub recorded zero archive, new-session or message writes. In the iPhone HealthView source, the native swipe and contextual menu invoke the archive action directly. Its three-mode device visit was read-only. [Open and return](history-open.png) · [Long press](history-long-press.png) · [Short swipe](history-swipe.png)

An earlier independent synthetic pairing opened the same report title once. Its original image and accessibility tree show the expected session and message. That checker stopped on the rendered message-ID prefix and the fixture was cleaned before Back. Both samples retain their own source hashes. [Earlier read-only screen](originals/prior-read-only-open.png)

The independent intake run recorded one server-sized cup, a fractional protein amount and a custom water amount on the synthetic Hub. Each accepted entry was undone. The manifest distinguishes a lost reply and explicit idempotent retry from a second entry. The final totals and original entries were restored. The panel compares the printed amount, native screen-reader description, circular pixels and an independent GET after each action. [Native intake steps](intake-readback.png)

普通首页、进入已有会话和独立摄入记录各有单独回执。原生读屏动作由应用定义，本次设备操作使用触屏。收据只公开合成数据、界面、控件几何和哈希。

[SHA-256 inventory](SHA256SUMS)
