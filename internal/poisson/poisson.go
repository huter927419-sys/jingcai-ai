package poisson

import "math"

const (
	Rho      = -0.13
	MaxGoals = 8
	GridMax  = 4
)

type ScoreProb struct {
	Home  int     `json:"home"`
	Away  int     `json:"away"`
	Score string  `json:"score"`
	P     float64 `json:"p"`
}

type Result struct {
	HomeWin float64     `json:"homeWin"`
	Draw    float64     `json:"draw"`
	AwayWin float64     `json:"awayWin"`
	Over25  float64     `json:"over25"`
	Under25 float64     `json:"under25"`
	Top     []ScoreProb `json:"topScores"`
	Grid    [][]float64 `json:"grid"`
}

func Evaluate(lambdaHome, lambdaAway float64) Result {
	if lambdaHome < 0.15 {
		lambdaHome = 0.15
	}
	if lambdaAway < 0.15 {
		lambdaAway = 0.15
	}
	if lambdaHome > 4.8 {
		lambdaHome = 4.8
	}
	if lambdaAway > 4.8 {
		lambdaAway = 4.8
	}

	raw := make([][]float64, MaxGoals+1)
	var tot float64
	for h := 0; h <= MaxGoals; h++ {
		raw[h] = make([]float64, MaxGoals+1)
		ph := pmf(h, lambdaHome)
		for a := 0; a <= MaxGoals; a++ {
			p := ph * pmf(a, lambdaAway) * tau(h, a, lambdaHome, lambdaAway, Rho)
			raw[h][a] = p
			tot += p
		}
	}
	if tot <= 0 {
		tot = 1
	}

	var homeWin, draw, awayWin, over float64
	scores := make([]ScoreProb, 0, (MaxGoals+1)*(MaxGoals+1))
	for h := 0; h <= MaxGoals; h++ {
		for a := 0; a <= MaxGoals; a++ {
			p := raw[h][a] / tot
			raw[h][a] = p
			switch {
			case h > a:
				homeWin += p
			case h == a:
				draw += p
			default:
				awayWin += p
			}
			if h+a >= 3 {
				over += p
			}
			scores = append(scores, ScoreProb{
				Home:  h,
				Away:  a,
				Score: itoa(h) + "-" + itoa(a),
				P:     pct1(p * 100),
			})
		}
	}

	sortScores(scores)
	topN := 5
	if len(scores) < topN {
		topN = len(scores)
	}

	grid := make([][]float64, GridMax+1)
	for h := 0; h <= GridMax; h++ {
		grid[h] = make([]float64, GridMax+1)
		for a := 0; a <= GridMax; a++ {
			grid[h][a] = pct1(raw[h][a] * 100)
		}
	}

	return Result{
		HomeWin: pct1(homeWin * 100),
		Draw:    pct1(draw * 100),
		AwayWin: pct1(awayWin * 100),
		Over25:  pct1(over * 100),
		Under25: pct1((1 - over) * 100),
		Top:     scores[:topN],
		Grid:    grid,
	}
}

func pmf(k int, lambda float64) float64 {
	return math.Exp(-lambda) * math.Pow(lambda, float64(k)) / factorial(k)
}

func factorial(n int) float64 {
	x := 1.0
	for i := 2; i <= n; i++ {
		x *= float64(i)
	}
	return x
}

func tau(h, a int, lh, la, rho float64) float64 {
	switch {
	case h == 0 && a == 0:
		return 1 - lh*la*rho
	case h == 0 && a == 1:
		return 1 + lh*rho
	case h == 1 && a == 0:
		return 1 + la*rho
	case h == 1 && a == 1:
		return 1 - rho
	default:
		return 1
	}
}

func pct1(v float64) float64 {
	return math.Round(v*10) / 10
}

func itoa(n int) string {
	if n >= 0 && n < 10 {
		return string(rune('0' + n))
	}
	return "10"
}

func sortScores(s []ScoreProb) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && (s[j].P > s[j-1].P || (s[j].P == s[j-1].P && s[j].Score < s[j-1].Score)) {
			s[j], s[j-1] = s[j-1], s[j]
			j--
		}
	}
}
