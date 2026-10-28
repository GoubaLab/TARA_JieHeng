package assurance

import (
	"context"
	"crypto/cipher"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("记录不存在")

type ConflictError struct{ Detail string }

func (e *ConflictError) Error() string { return e.Detail }

type RevisionError struct{ Current int64 }

func (e *RevisionError) Error() string {
	return fmt.Sprintf("工作区已有更新，请刷新后再保存。当前版本：%d", e.Current)
}

type PrepareChange func(action string, payload, current Object) (Object, error)
type Repository struct {
	db       *sql.DB
	vault    cipher.AEAD
	dataPath string
}

func NewRepository(path string) (*Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("数据库路径不能为空")
	}
	var dsn string
	if path == ":memory:" {
		dsn = "file:" + Identity("memory") + "?mode=memory&cache=shared&_txlock=immediate&_busy_timeout=5000"
	} else {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(abs, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err = f.Chmod(0600); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		u := url.URL{Scheme: "file", Path: abs}
		q := u.Query()
		q.Set("_txlock", "immediate")
		q.Set("_busy_timeout", "5000")
		u.RawQuery = q.Encode()
		dsn = u.String()
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	schema := `PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS workspaces (id TEXT PRIMARY KEY, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS journal (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, revision INTEGER NOT NULL,
 action TEXT NOT NULL, at TEXT NOT NULL, body TEXT NOT NULL, UNIQUE(workspace_id,revision));
 CREATE TABLE IF NOT EXISTS receipts (workspace_id TEXT NOT NULL, request_id TEXT NOT NULL,
 fingerprint TEXT NOT NULL, body TEXT NOT NULL, PRIMARY KEY(workspace_id,request_id));
 CREATE TABLE IF NOT EXISTS baselines (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS creation_receipts (request_id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL,
 workspace_id TEXT NOT NULL, body TEXT NOT NULL);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	r := &Repository{db: db}
	if err = r.initPlatform(path); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}
func (r *Repository) Close() error { return r.db.Close() }
func repoNumber(v any) (int64, error) {
	switch x := v.(type) {
	case json.Number:
		return x.Int64()
	case int:
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		if x == float64(int64(x)) {
			return int64(x), nil
		}
	}
	return 0, fmt.Errorf("无效修订号")
}
func repoBody(v any) (string, error) { b, err := CanonicalJSON(v); return string(b), err }
func readObject(row *sql.Row) (Object, error) {
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return DecodeJSON([]byte(raw))
}
func (r *Repository) Get(ctx context.Context, key string) (Object, error) {
	return readObject(r.db.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", key))
}
func (r *Repository) List(ctx context.Context, offset, limit int) ([]Object, int, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, &ValidationError{Detail: "分页范围无效"}
	}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspaces").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT body FROM workspaces ORDER BY rowid DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []Object{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		ws, e := DecodeJSON([]byte(raw))
		if e != nil {
			return nil, 0, e
		}
		item := Object{}
		for _, k := range []string{"id", "title", "description", "revision", "updated_at"} {
			item[k] = ws[k]
		}
		for _, k := range []string{"assets", "scenarios", "baselines"} {
			item[k] = repoListLen(ws[k])
		}
		result = append(result, item)
	}
	return result, total, rows.Err()
}
func repoListLen(v any) int {
	switch a := v.(type) {
	case []any:
		return len(a)
	case []Object:
		return len(a)
	}
	return 0
}
func validateIdempotency(key string) error {
	if len(key) < 8 || len(key) > 100 {
		return &ValidationError{Detail: "Idempotency-Key 长度必须为 8–100 个字符"}
	}
	for _, c := range key {
		if c < 33 || c > 126 {
			return &ValidationError{Detail: "Idempotency-Key 必须为可见 ASCII 字符"}
		}
	}
	return nil
}
func (r *Repository) Create(ctx context.Context, title, description, key string) (Object, bool, error) {
	if err := validateIdempotency(key); err != nil {
		return nil, false, err
	}
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(description) > 4000 {
		return nil, false, &ValidationError{Detail: "项目名称须为 1–200 字，说明最多 4000 字"}
	}
	fp := Digest(Object{"title": title, "description": description})
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var oldFP, raw string
	err = tx.QueryRowContext(ctx, "SELECT fingerprint,body FROM creation_receipts WHERE request_id=?", key).Scan(&oldFP, &raw)
	if err == nil {
		if oldFP != fp {
			return nil, false, &ConflictError{Detail: "同一请求标识对应了不同内容，请使用新的 Idempotency-Key"}
		}
		ws, e := DecodeJSON([]byte(raw))
		return ws, true, e
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	ws := NewWorkspace(title, description)
	body, err := repoBody(ws)
	if err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO workspaces(id,body) VALUES(?,?)", ws["id"], body); err != nil {
		return nil, false, err
	}
	if user, ok := ctx.Value(principalContextKey{}).(map[string]any); ok {
		role := dStr(user["role"])
		if role == "admin" {
			role = "reviewer"
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO enterprise_members(workspace_id,user_id,role) VALUES(?,?,?)", ws["id"], user["id"], role); err != nil {
			return nil, false, err
		}
	}
	if err = repoEvent(ctx, tx, ws, "open-workspace", Object{"title": title}); err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO creation_receipts(request_id,fingerprint,workspace_id,body) VALUES(?,?,?,?)", key, fp, ws["id"], body); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return ws, false, nil
}
func repoEvent(ctx context.Context, tx *sql.Tx, ws Object, action string, details Object) error {
	if user, ok := ctx.Value(principalContextKey{}).(map[string]any); ok {
		details = Clone(details)
		details["actor"] = user["id"]
	}
	body, err := repoBody(details)
	if err != nil {
		return err
	}
	revision, err := repoNumber(ws["revision"])
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO journal(id,workspace_id,revision,action,at,body) VALUES(?,?,?,?,?,?)", Identity("event"), ws["id"], revision, action, Stamp(), body)
	return err
}
func (r *Repository) Change(ctx context.Context, key string, expected int64, requestID, action string, payload Object, prepare PrepareChange) (Object, bool, error) {
	if err := validateIdempotency(requestID); err != nil {
		return nil, false, err
	}
	if expected < 0 {
		return nil, false, &ValidationError{Detail: "修订号不能为负数"}
	}
	fp := Digest(Object{"action": action, "payload": payload})
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var oldFP, raw string
	err = tx.QueryRowContext(ctx, "SELECT fingerprint,body FROM receipts WHERE workspace_id=? AND request_id=?", key, requestID).Scan(&oldFP, &raw)
	if err == nil {
		result, e := DecodeJSON([]byte(raw))
		if e != nil {
			return nil, false, e
		}
		matches := oldFP == fp
		if !matches {
			rev, e := repoNumber(result["revision"])
			if e != nil {
				return nil, false, e
			}
			legacy := func(p Object) string {
				return Digest(Object{"request_id": requestID, "expected_revision": rev - 1, "action": action, "payload": p})
			}
			matches = oldFP == legacy(payload)
			if !matches && prepare != nil && action == "apply-reference-selection" {
				p, e := prepare(action, Clone(payload), Clone(result))
				if e == nil {
					matches = oldFP == legacy(p)
				}
			}
		}
		if !matches {
			return nil, false, &ConflictError{Detail: "同一请求标识对应了不同内容，请使用新的 Idempotency-Key"}
		}
		return result, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	ws, err := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", key))
	if err != nil {
		return nil, false, err
	}
	revision, err := repoNumber(ws["revision"])
	if err != nil {
		return nil, false, err
	}
	if revision != expected {
		return nil, false, &RevisionError{Current: revision}
	}
	if collection := map[string]string{"record-asset": "assets", "record-scenario": "scenarios", "specify-requirement": "requirements"}[action]; collection != "" {
		if id, supplied := payload["id"].(string); supplied && id != "" {
			found := false
			for _, raw := range ws[collection].([]any) {
				if row, ok := raw.(map[string]any); ok && row["id"] == id {
					found = true
					break
				}
			}
			if !found {
				return nil, false, ErrNotFound
			}
		}
	}
	effective := Clone(payload)
	if prepare != nil {
		effective, err = prepare(action, effective, Clone(ws))
		if err != nil {
			return nil, false, err
		}
	}
	before := Clone(ws)
	next, entityID, err := Apply(ws, action, effective)
	if err != nil {
		return nil, false, err
	}
	body, err := repoBody(next)
	if err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE workspaces SET body=? WHERE id=?", body, key); err != nil {
		return nil, false, err
	}
	eventPayload := effective
	if action == "import-tara" {
		eventPayload = taraSummary(next)
	}
	if err = repoEvent(ctx, tx, next, action, Object{"payload": eventPayload, "entity_id": entityID}); err != nil {
		return nil, false, err
	}
	if action == "seal-baseline" {
		old, err := repoBody(before)
		if err != nil {
			return nil, false, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO baselines(id,workspace_id,body) VALUES(?,?,?)", entityID, key, old); err != nil {
			return nil, false, err
		}
	}
	result := Object{"workspace_id": key, "revision": next["revision"], "entity_id": entityID}
	if action == "replace-architecture" {
		result["asset_sync"] = architectureAssetSummary(before, next)
	}
	if action == "import-tara" {
		result["import_summary"] = taraSummary(next)
	}
	receipt, err := repoBody(result)
	if err != nil {
		return nil, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO receipts(workspace_id,request_id,fingerprint,body) VALUES(?,?,?,?)", key, requestID, fp, receipt); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return result, false, nil
}
func (r *Repository) Baseline(ctx context.Context, key, id string) (Object, error) {
	return readObject(r.db.QueryRowContext(ctx, "SELECT body FROM baselines WHERE id=? AND workspace_id=?", id, key))
}
func (r *Repository) Events(ctx context.Context, key string, offset, limit int) ([]Object, int, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, &ValidationError{Detail: "分页范围无效"}
	}
	if _, err := r.Get(ctx, key); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM journal WHERE workspace_id=?", key).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT id,workspace_id,revision,action,at,body FROM journal WHERE workspace_id=? ORDER BY revision DESC LIMIT ? OFFSET ?", key, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out, err := repoReadEvents(rows)
	return out, total, err
}
func repoReadEvents(rows *sql.Rows) ([]Object, error) {
	out := []Object{}
	for rows.Next() {
		var id, key, action, at, raw string
		var revision int64
		if err := rows.Scan(&id, &key, &revision, &action, &at, &raw); err != nil {
			return nil, err
		}
		body, err := DecodeJSON([]byte(raw))
		if err != nil {
			return nil, err
		}
		out = append(out, Object{"id": id, "workspace_id": key, "revision": json.Number(strconv.FormatInt(revision, 10)), "action": action, "at": at, "body": body})
	}
	return out, rows.Err()
}
func (r *Repository) AllEvents(ctx context.Context, key string) ([]Object, error) {
	if _, err := r.Get(ctx, key); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT id,workspace_id,revision,action,at,body FROM journal WHERE workspace_id=? ORDER BY revision DESC", key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return repoReadEvents(rows)
}
