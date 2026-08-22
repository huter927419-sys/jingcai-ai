package experts

import (
	"testing"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

func jcBoard(h, d, a float64, mh, md, ma float64) *eval.Board {
	return &eval.Board{
		HasHAD:  true,
		HAD:     sporttery.Odds{H: h, D: d, A: a},
		MarketH: mh, MarketD: md, MarketA: ma,
	}
}

func voices(picks ...string) []store.ModelTake {
	names := []string{"Grok", "ChatGPT", "Claude", "DeepSeek"}
	out := make([]store.ModelTake, 0, len(picks))
	for i, p := range picks {
		out = append(out, store.ModelTake{Name: names[i], Pick1X2: p, Verdict: "谨慎"})
	}
	return out
}

func keysOf(hints []RiskHint) map[string]bool {
	m := map[string]bool{}
	for _, h := range hints {
		m[h.Key] = true
	}
	return m
}

func TestRiskHintsFormVsPrice(t *testing.T) {
	takes := voices("胜", "胜", "胜")
	odds := jcBoard(1.53, 3.77, 4.75, 57, 23, 20)
	home := FormSide{Wins: 1, Games: 5, Rating: 4.7}
	away := FormSide{Wins: 2, Games: 5, Rating: 6.0}
	got := keysOf(RiskHints(takes, odds, 58, 24, 18, home, away))
	if !got["formVsPrice"] {
		t.Fatalf("want formVsPrice: %v", got)
	}
	// equal form should not fire
	even := FormSide{Wins: 2, Games: 5, Rating: 6}
	got = keysOf(RiskHints(takes, odds, 58, 24, 18, even, even))
	if got["formVsPrice"] {
		t.Fatalf("even form should be quiet: %v", got)
	}
}

func TestRiskHintsDrawUncovered(t *testing.T) {
	takes := voices("胜", "胜")
	odds := jcBoard(2.03, 2.84, 3.52, 41, 32, 27)
	got := keysOf(RiskHints(takes, odds, 44, 32, 24, FormSide{}, FormSide{}))
	if !got["drawUncovered"] {
		t.Fatalf("want drawUncovered: %v", got)
	}
	withDraw := voices("胜", "胜", "平")
	got = keysOf(RiskHints(withDraw, odds, 44, 32, 24, FormSide{}, FormSide{}))
	if got["drawUncovered"] {
		t.Fatalf("someone picked draw: %v", got)
	}
}

func TestRiskHintsHandicapSplit(t *testing.T) {
	takes := []store.ModelTake{
		{Name: "Grok", Pick1X2: "胜", PickHandicap: "让负"},
		{Name: "ChatGPT", Pick1X2: "胜", PickHandicap: "让负"},
		{Name: "Claude", Pick1X2: "胜", PickHandicap: "让胜"},
		{Name: "DeepSeek", Pick1X2: "胜", PickHandicap: "让胜"},
	}
	odds := jcBoard(1.58, 4, 4.05, 52, 25, 23)
	got := keysOf(RiskHints(takes, odds, 52, 25, 23, FormSide{Wins: 3, Games: 5, Rating: 6.8}, FormSide{Wins: 2, Games: 5, Rating: 5.9}))
	if !got["handicapSplit"] {
		t.Fatalf("want handicapSplit: %v", got)
	}
}

func TestRiskHintsFadeShape(t *testing.T) {
	takes := voices("负", "负", "负")
	odds := jcBoard(2.13, 3.37, 2.75, 42, 27, 31)
	got := keysOf(RiskHints(takes, odds, 39, 27, 34, FormSide{Wins: 1, Games: 5, Rating: 5.4}, FormSide{Wins: 1, Games: 5, Rating: 4.1}))
	if !got["fadeShape"] {
		t.Fatalf("want fadeShape: %v", got)
	}
}

func TestRiskHintsSkipBaseline(t *testing.T) {
	takes := []store.ModelTake{
		{Name: BaselineName, Pick1X2: "胜"},
		{Name: "Grok", Pick1X2: "负"},
		{Name: "DeepSeek", Pick1X2: "负"},
	}
	odds := jcBoard(2.13, 3.37, 2.75, 42, 27, 31)
	got := RiskHints(takes, odds, 39, 27, 34, FormSide{}, FormSide{})
	if majorityPick(expertVoices(takes)) != "负" {
		t.Fatal("baseline should not vote")
	}
	if !keysOf(got)["fadeShape"] {
		t.Fatalf("%+v", got)
	}
}

func TestRiskHintsNeedTwoVoices(t *testing.T) {
	takes := voices("胜")
	odds := jcBoard(1.5, 4, 6, 60, 22, 18)
	if hints := RiskHints(takes, odds, 60, 22, 18, FormSide{}, FormSide{}); len(hints) != 0 {
		t.Fatalf("%+v", hints)
	}
}
