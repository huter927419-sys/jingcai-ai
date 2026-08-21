package experts

import (
	"fmt"
	"strings"

	"jingcai-ai/internal/store"
)

type Role struct {
	Key   string
	Title string
	Hint  string
}

func Of(model string) Role {
	switch strings.TrimSpace(model) {
	case "DeepSeek":
		return Role{Key: "value", Title: "价值猎手", Hint: "定方向，过价值差、松紧、热门三关"}
	case "Grok":
		return Role{Key: "market", Title: "盘口专家", Hint: "对照 Bet365 和必发，看有没有过热"}
	case "ChatGPT":
		return Role{Key: "goals", Title: "进球专家", Hint: "只看大小 2.5 和更常见的比分"}
	case "Claude":
		return Role{Key: "lineup", Title: "阵容专家", Hint: "看首发和近期状态能不能撑住这个判断"}
	default:
		return Role{Key: "value", Title: "价值猎手", Hint: "定方向，过三关"}
	}
}

func Catalog(models []string) []map[string]string {
	out := make([]map[string]string, 0, len(models))
	for _, n := range models {
		r := Of(n)
		out = append(out, map[string]string{"name": n, "role": r.Title, "key": r.Key, "hint": r.Hint})
	}
	return out
}

func Decorate(t *store.ModelTake) {
	if t == nil {
		return
	}
	r := Of(t.Name)
	if t.Role == "" {
		t.Role = r.Title
	}
	if t.RoleKey == "" {
		t.RoleKey = r.Key
	}
	t.Pick1X2 = Norm1X2(t.Pick1X2)
	t.PickOU = NormOU(t.PickOU)
	t.Verdict = NormVerdict(t.Verdict)
	if t.Pick1X2 == "" {
		t.Pick1X2 = maxSide(t.HomeWin, t.Draw, t.AwayWin)
	}
	if t.PickOU == "" {
		if t.Over25 >= t.Under25 {
			t.PickOU = "大"
		} else {
			t.PickOU = "小"
		}
	}
	if t.Verdict == "" {
		t.Verdict = "可看"
	}
	if strings.TrimSpace(t.BuyTalk) == "" {
		t.BuyTalk = DefaultAdvice(*t)
	}
}

func DefaultAdvice(t store.ModelTake) string {
	if t.Verdict == "放弃" {
		return "建议放弃。胜平负和大小都先不买。"
	}
	had := "胜平负看" + t.Pick1X2
	ou := "大小 2.5 看" + t.PickOU
	if t.Verdict == "主推" {
		return fmt.Sprintf("竞彩%s，结论主推。%s。优先买这一侧，别再追另一边。", had, ou)
	}
	return fmt.Sprintf("竞彩%s，结论可看。%s。可以看，不必急着上。", had, ou)
}

func Norm1X2(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "胜", "主", "主胜", "home", "h", "1":
		return "胜"
	case "平", "平局", "draw", "d", "x":
		return "平"
	case "负", "客", "客胜", "away", "a", "2":
		return "负"
	default:
		return ""
	}
}

func NormOU(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "大", "大2.5", "over", "o", "over25":
		return "大"
	case "小", "小2.5", "under", "u", "under25":
		return "小"
	default:
		return ""
	}
}

func NormVerdict(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "主推", "可看", "放弃":
		return s
	default:
		return ""
	}
}

func maxSide(h, d, a float64) string {
	switch {
	case h >= d && h >= a:
		return "胜"
	case a >= h && a >= d:
		return "负"
	default:
		return "平"
	}
}

func Actual1X2(home, away int) string {
	switch {
	case home > away:
		return "胜"
	case home < away:
		return "负"
	default:
		return "平"
	}
}

func ActualOU(home, away int) string {
	if home+away >= 3 {
		return "大"
	}
	return "小"
}

type Grade struct {
	Hit1X2 bool `json:"hit1x2"`
	HitOU  bool `json:"hitOu"`
	Points int  `json:"points"`
}

func GradeTake(t store.ModelTake, home, away int) Grade {
	Decorate(&t)
	g := Grade{
		Hit1X2: t.Pick1X2 == Actual1X2(home, away),
		HitOU:  t.PickOU == ActualOU(home, away),
	}
	if g.Hit1X2 {
		g.Points++
	}
	if g.HitOU {
		g.Points++
	}
	if t.Verdict == "主推" && g.Hit1X2 {
		g.Points++
	}
	if t.Verdict == "主推" && !g.Hit1X2 {
		g.Points--
	}
	return g
}

type BoardRow struct {
	Name    string  `json:"name"`
	Role    string  `json:"role"`
	RoleKey string  `json:"roleKey"`
	Games   int     `json:"games"`
	Hit1X2  int     `json:"hit1x2"`
	HitOU   int     `json:"hitOu"`
	Rate1X2 float64 `json:"rate1x2"`
	RateOU  float64 `json:"rateOu"`
	Points  int     `json:"points"`
}

func Board(rows []BoardRow) []BoardRow {
	for i := range rows {
		if rows[i].Games > 0 {
			rows[i].Rate1X2 = float64(rows[i].Hit1X2) / float64(rows[i].Games) * 100
			rows[i].RateOU = float64(rows[i].HitOU) / float64(rows[i].Games) * 100
		}
		r := Of(rows[i].Name)
		if rows[i].Role == "" {
			rows[i].Role = r.Title
		}
		if rows[i].RoleKey == "" {
			rows[i].RoleKey = r.Key
		}
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if lessBoard(rows[j], rows[i]) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	return rows
}

func lessBoard(a, b BoardRow) bool {
	if a.Games == 0 && b.Games > 0 {
		return false
	}
	if b.Games == 0 && a.Games > 0 {
		return true
	}
	if a.Rate1X2 != b.Rate1X2 {
		return a.Rate1X2 > b.Rate1X2
	}
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	if a.Games != b.Games {
		return a.Games > b.Games
	}
	return a.Name < b.Name
}
