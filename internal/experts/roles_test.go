package experts

import (
	"strings"
	"testing"

	"jingcai-ai/internal/store"
)

func storeTake(name string, h, d, a, o, u float64, p1, pou, v string) store.ModelTake {
	return store.ModelTake{
		Name: name, HomeWin: h, Draw: d, AwayWin: a, Over25: o, Under25: u,
		Pick1X2: p1, PickOU: pou, Verdict: v,
	}
}

func TestNormAndGrade(t *testing.T) {
	if Norm1X2("home") != "胜" || NormOU("over") != "大" || NormVerdict("主推") != "主推" {
		t.Fatal("norm")
	}
	take := storeTake("DeepSeek", 50, 25, 25, 60, 40, "胜", "大", "主推")
	g := GradeTake(take, 2, 1)
	if !g.Hit1X2 || !g.HitOU || g.Points != 3 {
		t.Fatalf("%+v", g)
	}
	miss := GradeTake(take, 0, 1)
	if miss.Hit1X2 || miss.Points != -1 {
		t.Fatalf("miss %+v", miss)
	}
}

func TestFillPicksFromShape(t *testing.T) {
	take := storeTake("Grok", 20, 22, 58, 30, 70, "", "", "")
	Decorate(&take)
	if take.Role != "盘口分析师" || take.Pick1X2 != "负" || take.PickOU != "小" || take.Verdict != "可看" {
		t.Fatalf("%+v", take)
	}
	if !strings.HasPrefix(take.BuyTalk, "参考买入：") || !strings.Contains(take.BuyTalk, "临场") {
		t.Fatalf("advice missing reference or invalidation condition: %q", take.BuyTalk)
	}
}

func TestBoardOrder(t *testing.T) {
	out := Board([]BoardRow{
		{Name: "Claude", Games: 4, Hit1X2: 1, HitOU: 2, Points: 3},
		{Name: "DeepSeek", Games: 4, Hit1X2: 3, HitOU: 2, Points: 8},
		{Name: "Grok", Games: 0},
	})
	if out[0].Name != "DeepSeek" || out[1].Name != "Claude" {
		t.Fatalf("%+v", out)
	}
}
