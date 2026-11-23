# Sources and verification

[Gallery](README.md) · [Machine-readable digests](manifest.json)

## Capture method

The iPhone test runner captured the native screen and XCTest accessibility hierarchy on iPhone 18 Pro, iOS 27.2, at 402 pt, using its standard content size or Accessibility Extra Extra Extra Large. The two Chinese HarmonyOS API 26 guests each ran 17 native UI groups covering the same 37 pages across three appearances. Both HarmonyOS guests used the same measured HAP at system font scale 1.0 or 3.2. The 402 vp guest produced 1206 × 2622 px frames. The standard Mate produced 1280 × 2832 px frames at the device-reported 365 vp width. The HarmonyOS UI used a synthetic Hub for the 37-page matrix. The two separately paired worktree cycles used the primary HAP and a controlled disposable Git repository.

The three-way boards are resampled sRGB previews of entire PNGs. Each single-page original archive contains the untouched source PNGs and their native hierarchy files in light, dark and large directories. All 333 PNG digests and all 333 hierarchy digests were compared with their separate original manifests before publication. Time stamps, operating-system UI, live sample values and scroll state follow the capture. Static screen counts and matching source trees do not imply pixel or interactive equivalence.

## Signed-off capture inputs

| Input | Revision, Git tree or installed SHA-256 |
| --- | --- |
| Captured HarmonyOS revision | [e0455e3c2e099d1b31eb75ca483068347cdb3bff](https://github.com/nwverse/persona/tree/e0455e3c2e099d1b31eb75ca483068347cdb3bff) |
| Captured iPhone revision | [a21e4c50893c65a5e6816bfe1c2302f821b960c7](https://github.com/nwverse/persona/tree/a21e4c50893c65a5e6816bfe1c2302f821b960c7) |
| Public PR source checked | [e0455e3c2e099d1b31eb75ca483068347cdb3bff](https://github.com/nwverse/persona/tree/e0455e3c2e099d1b31eb75ca483068347cdb3bff) |
| HarmonyOS entry/src/main Git tree, captured and public | 0fc31a8ed026d353719f1b1519dcbd59bb7e84cb |
| Apple application Git tree | aee1002d17b57395ecf8a5ba9469f33e76cbebf5 |
| Shared native Git tree | 0af7be009bd4a58b4449c9b3c1b07e69da5116a3 |
| Matrix visual HAP SHA-256 | 10a4850f08703981470052d7bcf56665f560c50c3fdbbcc8a12b6cf9c34685e3 |
| Installed paired HAP SHA-256 | dc398981154acfeddc7307e4a89873c66ff3abab9523c79b90ceafc15231e6d6 |
| Installed iPhone Persona | a6a0ada4be32425e914c315ff753976eb934ab725762c31aac1a6a64508e3cc6 |
| Installed iPhone Persona.debug.dylib | 38192d6c5f7239473a3b70db90d74260564294581dab62fe365d6ead66fe965f |

## Capture receipts

The following SHA-256 values identify the native acquisition and verification files. Original hierarchy content is included per page.

| Receipt | SHA-256 |
| --- | --- |
| two-width-native-current.json | d8ac6d60a6b06dde0333a51dbe1c928db33b66d2bb3a33bfbea819b9b4e28a3e |
| raw-37x3-standard.json | a91b5627caf4587a5a05f50c502731765cf3f05c9eb9238bd11518d35bad69fb |
| raw-37x3-matched.json | c2038d25609dfe351d0ce11c5615cd499a4b5b13d8e42a1b01ee199f5260443e |
| guest-cleanup.json | b0d4ff4144fb569ac9fd6cd709d681087b609da807edad106048d036a9d954f7 |
| native-37-source-bind.json | b74ce694a0f3572aef13f6e05ffcef33c294796bbdd1807f0eb5ca6f1e4a478d |
| paired/receipt.json | 08fda5bd42c8fe87f991169a058176462c83d6883f1fdf539c8543832e46a21a |
| iPhone-manifest.json | 267216b781c010e745070084a05628b22d9c5199fd2dad104e1e25809e90b1bb |

## Previously published evidence

The [earlier 11ee8cec HarmonyOS / 870924d3 iPhone gallery](https://github.com/nwverse/persona/blob/d1628b3dd0dd5267dd44646b8be544c4007880c7/pr-assets/12495/phone-review-final/README.md) uses different capture revisions. Its supplemental checks and pictures belong to that earlier run. All 111 boards in the current gallery come from the sources above.

## Original package layout

Each page ZIP contains light, dark and large; each appearance contains iphone.png, iphone.ax, matched.png, matched.ax.json, standard.png and standard.ax.json. The SHA-256 of each unmodified file appears in the public manifest. To check the complete download, use verify_download.py in the expanded gallery folder with the originals directory present. The compact offline ZIP only requires a browser.

Capture and verification attribution: native Persona XCTest and HarmonyOS UI test drivers, assembled and checked for PR #12501 by Codex on the maintainer’s local QA workspace.
