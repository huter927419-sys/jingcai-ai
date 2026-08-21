package titan007

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"jingcai-ai/internal/fetchhttp"
	"jingcai-ai/internal/market"
)

const (
	ua           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	bfBase       = "https://bf.titan007.com/football"
	euJS         = "https://1x2d.titan007.com/%d.js"
	euReferer    = "https://1x2.titan007.com/"
	changeAH     = "https://vip.titan007.com/changeDetail/handicap.aspx?id=%d&companyID=1&l=0"
	changeOU     = "https://vip.titan007.com/changeDetail/overunder.aspx?id=%d&companyID=1&l=0"
	asianSnap    = "https://vip.titan007.com/AsianOdds_n.aspx?id=%d"
	ouSnap       = "https://vip.titan007.com/OverDown_n.aspx?id=%d&l=0"
	titanReferer = "https://www.titan007.com/"
)

type Client struct {
	HTTP *http.Client
}

func New(proxy string) *Client {
	return &Client{HTTP: fetchhttp.Client(25*time.Second, proxy)}
}

type Match struct {
	ID     int64
	NumStr string
	League string
	Home   string
	Away   string
	Kick   string
}

type Odds struct {
	ID         int64
	Books      []market.EUBook
	Asian      *market.LineMove
	OU         *market.LineMove
	AsianBooks []market.LineMove
	OUBooks    []market.LineMove
}

func (c *Client) FetchJingzu(now time.Time) ([]Match, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var out []Match
	seen := map[int64]bool{}
	var lastErr error
	for i := -1; i <= 4; i++ {
		d := now.AddDate(0, 0, i)
		url := fmt.Sprintf("%s/Next_%s.htm", bfBase, d.Format("20060102"))
		html, err := c.get(url, titanReferer, true)
		if err != nil {
			lastErr = err
			continue
		}
		for _, m := range ParseSchedule(html) {
			if m.ID == 0 || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("titan: no jingzu rows")
	}
	return out, nil
}

func (c *Client) FetchOdds(id int64) (*Odds, error) {
	if id <= 0 {
		return nil, fmt.Errorf("titan: missing id")
	}
	js, err := c.get(fmt.Sprintf(euJS, id), euReferer, false)
	if err != nil {
		return nil, err
	}
	books := ParseEuropean(js)
	ahHTML, _ := c.get(fmt.Sprintf(changeAH, id), titanReferer, true)
	ouHTML, _ := c.get(fmt.Sprintf(changeOU, id), titanReferer, true)
	ahSnapHTML, _ := c.get(fmt.Sprintf(asianSnap, id), titanReferer, true)
	ouSnapHTML, _ := c.get(fmt.Sprintf(ouSnap, id), titanReferer, true)
	asian := ParseChangeDetail(ahHTML, "澳门")
	ou := ParseChangeDetail(ouHTML, "澳门")
	out := &Odds{
		ID:         id,
		Books:      books,
		Asian:      asian,
		OU:         ou,
		AsianBooks: MergeMoves(asian, ParseSnapshot(ahSnapHTML, "asian"), 6),
		OUBooks:    MergeMoves(ou, ParseSnapshot(ouSnapHTML, "ou"), 6),
	}
	if len(out.Books) == 0 && (out.Asian == nil || out.Asian.NodeCount == 0) && (out.OU == nil || out.OU.NodeCount == 0) {
		return nil, fmt.Errorf("titan: empty odds %d", id)
	}
	return out, nil
}

func FindMatch(rows []Match, numStr, home, away string) *Match {
	numStr, home, away = compact(numStr), compact(home), compact(away)
	for i := range rows {
		if rows[i].NumStr == numStr && numStr != "" {
			return &rows[i]
		}
	}
	for i := range rows {
		if teamClose(home, rows[i].Home) && teamClose(away, rows[i].Away) {
			return &rows[i]
		}
	}
	return nil
}

func Apply(q *market.Quote, o *Odds) {
	if q == nil || o == nil {
		return
	}
	q.TitanID = o.ID
	if len(o.Books) > 0 {
		q.Books = o.Books
	}
	if o.Asian != nil && o.Asian.NodeCount > 0 {
		q.AsianMove = o.Asian
	}
	if o.OU != nil && o.OU.NodeCount > 0 {
		q.OUMove = o.OU
	}
	if len(o.AsianBooks) > 0 {
		q.AsianBooks = o.AsianBooks
	}
	if len(o.OUBooks) > 0 {
		q.OUBooks = o.OUBooks
	}
	if b := bookByID(o.Books, 281); b != nil {
		if q.EU == nil && b.Current != nil && b.Current.H > 1 {
			cur := *b.Current
			cur.Company = "Bet365"
			q.EU = &cur
			q.Company = "Bet365"
		}
		if q.EU != nil && b.Opening != nil && b.Opening.H > 1 && q.EU.H0 <= 1 {
			q.EU.H0, q.EU.D0, q.EU.A0 = b.Opening.H, b.Opening.D, b.Opening.A
		}
	}
	q.FillImplied()
}

func bookByID(books []market.EUBook, id int) *market.EUBook {
	for i := range books {
		if books[i].CompanyID == id {
			return &books[i]
		}
	}
	return nil
}

func (c *Client) get(url, referer string, html bool) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if html {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	} else {
		req.Header.Set("Accept", "application/javascript,text/javascript,*/*")
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("titan HTTP %d", res.StatusCode)
	}
	if html {
		return decodeGBK(body), nil
	}
	return string(bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})), nil
}

func decodeGBK(b []byte) string {
	out, err := io.ReadAll(transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewDecoder()))
	if err != nil {
		return string(b)
	}
	return string(out)
}
