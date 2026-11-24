# Registered worktree creation and compact conversation reading

The stock iPhone app ran against an isolated synthetic Hub. From the MacBook fixture’s registered Orca worktree, the tester opened **New Child Worktree**, confirmed the new branch, reviewed the prefilled **Join Space** sheet, then tapped Close. The original repository list retained the search and showed both the original parent and the new empty child. The Hub recorded one worktree creation and zero Agent session starts. The fixture returned child ID `w3`. The isolated simulator returned to its original light appearance and standard text size before shutdown.

[Original repository row](originals/repository-row.png) · [Native worktree menu](originals/repository-context-menu.png) · [Repository after Close](originals/repository-after-cancel.png)

The fixture branch was `qa/mini-login-form`. The source and child-parent identifiers matched across preparation, creation, and the host inventory. The three path-bearing originals stay in the private evidence set. Their screenshot and native-tree digests, the complete checkout path digest, and the original parent identity digest are available in the manifest. A separate older space-group UI test could not locate its former menu control and issued no create request.

| Main conversation at entry | Visible height | Last task top relative to the visible region | Gap to the decision dock |
|---|---:|---:|---:|
| iPhone 13 mini, 375pt, standard | 136.4pt | +52.5pt | 39.8pt |
| iPhone 18 Pro, 402pt, standard | 186.4pt | +102.5pt | 39.8pt |
| iPhone 13 mini, 375pt, largest | 83.3pt | −7.9pt | 39.7pt |
| iPhone 18 Pro, 402pt, largest | 92.0pt | +1.0pt | 39.8pt |

[375pt standard](originals/main-compact-375-light.png) · [375pt largest](originals/main-compact-375-large.png) · [402pt standard](originals/main-reference-402-light.png) · [402pt largest](originals/main-reference-402-large.png)

Both iPhones ran the same application executable and showed decision 1 of 5. At the largest text size, the header covers the top of the last task on the iPhone 13 mini. The separate HarmonyOS B850 sample at 365.71vp likewise clips 6.0vp after tapping the native Latest button, with a 40.29vp gap below the card. Its standard entry had a 0vp gap, which became 40vp after Latest. These widths and navigation gestures are recorded separately. A same-width comparison requires matched screens at 402.

[Source commits, controls, measurements and native frame hashes](manifest.json) · [Checksums](SHA256SUMS)
