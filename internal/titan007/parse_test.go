package titan007

import (
	"strings"
	"testing"

	"jingcai-ai/internal/market"
)

func TestParseEuropeanWhitelist(t *testing.T) {
	js := `var game=Array(
"281|rec365|Bet 365|1.61|3.7|5|38|16|12|96|1.48|4.2|6.5|40|14|11|95|1.1|1|0.9|08-21 18:00|Bet365(英国)|x|x|1.2|1.1|1.0",
"177|recpin|Pinnacle|1.60|4.16|5.78|40|15|12|96|1.45|4.35|7.13|41|14|11|95|1|1|1|08-21 18:00|平博|x|x|1|1|1",
"5|recmac|Macau|2|3|4|1|1|1|90|2|3|4|1|1|1|90|1|1|1|08-21 18:00|澳门|x|x|1|1|1"
);
var gameDetail=Array(
"rec365^1.61|3.7|5|08-20 10:00|1|1|1|2026;1.48|4.2|6.5|08-21 17:50|1|1|1|2026",
"recpin^1.60|4.16|5.78|08-12 21:00|1|1|1|2026;1.45|4.35|7.13|08-21 17:40|1|1|1|2026"
);`
	books := ParseEuropean(js)
	if len(books) != 2 {
		t.Fatalf("books %d", len(books))
	}
	if books[0].CompanyID != 281 || books[0].Current.H != 1.48 || books[0].Opening.H != 1.61 {
		t.Fatalf("bet365 %+v", books[0])
	}
	if len(books[0].Nodes) != 2 || !strings.HasPrefix(books[0].Nodes[0].Time, "2026-08-20") {
		t.Fatalf("nodes %+v", books[0].Nodes)
	}
}

func TestParseScheduleJingzu(t *testing.T) {
	html := `jinZuSchedule = "3000414,3000413|周五001,周五002";
<tr height=18 sId='3000414'><td>日职联</td><td>8-21 18:00</td><td> </td><td>[3]柏太阳神</td><td>-</td><td>长崎成功丸</td></tr>
<tr sId='9'><td>英超</td><td>8-21 19:00</td><td></td><td>阿森纳</td><td>-</td><td>利物浦</td></tr>`
	rows := ParseSchedule(html)
	if len(rows) != 1 || rows[0].NumStr != "周五001" || rows[0].Home != "柏太阳神" || rows[0].Away != "长崎成功丸" {
		t.Fatalf("%+v", rows)
	}
}

func TestParseChangeDetailSkipsInPlay(t *testing.T) {
	html := `<span id="odds2"><table>
<tr align=center><td>变化时间</td><td>比分</td><td>主</td><td>盘</td><td>客</td><td>变化时间</td><td>状态</td></tr>
<tr align=center><td>08-20 10:00</td><td></td><td>0.93</td><td>半球/一球</td><td>0.85</td><td>08-20 10:00</td><td>初</td></tr>
<tr align=center><td>08-21 17:50</td><td></td><td>0.80</td><td>一球</td><td>1.00</td><td>08-21 17:50</td><td>即</td></tr>
<tr align=center><td>84</td><td>4-2</td><td colspan='3'>封</td><td>8-21 19:52</td><td>滚</td></tr>
</table></span>`
	mv := ParseChangeDetail(html, "澳门")
	if mv == nil || mv.OpeningLine != "半球/一球" || mv.CurrentLine != "一球" || mv.CurrentLeft != 0.8 {
		t.Fatalf("%+v", mv)
	}
	if mv.NodeCount != 2 {
		t.Fatalf("nodes %d", mv.NodeCount)
	}
}

func TestApplyKeepsBet365Asian(t *testing.T) {
	q := &market.Quote{
		Asian: &market.Handicap{Company: "Bet365", Line: "-1", Home: 0.95, Away: 0.85},
		EU:    &market.Trio{Company: "Bet365", H: 1.5, D: 4, A: 6},
	}
	Apply(q, &Odds{
		ID: 3000414,
		Books: []market.EUBook{{
			CompanyID: 281,
			Company:   "Bet365",
			Opening:   &market.Trio{H: 1.61, D: 3.7, A: 5},
			Current:   &market.Trio{H: 1.48, D: 4.2, A: 6.5},
		}},
		Asian: &market.LineMove{Company: "澳门", CurrentLine: "一球", CurrentLeft: 0.8, CurrentRight: 1, NodeCount: 2},
	})
	if q.Asian.Company != "Bet365" || q.Asian.Line != "-1" {
		t.Fatalf("asian overwritten %+v", q.Asian)
	}
	if q.EU.H0 != 1.61 || q.TitanID != 3000414 || q.AsianMove == nil {
		t.Fatalf("merge %+v", q)
	}
}

func TestFindMatch(t *testing.T) {
	rows := []Match{{ID: 1, NumStr: "周五001", Home: "柏太阳神", Away: "长崎成功丸"}}
	if FindMatch(rows, "周五001", "x", "y").ID != 1 {
		t.Fatal("num")
	}
	if FindMatch(rows, "", "柏太阳神", "长崎成功丸").ID != 1 {
		t.Fatal("team")
	}
}

func TestParseSnapshotAsian(t *testing.T) {
	html := `<tr bgcolor="#FFFFFF">
<td width="35"><input type="checkbox" name="oddsShow" data-id="1" value="0"></td>
<td height="25">澳*</td>
<td><span class='down' companyID='1'></span></td>
<td title="2026-08-18 21:09">0.91</td>
<td title="2026-08-18 21:09" goals="-1.75">受让球半/两球</td>
<td title="2026-08-18 21:09">0.87</td>
<td oddstype="wholeLastOdds">0.88</td>
<td goals="-1.75" oddstype="wholeLastOdds">受让球半/两球</td>
<td oddstype="wholeLastOdds">0.90</td>
<td oddstype="wholeOdds">0.88</td>
<td goals="-1.75" oddstype="wholeOdds">受让球半/两球</td>
<td oddstype="wholeOdds">0.90</td>
</tr>
<tr bgcolor="#FFFFFF">
<td><input type="checkbox" name="oddsShow" data-id="3"></td>
<td height="25">Crow</td>
<td><span companyID='3'></span></td>
<td title="2026-08-18 21:00">0.95</td>
<td title="2026-08-18 21:00" goals="-1.5">受让球半</td>
<td title="2026-08-18 21:00">0.85</td>
<td oddstype="wholeOdds">0.90</td>
<td goals="-1.5" oddstype="wholeOdds">受让球半</td>
<td oddstype="wholeOdds">0.90</td>
</tr>
<tr style="display: none;" companyID=1><td colspan="9"></td></tr>`
	rows := ParseSnapshot(html, "asian")
	if len(rows) < 2 {
		t.Fatalf("rows %d %+v", len(rows), rows)
	}
	if rows[0].CompanyID != 1 || rows[0].OpeningLine != "-1.75" || rows[0].CurrentLeft != 0.88 {
		t.Fatalf("macau %+v", rows[0])
	}
	if rows[1].CompanyID != 3 || rows[1].Company != "Crown" {
		t.Fatalf("crown %+v", rows[1])
	}
}

func TestApplyAsianBooks(t *testing.T) {
	q := &market.Quote{Asian: &market.Handicap{Company: "Bet365", Line: "-1", Home: 0.95, Away: 0.85}}
	Apply(q, &Odds{
		ID:         1,
		AsianBooks: []market.LineMove{{CompanyID: 1, Company: "澳门", CurrentLine: "-1.5", CurrentLeft: 0.88, CurrentRight: 0.92}},
		OUBooks:    []market.LineMove{{CompanyID: 3, Company: "Crown", CurrentLine: "2.5", CurrentLeft: 0.9, CurrentRight: 0.9}},
	})
	if q.Asian.Company != "Bet365" {
		t.Fatalf("value asian %+v", q.Asian)
	}
	if len(q.AsianBooks) != 1 || q.OUBooks[0].Company != "Crown" {
		t.Fatalf("books %+v %+v", q.AsianBooks, q.OUBooks)
	}
}
