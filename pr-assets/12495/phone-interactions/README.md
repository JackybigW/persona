# Phone interaction checks · 手机交互对照

These native screenshots complement the 37-page matrix. Each plate states its capture source. The JSON file contains the source and image fingerprints, native bounds and color samples. [Full page gallery](https://github.com/nwverse/persona/blob/f3dbffd7c7f4d6db571c4b9ebedc1d31655ed252/pr-assets/12495/native-9464459a/README.md) · [Evidence](evidence.json)

## Welcome · 首次启动

The four main welcome actions were measured in light, dark and largest text at equal width. The largest edge or dimension difference was 0.63 pt/vp. [Native screenshots](welcome.jpg)

## Agent text selection · 聊天选区

After selecting the fifth historical reply and switching to the largest accessibility size, both native apps kept that reply selected. iPhone presents its selection in a lifted text panel. In this HarmonyOS capture the selected text stays highlighted in the transcript. The iPhone capture used `870924d3`. Its Apple Chat view tree matches the later `a21e4c50` reference. The HarmonyOS before and after sources were `49adc1e9` and `3c69d0f2`. [Native screenshots](selection.jpg)

## Dark sheets · 深色弹层

The native iPhone and revised HarmonyOS sheets use RGB 28, 28, 30 behind their cards and RGB 44, 44, 46 inside. The visual comparison used `9464459a` and `33748351` on HarmonyOS. The four featured components match the public `dc2772d6` source. The keyboard is controlled by the device's input method and has a separate check. [Four native page comparisons](dark-sheets.jpg)

## Subscription meters · 机器订阅用量

The meter with a pacing indicator measured 8 pt/vp on each platform in all three appearances. The dark blue differs by one red-channel level. The hatched area starts at the verified protection threshold. The HarmonyOS capture used `64ccc8f9`. [Native screenshots](subscriptions.png)

## Pairing, camera and QR photos · 配对、相机与照片

These are the ordinary connection screens on the public `8572e8b8` build. Across light appearance, dark appearance and largest text, the largest measured difference among the listed controls was 1.033 pt/vp. In a separate camera-permitted run the system service started the inline 1280×720 stream in all three appearances. The stock photo picker returned the synthetic QR. Cancelling its decoded destination sent zero requests to it. [Six native pairing comparisons](pairing-sheets.jpg)

## SSH fingerprint approval · SSH 指纹确认

The ordinary iPhone and HarmonyOS forms displayed the first host fingerprint and a changed fingerprint. The first prompt sent zero authentication requests. Rotating the key again before tapping confirmation also sent zero. Confirming an unchanged, displayed key authorized the selected test destination. HarmonyOS source `c5ca42d9` and iPhone source `a21e4c50`. [Native first-use and changed-key screens](ssh-fingerprint-comparison.png)

## Knowledge map · 知识导图

Both clients read 5 topics, 27 points and 9 distinct sources. The native routes cover topic and project grouping, expanding and collapsing priorities, reading a point and returning to the original scope. The reference iPhone used `a21e4c50` and the HarmonyOS capture used `b8352dae`. The upper edge of the statistics card was within 0.37 pt/vp and the point reader’s close-button center was within 1.17 pt/vp in the three appearances.

| Appearance | Topics | Projects | Point detail |
| --- | --- | --- | --- |
| Light | [topics](knowledge-light-topics.jpg) | [projects](knowledge-light-projects.jpg) | [point](knowledge-light-point.jpg) |
| Dark | [topics](knowledge-dark-topics.jpg) | [projects](knowledge-dark-projects.jpg) | [point](knowledge-dark-point.jpg) |
| Largest type | [topics](knowledge-large-topics.jpg) | [projects](knowledge-large-projects.jpg) | [point](knowledge-large-point.jpg) |

## System keyboard · 系统输入法

The same stock HarmonyOS input method was measured on the same device with the application held in dark appearance. Switching the operating system to dark moved its keyboard-region median from 225 to 70. The iPhone dark keyboard measured 68. The image preserves each platform's real keyboard and layout. The HarmonyOS capture used `33748351`. [Native keyboards in all three states](keyboard-appearance.jpg)
