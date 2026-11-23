# Two paired-worktree cycles · 两轮真实工作树往返

[Gallery](README.md) · [Source](SOURCES.md) · [Structured results](manifest.json)

The paired Mate used its existing association and the primary HAP. A dedicated temporary Git repository and the actual Hub confirmed both cycles. The user-visible paths were create, rename, preview a single dirty change, cancel and keep the file, then explicitly clean. Git HEAD and the pre-existing pairing were preserved. The public summary includes source fingerprints for the original local journal and screenshots. Raw paired screenshots can contain unrelated real workspace identities and are kept in the private local viewer. The synthetic matrix PNGs and accessibility trees are public for all 37 pages.

Primary HAP SHA-256: 5bc4910c784f2537d29a887cb67d4a1cd555a4ee8caeefe82cdd606b5fd7a1e2

Top-level lifecycle receipt SHA-256: 5f9248c225718a4e24fd9921c2e91f360edabf79a2c3eb96b4746631e17a2b3d

## Cycle 1 · qa/mobile-1123-182513

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: e676c852e80380aaec7b2823f8e787488d4aae161f8be9ac0285553ba273f75d

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T18:25:14+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T18:26:04+08:00 | 7877fa436ecdc9c78d443e282a0026c8cdda38c58c4ccac31dee8cbcc055aaf1 |
| Rename: same branch and directory | 2026-11-23T19:08:36+08:00 | 591be0d57a1d8e65d95bb025faf6ea0a88d32f821c7ece7d1034b17d5f464d4f |
| Preview: one changed file | 2026-11-23T19:08:47+08:00 | 07257efa0ca9a999c0ec9e96f9894db359138222fa12ea13dfb385ae6ce4d73b |
| Cancel: fixture file preserved | 2026-11-23T19:08:50+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T19:09:09+08:00 | 9fcee466c0722d470891e6b50ce0b6ebe8e2f60b222a6c5185a6a8fbb5018388 |

## Cycle 2 · qa/mobile-1123-191213

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: cd11f81e6f96c85a1e21073b490ed66afa84d787020b81f680fe7d0e1701d76e

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T19:12:14+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T19:14:21+08:00 | 235d8b70111ad0c943dc2994fbaa0a6bbb48fbc37b8944b577f59743a3247da0 |
| Rename: same branch and directory | 2026-11-23T19:21:11+08:00 | 041148116154c1afbcbfe95b8f7ea97c349ae55aabab98f1647477f51d74ffb2 |
| Preview: one changed file | 2026-11-23T19:21:28+08:00 | c5f992ac0e90282ab42e5ebd5462c348ba420ee646552476831dad2ba3f2f322 |
| Cancel: fixture file preserved | 2026-11-23T19:21:35+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T19:21:58+08:00 | e1e482cad9b343c14e6fc53f0038fe668829de5d6df3f9f0eb5f667ef043c435 |

## How the two native runs resumed

A different, previously completed checkout left a local rename reconciliation open. The normal successor UI marked that older row checked. The original 7ef860f4 package was restored before sending the current rename. Both create, rename, dirty cancellation and cleanup sequences were submitted on 7ef860f4. The earlier confirmation used build 448d8854ab8791cd160e07dbea1667a9ea897a36.

The second case resumed its confirmed creation after an application watchdog restart. Its native rename and cleanup then completed.

Reconciliation receipt SHA-256 df54f8b5840c6cc41490310496e6e7ab46c241434e9b64a65041642660067aa7.
