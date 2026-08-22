package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"jingcai-ai/internal/ailog"
)

var requestSequence atomic.Uint64

type Client struct {
	Name    string
	Key     string
	BaseURL string
	Model   string
	HTTP    *http.Client
	Audit   *ailog.Logger
}

type Output struct {
	LambdaHome   float64  `json:"lambda_home"`
	LambdaAway   float64  `json:"lambda_away"`
	Headline     string   `json:"headline"`
	PlainTalk    string   `json:"plain_talk"`
	Pick1X2      string   `json:"pick_1x2"`
	PickOU       string   `json:"pick_ou"`
	Verdict      string   `json:"verdict"`
	BuyTalk      string   `json:"buy_talk"`
	Pattern      string   `json:"pattern"`
	Scores       []string `json:"scores"`
	PickHandicap string   `json:"pick_handicap"`
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
	maxTokens := 800
	if strings.Contains(strings.ToLower(c.Model), "claude") {
		maxTokens = 1600
	}
	var out Output
	if err := c.chatJSON(c.systemText(), prompt, maxTokens, &out); err != nil {
		return Output{}, err
	}
	if out.LambdaHome < 0.15 || out.LambdaAway < 0.15 {
		return Output{}, fmt.Errorf("%s lambda invalid", c.tag())
	}
	return out, nil
}

type ReviewOut struct {
	Headline      string   `json:"headline"`
	PlainTalk     string   `json:"plain_talk"`
	MissType      string   `json:"miss_type"`
	VisibleBefore []string `json:"visible_before"`
	Overread      []string `json:"overread"`
	Lesson        string   `json:"lesson"`
}

func (c *Client) Review(prompt string) (ReviewOut, error) {
	var out ReviewOut
	if err := c.chatJSON(reviewSystem, prompt, 1400, &out); err != nil {
		return ReviewOut{}, err
	}
	if strings.TrimSpace(out.PlainTalk) == "" {
		return ReviewOut{}, fmt.Errorf("%s empty review", c.tag())
	}
	return out, nil
}

func (c *Client) chatJSON(system, prompt string, maxTokens int, dest any) (err error) {
	if !c.Enabled() {
		return fmt.Errorf("no api key")
	}
	started := time.Now()
	requestID := fmt.Sprintf("%d-%06d", started.UnixMilli(), requestSequence.Add(1))
	endpoint := c.BaseURL + "/chat/completions"
	stage := "build_request"
	status := 0
	requestBody := ""
	responseBody := ""
	defer func() {
		if c.Audit == nil {
			return
		}
		event := ailog.Event{
			Timestamp:  started,
			RequestID:  requestID,
			Provider:   c.Name,
			Model:      c.Model,
			Endpoint:   endpoint,
			DurationMS: time.Since(started).Milliseconds(),
			HTTPStatus: status,
			Success:    err == nil,
			Stage:      stage,
			Request:    requestBody,
			Response:   responseBody,
		}
		if err != nil {
			event.Error = err.Error()
		}
		c.Audit.Log(event)
	}()
	if maxTokens < 200 {
		maxTokens = 800
	}
	body := map[string]any{
		"model":      c.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": system},
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
	requestBody = string(raw)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	stage = "http_request"
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	status = res.StatusCode
	stage = "response_read"
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	responseBody = string(respBody)
	if res.StatusCode >= 300 {
		stage = "http_status"
		return fmt.Errorf("%s HTTP %d: %s", c.tag(), res.StatusCode, truncate(string(respBody), 400))
	}
	stage = "response_envelope"
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return err
	}
	if len(parsed.Choices) == 0 {
		return fmt.Errorf("%s empty choices", c.tag())
	}
	stage = "response_content"
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
	stage = "response_json"
	if err := json.Unmarshal([]byte(content), dest); err != nil {
		return fmt.Errorf("%s json: %w (%s)", c.tag(), err, truncate(content, 240))
	}
	stage = "complete"
	return nil
}

const reviewSystem = `你是复盘分析师，只做赛后归因，不改写赛前结论，不下新的买入指令。
只输出 JSON：
{"headline":"不超过22字","miss_type":"全错或胜平负看错","plain_talk":"220-340字","visible_before":["事前能看到的信号"],"overread":["赛前被读过头的信号"],"lesson":"不超过40字的一条教训"}
硬性规定：
- 用赛前已经存在的票面、近况、专家选项和风险提示解释为什么看错，禁止用赛果倒推“早该看出来”。
- 没有的资料写未确认，严禁编造伤停、换人和内部消息。
- headline、plain_talk、lesson 禁止出现：AI、模型名、凯利、价值差、泊松、xG、λ、去水、立即抓取。
- 不要推荐下一场怎么买。不要把冷门说成可以预判。
- visible_before 和 overread 各最多 4 条，没有就给空数组。`

func (c *Client) systemText() string {
	if c != nil && strings.Contains(strings.ToLower(c.Model), "claude") {
		return claudePrompt
	}
	return systemPrompt
}

const claudePrompt = `你是专业足球盘口分析师和赛事解说员。只输出 JSON：
{"lambda_home":数字,"lambda_away":数字,"headline":"不超过22字","pattern":"20-40字比赛格局","plain_talk":"220-320字专业解盘","pick_1x2":"胜或平或负","pick_ou":"大或小","pick_handicap":"让胜或让平或让负或放弃","scores":["情景比分1","情景比分2"],"verdict":"主推或关注或放弃","buy_talk":"70-120字参考买入，明确主方向、次方向、触发条件和风险边界"}
lambda 是90分钟期望进球，范围0.3-4.2，贴近初值，微调不超过0.35。
plain_talk 依次说明：盘口与水位表达的态度、阵型和首发对攻防的影响、伤停是否有可靠数据、价值方向如何变化。允许使用升盘、降盘、退盘、阻上、低水、高水、赔付压力、攻守转换、肋部、边路纵深等专业术语，但必须解释清楚。没有伤停资料就明确写未确认，严禁虚构。不要把比分说成一定会出。`

const systemPrompt = `你是专业足球盘口分析师和赛事解说员，按指定研判视角给出可执行结论。解读要像专业赛前节目：先定比赛格局，再解释盘口、人员和价值。
只输出 JSON：
{"lambda_home":数字,"lambda_away":数字,"headline":"不超过22字","pattern":"20-40字比赛格局","plain_talk":"220-340字专业解盘","pick_1x2":"胜|平|负","pick_ou":"大|小","pick_handicap":"让胜|让平|让负|放弃","scores":["情景比分1","情景比分2"],"verdict":"主推|关注|放弃","buy_talk":"70-120字参考买入，明确主方向、次方向、触发条件和风险边界"}
硬性规定：
- lambda 是 90 分钟期望进球，范围 0.3-4.2，必须贴近给定初值，微调不超过 0.35。
- plain_talk 必须按自然段语义覆盖：①初盘到当前盘的升降、欧亚赔和大小球态度；②阵型、首发、替补深度与攻守转换；③可靠伤停影响，无数据必须写“伤停信息未确认”，严禁虚构；④当前价格相对比赛概率的价值变化。
- 允许使用升盘、降盘、退盘、阻上、低水、高水、赔付压力、肋部、边路纵深、高位压迫、攻守转换等专业术语，但要说清它对比赛的含义。
- headline、plain_talk、buy_talk 禁止出现：AI、模型名、凯利、价值差、泊松、xG、λ、去水。
- 价值看 Bet365，不要用竞彩 SP 判断值不值。票面只告诉用户买哪个玩法。
- pick_1x2 只选一侧；scores 给两个合理的情景比分，只用于表达比赛路径，严禁暗示精确命中；buy_talk 必须依次写“格局、主方向、次方向、比分、防范”，不能含糊。
- 主推必须同时有盘口、人员或价值依据；拿不准写关注或放弃。禁止写可看。关注表示可以留意，不是建议现在买。
- plain_talk 禁止使用“看起来、感觉、别追太死、可以看看”等泛化口语。每一个方向都必须落到盘位、水位、阵型对位、人员结构或定价信号上。
- buy_talk 必须以“参考买入：”开头，使用“主方向、次方向、防范、失效条件”等专业表达，不得使用“梭哈、稳胆、必赢、包中”。
- buy_talk 必须明确这是条件性参考，指出至少一个会导致观点失效的临场风险；不要承诺结果或收益。
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
