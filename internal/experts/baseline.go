package experts

import (
	"fmt"
	"strings"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/store"
)

const BaselineName = "基本盘"

func FromSnapshot(sn *store.Snapshot) store.ModelTake {
	if sn == nil {
		return store.ModelTake{}
	}
	r := sn.Result
	if r.HomeWin+r.Draw+r.AwayWin < 1 {
		return store.ModelTake{}
	}
	pick1 := maxSide(r.HomeWin, r.Draw, r.AwayWin)
	pickOU := "小"
	if r.Over25 >= r.Under25 {
		pickOU = "大"
	}
	scores := make([]string, 0, 2)
	for _, s := range r.Top {
		if strings.TrimSpace(s.Score) == "" {
			continue
		}
		scores = append(scores, s.Score)
		if len(scores) == 2 {
			break
		}
	}
	odds := eval.BoardFromJSON(sn.OddsJSON)
	hhad := baselineHHAD(odds, sn.LambdaH, sn.LambdaA)
	t := store.ModelTake{
		Name:         BaselineName,
		Headline:     baselineHeadline(r.HomeWin, r.Draw, r.AwayWin, r.Over25),
		PlainTalk:    baselineTalk(r, odds, hhad),
		Pattern:      baselinePattern(r),
		Scores:       scores,
		PickHandicap: hhad,
		HomeWin:      r.HomeWin,
		Draw:         r.Draw,
		AwayWin:      r.AwayWin,
		Over25:       r.Over25,
		Under25:      r.Under25,
		Pick1X2:      pick1,
		PickOU:       pickOU,
		BuyTalk:      "这是盘面基准，用来对照后面四位专家，不是买入指令。",
	}
	Decorate(&t)
	return t
}

func WithBaseline(takes []store.ModelTake, sn *store.Snapshot) []store.ModelTake {
	base := FromSnapshot(sn)
	if base.Name == "" {
		return takes
	}
	for _, t := range takes {
		if t.Name == BaselineName {
			return takes
		}
	}
	out := make([]store.ModelTake, 0, len(takes)+1)
	out = append(out, base)
	return append(out, takes...)
}

func baselineHHAD(odds *eval.Board, lh, la float64) string {
	if odds == nil {
		return ""
	}
	h, d, a := odds.HHADMarketH, odds.HHADMarketD, odds.HHADMarketA
	if h+d+a < 1 {
		line, ok := ParseHHADLine(odds.HHADLine)
		if !ok || lh < 0.1 || la < 0.1 {
			return ""
		}
		h, d, a = poisson.Handicap(lh, la, line)
	}
	if h+d+a < 1 {
		return ""
	}
	switch {
	case h >= d && h >= a:
		return "让胜"
	case a >= h && a >= d:
		return "让负"
	default:
		return "让平"
	}
}

func baselineHeadline(home, draw, away, over float64) string {
	sides := []struct {
		n string
		p float64
	}{
		{"主胜", home},
		{"平局", draw},
		{"客胜", away},
	}
	if sides[1].p > sides[0].p {
		sides[0], sides[1] = sides[1], sides[0]
	}
	if sides[2].p > sides[0].p {
		sides[0], sides[2] = sides[2], sides[0]
	}
	if sides[2].p > sides[1].p {
		sides[1], sides[2] = sides[2], sides[1]
	}
	gap := sides[0].p - sides[1].p
	strength := "双方胶着"
	if gap >= 18 {
		strength = "格局较清晰"
	} else if gap >= 8 {
		strength = "略有倾向"
	}
	goals := "大小相对均衡"
	if over >= 52 {
		goals = "进球略偏多"
	} else if over <= 48 {
		goals = "进球略偏少"
	}
	return fmt.Sprintf("%s更常见，%s，%s", sides[0].n, strength, goals)
}

func baselinePattern(r poisson.Result) string {
	side := "主队掌握更多主动权"
	if r.AwayWin > r.HomeWin {
		side = "客队反击空间更值得关注"
	}
	if r.Over25 >= 52 {
		return side + "，对攻倾向"
	}
	return side + "，节奏偏谨慎"
}

func baselineTalk(r poisson.Result, odds *eval.Board, hhad string) string {
	parts := []string{
		fmt.Sprintf("盘面基准看%s。", maxSide(r.HomeWin, r.Draw, r.AwayWin)),
	}
	if r.Over25 >= r.Under25 {
		parts = append(parts, "进球倾向偏大。")
	} else {
		parts = append(parts, "进球倾向偏小。")
	}
	if odds != nil && strings.TrimSpace(odds.HHADLine) != "" {
		line := strings.TrimSpace(odds.HHADLine)
		if hhad != "" {
			parts = append(parts, fmt.Sprintf("竞彩让球 %s，基准看%s。", line, hhad))
		} else {
			parts = append(parts, fmt.Sprintf("竞彩让球 %s。", line))
		}
	}
	if len(r.Top) > 0 && r.Top[0].Score != "" {
		parts = append(parts, fmt.Sprintf("相对更常见的比分路径是 %s。", r.Top[0].Score))
	}
	parts = append(parts, "这是价格和分布给出来的基准，后面四位专家可以和它不一致。")
	return strings.Join(parts, "")
}
