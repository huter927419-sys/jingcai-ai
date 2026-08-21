package market

import "testing"

func TestParsePreview(t *testing.T) {
	html := `
<div class="M_box starting">
  <div class="team_a">
    <div class="team_name">巴列卡诺阵型:&nbsp;</div>
    <table>
<tr><td class="td_one"><span class="td_sp3">11</span>兰迪·恩特卡(前锋)</td><td><span class="td_sp3">10</span>塞吉奥·卡梅略(前锋)</td></tr>
<tr><td class="td_one"><span class="td_sp3">18</span>阿尔瓦罗·加西亚(中场)</td><td><span class="td_sp3">9</span>亚历山大(前锋)</td></tr>
<tr><td class="td_one"><span class="td_sp3">20</span>伊万·巴利乌(后卫)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">6</span>帕特·西塞(后卫)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">24</span>弗洛里安(后卫)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">2</span>安德烈(后卫)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">8</span>乌奈(中场)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">7</span>伊西(中场)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">19</span>豪尔赫(中场)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">23</span>奥斯卡(中场)</td><td><span class="td_sp3"></span></td></tr>
<tr><td class="td_one"><span class="td_sp3">13</span>奥古斯托(守门员)</td><td><span class="td_sp3"></span></td></tr>
          <tr><th>- 伤病 -</th><th>- 停赛 -</th></tr>
    </table>
  </div>
  <div class="team_b">
    <div class="team_name">阿拉维斯阵型:&nbsp;4-4-2</div>
    <table>
<tr><td class="td_one"><span class="td_sp3">20</span>艾托尔(前锋)</td><td><span class="td_sp3">31</span>格雷瓜尔(守门员)</td></tr>
<tr><td class="td_one"><span class="td_sp3">1</span>安东尼奥(守门员)</td><td><span class="td_sp3"></span></td></tr>
    </table>
  </div>
  <div class="clearb"></div>
</div>
<div id="team_zhanji_1">
<table class="pub_table"><tbody>
<tr class="tr3 bmatch"><td></td><td>26-08-21</td><td><em>VS</em></td></tr>
<tr class="tr1" ><td class="td_one"><a href="https://liansai.500.com/zuqiu-19947/" target="_blank" title="西班牙甲级联赛" rel="nofollow" >西甲</a></td><td>26-08-16</td><td class="dz"><a href="./shuju-1.shtml"><span class="dz-l ">塞维利亚</span><em>2:<span class="shu">1</span></em><span class="dz-r zhu">巴列卡诺</span></a></td><td>0</td><td>0:1</td><td><span class="shu">负</span></td></tr>
<tr class="tr2" ><td class="td_one"><a rel="nofollow" >球会友谊</a></td><td>26-08-08</td><td class="dz"><span class="dz-l ">伊普斯维奇</span><em>3:<span class="shu">0</span></em><span class="dz-r zhu">巴列卡诺</span></td><td></td><td></td><td><span class="shu">负</span></td></tr>
</tbody></table>
</div>
<div id="team_zhanji_0">
<table class="pub_table"><tbody>
<tr class="tr1" ><td class="td_one"><a rel="nofollow" >西甲</a></td><td>26-08-15</td><td class="dz"><span class="dz-l ">阿拉维斯</span><em>3:<span class="ying">0</span></em><span class="dz-r ">赫塔费</span></td><td></td><td></td><td><span class="ying">胜</span></td></tr>
</tbody></table>
</div>
`
	p := ParsePreview(html)
	if p == nil {
		t.Fatal("nil")
	}
	if p.Home.Name != "巴列卡诺" || p.Home.Formation != "4-5-1" {
		t.Fatalf("home %+v", p.Home)
	}
	if len(p.Home.Starters) != 11 || p.Home.Starters[0].Name != "兰迪·恩特卡" || p.Home.Starters[0].Pos != "前锋" {
		t.Fatalf("starters %+v", p.Home.Starters)
	}
	if len(p.Home.Bench) < 1 || p.Home.Bench[0].Name != "塞吉奥·卡梅略" {
		t.Fatalf("bench %+v", p.Home.Bench)
	}
	if p.Away.Formation != "4-4-2" {
		t.Fatalf("away formation %q", p.Away.Formation)
	}
	if len(p.Home.Form) != 2 || p.Home.Form[0].Result != "负" || p.Home.Form[0].Score != "2:1" {
		t.Fatalf("form %+v", p.Home.Form)
	}
	if p.Home.Form[0].Rating <= 0 || p.Away.Form[0].Rating < 8 {
		t.Fatalf("rating home=%v away=%v", p.Home.Form[0].Rating, p.Away.Form[0].Rating)
	}
}

func TestMatchRating(t *testing.T) {
	if matchRating("胜", "3:0") < 8 {
		t.Fatal(matchRating("胜", "3:0"))
	}
	if matchRating("负", "2:1") >= 5 {
		t.Fatal(matchRating("负", "2:1"))
	}
}
