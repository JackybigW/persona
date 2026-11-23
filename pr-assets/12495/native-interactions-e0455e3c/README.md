# Native chat, recap and SSH comparisons · 原生聊天、回顾与连接设置

The iPhone reference is `a21e4c50`. The selection and recap plates use the reviewed HarmonyOS build `e0455e3c2e09`. The palette and SSH plates state the exact feature build below. The full gallery includes the underlying 37 pages at two HarmonyOS widths. [37-page gallery](https://github.com/nwverse/persona/blob/38e96fd10c031c64f60d38d197a719c6faba72e6/pr-assets/12495/native-8572e8b8/README.md) · [Source and image fingerprints](evidence.json)

## Message selection · 消息选文

Long-pressing a historical reply and choosing Select Text opens the lifted reading panel. The full text can go into a quote on the original draft or be copied and pasted through the system clipboard. A second long press can quote a shorter span. Font-size changes kept the panel edges fixed on both platforms. In these captures the iPhone shows Look Up and Translate directly, while HarmonyOS exposes search, translate and share in the native More menu. A separate word-level run also recorded each platform's actual selection after a live text-size change. [Native reading, system actions and quotation](chat-message-selection.png)

## Recap reader · 离开期间回顾

At equal width, the iPhone sheet occupies x=8…394 and y=415…866 pt. The HarmonyOS sheet occupies x=8…394 and y=414.67…866 vp. The native drag bar opens the expanded reader and returns it to medium. Long-pressing its history entry copies the raw recap Markdown, confirmed by pasting into an empty isolated draft. The first four line tops were within 1 vp at regular size and 1.33 vp at maximum text. The images preserve the actual line wrapping and platform fonts. [Light](recap-light.jpg) · [Dark](recap-dark.jpg) · [Largest text](recap-large.jpg)

## Dark chat surfaces · 深色会话

The measured default conversation canvas is black on both platforms. The outgoing bubble uses #1D3557 on iPhone and in the HarmonyOS style resource. The flat area in the HarmonyOS screenshot is #1E3557. The ordinary Coral and Board choices and returning to the default were also exercised. The HarmonyOS capture used `9b6b7de7`. [Agent](agent-chat-dark.png) · [Reading history](reading-history-dark.png) · [Composer](composer-dark.png) · [Assistant](persona-dark.png)

## SSH target and route policy · SSH 隧道目标与连接顺序

The ordinary pairing forms preserve a custom forwarding target. With default settings the client used a reachable direct address and otherwise fell back to the selected SSH tunnel. Choosing Always SSH used the tunnel even with a reachable direct address. Host, username and both ports remain labeled after entry. On the stable-tunnel test, 14 live heartbeats kept the original connection open. Direct probing resumed at 119.666 seconds and the verified Hub was using the direct connection at 120.361 seconds. The HarmonyOS capture used `6bc9ed8a`. [Native route settings](ssh-route-settings.png)
