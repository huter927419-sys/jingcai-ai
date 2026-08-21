package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"jingcai-ai/internal/config"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/grok"
)

type injuryJob struct {
	Num           string `json:"num"`
	League        string `json:"league"`
	Home          string `json:"home"`
	Away          string `json:"away"`
	HomeFormation string `json:"home_formation"`
	AwayFormation string `json:"away_formation"`
	HomeStarters  string `json:"home_starters"`
	AwayStarters  string `json:"away_starters"`
	HomeOut       string `json:"home_out"`
	AwayOut       string `json:"away_out"`
	HomeN         int    `json:"home_n"`
	AwayN         int    `json:"away_n"`
	LineupType    string `json:"lineup_type"`
	SP            string `json:"sp"`
}

func main() {
	cfg := config.Load()
	claude := grok.NewNamed("Claude", cfg.ClaudeKey, cfg.ShqbbBase, cfg.ClaudeModel)
	if !claude.Enabled() {
		log.Fatal("Claude not configured")
	}
	path := "/tmp/injury-jobs.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var jobs []injuryJob
	if err := json.Unmarshal(raw, &jobs); err != nil {
		log.Fatal(err)
	}
	role := experts.Of("Claude")
	ok := 0
	for i, j := range jobs {
		if j.HomeN+j.AwayN == 0 {
			fmt.Printf("\n======== %s %s vs %s ========\nskip: 伤停名单未列入\n", j.Num, j.Home, j.Away)
			continue
		}
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		kind := "公开伤停/停赛名单，不是官方正式首发"
		switch j.LineupType {
		case "predicted":
			kind += "；阵型为首发预测"
		case "lastStarting11":
			kind += "；阵型为上场首发沿用，本场可能变化"
		}
		body := fmt.Sprintf(`场次 %s %s %s vs %s。%s
赛前伤停/停赛（%s；预测十一人与伤停名单重叠者按未确认处理）：
主队%s：%s。预测阵型 %s。预测/参考首发：%s。
客队%s：%s。预测阵型 %s。预测/参考首发：%s。
正式首发仍可能变化。没有列入的伤停写未确认，严禁补造。请输出 JSON。`,
			j.Num, j.League, j.Home, j.Away, firstNonEmpty(j.SP, "竞彩 SP 见票面。"),
			kind, j.Home, j.HomeOut, j.HomeFormation, j.HomeStarters,
			j.Away, j.AwayOut, j.AwayFormation, j.AwayStarters)
		p := fmt.Sprintf("角色：%s。%s\n%s\n只根据赛前阵容和伤停判断对位与攻守结构，不要编造未提供的伤停。plain_talk 必须点名至少两名缺阵/停赛球员及其影响，并写明正式首发仍可能变化。", role.Title, role.Hint, body)
		fmt.Printf("\n======== %s %s vs %s ========\n", j.Num, j.Home, j.Away)
		out, err := claude.Analyze(p)
		if err != nil {
			fmt.Println("Claude ERR:", err)
			continue
		}
		ok++
		fmt.Println("model: Claude")
		fmt.Println("headline:", out.Headline)
		fmt.Println("verdict:", out.Verdict, "1x2:", out.Pick1X2, "ou:", out.PickOU, "hhad:", out.PickHandicap)
		fmt.Println("pattern:", out.Pattern)
		fmt.Println("plain_talk:\n", strings.TrimSpace(out.PlainTalk))
		fmt.Println("buy_talk:\n", strings.TrimSpace(out.BuyTalk))
	}
	fmt.Printf("\nclaude_ok %d\n", ok)
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
