package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jingcai-ai/internal/eval"
	"jingcai-ai/internal/market"
	"jingcai-ai/internal/poisson"
	"jingcai-ai/internal/sporttery"

	_ "modernc.org/sqlite"
)

type SnapshotKind string

const (
	KindOpen  SnapshotKind = "open"
	KindClose SnapshotKind = "close"
)

type MatchRow struct {
	ID           int64     `json:"id"`
	NumStr       string    `json:"numStr"`
	League       string    `json:"league"`
	LeagueAbb    string    `json:"leagueAbb"`
	Home         string    `json:"home"`
	Away         string    `json:"away"`
	Kickoff      time.Time `json:"kickoff"`
	BusinessDate string    `json:"businessDate"`
	HasOpen      bool      `json:"hasOpen"`
	HasClose     bool      `json:"hasClose"`
	HomeGoals    *int      `json:"homeGoals,omitempty"`
	AwayGoals    *int      `json:"awayGoals,omitempty"`
	Finished     bool      `json:"finished"`
}

type Snapshot struct {
	MatchID    int64
	Kind       SnapshotKind
	FetchedAt  time.Time
	OddsJSON   string
	LambdaH    float64
	LambdaA    float64
	Headline   string
	PlainTalk  string
	Result     poisson.Result
	Eval       []eval.Side
	Handicap   *eval.Advice
	UsedAI     bool
	UsedModels []string
	Takes      []ModelTake
	ExpertDone bool
}

type ModelTake struct {
	Name         string   `json:"name"`
	Role         string   `json:"role,omitempty"`
	RoleKey      string   `json:"roleKey,omitempty"`
	Headline     string   `json:"headline"`
	PlainTalk    string   `json:"plainTalk"`
	HomeWin      float64  `json:"homeWin"`
	Draw         float64  `json:"draw"`
	AwayWin      float64  `json:"awayWin"`
	Over25       float64  `json:"over25"`
	Under25      float64  `json:"under25"`
	BuyTalk      string   `json:"buyTalk,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	Scores       []string `json:"scores,omitempty"`
	PickHandicap string   `json:"pickHandicap,omitempty"`
	Pick1X2      string   `json:"pick1x2,omitempty"`
	PickOU       string   `json:"pickOu,omitempty"`
	Verdict      string   `json:"verdict,omitempty"`
	Hit1X2       *bool    `json:"hit1x2,omitempty"`
	HitOU        *bool    `json:"hitOu,omitempty"`
}

type persistedResult struct {
	poisson.Result
	Eval       []eval.Side  `json:"eval,omitempty"`
	Handicap   *eval.Advice `json:"handicap,omitempty"`
	UsedModels []string     `json:"usedModels,omitempty"`
	Takes      []ModelTake  `json:"takes,omitempty"`
	ExpertDone bool         `json:"expertDone,omitempty"`
}

type PublicSnapshot struct {
	Kind       SnapshotKind        `json:"kind"`
	FetchedAt  time.Time           `json:"fetchedAt"`
	Headline   string              `json:"headline"`
	PlainTalk  string              `json:"plainTalk"`
	HomeWin    float64             `json:"homeWin"`
	Draw       float64             `json:"draw"`
	AwayWin    float64             `json:"awayWin"`
	Over25     float64             `json:"over25"`
	Under25    float64             `json:"under25"`
	TopScores  []poisson.ScoreProb `json:"topScores"`
	Grid       [][]float64         `json:"grid"`
	Eval       []eval.Side         `json:"eval"`
	Handicap   *eval.Advice        `json:"handicap,omitempty"`
	Odds       *eval.Board         `json:"odds,omitempty"`
	Market     *market.Quote       `json:"market,omitempty"`
	UsedAI     bool                `json:"usedAI"`
	UsedModels []string            `json:"usedModels,omitempty"`
	Takes      []ModelTake         `json:"takes,omitempty"`
}

type Store struct {
	DB *sql.DB
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "jingcai.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	_, err := s.DB.Exec(`
CREATE TABLE IF NOT EXISTS matches (
  id INTEGER PRIMARY KEY,
  num_str TEXT NOT NULL,
  league TEXT,
  league_abb TEXT,
  home TEXT NOT NULL,
  away TEXT NOT NULL,
  kickoff TEXT NOT NULL,
  business_date TEXT,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS snapshots (
  match_id INTEGER NOT NULL,
  kind TEXT NOT NULL,
  fetched_at TEXT NOT NULL,
  odds_json TEXT,
  lambda_home REAL,
  lambda_away REAL,
  headline TEXT,
  plain_talk TEXT,
  result_json TEXT NOT NULL,
  used_ai INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (match_id, kind),
  FOREIGN KEY (match_id) REFERENCES matches(id)
);
CREATE INDEX IF NOT EXISTS idx_matches_kickoff ON matches(kickoff);
CREATE TABLE IF NOT EXISTS market_quotes (
  match_id INTEGER PRIMARY KEY,
  fid INTEGER,
  fetched_at TEXT NOT NULL,
  quote_json TEXT NOT NULL,
  FOREIGN KEY (match_id) REFERENCES matches(id)
);
CREATE TABLE IF NOT EXISTS match_previews (
  match_id INTEGER PRIMARY KEY,
  fid INTEGER,
  fetched_at TEXT NOT NULL,
  preview_json TEXT NOT NULL,
  FOREIGN KEY (match_id) REFERENCES matches(id)
);
CREATE TABLE IF NOT EXISTS access_codes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code_hash TEXT NOT NULL UNIQUE,
  code_display TEXT NOT NULL UNIQUE,
  duration_days INTEGER NOT NULL CHECK(duration_days IN (3,7,15,30)),
  created_at TEXT NOT NULL,
  activated_at TEXT,
  expires_at TEXT,
  terminated_at TEXT,
  last_seen_at TEXT,
  activation_ip TEXT,
  use_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS access_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code_id INTEGER NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  revoked_at TEXT,
  FOREIGN KEY(code_id) REFERENCES access_codes(id)
);
CREATE TABLE IF NOT EXISTS admin_sessions (
  token_hash TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_access_codes_duration ON access_codes(duration_days);
CREATE INDEX IF NOT EXISTS idx_access_sessions_code ON access_sessions(code_id);
`)
	if err != nil {
		return err
	}
	_ = s.ensureColumn("matches", "home_goals", "INTEGER")
	_ = s.ensureColumn("matches", "away_goals", "INTEGER")
	return nil
}

func (s *Store) ensureColumn(table, col, decl string) error {
	rows, err := s.DB.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == col {
			return nil
		}
	}
	_, err = s.DB.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + col + ` ` + decl)
	return err
}

func (s *Store) UpsertMatch(m sporttery.Match) error {
	_, err := s.DB.Exec(`
INSERT INTO matches (id, num_str, league, league_abb, home, away, kickoff, business_date, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  num_str=excluded.num_str,
  league=excluded.league,
  league_abb=excluded.league_abb,
  home=excluded.home,
  away=excluded.away,
  kickoff=excluded.kickoff,
  business_date=excluded.business_date,
  updated_at=excluded.updated_at
`, m.ID, m.NumStr, m.League, m.LeagueAbb, m.Home, m.Away, m.Kickoff.Format(time.RFC3339), m.BusinessDate, time.Now().Format(time.RFC3339))
	return err
}

func (s *Store) HasSnapshot(id int64, kind SnapshotKind) (bool, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(1) FROM snapshots WHERE match_id=? AND kind=?`, id, kind).Scan(&n)
	return n > 0, err
}

func (s *Store) SaveSnapshot(sn Snapshot) error {
	raw, err := json.Marshal(persistedResult{Result: sn.Result, Eval: sn.Eval, Handicap: sn.Handicap, UsedModels: sn.UsedModels, Takes: sn.Takes, ExpertDone: sn.ExpertDone})
	if err != nil {
		return err
	}
	used := 0
	if sn.UsedAI {
		used = 1
	}
	_, err = s.DB.Exec(`
INSERT INTO snapshots (match_id, kind, fetched_at, odds_json, lambda_home, lambda_away, headline, plain_talk, result_json, used_ai)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(match_id, kind) DO UPDATE SET
  fetched_at=excluded.fetched_at,
  odds_json=excluded.odds_json,
  lambda_home=excluded.lambda_home,
  lambda_away=excluded.lambda_away,
  headline=excluded.headline,
  plain_talk=excluded.plain_talk,
  result_json=excluded.result_json,
  used_ai=excluded.used_ai
`, sn.MatchID, sn.Kind, sn.FetchedAt.Format(time.RFC3339), sn.OddsJSON, sn.LambdaH, sn.LambdaA, sn.Headline, sn.PlainTalk, string(raw), used)
	return err
}

func (s *Store) GetSnapshot(id int64, kind SnapshotKind) (*Snapshot, error) {
	row := s.DB.QueryRow(`
SELECT match_id, kind, fetched_at, odds_json, lambda_home, lambda_away, headline, plain_talk, result_json, used_ai
FROM snapshots WHERE match_id=? AND kind=?`, id, kind)
	return scanSnapshot(row)
}

func (s *Store) PreferredSnapshot(id int64) (*Snapshot, error) {
	sn, err := s.GetSnapshot(id, KindClose)
	if err == nil && sn != nil {
		return sn, nil
	}
	return s.GetSnapshot(id, KindOpen)
}

func (s *Store) AvailableKinds(id int64) ([]SnapshotKind, error) {
	rows, err := s.DB.Query(`SELECT kind FROM snapshots WHERE match_id=? ORDER BY kind DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SnapshotKind
	for rows.Next() {
		var k SnapshotKind
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) GetMatch(id int64) (*MatchRow, error) {
	row := s.DB.QueryRow(matchSelect+` WHERE m.id=?`, id)
	return scanMatch(row)
}

func (s *Store) ListUpcoming(from time.Time) ([]MatchRow, error) {
	rows, err := s.DB.Query(matchSelect+`
WHERE m.kickoff >= ?
ORDER BY m.business_date ASC, CAST(substr(m.num_str, -3) AS INTEGER) ASC, m.kickoff ASC
`, from.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return scanMatches(rows)
}

func (s *Store) ListBetween(from, to time.Time) ([]MatchRow, error) {
	rows, err := s.DB.Query(matchSelect+`
WHERE m.kickoff >= ? AND m.kickoff < ?
ORDER BY m.business_date ASC, CAST(substr(m.num_str, -3) AS INTEGER) ASC, m.kickoff ASC
`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return scanMatches(rows)
}

const matchSelect = `
SELECT m.id, m.num_str, m.league, m.league_abb, m.home, m.away, m.kickoff, m.business_date,
  EXISTS(SELECT 1 FROM snapshots s WHERE s.match_id=m.id AND s.kind='open'),
  EXISTS(SELECT 1 FROM snapshots s WHERE s.match_id=m.id AND s.kind='close'),
  m.home_goals, m.away_goals
FROM matches m
`

func (s *Store) PruneOlderThan(t time.Time) error {
	_, err := s.DB.Exec(`DELETE FROM match_previews WHERE match_id IN (SELECT id FROM matches WHERE kickoff < ? AND home_goals IS NULL)`, t.Format(time.RFC3339))
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`DELETE FROM market_quotes WHERE match_id IN (SELECT id FROM matches WHERE kickoff < ? AND home_goals IS NULL)`, t.Format(time.RFC3339))
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`DELETE FROM snapshots WHERE match_id IN (SELECT id FROM matches WHERE kickoff < ? AND home_goals IS NULL)`, t.Format(time.RFC3339))
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`DELETE FROM matches WHERE kickoff < ? AND home_goals IS NULL`, t.Format(time.RFC3339))
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMatch(row rowScanner) (*MatchRow, error) {
	var m MatchRow
	var kick string
	var open, close int
	var hg, ag sql.NullInt64
	if err := row.Scan(&m.ID, &m.NumStr, &m.League, &m.LeagueAbb, &m.Home, &m.Away, &kick, &m.BusinessDate, &open, &close, &hg, &ag); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, kick)
	if err != nil {
		return nil, err
	}
	m.Kickoff = t
	m.HasOpen = open == 1
	m.HasClose = close == 1
	if hg.Valid && ag.Valid {
		h, a := int(hg.Int64), int(ag.Int64)
		m.HomeGoals = &h
		m.AwayGoals = &a
		m.Finished = true
	}
	return &m, nil
}

func scanMatches(rows *sql.Rows) ([]MatchRow, error) {
	defer rows.Close()
	var out []MatchRow
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func scanSnapshot(row rowScanner) (*Snapshot, error) {
	var sn Snapshot
	var fetched, resultJSON string
	var used int
	if err := row.Scan(&sn.MatchID, &sn.Kind, &fetched, &sn.OddsJSON, &sn.LambdaH, &sn.LambdaA, &sn.Headline, &sn.PlainTalk, &resultJSON, &used); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, fetched)
	if err != nil {
		return nil, err
	}
	sn.FetchedAt = t
	sn.UsedAI = used == 1
	var pr persistedResult
	if err := json.Unmarshal([]byte(resultJSON), &pr); err != nil {
		return nil, err
	}
	sn.Result = pr.Result
	sn.Eval = pr.Eval
	sn.Handicap = pr.Handicap
	sn.UsedModels = pr.UsedModels
	sn.Takes = pr.Takes
	sn.ExpertDone = pr.ExpertDone || len(pr.Takes) > 0
	return &sn, nil
}

func ToPublic(sn *Snapshot, q *market.Quote) PublicSnapshot {
	ev, hc := eval.FromQuote(q, sn.Result, sn.LambdaH, sn.LambdaA)
	return PublicSnapshot{
		Kind:       sn.Kind,
		FetchedAt:  sn.FetchedAt,
		Headline:   sn.Headline,
		PlainTalk:  sn.PlainTalk,
		HomeWin:    sn.Result.HomeWin,
		Draw:       sn.Result.Draw,
		AwayWin:    sn.Result.AwayWin,
		Over25:     sn.Result.Over25,
		Under25:    sn.Result.Under25,
		TopScores:  sn.Result.Top,
		Grid:       sn.Result.Grid,
		Eval:       ev,
		Handicap:   hc,
		Odds:       eval.BoardFromJSON(sn.OddsJSON),
		Market:     q,
		UsedAI:     sn.UsedAI,
		UsedModels: sn.UsedModels,
		Takes:      sn.Takes,
	}
}

func (s *Store) SaveQuote(q *market.Quote) error {
	if q == nil || q.MatchID == 0 {
		return nil
	}
	q.FillImplied()
	raw, err := json.Marshal(q)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`
INSERT INTO market_quotes (match_id, fid, fetched_at, quote_json)
VALUES (?, ?, ?, ?)
ON CONFLICT(match_id) DO UPDATE SET
  fid=excluded.fid,
  fetched_at=excluded.fetched_at,
  quote_json=excluded.quote_json
`, q.MatchID, q.Fid, q.FetchedAt.Format(time.RFC3339), string(raw))
	return err
}

func (s *Store) GetQuote(id int64) (*market.Quote, error) {
	var raw, fetched string
	var fid int64
	err := s.DB.QueryRow(`SELECT fid, fetched_at, quote_json FROM market_quotes WHERE match_id=?`, id).Scan(&fid, &fetched, &raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var q market.Quote
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		return nil, err
	}
	q.MatchID = id
	if q.Fid == 0 {
		q.Fid = fid
	}
	if t, err := time.Parse(time.RFC3339, fetched); err == nil {
		q.FetchedAt = t
	}
	q.FillImplied()
	return &q, nil
}

func (s *Store) SavePreview(p *market.Preview) error {
	if p == nil || p.MatchID == 0 {
		return nil
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`
INSERT INTO match_previews (match_id, fid, fetched_at, preview_json)
VALUES (?, ?, ?, ?)
ON CONFLICT(match_id) DO UPDATE SET
  fid=excluded.fid,
  fetched_at=excluded.fetched_at,
  preview_json=excluded.preview_json
`, p.MatchID, p.Fid, time.Now().Format(time.RFC3339), string(raw))
	return err
}

func (s *Store) GetPreview(id int64) (*market.Preview, error) {
	var raw string
	err := s.DB.QueryRow(`SELECT preview_json FROM match_previews WHERE match_id=?`, id).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var p market.Preview
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	p.MatchID = id
	return &p, nil
}

func (s *Store) QuotesMap(ids []int64) (map[int64]*market.Quote, error) {
	out := map[int64]*market.Quote{}
	if len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		q, err := s.GetQuote(id)
		if err != nil {
			return nil, err
		}
		if q != nil {
			out[id] = q
		}
	}
	return out, nil
}

func (s *Store) SaveFT(id int64, home, away int) error {
	_, err := s.DB.Exec(`UPDATE matches SET home_goals=?, away_goals=?, updated_at=? WHERE id=?`, home, away, time.Now().Format(time.RFC3339), id)
	return err
}

func (s *Store) FidMap() (map[int64]int64, error) {
	rows, err := s.DB.Query(`SELECT match_id, fid FROM market_quotes WHERE fid > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, fid int64
		if err := rows.Scan(&id, &fid); err != nil {
			return nil, err
		}
		out[fid] = id
	}
	return out, rows.Err()
}

func (s *Store) ListSettled(limit int) ([]MatchRow, error) {
	if limit <= 0 {
		limit = 40
	}
	rows, err := s.DB.Query(matchSelect+`
WHERE m.home_goals IS NOT NULL AND m.away_goals IS NOT NULL
ORDER BY m.kickoff DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	return scanMatches(rows)
}

func (s *Store) ListFinishedSince(from time.Time) ([]MatchRow, error) {
	rows, err := s.DB.Query(matchSelect+`
WHERE m.home_goals IS NOT NULL AND m.away_goals IS NOT NULL AND m.kickoff >= ?
ORDER BY m.kickoff DESC
`, from.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return scanMatches(rows)
}

// WeekStart is Thursday 00:00 in loc — 竞彩一周从周四算起。
func WeekStart(now time.Time) time.Time {
	now = now.In(now.Location())
	since := (int(now.Weekday()) + 3) % 7
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return day.AddDate(0, 0, -since)
}
