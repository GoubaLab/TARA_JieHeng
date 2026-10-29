package assurance

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

func (s *HTTPServer) health(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	mode := "local"
	if s.auth != nil {
		mode = "enterprise"
	}
	return writeJSON(w, r, 200, Object{"application": "jieheng", "mode": mode, "version": "2.2.2", "backend": "go", "outbound_connections": true, "outbound_policy": "configured_models_explicit_requests", "excel_export": ExcelAvailable()})
}
func (s *HTTPServer) document(name, media string) endpoint {
	return func(w http.ResponseWriter, r *http.Request) error {
		if _, e := httpQuery(r); e != nil {
			return e
		}
		b, e := os.ReadFile(filepath.Join(s.root, "next", "server", name))
		if e != nil {
			return ErrNotFound
		}
		w.Header().Set("Content-Type", media)
		if r.Method != "HEAD" {
			_, e = w.Write(b)
		}
		return e
	}
}
func (s *HTTPServer) listWorkspaces(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "offset", "limit")
	if e != nil {
		return e
	}
	o, l, e := httpPage(q, 100)
	if e != nil {
		return e
	}
	var items []Object
	var total int
	if s.auth != nil && principal(r)["role"] != "admin" {
		items, total, e = s.repository.listForUser(r.Context(), actorID(r), o, l)
	} else {
		items, total, e = s.repository.List(r.Context(), o, l)
	}
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"items": items, "total": total, "offset": o, "limit": l})
}
func (s *HTTPServer) createWorkspace(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	if s.auth != nil {
		key = "create_" + Digest(Object{"actor": actorID(r), "key": key})
	}
	if len(r.Header.Values("If-Match")) > 0 {
		return fail(400, "创建不接受版本前提", "创建项目不使用 If-Match")
	}
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	for k := range p {
		if k != "title" && k != "description" {
			return &ValidationError{Detail: "未知字段：" + k}
		}
	}
	title, ok := p["title"].(string)
	if !ok {
		return &ValidationError{Detail: "title 必须为字符串"}
	}
	description := ""
	if v, exists := p["description"]; exists {
		description, ok = v.(string)
		if !ok {
			return &ValidationError{Detail: "description 必须为字符串"}
		}
	}
	ws, _, e := s.repository.Create(r.Context(), title, description, key)
	if e != nil {
		return e
	}
	setETag(w, ws)
	w.Header().Set("Location", apiPrefix+"/workspaces/"+fmt.Sprint(ws["id"]))
	return writeJSON(w, r, 201, ws)
}
func (s *HTTPServer) workspace(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	role, e := s.projectRole(r, r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	w.Header().Set("X-Workspace-Role", role)
	setETag(w, ws)
	if source := dObj(ws["tara_import"]); source != nil {
		delete(source, "original_base64")
	}
	return writeJSON(w, r, 200, ws)
}
func (s *HTTPServer) review(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, Review(ws))
}
func (s *HTTPServer) singleton(field string) endpoint {
	return func(w http.ResponseWriter, r *http.Request) error {
		if _, e := httpQuery(r); e != nil {
			return e
		}
		ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
		if e != nil {
			return e
		}
		setETag(w, ws)
		value := ws[field]
		if field == "numbering" {
			value = EffectiveNumbering(dObj(value))
		}
		if field == "regulation_refs" {
			if value == nil {
				value = []any{}
			}
			value = Object{"references": value}
		}
		return writeJSON(w, r, 200, value)
	}
}
func httpObjectArray(value any) []Object {
	out := []Object{}
	switch rows := value.(type) {
	case []Object:
		return rows
	case []any:
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}
func (s *HTTPServer) resource(collection, idField string) endpoint {
	return func(w http.ResponseWriter, r *http.Request) error {
		if _, e := httpQuery(r); e != nil {
			return e
		}
		ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
		if e != nil {
			return e
		}
		for _, row := range httpObjectArray(ws[collection]) {
			if row[idField] == r.PathValue("resource_id") {
				setETag(w, ws)
				return writeJSON(w, r, 200, row)
			}
		}
		return ErrNotFound
	}
}
func (s *HTTPServer) referenceImports(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "offset", "limit")
	if e != nil {
		return e
	}
	o, l, e := httpPage(q, 100)
	if e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	items := []Object{}
	if records, ok := ws["reference_imports"].(map[string]any); ok {
		keys := []string{}
		for key := range records {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if row, ok := records[key].(map[string]any); ok {
				copy := Clone(row)
				copy["key"] = key
				items = append(items, copy)
			}
		}
	}
	total := len(items)
	start := min(o, total)
	end := min(start+l, total)
	setETag(w, ws)
	return writeJSON(w, r, 200, Object{"items": items[start:end], "total": total, "offset": o, "limit": l})
}
func (s *HTTPServer) mutate(action, idField string, createID bool) endpoint {
	return func(w http.ResponseWriter, r *http.Request) error {
		if _, e := httpQuery(r); e != nil {
			return e
		}
		revision, e := expectedRevision(r)
		if e != nil {
			return e
		}
		key, e := requestKey(r)
		if e != nil {
			return e
		}
		payload, e := readPayload(w, r)
		if e != nil {
			return e
		}
		if user := principal(r); user != nil && (action == "record-evidence" || action == "seal-baseline") {
			payload["reviewer"] = user["name"]
		}
		if createID {
			if _, exists := payload["id"]; exists {
				return &ValidationError{Detail: "创建资源不能指定 id"}
			}
		}
		if idField != "" {
			id := r.PathValue("resource_id")
			if value, exists := payload[idField]; exists {
				if idField == "scenario_id" {
					return &ValidationError{Detail: "scenario_id 由路径指定，不能在请求体中提供"}
				}
				if value != id {
					return &ValidationError{Detail: "请求体 id 与路径不一致"}
				}
			}
			payload[idField] = id
		}
		prepare := func(a string, p, current Object) (Object, error) {
			if a == "apply-reference-selection" {
				resolved, err := ResolveSelection(s.catalog, p)
				if err != nil {
					return nil, &ValidationError{Detail: err.Error()}
				}
				return resolved, nil
			}
			if e := s.catalog.ValidateReferencesForWorkspace(a, p, current); e != nil {
				return nil, &ValidationError{Detail: e.Error()}
			}
			return p, nil
		}
		result, _, e := s.repository.Change(r.Context(), r.PathValue("workspace_id"), revision, key, action, payload, prepare)
		if e != nil {
			return e
		}
		setETag(w, result)
		status := 200
		if r.Method == "POST" {
			status = 201
			base := apiPrefix + "/workspaces/" + r.PathValue("workspace_id")
			collection := map[string]string{"record-asset": "assets", "record-scenario": "scenarios", "specify-requirement": "requirements", "record-evidence": "evidence", "seal-baseline": "baselines", "apply-reference-selection": "reference-imports"}[action]
			if action == "import-tara" {
				collection = "tara-imports"
			}
			if action == "record-damage" {
				collection = "damage-scenarios"
			}
			if action == "record-goal" {
				collection = "security-goals"
			}
			location := base + "/" + collection
			if action != "apply-reference-selection" {
				location += "/" + fmt.Sprint(result["entity_id"])
			}
			w.Header().Set("Location", location)
		}
		return writeJSON(w, r, status, result)
	}
}
func (s *HTTPServer) recommendations(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "asset_id", "reference")
	if e != nil {
		return e
	}
	asset, ref := q.Get("asset_id"), q.Get("reference")
	if asset == "" || len(asset) > 100 || !regexp.MustCompile(`^systems:[0-9]+$`).MatchString(ref) {
		return fail(400, "查询无效", "需要有效的 asset_id 和 systems:编号 reference")
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	out, e := Recommendations(ws, s.catalog, asset, ref)
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) events(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "offset", "limit")
	if e != nil {
		return e
	}
	o, l, e := httpPage(q, 100)
	if e != nil {
		return e
	}
	items, total, e := s.repository.Events(r.Context(), r.PathValue("workspace_id"), o, l)
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"entries": items, "total": total, "offset": o, "limit": l})
}
func (s *HTTPServer) baseline(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Baseline(r.Context(), r.PathValue("workspace_id"), r.PathValue("baseline_id"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, ws)
}
func (s *HTTPServer) report(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "format", "baseline_id")
	if e != nil {
		return e
	}
	format := q.Get("format")
	if format == "" {
		if _, ok := q["format"]; ok {
			return fail(400, "格式无效", "format 必须为 json、html 或 xlsx")
		}
		format = "json"
	}
	if format != "json" && format != "html" && format != "xlsx" {
		return fail(400, "格式无效", "format 必须为 json、html 或 xlsx")
	}
	key := r.PathValue("workspace_id")
	var ws Object
	if v, ok := q["baseline_id"]; ok {
		if v[0] == "" || len(v[0]) > 100 {
			return fail(400, "基线标识无效", "需要有效的 baseline_id")
		}
		ws, e = s.repository.Baseline(r.Context(), key, v[0])
	} else {
		ws, e = s.repository.Get(r.Context(), key)
	}
	if e != nil {
		return e
	}
	all, e := s.repository.AllEvents(r.Context(), key)
	if e != nil {
		return e
	}
	rev, e := repoNumber(ws["revision"])
	if e != nil {
		return e
	}
	events := []Object{}
	for _, event := range all {
		n, e := repoNumber(event["revision"])
		if e != nil {
			return e
		}
		if n <= rev {
			events = append(events, event)
		}
	}
	var data []byte
	media, name := "application/json; charset=utf-8", "assurance-work-products.json"
	switch format {
	case "json":
		data, e = CanonicalJSON(Object{"schema": "jieheng.work-products/1", "workspace": ws, "review": Review(ws)})
	case "html":
		media, name = "text/html; charset=utf-8", "assurance-report.html"
		data, e = Printable(ws, events, s.catalog)
	case "xlsx":
		if !ExcelAvailable() {
			return fail(503, "Excel 导出不可用", "本机缺少 Excel 渲染组件，可使用 HTML 或 JSON")
		}
		media, name = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "assurance-report.xlsx"
		data, e = Excel(ws, events, s.catalog)
	}
	if e != nil {
		return e
	}
	setETag(w, ws)
	w.Header().Set("Content-Type", media)
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if r.Method != "HEAD" {
		_, e = w.Write(data)
	}
	return e
}
func (s *HTTPServer) catalogSummary(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	return writeJSON(w, r, 200, s.catalog.Summary())
}
func (s *HTTPServer) catalogEntries(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "q", "offset", "limit")
	if e != nil {
		return e
	}
	o, l, e := httpPage(q, 24)
	if e != nil {
		return e
	}
	text, e := queryText(q)
	if e != nil {
		return e
	}
	out, e := s.catalog.Search(r.PathValue("category"), text, o, l)
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) reference(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	out, e := s.catalog.Get(r.PathValue("key"))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) regulations(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	return writeJSON(w, r, 200, s.catalog.Regulations.Summary())
}
func (s *HTTPServer) clauses(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "q", "offset", "limit")
	if e != nil {
		return e
	}
	o, l, e := httpPage(q, 20)
	if e != nil {
		return e
	}
	text, e := queryText(q)
	if e != nil {
		return e
	}
	out, e := s.catalog.Regulations.Clauses(r.PathValue("document_id"), text, o, l)
	if e != nil {
		return e
	}
	// Clauses returns an isolated copy. Enrich only this HTTP representation;
	// the versioned source records and their golden comparison stay unchanged.
	for _, row := range libRows(out["items"]) {
		row["document_title"] = libObj(out["document"])["code"]
		if libInt(row["page"]) == 0 {
			pages := libList(row["source_pages"])
			if len(pages) > 0 {
				first, last := libInt(pages[0]), libInt(pages[0])
				for _, value := range pages[1:] {
					page := libInt(value)
					first, last = min(first, page), max(last, page)
				}
				row["page"], row["page_end"] = first, last
			}
		}
		if parentKey := libStr(row["parent"]); parentKey != "" {
			parent, err := s.catalog.Regulations.Get(parentKey)
			if err != nil {
				return err
			}
			row["parent_label"] = parent["number"]
		}
	}
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) source(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	index, e := strconv.Atoi(r.PathValue("index"))
	if e != nil || index < 0 {
		return fail(400, "来源索引无效", "来源文件索引必须为非负整数")
	}
	path, e := s.catalog.Regulations.Source(r.PathValue("document_id"), index)
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filepath.Base(path)}))
	http.ServeFile(w, r, path)
	return nil
}
