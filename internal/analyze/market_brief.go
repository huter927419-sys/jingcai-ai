package analyze

import (
	"fmt"
	"log"
	"strings"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/lambdaest"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

func marketBrief(q *market.Quote) string {
	if q == nil {
		return ""
	}
	var b strings.Builder
	if len(q.Books) > 0 {
		b.WriteString("欧赔多家对照（初→即）：")
		for i, bk := range q.Books {
			if i > 0 {
				b.WriteString("；")
			}
			op, cur := bk.Opening, bk.Current
			if op != nil && cur != nil && op.H > 1 && cur.H > 1 {
				fmt.Fprintf(&b, "%s %.2f/%.2f/%.2f → %.2f/%.2f/%.2f", bk.Company, op.H, op.D, op.A, cur.H, cur.D, cur.A)
				if dh := cur.H - op.H; dh <= -0.08 {
					b.WriteString("主胜降赔")
				} else if dh >= 0.08 {
					b.WriteString("主胜升赔")
				}
			}
		}
		b.WriteString("。\n")
	}
	writeSnaps := func(title string, rows []market.LineMove, hist *market.LineMove) {
		if len(rows) == 0 && hist == nil {
			return
		}
		fmt.Fprintf(&b, "%s：", title)
		n := 0
		if hist != nil && hist.OpeningLine != "" {
			fmt.Fprintf(&b, "澳门变盘 初 %s %.2f/%.2f → 即 %s %.2f/%.2f",
				hist.OpeningLine, hist.OpeningLeft, hist.OpeningRight, hist.CurrentLine, hist.CurrentLeft, hist.CurrentRight)
			n++
		}
		for _, r := range rows {
			if hist != nil && (r.CompanyID == 1 || r.Company == "澳门") {
				continue
			}
			if n > 0 {
				b.WriteString("；")
			}
			fmt.Fprintf(&b, "%s 初 %s %.2f/%.2f → 即 %s %.2f/%.2f",
				r.Company, r.OpeningLine, r.OpeningLeft, r.OpeningRight, r.CurrentLine, r.CurrentLeft, r.CurrentRight)
			n++
		}
		b.WriteString("。\n")
	}
	writeSnaps("亚盘多家对照", q.AsianBooks, q.AsianMove)
	writeSnaps("大小多家对照", q.OUBooks, q.OUMove)
	return b.String()
}

func buildMarketExpertPrompt(role experts.Role, m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview, sfc bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "你现在只扮演「%s」。%s\n", role.Title, role.Hint)
	if sfc {
		fmt.Fprintf(&b, "场次 %s %s %s vs %s。本场没有竞彩开售。\n", m.NumStr, m.LeagueAbb, m.Home, m.Away)
	} else {
		b.WriteString(matchLine(m, kind, seed))
	}
	b.WriteString(marketLine(q))
	b.WriteString(marketBrief(q))
	if q != nil && q.Betfair != nil {
		fmt.Fprintf(&b, "必发成交 主%.0f 平%.0f 客%.0f。%s\n", q.Betfair.HomeVol, q.Betfair.DrawVol, q.Betfair.AwayVol, q.Betfair.Note)
	}
	b.WriteString("只做盘口解读，不要改写价值等级算法。必须点名至少三家欧赔，并在有数据时比较至少三家亚盘和大小球。写清升盘、降盘、升水、降水、阻上、诱盘或盘赔背离，以及机构共识与分歧。亚盘用主队/客队+具体盘口表达。竞彩 SP 只作票面对照。plain_talk 按‘欧赔共识与分歧—亚盘升降—大小球—机构态度结论—失效条件’组织。资料没有的写未确认。\n")
	return b.String()
}

func (e *Engine) RefreshMarketTake(id int64) (bool, error) {
	if e == nil || e.Store == nil {
		return false, nil
	}
	row, err := e.Store.GetMatch(id)
	if err != nil || row == nil || row.Finished {
		return false, err
	}
	sn, err := e.Store.PreferredSnapshot(id)
	if err != nil || sn == nil || len(sn.Takes) == 0 {
		return false, err
	}
	q, _ := e.Store.GetQuote(id)
	sig := q.MarketSig()
	if sig == "" || sig == sn.MarketSig {
		return false, nil
	}
	odds, ok := eval.MatchFromJSON(sn.OddsJSON)
	sfc := row.Origin == "sfc"
	if !ok && !sfc {
		return false, nil
	}
	m := sporttery.Match{ID: row.ID, NumStr: row.NumStr, League: row.League, LeagueAbb: row.LeagueAbb, Home: row.Home, Away: row.Away, Kickoff: row.Kickoff, BusinessDate: row.BusinessDate}
	if ok {
		m.HAD, m.TTG, m.HHAD, m.HHADLine = odds.HAD, odds.TTG, odds.HHAD, odds.HHADLine
		m.HasHAD, m.HasTTG, m.HasHHAD = odds.HasHAD, odds.HasTTG, odds.HasHHAD
	}
	prev, _ := e.Store.GetPreview(id)
	seed := lambdaest.FromMatch(m)
	if sfc && q != nil && q.EU != nil && q.EU.H > 1 {
		seed = lambdaest.FromOdds(q.EU.H, q.EU.D, q.EU.A)
	}
	hits := e.collectRole(m, sn.Kind, seed, q, prev, "Grok", sfc)
	if len(hits) == 0 {
		return false, fmt.Errorf("盘口专家没有返回")
	}
	take := takesFromHits(hits, seed)[0]
	replaced := false
	for i := range sn.Takes {
		if strings.EqualFold(sn.Takes[i].Name, "Grok") || sn.Takes[i].RoleKey == "market" {
			sn.Takes[i] = take
			replaced = true
			break
		}
	}
	if !replaced {
		sn.Takes = append([]store.ModelTake{take}, sn.Takes...)
	}
	sn.MarketSig = sig
	sn.UsedAI = true
	if err := e.Store.SaveSnapshot(*sn); err != nil {
		return false, err
	}
	log.Printf("market take %s sig refreshed", m.NumStr)
	return true, nil
}

func (e *Engine) collectRole(m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview, name string, sfc bool) []modelHit {
	var hits []modelHit
	for _, c := range e.Models {
		if c == nil || !c.Enabled() || !strings.EqualFold(c.Name, name) {
			continue
		}
		role := experts.Of(c.Name)
		p := buildRolePrompt(role, m, kind, seed, q, prev)
		if sfc {
			p = buildSFCRolePrompt(role, m, seed, q, prev)
		}
		if role.Key == "market" {
			p = buildMarketExpertPrompt(role, m, kind, seed, q, prev, sfc)
		}
		out, err := c.Analyze(p)
		if err != nil {
			log.Printf("%s %s: %v", c.Name, m.NumStr, err)
			return nil
		}
		hits = append(hits, modelHit{name: c.Name, out: out})
	}
	return hits
}
