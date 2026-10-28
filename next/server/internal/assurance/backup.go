package assurance

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// BackupDatabase creates an independent SQLite snapshot and its credential key.
// The new directory is exclusively created; existing backups are never overwritten.
func BackupDatabase(ctx context.Context, source, destination string) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("数据库必须为普通文件")
	}
	key, err := backupKey(source + ".key")
	if err != nil {
		return err
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("备份目录必须为尚不存在的新目录: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(destination)
		}
	}()
	target := filepath.Join(destination, "assurance.sqlite3")
	db, err := backupOpen(source, true)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", target); err != nil {
		return err
	}
	if err = os.Chmod(target, 0600); err != nil {
		return err
	}
	if err = os.WriteFile(target+".key", key, 0600); err != nil {
		return err
	}
	files := map[string]string{}
	for _, name := range []string{"assurance.sqlite3", "assurance.sqlite3.key"} {
		hash, err := fileSHA(filepath.Join(destination, name))
		if err != nil {
			return err
		}
		files[name] = hash
	}
	manifest, _ := json.MarshalIndent(Object{"schema": "jieheng.backup/1", "created_at": Stamp(), "files": files}, "", "  ")
	if err = os.WriteFile(filepath.Join(destination, "manifest.json"), manifest, 0600); err != nil {
		return err
	}
	if err = VerifyBackup(ctx, destination); err != nil {
		return err
	}
	success = true
	return nil
}
func backupOpen(path string, readOnly bool) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: abs}
	q := u.Query()
	if readOnly {
		q.Set("mode", "ro")
	}
	q.Set("_busy_timeout", "5000")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err == nil {
		db.SetMaxOpenConns(1)
	}
	return db, err
}
func backupKey(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("凭据主密钥须为权限 0600 的普通文件")
	}
	key, err := os.ReadFile(path)
	if err == nil && len(key) != 32 {
		err = fmt.Errorf("凭据主密钥长度无效")
	}
	return key, err
}
func fileSHA(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("备份文件必须为普通文件")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func VerifyBackup(ctx context.Context, directory string) error {
	data, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	manifest, err := DecodeJSON(data)
	if err != nil {
		return err
	}
	if manifest["schema"] != "jieheng.backup/1" {
		return fmt.Errorf("备份格式不受支持")
	}
	for _, name := range []string{"assurance.sqlite3", "assurance.sqlite3.key"} {
		hash, err := fileSHA(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		if dObj(manifest["files"])[name] != hash {
			return fmt.Errorf("备份校验失败: %s", name)
		}
	}
	if _, err = backupKey(filepath.Join(directory, "assurance.sqlite3.key")); err != nil {
		return err
	}
	db, err := backupOpen(filepath.Join(directory, "assurance.sqlite3"), true)
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("数据库完整性检查未通过")
	}
	return nil
}

// RestoreBackup writes only into a new directory, keeping the running database intact.
func RestoreBackup(ctx context.Context, source, destination string) error {
	if err := VerifyBackup(ctx, source); err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("还原目录必须为尚不存在的新目录: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(destination)
		}
	}()
	for _, name := range []string{"assurance.sqlite3", "assurance.sqlite3.key"} {
		in, err := os.Open(filepath.Join(source, name))
		if err != nil {
			return err
		}
		out, err := os.OpenFile(filepath.Join(destination, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		syncErr := out.Sync()
		closeErr := out.Close()
		in.Close()
		for _, e := range []error{copyErr, syncErr, closeErr} {
			if e != nil {
				return e
			}
		}
		want, err := fileSHA(filepath.Join(source, name))
		if err != nil {
			return err
		}
		got, err := fileSHA(filepath.Join(destination, name))
		if err != nil || got != want {
			return fmt.Errorf("还原文件校验失败: %s", name)
		}
	}
	db, err := backupOpen(filepath.Join(destination, "assurance.sqlite3"), false)
	if err != nil {
		return err
	}
	defer db.Close()
	// A restored service must require fresh sign-in, never revive old sessions.
	if _, err = db.ExecContext(ctx, "DELETE FROM enterprise_sessions"); err != nil {
		return err
	}
	success = true
	return nil
}
