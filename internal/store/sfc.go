package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"jingcai-ai/internal/market"
)

func (s *Store) SaveSFC(board *market.SFCBoard) error {
	if board == nil {
		return nil
	}
	raw, err := json.Marshal(board.Matches)
	if err != nil {
		return err
	}
	issue := strings.TrimSpace(board.Issue)
	if issue == "" {
		issue = board.FetchedAt.Format("20060102")
	}
	_, err = s.DB.Exec(`INSERT INTO sfc_issues(issue,fetched_at,matches_json) VALUES(?,?,?)
ON CONFLICT(issue) DO UPDATE SET fetched_at=excluded.fetched_at, matches_json=excluded.matches_json`,
		issue, board.FetchedAt.UTC().Format(time.RFC3339Nano), string(raw))
	return err
}

func (s *Store) LatestSFC() (*market.SFCBoard, error) {
	var issue, fetched, raw string
	err := s.DB.QueryRow(`SELECT issue,fetched_at,matches_json FROM sfc_issues ORDER BY fetched_at DESC LIMIT 1`).Scan(&issue, &fetched, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	board := &market.SFCBoard{Issue: issue}
	board.FetchedAt, _ = time.Parse(time.RFC3339Nano, fetched)
	if board.FetchedAt.IsZero() {
		board.FetchedAt, _ = time.Parse(time.RFC3339, fetched)
	}
	if err := json.Unmarshal([]byte(raw), &board.Matches); err != nil {
		return nil, err
	}
	return board, nil
}

func (s *Store) MatchSFC(row market.SFCMatch, around time.Time) *MatchRow {
	if row.Fid > 0 {
		var id int64
		if err := s.DB.QueryRow(`SELECT match_id FROM market_quotes WHERE fid=?`, row.Fid).Scan(&id); err == nil && id > 0 {
			m, err := s.GetMatch(id)
			if err == nil {
				return m
			}
		}
	}
	from := around.Add(-4 * 24 * time.Hour)
	to := around.Add(5 * 24 * time.Hour)
	list, err := s.ListBetween(from, to)
	if err != nil {
		return nil
	}
	var best MatchRow
	bestScore := 0
	found := false
	for i := range list {
		score := 0
		if teamClose(row.Home, list[i].Home) {
			score += 2
		}
		if teamClose(row.Away, list[i].Away) {
			score += 2
		}
		if score >= 4 && score > bestScore {
			bestScore = score
			best = list[i]
			found = true
		}
	}
	if !found {
		return nil
	}
	return &best
}

func teamClose(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b || strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	ra, rb := []rune(a), []rune(b)
	n := 0
	for i := 0; i < len(ra) && i < len(rb) && ra[i] == rb[i]; i++ {
		if ra[i] > 127 {
			n++
		}
	}
	return n >= 2
}
