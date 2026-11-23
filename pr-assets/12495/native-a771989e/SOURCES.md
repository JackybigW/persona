# Sources and verification

[Gallery](README.md) · [Machine-readable digests](manifest.json)

## Capture method

The iPhone test runner captured the native screen and XCTest accessibility hierarchy on iPhone 18 Pro, iOS 27.2, at 402 pt, using its standard content size or Accessibility Extra Extra Extra Large. The two Chinese HarmonyOS API 26 guests each ran 17 native UI groups covering the same 37 pages across three appearances. Both HarmonyOS guests used the same measured HAP at system font scale 1.0 or 3.2. The 402 vp guest produced 1206 × 2622 px frames. The standard Mate produced 1280 × 2832 px frames at the device-reported 365 vp width. The HarmonyOS UI used a synthetic Hub for the 37-page matrix. The two separately paired worktree cycles used the primary HAP and a controlled disposable Git repository.

The three-way boards are resampled sRGB previews of entire PNGs. Each single-page original archive contains the untouched source PNGs and their native hierarchy files in light, dark and large directories. All 333 PNG digests and all 333 hierarchy digests were compared with their separate original manifests before publication. Time stamps, operating-system UI, live sample values and scroll state follow the capture. Static screen counts and matching source trees do not imply pixel or interactive equivalence.

## Signed-off capture inputs

| Input | Revision, Git tree or installed SHA-256 |
| --- | --- |
| Captured HarmonyOS revision | [a771989eadf9c62b4edca4c201de4c4869f3e4f0](https://github.com/nwverse/persona/tree/a771989eadf9c62b4edca4c201de4c4869f3e4f0) |
| Captured iPhone revision | [c2d44cc1406c9127e585f430ed692261edf24eff](https://github.com/nwverse/persona/tree/c2d44cc1406c9127e585f430ed692261edf24eff) |
| Public PR source checked | [a771989eadf9c62b4edca4c201de4c4869f3e4f0](https://github.com/nwverse/persona/tree/a771989eadf9c62b4edca4c201de4c4869f3e4f0) |
| HarmonyOS entry/src/main Git tree, captured and public | f23521e318b6079fee649c7ad0a6c829e4f95a07 |
| Apple application Git tree | 8c901a437c26520cf2cc3bac1f4c2258ac0ca7e5 |
| Shared native Git tree | 0af7be009bd4a58b4449c9b3c1b07e69da5116a3 |
| Matrix visual HAP SHA-256 | df8c7657c94fe94d717cd758b824ab6eb698fd5e264e1fabc1f83aa713f0a04f |
| Installed paired HAP SHA-256 | 2bfd33507e577050a2de6bbaf4ef99b413725860d7f2c9f814ea411d67f3e6d1 |
| Installed iPhone Persona | 65e36482774bab1500cf2d95399f922c3ec9d657d65801a1b886a366f30aa685 |
| Installed iPhone Persona.debug.dylib | bfd66cf5ef3f3efeecbf5114aa063d19dbffceb43f7850b78f89683b0c70c00a |

## Capture receipts

The following SHA-256 values identify the native acquisition and verification files. Original hierarchy content is included per page.

| Receipt | SHA-256 |
| --- | --- |
| two-width-native-current.json | c1eab229aba87ead4eee88e8713fa9a9bc221b5f15bd95d8eb1d53636519a53f |
| raw-37x3-standard.json | df7266d05962f9f27477d5f655e681476db587630d61176eb6da0a27c69e501a |
| raw-37x3-matched.json | 2a174334dd6281d74477c4b41e8e220157945c46fe7ed627d118a625b9a1cb05 |
| guest-cleanup.json | 51471b996c4807b0fdab734cb4083e2f70933d3acda98cbcde627f8ca4597eb0 |
| native-37-source-bind.json | c9b39ab07613cde746d72b7ff511191bef0f0b9010eda56ac0b10eaf6c675abd |
| paired/receipt.json | 0cb7a9a192822355bea0292cc205ed4e8295c3cec15684fd92f0eba984fc5131 |
| iPhone-manifest.json | 3973ceb531289225d5d90fb8640c3f5b73858aa0e39e2c6e1b40d501d28184af |

## Previously published evidence

The [earlier 11ee8cec HarmonyOS / 870924d3 iPhone gallery](https://github.com/nwverse/persona/blob/d1628b3dd0dd5267dd44646b8be544c4007880c7/pr-assets/12495/phone-review-final/README.md) uses different capture revisions. Its supplemental checks and pictures belong to that earlier run. All 111 boards in the current gallery come from the sources above.

## Original package layout

Each page ZIP contains light, dark and large; each appearance contains iphone.png, iphone.ax, matched.png, matched.ax.json, standard.png and standard.ax.json. The SHA-256 of each unmodified file appears in the public manifest. To check the complete download, use verify_download.py in the expanded gallery folder with the originals directory present. The compact offline ZIP only requires a browser.

Capture and verification attribution: native Persona XCTest and HarmonyOS UI test drivers, assembled and checked for PR #12501 by Codex on the maintainer’s local QA workspace.
