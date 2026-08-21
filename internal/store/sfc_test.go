package store

import "testing"

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
