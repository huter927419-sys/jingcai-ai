package experts

import (
	"fmt"
	"strconv"
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
	case BaselineName:
		return Role{Key: "shape", Title: "基本盘分析师", Hint: "从欧赔重心、竞彩让球和进球分布给出盘面基准，不解读诱盘故事，也不给买入指令。"}
	default:
		return Role{Key: "value", Title: "价值研判师", Hint: "综合市场定价、价格保护与风险信号给出条件性结论。"}
	}
}

func Catalog(models []string) []map[string]string {
	out := make([]map[string]string, 0, len(models)+1)
	r := Of(BaselineName)
	out = append(out, map[string]string{"name": BaselineName, "role": r.Title, "key": r.Key, "hint": r.Hint})
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
	t.Role = r.Title
	t.RoleKey = r.Key
	t.Pick1X2 = Norm1X2(t.Pick1X2)
	t.PickOU = NormOU(t.PickOU)
	t.PickHandicap = NormHandicap(t.PickHandicap)
	t.Scores = CleanScores(t.Scores)
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
	if t.Name == BaselineName {
		return
	}
	if t.Verdict == "" {
		t.Verdict = "谨慎"
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
	return fmt.Sprintf("参考买入：主方向%s，次方向%s；当前为谨慎等级，可以留意但不宜急于执行，建议等待临场盘价和首发进一步确认。", had, ou)
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
	case "主推":
		return "主推"
	case "关注", "谨慎", "观望", "可看", "观察":
		return "谨慎"
	case "放弃", "回避":
		return "放弃"
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

func NormHandicap(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	switch {
	case s == "" || s == "放弃" || s == "观望" || s == "暂不介入" || strings.EqualFold(s, "skip"):
		return ""
	case strings.Contains(s, "让平"):
		return "让平"
	case strings.Contains(s, "让负"):
		return "让负"
	case strings.Contains(s, "让胜"):
		return "让胜"
	default:
		return ""
	}
}

func ParseHHADLine(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func ActualHHAD(home, away int, line float64) string {
	adj := float64(home-away) + line
	switch {
	case adj > 0.05:
		return "让胜"
	case adj < -0.05:
		return "让负"
	default:
		return "让平"
	}
}

func ParseScore(s string) (home, away int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	repl := []string{"：", "-", ":", "-", "比", "-", "—", "-", "–", "-", " ", ""}
	for i := 0; i+1 < len(repl); i += 2 {
		s = strings.ReplaceAll(s, repl[i], repl[i+1])
	}
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, e1 := strconv.Atoi(parts[0])
	a, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || h < 0 || a < 0 || h > 20 || a > 20 {
		return 0, 0, false
	}
	return h, a, true
}

func CleanScores(ss []string) []string {
	out := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, raw := range ss {
		h, a, ok := ParseScore(raw)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%d-%d", h, a)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
		if len(out) == 2 {
			break
		}
	}
	return out
}

func HitScore(preds []string, home, away int) bool {
	for _, p := range preds {
		h, a, ok := ParseScore(p)
		if ok && h == home && a == away {
			return true
		}
	}
	return false
}

type Grade struct {
	Hit1X2   bool `json:"hit1x2"`
	HitOU    bool `json:"hitOu"`
	HasHC    bool `json:"hasHc,omitempty"`
	HitHC    bool `json:"hitHc,omitempty"`
	HasScore bool `json:"hasScore,omitempty"`
	HitScore bool `json:"hitScore,omitempty"`
	Points   int  `json:"points"`
}

func GradeTake(t store.ModelTake, home, away int, hhadLine string) Grade {
	Decorate(&t)
	g := Grade{
		Hit1X2: t.Pick1X2 == Actual1X2(home, away),
		HitOU:  t.PickOU == ActualOU(home, away),
	}
	if line, ok := ParseHHADLine(hhadLine); ok && t.PickHandicap != "" {
		g.HasHC = true
		g.HitHC = t.PickHandicap == ActualHHAD(home, away, line)
	}
	if len(t.Scores) > 0 {
		g.HasScore = true
		g.HitScore = HitScore(t.Scores, home, away)
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
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	RoleKey    string  `json:"roleKey"`
	Games      int     `json:"games"`
	Hit1X2     int     `json:"hit1x2"`
	HitOU      int     `json:"hitOu"`
	GamesHC    int     `json:"gamesHc"`
	HitHC      int     `json:"hitHc"`
	GamesScore int     `json:"gamesScore"`
	HitScore   int     `json:"hitScore"`
	Rate1X2    float64 `json:"rate1x2"`
	RateOU     float64 `json:"rateOu"`
	RateHC     float64 `json:"rateHc"`
	RateScore  float64 `json:"rateScore"`
	Points     int     `json:"points"`
}

func Board(rows []BoardRow) []BoardRow {
	for i := range rows {
		if rows[i].Games > 0 {
			rows[i].Rate1X2 = float64(rows[i].Hit1X2) / float64(rows[i].Games) * 100
			rows[i].RateOU = float64(rows[i].HitOU) / float64(rows[i].Games) * 100
		}
		if rows[i].GamesHC > 0 {
			rows[i].RateHC = float64(rows[i].HitHC) / float64(rows[i].GamesHC) * 100
		}
		if rows[i].GamesScore > 0 {
			rows[i].RateScore = float64(rows[i].HitScore) / float64(rows[i].GamesScore) * 100
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
