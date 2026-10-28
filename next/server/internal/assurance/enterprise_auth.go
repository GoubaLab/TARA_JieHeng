package assurance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var dummyLoginHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("not-a-valid-enterprise-login"), 12)
	if err != nil {
		panic(err)
	}
	return hash
}()

type EnterpriseAuth struct{ origin string }
type principalContextKey struct{}

func principal(r *http.Request) Object {
	p, _ := r.Context().Value(principalContextKey{}).(map[string]any)
	return p
}
func actorID(r *http.Request) string {
	if p := principal(r); p != nil {
		return dStr(p["id"])
	}
	return "local"
}
func roleLevel(role string) int {
	return map[string]int{"viewer": 1, "analyst": 2, "reviewer": 3, "admin": 4}[role]
}
func NewEnterpriseHTTPHandler(repo *Repository, catalog *Catalog, root string, logger *log.Logger, origin string) (http.Handler, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("企业模式须配置不含路径的 HTTPS --public-origin，并由本机反向代理终止 TLS")
	}
	var n int
	if e = repo.db.QueryRow("SELECT COUNT(*) FROM enterprise_users WHERE json_extract(body,'$.role')='admin' AND json_extract(body,'$.enabled')=1").Scan(&n); e != nil {
		return nil, e
	}
	if n == 0 {
		return nil, fmt.Errorf("企业模式尚无管理员，请先用 --bootstrap-admin 与 --bootstrap-password-file 初始化")
	}
	s := NewHTTPHandler(repo, catalog, root, logger).(*HTTPServer)
	s.auth = &EnterpriseAuth{origin: strings.TrimRight(origin, "/")}
	return s, nil
}
func (s *HTTPServer) Close() { s.ai.Close() }
func (r *Repository) BootstrapAdmin(ctx context.Context, username, password string) error {
	if len(password) < 12 || len(password) > 72 {
		return dError("管理员初始口令须为 12–72 字节")
	}
	username = strings.TrimSpace(username)
	if len(username) < 2 || len(username) > 80 {
		return dError("管理员账号须为 2–80 字节")
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), 12)
	if e != nil {
		return e
	}
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM enterprise_users").Scan(&n); e != nil {
		return e
	}
	if n != 0 {
		return dError("已存在企业账户，不能重复初始化")
	}
	id := Identity("user")
	body, _ := repoBody(Object{"id": id, "username": username, "name": username, "role": "admin", "enabled": true, "revision": 0, "created_at": Stamp()})
	if _, e = tx.ExecContext(ctx, "INSERT INTO enterprise_users(id,username,body,password_hash) VALUES(?,?,?,?)", id, username, body, string(hash)); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *HTTPServer) authRoutes() {
	for _, v := range []struct {
		method, path string
		fn           endpoint
	}{{"GET", "auth/session", s.session}, {"POST", "auth/sessions", s.login}, {"POST", "auth/logout", s.logout}, {"GET", "enterprise/users", s.users}, {"POST", "enterprise/users", s.saveUser}, {"PUT", "enterprise/users/{resource_id}", s.saveUser}, {"GET", "workspaces/{workspace_id}/members", s.members}, {"PUT", "workspaces/{workspace_id}/members/{resource_id}", s.saveMember}} {
		s.route(v.method+" "+apiPrefix+"/"+v.path, v.fn)
	}
}
func (s *HTTPServer) enterpriseRequest(r *http.Request) (*http.Request, error) {
	u, _ := url.Parse(s.auth.origin)
	if !strings.EqualFold(r.Host, u.Host) {
		return r, fail(400, "访问地址不匹配", "请通过配置的企业 HTTPS 入口访问")
	}
	ip, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil || net.ParseIP(ip) == nil || !net.ParseIP(ip).IsLoopback() {
		return r, fail(403, "入口不受信任", "企业服务仅接受本机反向代理连接")
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("X-Workspace-Client") != "jieheng" {
			return r, fail(403, "缺少应用标识", "请通过企业工作台提交")
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.auth.origin {
			return r, fail(403, "来源不匹配", "请求必须来自当前企业入口")
		}
	}
	if cookie, e := r.Cookie("jieheng_session"); e == nil {
		sum := sha256.Sum256([]byte(cookie.Value))
		token := hex.EncodeToString(sum[:])
		var raw string
		e = s.repository.db.QueryRowContext(r.Context(), "SELECT u.body FROM enterprise_sessions s JOIN enterprise_users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?", token, Stamp()).Scan(&raw)
		if e == nil {
			user, e := DecodeJSON([]byte(raw))
			if e == nil && dBool(user["enabled"]) {
				r = r.WithContext(context.WithValue(r.Context(), principalContextKey{}, user))
			}
		}
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, apiPrefix+"/") || path == apiPrefix+"/auth/session" || path == apiPrefix+"/auth/sessions" || path == apiPrefix+"/health" {
		return r, nil
	}
	p := principal(r)
	if p == nil {
		return r, fail(401, "需要登录", "请登录企业工作台")
	}
	if p["role"] == "admin" {
		return r, nil
	}
	rel := strings.TrimPrefix(path, apiPrefix+"/")
	parts := strings.Split(rel, "/")
	write := r.Method != "GET" && r.Method != "HEAD"
	if strings.HasPrefix(rel, "enterprise/") || rel == "platform-events" {
		return r, fail(403, "需要管理员权限", "该操作仅限企业管理员")
	}
	if strings.HasPrefix(rel, "model-provider") || strings.HasPrefix(rel, "prompt-templates") {
		if write {
			return r, fail(403, "需要管理员权限", "模型服务与组织提示词由管理员维护")
		}
		return r, nil
	}
	if parts[0] == "workspaces" {
		if len(parts) == 1 {
			if write && roleLevel(dStr(p["role"])) < 2 {
				return r, fail(403, "只读账户", "此账户不能创建项目")
			}
			return r, nil
		}
		role, e := s.projectRole(r, parts[1])
		if e != nil {
			return r, e
		}
		if role == "" {
			return r, ErrNotFound
		}
		if len(parts) > 2 && parts[2] == "members" && write {
			return r, fail(403, "需要管理员权限", "项目成员由管理员维护")
		}
		required := 1
		if write {
			required = 2
		}
		if len(parts) > 2 && (parts[2] == "baselines" || parts[2] == "restorations") && write {
			required = 3
		}
		if roleLevel(role) < required {
			return r, fail(403, "项目权限不足", "此操作需要对应的分析或评审角色")
		}
		return r, nil
	}
	if write && roleLevel(dStr(p["role"])) < 2 && rel != "auth/logout" && rel != "knowledge-searches" {
		return r, fail(403, "只读账户", "此账户没有写入权限")
	}
	return r, nil
}
func (s *HTTPServer) projectRole(r *http.Request, id string) (string, error) {
	if s.auth == nil {
		return "admin", nil
	}
	p := principal(r)
	if p == nil {
		return "", fail(401, "需要登录", "请登录企业工作台")
	}
	if p["role"] == "admin" {
		return "admin", nil
	}
	var role string
	e := s.repository.db.QueryRowContext(r.Context(), "SELECT role FROM enterprise_members WHERE workspace_id=? AND user_id=?", id, p["id"]).Scan(&role)
	if e != nil {
		return "", nil
	}
	if roleLevel(dStr(p["role"])) < roleLevel(role) {
		role = dStr(p["role"])
	}
	return role, nil
}
func (s *HTTPServer) readProject(r *http.Request, id string) error {
	if id == "" || s.auth == nil {
		return nil
	}
	role, e := s.projectRole(r, id)
	if e != nil {
		return e
	}
	if role == "" {
		return ErrNotFound
	}
	return nil
}
func (s *HTTPServer) manageKnowledge(r *http.Request, id string) error {
	if s.auth == nil {
		return nil
	}
	p := principal(r)
	if p == nil {
		return fail(401, "需要登录", "请登录企业工作台")
	}
	if p["role"] == "admin" {
		return nil
	}
	if id == "" {
		return fail(403, "需要管理员权限", "组织级知识资料由管理员维护")
	}
	role, e := s.projectRole(r, id)
	if e != nil {
		return e
	}
	if roleLevel(role) < 2 {
		return fail(403, "项目权限不足", "需要该项目的分析权限")
	}
	return nil
}
func (s *HTTPServer) session(w http.ResponseWriter, r *http.Request) error {
	if s.auth == nil {
		return writeJSON(w, r, 200, Object{"mode": "local", "user": Object{"id": "local", "name": "本地管理员", "role": "admin"}})
	}
	return writeJSON(w, r, 200, Object{"mode": "enterprise", "user": principal(r)})
}
func (s *HTTPServer) login(w http.ResponseWriter, r *http.Request) error {
	if s.auth == nil {
		return fail(409, "本地模式无需登录", "企业账号仅在企业部署模式启用")
	}
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	username, password := strings.TrimSpace(dStr(p["username"])), dStr(p["password"])
	if len(username) < 2 || len(username) > 80 || len(password) > 72 {
		return fail(401, "登录失败", "账号或口令不正确")
	}
	limitKey := Digest(Object{"username": strings.ToLower(username)})
	var failures int
	var until string
	s.repository.db.QueryRowContext(r.Context(), "SELECT failures,until_at FROM enterprise_login_limits WHERE key=?", limitKey).Scan(&failures, &until)
	if until > Stamp() && failures >= 5 {
		return fail(429, "登录尝试过多", "请 15 分钟后再试，或联系管理员")
	}
	var raw, hash string
	e = s.repository.db.QueryRowContext(r.Context(), "SELECT body,password_hash FROM enterprise_users WHERE username=?", username).Scan(&raw, &hash)
	user, _ := DecodeJSON([]byte(raw))
	loginHash := []byte(hash)
	if e != nil {
		loginHash = dummyLoginHash
	}
	passwordErr := bcrypt.CompareHashAndPassword(loginHash, []byte(password))
	if e != nil || !dBool(user["enabled"]) || passwordErr != nil {
		if until <= Stamp() {
			failures = 0
		}
		s.repository.db.ExecContext(r.Context(), "INSERT INTO enterprise_login_limits(key,failures,until_at) VALUES(?,?,?) ON CONFLICT(key) DO UPDATE SET failures=excluded.failures,until_at=excluded.until_at", limitKey, failures+1, time.Now().UTC().Add(15*time.Minute).Format(time.RFC3339))
		return fail(401, "登录失败", "账号或口令不正确")
	}
	b := make([]byte, 32)
	if _, e = rand.Read(b); e != nil {
		return e
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(8 * time.Hour)
	tx, e := s.repository.db.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO enterprise_sessions(token_hash,user_id,expires_at) VALUES(?,?,?)", hex.EncodeToString(sum[:]), user["id"], expires.Format(time.RFC3339)); e != nil {
		return e
	}
	tx.ExecContext(r.Context(), "DELETE FROM enterprise_login_limits WHERE key=? OR until_at<?", limitKey, Stamp())
	tx.ExecContext(r.Context(), "DELETE FROM enterprise_sessions WHERE expires_at<?", Stamp())
	if e = platformEvent(r.Context(), tx, dStr(user["id"]), "login", dStr(user["id"]), Object{}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "jieheng_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 8 * 3600})
	return writeJSON(w, r, 200, Object{"mode": "enterprise", "user": user})
}
func (s *HTTPServer) logout(w http.ResponseWriter, r *http.Request) error {
	if cookie, e := r.Cookie("jieheng_session"); e == nil {
		sum := sha256.Sum256([]byte(cookie.Value))
		s.repository.db.ExecContext(r.Context(), "DELETE FROM enterprise_sessions WHERE token_hash=?", hex.EncodeToString(sum[:]))
	}
	http.SetCookie(w, &http.Cookie{Name: "jieheng_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	return writeJSON(w, r, 200, Object{"ok": true})
}
func (s *HTTPServer) users(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.db.QueryContext(r.Context(), "SELECT body FROM enterprise_users ORDER BY username")
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return e
		}
		o, _ := DecodeJSON([]byte(raw))
		out = append(out, o)
	}
	return writeJSON(w, r, 200, Object{"items": out, "mode": func() string {
		if s.auth == nil {
			return "local"
		}
		return "enterprise"
	}()})
}
func (s *HTTPServer) saveUser(w http.ResponseWriter, r *http.Request) error {
	if s.auth == nil {
		return fail(409, "请使用企业部署模式", "本地模式不维护企业账户")
	}
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	for key := range p {
		if !map[string]bool{"username": true, "name": true, "role": true, "enabled": true, "password": true}[key] {
			return dError("未知账户字段")
		}
	}
	for _, key := range []string{"username", "name"} {
		if _, e = dValidateField(ds(2, 80), p[key]); e != nil {
			return dError("账号与姓名须为 2–80 字")
		}
	}
	if roleLevel(dStr(p["role"])) == 0 {
		return dError("账户角色无效")
	}
	enabled, ok := p["enabled"].(bool)
	if !ok {
		return dError("启用状态必须为布尔值")
	}
	password := dStr(p["password"])
	id := r.PathValue("resource_id")
	revision := int64(-1)
	if id != "" {
		revision, e = expectedRevision(r)
		if e != nil {
			return e
		}
	} else {
		id = Identity("user")
	}
	hash := ""
	if password != "" {
		if len(password) < 12 || len(password) > 72 {
			return dError("口令须为 12–72 字节")
		}
		b, e := bcrypt.GenerateFromPassword([]byte(password), 12)
		if e != nil {
			return e
		}
		hash = string(b)
	} else if revision < 0 {
		return dError("新账户须设置初始口令")
	}
	tx, e := s.repository.db.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	old, e := readObject(tx.QueryRowContext(r.Context(), "SELECT body FROM enterprise_users WHERE id=?", id))
	if e != nil && e != ErrNotFound {
		return e
	}
	if revision >= 0 && (old == nil || int64(dInt(old["revision"])) != revision) {
		return &RevisionError{Current: int64(dInt(old["revision"]))}
	}
	if old != nil && old["role"] == "admin" && dBool(old["enabled"]) && (!enabled || p["role"] != "admin") {
		var n int
		tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM enterprise_users WHERE id<>? AND json_extract(body,'$.role')='admin' AND json_extract(body,'$.enabled')=1", id).Scan(&n)
		if n == 0 {
			return dError("必须保留至少一个启用的管理员")
		}
	}
	if hash == "" {
		if e = tx.QueryRowContext(r.Context(), "SELECT password_hash FROM enterprise_users WHERE id=?", id).Scan(&hash); e != nil {
			return e
		}
	}
	user := Object{"id": id, "username": p["username"], "name": p["name"], "role": p["role"], "enabled": enabled, "revision": revision + 1, "updated_at": Stamp(), "created_at": Stamp()}
	if old != nil {
		user["created_at"] = old["created_at"]
	}
	raw, _ := repoBody(user)
	_, e = tx.ExecContext(r.Context(), "INSERT INTO enterprise_users(id,username,body,password_hash) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET username=excluded.username,body=excluded.body,password_hash=excluded.password_hash", id, p["username"], raw, hash)
	if e != nil {
		return &ConflictError{Detail: "账号名称已存在或账户未能保存"}
	}
	if _, e = tx.ExecContext(r.Context(), "DELETE FROM enterprise_sessions WHERE user_id=?", id); e != nil {
		return e
	}
	if e = platformEvent(r.Context(), tx, actorID(r), "save-user", id, Object{"role": user["role"], "enabled": enabled, "credentials_rotated": password != ""}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	setETag(w, user)
	return writeJSON(w, r, 200, user)
}
func (s *HTTPServer) members(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.db.QueryContext(r.Context(), "SELECT m.user_id,m.role,u.body FROM enterprise_members m JOIN enterprise_users u ON u.id=m.user_id WHERE m.workspace_id=?", r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []Object{}
	for rows.Next() {
		var id, role, raw string
		if e = rows.Scan(&id, &role, &raw); e != nil {
			return e
		}
		u, _ := DecodeJSON([]byte(raw))
		out = append(out, Object{"user_id": id, "name": u["name"], "username": u["username"], "role": role, "enabled": u["enabled"]})
	}
	return writeJSON(w, r, 200, Object{"items": out})
}
func (s *HTTPServer) saveMember(w http.ResponseWriter, r *http.Request) error {
	if s.auth == nil {
		return fail(409, "请使用企业部署模式", "本地模式没有项目成员")
	}
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	role := dStr(p["role"])
	if role != "none" && (roleLevel(role) < 1 || roleLevel(role) > 3) {
		return dError("项目角色须为 viewer、analyst、reviewer 或 none")
	}
	uid, wid := r.PathValue("resource_id"), r.PathValue("workspace_id")
	tx, e := s.repository.db.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = readObject(tx.QueryRowContext(r.Context(), "SELECT body FROM enterprise_users WHERE id=?", uid)); e != nil {
		return e
	}
	if _, e = readObject(tx.QueryRowContext(r.Context(), "SELECT body FROM workspaces WHERE id=?", wid)); e != nil {
		return e
	}
	if role == "none" {
		_, e = tx.ExecContext(r.Context(), "DELETE FROM enterprise_members WHERE workspace_id=? AND user_id=?", wid, uid)
	} else {
		_, e = tx.ExecContext(r.Context(), "INSERT INTO enterprise_members(workspace_id,user_id,role) VALUES(?,?,?) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=excluded.role", wid, uid, role)
	}
	if e != nil {
		return e
	}
	if e = platformEvent(r.Context(), tx, actorID(r), "set-project-member", wid, Object{"user_id": uid, "role": role}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"workspace_id": wid, "user_id": uid, "role": role})
}

var _ = errors.Is

func (r *Repository) listForUser(ctx context.Context, userID string, offset, limit int) ([]Object, int, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, dError("分页范围无效")
	}
	var total int
	if e := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspaces w JOIN enterprise_members m ON m.workspace_id=w.id WHERE m.user_id=?", userID).Scan(&total); e != nil {
		return nil, 0, e
	}
	rows, e := r.db.QueryContext(ctx, "SELECT w.body,m.role FROM workspaces w JOIN enterprise_members m ON m.workspace_id=w.id WHERE m.user_id=? ORDER BY w.rowid DESC LIMIT ? OFFSET ?", userID, limit, offset)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []Object{}
	for rows.Next() {
		var raw, role string
		if e = rows.Scan(&raw, &role); e != nil {
			return nil, 0, e
		}
		ws, e := DecodeJSON([]byte(raw))
		if e != nil {
			return nil, 0, e
		}
		item := Object{"project_role": role}
		for _, key := range []string{"id", "title", "description", "revision", "updated_at"} {
			item[key] = ws[key]
		}
		for _, key := range []string{"assets", "scenarios", "baselines"} {
			item[key] = len(dList(ws[key]))
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
