package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"jingcai-ai/internal/analyze"
	"jingcai-ai/internal/config"
	"jingcai-ai/internal/grok"
	"jingcai-ai/internal/httpapi"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/scheduler"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
)

func main() {
	cfg := config.Load()
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	var models []*grok.Client
	if cfg.APIKey != "" {
		models = append(models, grok.NewNamed("Grok", cfg.APIKey, cfg.BaseURL, cfg.Model))
	}
	if cfg.DeepSeekKey != "" {
		models = append(models, grok.NewNamed("DeepSeek", cfg.DeepSeekKey, cfg.DeepSeekBase, cfg.DeepSeekModel))
	}
	if cfg.ShqbbKey != "" {
		models = append(models, grok.NewNamed("ChatGPT", cfg.ShqbbKey, cfg.ShqbbBase, cfg.GPTModel))
		models = append(models, grok.NewNamed("Claude", cfg.ShqbbKey, cfg.ShqbbBase, cfg.ClaudeModel))
	}
	eng := &analyze.Engine{Store: st, Models: models}
	names := make([]string, 0, len(models))
	logs := make([]string, 0, len(models))
	for _, m := range models {
		names = append(names, m.Name)
		logs = append(logs, m.Name+"/"+m.Model)
	}
	if len(models) == 0 {
		log.Print("未配置模型密钥：仍会抓竞彩并写库，白话用本地模板")
	} else {
		log.Printf("模型已启用：%s", strings.Join(logs, ", "))
	}

	sched := scheduler.New(st, sporttery.New(), market.New(), eng, cfg.Location)
	if err := sched.Start(); err != nil {
		log.Fatal(err)
	}
	defer sched.Stop()

	webDir := "web/dist"
	if _, err := os.Stat(webDir); err != nil {
		webDir = ""
	}
	api := &httpapi.Server{
		Store:     st,
		Location:  cfg.Location,
		Refresh:   sched.RefreshNow,
		WebDir:   webDir,
		Models:   names,
	}
	log.Printf("listening %s  sqlite=%s/jingcai.db", cfg.Listen, cfg.DataDir)
	if err := http.ListenAndServe(cfg.Listen, api.Handler()); err != nil {
		log.Fatal(err)
	}
}
