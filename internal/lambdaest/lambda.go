package lambdaest

import (
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"
	"math"
	"strconv"
	"strings"
)

type Seed struct {
	Home, Away float64
	ImpliedH   float64
	ImpliedD   float64
	ImpliedA   float64
	ImpliedO25 float64
}

func FromMatch(m sporttery.Match) Seed {
	var ih, id, ia, io float64
	if m.HasHAD {
		p := dewater([]float64{m.HAD.H, m.HAD.D, m.HAD.A})
		if len(p) == 3 {
			ih, id, ia = p[0], p[1], p[2]
		}
	}
	if m.HasTTG {
		under := []float64{}
		over := []float64{}
		for i, o := range m.TTG {
			if o <= 1 {
				continue
			}
			if i <= 2 {
				under = append(under, o)
			} else {
				over = append(over, o)
			}
		}
		pu := invSum(under)
		po := invSum(over)
		if pu+po > 0 {
			io = po / (pu + po)
		}
	}
	var diff float64
	hasDiff := false
	if !m.HasHAD && m.HasHHAD {
		if d, ok := goalDiffFromLine(m.HHADLine); ok {
			diff, hasDiff = d, true
		}
	}
	lh, la := search(ih, id, ia, io, diff, hasDiff)
	return Seed{Home: lh, Away: la, ImpliedH: ih, ImpliedD: id, ImpliedA: ia, ImpliedO25: io}
}

func goalDiffFromLine(line string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
	if err != nil {
		return 0, false
	}
	return -f, true
}

func search(ih, id, ia, io, diff float64, hasDiff bool) (float64, float64) {
	bestLH, bestLA := 1.25, 1.10
	best := 1e9
	for i := 8; i <= 70; i++ {
		lh := float64(i) / 20
		for j := 8; j <= 70; j++ {
			la := float64(j) / 20
			r := poisson.Evaluate(lh, la)
			err := 0.0
			if ih+id+ia > 0.5 {
				err += math.Abs(r.HomeWin/100-ih)*1.2 + math.Abs(r.Draw/100-id) + math.Abs(r.AwayWin/100-ia)*1.2
			}
			if io > 0.05 {
				err += math.Abs(r.Over25/100-io) * 1.4
			}
			if hasDiff {
				err += math.Abs((lh-la)-diff) * 1.8
			}
			if err < best {
				best = err
				bestLH, bestLA = lh, la
			}
		}
	}
	return bestLH, bestLA
}

func dewater(odds []float64) []float64 {
	inv := make([]float64, len(odds))
	var s float64
	for i, o := range odds {
		if o <= 1 {
			return nil
		}
		inv[i] = 1 / o
		s += inv[i]
	}
	if s == 0 {
		return nil
	}
	for i := range inv {
		inv[i] /= s
	}
	return inv
}

func invSum(odds []float64) float64 {
	var s float64
	for _, o := range odds {
		if o > 1 {
			s += 1 / o
		}
	}
	return s
}
