package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"jingcai-ai/internal/ailog"
	"jingcai-ai/internal/analyze"
	"jingcai-ai/internal/config"
	"jingcai-ai/internal/grok"
	"jingcai-ai/internal/httpapi"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/scheduler"
	"jingcai-ai/internal/sporttery"
	"jingcai-ai/internal/store"
	"jingcai-ai/internal/titan007"
)

func main() {
	cfg := config.Load()
	audit, err := ailog.New(cfg.AILogDir, cfg.AILogRetention)
	if err != nil {
		log.Fatal(err)
	}
	defer audit.Close()
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	for _, days := range []int{3, 7, 15, 30} {
		if err := st.EnsureAccessPool(days, 1000); err != nil {
			log.Fatal(err)
		}
	}

	var models []*grok.Client
	if cfg.APIKey != "" {
		models = append(models, grok.NewNamed("Grok", cfg.APIKey, cfg.BaseURL, cfg.Model))
	}
	if cfg.DeepSeekKey != "" {
		models = append(models, grok.NewNamed("DeepSeek", cfg.DeepSeekKey, cfg.DeepSeekBase, cfg.DeepSeekModel))
	}
	if cfg.ShqbbKey != "" {
		models = append(models, grok.NewNamed("ChatGPT", cfg.ShqbbKey, cfg.ShqbbBase, cfg.GPTModel))
	}
	if cfg.ClaudeKey != "" {
		models = append(models, grok.NewNamed("Claude", cfg.ClaudeKey, cfg.ShqbbBase, cfg.ClaudeModel))
	}
	for _, model := range models {
		model.Audit = audit
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
		log.Printf("AI 审计日志：%s（保留 %s）", cfg.AILogDir, cfg.AILogRetention)
	}

	if cfg.FetchProxy != "" {
		log.Printf("抓盘走代理 %s", cfg.FetchProxy)
	}
	sched := scheduler.New(st, sporttery.New(cfg.FetchProxy), market.New(cfg.FetchProxy), titan007.New(cfg.FetchProxy), eng, cfg.Location)
	if err := sched.Start(); err != nil {
		log.Fatal(err)
	}
	defer sched.Stop()

	webDir := "web/dist"
	if _, err := os.Stat(webDir); err != nil {
		webDir = ""
	}
	api := &httpapi.Server{
		Store:         st,
		Location:      cfg.Location,
		Refresh:       sched.RefreshNow,
		SFCRefresh:    sched.RefreshSFC,
		WebDir:        webDir,
		Models:        names,
		AdminUsername: cfg.AdminUsername, AdminPassword: cfg.AdminPassword, AdminPath: cfg.AdminPath, CookieSecure: cfg.CookieSecure,
	}
	log.Printf("listening %s  sqlite=%s/jingcai.db", cfg.Listen, cfg.DataDir)
	if err := http.ListenAndServe(cfg.Listen, api.Handler()); err != nil {
		log.Fatal(err)
	}
}
