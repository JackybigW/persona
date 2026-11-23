# Failed task recovery

On the conversation task board, opening the task that failed to start shows its actual startup error. Ordinary to-do editing remains available. The source screenshot before this change is included in the comparison.

[Native comparison](failure-recovery.png) · [Source, coordinates and frame fingerprints](manifest.json)

On two independently seeded mock Hubs, iPhone and HarmonyOS each sent one Retry for the original task with a zero-byte request body. The recovery reached the new Agent and returned to the originating conversation. The HarmonyOS capture also exercised a read-only task and a task replaced between review and submission. Neither sent Retry.

Removing a task that never ran takes one explicit tap. If it has Agent history, removal asks for confirmation. The history-bearing test was cancelled, and the separate never-started sample was removed. The agents and archived source recordings are not part of the to-do removal.

| State | Platform and source | Image | Accessible controls |
| --- | --- | --- | --- |
| failure / retry | iphone `c2d44cc1` | [PNG](screens/iphone/failure/retry.png) | [Controls](accessibility/iphone/failure/retry.json) |
| failure / retry | harmony `adca61b0` | [PNG](screens/harmony/failure/retry.png) | [Controls](accessibility/harmony/failure/retry.json) |
| comparison / prior details | harmony `449ae587` | [PNG](screens/harmony/comparison/prior-details.png) | [Controls](accessibility/harmony/comparison/prior-details.json) |
| checks / failure readonly | harmony `adca61b0` | [PNG](screens/harmony/checks/failure-readonly.png) | [Controls](accessibility/harmony/checks/failure-readonly.json) |
| checks / failure updated | harmony `adca61b0` | [PNG](screens/harmony/checks/failure-updated.png) | [Controls](accessibility/harmony/checks/failure-updated.json) |
| checks / remove confirmation | harmony `adca61b0` | [PNG](screens/harmony/checks/remove-confirmation.png) | [Controls](accessibility/harmony/checks/remove-confirmation.json) |
| checks / retry agent | harmony `adca61b0` | [PNG](screens/harmony/checks/retry-agent.png) | [Controls](accessibility/harmony/checks/retry-agent.json) |
| checks / ordinary editor | harmony `adca61b0` | [PNG](screens/harmony/checks/ordinary-editor.png) | [Controls](accessibility/harmony/checks/ordinary-editor.json) |
| checks / return | iphone `c2d44cc1` | [PNG](screens/iphone/checks/return.png) | [Controls](accessibility/iphone/checks/return.json) |
| checks / return | harmony `adca61b0` | [PNG](screens/harmony/checks/return.png) | [Controls](accessibility/harmony/checks/return.json) |

Images and visible controls came from the native clients at 402 points or vp. Each export links to its original capture by SHA-256.
