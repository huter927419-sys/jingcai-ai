package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

var ErrInvalidCode = errors.New("invalid access code")

type AccessCode struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	DurationDays int        `json:"durationDays"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	ActivatedAt  *time.Time `json:"activatedAt,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	TerminatedAt *time.Time `json:"terminatedAt,omitempty"`
	LastSeenAt   *time.Time `json:"lastSeenAt,omitempty"`
	ActivationIP string     `json:"activationIp,omitempty"`
	UseCount     int        `json:"useCount"`
}

type AccessGrant struct {
	CodeID                 int64
	DurationDays           int
	ActivatedAt, ExpiresAt time.Time
}

func hashSecret(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func NormalizeCode(v string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(v)))
}

func NewDisplayCode() (string, error) {
	b := make([]byte, 10)
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(buf[i])%len(codeAlphabet)]
	}
	return string(b[:5]) + "-" + string(b[5:]), nil
}

func (s *Store) EnsureAccessPool(days, available int) error {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM access_codes WHERE duration_days=? AND activated_at IS NULL AND terminated_at IS NULL`, days).Scan(&n)
	if err != nil {
		return err
	}
	for n < available {
		code, err := NewDisplayCode()
		if err != nil {
			return err
		}
		_, err = s.DB.Exec(`INSERT OR IGNORE INTO access_codes(code_hash,code_display,duration_days,created_at) VALUES(?,?,?,?)`, hashSecret(NormalizeCode(code)), code, days, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		if changes, _ := s.changes(); changes > 0 {
			n++
		}
	}
	return nil
}

func (s *Store) changes() (int64, error) {
	var n int64
	err := s.DB.QueryRow(`SELECT changes()`).Scan(&n)
	return n, err
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) RedeemAccessCode(code, ip string, now time.Time) (AccessGrant, string, error) {
	tx, err := s.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return AccessGrant{}, "", err
	}
	defer tx.Rollback()
	var id int64
	var days int
	var activated, terminated sql.NullString
	err = tx.QueryRow(`SELECT id,duration_days,activated_at,terminated_at FROM access_codes WHERE code_hash=?`, hashSecret(NormalizeCode(code))).Scan(&id, &days, &activated, &terminated)
	if err != nil || activated.Valid || terminated.Valid {
		return AccessGrant{}, "", ErrInvalidCode
	}
	start, expires := now.UTC(), now.UTC().AddDate(0, 0, days)
	res, err := tx.Exec(`UPDATE access_codes SET activated_at=?,expires_at=?,last_seen_at=?,activation_ip=?,use_count=1 WHERE id=? AND activated_at IS NULL AND terminated_at IS NULL`, start.Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano), start.Format(time.RFC3339Nano), ip, id)
	if err != nil {
		return AccessGrant{}, "", err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return AccessGrant{}, "", ErrInvalidCode
	}
	token, err := randomToken()
	if err != nil {
		return AccessGrant{}, "", err
	}
	_, err = tx.Exec(`INSERT INTO access_sessions(code_id,token_hash,created_at,last_seen_at) VALUES(?,?,?,?)`, id, hashSecret(token), start.Format(time.RFC3339Nano), start.Format(time.RFC3339Nano))
	if err != nil {
		return AccessGrant{}, "", err
	}
	if err = tx.Commit(); err != nil {
		return AccessGrant{}, "", err
	}
	return AccessGrant{CodeID: id, DurationDays: days, ActivatedAt: start, ExpiresAt: expires}, token, nil
}

func (s *Store) ValidateAccess(token string, now time.Time) (AccessGrant, error) {
	var g AccessGrant
	var a, e string
	err := s.DB.QueryRow(`SELECT c.id,c.duration_days,c.activated_at,c.expires_at FROM access_sessions s JOIN access_codes c ON c.id=s.code_id WHERE s.token_hash=? AND s.revoked_at IS NULL AND c.terminated_at IS NULL AND c.activated_at IS NOT NULL AND c.expires_at>?`, hashSecret(token), now.UTC().Format(time.RFC3339Nano)).Scan(&g.CodeID, &g.DurationDays, &a, &e)
	if err != nil {
		return g, ErrInvalidCode
	}
	g.ActivatedAt, _ = time.Parse(time.RFC3339Nano, a)
	g.ExpiresAt, _ = time.Parse(time.RFC3339Nano, e)
	stamp := now.UTC().Format(time.RFC3339Nano)
	_, _ = s.DB.Exec(`UPDATE access_sessions SET last_seen_at=? WHERE token_hash=?`, stamp, hashSecret(token))
	_, _ = s.DB.Exec(`UPDATE access_codes SET last_seen_at=? WHERE id=?`, stamp, g.CodeID)
	return g, nil
}

func (s *Store) RevokeAccessSession(token string) {
	_, _ = s.DB.Exec(`UPDATE access_sessions SET revoked_at=? WHERE token_hash=?`, time.Now().UTC().Format(time.RFC3339Nano), hashSecret(token))
}

func (s *Store) CreateAdminSession(now time.Time) (string, error) {
	t, e := randomToken()
	if e != nil {
		return "", e
	}
	_, e = s.DB.Exec(`INSERT INTO admin_sessions(token_hash,created_at,expires_at) VALUES(?,?,?)`, hashSecret(t), now.UTC().Format(time.RFC3339Nano), now.UTC().Add(12*time.Hour).Format(time.RFC3339Nano))
	return t, e
}
func (s *Store) ValidateAdminSession(token string, now time.Time) bool {
	var n int
	e := s.DB.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE token_hash=? AND expires_at>?`, hashSecret(token), now.UTC().Format(time.RFC3339Nano)).Scan(&n)
	return e == nil && n == 1
}
func (s *Store) RevokeAdminSession(token string) {
	_, _ = s.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash=?`, hashSecret(token))
}

func scanTime(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, e := time.Parse(time.RFC3339Nano, v.String)
	if e != nil {
		return nil
	}
	return &t
}

type AccessPoolStat struct {
	DurationDays int `json:"durationDays"`
	Total        int `json:"total"`
	Unused       int `json:"unused"`
	Active       int `json:"active"`
	Expired      int `json:"expired"`
	Terminated   int `json:"terminated"`
}

func (s *Store) AccessPoolStats(now time.Time) ([]AccessPoolStat, error) {
	stamp := now.UTC().Format(time.RFC3339Nano)
	rows, err := s.DB.Query(`SELECT duration_days,
		COUNT(*),
		SUM(CASE WHEN activated_at IS NULL AND terminated_at IS NULL THEN 1 ELSE 0 END),
		SUM(CASE WHEN activated_at IS NOT NULL AND terminated_at IS NULL AND expires_at>? THEN 1 ELSE 0 END),
		SUM(CASE WHEN activated_at IS NOT NULL AND terminated_at IS NULL AND expires_at<=? THEN 1 ELSE 0 END),
		SUM(CASE WHEN terminated_at IS NOT NULL THEN 1 ELSE 0 END)
		FROM access_codes GROUP BY duration_days ORDER BY duration_days`, stamp, stamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDays := map[int]AccessPoolStat{}
	for rows.Next() {
		var st AccessPoolStat
		if err = rows.Scan(&st.DurationDays, &st.Total, &st.Unused, &st.Active, &st.Expired, &st.Terminated); err != nil {
			return nil, err
		}
		byDays[st.DurationDays] = st
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := make([]AccessPoolStat, 0, 4)
	for _, d := range []int{3, 7, 15, 30} {
		st := byDays[d]
		st.DurationDays = d
		out = append(out, st)
	}
	return out, nil
}

type AccessListResult struct {
	Codes    []AccessCode `json:"codes"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Pages    int          `json:"pages"`
}

func clampAccessPage(page, size, total int) (int, int, int, int) {
	if size != 20 && size != 50 && size != 100 {
		size = 50
	}
	if page < 1 {
		page = 1
	}
	pages := (total + size - 1) / size
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	return page, size, (page - 1) * size, pages
}

func (s *Store) ListAccessCodes(days int, status, search string, page, pageSize int, now time.Time) (AccessListResult, error) {
	where := []string{"1=1"}
	args := []any{}
	if days == 3 || days == 7 || days == 15 || days == 30 {
		where = append(where, "duration_days=?")
		args = append(args, days)
	}
	search = NormalizeCode(search)
	if search != "" {
		where = append(where, "REPLACE(code_display,'-','') LIKE ?")
		args = append(args, "%"+search+"%")
	}
	switch status {
	case "unused":
		where = append(where, "activated_at IS NULL AND terminated_at IS NULL")
	case "active":
		where = append(where, "activated_at IS NOT NULL AND terminated_at IS NULL AND expires_at>?")
		args = append(args, now.UTC().Format(time.RFC3339Nano))
	case "expired":
		where = append(where, "activated_at IS NOT NULL AND terminated_at IS NULL AND expires_at<=?")
		args = append(args, now.UTC().Format(time.RFC3339Nano))
	case "terminated":
		where = append(where, "terminated_at IS NOT NULL")
	}
	statusExpr := `CASE WHEN terminated_at IS NOT NULL THEN 'terminated' WHEN activated_at IS NULL THEN 'unused' WHEN expires_at<=? THEN 'expired' ELSE 'active' END`
	qwhere := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM access_codes WHERE `+qwhere, args...).Scan(&total); err != nil {
		return AccessListResult{}, err
	}
	page, pageSize, offset, pages := clampAccessPage(page, pageSize, total)
	query := `SELECT id,code_display,duration_days,` + statusExpr + `,created_at,activated_at,expires_at,terminated_at,last_seen_at,COALESCE(activation_ip,''),use_count FROM access_codes WHERE ` + qwhere + ` ORDER BY duration_days ASC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append([]any{now.UTC().Format(time.RFC3339Nano)}, args...)
	queryArgs = append(queryArgs, pageSize, offset)
	rows, err := s.DB.Query(query, queryArgs...)
	if err != nil {
		return AccessListResult{}, err
	}
	defer rows.Close()
	out := []AccessCode{}
	for rows.Next() {
		var c AccessCode
		var cr string
		var a, e, t, l sql.NullString
		if err = rows.Scan(&c.ID, &c.Code, &c.DurationDays, &c.Status, &cr, &a, &e, &t, &l, &c.ActivationIP, &c.UseCount); err != nil {
			return AccessListResult{}, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, cr)
		c.ActivatedAt = scanTime(a)
		c.ExpiresAt = scanTime(e)
		c.TerminatedAt = scanTime(t)
		c.LastSeenAt = scanTime(l)
		out = append(out, c)
	}
	if err = rows.Err(); err != nil {
		return AccessListResult{}, err
	}
	return AccessListResult{Codes: out, Total: total, Page: page, PageSize: pageSize, Pages: pages}, nil
}

func (s *Store) TerminateAccessCode(id int64, now time.Time) error {
	stamp := now.UTC().Format(time.RFC3339Nano)
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE access_codes SET terminated_at=? WHERE id=? AND terminated_at IS NULL`, stamp, id); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE access_sessions SET revoked_at=? WHERE code_id=? AND revoked_at IS NULL`, stamp, id); e != nil {
		return e
	}
	return tx.Commit()
}
