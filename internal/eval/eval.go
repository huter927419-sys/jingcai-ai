package eval

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"
)

type Side struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Model     float64 `json:"model"`
	Market    float64 `json:"market"`
	Odds      float64 `json:"odds"`
	Value     float64 `json:"value"`
	ValueBand string  `json:"valueBand"`
	Kelly     float64 `json:"kelly"`
	KellyBand string  `json:"kellyBand"`
}

func KellyBand(k float64) string {
	switch {
	case k > 1.02:
		return "松"
	case k < 0.96:
		return "紧"
	default:
		return "中"
	}
}

func ValueBand(v float64) string {
	switch {
	case v >= 3:
		return "有价值"
	case v >= 0:
		return "边缘"
	default:
		return "无价值"
	}
}

func FromQuote(q *market.Quote, r poisson.Result, lh, la float64) ([]Side, *Advice) {
	if q == nil {
		return nil, nil
	}
	q.FillImplied()
	var out []Side
	if q.EU != nil && q.EU.H > 1 && q.EU.D > 1 && q.EU.A > 1 {
		out = append(out,
			side("home", "胜", r.HomeWin, q.EU.PH, q.EU.H),
			side("draw", "平", r.Draw, q.EU.PD, q.EU.D),
			side("away", "负", r.AwayWin, q.EU.PA, q.EU.A),
		)
	}
	if q.OU != nil && q.OU.Over > 0 && q.OU.Under > 0 {
		over, under := poisson.OverUnder(lh, la, q.OU.Line)
		out = append(out,
			side("over", fmtOU("大", q.OU.Line), over, q.OU.PO, hkOdd(q.OU.Over)),
			side("under", fmtOU("小", q.OU.Line), under, q.OU.PU, hkOdd(q.OU.Under)),
		)
	}
	return out, asianAdvice(q, lh, la)
}

type Advice struct {
	Line     string  `json:"line"`
	LineText string  `json:"lineText"`
	Pick     string  `json:"pick"`
	Talk     string  `json:"talk"`
	Home     float64 `json:"home"`
	Draw     float64 `json:"draw"`
	Away     float64 `json:"away"`
	Sides    []Side  `json:"sides"`
}

func asianAdvice(q *market.Quote, lh, la float64) *Advice {
	if q == nil || q.Asian == nil || q.Asian.Home <= 0 || q.Asian.Away <= 0 {
		return nil
	}
	if lh < 0.1 || la < 0.1 {
		return nil
	}
	mh, ma := poisson.Asian(lh, la, q.Asian.LineNum)
	sides := []Side{
		side("hc-home", "主队", mh, q.Asian.PH, hkOdd(q.Asian.Home)),
		side("hc-away", "客队", ma, q.Asian.PA, hkOdd(q.Asian.Away)),
	}
	best := sides[0]
	if sides[1].Value > best.Value {
		best = sides[1]
	}
	text := signedLine(q.Asian.LineNum)
	pick, talk := "不建议", "亚盘偏紧或没有空间，不必硬追。"
	if best.Value >= 3 && best.KellyBand != "紧" {
		pick = best.Label
		talk = "建议走" + text + "的" + best.Label + "。"
	} else if best.Value >= 0 && best.KellyBand != "紧" {
		pick = best.Label
		talk = text + "可看" + best.Label + "，空间一般。"
	}
	return &Advice{
		Line:     q.Asian.Line,
		LineText: text,
		Pick:     pick,
		Talk:     talk,
		Home:     mh,
		Away:     ma,
		Sides:    sides,
	}
}

func fmtOU(side string, line float64) string {
	return side + " " + signedLine(line)
}

func hkOdd(w float64) float64 {
	if w <= 0 {
		return 0
	}
	if w < 1.01 {
		return round2(1 + w)
	}
	return round2(w)
}

func parseLine(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func lineText(line float64) string {
	return signedLine(line)
}

func signedLine(v float64) string {
	if v == 0 {
		return "0"
	}
	s := trimFloat(absFloat(v))
	if v > 0 {
		return "+" + s
	}
	return "-" + s
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func trimFloat(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func FromOddsJSON(oddsJSON string, r poisson.Result, lh, la float64) ([]Side, *Advice) {
	_ = oddsJSON
	_ = r
	_ = lh
	_ = la
	return nil, nil
}

func MatchFromJSON(oddsJSON string) (sporttery.Match, bool) {
	if oddsJSON == "" {
		return sporttery.Match{}, false
	}
	var raw struct {
		HAD  sporttery.Odds `json:"had"`
		TTG  [8]float64     `json:"ttg"`
		HHAD struct {
			Line string         `json:"line"`
			Odds sporttery.Odds `json:"odds"`
		} `json:"hhad"`
	}
	if err := json.Unmarshal([]byte(oddsJSON), &raw); err != nil {
		return sporttery.Match{}, false
	}
	m := sporttery.Match{HAD: raw.HAD, TTG: raw.TTG, HHAD: raw.HHAD.Odds, HHADLine: raw.HHAD.Line}
	m.HasHAD = raw.HAD.H > 1 && raw.HAD.D > 1 && raw.HAD.A > 1
	m.HasHHAD = raw.HHAD.Line != "" && raw.HHAD.Odds.H > 1
	n := 0
	for _, o := range raw.TTG {
		if o > 1 {
			n++
		}
	}
	m.HasTTG = n >= 3
	return m, m.HasHAD || m.HasHHAD || m.HasTTG
}

type Board struct {
	HAD         sporttery.Odds `json:"had"`
	HasHAD      bool           `json:"hasHad"`
	MarketH     float64        `json:"marketH"`
	MarketD     float64        `json:"marketD"`
	MarketA     float64        `json:"marketA"`
	HHADLine    string         `json:"hhadLine"`
	HHADText    string         `json:"hhadText"`
	HHAD        sporttery.Odds `json:"hhad"`
	HHADMarketH float64        `json:"hhadMarketH"`
	HHADMarketD float64        `json:"hhadMarketD"`
	HHADMarketA float64        `json:"hhadMarketA"`
	Over        float64        `json:"over"`
	Under       float64        `json:"under"`
	MarketOver  float64        `json:"marketOver"`
	MarketUnder float64        `json:"marketUnder"`
}

func BoardFromJSON(oddsJSON string) *Board {
	if oddsJSON == "" {
		return nil
	}
	var raw struct {
		HAD  sporttery.Odds `json:"had"`
		TTG  [8]float64     `json:"ttg"`
		HHAD struct {
			Line string         `json:"line"`
			Odds sporttery.Odds `json:"odds"`
		} `json:"hhad"`
	}
	if err := json.Unmarshal([]byte(oddsJSON), &raw); err != nil {
		return nil
	}
	b := &Board{HAD: raw.HAD, HHAD: raw.HHAD.Odds, HHADLine: raw.HHAD.Line}
	b.HasHAD = raw.HAD.H > 1 && raw.HAD.D > 1 && raw.HAD.A > 1
	if raw.HAD.H > 1 && raw.HAD.D > 1 && raw.HAD.A > 1 {
		h, d, a := dewater3(raw.HAD.H, raw.HAD.D, raw.HAD.A)
		b.MarketH, b.MarketD, b.MarketA = round1(h*100), round1(d*100), round1(a*100)
	}
	if raw.HHAD.Line != "" {
		if line, ok := parseLine(raw.HHAD.Line); ok {
			b.HHADText = lineText(line)
		} else {
			b.HHADText = raw.HHAD.Line
		}
	}
	if raw.HHAD.Odds.H > 1 && raw.HHAD.Odds.D > 1 && raw.HHAD.Odds.A > 1 {
		h, d, a := dewater3(raw.HHAD.Odds.H, raw.HHAD.Odds.D, raw.HHAD.Odds.A)
		b.HHADMarketH, b.HHADMarketD, b.HHADMarketA = round1(h*100), round1(d*100), round1(a*100)
	}
	under, over := ttgOdds(raw.TTG)
	b.Under, b.Over = round2(under), round2(over)
	iu, io := ttgImplied(raw.TTG)
	b.MarketUnder, b.MarketOver = round1(iu*100), round1(io*100)
	if b.HAD.H <= 1 && b.Over <= 1 && b.HHADLine == "" {
		return nil
	}
	return b
}

func FillSides(dst, src []Side) []Side {
	if len(dst) == 0 {
		return src
	}
	idx := map[string]Side{}
	for _, s := range src {
		idx[s.Key] = s
	}
	out := make([]Side, len(dst))
	for i, s := range dst {
		if f, ok := idx[s.Key]; ok {
			if s.Model == 0 {
				s.Model = f.Model
			}
			if s.Market == 0 {
				s.Market = f.Market
			}
			if s.Odds == 0 {
				s.Odds = f.Odds
			}
			if s.ValueBand == "" {
				s.ValueBand = f.ValueBand
			}
		}
		out[i] = s
	}
	return out
}

func side(key, label string, modelPct, marketPct, odds float64) Side {
	k := round2(odds * modelPct / 100)
	return Side{
		Key:       key,
		Label:     label,
		Model:     round1(modelPct),
		Market:    round1(marketPct),
		Odds:      round2(odds),
		Value:     round1(modelPct - marketPct),
		ValueBand: ValueBand(round1(modelPct - marketPct)),
		Kelly:     k,
		KellyBand: KellyBand(k),
	}
}

func dewater3(h, d, a float64) (float64, float64, float64) {
	ih, id, ia := 1/h, 1/d, 1/a
	s := ih + id + ia
	if s <= 0 {
		return 0, 0, 0
	}
	return ih / s, id / s, ia / s
}

func ttgOdds(ttg [8]float64) (under, over float64) {
	var iu, io float64
	for i, o := range ttg {
		if o <= 1 {
			continue
		}
		if i <= 2 {
			iu += 1 / o
		} else {
			io += 1 / o
		}
	}
	if iu > 0 {
		under = 1 / iu
	}
	if io > 0 {
		over = 1 / io
	}
	return
}

func ttgImplied(ttg [8]float64) (under, over float64) {
	u, o := ttgOdds(ttg)
	if u <= 1 || o <= 1 {
		return 0, 0
	}
	iu, io := 1/u, 1/o
	s := iu + io
	return iu / s, io / s
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
