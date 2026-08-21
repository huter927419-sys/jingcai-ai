package analyze

import (
	"strings"
	"testing"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
)

func TestMarketTalk(t *testing.T) {
	r := poisson.Result{HomeWin: 39, Draw: 29, AwayWin: 32}
	q := &market.Quote{
		Asian: &market.Handicap{Line: "平手/半球", LineNum: -0.25, Home: 0.94, Away: 0.76, PH: 52, PA: 48},
		OU:    &market.OU{Line: 2.5, Over: 0.85, Under: 0.95},
	}
	talk := MarketTalk("布洛涅", "圣红星", r, q, &eval.Advice{Talk: "可看主队，空间一般。"})
	for _, want := range []string{"布洛涅", "主胜", "亚盘", "大小球", "没有竞彩票面"} {
		if !strings.Contains(talk, want) {
			t.Fatalf("missing %s in %s", want, talk)
		}
	}
}
