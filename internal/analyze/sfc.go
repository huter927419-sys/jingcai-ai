package analyze

import (
	"fmt"
	"strings"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/lambdaest"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
)

func ApplyQuote(row *market.SFCMatch, q *market.Quote) {
	if row == nil || q == nil {
		return
	}
	row.Quote = q
	h, d, a := row.EUHome, row.EUDraw, row.EUAway
	if q.EU != nil && q.EU.H > 1 && q.EU.D > 1 && q.EU.A > 1 {
		h, d, a = q.EU.H, q.EU.D, q.EU.A
		row.EUHome, row.EUDraw, row.EUAway = h, d, a
	}
	if h > 1 && d > 1 && a > 1 {
		res := ProbsFrom1X2(h, d, a)
		row.AnalyzedHome, row.AnalyzedDraw, row.AnalyzedAway = res.HomeWin, res.Draw, res.AwayWin
	}
}

func PackMarket(home, away string, homeWin, draw, awayWin float64, q *market.Quote) (string, *eval.Advice, []eval.Side) {
	r := poisson.Result{HomeWin: homeWin, Draw: draw, AwayWin: awayWin}
	var seed lambdaest.Seed
	if q != nil && q.EU != nil && q.EU.H > 1 {
		seed = lambdaest.FromOdds(q.EU.H, q.EU.D, q.EU.A)
	}
	sides, hc := eval.FromQuote(q, r, seed.Home, seed.Away)
	return MarketTalk(home, away, r, q, hc), hc, sides
}

func MarketTalk(home, away string, r poisson.Result, q *market.Quote, hc *eval.Advice) string {
	pick := "平局"
	switch {
	case r.HomeWin >= r.Draw && r.HomeWin >= r.AwayWin:
		pick = "主胜"
	case r.AwayWin >= r.Draw && r.AwayWin >= r.HomeWin:
		pick = "客胜"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s对%s，胜平负更偏向%s（主胜 %.0f%%、平局 %.0f%%、客胜 %.0f%%）。",
		home, away, pick, r.HomeWin, r.Draw, r.AwayWin)
	if q != nil && q.Asian != nil && q.Asian.Home > 0 {
		side := "主让"
		if q.Asian.LineNum > 0 {
			side = "主受让"
		} else if q.Asian.LineNum == 0 {
			side = "平手"
		}
		line := strings.TrimSpace(q.Asian.Line)
		if line == "" {
			line = hcLine(q.Asian.LineNum)
		}
		fmt.Fprintf(&b, "亚盘%s%s，主队水位 %.2f、客队水位 %.2f。", side, line, q.Asian.Home, q.Asian.Away)
	}
	if hc != nil && strings.TrimSpace(hc.Talk) != "" {
		b.WriteString(hc.Talk)
	}
	if q != nil && q.OU != nil && q.OU.Over > 0 {
		fmt.Fprintf(&b, "大小球 %.1f，大 %.2f / 小 %.2f。", q.OU.Line, q.OU.Over, q.OU.Under)
	}
	b.WriteString("本场没有竞彩票面，以上依据市场盘口，仅供参考。")
	return b.String()
}

func hcLine(n float64) string {
	if n == 0 {
		return "平手"
	}
	if n < 0 {
		return fmt.Sprintf("%.1f", -n)
	}
	return fmt.Sprintf("%.1f", n)
}
