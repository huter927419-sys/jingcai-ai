package titan007

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"jingcai-ai/internal/market"
)

var (
	euWhitelist = []int{177, 115, 82, 281, 104, 386, 80, 545}
	euNames     = map[int]string{
		177: "平博",
		115: "威廉希尔",
		82:  "立博",
		281: "Bet365",
		104: "Interwetten",
		386: "Unibet",
		80:  "澳门",
		545: "Crown",
	}
	snapNames = map[int]string{
		1:  "澳门",
		3:  "Crown",
		8:  "36",
		12: "易胜博",
		14: "伟德",
		17: "明升",
		24: "12BET",
		31: "利记",
		35: "盈禾",
		42: "18",
		47: "平博",
		48: "香港马会",
	}
	reGame       = regexp.MustCompile(`(?is)var\s+game\s*=\s*Array\s*\((.*?)\)\s*;`)
	reDetail     = regexp.MustCompile(`(?is)(?:var\s+)?gameDetail\s*=\s*Array\s*\((.*?)\)\s*;`)
	reQuoted     = regexp.MustCompile(`"((?:\\.|[^"\\])*)"`)
	reJinzu      = regexp.MustCompile(`jinZuSchedule\s*=\s*["']([^"']+)["']`)
	reRow        = regexp.MustCompile(`(?is)<tr\b([^>]*)>(.*?)</tr>`)
	reSID        = regexp.MustCompile(`(?i)sid=['"](\d+)['"]`)
	reTD         = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)
	reChangeRow  = regexp.MustCompile(`(?is)<tr[^>]*align=['"]?center['"]?[^>]*>(.*?)</tr>`)
	reChangeCell = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td>`)
	reTDFull     = regexp.MustCompile(`(?is)<td\b([^>]*)>(.*?)</td>`)
	reAttr       = regexp.MustCompile(`([\w:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	rePanN       = regexp.MustCompile(`^盘\d+$`)
	reCompanyID  = regexp.MustCompile(`companyid=['"]?(\d+)`)
	reTags       = regexp.MustCompile(`(?is)<[^>]+>`)
)

func ParseEuropean(js string) []market.EUBook {
	body := ""
	if m := reGame.FindStringSubmatch(js); len(m) == 2 {
		body = m[1]
	}
	type raw struct {
		id, rec int
		name    string
		open    market.Trio
		cur     market.Trio
	}
	var rows []raw
	recToID := map[string]int{}
	for _, q := range quoted(body) {
		f := strings.Split(q, "|")
		if len(f) < 17 {
			continue
		}
		id := atoi(f[0])
		if !whitelisted(id) {
			continue
		}
		rec := compact(f[1])
		name := euNames[id]
		if name == "" {
			name = compact(index(f, 21))
		}
		r := raw{
			id:   id,
			name: name,
			open: market.Trio{Company: name, H: atof(f[3]), D: atof(f[4]), A: atof(f[5])},
			cur:  market.Trio{Company: name, H: atof(f[10]), D: atof(f[11]), A: atof(f[12])},
		}
		rows = append(rows, r)
		if rec != "" {
			recToID[rec] = id
		}
	}
	hist := map[int][]market.EUNode{}
	detail := ""
	if m := reDetail.FindStringSubmatch(js); len(m) == 2 {
		detail = m[1]
	}
	for _, q := range quoted(detail) {
		rec, rest, ok := strings.Cut(q, "^")
		if !ok {
			continue
		}
		id := recToID[compact(rec)]
		if !whitelisted(id) {
			continue
		}
		var nodes []market.EUNode
		for _, entry := range strings.Split(rest, ";") {
			if compact(entry) == "" {
				continue
			}
			p := strings.Split(entry, "|")
			nodes = append(nodes, market.EUNode{
				Time: histTime(index(p, 3), index(p, 7)),
				H:    atof(index(p, 0)),
				D:    atof(index(p, 1)),
				A:    atof(index(p, 2)),
			})
		}
		if len(nodes) > 0 {
			hist[id] = capEUNodes(nodes)
		}
	}
	out := make([]market.EUBook, 0, len(rows))
	for _, r := range rows {
		open, cur := r.open, r.cur
		out = append(out, market.EUBook{CompanyID: r.id, Company: r.name, Opening: &open, Current: &cur, Nodes: hist[r.id]})
	}
	return out
}

func ParseSchedule(html string) []Match {
	numByID := map[int64]string{}
	if m := reJinzu.FindStringSubmatch(html); len(m) == 2 {
		idsPart, numsPart, _ := strings.Cut(m[1], "|")
		ids := strings.Split(idsPart, ",")
		nums := strings.Split(numsPart, ",")
		for i, id := range ids {
			n, _ := strconv.ParseInt(compact(id), 10, 64)
			if n <= 0 {
				continue
			}
			num := ""
			if i < len(nums) {
				num = compact(nums[i])
			}
			numByID[n] = num
		}
	}
	var out []Match
	for _, row := range reRow.FindAllStringSubmatch(html, -1) {
		attrs, body := row[1], row[2]
		sidm := reSID.FindStringSubmatch(attrs)
		if sidm == nil {
			continue
		}
		id, _ := strconv.ParseInt(sidm[1], 10, 64)
		num := numByID[id]
		if num == "" {
			continue
		}
		cells := tds(body)
		if len(cells) < 6 {
			continue
		}
		home := stripRank(cells[3])
		away := stripRank(cells[5])
		out = append(out, Match{
			ID:     id,
			NumStr: num,
			League: compact(cells[0]),
			Home:   home,
			Away:   away,
			Kick:   compact(cells[1]),
		})
	}
	return out
}

func ParseChangeDetail(html, company string) *market.LineMove {
	span := html
	if i := strings.Index(strings.ToLower(html), `id="odds2"`); i >= 0 {
		span = html[i:]
	} else if i := strings.Index(strings.ToLower(html), `id='odds2'`); i >= 0 {
		span = html[i:]
	}
	var nodes []market.LineNode
	for _, row := range reChangeRow.FindAllStringSubmatch(span, -1) {
		body := row[1]
		if strings.Contains(body, "变化时间") && strings.Contains(body, "状态") {
			continue
		}
		if strings.Contains(body, "colspan") || strings.Contains(body, ">封<") {
			continue
		}
		cells := changeCells(body)
		if len(cells) < 5 {
			continue
		}
		var left, line, right, tm, status string
		if len(cells) >= 7 {
			left, line, right, tm, status = cells[len(cells)-5], cells[len(cells)-4], cells[len(cells)-3], cells[len(cells)-2], cells[len(cells)-1]
		} else {
			left, line, right, tm, status = cells[0], cells[1], cells[2], cells[3], cells[4]
		}
		st := compact(status)
		if st == "滚" {
			continue
		}
		nodes = append(nodes, market.LineNode{
			Time:   compact(tm),
			Line:   compact(line),
			Left:   atof(left),
			Right:  atof(right),
			Status: st,
		})
	}
	if len(nodes) == 0 {
		return nil
	}
	open, cur := pickOpenCurrent(nodes)
	return &market.LineMove{
		CompanyID:    1,
		Company:      company,
		OpeningLine:  open.Line,
		CurrentLine:  cur.Line,
		OpeningLeft:  open.Left,
		OpeningRight: open.Right,
		CurrentLeft:  cur.Left,
		CurrentRight: cur.Right,
		NodeCount:    len(nodes),
		Nodes:        capLineNodes(nodes),
	}
}

func ParseSnapshot(html, kind string) []market.LineMove {
	var out []market.LineMove
	seen := map[int]bool{}
	for _, row := range reRow.FindAllStringSubmatch(html, -1) {
		attrs, body := row[1], row[2]
		low := strings.ToLower(attrs)
		if strings.Contains(low, "display:none") || strings.Contains(low, "display: none") {
			continue
		}
		mv := parseSnapshotRow(attrs, body)
		if mv == nil || mv.CurrentLine == "" {
			continue
		}
		if mv.CompanyID != 0 && seen[mv.CompanyID] {
			continue
		}
		if mv.CompanyID != 0 {
			seen[mv.CompanyID] = true
		}
		out = append(out, *mv)
	}
	return preferSnapshot(out)
}

func parseSnapshotRow(rowAttrs, body string) *market.LineMove {
	cells := snapshotCells(body)
	if len(cells) < 5 {
		return nil
	}
	label := ""
	for i := 0; i < len(cells) && i < 4; i++ {
		if cells[i].text != "" {
			label = cells[i].text
			break
		}
	}
	if rePanN.MatchString(label) {
		return nil
	}
	cid := snapshotCompanyID(rowAttrs + " " + body)
	name := snapNames[cid]
	if name == "" {
		name = strings.ReplaceAll(label, "*", "")
	}
	if name == "" && cid == 0 {
		return nil
	}
	openIdx, curIdx := -1, -1
	fallback := -1
	for i, c := range cells {
		if c.attrs["goals"] == "" {
			continue
		}
		hidden := strings.Contains(strings.ToLower(c.attrs["style"]), "display:none") || strings.Contains(strings.ToLower(c.attrs["style"]), "display: none")
		ot := c.attrs["oddstype"]
		if ot == "" && c.attrs["title"] != "" && openIdx < 0 {
			openIdx = i
		}
		if ot == "wholeOdds" && !hidden && curIdx < 0 {
			curIdx = i
		}
		if ot == "wholeLastOdds" && !hidden && fallback < 0 {
			fallback = i
		}
	}
	if curIdx < 0 {
		curIdx = fallback
	}
	open := snapshotTriplet(cells, openIdx)
	cur := snapshotTriplet(cells, curIdx)
	if open == nil || cur == nil {
		return nil
	}
	return &market.LineMove{
		CompanyID:    cid,
		Company:      name,
		OpeningLine:  open.line,
		CurrentLine:  cur.line,
		OpeningLeft:  open.left,
		OpeningRight: open.right,
		CurrentLeft:  cur.left,
		CurrentRight: cur.right,
	}
}

type snapTrip struct {
	line        string
	left, right float64
}

func snapshotTriplet(cells []snapCell, i int) *snapTrip {
	if i <= 0 || i >= len(cells)-1 {
		return nil
	}
	line := cells[i].attrs["goals"]
	if line == "" {
		line = cells[i].text
	}
	return &snapTrip{line: line, left: atof(cells[i-1].text), right: atof(cells[i+1].text)}
}

type snapCell struct {
	attrs map[string]string
	text  string
}

func snapshotCells(body string) []snapCell {
	var out []snapCell
	for _, m := range reTDFull.FindAllStringSubmatch(body, -1) {
		out = append(out, snapCell{attrs: parseAttrs(m[1]), text: compact(stripTags(m[2]))})
	}
	return out
}

func parseAttrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range reAttr.FindAllStringSubmatch(tag, -1) {
		out[strings.ToLower(m[1])] = m[2] + m[3] + m[4]
	}
	return out
}

func snapshotCompanyID(s string) int {
	low := strings.ToLower(s)
	for _, m := range reCompanyID.FindAllStringSubmatch(low, -1) {
		id := atoi(m[1])
		if id > 0 {
			return id
		}
	}
	return 0
}

func preferSnapshot(in []market.LineMove) []market.LineMove {
	order := []int{1, 3, 8, 12, 14, 47, 31}
	rank := map[int]int{}
	for i, id := range order {
		rank[id] = i
	}
	out := append([]market.LineMove{}, in...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			ri, rok := rank[out[i].CompanyID]
			rj, jok := rank[out[j].CompanyID]
			if !rok {
				ri = 50
			}
			if !jok {
				rj = 50
			}
			if rj < ri {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func MergeMoves(macau *market.LineMove, snaps []market.LineMove, max int) []market.LineMove {
	var out []market.LineMove
	seen := map[string]bool{}
	add := func(m market.LineMove) {
		if max > 0 && len(out) >= max {
			return
		}
		key := fmt.Sprintf("%d:%s", m.CompanyID, m.Company)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, m)
	}
	if macau != nil && (macau.CurrentLine != "" || macau.NodeCount > 0) {
		if macau.CompanyID == 0 {
			macau.CompanyID = 1
		}
		if macau.Company == "" {
			macau.Company = "澳门"
		}
		add(*macau)
	}
	for _, s := range snaps {
		add(s)
	}
	return out
}

func pickOpenCurrent(nodes []market.LineNode) (market.LineNode, market.LineNode) {
	open, cur := nodes[0], nodes[len(nodes)-1]
	for _, n := range nodes {
		if n.Status == "初" {
			open = n
			break
		}
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Status == "即" || nodes[i].Status == "初" {
			cur = nodes[i]
			break
		}
	}
	return open, cur
}

func capEUNodes(nodes []market.EUNode) []market.EUNode {
	if len(nodes) <= 8 {
		return nodes
	}
	return append(append([]market.EUNode{}, nodes[:2]...), nodes[len(nodes)-6:]...)
}

func capLineNodes(nodes []market.LineNode) []market.LineNode {
	var keep []market.LineNode
	for _, n := range nodes {
		if n.Status == "初" || n.Status == "即" {
			keep = append(keep, n)
		}
	}
	if len(keep) == 0 {
		keep = nodes
	}
	if len(keep) <= 8 {
		return keep
	}
	return append(append([]market.LineNode{}, keep[:2]...), keep[len(keep)-6:]...)
}

func quoted(body string) []string {
	var out []string
	for _, m := range reQuoted.FindAllStringSubmatch(body, -1) {
		out = append(out, strings.ReplaceAll(m[1], `\"`, `"`))
	}
	return out
}

func tds(body string) []string {
	var out []string
	for _, m := range reTD.FindAllStringSubmatch(body, -1) {
		out = append(out, compact(stripTags(m[1])))
	}
	return out
}

func changeCells(body string) []string {
	var out []string
	for _, m := range reChangeCell.FindAllStringSubmatch(body, -1) {
		out = append(out, compact(stripTags(m[1])))
	}
	return out
}

func stripTags(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = reTags.ReplaceAllString(s, " ")
	return s
}

func stripRank(s string) string {
	out := s
	for {
		i := strings.Index(out, "[")
		j := strings.Index(out, "]")
		if i < 0 || j < i {
			break
		}
		out = out[:i] + out[j+1:]
	}
	return compact(out)
}

func teamClose(a, b string) bool {
	a, b = compact(a), compact(b)
	if a == "" || b == "" {
		return false
	}
	if a == b || strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	ra, rb := []rune(a), []rune(b)
	n := 0
	for i := 0; i < len(ra) && i < len(rb) && ra[i] == rb[i]; i++ {
		if ra[i] > 127 {
			n++
		}
	}
	return n >= 2
}

func histTime(md, year string) string {
	md, year = compact(md), compact(year)
	if year != "" && md != "" && !strings.HasPrefix(md, "20") {
		return year + "-" + md
	}
	return md
}

func compact(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(s), " ")
}

func whitelisted(id int) bool {
	for _, w := range euWhitelist {
		if w == id {
			return true
		}
	}
	return false
}

func atoi(s string) int {
	n, _ := strconv.Atoi(compact(s))
	return n
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(compact(s), 64)
	return v
}

func index(f []string, i int) string {
	if i < 0 || i >= len(f) {
		return ""
	}
	return f[i]
}
