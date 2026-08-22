package experts

import "jingcai-ai/internal/store"

const (
	MissNone = ""
	MissAll  = "全错"
	MissHAD  = "胜平负看错"
)

func ClassifyMiss(takes []store.ModelTake, home, away int) string {
	actual := Actual1X2(home, away)
	voices := expertVoices(takes)
	if len(voices) < 2 {
		return MissNone
	}
	hits := 0
	counts := map[string]int{}
	for _, t := range voices {
		counts[t.Pick1X2]++
		if t.Pick1X2 == actual {
			hits++
		}
	}
	if hits == 0 {
		return MissAll
	}
	cons := majorityPick(voices)
	if cons == "" || cons == actual {
		return MissNone
	}
	top := counts[cons]
	tied := false
	for side, n := range counts {
		if side != cons && n == top {
			tied = true
			break
		}
	}
	if tied {
		return MissNone
	}
	return MissHAD
}
