# HarmonyOS phone appearance and Agents review

The HarmonyOS captures use product commit `361c7892700b3bf63ab83ca16502cede6c7a0d09` and visual HAP SHA-256 `baa945b9aa0499e1a3c8af0fa1b3e5eb32c99db237ea11559b4d22805876b640`. The reference iPhone uses commit `c2d44cc1406c9127e585f430ed692261edf24eff`. The two phones have a logical width of 402 and native screenshots of 1206 × 2622 pixels. The application UI is in Simplified Chinese.

| Review section | Native images with accessibility trees |
| --- | ---: |
| [Appearance settings](settings/README.md) | 94 |
| [Agents directory](agents/README.md) | 35 |
| [Older Hub provider metadata](legacy-hub/README.md) | 11 |

Each section pairs native PNGs with its captured accessibility trees, includes the relevant controls and observations, and records independent SHA-256 hashes. The smaller comparison images link from their respective sections. The directory and older-Hub scenarios use isolated synthetic workspaces.

The [package manifest](manifest.json) records the source trees, installation hash and section hashes. The [checksum list](SHA256SUMS) can be verified locally with `shasum -a 256 -c SHA256SUMS` from this directory. Original and public hashes are provided separately whenever privacy edits change an asset.
