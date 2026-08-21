package fetchhttp

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client returns an HTTP client. If proxy is set (http://host:port), only this
// client uses it — model API calls should keep using the default client.
func Client(timeout time.Duration, proxy string) *http.Client {
	c := &http.Client{Timeout: timeout}
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return c
	}
	u, err := url.Parse(proxy)
	if err != nil || u.Host == "" {
		return c
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyURL(u)
	c.Transport = tr
	return c
}
