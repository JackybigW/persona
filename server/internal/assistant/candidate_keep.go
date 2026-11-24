package assistant

import (
	"time"
)

// A continuation keeps its candidate (docs/assistant/model-profiles.md "Continuations keep the candidate"): the
// next Agent of a task (task_plan continues, task_retry, the card's Retry, a queued or restarted start read back)
// starts on the launcher + model the previous Agent ran on, so it reuses that Agent's context and cache, while the
// candidate is still listed in the tier, present on the computer and healthy. Only a clear problem re-picks:
//
//   - quota: the candidate has no headroom and none recovers within keepQuotaWait;
//   - rework: the to-do has been sent back for rework keepReworkRounds times;
//   - failed starts: keepFailStreak starts in a row failed.
//
// Explicit caller fields still win field by field.

const (
	keepQuotaWait    = 2 * time.Hour
	keepReworkRounds = 2
	keepFailStreak   = 3
)

// Why a continuation re-picks (dispatch args "rule").
const (
	switchQuota   = "quota"
	switchRework  = "rework"
	switchFailing = "failed_starts"
)

// keepFacts is what the re-pick rule reads about the kept candidate and the to-do.
type keepFacts struct {
	Headroom     candidateHeadroom
	ReworkRounds int
	FailStreak   int
}

// keepSwitch is the §6.2 rule: "" keeps the candidate, otherwise the rule that re-picks and its arguments. Pure.
func keepSwitch(f keepFacts, now time.Time) (string, map[string]any) {
	if f.ReworkRounds >= keepReworkRounds {
		return switchRework, map[string]any{"rule": switchRework, "rework_rounds": f.ReworkRounds}
	}
	if f.FailStreak >= keepFailStreak {
		return switchFailing, map[string]any{"rule": switchFailing, "failed_starts": f.FailStreak}
	}
	h := f.Headroom
	if h.Known && h.H <= 0 && (h.RecoversAt == nil || h.RecoversAt.Sub(now) > keepQuotaWait) {
		args := map[string]any{"rule": switchQuota, "headroom": roundScore(h.Percent)}
		if h.RecoversAt != nil {
			args["recovers_at"] = h.RecoversAt.UTC().Format(time.RFC3339)
		}
		return switchQuota, args
	}
	return "", nil
}
