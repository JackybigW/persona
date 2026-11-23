# Sources and verification

[Gallery](README.md) · [Machine-readable digests](manifest.json)

## Capture method

The iPhone test runner captured the native screen and XCTest accessibility hierarchy on iPhone 18 Pro, iOS 27.2, at 402 pt, using its standard content size or Accessibility Extra Extra Extra Large. The two Chinese HarmonyOS API 26 guests each ran 17 native UI groups covering the same 37 pages across three appearances. Both HarmonyOS guests used the same measured HAP at system font scale 1.0 or 3.2. The 402 vp guest produced 1206 × 2622 px frames. The standard Mate produced 1280 × 2832 px frames at the device-reported 365 vp width. The HarmonyOS UI used a synthetic Hub for the 37-page matrix. The two separately paired worktree cycles used the primary HAP and a controlled disposable Git repository.

The three-way boards are resampled sRGB previews of entire PNGs. Each single-page original archive contains the untouched source PNGs and their native hierarchy files in light, dark and large directories. All 333 PNG digests and all 333 hierarchy digests were compared with their separate original manifests before publication. Time stamps, operating-system UI, live sample values and scroll state follow the capture. Static screen counts and matching source trees do not imply pixel or interactive equivalence.

## Signed-off capture inputs

| Input | Revision, Git tree or installed SHA-256 |
| --- | --- |
| Captured HarmonyOS revision | [9464459adb8e0dd6d38b0128f20d6b2d6bec3e08](https://github.com/nwverse/persona/tree/9464459adb8e0dd6d38b0128f20d6b2d6bec3e08) |
| Captured iPhone revision | [a21e4c50893c65a5e6816bfe1c2302f821b960c7](https://github.com/nwverse/persona/tree/a21e4c50893c65a5e6816bfe1c2302f821b960c7) |
| Public PR source checked | [0e3faa35d8f1aaa795d68a1e4eb646a08aaefead](https://github.com/nwverse/persona/tree/0e3faa35d8f1aaa795d68a1e4eb646a08aaefead) |
| HarmonyOS entry/src/main Git tree, captured and public | 349dfbcf6d6053b86562be95ee726d15fa22c778 |
| Apple application Git tree | aee1002d17b57395ecf8a5ba9469f33e76cbebf5 |
| Shared native Git tree | 0af7be009bd4a58b4449c9b3c1b07e69da5116a3 |
| Matrix visual HAP SHA-256 | 2e7d3aab8c798b7aef6a061278f47a78b3db50bab298609fbf047e01573ce126 |
| Installed paired HAP SHA-256 | 50c89b553721a5711487663015afd7939d181da0836d58466a5e261694eea1a0 |
| Installed iPhone Persona | a6a0ada4be32425e914c315ff753976eb934ab725762c31aac1a6a64508e3cc6 |
| Installed iPhone Persona.debug.dylib | 38192d6c5f7239473a3b70db90d74260564294581dab62fe365d6ead66fe965f |

## Capture receipts

The following SHA-256 values identify the native acquisition and verification files. Original hierarchy content is included per page.

| Receipt | SHA-256 |
| --- | --- |
| two-width-native-current.json | 7dd4b30d3882e1e2d58324a8881597326f50b574ddf251d5ad25afe709404991 |
| raw-37x3-standard.json | 448b23b2f6d691e0035cc90380ea0dda8566fde9523a49eb3b2f05ee06528d07 |
| raw-37x3-matched.json | c2dd9f234d1b42c8266a9862f737c8837e94b18968073d82c8a8b73f406b3a51 |
| width-full/manifest.json | 25ff6f3fd9b4d07fa8e08a07780f22748d3cc3e8aa8b4d7b2fd8b095ed754639 |
| standard-device-release.json | 7fcb6e1b7cb0df26edbaef3be694bd50de89f692c16054f2b7243f778dad3efa |
| matched-device-release.json | 8ede24d18a676fb3c942d3cdae31f3aaf7e255e46f6cf88e6153025a8377f4f2 |
| iPhone-manifest.json | 267216b781c010e745070084a05628b22d9c5199fd2dad104e1e25809e90b1bb |

## Previously published evidence

The [earlier 11ee8cec HarmonyOS / 870924d3 iPhone gallery](https://github.com/nwverse/persona/blob/d1628b3dd0dd5267dd44646b8be544c4007880c7/pr-assets/12495/phone-review-final/README.md) uses different capture revisions. Its supplemental checks and pictures belong to that earlier run. All 111 boards in the current gallery come from the sources above.

## Original package layout

Each page ZIP contains light, dark and large; each appearance contains iphone.png, iphone.ax, matched.png, matched.ax.json, standard.png and standard.ax.json. The SHA-256 of each unmodified file appears in the public manifest. To check the complete download, use verify_download.py in the expanded gallery folder with the originals directory present. The compact offline ZIP only requires a browser.

Capture and verification attribution: native Persona XCTest and HarmonyOS UI test drivers, assembled and checked for PR #12501 by Codex on the maintainer’s local QA workspace.
