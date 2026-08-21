package market

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reOuzhiSame = regexp.MustCompile(`ouzhi_same\.php\?cid=(\d+)&win=([0-9.]+)&draw=([0-9.]+)&lost=([0-9.]+)`)
	reOuzhiNums = regexp.MustCompile(`>([0-9]+\.[0-9]+)\s*<`)
	reAsianRow  = regexp.MustCompile(`>([0-9.]+)[↑↓]?<[\s\S]{0,500}?ref="(-?[0-9.]+)"[^>]*>([^<]+)[\s\S]{0,400}?>([0-9.]+)[↑↓]?<`)
	reOURow     = regexp.MustCompile(`>([0-9.]+)[↑↓]?<[\s\S]{0,400}?ref="(-?[0-9.]+)"[^>]*>([0-9.]+)[\s\S]{0,400}?>([0-9.]+)[↑↓]?<`)
	reLis       = regexp.MustCompile(`<li>([0-9,]+)</li>`)
	reThinTalk  = regexp.MustCompile(`规模偏小|成交清淡|样本偏小|交易规模较小`)
)

// 3=Bet365, 5=澳门, 293=威廉. 主用 365，缺盘再退。
var preferCID = []string{"3", "5", "293"}

func ParseIDMap(html string) map[int64]int64 {
	out := map[int64]int64{}
	reID := regexp.MustCompile(`data-id="(\d+)"`)
	reFid := regexp.MustCompile(`data-fixtureid="(\d+)"`)
	for _, part := range strings.Split(html, "<tr") {
		chunk := part
		if len(chunk) > 900 {
			chunk = chunk[:900]
		}
		idm := reID.FindStringSubmatch(chunk)
		fidm := reFid.FindStringSubmatch(chunk)
		if idm == nil || fidm == nil {
			continue
		}
		id, _ := strconv.ParseInt(idm[1], 10, 64)
		fid, _ := strconv.ParseInt(fidm[1], 10, 64)
		if id > 0 && fid > 0 {
			out[id] = fid
		}
	}
	return out
}

func ParseEU(html string) *Trio {
	if cid, row := rowByPreferCID(html); row != "" {
		if t := trioFromChunk(row, cid); t != nil {
			return t
		}
	}
	byCID := map[string]Trio{}
	for _, m := range reOuzhiSame.FindAllStringSubmatch(html, -1) {
		h, _ := strconv.ParseFloat(m[2], 64)
		d, _ := strconv.ParseFloat(m[3], 64)
		a, _ := strconv.ParseFloat(m[4], 64)
		if h > 1 && d > 1 && a > 1 {
			byCID[m[1]] = Trio{H: h, D: d, A: a, Company: companyName(m[1])}
		}
	}
	if t := pickTrio(byCID); t != nil {
		return t
	}
	return parseEUFromRows(html)
}

func parseEUFromRows(html string) *Trio {
	byCID := map[string]Trio{}
	for _, cid := range preferCID {
		_, row := rowByCID(html, cid)
		if row == "" {
			continue
		}
		if t := trioFromChunk(row, cid); t != nil {
			byCID[cid] = *t
		}
	}
	return pickTrio(byCID)
}

func trioFromChunk(row, cid string) *Trio {
	chunk := row
	if tabs := oddsTables(row, 1); len(tabs) > 0 {
		chunk = tabs[0]
	}
	nums := reOuzhiNums.FindAllStringSubmatch(chunk, 6)
	if len(nums) < 3 {
		return nil
	}
	h, _ := strconv.ParseFloat(nums[0][1], 64)
	d, _ := strconv.ParseFloat(nums[1][1], 64)
	a, _ := strconv.ParseFloat(nums[2][1], 64)
	if h <= 1 || d <= 1 || a <= 1 {
		return nil
	}
	t := &Trio{H: h, D: d, A: a, Company: companyName(cid)}
	if len(nums) >= 6 {
		h0, _ := strconv.ParseFloat(nums[3][1], 64)
		d0, _ := strconv.ParseFloat(nums[4][1], 64)
		a0, _ := strconv.ParseFloat(nums[5][1], 64)
		if h0 > 1 && d0 > 1 && a0 > 1 {
			t.H0, t.D0, t.A0 = h0, d0, a0
		}
	}
	return t
}

func ParseAsian(html string) *Handicap {
	cid, row := rowByPreferCID(html)
	if row == "" {
		return nil
	}
	tabs := oddsTables(row, 2)
	cur := handicapFromChunk(row, cid)
	if len(tabs) > 0 {
		if t := handicapFromChunk(tabs[0], cid); t != nil {
			cur = t
		}
	}
	if cur == nil {
		return nil
	}
	if len(tabs) > 1 {
		if open := handicapFromChunk(tabs[1], cid); open != nil {
			cur.Home0, cur.Away0 = open.Home, open.Away
			cur.Line0, cur.LineNum0 = open.Line, open.LineNum
		}
	}
	return cur
}

func ParseOU(html string) *OU {
	cid, row := rowByPreferCID(html)
	if row == "" {
		return nil
	}
	tabs := oddsTables(row, 2)
	cur := ouFromChunk(row, cid)
	if len(tabs) > 0 {
		if t := ouFromChunk(tabs[0], cid); t != nil {
			cur = t
		}
	}
	if cur == nil {
		return nil
	}
	if len(tabs) > 1 {
		if open := ouFromChunk(tabs[1], cid); open != nil {
			cur.Over0, cur.Under0, cur.Line0 = open.Over, open.Under, open.Line
		}
	}
	return cur
}

func handicapFromChunk(row, cid string) *Handicap {
	m := reAsianRow.FindStringSubmatch(row)
	if m == nil {
		return nil
	}
	home, _ := strconv.ParseFloat(m[1], 64)
	away, _ := strconv.ParseFloat(m[4], 64)
	lineNum, _ := strconv.ParseFloat(m[2], 64)
	line := strings.TrimSpace(m[3])
	line = strings.TrimSpace(strings.NewReplacer("升", "", "降", "", " ", "").Replace(line))
	if home <= 0 || away <= 0 {
		return nil
	}
	return &Handicap{Company: companyName(cid), Line: line, LineNum: lineNum, Home: home, Away: away}
}

func ouFromChunk(row, cid string) *OU {
	m := reOURow.FindStringSubmatch(row)
	if m == nil {
		m = reAsianRow.FindStringSubmatch(row)
	}
	if m == nil {
		return nil
	}
	over, _ := strconv.ParseFloat(m[1], 64)
	under, _ := strconv.ParseFloat(m[4], 64)
	line, _ := strconv.ParseFloat(strings.TrimSpace(m[3]), 64)
	if line == 0 {
		if ref, err := strconv.ParseFloat(m[2], 64); err == nil {
			line = absFloat(ref)
		}
	}
	if over <= 0 || under <= 0 || line <= 0 {
		return nil
	}
	return &OU{Company: companyName(cid), Line: line, Over: over, Under: under}
}

func ParseBetfair(html string) *Betfair {
	idx := strings.Index(html, "总交易")
	chunk := html
	if idx >= 0 {
		end := idx + 400
		if end > len(html) {
			end = len(html)
		}
		chunk = html[idx:end]
	}
	lis := reLis.FindAllStringSubmatch(chunk, 3)
	if len(lis) < 3 {
		return nil
	}
	bf := &Betfair{
		HomeVol: parseComma(lis[0][1]),
		DrawVol: parseComma(lis[1][1]),
		AwayVol: parseComma(lis[2][1]),
	}
	if reThinTalk.MatchString(html) {
		bf.Thin = true
		bf.Note = "样本偏小"
	}
	return bf
}

func rowByPreferCID(html string) (string, string) {
	for _, cid := range preferCID {
		if _, row := rowByCID(html, cid); row != "" {
			return cid, row
		}
	}
	return "", ""
}

func rowByCID(html, cid string) (string, string) {
	re := regexp.MustCompile(`<tr[^>]*id="` + cid + `"[^>]*>`)
	loc := re.FindStringIndex(html)
	if loc == nil {
		return "", ""
	}
	end := loc[0] + 5000
	if end > len(html) {
		end = len(html)
	}
	return cid, html[loc[0]:end]
}

func oddsTables(row string, n int) []string {
	var out []string
	rest := row
	for len(out) < n {
		i := strings.Index(rest, `class="pl_table_data"`)
		if i < 0 {
			break
		}
		rest = rest[i:]
		j := strings.Index(rest, `</table>`)
		if j < 0 {
			break
		}
		out = append(out, rest[:j])
		rest = rest[j+8:]
	}
	return out
}

func pickTrio(byCID map[string]Trio) *Trio {
	for _, cid := range preferCID {
		if t, ok := byCID[cid]; ok {
			out := t
			return &out
		}
	}
	return nil
}

func companyName(cid string) string {
	switch cid {
	case "3":
		return "Bet365"
	case "5":
		return "澳门"
	case "293":
		return "威廉"
	case "2":
		return "立博"
	default:
		return "机构"
	}
}

func parseComma(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
