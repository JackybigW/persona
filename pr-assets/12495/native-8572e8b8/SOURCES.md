# Sources and verification

[Gallery](README.md) · [Machine-readable digests](manifest.json)

## Capture method

The iPhone test runner captured the native screen and XCTest accessibility hierarchy on iPhone 18 Pro, iOS 27.2, at 402 pt, using its standard content size or Accessibility Extra Extra Extra Large. The two Chinese HarmonyOS API 26 guests each ran 17 native UI groups covering the same 37 pages across three appearances. Both HarmonyOS guests used the same measured HAP at system font scale 1.0 or 3.2. The 402 vp guest produced 1206 × 2622 px frames. The standard Mate produced 1280 × 2832 px frames at the device-reported 365 vp width. The HarmonyOS UI used a synthetic Hub for the 37-page matrix. The two separately paired worktree cycles used the primary HAP and a controlled disposable Git repository.

The three-way boards are resampled sRGB previews of entire PNGs. Each single-page original archive contains the untouched source PNGs and their native hierarchy files in light, dark and large directories. All 333 PNG digests and all 333 hierarchy digests were compared with their separate original manifests before publication. Time stamps, operating-system UI, live sample values and scroll state follow the capture. Static screen counts and matching source trees do not imply pixel or interactive equivalence.

## Signed-off capture inputs

| Input | Revision, Git tree or installed SHA-256 |
| --- | --- |
| Captured HarmonyOS revision | [8572e8b859c4683ebfdae911ebbd4c07719c83ac](https://github.com/nwverse/persona/tree/8572e8b859c4683ebfdae911ebbd4c07719c83ac) |
| Captured iPhone revision | [a21e4c50893c65a5e6816bfe1c2302f821b960c7](https://github.com/nwverse/persona/tree/a21e4c50893c65a5e6816bfe1c2302f821b960c7) |
| Public PR source checked | [8572e8b859c4683ebfdae911ebbd4c07719c83ac](https://github.com/nwverse/persona/tree/8572e8b859c4683ebfdae911ebbd4c07719c83ac) |
| HarmonyOS entry/src/main Git tree, captured and public | 58d2f674a99a7652a3d9bdc1f40e0829c9d176d5 |
| Apple application Git tree | aee1002d17b57395ecf8a5ba9469f33e76cbebf5 |
| Shared native Git tree | 0af7be009bd4a58b4449c9b3c1b07e69da5116a3 |
| Matrix visual HAP SHA-256 | 1e749ad38351b3278599c6e9edac19a71a1799727865b168bdce8cd8b54b16e0 |
| Installed paired HAP SHA-256 | e2d52c04e484616e9bd78c9d14233684a26d26cbc29da89ee4a65c6ae100cdcd |
| Installed iPhone Persona | a6a0ada4be32425e914c315ff753976eb934ab725762c31aac1a6a64508e3cc6 |
| Installed iPhone Persona.debug.dylib | 38192d6c5f7239473a3b70db90d74260564294581dab62fe365d6ead66fe965f |

## Capture receipts

The following SHA-256 values identify the native acquisition and verification files. Original hierarchy content is included per page.

| Receipt | SHA-256 |
| --- | --- |
| two-width-native-current.json | 5a57de9c1c8de5bf12afd7db43a12b319301146575abce7824a1c94a1316b843 |
| raw-37x3-standard.json | c2fbade33f413b34faa21b46addc6a5bba9a8448a537db2c634cf1a974b14a25 |
| raw-37x3-matched.json | 2565c664a59a9505a715019879253126ef4a0d3cefa0a18f5501f4462da2ebc7 |
| guest-cleanup.json | 84746c6669247042ab3944a94a7f72aedc6bcebafbecb994b010336a8e778e6c |
| native-37-source-bind.json | 915c02a498fad57fe4f0c2df1bf4914d4e6826d7c8a9790798cc270e60866bd5 |
| paired/receipt.json | 28436645f964c6ea122655dcee5a8bf75ea0496735e3c2cda1e167e4909a06fe |
| iPhone-manifest.json | 267216b781c010e745070084a05628b22d9c5199fd2dad104e1e25809e90b1bb |

## Previously published evidence

The [earlier 11ee8cec HarmonyOS / 870924d3 iPhone gallery](https://github.com/nwverse/persona/blob/d1628b3dd0dd5267dd44646b8be544c4007880c7/pr-assets/12495/phone-review-final/README.md) uses different capture revisions. Its supplemental checks and pictures belong to that earlier run. All 111 boards in the current gallery come from the sources above.

## Original package layout

Each page ZIP contains light, dark and large; each appearance contains iphone.png, iphone.ax, matched.png, matched.ax.json, standard.png and standard.ax.json. The SHA-256 of each unmodified file appears in the public manifest. To check the complete download, use verify_download.py in the expanded gallery folder with the originals directory present. The compact offline ZIP only requires a browser.

Capture and verification attribution: native Persona XCTest and HarmonyOS UI test drivers, assembled and checked for PR #12501 by Codex on the maintainer’s local QA workspace.
