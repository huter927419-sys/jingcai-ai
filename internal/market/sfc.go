package market

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const sfcURL = "https://trade.500.com/sfc/"

type SFCMatch struct {
	No           int     `json:"no"`
	League       string  `json:"league"`
	Kickoff      string  `json:"kickoff"`
	Home         string  `json:"home"`
	Away         string  `json:"away"`
	Fid          int64   `json:"fid,omitempty"`
	Asian        string  `json:"asian,omitempty"`
	MarketHome   float64 `json:"marketHome"`
	MarketDraw   float64 `json:"marketDraw"`
	MarketAway   float64 `json:"marketAway"`
	EUHome       float64 `json:"euHome,omitempty"`
	EUDraw       float64 `json:"euDraw,omitempty"`
	EUAway       float64 `json:"euAway,omitempty"`
	AnalyzedHome float64 `json:"analyzedHome,omitempty"`
	AnalyzedDraw float64 `json:"analyzedDraw,omitempty"`
	AnalyzedAway float64 `json:"analyzedAway,omitempty"`
	Quote        *Quote  `json:"quote,omitempty"`
}

type SFCBoard struct {
	Issue     string     `json:"issue"`
	FetchedAt time.Time  `json:"fetchedAt"`
	Matches   []SFCMatch `json:"matches"`
}

var (
	reSFCExpect = regexp.MustCompile(`expect=(\d{5})`)
	reSFCChunk  = regexp.MustCompile(`(?s)class="bet-tb-tr[^"]*"[^>]*>.*?</tr>`)
	reSFCVs     = regexp.MustCompile(`data-vs="([^"]+)"`)
	reSFCAsian  = regexp.MustCompile(`data-asian="([^"]*)"`)
	reSFCPjgl   = regexp.MustCompile(`data-pjgl="([^"]*)"`)
	reSFCBjpl   = regexp.MustCompile(`data-bjpl="([^"]*)"`)
	reSFCNo     = regexp.MustCompile(`td-no">(\d+)`)
	reSFCLeague = regexp.MustCompile(`td-evt"><a [^>]*>([^<]+)</a>`)
	reSFCTime   = regexp.MustCompile(`td-endtime">([^<]+)`)
	reSFCFid    = regexp.MustCompile(`shuju-(\d+)`)
)

func (c *Client) FetchSFC() (*SFCBoard, error) {
	html, err := c.get(sfcURL, "https://trade.500.com/")
	if err != nil {
		return nil, err
	}
	board := ParseSFC(html)
	if board == nil || len(board.Matches) == 0 {
		return nil, fmt.Errorf("500.com: empty 胜负彩 board")
	}
	board.FetchedAt = time.Now()
	return board, nil
}

func ParseSFC(html string) *SFCBoard {
	issue := ""
	if m := reSFCExpect.FindStringSubmatch(html); len(m) == 2 {
		issue = m[1]
	}
	chunks := reSFCChunk.FindAllString(html, -1)
	if len(chunks) == 0 {
		chunks = sfcWindows(html)
	}
	out := make([]SFCMatch, 0, 14)
	seen := map[int]bool{}
	for _, chunk := range chunks {
		row, ok := parseSFCChunk(chunk)
		if !ok || row.No < 1 || row.No > 14 || seen[row.No] {
			continue
		}
		seen[row.No] = true
		out = append(out, row)
	}
	if len(out) == 0 {
		return nil
	}
	sortSFC(out)
	return &SFCBoard{Issue: issue, Matches: out}
}

func sfcWindows(html string) []string {
	idxs := []int{}
	for i := 0; ; {
		j := strings.Index(html[i:], `data-vs="`)
		if j < 0 {
			break
		}
		idxs = append(idxs, i+j)
		i += j + 8
	}
	out := make([]string, 0, len(idxs))
	for _, idx := range idxs {
		from := idx - 80
		if from < 0 {
			from = 0
		}
		rest := html[idx:]
		end := strings.Index(strings.ToLower(rest), "</tr>")
		to := idx + len(rest)
		if end >= 0 {
			to = idx + end + 5
		} else if to > idx+1800 {
			to = idx + 1800
		}
		out = append(out, html[from:to])
	}
	return out
}

func parseSFCChunk(chunk string) (SFCMatch, bool) {
	vs := attr(reSFCVs, chunk)
	home, away, ok := splitVS(vs)
	if !ok {
		return SFCMatch{}, false
	}
	no, _ := strconv.Atoi(attr(reSFCNo, chunk))
	if no == 0 {
		if cid := regexp.MustCompile(`data-cid="(\d+)"`).FindStringSubmatch(chunk); len(cid) == 2 {
			no, _ = strconv.Atoi(cid[1])
		}
	}
	fid, _ := strconv.ParseInt(attr(reSFCFid, chunk), 10, 64)
	h, d, a := parsePjgl(attr(reSFCPjgl, chunk))
	eh, ed, ea := parsePjgl(attr(reSFCBjpl, chunk))
	return SFCMatch{
		No:         no,
		League:     strings.TrimSpace(attr(reSFCLeague, chunk)),
		Kickoff:    strings.TrimSpace(attr(reSFCTime, chunk)),
		Home:       home,
		Away:       away,
		Fid:        fid,
		Asian:      strings.TrimSpace(attr(reSFCAsian, chunk)),
		MarketHome: h,
		MarketDraw: d,
		MarketAway: a,
		EUHome:     eh,
		EUDraw:     ed,
		EUAway:     ea,
	}, true
}

func attr(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func splitVS(vs string) (string, string, bool) {
	vs = strings.TrimSpace(vs)
	i := strings.Index(strings.ToLower(vs), "vs")
	if i <= 0 {
		return "", "", false
	}
	home := strings.TrimSpace(vs[:i])
	away := strings.TrimSpace(vs[i+2:])
	return home, away, home != "" && away != ""
}

func parsePjgl(s string) (h, d, a float64) {
	parts := strings.Split(s, ",")
	if len(parts) < 3 {
		return
	}
	h, _ = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	d, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	a, _ = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	return
}

func sortSFC(rows []SFCMatch) {
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && rows[j].No < rows[j-1].No {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
}
