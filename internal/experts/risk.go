package experts

import (
	"fmt"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/store"
)

// TrialUntil is the last calendar day (Asia/Shanghai) this hint set is treated as a live test.
const TrialUntil = "2026-08-27"

const (
	drawUncoveredMin = 26.0
	formMinGames     = 4
	formRatingGap    = 1.0
)

type RiskHint struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type FormSide struct {
	Wins   int
	Games  int
	Rating float64
}

func FormFromPreview(p *market.Preview) (home, away FormSide) {
	if p == nil {
		return FormSide{}, FormSide{}
	}
	return formOf(p.Home), formOf(p.Away)
}

func formOf(side market.SidePreview) FormSide {
	n := len(side.Form)
	if n > 5 {
		n = 5
	}
	wins := 0
	for _, g := range side.Form[:n] {
		if g.Result == "胜" {
			wins++
		}
	}
	return FormSide{Wins: wins, Games: n, Rating: side.AvgRating}
}

func CollectRiskHints(takes []store.ModelTake, sn *store.Snapshot, prev *market.Preview) []RiskHint {
	if sn == nil {
		return nil
	}
	home, away := FormFromPreview(prev)
	return RiskHints(takes, eval.BoardFromJSON(sn.OddsJSON), sn.Result.HomeWin, sn.Result.Draw, sn.Result.AwayWin, home, away)
}

func RiskHints(takes []store.ModelTake, odds *eval.Board, homeWin, draw, awayWin float64, home, away FormSide) []RiskHint {
	voices := expertVoices(takes)
	if len(voices) < 2 {
		return nil
	}
	consensus := majorityPick(voices)
	if consensus == "" {
		return nil
	}
	ticket := ticketFav(odds)
	shape := maxSide(homeWin, draw, awayWin)

	var out []RiskHint
	if h := formVsPrice(consensus, ticket, home, away); h != nil {
		out = append(out, *h)
	}
	if h := drawUncovered(voices, odds, draw); h != nil {
		out = append(out, *h)
	}
	if h := handicapSplit(voices, consensus); h != nil {
		out = append(out, *h)
	}
	if h := fadeShape(consensus, ticket, shape); h != nil {
		out = append(out, *h)
	}
	return out
}

func expertVoices(takes []store.ModelTake) []store.ModelTake {
	out := make([]store.ModelTake, 0, len(takes))
	for _, t := range takes {
		cp := t
		Decorate(&cp)
		if cp.Name == BaselineName || cp.RoleKey == "shape" {
			continue
		}
		if cp.Pick1X2 == "" {
			continue
		}
		out = append(out, cp)
	}
	return out
}

func majorityPick(takes []store.ModelTake) string {
	counts := map[string]int{}
	best, n := "", 0
	for _, t := range takes {
		p := t.Pick1X2
		counts[p]++
		if counts[p] > n {
			best, n = p, counts[p]
		}
	}
	return best
}

func ticketFav(odds *eval.Board) string {
	if odds == nil || !odds.HasHAD {
		return ""
	}
	h, d, a := odds.HAD.H, odds.HAD.D, odds.HAD.A
	if h <= 1 || d <= 1 || a <= 1 {
		return ""
	}
	m := h
	fav := "胜"
	if d < m {
		m, fav = d, "平"
	}
	if a < m {
		fav = "负"
	}
	return fav
}

func formVsPrice(consensus, ticket string, home, away FormSide) *RiskHint {
	if ticket == "" || consensus != ticket {
		return nil
	}
	if home.Games < formMinGames || away.Games < formMinGames {
		return nil
	}
	switch consensus {
	case "胜":
		worseWins := home.Wins < away.Wins
		worseRating := away.Rating-home.Rating >= formRatingGap
		if !worseWins && !worseRating {
			return nil
		}
		return &RiskHint{
			Key:    "formVsPrice",
			Title:  "近况逆价格",
			Detail: fmt.Sprintf("专家多数和票面都看主胜，但主队近%d场%d胜、评分%.1f，客队%d胜、评分%.1f。近况在帮客队，不是改口去追冷，只是共识可能偏热。", home.Games, home.Wins, home.Rating, away.Wins, away.Rating),
		}
	case "负":
		worseWins := away.Wins < home.Wins
		worseRating := home.Rating-away.Rating >= formRatingGap
		if !worseWins && !worseRating {
			return nil
		}
		return &RiskHint{
			Key:    "formVsPrice",
			Title:  "近况逆价格",
			Detail: fmt.Sprintf("专家多数和票面都看客胜，但客队近%d场%d胜、评分%.1f，主队%d胜、评分%.1f。近况在帮主队，只提示共识可能偏热。", away.Games, away.Wins, away.Rating, home.Wins, home.Rating),
		}
	default:
		return nil
	}
}

func drawUncovered(takes []store.ModelTake, odds *eval.Board, shapeDraw float64) *RiskHint {
	drawP := 0.0
	if odds != nil && odds.MarketD > 0 {
		drawP = odds.MarketD
	}
	if shapeDraw > drawP {
		drawP = shapeDraw
	}
	if drawP < drawUncoveredMin {
		return nil
	}
	for _, t := range takes {
		if t.Pick1X2 == "平" {
			return nil
		}
	}
	return &RiskHint{
		Key:    "drawUncovered",
		Title:  "平局未被覆盖",
		Detail: fmt.Sprintf("平局定价约%.0f%%，四位专家选项里没有平。胶着场次里这是最容易被写进分析、却不写进结论的一侧。", drawP),
	}
}

func handicapSplit(takes []store.ModelTake, consensus string) *RiskHint {
	homeHC, awayHC := false, false
	for _, t := range takes {
		switch t.PickHandicap {
		case "让胜":
			homeHC = true
		case "让负":
			awayHC = true
		}
	}
	if !homeHC || !awayHC {
		return nil
	}
	dir := consensus
	if dir == "" {
		dir = "同一侧"
	}
	return &RiskHint{
		Key:    "handicapSplit",
		Title:  "让球内部分裂",
		Detail: "胜平负看起来都看" + dir + "，让胜和让负同时出现。方向表面统一，盘口信心其实没有统一。",
	}
}

func fadeShape(consensus, ticket, shape string) *RiskHint {
	if consensus == "" || shape == "" {
		return nil
	}
	if consensus == shape {
		return nil
	}
	if ticket != "" && consensus == ticket {
		return nil
	}
	against := "基本盘看" + shape
	if ticket != "" && ticket == shape {
		against = "票面和基本盘都看" + shape
	} else if ticket != "" {
		against = "票面看" + ticket + "，基本盘看" + shape
	}
	return &RiskHint{
		Key:    "fadeShape",
		Title:  "专家逆基本盘",
		Detail: fmt.Sprintf("专家多数看%s，%s。把盘路故事压过结构时先标风险，不要把升温或降温单独写成结论。", consensus, against),
	}
}
