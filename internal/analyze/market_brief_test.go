package analyze

import (
	"strings"
	"testing"

	"jingcai-ai/internal/market"
)

func TestMarketBriefNamesBooks(t *testing.T) {
	q := &market.Quote{
		Books: []market.EUBook{{
			Company: "平博",
			Opening: &market.Trio{H: 1.60, D: 4.0, A: 5.5},
			Current: &market.Trio{H: 1.48, D: 4.2, A: 6.5},
		}},
		AsianMove: &market.LineMove{Company: "澳门", OpeningLine: "半球", CurrentLine: "一球", OpeningLeft: 0.93, OpeningRight: 0.85, CurrentLeft: 0.80, CurrentRight: 1.00},
		AsianBooks: []market.LineMove{
			{CompanyID: 3, Company: "Crown", OpeningLine: "-0.5", CurrentLine: "-1", OpeningLeft: 0.9, OpeningRight: 0.9, CurrentLeft: 0.85, CurrentRight: 0.95},
		},
	}
	s := marketBrief(q)
	for _, want := range []string{"平博", "澳门", "Crown", "一球"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
	if q.MarketSig() == "" {
		t.Fatal("empty sig")
	}
}
