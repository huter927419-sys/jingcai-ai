package analyze

import (
	"fmt"
	"log"
	"strings"
	"time"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/grok"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/store"
)

func (e *Engine) reviewer() *grok.Client {
	var fallback *grok.Client
	for _, c := range e.Models {
		if c == nil || !c.Enabled() {
			continue
		}
		if c.Name == "DeepSeek" {
			return c
		}
		if fallback == nil {
			fallback = c
		}
	}
	return fallback
}

// ReviewMisses writes a dedicated post-match note for yesterday's (and the day
// before, if scores arrived late) 1X2 misses. Existing reviews are left alone.
func (e *Engine) ReviewMisses(now time.Time) error {
	if e == nil || e.Store == nil {
		return nil
	}
	reviewer := e.reviewer()
	if reviewer == nil {
		log.Print("miss review: no model configured, skip")
		return nil
	}
	now = now.In(now.Location())
	biz := now.Add(-4 * time.Hour)
	days := []string{
		biz.AddDate(0, 0, -1).Format("2006-01-02"),
		biz.AddDate(0, 0, -2).Format("2006-01-02"),
	}
	seen := map[int64]bool{}
	var nOK, nSkip, nFail int
	for _, day := range days {
		list, err := e.Store.ListFinishedOnBusinessDate(day)
		if err != nil {
			return err
		}
		for _, m := range list {
			if seen[m.ID] || m.Origin == "sfc" || m.HomeGoals == nil || m.AwayGoals == nil {
				continue
			}
			seen[m.ID] = true
			ok, err := e.Store.HasMissReview(m.ID)
			if err != nil {
				return err
			}
			if ok {
				nSkip++
				continue
			}
			wrote, err := e.reviewMatch(reviewer, m)
			if err != nil {
				nFail++
				log.Printf("miss review %s: %v", m.NumStr, err)
				continue
			}
			if wrote {
				nOK++
			} else {
				nSkip++
			}
		}
	}
	log.Printf("miss review: wrote %d, skipped %d, failed %d", nOK, nSkip, nFail)
	return nil
}

func (e *Engine) reviewMatch(c *grok.Client, m store.MatchRow) (bool, error) {
	sn, err := e.Store.AuditSnapshot(m.ID)
	if err != nil || sn == nil {
		return false, err
	}
	takes := experts.WithBaseline(sn.Takes, sn)
	kind := experts.ClassifyMiss(takes, *m.HomeGoals, *m.AwayGoals)
	if kind == experts.MissNone {
		return false, nil
	}
	prev, _ := e.Store.GetPreview(m.ID)
	q, _ := e.Store.GetQuote(m.ID)
	odds := eval.BoardFromJSON(sn.OddsJSON)
	hints := experts.CollectRiskHints(takes, sn, prev)
	prompt := buildMissPrompt(m, sn, takes, odds, q, prev, hints, kind)
	out, err := c.Review(prompt)
	if err != nil {
		return false, err
	}
	return true, e.Store.SaveMissReview(store.MissReview{
		MatchID:       m.ID,
		GeneratedAt:   time.Now(),
		MissKind:      kind,
		Model:         c.Name,
		Headline:      clip(out.Headline, 40),
		PlainTalk:     strings.TrimSpace(out.PlainTalk),
		VisibleBefore: clipList(out.VisibleBefore, 4),
		Overread:      clipList(out.Overread, 4),
		Lesson:        clip(out.Lesson, 80),
	})
}

func buildMissPrompt(m store.MatchRow, sn *store.Snapshot, takes []store.ModelTake, odds *eval.Board, q *market.Quote, prev *market.Preview, hints []experts.RiskHint, kind string) string {
	var b strings.Builder
	actual := experts.Actual1X2(*m.HomeGoals, *m.AwayGoals)
	fmt.Fprintf(&b, "场次 %s %s %s vs %s，完场 %d-%d，实际胜平负 %s。本场属于%s，请只做赛后归因。\n",
		m.NumStr, m.LeagueAbb, m.Home, m.Away, *m.HomeGoals, *m.AwayGoals, actual, kind)
	if odds != nil && odds.HasHAD {
		fmt.Fprintf(&b, "竞彩胜平负 SP：主 %.2f / 平 %.2f / 客 %.2f。票面热门按最低 SP。\n", odds.HAD.H, odds.HAD.D, odds.HAD.A)
	}
	if q != nil && q.Asian != nil {
		fmt.Fprintf(&b, "亚洲盘 %s 主 %.2f 客 %.2f。\n", q.Asian.Line, q.Asian.Home, q.Asian.Away)
	}
	fmt.Fprintf(&b, "基本盘结构估计：主 %.0f%% 平 %.0f%% 客 %.0f%%。\n", sn.Result.HomeWin, sn.Result.Draw, sn.Result.AwayWin)
	home, away := experts.FormFromPreview(prev)
	if home.Games > 0 || away.Games > 0 {
		fmt.Fprintf(&b, "近况：主队近%d场%d胜、评分%.1f；客队近%d场%d胜、评分%.1f。\n", home.Games, home.Wins, home.Rating, away.Games, away.Wins, away.Rating)
	}
	if len(hints) > 0 {
		b.WriteString("赛前风险提示：")
		for i, h := range hints {
			if i > 0 {
				b.WriteString("；")
			}
			b.WriteString(h.Title + "——" + h.Detail)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("赛前没有触发风险提示。\n")
	}
	b.WriteString("专家赛前选项（不含基本盘）：\n")
	for _, t := range takes {
		cp := t
		experts.Decorate(&cp)
		if cp.Name == experts.BaselineName || cp.RoleKey == "shape" {
			continue
		}
		fmt.Fprintf(&b, "- %s：胜平负%s", cp.Role, cp.Pick1X2)
		if cp.PickHandicap != "" {
			fmt.Fprintf(&b, "，让球%s", cp.PickHandicap)
		}
		if cp.Verdict != "" {
			fmt.Fprintf(&b, "，等级%s", cp.Verdict)
		}
		if cp.Headline != "" {
			fmt.Fprintf(&b, "。标题：%s", cp.Headline)
		}
		if talk := strings.TrimSpace(cp.PlainTalk); talk != "" {
			if len([]rune(talk)) > 180 {
				talk = string([]rune(talk)[:180]) + "…"
			}
			fmt.Fprintf(&b, "。解盘：%s", talk)
		}
		b.WriteString("\n")
	}
	b.WriteString("请判断：哪些信号赛前就在、却没写进选项；哪些盘路故事被读过头。不要用赛果倒推必然性。")
	return b.String()
}

func clipList(ss []string, n int) []string {
	out := make([]string, 0, n)
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, clip(s, 80))
		if len(out) == n {
			break
		}
	}
	return out
}
