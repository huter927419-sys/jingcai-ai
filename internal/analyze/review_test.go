package analyze

import (
	"strings"
	"testing"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

func TestBuildMissPromptIncludesScoreAndKind(t *testing.T) {
	h, a := 0, 1
	m := store.MatchRow{NumStr: "周四006", LeagueAbb: "欧罗巴", Home: "特拉布宗体育", Away: "费伦茨瓦罗斯", HomeGoals: &h, AwayGoals: &a}
	sn := &store.Snapshot{Result: poisson.Result{HomeWin: 58, Draw: 24, AwayWin: 18}, OddsJSON: `{"had":{"H":1.53,"D":3.77,"A":4.75}}`}
	takes := []store.ModelTake{
		{Name: "Grok", Pick1X2: "胜", Headline: "主胜略满"},
		{Name: "DeepSeek", Pick1X2: "胜", Headline: "主队稳但盘口略热"},
	}
	odds := eval.BoardFromJSON(sn.OddsJSON)
	if odds != nil {
		odds.HAD = sporttery.Odds{H: 1.53, D: 3.77, A: 4.75}
		odds.HasHAD = true
	}
	p := buildMissPrompt(m, sn, takes, odds, nil, nil, nil, experts.MissAll)
	if !strings.Contains(p, "0-1") || !strings.Contains(p, "全错") || !strings.Contains(p, "主胜略满") {
		t.Fatal(p)
	}
}
