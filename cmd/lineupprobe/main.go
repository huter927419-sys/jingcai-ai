package main

import (
	"fmt"
	"log"
	"strings"

	"jingcai-ai/internal/config"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/grok"
)

func main() {
	cfg := config.Load()
	claude := grok.NewNamed("Claude", cfg.ClaudeKey, cfg.ShqbbBase, cfg.ClaudeModel)
	if !claude.Enabled() {
		log.Fatal("Claude not configured")
	}
	role := experts.Of("Claude")
	jobs := []struct {
		num, title, body string
	}{
		{"周五010", "阿森纳 vs 考文垂", `场次 周五010 英超 阿森纳 vs 考文垂。竞彩主胜 SP 偏低，客胜高水。
赛前伤停/停赛（公开名单，不是官方正式首发；预测十一人与伤停名单可能重叠，重叠者按未确认处理）：
主队阿森纳：William Saliba 背部伤，预计 2026 年 10 月下旬回归；Jurriën Timber 脚踝伤，约 1–2 周；Bruno Guimarães 伤缺，约 1–2 周。预测阵型 4-3-3。预测首发：Raya，White，Mosquera，Gabriel，Calafiori，Ødegaard，Lewis-Skelly，Rice，Madueke，Havertz，Tzolis。
客队考文垂：Haji Wright 肌肉伤，预计 11 月初；Ephron Mason-Clark 腘绳肌，疑似出战；Jack Rudoni 肩伤，疑似出战。预测阵型 4-3-3。预测首发：Rushworth，van Ewijk，Bobby Thomas，Amenda，Dasilva，Yirenkyi，Grimes，Torp，Tchaouna，Simms，Thomas-Asante。
正式首发仍可能变化。没有列入的伤停写未确认，严禁补造。请输出 JSON。`},
		{"周五009", "马赛 vs 斯特拉斯堡", `场次 周五009 法甲 马赛 vs 斯特拉斯堡。竞彩主胜 SP 约 1.65。
赛前伤停/停赛（公开名单，不是官方正式首发）：
主队马赛：Quinten Timber 停赛，预计 8 月下旬期满；Alexi Koum 停赛，预计 8 月下旬期满。预测阵型 4-2-3-1。预测首发：De Lange，Weah，Balerdi，Cornelius，Emerson，Nnadi，Højbjerg，Harit，Angel Gomes，Paixão，Gouiri。
客队斯特拉斯堡：Joaquín Panichelli 十字韧带类伤，预计 12 月初回归，本场确定缺阵。预测阵型 4-2-3-1。预测首发：Jörgensen，Guéla Doué，Omobamidele，Doukouré，Chilwell，Oyedele，El Mourabet，Reyna，Nanasi，Godo，Mara。
正式首发仍可能变化。没有列入的伤停写未确认，严禁补造。请输出 JSON。`},
		{"周五005", "天狼星 vs 赫根", `场次 周五005 瑞典超 天狼星 vs 赫根。竞彩主胜 SP 约 1.70。
赛前伤停/停赛（公开名单，不是官方正式首发；预测十一人与伤停名单重叠者按未确认）：
主队天狼星：Jesper Uneken 伤缺，预计 9 月初；Matthias Nartey 伤缺，预计 9 月初；Otso Liimatta 伤缺预计 9 月中，但预测首发仍列入其名，按能否出场未确认。预测阵型 4-3-3。预测首发：Celic，Castegren，Soumah，Anker，Krusnell，Victor Svensson，Heier，Lindberg，Milleskog，Liimatta，Bjerkebo。
客队赫根：Andreas Linde 腿伤，预计 9 月初；Filip Öhman 肩伤，预计 9 月初；Ben Engdahl 伤缺至赛季报销。预测阵型 4-4-2。预测首发：Berisha，Ibrahim，Väisänen，Helander，Lundkvist，Julius Lindberg，Seger，Doumbia，Rygaard，Lindgren，Svanbäck。
正式首发仍可能变化。没有列入的伤停写未确认，严禁补造。请输出 JSON。`},
	}
	for _, j := range jobs {
		p := fmt.Sprintf("角色：%s。%s\n%s\n只根据赛前阵容和伤停判断对位与攻守结构，不要编造未提供的伤停。plain_talk 必须点名至少两名缺阵/停赛球员及其影响，并写明正式首发仍可能变化。", role.Title, role.Hint, j.body)
		fmt.Printf("\n======== %s %s ========\n", j.num, j.title)
		out, err := claude.Analyze(p)
		if err != nil {
			fmt.Println("Claude ERR:", err)
			continue
		}
		fmt.Println("model: Claude")
		fmt.Println("headline:", out.Headline)
		fmt.Println("verdict:", out.Verdict, "1x2:", out.Pick1X2, "ou:", out.PickOU, "hhad:", out.PickHandicap)
		fmt.Println("pattern:", out.Pattern)
		fmt.Println("plain_talk:\n", strings.TrimSpace(out.PlainTalk))
		fmt.Println("buy_talk:\n", strings.TrimSpace(out.BuyTalk))
	}
}
