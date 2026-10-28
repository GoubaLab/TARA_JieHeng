package assurance

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const platformSchema = `
CREATE TABLE IF NOT EXISTS platform_resources(kind TEXT NOT NULL,id TEXT NOT NULL,body TEXT NOT NULL,PRIMARY KEY(kind,id));
CREATE TABLE IF NOT EXISTS platform_events(id TEXT PRIMARY KEY,at TEXT NOT NULL,actor TEXT NOT NULL,action TEXT NOT NULL,resource_id TEXT NOT NULL,detail TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS ai_jobs(id TEXT PRIMARY KEY,workspace_id TEXT NOT NULL,request_key TEXT NOT NULL,fingerprint TEXT NOT NULL,body TEXT NOT NULL,UNIQUE(workspace_id,request_key));
CREATE TABLE IF NOT EXISTS workspace_checkpoints(id TEXT PRIMARY KEY,workspace_id TEXT NOT NULL,body TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS enterprise_users(id TEXT PRIMARY KEY,username TEXT UNIQUE NOT NULL,body TEXT NOT NULL,password_hash TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS enterprise_sessions(token_hash TEXT PRIMARY KEY,user_id TEXT NOT NULL,expires_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS enterprise_members(workspace_id TEXT NOT NULL,user_id TEXT NOT NULL,role TEXT NOT NULL,PRIMARY KEY(workspace_id,user_id));
CREATE TABLE IF NOT EXISTS enterprise_login_limits(key TEXT PRIMARY KEY,failures INTEGER NOT NULL,until_at TEXT NOT NULL);
`

func (r *Repository) initPlatform(path string) error {
	if _, e := r.db.Exec(platformSchema); e != nil {
		return e
	}
	var key []byte
	if path == ":memory:" {
		key = make([]byte, 32)
		if _, e := rand.Read(key); e != nil {
			return e
		}
	} else {
		abs, e := filepath.Abs(path)
		if e != nil {
			return e
		}
		r.dataPath = abs
		file := abs + ".key"
		info, e := os.Lstat(file)
		if e == nil {
			if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
				return fmt.Errorf("模型密钥文件必须为普通文件且仅当前用户可读写：%s", file)
			}
			key, e = os.ReadFile(file)
			if e != nil {
				return e
			}
		} else if errors.Is(e, os.ErrNotExist) {
			var n int
			if e = r.db.QueryRow(`SELECT COUNT(*) FROM platform_resources WHERE kind='provider' AND body LIKE '%secret_ciphertext%'`).Scan(&n); e != nil {
				return e
			}
			if n > 0 {
				return fmt.Errorf("模型密钥文件缺失，请从备份恢复数据库对应的 .key 文件")
			}
			key = make([]byte, 32)
			if _, e = rand.Read(key); e != nil {
				return e
			}
			f, e := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e != nil {
				return e
			}
			_, e = f.Write(key)
			if e == nil {
				e = f.Sync()
			}
			closeErr := f.Close()
			if e != nil {
				return e
			}
			if closeErr != nil {
				return closeErr
			}
		} else {
			return e
		}
	}
	if len(key) != 32 {
		return fmt.Errorf("模型密钥文件无效；请恢复原始 32 字节密钥")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return e
	}
	r.vault, e = cipher.NewGCM(block)
	return e
}
func (r *Repository) encryptSecret(id, value string) (string, error) {
	nonce := make([]byte, r.vault.NonceSize())
	if _, e := io.ReadFull(rand.Reader, nonce); e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(r.vault.Seal(nonce, nonce, []byte(value), []byte(id))), nil
}
func (r *Repository) decryptSecret(id, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	b, e := base64.StdEncoding.DecodeString(value)
	if e != nil || len(b) < r.vault.NonceSize() {
		return "", fmt.Errorf("凭据无法解密，请恢复密钥或重新配置")
	}
	plain, e := r.vault.Open(nil, b[:r.vault.NonceSize()], b[r.vault.NonceSize():], []byte(id))
	if e != nil {
		return "", fmt.Errorf("凭据无法解密，请恢复密钥或重新配置")
	}
	return string(plain), nil
}
func (r *Repository) platformGet(ctx context.Context, kind, id string) (Object, error) {
	return readObject(r.db.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind=? AND id=?", kind, id))
}
func (r *Repository) platformList(ctx context.Context, kind string) ([]Object, error) {
	rows, e := r.db.QueryContext(ctx, "SELECT body FROM platform_resources WHERE kind=? ORDER BY rowid", kind)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		o, e := DecodeJSON([]byte(raw))
		if e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func platformEvent(ctx context.Context, tx *sql.Tx, actor, action, id string, detail Object) error {
	raw, e := repoBody(detail)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO platform_events(id,at,actor,action,resource_id,detail) VALUES(?,?,?,?,?,?)", Identity("pe"), Stamp(), actor, action, id, raw)
	return e
}
func (r *Repository) platformPut(ctx context.Context, kind, id string, expected int64, value Object, actor string) (Object, error) {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	old, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind=? AND id=?", kind, id))
	revision := int64(-1)
	if e == nil {
		revision, _ = repoNumber(old["revision"])
	} else if !errors.Is(e, ErrNotFound) {
		return nil, e
	}
	if expected == -1 && old != nil && creationFingerprint(ctx) != "" {
		if old["_create_fingerprint"] != creationFingerprint(ctx) {
			return nil, &ConflictError{Detail: "同一请求标识对应不同内容"}
		}
		return old, nil
	}
	if revision != expected {
		return nil, &RevisionError{Current: revision}
	}
	out := Clone(value)
	if kind == "knowledge" {
		if e = prepareKnowledgeReview(ctx, tx, id, old, out, actor); e != nil {
			return nil, e
		}
	}
	if old != nil {
		out["_create_fingerprint"] = old["_create_fingerprint"]
	} else if fp := creationFingerprint(ctx); fp != "" {
		out["_create_fingerprint"] = fp
	}
	if kind == "prompt" && old != nil {
		archive, _ := repoBody(old)
		if _, e = tx.ExecContext(ctx, "INSERT OR IGNORE INTO platform_resources(kind,id,body) VALUES('prompt_revision',?,?)", fmt.Sprintf("%s@%d", id, revision), archive); e != nil {
			return nil, e
		}
	}
	out["id"], out["revision"], out["updated_at"] = id, revision+1, Stamp()
	if old != nil {
		out["created_at"] = old["created_at"]
	} else {
		out["created_at"] = Stamp()
	}
	raw, e := repoBody(out)
	if e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO platform_resources(kind,id,body) VALUES(?,?,?) ON CONFLICT(kind,id) DO UPDATE SET body=excluded.body", kind, id, raw)
	if e != nil {
		return nil, e
	}
	if e = platformEvent(ctx, tx, actor, "save-"+kind, id, Object{"revision": revision + 1, "name": out["name"]}); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return out, nil
}
func (r *Repository) platformAudit(ctx context.Context) ([]Object, error) {
	rows, e := r.db.QueryContext(ctx, "SELECT id,at,actor,action,resource_id,detail FROM platform_events ORDER BY rowid DESC LIMIT 200")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, at, actor, action, resource, detail string
		if e = rows.Scan(&id, &at, &actor, &action, &resource, &detail); e != nil {
			return nil, e
		}
		d, _ := DecodeJSON([]byte(detail))
		out = append(out, Object{"id": id, "at": at, "actor": actor, "action": action, "resource_id": resource, "detail": d})
	}
	return out, rows.Err()
}
