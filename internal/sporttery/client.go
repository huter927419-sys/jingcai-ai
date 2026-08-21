package sporttery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jingcai-ai/internal/fetchhttp"
)

const Endpoint = "https://webapi.sporttery.cn/gateway/jc/football/getMatchCalculatorV1.qry?poolCode=hhad,had,crs,ttg,hafu&channel=c"

type Odds struct {
	H, D, A float64
}

type Match struct {
	ID           int64
	NumStr       string
	League       string
	LeagueAbb    string
	Home         string
	Away         string
	HomeAbb      string
	AwayAbb      string
	Kickoff      time.Time
	BusinessDate string
	Status       string
	HAD          Odds
	HHAD         Odds
	HHADLine     string
	TTG          [8]float64
	HasHAD       bool
	HasTTG       bool
	HasHHAD      bool
}

type Client struct {
	HTTP *http.Client
}

func New(proxy string) *Client {
	return &Client{HTTP: fetchhttp.Client(25*time.Second, proxy)}
}

func (c *Client) Fetch(loc *time.Location) ([]Match, error) {
	req, err := http.NewRequest(http.MethodGet, Endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1")
	req.Header.Set("Referer", "https://m.sporttery.cn/")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("sporttery HTTP %d", res.StatusCode)
	}
	return Parse(body, loc)
}

type payload struct {
	Success bool `json:"success"`
	Value   struct {
		MatchInfoList []struct {
			BusinessDate string     `json:"businessDate"`
			SubMatchList []rawMatch `json:"subMatchList"`
		} `json:"matchInfoList"`
	} `json:"value"`
}

type rawMatch struct {
	MatchID         int64          `json:"matchId"`
	MatchNumStr     string         `json:"matchNumStr"`
	LeagueAllName   string         `json:"leagueAllName"`
	LeagueAbbName   string         `json:"leagueAbbName"`
	HomeTeamAllName string         `json:"homeTeamAllName"`
	AwayTeamAllName string         `json:"awayTeamAllName"`
	HomeTeamAbbName string         `json:"homeTeamAbbName"`
	AwayTeamAbbName string         `json:"awayTeamAbbName"`
	MatchDate       string         `json:"matchDate"`
	MatchTime       string         `json:"matchTime"`
	BusinessDate    string         `json:"businessDate"`
	MatchStatus     string         `json:"matchStatus"`
	HAD             map[string]any `json:"had"`
	HHAD            map[string]any `json:"hhad"`
	TTG             map[string]any `json:"ttg"`
}

func Parse(body []byte, loc *time.Location) ([]Match, error) {
	if loc == nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if !p.Success {
		return nil, fmt.Errorf("sporttery success=false")
	}
	out := make([]Match, 0, 32)
	for _, day := range p.Value.MatchInfoList {
		for _, raw := range day.SubMatchList {
			m, err := normalize(raw, day.BusinessDate, loc)
			if err != nil {
				continue
			}
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sporttery: no matches")
	}
	return out, nil
}

func normalize(raw rawMatch, business string, loc *time.Location) (Match, error) {
	if raw.MatchID == 0 {
		return Match{}, fmt.Errorf("no id")
	}
	kick := parseKick(raw.MatchDate, raw.MatchTime, loc)
	m := Match{
		ID:           raw.MatchID,
		NumStr:       raw.MatchNumStr,
		League:       raw.LeagueAllName,
		LeagueAbb:    raw.LeagueAbbName,
		Home:         raw.HomeTeamAllName,
		Away:         raw.AwayTeamAllName,
		HomeAbb:      raw.HomeTeamAbbName,
		AwayAbb:      raw.AwayTeamAbbName,
		Kickoff:      kick,
		BusinessDate: firstNonEmpty(raw.BusinessDate, business),
		Status:       raw.MatchStatus,
	}
	if h, d, a, ok := parseHAD(raw.HAD); ok {
		m.HAD = Odds{H: h, D: d, A: a}
		m.HasHAD = true
	}
	if h, d, a, ok := parseHAD(raw.HHAD); ok {
		m.HHAD = Odds{H: h, D: d, A: a}
		m.HHADLine = strVal(raw.HHAD, "goalLine")
		m.HasHHAD = m.HHADLine != ""
	}
	if ttg, ok := parseTTG(raw.TTG); ok {
		m.TTG = ttg
		m.HasTTG = true
	}
	return m, nil
}

func parseKick(date, clock string, loc *time.Location) time.Time {
	clock = strings.TrimSpace(clock)
	if len(clock) >= 8 {
		clock = clock[:8]
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", date+" "+clock, loc)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02 15:04", date+" "+clock, loc)
	}
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseHAD(m map[string]any) (h, d, a float64, ok bool) {
	h = parseFloat(m["h"])
	d = parseFloat(m["d"])
	a = parseFloat(m["a"])
	ok = h > 1 && d > 1 && a > 1
	return
}

func parseTTG(m map[string]any) ([8]float64, bool) {
	var ttg [8]float64
	n := 0
	for i := 0; i <= 7; i++ {
		v := parseFloat(m["s"+strconv.Itoa(i)])
		ttg[i] = v
		if v > 1 {
			n++
		}
	}
	return ttg, n >= 3
}

func parseFloat(v any) float64 {
	switch t := v.(type) {
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return 0
		}
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func strVal(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	s, _ := m[k].(string)
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
