package eval

import (
	"testing"

	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
)

func TestKellyBand(t *testing.T) {
	if KellyBand(1.05) != "松" || KellyBand(0.93) != "紧" || KellyBand(0.99) != "中" {
		t.Fatal(KellyBand(1.05), KellyBand(0.93), KellyBand(0.99))
	}
}

func TestFromQuoteUsesBooksNotJingcai(t *testing.T) {
	q := &market.Quote{
		EU:    &market.Trio{H: 2.00, D: 3.50, A: 3.50},
		OU:    &market.OU{Line: 2.5, Over: 0.90, Under: 0.90},
		Asian: &market.Handicap{Line: "半球", LineNum: -0.5, Home: 0.85, Away: 0.95},
	}
	r := poisson.Result{HomeWin: 48, Draw: 28, AwayWin: 24, Over25: 40, Under25: 60}
	sides, hc := FromQuote(q, r, 1.4, 1.1)
	if len(sides) < 3 {
		t.Fatalf("%+v", sides)
	}
	home := sides[0]
	if home.Label != "胜" || home.Odds != 2 {
		t.Fatalf("%+v", home)
	}
	if home.Market == 0 || home.Value == 0 {
		t.Fatalf("model/market %+v", home)
	}
	if hc == nil || len(hc.Sides) != 2 {
		t.Fatalf("asian %+v", hc)
	}
}

func TestFromMatchDoesNotUseJingcai(t *testing.T) {
	sides, hc := FromOddsJSON(`{"had":{"H":2,"D":3.5,"A":3.5},"hhad":{"line":"-1","odds":{"H":1.5,"D":3.8,"A":4.5}}}`, poisson.Result{HomeWin: 48}, 1.4, 1.1)
	if len(sides) != 0 || hc != nil {
		t.Fatalf("value must not use 竞彩 SP: %+v %+v", sides, hc)
	}
}
