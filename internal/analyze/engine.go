package analyze

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/grok"
	"jingcai-ai/internal/lambdaest"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

type Engine struct {
	Store  *store.Store
	Models []*grok.Client
}

type Outcome struct {
	Skipped bool
	UsedAI  bool
}

// Run writes one snapshot. If that snapshot already exists, it does nothing
// and never calls the model — HTTP reads must go through the store, not here.
func (e *Engine) Run(m sporttery.Match, kind store.SnapshotKind) (Outcome, error) {
	exists, err := e.Store.HasSnapshot(m.ID, kind)
	if err != nil {
		return Outcome{}, err
	}
	if exists {
		return Outcome{Skipped: true}, nil
	}
	if err := e.Store.UpsertMatch(m); err != nil {
		return Outcome{}, err
	}

	seed := lambdaest.FromMatch(m)
	lh, la := seed.Home, seed.Away
	usedAI := false
	var usedModels []string
	headline, talk := fallbackCopy(m, poisson.Evaluate(lh, la))

	q, _ := e.Store.GetQuote(m.ID)
	hits := e.collectFor(m, kind, seed, q, nil)
	if len(hits) > 0 {
		usedAI = true
		var sumH, sumA float64
		nH, nA := 0, 0
		for _, h := range hits {
			usedModels = append(usedModels, h.name)
			if d := abs(h.out.LambdaHome - seed.Home); d <= 0.35 {
				sumH += h.out.LambdaHome
				nH++
			}
			if d := abs(h.out.LambdaAway - seed.Away); d <= 0.35 {
				sumA += h.out.LambdaAway
				nA++
			}
		}
		if nH > 0 {
			lh = sumH / float64(nH)
		}
		if nA > 0 {
			la = sumA / float64(nA)
		}
		if pick := pickCopy(hits); pick != nil {
			if strings.TrimSpace(pick.Headline) != "" {
				headline = clip(pick.Headline, 40)
			}
			if strings.TrimSpace(pick.PlainTalk) != "" {
				talk = strings.TrimSpace(pick.PlainTalk)
			}
		}
	}

	res := poisson.Evaluate(lh, la)
	if !usedAI {
		headline, talk = fallbackCopy(m, res)
	}
	oddsRaw, _ := json.Marshal(map[string]any{
		"had":  m.HAD,
		"hhad": map[string]any{"line": m.HHADLine, "odds": m.HHAD},
		"ttg":  m.TTG,
	})
	sides, hc := eval.FromQuote(q, res, lh, la)
	err = e.Store.SaveSnapshot(store.Snapshot{
		MatchID:    m.ID,
		Kind:       kind,
		FetchedAt:  time.Now(),
		OddsJSON:   string(oddsRaw),
		LambdaH:    lh,
		LambdaA:    la,
		Headline:   headline,
		PlainTalk:  talk,
		Result:     res,
		Eval:       sides,
		Handicap:   hc,
		UsedAI:     usedAI,
		UsedModels: usedModels,
		Takes:      takesFromHits(hits, seed),
		ExpertDone: true,
	})
	return Outcome{UsedAI: usedAI}, err
}

type modelHit struct {
	name string
	out  grok.Output
}

func (e *Engine) collectFor(m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) []modelHit {
	if prev == nil {
		prev, _ = e.Store.GetPreview(m.ID)
	}
	if q == nil {
		q, _ = e.Store.GetQuote(m.ID)
	}
	return e.collectWith(m, kind, seed, q, prev)
}

func (e *Engine) collectWith(m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) []modelHit {
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		hits []modelHit
	)
	for _, c := range e.Models {
		if c == nil || !c.Enabled() {
			continue
		}
		wg.Add(1)
		go func(c *grok.Client) {
			defer wg.Done()
			role := experts.Of(c.Name)
			p := buildRolePrompt(role, m, kind, seed, q, prev)
			if strings.Contains(strings.ToLower(c.Model), "claude") {
				p = buildSoftRolePrompt(role, m, kind, seed, q, prev)
			}
			out, err := c.Analyze(p)
			if err != nil {
				log.Printf("%s %s: %v", c.Name, m.NumStr, err)
				return
			}
			mu.Lock()
			hits = append(hits, modelHit{name: c.Name, out: out})
			mu.Unlock()
		}(c)
	}
	wg.Wait()
	return hits
}

func takesFromHits(hits []modelHit, seed lambdaest.Seed) []store.ModelTake {
	out := make([]store.ModelTake, 0, len(hits))
	for _, h := range hits {
		lh, la := seed.Home, seed.Away
		if d := abs(h.out.LambdaHome - seed.Home); d <= 0.35 {
			lh = h.out.LambdaHome
		}
		if d := abs(h.out.LambdaAway - seed.Away); d <= 0.35 {
			la = h.out.LambdaAway
		}
		res := poisson.Evaluate(lh, la)
		t := store.ModelTake{
			Name:         h.name,
			Headline:     clip(h.out.Headline, 40),
			PlainTalk:    strings.TrimSpace(h.out.PlainTalk),
			BuyTalk:      strings.TrimSpace(h.out.BuyTalk),
			Pattern:      strings.TrimSpace(h.out.Pattern),
			Scores:       h.out.Scores,
			PickHandicap: strings.TrimSpace(h.out.PickHandicap),
			HomeWin:      res.HomeWin,
			Draw:         res.Draw,
			AwayWin:      res.AwayWin,
			Over25:       res.Over25,
			Under25:      res.Under25,
			Pick1X2:      h.out.Pick1X2,
			PickOU:       h.out.PickOU,
			Verdict:      h.out.Verdict,
		}
		experts.Decorate(&t)
		out = append(out, t)
	}
	order := map[string]int{"Grok": 0, "ChatGPT": 1, "Claude": 2, "DeepSeek": 3}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			ai, aok := order[out[i].Name]
			bi, bok := order[out[j].Name]
			if !aok {
				ai = 9
			}
			if !bok {
				bi = 9
			}
			if bi < ai || (bi == ai && out[j].Name < out[i].Name) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (e *Engine) FillTakes(id int64, kind store.SnapshotKind) ([]store.ModelTake, error) {
	return e.fillTakes(id, kind, false)
}

func (e *Engine) FillMissingTakes(id int64, kind store.SnapshotKind) ([]store.ModelTake, error) {
	return e.fillTakes(id, kind, true)
}

func (e *Engine) fillTakes(id int64, kind store.SnapshotKind, allowEmptyDone bool) ([]store.ModelTake, error) {
	var sn *store.Snapshot
	var err error
	if kind == store.KindOpen || kind == store.KindClose {
		sn, err = e.Store.GetSnapshot(id, kind)
	} else {
		sn, err = e.Store.PreferredSnapshot(id)
	}
	if err != nil {
		return nil, err
	}
	if sn == nil {
		return nil, fmt.Errorf("no snapshot")
	}
	if len(sn.Takes) > 0 {
		return sn.Takes, nil
	}
	if sn.ExpertDone && !allowEmptyDone {
		return sn.Takes, nil
	}
	row, err := e.Store.GetMatch(id)
	if err != nil || row == nil {
		return nil, err
	}
	odds, ok := eval.MatchFromJSON(sn.OddsJSON)
	if !ok {
		return nil, fmt.Errorf("no odds")
	}
	m := odds
	m.ID = row.ID
	m.NumStr = row.NumStr
	m.League = row.League
	m.LeagueAbb = row.LeagueAbb
	m.Home = row.Home
	m.Away = row.Away
	m.Kickoff = row.Kickoff
	m.BusinessDate = row.BusinessDate
	seed := lambdaest.FromMatch(m)
	q, _ := e.Store.GetQuote(id)
	prev, _ := e.Store.GetPreview(id)
	hits := e.collectFor(m, sn.Kind, seed, q, prev)
	sn.ExpertDone = true
	if len(hits) == 0 {
		_ = e.Store.SaveSnapshot(*sn)
		return nil, fmt.Errorf("模型都没有返回")
	}
	sn.Takes = takesFromHits(hits, seed)
	sn.UsedModels = nil
	for _, t := range sn.Takes {
		sn.UsedModels = append(sn.UsedModels, t.Name)
	}
	sn.UsedAI = true
	sn.FetchedAt = time.Now()
	if err := e.Store.SaveSnapshot(*sn); err != nil {
		return nil, err
	}
	return sn.Takes, nil
}

func pickCopy(hits []modelHit) *grok.Output {
	prefer := []string{"DeepSeek", "Claude", "ChatGPT", "Grok"}
	for _, name := range prefer {
		for i := range hits {
			if strings.EqualFold(hits[i].name, name) && strings.TrimSpace(hits[i].out.PlainTalk) != "" {
				out := hits[i].out
				return &out
			}
		}
	}
	for i := range hits {
		if strings.TrimSpace(hits[i].out.PlainTalk) != "" || strings.TrimSpace(hits[i].out.Headline) != "" {
			out := hits[i].out
			return &out
		}
	}
	return nil
}

func buildRolePrompt(role experts.Role, m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "你现在只扮演「%s」。%s\n", role.Title, role.Hint)
	b.WriteString(matchLine(m, kind, seed))
	b.WriteString(lineupLine(prev))
	b.WriteString(marketLine(q))
	b.WriteString(valueLine(seed, q))
	if q != nil && q.Betfair != nil {
		fmt.Fprintf(&b, "必发成交 主%.0f 平%.0f 客%.0f。%s\n", q.Betfair.HomeVol, q.Betfair.DrawVol, q.Betfair.AwayVol, q.Betfair.Note)
	}
	b.WriteString("严格围绕当前角色的专业职责展开，不要平均复述所有数据。文案采用专业解盘口吻，按以下顺序组织：1）初盘到后市的盘口/水位变化及机构态度；2）阵型、首发/伤停或比赛节奏对位；3）胜平负与让球的格局方向；4）竞彩参考、两个情景比分和方向温度（温度只是概率强弱表达，不是命中率）。让球建议必须明确写成让胜、让平或让负，并在 plain_talk 和 buy_talk 中引用该方向对应的官方 SP；同时说明 SP 仅为官方票面定价，不能单独作为赛果依据。必须引用至少两项具体证据，并说明证据如何支持或削弱结论。参考买入必须写明临场失效条件和风险，不能承诺结果或收益。资料没有提供的内容必须写未确认，严禁补造。只根据赛前资料预测，不要假设已经踢完。\n")
	return b.String()
}

func buildSoftRolePrompt(role experts.Role, m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed, q *market.Quote, prev *market.Preview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "角色：%s。%s %s %s vs %s。", role.Title, m.NumStr, m.LeagueAbb, m.Home, m.Away)
	fmt.Fprintf(&b, "竞彩 SP 主%.2f 平%.2f 客%.2f。", m.HAD.H, m.HAD.D, m.HAD.A)
	b.WriteString(lineupLine(prev))
	b.WriteString(marketLine(q))
	b.WriteString(valueLine(seed, q))
	b.WriteString("伤停名单未提供时必须明确写未确认。请输出 JSON。plain_talk 按‘盘口变化与机构态度—阵型/首发/伤停对位—胜平负与让球格局—大小球节奏—结论与失效条件’组织，使用升盘、退盘、降水、阻上、盘赔背离、肋部、压迫、转换等专业解盘术语；必须明确胜平负、让胜/让平/让负、对应官方 SP、大小球和两个情景比分，并说明 SP 仅为票面定价。比分仅用于描述比赛路径，不得写成确定结果。buy_talk 必须带让球方向及对应 SP，并写参考买入和风险提示。")
	return b.String()
}

func valueLine(seed lambdaest.Seed, q *market.Quote) string {
	if q == nil {
		return "价值参数还没齐，先看基础数据。\n"
	}
	res := poisson.Evaluate(seed.Home, seed.Away)
	sides, hc := eval.FromQuote(q, res, seed.Home, seed.Away)
	if len(sides) == 0 && hc == nil {
		return "价值参数还没齐。\n"
	}
	var b strings.Builder
	b.WriteString("内部价值参数（勿写进对外文案的术语）：")
	for i, s := range sides {
		if i > 0 {
			b.WriteString("；")
		}
		hot := ""
		if s.Market > s.Model+1 {
			hot = "偏热"
		}
		fmt.Fprintf(&b, "%s 模型%.0f 盘%.0f 差%+.1f %s 松紧%s%s", s.Label, s.Model, s.Market, s.Value, s.ValueBand, s.KellyBand, hot)
	}
	if hc != nil && hc.Talk != "" {
		fmt.Fprintf(&b, "。亚盘：%s", hc.Talk)
	}
	b.WriteByte('\n')
	return b.String()
}

func matchLine(m sporttery.Match, kind store.SnapshotKind, seed lambdaest.Seed) string {
	label := "赛前"
	if kind == store.KindClose {
		label = "赛前半小时临场"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "场次 %s %s %s vs %s，开球 %s，这是%s。\n", m.NumStr, m.LeagueAbb, m.Home, m.Away, m.Kickoff.Format("01-02 15:04"), label)
	if m.HasHAD {
		fmt.Fprintf(&b, "竞彩胜平负 SP：主 %.2f / 平 %.2f / 客 %.2f。票面只作参考，价值不要用这组 SP。\n", m.HAD.H, m.HAD.D, m.HAD.A)
	}
	if m.HasTTG {
		fmt.Fprintf(&b, "竞彩大小看 2.5。\n")
	}
	if m.HHADLine != "" {
		fmt.Fprintf(&b, "竞彩让球 %s，官方 SP：让胜 %.2f / 让平 %.2f / 让负 %.2f。解读必须明确推荐让胜、让平或让负，并引用对应 SP；SP 只作票面参考，不参与价值判断。\n", m.HHADLine, m.HHAD.H, m.HHAD.D, m.HHAD.A)
	}
	return b.String()
}

func marketLine(q *market.Quote) string {
	if q == nil {
		return "Bet365 盘还没齐。\n"
	}
	var b strings.Builder
	if q.EU != nil && q.EU.H > 1 {
		fmt.Fprintf(&b, "Bet365 欧赔 主 %.2f 平 %.2f 客 %.2f，去水大约 主%.0f%% 平%.0f%% 客%.0f%%。\n",
			q.EU.H, q.EU.D, q.EU.A, q.EU.PH, q.EU.PD, q.EU.PA)
	}
	if q.OU != nil {
		fmt.Fprintf(&b, "Bet365 大小 %.1f 大 %.2f 小 %.2f。\n", q.OU.Line, q.OU.Over, q.OU.Under)
	}
	if q.Asian != nil {
		fmt.Fprintf(&b, "Bet365 亚盘 %s 主 %.2f 客 %.2f。\n", q.Asian.Line, q.Asian.Home, q.Asian.Away)
	}
	if b.Len() == 0 {
		return "Bet365 盘还没齐。\n"
	}
	return b.String()
}

func lineupLine(p *market.Preview) string {
	if p == nil {
		return "阵型、首发和伤停资料还没齐；不得推测具体缺阵球员。\n"
	}
	var b strings.Builder
	writeSide := func(tag string, s market.SidePreview) {
		if s.Name == "" && s.Formation == "" {
			return
		}
		fmt.Fprintf(&b, "%s阵型 %s，近期评分 %.1f。", tag, firstNonEmpty(s.Formation, "未定"), s.AvgRating)
		if len(s.Form) > 0 {
			b.WriteString("近况 ")
			n := len(s.Form)
			if n > 3 {
				n = 3
			}
			for i := 0; i < n; i++ {
				fmt.Fprintf(&b, "%s%s ", s.Form[i].Result, s.Form[i].Score)
			}
		}
		b.WriteByte('\n')
	}
	writeSide("主队", p.Home)
	writeSide("客队", p.Away)
	b.WriteString("伤停名单未接入；只能依据已确认首发、替补和近期状态判断，不得虚构伤停。\n")
	if b.Len() == 0 {
		return "阵容还没齐。\n"
	}
	return b.String()
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func fallbackCopy(m sporttery.Match, r poisson.Result) (string, string) {
	side := "更接近均势"
	switch {
	case r.HomeWin >= r.Draw && r.HomeWin >= r.AwayWin && r.HomeWin >= 42:
		side = "主队更占优"
	case r.AwayWin >= r.Draw && r.AwayWin >= r.HomeWin && r.AwayWin >= 42:
		side = "客队更占优"
	case r.Draw >= 30:
		side = "平局不低"
	}
	ou := "进球不会太多"
	if r.Over25 >= 52 {
		ou = "进球可能偏多"
	} else if r.Over25 >= 46 {
		ou = "大小球差不太多"
	}
	headline := side + "，" + ou
	top := "1-1"
	if len(r.Top) > 0 {
		top = r.Top[0].Score
	}
	talk := fmt.Sprintf("%s对%s，胜平负大约是主胜 %.0f%%、平局 %.0f%%、客胜 %.0f%%。大2.5大约 %.0f%%、小2.5大约 %.0f%%，所以%s。比分里 %s 更常见，但任何一场都可能打出别的结果，把它当参考就好。",
		m.Home, m.Away, r.HomeWin, r.Draw, r.AwayWin, r.Over25, r.Under25, ou, top)
	return headline, talk
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func clip(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n])
}
