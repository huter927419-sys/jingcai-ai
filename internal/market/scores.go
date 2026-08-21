package market

import (
	"fmt"
	"regexp"
	"strconv"
)

var reFT = regexp.MustCompile(`detail\.php\?fid=(\d+)[^>]*class="clt1"[^>]*>\s*(\d+)\s*</a>\s*<span>-</span>\s*<a[^>]*class="clt3"[^>]*>\s*(\d+)`)

func ParseFTScores(html string) map[int64][2]int {
	out := map[int64][2]int{}
	for _, m := range reFT.FindAllStringSubmatch(html, -1) {
		fid, _ := strconv.ParseInt(m[1], 10, 64)
		h, _ := strconv.Atoi(m[2])
		a, _ := strconv.Atoi(m[3])
		if fid > 0 {
			out[fid] = [2]int{h, a}
		}
	}
	return out
}

func (c *Client) FetchFTScores() (map[int64][2]int, error) {
	out := map[int64][2]int{}
	for _, u := range []string{"https://live.500.com/wanchang.php", "https://live.500.com/2h1.php"} {
		html, err := c.get(u, "https://live.500.com/")
		if err != nil {
			continue
		}
		for fid, sc := range ParseFTScores(html) {
			out[fid] = sc
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("500.com: no finished scores")
	}
	return out, nil
}
