package config

import (
	"bufio"
	"os"
	"strings"
	"time"
)

type Config struct {
	APIKey         string
	BaseURL        string
	Model          string
	DeepSeekKey    string
	DeepSeekBase   string
	DeepSeekModel  string
	ShqbbKey       string
	ShqbbBase      string
	ClaudeModel    string
	GPTModel       string
	Listen         string
	DataDir        string
	AILogDir       string
	AILogRetention time.Duration
	FetchProxy     string
	Location       *time.Location
	AdminUsername  string
	AdminPassword  string
	CookieSecure   bool
	AdminPath      string
}

func Load() Config {
	loadDotEnv(".env")
	locName := getenv("TZ", "Asia/Shanghai")
	loc, err := time.LoadLocation(locName)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return Config{
		APIKey:         os.Getenv("APINEBULA_API_KEY"),
		BaseURL:        getenv("APINEBULA_BASE_URL", "https://apinebula.ai/v1"),
		Model:          getenv("GROK_MODEL", "grok-4.6"),
		DeepSeekKey:    os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBase:   getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
		DeepSeekModel:  getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		ShqbbKey:       os.Getenv("SHQBB_API_KEY"),
		ShqbbBase:      getenv("SHQBB_BASE_URL", "https://api.shqbb.com/v1"),
		ClaudeModel:    getenv("CLAUDE_MODEL", "claude-haiku-4-5-20251001"),
		GPTModel:       getenv("GPT_MODEL", "gpt-5.4-mini"),
		Listen:         getenv("LISTEN_ADDR", ":8080"),
		DataDir:        getenv("DATA_DIR", "data"),
		AILogDir:       getenv("AI_LOG_DIR", "logs/ai"),
		AILogRetention: durationEnv("AI_LOG_RETENTION", 48*time.Hour),
		FetchProxy:     os.Getenv("FETCH_PROXY"),
		Location:       loc,
		AdminUsername:  getenv("ADMIN_USERNAME", "admin"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		CookieSecure:   strings.EqualFold(os.Getenv("ACCESS_COOKIE_SECURE"), "true"),
		AdminPath:      normalizePath(getenv("ADMIN_PATH", "/console-k7m4x9")),
	}
}

func normalizePath(v string) string {
	v = "/" + strings.Trim(strings.TrimSpace(v), "/")
	if v == "/" {
		return "/console-k7m4x9"
	}
	return v
}

func durationEnv(k string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
