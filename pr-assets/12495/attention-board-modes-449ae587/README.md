# Assistant attention and conversation tasks

The main conversation and an ordinary conversation were opened through the native clients on an equal 402-point or 402-vp viewport. All fixture data came from isolated mock Hubs. Each PNG has a matching accessibility excerpt and a fingerprint for the original unmodified device capture.

| Client | Source |
| --- | --- |
| iPhone | `c2d44cc1406c9127e585f430ed692261edf24eff` |
| HarmonyOS task board | `449ae5876bd4cfb4b552d0f84d57904cad2e7b86` |
| HarmonyOS ordinary conversation | `1d03f450d70b0c0d5bef585947ee508354022481` |
| Earlier HarmonyOS board in the first comparison | `1370b7f1320042e76e5ac0fb8e2a5ec2e03b66ee` |

[Task board comparison](board.png) · [Ordinary conversation comparison](ordinary.png) · [Build, geometry and frame fingerprints](manifest.json)

The Needs You tray showed all four native rows at once. The board contained four tasks or Agents needing attention, six working, two with unread results, and one failed launch. All 13 titles and available subtitles matched the iPhone. The first row measured 67.67 vp in HarmonyOS and 68 points on iPhone. Two tasks had the same creation time and appeared in different orders across two independent iPhone launches.

The failed launch opened the matching task on both clients. This HarmonyOS board revision opened the usual to-do editor. iPhone opened the hosted failure detail with a retry action. A completed task was opened separately from the home tray on both platforms. Opening that task can mark it seen.

The ordinary conversation has a separate synthetic plan and an approval. Both native clients opened them. The HarmonyOS inline plan made one data read when expanded with the follow-up change. The capture performed no plan start, approval, task edit or message send.

## Screens

| State | iPhone | HarmonyOS |
| --- | --- | --- |
| main / needs tray | [PNG](screens/iphone/main/needs-tray.png) · [controls](accessibility/iphone/main/needs-tray.json) | [PNG](screens/harmony/main/needs-tray.png) · [controls](accessibility/harmony/main/needs-tray.json) |
| main / board top | [PNG](screens/iphone/main/board-top.png) · [controls](accessibility/iphone/main/board-top.json) | [PNG](screens/harmony/main/board-top.png) · [controls](accessibility/harmony/main/board-top.json) |
| main / board bottom | [PNG](screens/iphone/main/board-bottom.png) · [controls](accessibility/iphone/main/board-bottom.json) | [PNG](screens/harmony/main/board-bottom.png) · [controls](accessibility/harmony/main/board-bottom.json) |
| main / board task detail | [PNG](screens/iphone/main/board-task-detail.png) · [controls](accessibility/iphone/main/board-task-detail.json) | [PNG](screens/harmony/main/board-task-detail.png) · [controls](accessibility/harmony/main/board-task-detail.json) |
| main / done tray | [PNG](screens/iphone/main/done-tray.png) · [controls](accessibility/iphone/main/done-tray.json) | [PNG](screens/harmony/main/done-tray.png) · [controls](accessibility/harmony/main/done-tray.json) |
| baseline / board top | — | [PNG](screens/harmony/baseline/board-top.png) · [controls](accessibility/harmony/baseline/board-top.json) |
| ordinary / ordinary plan and decision | [PNG](screens/iphone/ordinary/ordinary-plan-and-decision.png) · [controls](accessibility/iphone/ordinary/ordinary-plan-and-decision.json) | [PNG](screens/harmony/ordinary/ordinary-plan-and-decision.png) · [controls](accessibility/harmony/ordinary/ordinary-plan-and-decision.json) |
| ordinary / ordinary expanded plan | [PNG](screens/iphone/ordinary/ordinary-expanded-plan.png) · [controls](accessibility/iphone/ordinary/ordinary-expanded-plan.json) | [PNG](screens/harmony/ordinary/ordinary-expanded-plan.png) · [controls](accessibility/harmony/ordinary/ordinary-expanded-plan.json) |
| ordinary / ordinary decision card | [PNG](screens/iphone/ordinary/ordinary-decision-card.png) · [controls](accessibility/iphone/ordinary/ordinary-decision-card.json) | [PNG](screens/harmony/ordinary/ordinary-decision-card.png) · [controls](accessibility/harmony/ordinary/ordinary-decision-card.json) |
| dark / board top | [PNG](screens/iphone/dark/board-top.png) · [controls](accessibility/iphone/dark/board-top.json) | [PNG](screens/harmony/dark/board-top.png) · [controls](accessibility/harmony/dark/board-top.json) |
| dark / done tray | [PNG](screens/iphone/dark/done-tray.png) · [controls](accessibility/iphone/dark/done-tray.json) | [PNG](screens/harmony/dark/done-tray.png) · [controls](accessibility/harmony/dark/done-tray.json) |
| large / board top | [PNG](screens/iphone/large/board-top.png) · [controls](accessibility/iphone/large/board-top.json) | [PNG](screens/harmony/large/board-top.png) · [controls](accessibility/harmony/large/board-top.json) |
| large / done tray | [PNG](screens/iphone/large/done-tray.png) · [controls](accessibility/iphone/large/done-tray.json) | [PNG](screens/harmony/large/done-tray.png) · [controls](accessibility/harmony/large/done-tray.json) |

Screenshots with a local path or connection detail have that small region masked. The manifest lists the mask rectangles and hashes of both versions. Excerpts retain visible labels and measured bounds and omit process and window metadata.
