package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"jingcai-ai/internal/analyze"
	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/store"
)

type Server struct {
	Store         *store.Store
	Location      *time.Location
	Refresh       func() error
	SFCRefresh    func() error
	WebDir        string
	Models        []string
	AdminUsername string
	AdminPassword string
	AdminPath     string
	CookieSecure  bool
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/access/status", s.accessStatus)
	mux.HandleFunc("POST /api/access/redeem", s.accessRedeem)
	mux.HandleFunc("POST /api/access/logout", s.accessLogout)
	mux.HandleFunc("POST /api/admin/login", s.adminLogin)
	mux.HandleFunc("POST /api/admin/logout", s.adminLogout)
	mux.HandleFunc("GET /api/admin/status", s.adminStatus)
	mux.HandleFunc("GET /api/admin/access-codes", s.adminCodes)
	mux.HandleFunc("POST /api/admin/access-codes/generate", s.adminGenerate)
	mux.HandleFunc("POST /api/admin/access-codes/{id}/terminate", s.adminTerminate)
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/today", s.today)
	protected.HandleFunc("GET /api/week", s.week)
	protected.HandleFunc("GET /api/sfc", s.sfc)
	protected.HandleFunc("GET /api/experts", s.experts)
	protected.HandleFunc("GET /api/matches/{id}", s.match)
	protected.HandleFunc("POST /api/admin/refresh", s.refresh)
	mux.Handle("/api/", s.accessMiddleware(protected))
	if s.WebDir != "" {
		if _, err := os.Stat(s.WebDir); err == nil {
			mux.Handle("/", spa(s.WebDir))
		}
	}
	return withCORS(s.adminPath(mux))
}

func (s *Server) adminPath(next http.Handler) http.Handler {
	// Admin UI is only served at the configured non-public path.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/admin") || r.URL.Path == "/admin" {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) accessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/admin/") {
			if !s.isAdmin(r) {
				writeJSON(w, 401, map[string]any{"error": "admin login required"})
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie("jc_access")
		if err != nil {
			writeJSON(w, 401, map[string]any{"error": "access required"})
			return
		}
		if _, err = s.Store.ValidateAccess(c.Value, time.Now()); err != nil {
			writeJSON(w, 401, map[string]any{"error": "access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAdmin(r *http.Request) bool {
	c, e := r.Cookie("jc_admin")
	return e == nil && s.Store.ValidateAdminSession(c.Value, time.Now())
}
func setToken(w http.ResponseWriter, name, val string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: val, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure, MaxAge: maxAge})
}

func (s *Server) accessStatus(w http.ResponseWriter, r *http.Request) {
	c, e := r.Cookie("jc_access")
	if e != nil {
		writeJSON(w, 200, map[string]any{"authorized": false, "reason": "missing"})
		return
	}
	g, e := s.Store.ValidateAccess(c.Value, time.Now())
	if e != nil {
		writeJSON(w, 200, map[string]any{"authorized": false, "reason": "expired"})
		return
	}
	writeJSON(w, 200, map[string]any{"authorized": true, "durationDays": g.DurationDays, "activatedAt": g.ActivatedAt, "expiresAt": g.ExpiresAt, "remainingSeconds": int(time.Until(g.ExpiresAt).Seconds())})
}
func (s *Server) accessRedeem(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || len(store.NormalizeCode(in.Code)) != 10 {
		writeJSON(w, 400, map[string]any{"error": "访问码无效或已使用"})
		return
	}
	g, t, e := s.Store.RedeemAccessCode(in.Code, r.RemoteAddr, time.Now())
	if e != nil {
		writeJSON(w, 400, map[string]any{"error": "访问码无效或已使用"})
		return
	}
	setToken(w, "jc_access", t, int(time.Until(g.ExpiresAt).Seconds()), s.CookieSecure)
	writeJSON(w, 200, map[string]any{"authorized": true, "durationDays": g.DurationDays, "expiresAt": g.ExpiresAt})
}
func (s *Server) accessLogout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("jc_access"); e == nil {
		s.Store.RevokeAccessSession(c.Value)
	}
	setToken(w, "jc_access", "", -1, s.CookieSecure)
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if s.AdminPassword == "" || in.Username != s.AdminUsername || in.Password != s.AdminPassword {
		writeJSON(w, 401, map[string]any{"error": "账号或密码错误"})
		return
	}
	t, e := s.Store.CreateAdminSession(time.Now())
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	setToken(w, "jc_admin", t, 43200, s.CookieSecure)
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (s *Server) adminLogout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("jc_admin"); e == nil {
		s.Store.RevokeAdminSession(c.Value)
	}
	setToken(w, "jc_admin", "", -1, s.CookieSecure)
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (s *Server) adminStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"authenticated": s.isAdmin(r)})
}
func (s *Server) adminCodes(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "admin login required"})
		return
	}
	now := time.Now()
	q := r.URL.Query()
	days, _ := strconv.Atoi(q.Get("days"))
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	list, e := s.Store.ListAccessCodes(days, q.Get("status"), q.Get("q"), page, pageSize, now)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	pools, e := s.Store.AccessPoolStats(now)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]any{"codes": list.Codes, "total": list.Total, "page": list.Page, "pageSize": list.PageSize, "pages": list.Pages, "pools": pools})
}
func (s *Server) adminGenerate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "admin login required"})
		return
	}
	var in struct{ DurationDays, Count int }
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.Count < 1 || in.Count > 10000 || (in.DurationDays != 3 && in.DurationDays != 7 && in.DurationDays != 15 && in.DurationDays != 30) {
		writeJSON(w, 400, map[string]any{"error": "参数无效"})
		return
	}
	if e := s.Store.EnsureAccessPool(in.DurationDays, in.Count); e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (s *Server) adminTerminate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "admin login required"})
		return
	}
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || s.Store.TerminateAccessCode(id, time.Now()) != nil {
		writeJSON(w, 400, map[string]any{"error": "终止失败"})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) sfc(w http.ResponseWriter, r *http.Request) {
	board, err := s.Store.LatestSFC()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stale := board == nil || time.Since(board.FetchedAt) > 15*time.Minute
	if stale && s.SFCRefresh != nil {
		if e := s.SFCRefresh(); e != nil {
			log.Printf("sfc refresh: %v", e)
		} else if next, e2 := s.Store.LatestSFC(); e2 == nil && next != nil {
			board = next
		}
	}
	if board == nil {
		writeJSON(w, 200, map[string]any{"issue": "", "matches": []any{}, "note": "本期对阵尚未同步"})
		return
	}
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location)
	}
	type item struct {
		No          int     `json:"no"`
		League      string  `json:"league"`
		Kickoff     string  `json:"kickoff"`
		Home        string  `json:"home"`
		Away        string  `json:"away"`
		Asian       string  `json:"asian,omitempty"`
		HomeWin     float64 `json:"homeWin"`
		Draw        float64 `json:"draw"`
		AwayWin     float64 `json:"awayWin"`
		MarketHome  float64 `json:"marketHome"`
		MarketDraw  float64 `json:"marketDraw"`
		MarketAway  float64 `json:"marketAway"`
		Pick        string  `json:"pick"`
		Source      string  `json:"source"`
		MatchID     *int64  `json:"matchId,omitempty"`
		NumStr      string  `json:"numStr,omitempty"`
		JingcaiHome string  `json:"jingcaiHome,omitempty"`
		JingcaiAway string  `json:"jingcaiAway,omitempty"`
		Talk        string  `json:"talk,omitempty"`
		Market      any     `json:"market,omitempty"`
		Handicap    any     `json:"handicap,omitempty"`
		Eval        any     `json:"eval,omitempty"`
	}
	out := make([]item, 0, len(board.Matches))
	analyzed := 0
	for _, row := range board.Matches {
		it := item{
			No: row.No, League: row.League, Kickoff: row.Kickoff, Home: row.Home, Away: row.Away, Asian: row.Asian,
			HomeWin: row.MarketHome, Draw: row.MarketDraw, AwayWin: row.MarketAway, Source: "均赔",
			MarketHome: row.MarketHome, MarketDraw: row.MarketDraw, MarketAway: row.MarketAway,
		}
		if m := s.Store.MatchSFC(row, now); m != nil {
			id := m.ID
			it.MatchID = &id
			it.NumStr = m.NumStr
			it.JingcaiHome, it.JingcaiAway = m.Home, m.Away
			if sn, e := s.Store.PreferredSnapshot(m.ID); e == nil && sn != nil && (sn.Result.HomeWin+sn.Result.Draw+sn.Result.AwayWin) > 1 {
				it.HomeWin, it.Draw, it.AwayWin = sn.Result.HomeWin, sn.Result.Draw, sn.Result.AwayWin
				it.Source = "研判"
				analyzed++
			}
		}
		if it.Source != "研判" && row.AnalyzedHome+row.AnalyzedDraw+row.AnalyzedAway > 1 {
			it.HomeWin, it.Draw, it.AwayWin = row.AnalyzedHome, row.AnalyzedDraw, row.AnalyzedAway
			it.Source = "研判"
			analyzed++
		}
		it.Pick = sfcPick(it.HomeWin, it.Draw, it.AwayWin)
		if it.MatchID == nil {
			talk, hc, sides := analyze.PackMarket(row.Home, row.Away, it.HomeWin, it.Draw, it.AwayWin, row.Quote)
			it.Talk, it.Handicap, it.Eval = talk, hc, sides
			if row.Quote != nil {
				it.Market = row.Quote
			}
		}
		out = append(out, it)
	}
	writeJSON(w, 200, map[string]any{
		"issue":     board.Issue,
		"fetchedAt": board.FetchedAt,
		"matches":   out,
		"analyzed":  analyzed,
		"total":     len(out),
	})
}

func sfcPick(h, d, a float64) string {
	switch {
	case h >= d && h >= a:
		return "胜"
	case a >= d && a >= h:
		return "负"
	default:
		return "平"
	}
}

func (s *Server) week(w http.ResponseWriter, r *http.Request) {
	from := store.WeekStart(time.Now().In(s.Location))
	list, err := s.Store.ListBetween(from, from.AddDate(0, 0, 7))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type item struct {
		store.MatchRow
		Status        string `json:"status"`
		Kind          string `json:"kind,omitempty"`
		AnalysisCount int    `json:"analysisCount"`
		HasMarket     bool   `json:"hasMarket"`
		HasPreview    bool   `json:"hasPreview"`
	}
	out := make([]item, 0, len(list))
	for _, m := range list {
		it := item{MatchRow: m, Status: "待分析"}
		sn, _ := s.Store.PreferredSnapshot(m.ID)
		if sn != nil {
			it.Kind = string(sn.Kind)
			it.AnalysisCount = len(sn.Takes)
			if it.AnalysisCount == 0 && sn.Kind == store.KindClose {
				if open, _ := s.Store.GetSnapshot(m.ID, store.KindOpen); open != nil {
					it.AnalysisCount = len(open.Takes)
				}
			}
			it.Status = "赛前"
			if sn.Kind == store.KindClose {
				it.Status = "临场"
			}
		}
		if m.Finished {
			it.Status = "完场"
		}
		if q, err := s.Store.GetQuote(m.ID); err == nil && q != nil {
			it.HasMarket = true
		}
		if p, err := s.Store.GetPreview(m.ID); err == nil && p != nil {
			it.HasPreview = true
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":    from.Format("2006-01-02"),
		"to":      from.AddDate(0, 0, 6).Format("2006-01-02"),
		"matches": out,
		"total":   len(out),
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "models": s.Models, "experts": experts.Catalog(s.Models)})
}

func (s *Server) today(w http.ResponseWriter, r *http.Request) {
	from := time.Now().In(s.Location).Add(-4 * time.Hour)
	list, err := s.Store.ListUpcoming(from)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	type shapeJSON struct {
		HomeWin float64 `json:"homeWin"`
		Draw    float64 `json:"draw"`
		AwayWin float64 `json:"awayWin"`
		Over25  float64 `json:"over25"`
	}
	type item struct {
		store.MatchRow
		Status string        `json:"status"`
		Kind   string        `json:"kind"`
		Odds   *eval.Board   `json:"odds,omitempty"`
		Market *market.Quote `json:"market,omitempty"`
		Shape  *shapeJSON    `json:"shape,omitempty"`
	}
	out := make([]item, 0, len(list))
	finished := 0
	for _, m := range list {
		if m.Finished {
			finished++
			continue
		}
		it := item{MatchRow: m, Status: "分析中"}
		sn, err := s.Store.PreferredSnapshot(m.ID)
		if err == nil && sn != nil {
			it.Kind = string(sn.Kind)
			it.Odds = eval.BoardFromJSON(sn.OddsJSON)
			it.Shape = &shapeJSON{HomeWin: sn.Result.HomeWin, Draw: sn.Result.Draw, AwayWin: sn.Result.AwayWin, Over25: sn.Result.Over25}
			if m.Finished {
				it.Status = "完场"
			} else if sn.Kind == store.KindClose {
				it.Status = "临场"
			} else {
				it.Status = "赛前"
			}
		}
		if q, err := s.Store.GetQuote(m.ID); err == nil {
			it.Market = q
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"matches":  out,
		"finished": finished,
		"source":   "sqlite",
	})
}

func (s *Server) match(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	m, err := s.Store.GetMatch(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", 404)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	kinds, _ := s.Store.AvailableKinds(id)
	want := store.SnapshotKind(strings.TrimSpace(r.URL.Query().Get("kind")))
	var sn *store.Snapshot
	if want == store.KindOpen || want == store.KindClose {
		sn, err = s.Store.GetSnapshot(id, want)
	} else {
		sn, err = s.Store.PreferredSnapshot(id)
	}
	if errors.Is(err, sql.ErrNoRows) || sn == nil {
		q, _ := s.Store.GetQuote(id)
		prev, _ := s.Store.GetPreview(id)
		writeJSON(w, http.StatusOK, map[string]any{
			"match":     m,
			"available": kinds,
			"status":    "分析中",
			"source":    "sqlite",
			"market":    q,
			"preview":   prev,
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	q, _ := s.Store.GetQuote(id)
	prev, _ := s.Store.GetPreview(id)
	pub := store.ToPublic(sn, q)
	open, _ := s.Store.GetSnapshot(id, store.KindOpen)
	closeSn, _ := s.Store.GetSnapshot(id, store.KindClose)
	expertKind := sn.Kind
	if len(pub.Takes) == 0 && sn.Kind == store.KindClose && open != nil && len(open.Takes) > 0 {
		pub.Takes = append([]store.ModelTake(nil), open.Takes...)
		expertKind = store.KindOpen
	}
	pub.Takes = experts.WithBaseline(pub.Takes, sn)
	var oddsOpen, oddsClose *eval.Board
	if open != nil {
		oddsOpen = eval.BoardFromJSON(open.OddsJSON)
	}
	if closeSn != nil {
		oddsClose = eval.BoardFromJSON(closeSn.OddsJSON)
	}
	gradeTakes(pub.Takes, m, hhadLineOf(pub.Odds, oddsOpen, oddsClose))
	writeJSON(w, http.StatusOK, map[string]any{
		"match":      m,
		"available":  kinds,
		"status":     statusOfMatch(m, sn.Kind),
		"source":     "sqlite",
		"snapshot":   pub,
		"oddsOpen":   oddsOpen,
		"oddsClose":  oddsClose,
		"market":     q,
		"preview":    prev,
		"expertKind": expertKind,
	})
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if s.Refresh == nil {
		http.Error(w, "refresh unavailable", 500)
		return
	}
	go func() {
		if err := s.Refresh(); err != nil {
			log.Printf("refresh: %v", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "note": "已开始抓数；已有快照不会再打 AI"})
}

func (s *Server) experts(w http.ResponseWriter, r *http.Request) {
	from := store.WeekStart(time.Now().In(s.Location))
	settled, err := s.Store.ListFinishedSince(from)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	board := map[string]*experts.BoardRow{}
	base := experts.Of(experts.BaselineName)
	board[experts.BaselineName] = &experts.BoardRow{Name: experts.BaselineName, Role: base.Title, RoleKey: base.Key}
	for _, n := range s.Models {
		r := experts.Of(n)
		board[n] = &experts.BoardRow{Name: n, Role: r.Title, RoleKey: r.Key}
	}
	type item struct {
		Match      store.MatchRow     `json:"match"`
		Score      string             `json:"score"`
		HHADLine   string             `json:"hhadLine,omitempty"`
		ActualHHAD string             `json:"actualHhad,omitempty"`
		ExpertKind store.SnapshotKind `json:"expertKind,omitempty"`
		Takes      []store.ModelTake  `json:"takes"`
	}
	recent := make([]item, 0, len(settled))
	yesterday := make([]item, 0)
	pending := 0
	yestBiz := time.Now().In(s.Location).Add(-4*time.Hour).AddDate(0, 0, -1).Format("2006-01-02")
	for _, m := range settled {
		sn, err := s.Store.AuditSnapshot(m.ID)
		if err != nil {
			sn = nil
		}
		takes := []store.ModelTake{}
		line := hhadLineFromSnapshot(sn)
		if sn != nil {
			takes = experts.WithBaseline(sn.Takes, sn)
		}
		if len(takes) > 0 {
			takes = append([]store.ModelTake(nil), takes...)
			gradeTakes(takes, &m, line)
		}
		hasVoice := false
		for _, t := range takes {
			if t.Name != experts.BaselineName {
				hasVoice = true
				break
			}
		}
		if !hasVoice {
			pending++
		}
		hg, ag := *m.HomeGoals, *m.AwayGoals
		actualHC := ""
		if v, ok := experts.ParseHHADLine(line); ok {
			actualHC = experts.ActualHHAD(hg, ag, v)
		}
		for _, t := range takes {
			row := board[t.Name]
			if row == nil {
				r := experts.Of(t.Name)
				row = &experts.BoardRow{Name: t.Name, Role: r.Title, RoleKey: r.Key}
				board[t.Name] = row
			}
			row.Games++
			g := experts.GradeTake(t, hg, ag, line)
			if g.Hit1X2 {
				row.Hit1X2++
			}
			if g.HitOU {
				row.HitOU++
			}
			if g.HasHC {
				row.GamesHC++
				if g.HitHC {
					row.HitHC++
				}
			}
			if g.HasScore {
				row.GamesScore++
				if g.HitScore {
					row.HitScore++
				}
			}
			row.Points += g.Points
		}
		it := item{
			Match:      m,
			Score:      fmt.Sprintf("%d-%d", hg, ag),
			HHADLine:   line,
			ActualHHAD: actualHC,
			Takes:      takes,
		}
		if sn != nil {
			it.ExpertKind = sn.Kind
		}
		if m.BusinessDate == yestBiz {
			yesterday = append(yesterday, it)
		}
		recent = append(recent, it)
	}
	rows := make([]experts.BoardRow, 0, len(board))
	if board[experts.BaselineName] != nil {
		rows = append(rows, *board[experts.BaselineName])
		delete(board, experts.BaselineName)
	}
	for _, n := range s.Models {
		if board[n] != nil {
			rows = append(rows, *board[n])
			delete(board, n)
		}
	}
	for _, r := range board {
		rows = append(rows, *r)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"board":     experts.Board(rows),
		"yesterday": yesterday,
		"settled":   recent,
		"pending":   pending,
		"weekFrom":  from.Format("2006-01-02"),
		"source":    "sqlite",
	})
}

func gradeTakes(takes []store.ModelTake, m *store.MatchRow, hhadLine string) {
	for i := range takes {
		experts.Decorate(&takes[i])
		if m != nil && m.Finished && m.HomeGoals != nil && m.AwayGoals != nil {
			g := experts.GradeTake(takes[i], *m.HomeGoals, *m.AwayGoals, hhadLine)
			h, o := g.Hit1X2, g.HitOU
			takes[i].Hit1X2 = &h
			takes[i].HitOU = &o
			if g.HasHC {
				hc := g.HitHC
				takes[i].HitHC = &hc
			}
			if g.HasScore {
				sc := g.HitScore
				takes[i].HitScore = &sc
			}
		}
	}
}

func hhadLineFromSnapshot(sn *store.Snapshot) string {
	if sn == nil {
		return ""
	}
	return hhadLineOf(eval.BoardFromJSON(sn.OddsJSON))
}

func hhadLineOf(boards ...*eval.Board) string {
	for _, b := range boards {
		if b != nil && strings.TrimSpace(b.HHADLine) != "" {
			return strings.TrimSpace(b.HHADLine)
		}
	}
	return ""
}

func statusOfMatch(m *store.MatchRow, k store.SnapshotKind) string {
	if m != nil && m.Finished {
		return "完场"
	}
	return statusOf(k)
}

func statusOf(k store.SnapshotKind) string {
	if k == store.KindClose {
		return "临场"
	}
	return "赛前"
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spa(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
