package assurance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxHTTPBody int64 = 2_000_000
const apiPrefix = "/api/v1"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
var etagPattern = regexp.MustCompile(`^"(0|[1-9][0-9]*)"$`)

type requestIDKey struct{}
type httpFailure struct {
	status        int
	title, detail string
}

func (e *httpFailure) Error() string              { return e.detail }
func fail(status int, title, detail string) error { return &httpFailure{status, title, detail} }

type HTTPServer struct {
	repository *Repository
	catalog    *Catalog
	root       string
	logger     *log.Logger
	mux        *http.ServeMux
	static     http.Handler
}
type endpoint func(http.ResponseWriter, *http.Request) error

func NewHTTPHandler(repo *Repository, catalog *Catalog, root string, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	s := &HTTPServer{repo, catalog, root, logger, http.NewServeMux(), http.FileServer(http.Dir(filepath.Join(root, "next", "web")))}
	s.routes()
	return s
}
func (s *HTTPServer) route(pattern string, fn endpoint) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if e := fn(w, r); e != nil {
			s.writeError(w, r, e)
		}
	})
}
func (s *HTTPServer) routes() {
	for _, v := range []struct {
		path string
		fn   endpoint
	}{
		{"health", s.health}, {"openapi.json", s.document("openapi.json", "application/json")}, {"docs", s.document("docs.html", "text/html; charset=utf-8")},
		{"workspaces", s.listWorkspaces}, {"workspaces/{workspace_id}", s.workspace}, {"workspaces/{workspace_id}/review", s.review},
		{"workspaces/{workspace_id}/recommendations", s.recommendations}, {"workspaces/{workspace_id}/events", s.events},
		{"workspaces/{workspace_id}/baselines/{baseline_id}", s.baseline}, {"workspaces/{workspace_id}/reports", s.report},
		{"workspaces/{workspace_id}/reference-imports", s.referenceImports},
		{"catalog", s.catalogSummary}, {"catalog/{category}/entries", s.catalogEntries}, {"references/{key}", s.reference},
		{"regulations", s.regulations}, {"regulations/{document_id}/clauses", s.clauses}, {"regulations/{document_id}/sources/{index}", s.source},
	} {
		s.route("GET "+apiPrefix+"/"+v.path, v.fn)
	}
	s.route("POST "+apiPrefix+"/workspaces", s.createWorkspace)
	s.route("POST "+apiPrefix+"/workspaces/{workspace_id}/tara-import-previews", s.taraPreview)
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/tara-imports/{resource_id}", s.taraImportResource)
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/tara-imports/{resource_id}/source", s.taraSource)
	for _, v := range []struct{ path, field string }{{"numbering", "numbering"}, {"item", "item"}, {"architecture", "architecture"}, {"assessment-method", "method"}, {"regulation-references", "regulation_refs"}} {
		s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/"+v.path, s.singleton(v.field))
	}
	for _, c := range []string{"assets", "scenarios", "requirements", "evidence"} {
		s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/"+c+"/{resource_id}", s.resource(c, "id"))
	}
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/damage-scenarios/{resource_id}", s.resource("damage_scenarios", "id"))
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/security-goals/{resource_id}", s.resource("security_goals", "id"))
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/scenarios/{resource_id}/treatment", s.resource("decisions", "scenario_id"))
	for _, v := range []struct {
		method, path, action, idField string
		createID                      bool
	}{
		{"PUT", "numbering", "configure-numbering", "", false}, {"PUT", "item", "define-item", "", false}, {"PUT", "architecture", "replace-architecture", "", false},
		{"POST", "assets", "record-asset", "", true}, {"PUT", "assets/{resource_id}", "record-asset", "id", false},
		{"POST", "scenarios", "record-scenario", "", true}, {"PUT", "scenarios/{resource_id}", "record-scenario", "id", false},
		{"PUT", "scenarios/{resource_id}/treatment", "decide-treatment", "scenario_id", false},
		{"POST", "requirements", "specify-requirement", "", true}, {"PUT", "requirements/{resource_id}", "specify-requirement", "id", false},
		{"POST", "evidence", "record-evidence", "", true}, {"PUT", "assessment-method", "adopt-method", "", false},
		{"POST", "baselines", "seal-baseline", "", false}, {"POST", "reference-imports", "apply-reference-selection", "", false},
		{"PUT", "regulation-references", "select-regulations", "", false},
		{"POST", "tara-imports", "import-tara", "", false},
		{"POST", "damage-scenarios", "record-damage", "", true}, {"PUT", "damage-scenarios/{resource_id}", "record-damage", "id", false},
		{"POST", "security-goals", "record-goal", "", true}, {"PUT", "security-goals/{resource_id}", "record-goal", "id", false},
	} {
		s.route(v.method+" "+apiPrefix+"/workspaces/{workspace_id}/"+v.path, s.mutate(v.action, v.idField, v.createID))
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	if !requestIDPattern.MatchString(id) {
		id = Identity("request")
	}
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id))
	sw := &statusWriter{ResponseWriter: w}
	start := time.Now()
	defer func() {
		if recover() != nil {
			if sw.status == 0 {
				s.writeError(sw, r, fail(500, "服务内部错误", "请求未完成，请稍后重试"))
			}
			s.logger.Printf("request_id=%s panic=true", id)
		}
		status := sw.status
		if status == 0 {
			status = 200
		}
		s.logger.Printf("request_id=%s method=%s path=%q status=%d duration_ms=%d", id, r.Method, r.URL.Path, status, time.Since(start).Milliseconds())
	}()
	sw.Header().Set("X-Request-ID", id)
	sw.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	sw.Header().Set("X-Content-Type-Options", "nosniff")
	sw.Header().Set("Referrer-Policy", "no-referrer")
	sw.Header().Set("Cache-Control", "no-store")
	if e := localRequest(r); e != nil {
		s.writeError(sw, r, e)
		return
	}
	if r.URL.Path == "/service" || strings.HasPrefix(r.URL.Path, "/service/") {
		s.writeError(sw, r, ErrNotFound)
		return
	}
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
		_, pattern := s.mux.Handler(r)
		if pattern == "" {
			allowed := []string{}
			for _, method := range []string{"GET", "HEAD", "POST", "PUT"} {
				copy := r.Clone(r.Context())
				copy.Method = method
				_, p := s.mux.Handler(copy)
				if p != "" {
					allowed = append(allowed, method)
				}
			}
			if len(allowed) > 0 {
				sw.Header().Set("Allow", strings.Join(allowed, ", "))
				s.writeError(sw, r, fail(405, "方法不允许", "该资源不支持此 HTTP 方法"))
			} else {
				s.writeError(sw, r, ErrNotFound)
			}
			return
		}
		s.mux.ServeHTTP(sw, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		sw.Header().Set("Allow", "GET, HEAD")
		s.writeError(sw, r, fail(405, "方法不允许", "静态资源仅支持 GET 和 HEAD"))
		return
	}
	if r.URL.Path != "/" {
		path := filepath.Join(s.root, "next", "web", strings.TrimPrefix(filepath.Clean(r.URL.Path), "/"))
		if _, e := os.Stat(path); e != nil {
			if strings.Contains(filepath.Base(path), ".") {
				s.writeError(sw, r, ErrNotFound)
				return
			}
			copy := r.Clone(r.Context())
			u := *r.URL
			copy.URL = &u
			copy.URL.Path = "/"
			s.static.ServeHTTP(sw, copy)
			return
		}
	}
	s.static.ServeHTTP(sw, r)
}
func localRequest(r *http.Request) error {
	host := r.Host
	if strings.Contains(host, ":") {
		h, _, e := net.SplitHostPort(host)
		if e != nil {
			return fail(400, "主机无效", "仅允许本机回环主机")
		}
		host = h
	}
	if host != "127.0.0.1" && host != "::1" && !strings.EqualFold(host, "localhost") {
		return fail(400, "主机无效", "仅允许本机回环主机")
	}
	if r.RemoteAddr != "" {
		h, _, e := net.SplitHostPort(r.RemoteAddr)
		if e != nil || net.ParseIP(h) == nil || !net.ParseIP(h).IsLoopback() {
			return fail(403, "仅限本机访问", "只允许本机回环连接")
		}
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
		if r.Header.Get("X-Workspace-Client") != "jieheng" {
			return fail(403, "需要应用请求标识", "请从本地工作台提交操作")
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, e := url.Parse(origin)
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			if e != nil || u.Scheme != scheme || !strings.EqualFold(u.Host, r.Host) || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
				return fail(403, "来源不匹配", "只允许当前本地应用发起写入")
			}
		}
	}
	return nil
}
func (s *HTTPServer) writeError(w http.ResponseWriter, r *http.Request, e error) {
	status, title, detail := 500, "服务内部错误", "请求未完成，请稍后重试"
	var h *httpFailure
	var v *ValidationError
	var valueValidation ValidationError
	var c *ConflictError
	var rev *RevisionError
	switch {
	case errors.As(e, &h):
		status, title, detail = h.status, h.title, h.detail
	case errors.Is(e, ErrNotFound) || errors.Is(e, ErrReferenceNotFound):
		status, title, detail = 404, "记录不存在", "请求的工作产品或参考条目不存在"
	case errors.As(e, &rev):
		status, title, detail = 412, "保存版本已过期", rev.Error()
		w.Header().Set("ETag", fmt.Sprintf("\"%d\"", rev.Current))
	case errors.As(e, &c):
		status, title, detail = 409, "重复请求冲突", c.Error()
	case errors.As(e, &v):
		status, title, detail = 422, "工作产品校验未通过", v.Error()
	case errors.As(e, &valueValidation):
		status, title, detail = 422, "工作产品校验未通过", valueValidation.Error()
	case errors.Is(e, context.Canceled):
		status, title, detail = 408, "请求已取消", "请求已取消"
	case errors.Is(e, context.DeadlineExceeded):
		status, title, detail = 503, "服务暂时繁忙", "请求处理超时，请稍后重试"
	}
	data, _ := json.Marshal(Object{"type": fmt.Sprintf("urn:jieheng:problem:%d", status), "title": title, "status": status, "detail": detail, "instance": r.URL.Path, "request_id": r.Context().Value(requestIDKey{})})
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if r.Method != "HEAD" {
		_, _ = w.Write(data)
	}
}
func writeJSON(w http.ResponseWriter, r *http.Request, status int, value any) error {
	b, e := CanonicalJSON(value)
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if r.Method != "HEAD" {
		_, e = w.Write(b)
	}
	return e
}
func setETag(w http.ResponseWriter, ws Object) {
	if rev, e := repoNumber(ws["revision"]); e == nil {
		w.Header().Set("ETag", fmt.Sprintf("\"%d\"", rev))
	}
}
func httpQuery(r *http.Request, allowed ...string) (url.Values, error) {
	q, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil {
		return nil, fail(400, "查询无效", "查询参数格式不正确")
	}
	valid := map[string]bool{}
	for _, k := range allowed {
		valid[k] = true
	}
	for k, v := range q {
		if !valid[k] || len(v) != 1 {
			return nil, fail(400, "查询无效", "未知或重复的查询参数："+k)
		}
	}
	return q, nil
}
func httpPage(q url.Values, defaultLimit int) (int, int, error) {
	offset, limit := 0, defaultLimit
	for _, k := range []string{"offset", "limit"} {
		if v, ok := q[k]; ok {
			n, e := strconv.Atoi(v[0])
			if e != nil || n < 0 || (k == "limit" && (n < 1 || n > 100)) {
				return 0, 0, fail(400, "分页无效", "offset 必须非负，limit 必须为 1–100")
			}
			if k == "offset" {
				offset = n
			} else {
				limit = n
			}
		}
	}
	return offset, limit, nil
}
func queryText(q url.Values) (string, error) {
	v := q.Get("q")
	if utf8.RuneCountInString(v) > 200 {
		return "", fail(400, "检索内容过长", "检索内容最多 200 字")
	}
	return v, nil
}
func readPayload(w http.ResponseWriter, r *http.Request) (Object, error) {
	media, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return nil, fail(415, "内容类型不支持", "写入必须使用 application/json")
	}
	if cs := params["charset"]; cs != "" && !strings.EqualFold(cs, "utf-8") {
		return nil, fail(415, "字符编码不支持", "JSON 必须使用 UTF-8")
	}
	if ce := r.Header.Get("Content-Encoding"); ce != "" && ce != "identity" {
		return nil, fail(415, "内容编码不支持", "不接受压缩请求体")
	}
	if r.ContentLength > maxHTTPBody {
		return nil, fail(413, "内容过大", "单次保存限制为 2 MB")
	}
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, maxHTTPBody))
	if e != nil {
		var m *http.MaxBytesError
		if errors.As(e, &m) {
			return nil, fail(413, "内容过大", "单次保存限制为 2 MB")
		}
		return nil, fail(400, "请求体无效", "无法读取 JSON 请求体")
	}
	if !utf8.Valid(raw) {
		return nil, fail(400, "JSON 无效", "JSON 必须使用有效 UTF-8")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	value, e := readJSONValue(d, 0)
	if e != nil {
		return nil, fail(400, "JSON 无效", e.Error())
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, fail(400, "JSON 无效", "请求体只能包含一个 JSON 对象")
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, fail(400, "JSON 无效", "请求体必须为 JSON 对象")
	}
	if _, err := CanonicalJSON(obj); err != nil {
		return nil, fail(400, "JSON 无效", "JSON 数值或文本超出支持范围")
	}
	return obj, nil
}
func readJSONValue(d *json.Decoder, depth int) (any, error) {
	if depth > 100 {
		return nil, fmt.Errorf("JSON 嵌套层数过多")
	}
	t, e := d.Token()
	if e != nil {
		return nil, fmt.Errorf("JSON 语法不正确")
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	switch delim {
	case '{':
		m := map[string]any{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return nil, e
			}
			k, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("JSON 对象键无效")
			}
			if _, exists := m[k]; exists {
				return nil, fmt.Errorf("JSON 字段重复：%s", k)
			}
			v, e := readJSONValue(d, depth+1)
			if e != nil {
				return nil, e
			}
			m[k] = v
		}
		end, e := d.Token()
		if e != nil || end != json.Delim('}') {
			return nil, fmt.Errorf("JSON 对象未结束")
		}
		return m, nil
	case '[':
		a := []any{}
		for d.More() {
			v, e := readJSONValue(d, depth+1)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		end, e := d.Token()
		if e != nil || end != json.Delim(']') {
			return nil, fmt.Errorf("JSON 数组未结束")
		}
		return a, nil
	}
	return nil, fmt.Errorf("JSON 分隔符无效")
}
func requestKey(r *http.Request) (string, error) {
	v := r.Header.Values("Idempotency-Key")
	if len(v) != 1 {
		return "", fail(400, "缺少请求标识", "写入必须提供一个 Idempotency-Key")
	}
	if e := validateIdempotency(v[0]); e != nil {
		return "", fail(400, "请求标识无效", e.Error())
	}
	return v[0], nil
}
func expectedRevision(r *http.Request) (int64, error) {
	v := r.Header.Values("If-Match")
	if len(v) == 0 {
		return 0, fail(428, "需要版本前提", "保存必须提供 If-Match 修订号")
	}
	if len(v) != 1 {
		return 0, fail(400, "版本格式无效", "If-Match 必须为双引号包围的非负整数")
	}
	m := etagPattern.FindStringSubmatch(v[0])
	if m == nil {
		return 0, fail(400, "版本格式无效", "If-Match 必须为双引号包围的非负整数")
	}
	n, e := strconv.ParseInt(m[1], 10, 64)
	if e != nil {
		return 0, fail(400, "版本格式无效", "修订号超出范围")
	}
	return n, nil
}
