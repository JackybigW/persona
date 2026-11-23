# Older Hub provider metadata
This focused HarmonyOS run used the same application package as the directory captures and a separate synthetic Hub. Two MacBook sessions omitted the optional provider name, label, type and launcher fields. One retained an explicit Codex history source. A third session retained its declared Codex provider as a control.

| Native route | Fixture condition | Observed result |
| --- | --- | --- |
| By Project, widget session | Provider and source kind absent | Claude mark, Terminal label, correct conversation opened and returned |
| By Repository, tests session | Provider absent, Codex history source present | Codex mark, Terminal label, correct conversation opened and returned |
| By Project, orders session | Codex explicitly declared | Codex mark, correct conversation opened and returned |

At 42 physical pixels square, both inferred marks matched their explicitly declared controls with a foreground intersection-over-union of 1.000. The first-party iPhone Settings sample also omits the provider and exposes a Claude mark. Its [image](../settings/raw/iphone/large/layout.png) and [accessibility tree](../settings/raw/iphone/large/layout.ax) are included with Settings. The native directory captures below were taken on HarmonyOS.

## Full-size captures

| Captured state | iPhone | HarmonyOS |
| --- | --- | --- |
| Relaunch and reopen Agents | — | [PNG](raw/harmony/cold-shell-choice.png) · [AX](raw/harmony/cold-shell-choice.ax.json) |
| By Repository / Network tests / List row | — | [PNG](raw/harmony/repo-002-compact.png) · [AX](raw/harmony/repo-002-compact.ax.json) |
| By Repository / Network tests / Open conversation | — | [PNG](raw/harmony/repo-002-opened.png) · [AX](raw/harmony/repo-002-opened.ax.json) |
| By Repository / Network tests / Return to list | — | [PNG](raw/harmony/repo-002-returned.png) · [AX](raw/harmony/repo-002-returned.ax.json) |
| Open the selected Agents menu | — | [PNG](raw/harmony/selected-agents-menu.png) · [AX](raw/harmony/selected-agents-menu.ax.json) |
| By Project / Widget / List row | — | [PNG](raw/harmony/space-001-compact.png) · [AX](raw/harmony/space-001-compact.ax.json) |
| By Project / Widget / Open conversation | — | [PNG](raw/harmony/space-001-opened.png) · [AX](raw/harmony/space-001-opened.ax.json) |
| By Project / Widget / Return to list | — | [PNG](raw/harmony/space-001-returned.png) · [AX](raw/harmony/space-001-returned.ax.json) |
| By Project / Orders / List row | — | [PNG](raw/harmony/space-004-compact.png) · [AX](raw/harmony/space-004-compact.ax.json) |
| By Project / Orders / Open conversation | — | [PNG](raw/harmony/space-004-opened.png) · [AX](raw/harmony/space-004-opened.ax.json) |
| By Project / Orders / Return to list | — | [PNG](raw/harmony/space-004-returned.png) · [AX](raw/harmony/space-004-returned.ax.json) |

## Files and integrity

This section contains 11 images with their native accessibility captures. Its [machine-readable index](manifest.json) records the product revision and both acquisition and public byte hashes. [SHA-256 checksums](SHA256SUMS) cover the files in this section.

Public privacy edits affect 1 image(s) and 7 accessibility file(s). The acquisition hashes identify the local originals. On affected screenshots, the mask covers the transcript paragraph exposing workspace paths. Accessibility edits retain the control hierarchy and bounds.
