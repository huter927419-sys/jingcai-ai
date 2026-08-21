package sporttery

import "testing"

func TestParseSample(t *testing.T) {
	body := []byte(`{
	  "success": true,
	  "value": {
	    "matchInfoList": [{
	      "businessDate": "2026-08-20",
	      "subMatchList": [{
	        "matchId": 2040935,
	        "matchNumStr": "周四002",
	        "leagueAllName": "西班牙甲级联赛",
	        "leagueAbbName": "西甲",
	        "homeTeamAllName": "巴列卡诺",
	        "awayTeamAllName": "阿拉维斯",
	        "homeTeamAbbName": "巴列卡诺",
	        "awayTeamAbbName": "阿拉维斯",
	        "matchDate": "2026-08-21",
	        "matchTime": "03:00:00",
	        "matchStatus": "Selling",
	        "had": {"h":"2.03","d":"2.84","a":"3.52"},
	        "hhad": {"h":"1.50","d":"3.80","a":"4.50","goalLine":"-1"},
	        "ttg": {"s0":"6.90","s1":"3.55","s2":"3.05","s3":"4.25","s4":"8.25","s5":"20.00","s6":"40.00","s7":"80.00"}
	      }]
	    }]
	  }
	}`)
	ms, err := Parse(body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("len %d", len(ms))
	}
	m := ms[0]
	if m.ID != 2040935 || m.NumStr != "周四002" || !m.HasHAD || m.HAD.H != 2.03 {
		t.Fatalf("%+v", m)
	}
	if !m.HasTTG || m.TTG[0] != 6.9 {
		t.Fatalf("ttg %+v", m.TTG)
	}
	if m.Kickoff.Hour() != 3 {
		t.Fatalf("kickoff %v", m.Kickoff)
	}
}
