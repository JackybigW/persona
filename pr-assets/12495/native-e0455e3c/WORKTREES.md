# Two paired-worktree cycles · 两轮真实工作树往返

[Gallery](README.md) · [Source](SOURCES.md) · [Structured results](manifest.json)

The paired Mate used its existing association and the primary HAP. A dedicated temporary Git repository and the actual Hub confirmed both cycles. The user-visible paths were create, rename, preview a single dirty change, cancel and keep the file, then explicitly clean. Git HEAD and the pre-existing pairing were preserved. The public summary includes source fingerprints for the original local journal and screenshots. Raw paired screenshots can contain unrelated real workspace identities and are kept in the private local viewer. The synthetic matrix PNGs and accessibility trees are public for all 37 pages.

Primary HAP SHA-256: dc398981154acfeddc7307e4a89873c66ff3abab9523c79b90ceafc15231e6d6

Top-level lifecycle receipt SHA-256: 08fda5bd42c8fe87f991169a058176462c83d6883f1fdf539c8543832e46a21a

## Cycle 1 · qa/mobile-1123-153447

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: 3670c65c37d01453b8641e86b7ea752648a17e4ca95b3c23fea110cd80a7ff3f

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T15:34:48+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T15:35:48+08:00 | 99520fac889a41b7b8a1af5ac675f4d503401d2d1a936f406095e65294908429 |
| Rename: same branch and directory | 2026-11-23T15:36:35+08:00 | 76a9f7866c2da1997cf4f9ec1bd3aaa6504309cd714c459521a5fea58602bf50 |
| Preview: one changed file | 2026-11-23T15:36:45+08:00 | c7a18531c2411f37a65e2ef2341d2b326ae12a0d13d99bae99c2322d23d7eb8a |
| Cancel: fixture file preserved | 2026-11-23T15:36:47+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T15:37:05+08:00 | 539912282cc33d80120af8eb12091fd67a38a627968c7831f358b2eae3f6cdaf |

## Cycle 2 · qa/mobile-1123-153713

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: 3bae6df0b7b25296ccc802dcb9ed5f7a869120b7c37104d333971c1c13b12018

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T15:37:14+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T15:37:52+08:00 | 4e7164a85392192ab30d11b83aa154a557def50a3a05b16401211ce1faa6f48a |
| Rename: same branch and directory | 2026-11-23T15:39:11+08:00 | 2f92dc0b0dea22509e149d7e5a9e62f4a51f0dbf58f03d48cea08a991a0b634e |
| Preview: one changed file | 2026-11-23T15:39:21+08:00 | 60aaad0fcb6123e462fe52436ea70be96f8793fbce2b100882db41dc5c8987bb |
| Cancel: fixture file preserved | 2026-11-23T15:39:24+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T15:39:40+08:00 | 9a704717e5f74aa5622d17ecea39fd9a503d9330ab09070a70697e2816e05855 |
