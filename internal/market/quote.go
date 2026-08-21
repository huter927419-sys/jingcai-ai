package market

import "time"

const ThinVolume = 50000

type Trio struct {
	Company string  `json:"company,omitempty"`
	H       float64 `json:"h"`
	D       float64 `json:"d"`
	A       float64 `json:"a"`
	H0      float64 `json:"h0,omitempty"`
	D0      float64 `json:"d0,omitempty"`
	A0      float64 `json:"a0,omitempty"`
	PH      float64 `json:"pH"`
	PD      float64 `json:"pD"`
	PA      float64 `json:"pA"`
}

type Handicap struct {
	Company  string  `json:"company,omitempty"`
	Line     string  `json:"line"`
	LineNum  float64 `json:"lineNum"`
	Home     float64 `json:"home"`
	Away     float64 `json:"away"`
	Line0    string  `json:"line0,omitempty"`
	LineNum0 float64 `json:"lineNum0,omitempty"`
	Home0    float64 `json:"home0,omitempty"`
	Away0    float64 `json:"away0,omitempty"`
	PH       float64 `json:"pH"`
	PA       float64 `json:"pA"`
}

type OU struct {
	Company string  `json:"company,omitempty"`
	Line    float64 `json:"line"`
	Over    float64 `json:"over"`
	Under   float64 `json:"under"`
	Line0   float64 `json:"line0,omitempty"`
	Over0   float64 `json:"over0,omitempty"`
	Under0  float64 `json:"under0,omitempty"`
	PO      float64 `json:"pO"`
	PU      float64 `json:"pU"`
}

type Betfair struct {
	HomeVol float64 `json:"homeVol"`
	DrawVol float64 `json:"drawVol"`
	AwayVol float64 `json:"awayVol"`
	Total   float64 `json:"total"`
	Thin    bool    `json:"thin"`
	Note    string  `json:"note,omitempty"`
}

type Quote struct {
	MatchID   int64     `json:"matchId"`
	Fid       int64     `json:"fid,omitempty"`
	FetchedAt time.Time `json:"fetchedAt"`
	Company   string    `json:"company,omitempty"`
	EU        *Trio     `json:"eu,omitempty"`
	Asian     *Handicap `json:"asian,omitempty"`
	OU        *OU       `json:"ou,omitempty"`
	Betfair   *Betfair  `json:"betfair,omitempty"`
}

func (q *Quote) FillImplied() {
	if q == nil {
		return
	}
	if q.EU != nil && q.EU.H > 1 && q.EU.D > 1 && q.EU.A > 1 {
		h, d, a := dewater3(q.EU.H, q.EU.D, q.EU.A)
		q.EU.PH, q.EU.PD, q.EU.PA = round1(h*100), round1(d*100), round1(a*100)
		if q.EU.Company == "" {
			q.EU.Company = q.Company
		}
	}
	if q.Asian != nil && q.Asian.Home > 0 && q.Asian.Away > 0 {
		h, a := dewater2(hkDecimal(q.Asian.Home), hkDecimal(q.Asian.Away))
		q.Asian.PH, q.Asian.PA = round1(h*100), round1(a*100)
		if q.Asian.Company == "" {
			q.Asian.Company = q.Company
		}
	}
	if q.OU != nil && q.OU.Over > 0 && q.OU.Under > 0 {
		o, u := dewater2(hkDecimal(q.OU.Over), hkDecimal(q.OU.Under))
		q.OU.PO, q.OU.PU = round1(o*100), round1(u*100)
		if q.OU.Company == "" {
			q.OU.Company = q.Company
		}
	}
	if q.Betfair != nil {
		q.Betfair.Total = q.Betfair.HomeVol + q.Betfair.DrawVol + q.Betfair.AwayVol
		q.Betfair.Thin = q.Betfair.Total > 0 && q.Betfair.Total < ThinVolume
		if q.Betfair.Thin && q.Betfair.Note == "" {
			q.Betfair.Note = "样本偏小"
		}
	}
}

func Drift(from, to float64) string {
	if from <= 0 || to <= 0 {
		return ""
	}
	if to < from-0.015 {
		return "降"
	}
	if to > from+0.015 {
		return "升"
	}
	return "平"
}

func hkDecimal(w float64) float64 {
	if w <= 0 {
		return 0
	}
	if w < 1.01 {
		return 1 + w
	}
	return w
}

func dewater3(h, d, a float64) (float64, float64, float64) {
	ih, id, ia := 1/h, 1/d, 1/a
	s := ih + id + ia
	if s <= 0 {
		return 0, 0, 0
	}
	return ih / s, id / s, ia / s
}

func dewater2(h, a float64) (float64, float64) {
	if h <= 1 || a <= 1 {
		return 0, 0
	}
	ih, ia := 1/h, 1/a
	s := ih + ia
	return ih / s, ia / s
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
