package poisson

import "testing"

func TestEvaluateMineiroSeed(t *testing.T) {
	r := Evaluate(1.36, 1.00)
	assertNear(t, "homeWin", r.HomeWin, 43.4, 0.6)
	assertNear(t, "draw", r.Draw, 30.7, 0.6)
	assertNear(t, "awayWin", r.AwayWin, 25.9, 0.6)
	assertNear(t, "over25", r.Over25, 42.0, 0.6)
	assertNear(t, "nolo", r.HomeWin+r.Draw, 74.1, 0.8)
	if len(r.Top) == 0 || r.Top[0].Score != "1-1" {
		t.Fatalf("mode want 1-1, got %+v", r.Top)
	}
	assertNear(t, "1-1", r.Top[0].P, 14.5, 0.4)
	if r.Grid[1][1] != r.Top[0].P {
		t.Fatalf("grid 1-1 %v != top %v", r.Grid[1][1], r.Top[0].P)
	}
}

func TestAsianHomeFavorite(t *testing.T) {
	h, a := Asian(2.2, 0.8, -0.5)
	if h <= a {
		t.Fatalf("home cover %v away %v", h, a)
	}
	o, u := OverUnder(1.4, 1.1, 2.5)
	assertNear(t, "ou", o+u, 100, 0.2)
}

func assertNear(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	d := got - want
	if d < 0 {
		d = -d
	}
	if d > tol {
		t.Fatalf("%s got %v want %v ±%v", name, got, want, tol)
	}
}
