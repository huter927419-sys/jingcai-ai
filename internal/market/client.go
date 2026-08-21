package market

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"jingcai-ai/internal/fetchhttp"
)

const (
	listURL   = "https://trade.500.com/jczq/"
	ouzhiURL  = "https://odds.500.com/fenxi/ouzhi-%d.shtml"
	yazhiURL  = "https://odds.500.com/fenxi/yazhi-%d.shtml"
	daxiaoURL = "https://odds.500.com/fenxi/daxiao-%d.shtml"
	touzhuURL = "https://odds.500.com/fenxi/touzhu-%d.shtml"
	shujuURL  = "https://odds.500.com/fenxi/shuju-%d.shtml"
)

type Client struct {
	HTTP *http.Client
}

func New(proxy string) *Client {
	return &Client{HTTP: fetchhttp.Client(20*time.Second, proxy)}
}

func (c *Client) MapIDs() (map[int64]int64, error) {
	html, err := c.get(listURL, "https://trade.500.com/")
	if err != nil {
		return nil, err
	}
	m := ParseIDMap(html)
	if len(m) == 0 {
		return nil, fmt.Errorf("500.com: no fixture map")
	}
	return m, nil
}

func (c *Client) Fetch(matchID, fid int64) (*Quote, error) {
	if fid <= 0 {
		return nil, fmt.Errorf("500.com: missing fid")
	}
	q := &Quote{MatchID: matchID, Fid: fid, FetchedAt: time.Now(), Company: "Bet365"}
	if html, err := c.get(fmt.Sprintf(ouzhiURL, fid), listURL); err == nil {
		q.EU = ParseEU(html)
	}
	if html, err := c.get(fmt.Sprintf(yazhiURL, fid), listURL); err == nil {
		q.Asian = ParseAsian(html)
	}
	if html, err := c.get(fmt.Sprintf(daxiaoURL, fid), listURL); err == nil {
		q.OU = ParseOU(html)
	}
	if html, err := c.get(fmt.Sprintf(touzhuURL, fid), listURL); err == nil {
		q.Betfair = ParseBetfair(html)
	}
	if q.EU == nil && q.Asian == nil && q.OU == nil && q.Betfair == nil {
		return nil, fmt.Errorf("500.com: empty quote fid=%d", fid)
	}
	if q.EU != nil && q.EU.Company != "" {
		q.Company = q.EU.Company
	} else if q.Asian != nil && q.Asian.Company != "" {
		q.Company = q.Asian.Company
	} else if q.OU != nil && q.OU.Company != "" {
		q.Company = q.OU.Company
	}
	q.FillImplied()
	return q, nil
}

func (c *Client) FetchPreview(matchID, fid int64) (*Preview, error) {
	if fid <= 0 {
		return nil, fmt.Errorf("500.com: missing fid")
	}
	html, err := c.get(fmt.Sprintf(shujuURL, fid), listURL)
	if err != nil {
		return nil, err
	}
	p := ParsePreview(html)
	if p == nil {
		return nil, fmt.Errorf("500.com: empty preview fid=%d", fid)
	}
	p.MatchID = matchID
	p.Fid = fid
	return p, nil
}

func (c *Client) get(url, referer string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
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
		return "", fmt.Errorf("500.com HTTP %d", res.StatusCode)
	}
	return decodeGBK(body), nil
}

func decodeGBK(b []byte) string {
	out, err := io.ReadAll(transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewDecoder()))
	if err != nil {
		return string(b)
	}
	return string(out)
}
