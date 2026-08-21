package analyze

import (
	"fmt"
	"log"
	"strings"
	"time"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/lambdaest"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
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

func (e *Engine) SeedFromMarket(m sporttery.Match) error {
	if e == nil || e.Store == nil {
		return nil
	}
	if err := e.Store.UpsertSFCMatch(m); err != nil {
		return err
	}
	q, _ := e.Store.GetQuote(m.ID)
	seed := lambdaest.FromMatch(m)
	if q != nil && q.EU != nil && q.EU.H > 1 {
		seed = lambdaest.FromOdds(q.EU.H, q.EU.D, q.EU.A)
	}
	res := poisson.Evaluate(seed.Home, seed.Away)
	sides, hc := eval.FromQuote(q, res, seed.Home, seed.Away)
	sn, err := e.Store.GetSnapshot(m.ID, store.KindOpen)
	if err != nil {
		return err
	}
	if sn != nil {
		if len(sn.Eval) == 0 && q != nil {
			sn.Eval, sn.Handicap = sides, hc
			if len(sn.Takes) == 0 {
				headline, talk := fallbackCopy(m, res)
				sn.LambdaH, sn.LambdaA = seed.Home, seed.Away
				sn.Result = res
				sn.Headline, sn.PlainTalk = headline, talk
			}
			sn.FetchedAt = time.Now()
			return e.Store.SaveSnapshot(*sn)
		}
		return nil
	}
	headline, talk := fallbackCopy(m, res)
	return e.Store.SaveSnapshot(store.Snapshot{
		MatchID:   m.ID,
		Kind:      store.KindOpen,
		FetchedAt: time.Now(),
		OddsJSON:  `{}`,
		LambdaH:   seed.Home,
		LambdaA:   seed.Away,
		Headline:  headline,
		PlainTalk: talk,
		Result:    res,
		Eval:      sides,
		Handicap:  hc,
	})
}

func (e *Engine) CompleteSFC(id int64) error {
	takes, err := e.FillMissingTakes(id, store.KindOpen)
	if err != nil {
		return err
	}
	log.Printf("sfc experts %d takes %d", id, len(takes))
	return nil
}

func buildSFCRolePrompt(role experts.Role, m sporttery.Match, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "你现在只扮演「%s」。%s\n", role.Title, role.Hint)
	fmt.Fprintf(&b, "场次 %s %s %s vs %s，开球 %s。这是胜负彩场次，没有竞彩开售，禁止写竞彩 SP、让胜/让平/让负官方票面。\n",
		m.NumStr, m.LeagueAbb, m.Home, m.Away, m.Kickoff.Format("01-02 15:04"))
	b.WriteString("价值只对照市场欧赔、亚盘和大小球，不要用任何竞彩票面做价值。\n")
	b.WriteString(lineupLine(prev))
	b.WriteString(marketLine(q))
	b.WriteString(valueLine(seed, q))
	b.WriteString("严格围绕当前角色展开。盘口分析师看欧亚盘和大小球变化；价值研判师看定价偏差与执行等级；进球分析师看节奏和大小球；阵容分析师看阵型首发。让球用亚盘主队/客队表达。必须引用至少两项具体证据。参考买入必须写临场失效条件和风险，不能承诺结果。资料没有的内容写未确认。只根据赛前资料预测。\n")
	return b.String()
}

func buildSFCSoftPrompt(role experts.Role, m sporttery.Match, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "角色：%s。%s %s %s vs %s。本场没有竞彩票面。", role.Title, m.NumStr, m.LeagueAbb, m.Home, m.Away)
	b.WriteString(lineupLine(prev))
	b.WriteString(marketLine(q))
	b.WriteString(valueLine(seed, q))
	b.WriteString("伤停未提供时必须写未确认。请输出 JSON。不要写竞彩 SP。plain_talk 按盘口与机构态度、阵型对位、胜平负、亚盘主客、大小球、失效条件组织。buy_talk 写参考买入和风险。")
	return b.String()
}
