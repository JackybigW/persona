# Paired native Git worktree runs

The API 26 HarmonyOS production app at `b850faa917c125d3466b5c29c0be55d0a072eb10` completed both runs on the paired Mate 80 simulator. Each run used a separate real Git branch and linked checkout in the same disposable repository. The app opened the repository from its registered owner on the Projects screen.

| Run | Created branch | Saved display name | Confirmation result |
| --- | --- | --- | --- |
| 1 | `qa/mobile-1124-083046` | [Mobile review 1124-083046](rename-1.png) | Branch, folder and Hub entry removed |
| 2 | `qa/mobile-1124-083506` | [Mobile review 1124-083506](rename-2.png) | Branch, folder and Hub entry removed |

For each run, the first cleanup preview showed one changed file. Cancelling preserved the file. After restoring the disposable file, the phone opened a fresh preview and confirmed cleanup with branch removal selected. The original main revision and pairing identity were preserved.

The manifest records the original screenshot hashes and resolution for the creation, rename and both preview screens. The two linked rename screenshots are the original PNGs. The other original screens contain local paths and are available with the local installation. [Commit, operation outcomes and checksums](manifest.json)
