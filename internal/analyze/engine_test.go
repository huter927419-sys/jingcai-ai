package analyze

import (
	"testing"
	"time"

	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

func TestRunSkipsExistingSnapshot(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	e := &Engine{Store: st}
	m := sporttery.Match{
		ID: 1, NumStr: "周四001", Home: "主", Away: "客",
		Kickoff: time.Now().Add(2 * time.Hour), HasHAD: true,
		HAD:    sporttery.Odds{H: 2.03, D: 3.2, A: 3.5},
		HasTTG: true, TTG: [8]float64{8, 4, 3, 4, 8, 20, 40, 80},
	}
	first, err := e.Run(m, store.KindOpen)
	if err != nil || first.Skipped {
		t.Fatalf("first %+v %v", first, err)
	}
	second, err := e.Run(m, store.KindOpen)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Skipped {
		t.Fatal("expected sqlite hit, no second write")
	}
}
