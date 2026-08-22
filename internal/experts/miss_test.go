package experts

import "testing"

func TestClassifyMissAllWrong(t *testing.T) {
	takes := voices("胜", "胜", "胜")
	if g := ClassifyMiss(takes, 0, 1); g != MissAll {
		t.Fatalf("got %q", g)
	}
}

func TestClassifyMissConsensusWrong(t *testing.T) {
	takes := voices("胜", "胜", "胜", "平")
	if g := ClassifyMiss(takes, 1, 1); g != MissHAD {
		t.Fatalf("got %q", g)
	}
}

func TestClassifyMissSkipWhenMajorityHits(t *testing.T) {
	takes := voices("胜", "胜", "胜", "负")
	if g := ClassifyMiss(takes, 2, 0); g != MissNone {
		t.Fatalf("got %q", g)
	}
}

func TestClassifyMissSkipSplit(t *testing.T) {
	takes := voices("胜", "胜", "负", "负")
	if g := ClassifyMiss(takes, 1, 0); g != MissNone {
		t.Fatalf("got %q", g)
	}
}
