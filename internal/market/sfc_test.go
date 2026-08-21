package market

import "testing"

func TestParseSFC(t *testing.T) {
	html := `
expect=26108
<tr class="bet-tb-tr" data-cid="1" data-vs="阿森纳vs考文垂" data-asian="1.02,两球,0.82" data-pjgl="79.81,13.69,6.50">
<td class="td td-no">1</td><td class="td td-evt"><a href="#">英超</a></td>
<td class="td td-endtime">08-22 03:00</td>
<a href="https://odds.500.com/fenxi/shuju-1420315.shtml">析</a>
</tr>
<tr class="bet-tb-tr tr-even" data-cid="2" data-vs="贝蒂斯vs社会" data-bjpl="1.87,3.26,3.47" data-asian="0.90,半球,0.90" data-pjgl="45.60,27.28,27.12">
<td class="td td-no">2</td><td class="td td-evt"><a href="#">西甲</a></td>
<td class="td td-endtime">08-22 03:00</td>
<a href="https://odds.500.com/fenxi/shuju-1427950.shtml">析</a>
</tr>`
	board := ParseSFC(html)
	if board == nil || board.Issue != "26108" || len(board.Matches) != 2 {
		t.Fatalf("%+v", board)
	}
	a := board.Matches[0]
	if a.No != 1 || a.Home != "阿森纳" || a.Away != "考文垂" || a.Fid != 1420315 || a.MarketHome < 79 {
		t.Fatalf("%+v", a)
	}
	if board.Matches[1].League != "西甲" || board.Matches[1].EUHome < 1.8 {
		t.Fatalf("%+v", board.Matches[1])
	}
}
