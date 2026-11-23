# Sources and verification

[Gallery](README.md) · [Machine-readable digests](manifest.json)

## Capture method

The iPhone test runner captured the native screen and XCTest accessibility hierarchy on iPhone 18 Pro, iOS 27.2, at 402 pt, using its standard content size or Accessibility Extra Extra Extra Large. The two Chinese HarmonyOS API 26 guests each ran 17 native UI groups covering the same 37 pages across three appearances. Both HarmonyOS guests used the same measured HAP at system font scale 1.0 or 3.2. The 402 vp guest produced 1206 × 2622 px frames. The standard Mate produced 1280 × 2832 px frames at the device-reported 365 vp width. The HarmonyOS UI used a synthetic Hub for the 37-page matrix. The two separately paired worktree cycles used the primary HAP and a controlled disposable Git repository.

The three-way boards are resampled sRGB previews of entire PNGs. Each single-page original archive contains the untouched source PNGs and their native hierarchy files in light, dark and large directories. All 333 PNG digests and all 333 hierarchy digests were compared with their separate original manifests before publication. Time stamps, operating-system UI, live sample values and scroll state follow the capture. Static screen counts and matching source trees do not imply pixel or interactive equivalence.

## Signed-off capture inputs

| Input | Revision, Git tree or installed SHA-256 |
| --- | --- |
| Captured HarmonyOS revision | [7ef860f483d2b9c9113063ee3080a3393d6892a4](https://github.com/nwverse/persona/tree/7ef860f483d2b9c9113063ee3080a3393d6892a4) |
| Captured iPhone revision | [c2d44cc1406c9127e585f430ed692261edf24eff](https://github.com/nwverse/persona/tree/c2d44cc1406c9127e585f430ed692261edf24eff) |
| Public PR source checked | [7ef860f483d2b9c9113063ee3080a3393d6892a4](https://github.com/nwverse/persona/tree/7ef860f483d2b9c9113063ee3080a3393d6892a4) |
| HarmonyOS entry/src/main Git tree, captured and public | 80d5d32fc06ae62fcfdf30918c274ae8be9d5020 |
| Apple application Git tree | 8c901a437c26520cf2cc3bac1f4c2258ac0ca7e5 |
| Shared native Git tree | 0af7be009bd4a58b4449c9b3c1b07e69da5116a3 |
| Matrix visual HAP SHA-256 | 4c88ef149633c033e327f925d179f5bad749dd519abba5fddab2953a711b79c7 |
| Installed paired HAP SHA-256 | 5bc4910c784f2537d29a887cb67d4a1cd555a4ee8caeefe82cdd606b5fd7a1e2 |
| Installed iPhone Persona | 65e36482774bab1500cf2d95399f922c3ec9d657d65801a1b886a366f30aa685 |
| Installed iPhone Persona.debug.dylib | bfd66cf5ef3f3efeecbf5114aa063d19dbffceb43f7850b78f89683b0c70c00a |

## Capture receipts

The following SHA-256 values identify the native acquisition and verification files. Original hierarchy content is included per page.

| Receipt | SHA-256 |
| --- | --- |
| two-width-native-current.json | f2c15ab7df595fe04eb6202ca446253ed32b21ecb34d0a59ab0d28026e6e6037 |
| raw-37x3-standard.json | 58e27580882ad7b06ac99dd8c184938f13e187b1a9621db17816615131bacab1 |
| raw-37x3-matched.json | 32ea43baed8fde2001874a1b149ddf0e863424b77dbc31d6da9889605b4873b2 |
| guest-cleanup.json | f1427cff18c7927f1a64d18fd62c97b35a3bd5244def9d3192b92c3ad4d39f07 |
| native-37-source-bind.json | 9ec1bbd75827e032bfd514febd1bf1b419a84b08134b8073df419d82cd97414d |
| paired/receipt.json | 5f9248c225718a4e24fd9921c2e91f360edabf79a2c3eb96b4746631e17a2b3d |
| iPhone-manifest.json | 3973ceb531289225d5d90fb8640c3f5b73858aa0e39e2c6e1b40d501d28184af |

## Previously published evidence

The [earlier 11ee8cec HarmonyOS / 870924d3 iPhone gallery](https://github.com/nwverse/persona/blob/d1628b3dd0dd5267dd44646b8be544c4007880c7/pr-assets/12495/phone-review-final/README.md) uses different capture revisions. Its supplemental checks and pictures belong to that earlier run. All 111 boards in the current gallery come from the sources above.

## Original package layout

Each page ZIP contains light, dark and large; each appearance contains iphone.png, iphone.ax, matched.png, matched.ax.json, standard.png and standard.ax.json. The SHA-256 of each unmodified file appears in the public manifest. To check the complete download, use verify_download.py in the expanded gallery folder with the originals directory present. The compact offline ZIP only requires a browser.

Capture and verification attribution: native Persona XCTest and HarmonyOS UI test drivers, assembled and checked for PR #12501 by Codex on the maintainer’s local QA workspace.

## Settings used by the native capture sequences

The iPhone reference enabled simulator-wide dark appearance. The HarmonyOS probe in this gallery set application appearance. The IME remained on its own system palette.
The iPhone Todo tests retained Recent in the light sequence and Priority in the dark and largest-text sequences. The native images here retain those choices. A separate same-Hub run on 55847be4 explicitly selected both orders under all three appearances.
On returning to Settings search, the iPhone test pressed Return before taking its reference frame. The HarmonyOS test captured the state immediately on return.
