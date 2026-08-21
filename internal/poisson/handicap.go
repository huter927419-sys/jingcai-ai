package poisson

func Handicap(lambdaHome, lambdaAway, line float64) (home, draw, away float64) {
	raw, tot := dcMatrix(lambdaHome, lambdaAway)
	if tot <= 0 {
		tot = 1
	}
	for h := 0; h <= MaxGoals; h++ {
		for a := 0; a <= MaxGoals; a++ {
			p := raw[h][a] / tot
			adj := float64(h-a) + line
			switch {
			case adj > 0.05:
				home += p
			case adj < -0.05:
				away += p
			default:
				draw += p
			}
		}
	}
	return pct1(home * 100), pct1(draw * 100), pct1(away * 100)
}

func Asian(lambdaHome, lambdaAway, line float64) (home, away float64) {
	parts := splitLine(line)
	var h, a float64
	for _, p := range parts {
		hh, aa := asianOne(lambdaHome, lambdaAway, p)
		h += hh
		a += aa
	}
	n := float64(len(parts))
	if n == 0 {
		n = 1
	}
	return pct1(h / n * 100), pct1(a / n * 100)
}

func asianOne(lambdaHome, lambdaAway, line float64) (home, away float64) {
	raw, tot := dcMatrix(lambdaHome, lambdaAway)
	if tot <= 0 {
		tot = 1
	}
	for h := 0; h <= MaxGoals; h++ {
		for a := 0; a <= MaxGoals; a++ {
			p := raw[h][a] / tot
			adj := float64(h-a) + line
			switch {
			case adj > 0.05:
				home += p
			case adj < -0.05:
				away += p
			default:
				home += p / 2
				away += p / 2
			}
		}
	}
	return home, away
}

func OverUnder(lambdaHome, lambdaAway, line float64) (over, under float64) {
	parts := splitLine(line)
	var o, u float64
	for _, p := range parts {
		oo, uu := overUnderOne(lambdaHome, lambdaAway, p)
		o += oo
		u += uu
	}
	n := float64(len(parts))
	if n == 0 {
		n = 1
	}
	return pct1(o / n * 100), pct1(u / n * 100)
}

func overUnderOne(lambdaHome, lambdaAway, line float64) (over, under float64) {
	raw, tot := dcMatrix(lambdaHome, lambdaAway)
	if tot <= 0 {
		tot = 1
	}
	for h := 0; h <= MaxGoals; h++ {
		for a := 0; a <= MaxGoals; a++ {
			p := raw[h][a] / tot
			totg := float64(h + a)
			switch {
			case totg > line+0.05:
				over += p
			case totg < line-0.05:
				under += p
			default:
				over += p / 2
				under += p / 2
			}
		}
	}
	return over, under
}

func splitLine(line float64) []float64 {
	neg := line < 0
	abs := line
	if abs < 0 {
		abs = -abs
	}
	whole := float64(int(abs))
	frac := abs - whole
	var a, b float64
	switch {
	case frac > 0.2 && frac < 0.3:
		a, b = whole, whole+0.5
	case frac > 0.7 && frac < 0.8:
		a, b = whole+0.5, whole+1
	default:
		return []float64{line}
	}
	if neg {
		return []float64{-a, -b}
	}
	return []float64{a, b}
}

func dcMatrix(lambdaHome, lambdaAway float64) ([][]float64, float64) {
	if lambdaHome < 0.15 {
		lambdaHome = 0.15
	}
	if lambdaAway < 0.15 {
		lambdaAway = 0.15
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
	return raw, tot
}
