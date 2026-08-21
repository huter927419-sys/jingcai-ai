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

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/experts"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/store"
)

type Server struct {
	Store    *store.Store
	Location *time.Location
	Refresh  func() error
	WebDir   string
	Models   []string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/today", s.today)
	mux.HandleFunc("GET /api/experts", s.experts)
	mux.HandleFunc("GET /api/matches/{id}", s.match)
	mux.HandleFunc("POST /api/admin/refresh", s.refresh)
	if s.WebDir != "" {
		if _, err := os.Stat(s.WebDir); err == nil {
			mux.Handle("/", spa(s.WebDir))
		}
	}
	return withCORS(mux)
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
	gradeTakes(pub.Takes, m)
	open, _ := s.Store.GetSnapshot(id, store.KindOpen)
	closeSn, _ := s.Store.GetSnapshot(id, store.KindClose)
	var oddsOpen, oddsClose *eval.Board
	if open != nil {
		oddsOpen = eval.BoardFromJSON(open.OddsJSON)
	}
	if closeSn != nil {
		oddsClose = eval.BoardFromJSON(closeSn.OddsJSON)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"match":     m,
		"available": kinds,
		"status":    statusOfMatch(m, sn.Kind),
		"source":    "sqlite",
		"snapshot":  pub,
		"oddsOpen":  oddsOpen,
		"oddsClose": oddsClose,
		"market":    q,
		"preview":   prev,
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
	for _, n := range s.Models {
		r := experts.Of(n)
		board[n] = &experts.BoardRow{Name: n, Role: r.Title, RoleKey: r.Key}
	}
	type item struct {
		Match store.MatchRow    `json:"match"`
		Score string            `json:"score"`
		Takes []store.ModelTake `json:"takes"`
	}
	recent := make([]item, 0, len(settled))
	yesterday := make([]item, 0)
	pending := 0
	yestBiz := time.Now().In(s.Location).Add(-4*time.Hour).AddDate(0, 0, -1).Format("2006-01-02")
	for _, m := range settled {
		sn, err := s.Store.GetSnapshot(m.ID, store.KindOpen)
		if err != nil || sn == nil || len(sn.Takes) == 0 {
			if alt, e2 := s.Store.PreferredSnapshot(m.ID); e2 == nil && alt != nil {
				sn = alt
			}
		}
		takes := []store.ModelTake{}
		if sn != nil && len(sn.Takes) > 0 {
			takes = append([]store.ModelTake(nil), sn.Takes...)
			gradeTakes(takes, &m)
		} else {
			pending++
		}
		hg, ag := *m.HomeGoals, *m.AwayGoals
		for _, t := range takes {
			row := board[t.Name]
			if row == nil {
				r := experts.Of(t.Name)
				row = &experts.BoardRow{Name: t.Name, Role: r.Title, RoleKey: r.Key}
				board[t.Name] = row
			}
			row.Games++
			g := experts.GradeTake(t, hg, ag)
			if g.Hit1X2 {
				row.Hit1X2++
			}
			if g.HitOU {
				row.HitOU++
			}
			row.Points += g.Points
		}
		it := item{
			Match: m,
			Score: fmt.Sprintf("%d-%d", hg, ag),
			Takes: takes,
		}
		if m.BusinessDate == yestBiz {
			yesterday = append(yesterday, it)
		}
		recent = append(recent, it)
	}
	rows := make([]experts.BoardRow, 0, len(board))
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

func gradeTakes(takes []store.ModelTake, m *store.MatchRow) {
	for i := range takes {
		experts.Decorate(&takes[i])
		if m != nil && m.Finished && m.HomeGoals != nil && m.AwayGoals != nil {
			g := experts.GradeTake(takes[i], *m.HomeGoals, *m.AwayGoals)
			h, o := g.Hit1X2, g.HitOU
			takes[i].Hit1X2 = &h
			takes[i].HitOU = &o
		}
	}
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
