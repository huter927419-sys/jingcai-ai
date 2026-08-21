package store

import (
	"testing"
	"time"

	"jingcai-ai/internal/market"
	"jingcai-ai/internal/sporttery"
)

func TestTeamClose(t *testing.T) {
	pairs := [][2]string{
		{"阿森纳", "阿森纳"},
		{"贝蒂斯", "皇家贝蒂斯"},
		{"社会", "皇家社会"},
		{"斯特堡", "斯特拉斯堡"},
		{"蒙彼利", "蒙彼利埃"},
		{"埃因FC", "埃因霍温FC"},
		{"塞伊奈", "塞伊奈约基"},
	}
	for _, p := range pairs {
		if !teamClose(p[0], p[1]) {
			t.Fatalf("%s ~ %s", p[0], p[1])
		}
	}
	if teamClose("阿森纳", "考文垂") {
		t.Fatal("false positive")
	}
}

func TestSFCMatchID(t *testing.T) {
	if got, want := SFCMatchID("26108", 4), int64(8_002_610_804); got != want {
		t.Fatalf("id %d want %d", got, want)
	}
}

func TestSFCMatchHiddenFromToday(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	kick := time.Now().Add(2 * time.Hour)
	jc := sporttery.Match{ID: 1, NumStr: "周五001", Home: "主", Away: "客", Kickoff: kick, BusinessDate: kick.Format("2006-01-02")}
	sfc := sporttery.Match{ID: SFCMatchID("26108", 4), NumStr: "胜负04", Home: "布洛涅", Away: "圣红星", Kickoff: kick, BusinessDate: kick.Format("2006-01-02")}
	if err := st.UpsertMatch(jc); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertSFCMatch(sfc); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListUpcoming(time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != 1 {
		t.Fatalf("upcoming %+v", list)
	}
	got, err := st.GetMatch(sfc.ID)
	if err != nil || got == nil || got.Origin != "sfc" {
		t.Fatalf("get sfc %+v %v", got, err)
	}
	q := &market.Quote{MatchID: sfc.ID, Fid: 99, EU: &market.Trio{H: 2.1, D: 3.2, A: 3.4}}
	if err := st.SaveQuote(q); err != nil {
		t.Fatal(err)
	}
}
