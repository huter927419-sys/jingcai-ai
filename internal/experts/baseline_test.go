package experts

import (
	"testing"

	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/store"
)

func TestBaselineFromSnapshot(t *testing.T) {
	sn := &store.Snapshot{
		Result: poisson.Result{
			HomeWin: 48, Draw: 26, AwayWin: 26, Over25: 58, Under25: 42,
			Top: []poisson.ScoreProb{{Score: "1-0"}, {Score: "2-1"}},
		},
		OddsJSON: `{"hhad":{"line":"-1","odds":{"H":2.1,"D":3.4,"A":2.8}}}`,
		LambdaH:  1.4,
		LambdaA:  1.1,
	}
	got := FromSnapshot(sn)
	if got.Name != BaselineName || got.RoleKey != "shape" || got.Pick1X2 != "胜" || got.PickOU != "大" {
		t.Fatalf("%+v", got)
	}
	if got.PickHandicap == "" || len(got.Scores) != 2 {
		t.Fatalf("handicap/scores %+v", got)
	}
	if got.Verdict != "" {
		t.Fatalf("baseline should not carry value verdict: %q", got.Verdict)
	}
	dup := WithBaseline([]store.ModelTake{got}, sn)
	if len(dup) != 1 {
		t.Fatalf("dup %d", len(dup))
	}
	out := WithBaseline([]store.ModelTake{{Name: "Grok"}}, sn)
	if len(out) != 2 || out[0].Name != BaselineName || out[1].Name != "Grok" {
		t.Fatalf("%+v", out)
	}
}
