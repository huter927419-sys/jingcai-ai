package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Name    string
	Key     string
	BaseURL string
	Model   string
	HTTP    *http.Client
}

type Output struct {
	LambdaHome float64 `json:"lambda_home"`
	LambdaAway float64 `json:"lambda_away"`
	Headline   string  `json:"headline"`
	PlainTalk  string  `json:"plain_talk"`
	Pick1X2    string  `json:"pick_1x2"`
	PickOU     string  `json:"pick_ou"`
	Verdict    string  `json:"verdict"`
	BuyTalk    string  `json:"buy_talk"`
}

func New(key, base, model string) *Client {
	return NewNamed("", key, base, model)
}

func NewNamed(name, key, base, model string) *Client {
	model = strings.TrimSpace(model)
	name = strings.TrimSpace(name)
	if name == "" {
		name = model
	}
	return &Client{
		Name:    name,
		Key:     strings.TrimSpace(key),
		BaseURL: strings.TrimRight(strings.TrimSpace(base), "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.Key != ""
}

func (c *Client) Analyze(prompt string) (Output, error) {
	if !c.Enabled() {
		return Output{}, fmt.Errorf("no api key")
	}
	body := map[string]any{
		"model":      c.Model,
		"max_tokens": 800,
		"messages": []map[string]string{
			{"role": "system", "content": c.systemText()},
			{"role": "user", "content": prompt},
		},
	}
	if strings.Contains(strings.ToLower(c.Model), "grok") {
		body["reasoning_effort"] = "low"
	}
	if !strings.Contains(strings.ToLower(c.Model), "claude") {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Output{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Output{}, err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return Output{}, fmt.Errorf("%s HTTP %d: %s", c.tag(), res.StatusCode, truncate(string(respBody), 400))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Output{}, err
	}
	if len(parsed.Choices) == 0 {
		return Output{}, fmt.Errorf("%s empty choices", c.tag())
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = stripThink(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	if i := strings.Index(content, "{"); i >= 0 {
		if j := strings.LastIndex(content, "}"); j > i {
			content = content[i : j+1]
		}
	}
	var out Output
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return Output{}, fmt.Errorf("%s json: %w (%s)", c.tag(), err, truncate(content, 240))
	}
	if out.LambdaHome < 0.15 || out.LambdaAway < 0.15 {
		return Output{}, fmt.Errorf("%s lambda invalid", c.tag())
	}
	return out, nil
}

func (c *Client) systemText() string {
	if c != nil && strings.Contains(strings.ToLower(c.Model), "claude") {
		return claudePrompt
	}
	return systemPrompt
}

const claudePrompt = `你是足球赛前专家。只输出 JSON：
{"lambda_home":数字,"lambda_away":数字,"headline":"不超过22字","plain_talk":"120-180字完整解读，覆盖走势、阵容、值不值","pick_1x2":"胜或平或负","pick_ou":"大或小","verdict":"主推或可看或放弃","buy_talk":"40-80字，说明竞彩买哪一玩法哪一侧，或明确放弃。不要写金额"}
lambda 是90分钟期望进球，范围0.3-4.2，贴近初值，微调不超过0.35。
不要写术语，不要把某一个比分说成一定会出。`

const systemPrompt = `你是竞彩足球专家，按指定角色做完整解读。内部用价值差、松紧、过热思考，对外只说人话。
只输出 JSON：
{"lambda_home":数字,"lambda_away":数字,"headline":"不超过22字","plain_talk":"160-240字完整解读：这场怎么打、盘口热不热、值不值","pick_1x2":"胜|平|负","pick_ou":"大|小","verdict":"主推|可看|放弃","buy_talk":"40-90字，明确竞彩怎么买：胜平负/让球/大小2.5买哪一侧，或放弃。不要写金额"}
硬性规定：
- lambda 是 90 分钟期望进球，范围 0.3-4.2，必须贴近给定初值，微调不超过 0.35。
- headline、plain_talk、buy_talk 禁止出现：凯利、价值差、热门度、泊松、xG、λ、模型、庄家、去水。
- 价值看 Bet365，不要用竞彩 SP 判断值不值。票面只告诉用户买哪个玩法。
- pick_1x2 只选一侧。主推必须更值，拿不准写可看或放弃。
- 不要把某一个比分说成一定会出。不要写投注金额。`

func (c *Client) tag() string {
	if c != nil && strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	return "llm"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func stripThink(s string) string {
	for {
		start := strings.Index(s, "<think>")
		end := strings.Index(s, "</think>")
		if start < 0 || end < 0 || end < start {
			return strings.TrimSpace(s)
		}
		s = strings.TrimSpace(s[:start] + s[end+len("</think>"):])
	}
}
