package market

import "testing"

func TestParseIDMap(t *testing.T) {
	html := `<tr data-fixtureid="1464482" data-id="2040954"></tr>
<tr data-id="2040934" data-fixtureid="1467681"></tr>`
	m := ParseIDMap(html)
	if m[2040954] != 1464482 || m[2040934] != 1467681 {
		t.Fatalf("%v", m)
	}
}

func TestParseEU(t *testing.T) {
	html := `ouzhi_same.php?cid=293&win=1.15&draw=7.00&lost=13.00
ouzhi_same.php?cid=5&win=1.08&draw=6.30&lost=11.00
ouzhi_same.php?cid=3&win=1.61&draw=3.70&lost=5.00
ouzhi_same.php?cid=2&win=1.18&draw=8.00&lost=12.00`
	eu := ParseEU(html)
	if eu == nil || eu.H != 1.61 || eu.D != 3.70 || eu.A != 5.00 || eu.Company != "Bet365" {
		t.Fatalf("%+v", eu)
	}
}

func TestParseAsianAndOU(t *testing.T) {
	yazhi := `<tr class="tr2" id="5"><td>x</td>
<td class="ying">0.900↑</td>
<td onclick="javascript:openPl(this);" ref="-2.250">两球/两球半<font color="red"> 升</font></td>
<td class="ping">0.800↓</td></tr>
<tr class="tr2" id="3"><td>x</td>
<td class="ying">0.950↑</td>
<td onclick="javascript:openPl(this);" ref="-1.000">一球</td>
<td class="ping">0.850↓</td></tr>`
	a := ParseAsian(yazhi)
	if a == nil || a.Home != 0.95 || a.Away != 0.85 || a.LineNum != -1 || a.Company != "Bet365" {
		t.Fatalf("%+v", a)
	}
	daxiao := `<tr class="tr2" id="5">
<td>0.83</td>
<td onclick="javascript:openPl(this);" ref="-3.50" class="tb_tdul_pan ">3.5</td>
<td>0.79</td></tr>
<tr class="tr2" id="3">
<td>0.90</td>
<td onclick="javascript:openPl(this);" ref="-2.50" class="tb_tdul_pan ">2.5</td>
<td>0.88</td></tr>`
	ou := ParseOU(daxiao)
	if ou == nil || ou.Line != 2.5 || ou.Over != 0.90 || ou.Under != 0.88 || ou.Company != "Bet365" {
		t.Fatalf("%+v", ou)
	}
}

func TestParseFTScores(t *testing.T) {
	html := `<td align="center"><div class="pk"><a href="./detail.php?fid=1367802" target="_blank" class="clt1" >3</a><span>-</span><a href="./detail.php?fid=1367802" target="_blank" class="clt3" >1</a></div></td>`
	m := ParseFTScores(html)
	if m[1367802] != [2]int{3, 1} {
		t.Fatalf("%v", m)
	}
}

func TestParseBetfair(t *testing.T) {
	html := `总交易[港币]</span><div class="data-detail"><ul><li>187,732</li><li>12,397</li><li>16,783</li></ul>`
	bf := ParseBetfair(html)
	if bf == nil || bf.HomeVol != 187732 || bf.DrawVol != 12397 || bf.AwayVol != 16783 {
		t.Fatalf("%+v", bf)
	}
	q := &Quote{Betfair: bf}
	q.FillImplied()
	if q.Betfair.Thin {
		t.Fatal("should not be thin")
	}
}
