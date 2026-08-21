package market

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Player struct {
	No   string `json:"no"`
	Name string `json:"name"`
	Pos  string `json:"pos"`
}

type Recent struct {
	Date   string  `json:"date"`
	League string  `json:"league"`
	Home   string  `json:"home"`
	Away   string  `json:"away"`
	Score  string  `json:"score"`
	Result string  `json:"result"`
	Rating float64 `json:"rating"`
}

type SidePreview struct {
	Name      string   `json:"name"`
	Formation string   `json:"formation"`
	Starters  []Player `json:"starters"`
	Bench     []Player `json:"bench"`
	Form      []Recent `json:"form"`
	AvgRating float64  `json:"avgRating"`
}

type Preview struct {
	MatchID int64       `json:"matchId"`
	Fid     int64       `json:"fid,omitempty"`
	Home    SidePreview `json:"home"`
	Away    SidePreview `json:"away"`
}

var (
	reTeamNameForm = regexp.MustCompile(`>([^<]{1,20})阵型:&nbsp;([^<]*)`)
	reXIRow        = regexp.MustCompile(`<tr><td class="td_one"><span class="td_sp3">([^<]*)</span>([^<]*)</td><td><span class="td_sp3">([^<]*)</span>([^<]*)</td></tr>`)
	reFormRow      = regexp.MustCompile(`(?s)<tr class="tr[12]"[^>]*>.*?</tr>`)
	reFormLeague   = regexp.MustCompile(`rel="nofollow"\s*>([^<]+)</a>`)
	reFormDate     = regexp.MustCompile(`<td>(\d{2}-\d{2}-\d{2})</td>`)
	reFormHome     = regexp.MustCompile(`class="dz-l[^"]*">(?:<span class="gray">\[[^\]]*\]</span>)?([^<]+)`)
	reFormAway     = regexp.MustCompile(`class="dz-r[^"]*">([^<]+)`)
	reFormEm       = regexp.MustCompile(`<em>(.*?)</em>`)
	reFormResult   = regexp.MustCompile(`>(胜|平|负)</span>`)
	reTags         = regexp.MustCompile(`<[^>]+>`)
)

func ParsePreview(html string) *Preview {
	if html == "" {
		return nil
	}
	p := &Preview{
		Home: parseSideBlock(html, `class="team_a"`, `class="team_b"`, `id="team_zhanji_1"`),
		Away: parseSideBlock(html, `class="team_b"`, `class="clearb"`, `id="team_zhanji_0"`),
	}
	if len(p.Home.Starters) == 0 && len(p.Away.Starters) == 0 && len(p.Home.Form) == 0 && len(p.Away.Form) == 0 {
		return nil
	}
	return p
}

func parseSideBlock(html, start, end, zhanji string) SidePreview {
	s := SidePreview{}
	xiHTML := html
	if i := strings.Index(html, `M_box starting`); i >= 0 {
		xiHTML = html[i:]
		if j := strings.Index(xiHTML, "心水推荐"); j > 0 {
			xiHTML = xiHTML[:j]
		}
	}
	if i := strings.Index(xiHTML, start); i >= 0 {
		chunk := xiHTML[i:]
		if j := strings.Index(chunk[len(start):], end); j >= 0 {
			chunk = chunk[:len(start)+j]
		}
		s = parseXI(chunk)
	}
	s.Form = parseForm(html, zhanji)
	s.AvgRating = avgRating(s.Form)
	if s.Formation == "" {
		s.Formation = inferFormation(s.Starters)
	}
	return s
}

func parseXI(html string) SidePreview {
	out := SidePreview{}
	if m := reTeamNameForm.FindStringSubmatch(html); m != nil {
		out.Name = strings.TrimSpace(m[1])
		out.Formation = strings.TrimSpace(htmlUnescape(m[2]))
	}
	cut := html
	if k := strings.Index(html, "伤病"); k >= 0 {
		cut = html[:k]
	}
	for _, m := range reXIRow.FindAllStringSubmatch(cut, -1) {
		if p, ok := parsePlayer(m[1], m[2]); ok {
			out.Starters = append(out.Starters, p)
		}
		if p, ok := parsePlayer(m[3], m[4]); ok {
			out.Bench = append(out.Bench, p)
		}
	}
	return out
}

func parsePlayer(no, raw string) (Player, bool) {
	no = strings.TrimSpace(no)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Player{}, false
	}
	p := Player{No: no, Name: raw}
	if i := strings.LastIndex(raw, "("); i > 0 && strings.HasSuffix(raw, ")") {
		p.Name = strings.TrimSpace(raw[:i])
		p.Pos = strings.TrimSuffix(raw[i+1:], ")")
	}
	return p, p.Name != ""
}

func parseForm(html, id string) []Recent {
	i := strings.Index(html, id)
	if i < 0 {
		return nil
	}
	chunk := html[i:]
	if j := strings.Index(chunk[len(id):], `id="team_zhanji`); j >= 0 {
		chunk = chunk[:len(id)+j]
	} else if j := strings.Index(chunk, "M_box starting"); j >= 0 {
		chunk = chunk[:j]
	}
	var out []Recent
	for _, row := range reFormRow.FindAllString(chunk, 12) {
		if strings.Contains(row, ">VS<") || strings.Contains(row, "<em>VS</em>") {
			continue
		}
		r := Recent{}
		if m := reFormLeague.FindStringSubmatch(row); m != nil {
			r.League = strings.TrimSpace(m[1])
		}
		if m := reFormDate.FindStringSubmatch(row); m != nil {
			r.Date = m[1]
		}
		if m := reFormHome.FindStringSubmatch(row); m != nil {
			r.Home = strings.TrimSpace(m[1])
		}
		if m := reFormAway.FindStringSubmatch(row); m != nil {
			r.Away = strings.TrimSpace(strings.Split(m[1], `<`)[0])
		}
		if m := reFormEm.FindStringSubmatch(row); m != nil {
			r.Score = compactScore(m[1])
		}
		if m := reFormResult.FindStringSubmatch(row); m != nil {
			r.Result = m[1]
		}
		if r.Score == "" || r.Result == "" {
			continue
		}
		r.Rating = matchRating(r.Result, r.Score)
		out = append(out, r)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func compactScore(s string) string {
	s = reTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "")
	return strings.TrimSpace(s)
}

func matchRating(result, score string) float64 {
	a, b := splitScore(score)
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	var v float64
	switch result {
	case "胜":
		v = 6.8 + math.Min(2.4, float64(diff)*0.6)
	case "平":
		v = 5.5
		if a > 0 {
			v = 5.8
		}
	default:
		v = 4.2 - math.Min(2.2, float64(diff)*0.5)
	}
	return math.Round(v*10) / 10
}

func splitScore(s string) (int, int) {
	s = strings.ReplaceAll(s, "：", ":")
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0
	}
	a, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	b, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return a, b
}

func inferFormation(starters []Player) string {
	var d, m, f int
	for _, p := range starters {
		switch p.Pos {
		case "后卫":
			d++
		case "中场":
			m++
		case "前锋":
			f++
		}
	}
	if d+m+f == 0 {
		return ""
	}
	return fmt.Sprintf("%d-%d-%d", d, m, f)
}

func avgRating(form []Recent) float64 {
	if len(form) == 0 {
		return 0
	}
	var s float64
	for _, r := range form {
		s += r.Rating
	}
	return math.Round(s/float64(len(form))*10) / 10
}

func htmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(s)
}
