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
		return Role{Key: "value", Title: "价值研判师", Hint: "专注市场定价与比赛概率的偏差，结合价格保护、拥挤风险和临场变化给出执行等级。使用定价偏差、价格空间、市场保护、风险溢价、价值回落等专业表达，不透露专属权重和综合算法。"}
	case "Grok":
		return Role{Key: "market", Title: "盘口分析师", Hint: "专注多家欧赔、亚洲盘、大小球与成交的联动。必须比较至少三家机构的初盘到即时盘，解释升盘、降盘、退盘、升降水、阻上、诱盘或盘赔背离，并写明共识与分歧。价值仍以 Bet365 为准，其他公司只作盘路对照。"}
	case "ChatGPT":
		return Role{Key: "goals", Title: "进球分析师", Hint: "专注比赛节奏与进球路径。结合阵型、压迫强度、攻守转换、边路纵深、肋部利用、定位球和大小球盘，判断开放或胶着格局及合理进球区间。"}
	case "Claude":
		return Role{Key: "lineup", Title: "阵容分析师", Hint: "专注阵型对位、首发结构、替补深度和伤停影响。分析中前场配置、边翼卫站位、中场人数优势、防线速度与对位弱点；未提供的伤停不得推测。"}
	default:
		return Role{Key: "value", Title: "价值研判师", Hint: "综合市场定价、价格保护与风险信号给出条件性结论。"}
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
		return "参考买入：当前盘价缺乏足够保护，胜平负与大小球均建议回避，等待临场重新确认。"
	}
	had := "胜平负看" + t.Pick1X2
	ou := "大小 2.5 看" + t.PickOU
	if t.Verdict == "主推" {
		return fmt.Sprintf("参考买入：主方向%s，次方向%s；当前信号达到主推等级，临场若出现退盘或核心首发变化则降低等级。", had, ou)
	}
	return fmt.Sprintf("参考买入：主方向%s，次方向%s；当前为可看等级，建议等待临场盘价和首发信息进一步确认。", had, ou)
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
