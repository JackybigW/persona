# Two paired-worktree cycles · 两轮真实工作树往返

[Gallery](README.md) · [Source](SOURCES.md) · [Structured results](manifest.json)

The paired Mate used its existing association and the primary HAP. A dedicated temporary Git repository and the actual Hub confirmed both cycles. The user-visible paths were create, rename, preview a single dirty change, cancel and keep the file, then explicitly clean. Git HEAD and the pre-existing pairing were preserved. The public summary includes source fingerprints for the original local journal and screenshots. Raw paired screenshots can contain unrelated real workspace identities and are kept in the private local viewer. The synthetic matrix PNGs and accessibility trees are public for all 37 pages.

Primary HAP SHA-256: e2d52c04e484616e9bd78c9d14233684a26d26cbc29da89ee4a65c6ae100cdcd

Top-level lifecycle receipt SHA-256: 28436645f964c6ea122655dcee5a8bf75ea0496735e3c2cda1e167e4909a06fe

## Cycle 1 · qa/mobile-1123-144329

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: a599095fc2e2cb28ee652d5d2565043794f8b62a32e4715fd0599da6df5f5488

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T14:43:30+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T14:44:32+08:00 | d4dab32b4108a710b10fd3833bff64c2d61f54e60d815638f0e56f69f04ce6f5 |
| Rename: same branch and directory | 2026-11-23T14:46:01+08:00 | 7c2a88e741511b2a7ee61f0afc4d9b0d5c6095ea455b6b042827a07e9c72154e |
| Preview: one changed file | 2026-11-23T14:46:30+08:00 | 9479b19d5e39602a7886d14b047b3c06fbe14a3ec37f5c15c42948e19f1dbed5 |
| Cancel: fixture file preserved | 2026-11-23T14:46:31+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T14:46:46+08:00 | b2b067ad4bfef014036886c7e34cd83716a03b0bf0c1bba7ea3b583e43f2547b |

## Cycle 2 · qa/mobile-1123-144655

Baseline and final HEAD: 29b565474a93b3f28bc05fa08ffcb120b3a1f338

Lifecycle JSON SHA-256: 761e741a21f7b8a572e4093b72403f7308fd587ff144544d55e50185ab7424a4

| Observed step | Time | Original native screenshot SHA-256 |
| --- | --- | --- |
| Record original HEAD | 2026-11-23T14:46:55+08:00 | — |
| Create: Hub workspace, checkout and Git branch | 2026-11-23T14:47:35+08:00 | 8169d7a440d570851b6e47b3f8e60e56d0b72ec73e408c27e7ded0a852055d5f |
| Rename: same branch and directory | 2026-11-23T14:48:15+08:00 | 7b112f599d1cdceb561cf0499000fc16d56bf1a3152772be83d9e97b7afc6ee7 |
| Preview: one changed file | 2026-11-23T14:48:43+08:00 | 1789a244670b641fdf421e33325c0789bc93f3829bcafe3c36f555af7b52465c |
| Cancel: fixture file preserved | 2026-11-23T14:48:45+08:00 | — |
| Clean: Hub entry, directory and branch removed | 2026-11-23T14:49:02+08:00 | 06094d83bf61e143958c6c117adeb592fd0ef7ebc8b99d187a3b7f41ee597c98 |
