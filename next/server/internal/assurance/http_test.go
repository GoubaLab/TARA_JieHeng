package assurance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var httpCatalogOnce sync.Once
var httpCatalog *Catalog
var httpCatalogErr error

func httpTestServer(t *testing.T) (http.Handler, *Repository, *bytes.Buffer) {
	t.Helper()
	root, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	httpCatalogOnce.Do(func() {
		httpCatalog, httpCatalogErr = NewCatalog(filepath.Join(root, "data", "source-snapshot"), filepath.Join(root, "next", "data", "regulations"))
	})
	if httpCatalogErr != nil {
		t.Fatal(httpCatalogErr)
	}
	repo, _ := repositoryTest(t)
	logs := &bytes.Buffer{}
	return NewHTTPHandler(repo, httpCatalog, root, log.New(logs, "", 0)), repo, logs
}
func httpCall(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:8790"+path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:12345"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Workspace-Client", "jieheng")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func httpObject(t *testing.T, w *httptest.ResponseRecorder) Object {
	t.Helper()
	o, e := DecodeJSON(w.Body.Bytes())
	if e != nil {
		t.Fatalf("invalid response %d: %s", w.Code, w.Body.String())
	}
	return o
}
func httpCreate(t *testing.T, h http.Handler) string {
	t.Helper()
	w := httpCall(h, "POST", "/api/v1/workspaces", `{"title":"API验证"}`, map[string]string{"Idempotency-Key": "create-http-0001"})
	if w.Code != 201 {
		t.Fatalf("create %d: %s", w.Code, w.Body.String())
	}
	return fmt.Sprint(httpObject(t, w)["id"])
}
func TestHTTPCreateReplayLocationAndErrorContract(t *testing.T) {
	h, _, _ := httpTestServer(t)
	key := httpCreate(t, h)
	again := httpCall(h, "POST", "/api/v1/workspaces", `{"title":"API验证"}`, map[string]string{"Idempotency-Key": "create-http-0001"})
	if again.Code != 201 || httpObject(t, again)["id"] != key || again.Header().Get("ETag") != `"0"` {
		t.Fatal("create replay contract")
	}
	get := httpCall(h, "GET", again.Header().Get("Location"), "", nil)
	if get.Code != 200 {
		t.Fatal("Location cannot be followed")
	}
	conflict := httpCall(h, "POST", "/api/v1/workspaces", `{"title":"changed"}`, map[string]string{"Idempotency-Key": "create-http-0001"})
	if conflict.Code != 409 {
		t.Fatal(conflict.Code)
	}
	p := httpObject(t, conflict)
	for _, field := range []string{"type", "title", "status", "detail", "instance", "request_id"} {
		if p[field] == nil {
			t.Fatal("missing problem field", field)
		}
	}
	if conflict.Header().Get("Content-Type") != "application/problem+json" || p["request_id"] != conflict.Header().Get("X-Request-ID") {
		t.Fatal("problem headers")
	}
}
func TestHTTPPreconditionsReplayAndAtomicValidation(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	key := httpCreate(t, h)
	path := "/api/v1/workspaces/" + key + "/item"
	tests := []struct {
		name    string
		headers map[string]string
		body    string
		status  int
	}{{"missing", nil, `{}`, 428}, {"weak", map[string]string{"If-Match": `W/"0"`}, `{}`, 400}, {"missing-key", map[string]string{"If-Match": `"0"`}, `{}`, 400}, {"unknown", map[string]string{"If-Match": `"0"`, "Idempotency-Key": "unknown-field-001"}, `{"unknown":true}`, 422}, {"invalid-type", map[string]string{"If-Match": `"0"`, "Idempotency-Key": "invalid-type-001"}, `{"purpose":true}`, 422}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httpCall(h, "PUT", path, test.body, test.headers)
			if w.Code != test.status {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
	head := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "item-write-0001"}
	first := httpCall(h, "PUT", path, `{"purpose":"a"}`, head)
	if first.Code != 200 || first.Header().Get("ETag") != `"1"` {
		t.Fatalf("write %d %s", first.Code, first.Body.String())
	}
	again := httpCall(h, "PUT", path, `{"purpose":"a"}`, head)
	if again.Code != 200 || again.Body.String() != first.Body.String() {
		t.Fatal("replay failed")
	}
	head["Idempotency-Key"] = "item-write-0002"
	stale := httpCall(h, "PUT", path, `{"purpose":"b"}`, head)
	if stale.Code != 412 || stale.Header().Get("ETag") != `"1"` {
		t.Fatal("stale precondition", stale.Code)
	}
	_, events, e := repo.Events(context.Background(), key, 0, 100)
	if e != nil || events != 2 {
		t.Fatal("invalid request mutated journal")
	}
}
func TestHTTPStreamingLimitJSONAndContentType(t *testing.T) {
	h, _, _ := httpTestServer(t)
	key := httpCreate(t, h)
	path := "/api/v1/workspaces/" + key + "/item"
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "body-check-0001"}
	for _, body := range []string{`{"purpose":"x","purpose":"y"}`, `{} {}`, `[]`, `{"purpose":NaN}`, `{"purpose":1e999}`, `{"purpose":`} {
		w := httpCall(h, "PUT", path, body, headers)
		if w.Code != 400 {
			t.Fatalf("invalid JSON got %d %s", w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("PUT", "http://localhost"+path, strings.NewReader(`{"purpose":"`+strings.Repeat("x", int(maxHTTPBody))+`"}`))
	r.RemoteAddr = "127.0.0.1:1234"
	r.ContentLength = -1
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Workspace-Client", "jieheng")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatal("chunked body limit", w.Code)
	}
	headers["Content-Type"] = "text/plain"
	w = httpCall(h, "PUT", path, `{}`, headers)
	if w.Code != 415 {
		t.Fatal("content-type", w.Code)
	}
}
func TestHTTPQueriesMethodsHeadAndLocalBoundary(t *testing.T) {
	h, _, _ := httpTestServer(t)
	key := httpCreate(t, h)
	for _, path := range []string{"/api/v1/workspaces?limit=101", "/api/v1/workspaces?offset=-1", "/api/v1/workspaces?limit=1&limit=2", "/api/v1/health?unknown=true", "/api/v1/catalog/systems/entries?q=" + strings.Repeat("字", 201)} {
		w := httpCall(h, "GET", path, "", nil)
		if w.Code != 400 {
			t.Fatal(path, w.Code)
		}
	}
	w := httpCall(h, "DELETE", "/api/v1/workspaces/"+key, "", nil)
	if w.Code != 405 || w.Header().Get("Allow") != "GET, HEAD" {
		t.Fatal("405 contract", w.Code, w.Header())
	}
	w = httpCall(h, "HEAD", "/api/v1/workspaces/"+key, "", nil)
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("ETag") != `"0"` {
		t.Fatal("HEAD contract")
	}
	for _, path := range []string{"/service/health", "/api/v1/unknown"} {
		if w := httpCall(h, "GET", path, "", nil); w.Code != 404 {
			t.Fatal("legacy/unknown route", w.Code)
		}
	}
	r := httptest.NewRequest("GET", "http://evil.example/api/v1/health", nil)
	r.RemoteAddr = "127.0.0.1:1"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("host boundary")
	}
	r = httptest.NewRequest("GET", "http://localhost/api/v1/health", nil)
	r.RemoteAddr = "192.0.2.1:1"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("loopback boundary")
	}
	w = httpCall(h, "POST", "/api/v1/workspaces", `{"title":"x"}`, map[string]string{"Origin": "http://evil.example", "Idempotency-Key": "foreign-origin-1"})
	if w.Code != 403 {
		t.Fatal("origin boundary")
	}
}
func TestHTTPResourceCreationLocationAndPathID(t *testing.T) {
	h, _, _ := httpTestServer(t)
	key := httpCreate(t, h)
	path := "/api/v1/workspaces/" + key + "/assets"
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "asset-create-001"}
	body := `{"name":"资产","properties":["integrity"]}`
	w := httpCall(h, "POST", path, body, headers)
	if w.Code != 201 {
		t.Fatalf("asset %d %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	get := httpCall(h, "GET", location, "", nil)
	if get.Code != 200 || httpObject(t, get)["name"] != "资产" {
		t.Fatal("asset Location")
	}
	headers["If-Match"] = `"1"`
	headers["Idempotency-Key"] = "asset-id-reject1"
	w = httpCall(h, "POST", path, `{"id":null,"name":"资产","properties":["integrity"]}`, headers)
	if w.Code != 422 {
		t.Fatal("POST accepted id", w.Code)
	}
	w = httpCall(h, "PUT", location, `{"id":"other","name":"资产","properties":["integrity"]}`, headers)
	if w.Code != 422 {
		t.Fatal("PUT mismatched id", w.Code)
	}
	w = httpCall(h, "PUT", path+"/does-not-exist", body, headers)
	if w.Code != 404 {
		t.Fatalf("PUT missing target %d: %s", w.Code, w.Body.String())
	}
	w = httpCall(h, "GET", "/api/v1/workspaces/"+key+"/reference-imports?offset=100&limit=10", "", nil)
	if w.Code != 200 || httpObject(t, w)["total"] != json.Number("0") {
		t.Fatal("reference import paging")
	}
}
func TestHTTPRequestLoggingAndPanicProblem(t *testing.T) {
	h, _, logs := httpTestServer(t)
	key := httpCreate(t, h)
	secret := "PRIVATE-PAYLOAD-TEST"
	w := httpCall(h, "PUT", "/api/v1/workspaces/"+key+"/item", `{"purpose":"`+secret+`"}`, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "log-body-0001", "X-Request-ID": "chosen-request"})
	if w.Code != 200 || w.Header().Get("X-Request-ID") != "chosen-request" {
		t.Fatal("request id")
	}
	if strings.Contains(logs.String(), secret) || !strings.Contains(logs.String(), "chosen-request") {
		t.Fatal("log data minimization")
	}
	s := h.(*HTTPServer)
	s.route("GET /api/v1/test-panic", func(http.ResponseWriter, *http.Request) error { panic("PRIVATE-PANIC") })
	w = httpCall(h, "GET", "/api/v1/test-panic", "", nil)
	if w.Code != 500 || httpObject(t, w)["request_id"] == nil || strings.Contains(logs.String(), "PRIVATE-PANIC") {
		t.Fatal("panic boundary")
	}
}

func TestHTTPRegulationParentAndPhysicalPageDecorationIsIsolated(t *testing.T) {
	h, _, _ := httpTestServer(t)
	s := h.(*HTTPServer)
	// Source-only distributions contain no regulation text. Keep this behavior test
	// independent by supplying two small, original in-memory documents.
	documents := []Object{
		{"id": "synthetic-outline", "code": "DEMO-OUTLINE", "title": "原创章节示例", "edition": "test", "coverage": "partial", "clause_count": 2},
		{"id": "synthetic-pages", "code": "DEMO-PAGES", "title": "原创页码示例", "edition": "test", "coverage": "partial", "clause_count": 1},
	}
	rows := []Object{
		{"key": "reg:synthetic-outline:1", "document_id": "synthetic-outline", "number": "1", "title": "示例章节", "body": "这是原创测试章节。", "description": "这是原创测试章节。", "parent": nil},
		{"key": "reg:synthetic-outline:1.1", "document_id": "synthetic-outline", "number": "1.1", "title": "示例子项", "body": "此内容仅验证父级标签，不属于任何法规。", "description": "此内容仅验证父级标签，不属于任何法规。", "parent": "reg:synthetic-outline:1"},
		{"key": "reg:synthetic-pages:sample", "document_id": "synthetic-pages", "number": "sample", "title": "跨页示例", "body": "此内容仅验证页面显示，不属于任何法规。", "description": "此内容仅验证页面显示，不属于任何法规。", "parent": nil, "source_pages": []any{2, 3}},
	}
	regulations := &Regulations{documents: documents, byDocument: map[string]Object{}, rows: map[string][]Object{}, byKey: map[string]Object{}}
	for _, document := range documents {
		regulations.byDocument[libStr(document["id"])] = document
	}
	for _, row := range rows {
		id := libStr(row["document_id"])
		regulations.rows[id] = append(regulations.rows[id], row)
		regulations.byKey[libStr(row["key"])] = row
	}
	s.catalog = &Catalog{Regulations: regulations}
	for _, test := range []struct {
		doc, locator, code, parent string
		page, pageEnd              int
	}{
		{"synthetic-outline", "1.1", "DEMO-OUTLINE", "1", 0, 0},
		{"synthetic-pages", "sample", "DEMO-PAGES", "", 2, 3},
	} {
		t.Run(test.doc, func(t *testing.T) {
			key := "reg:" + test.doc + ":" + test.locator
			before, err := s.catalog.Regulations.Get(key)
			if err != nil {
				t.Fatal(err)
			}
			beforeDigest := Digest(before)
			if test.page > 0 && before["page"] != nil {
				t.Fatal("fixture must require source_pages fallback")
			}
			w := httpCall(h, "GET", "/api/v1/regulations/"+test.doc+"/clauses?q="+key, "", nil)
			if w.Code != 200 {
				t.Fatalf("query %d: %s", w.Code, w.Body.String())
			}
			var found Object
			for _, row := range libRows(httpObject(t, w)["items"]) {
				if row["key"] == key {
					found = row
					break
				}
			}
			if found == nil || found["document_title"] != test.code {
				t.Fatal("missing document label")
			}
			if test.parent != "" && found["parent_label"] != test.parent {
				t.Fatalf("parent label: %#v", found["parent_label"])
			}
			if test.page > 0 && (libInt(found["page"]) != test.page || libInt(found["page_end"]) != test.pageEnd) {
				t.Fatalf("physical pages: %#v / %#v", found["page"], found["page_end"])
			}
			after, err := s.catalog.Regulations.Get(key)
			if err != nil || Digest(after) != beforeDigest {
				t.Fatal("HTTP decoration changed library record")
			}
			raw, err := s.catalog.Regulations.Clauses(test.doc, key, 0, 100)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range libRows(raw["items"]) {
				if row["key"] == key && Digest(row) != beforeDigest {
					t.Fatal("HTTP decoration changed raw Clauses result")
				}
			}
		})
	}
}
