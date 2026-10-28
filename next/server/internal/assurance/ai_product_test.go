package assurance

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func providerFixture(endpoint, protocol string) Object {
	return Object{"name": "Synthetic model service", "protocol": protocol, "base_url": endpoint, "network_scope": "internal", "models": Object{"analysis": "analysis-fixture", "vision": "vision-fixture", "embedding": "embedding-fixture"}, "enabled": true, "retries": 0, "api_key": "synthetic-secret-not-for-export"}
}
func analysisFixture(citation string) Object {
	return Object{
		"summary": "原创合成系统的分析草案", "assumptions": []any{"仅用于集成测试"}, "questions": []any{"需由工程师复核评分"}, "citations": []any{citation},
		"assets":           []any{Object{"id": "A1", "name": "测试命令", "properties": []any{"integrity"}}},
		"damage_scenarios": []any{Object{"id": "DS1", "description": "合成损害", "consequence": "功能偏离预期", "asset_ids": []any{"A1"}, "properties": []any{"integrity"}, "impact": Object{"safety": 1, "financial": 0, "operational": 2, "privacy": 0, "rationale": "合成依据"}}},
		"scenarios":        []any{Object{"id": "TS1", "title": "合成篡改", "asset_id": "A1", "asset_ids": []any{"A1"}, "property": "integrity", "properties": []any{"integrity"}, "damage": "合成损害", "damage_ids": []any{"DS1"}, "threat": "未经授权修改命令", "paths": []any{Object{"name": "测试路径", "steps": "访问接口并提交无效命令", "feasibility": 2, "rationale": "合成依据"}}}},
		"security_goals":   []any{Object{"id": "G1", "statement": "拒绝非授权命令", "scenario_ids": []any{"TS1"}, "cal": 2}},
		"requirements":     []any{Object{"id": "R1", "scenario_id": "TS1", "scenario_ids": []any{"TS1"}, "goal_ids": []any{"G1"}, "statement": "校验命令身份及完整性", "allocation": "软件", "verification": "验证无效凭据和篡改消息被拒绝"}},
		"treatments":       []any{Object{"scenario_id": "TS1", "treatment": "reduce", "rationale": "降低合成风险", "goal_ids": []any{"G1"}, "goal": "拒绝非授权命令", "controls": "鉴权"}},
	}
}
func awaitJob(t *testing.T, a *AIService, wid, id string) Object {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		j, e := a.job(context.Background(), wid, id)
		if e != nil {
			t.Fatal(e)
		}
		if j["status"] != "queued" && j["status"] != "running" {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("task did not finish")
	return nil
}
func TestAIProviderVaultAdaptersAndScope(t *testing.T) {
	repo, path := repositoryTest(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer synthetic-secret-not-for-export" && r.Header.Get("x-api-key") != "synthetic-secret-not-for-export" {
			t.Error("credential not supplied to chosen endpoint")
		}
		switch r.URL.Path {
		case "/models":
			fmt.Fprint(w, `{"data":[{"id":"m1"}]}`)
		case "/chat/completions":
			fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"ok\":true}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":3}}`)
		case "/responses":
			fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"ok\":true}"}]}],"usage":{"input_tokens":4,"output_tokens":3}}`)
		case "/messages":
			fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"ok\":true}"}],"stop_reason":"end_turn","usage":{"input_tokens":4,"output_tokens":3}}`)
		case "/api/chat":
			fmt.Fprint(w, `{"message":{"content":"{\"ok\":true}"},"done":true,"prompt_eval_count":4,"eval_count":3}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	a := NewAIService(repo, nil)
	defer a.Close()
	for _, protocol := range []string{"openai_chat", "openai_responses", "anthropic", "ollama"} {
		p := providerFixture(server.URL, protocol)
		if protocol == "anthropic" {
			delete(dObj(p["models"]), "embedding")
		}
		saved, e := repo.saveProvider(context.Background(), "", -1, p, "test")
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := repoBody(saved)
		if strings.Contains(raw, "secret") || saved["has_key"] != true {
			t.Fatal("credential leaked", raw)
		}
		stored, e := repo.platformGet(context.Background(), "provider", dStr(saved["id"]))
		if e != nil {
			t.Fatal(e)
		}
		raw, _ = repoBody(stored)
		if strings.Contains(raw, "synthetic-secret") {
			t.Fatal("secret stored in plaintext")
		}
		result, attempts, e := a.generate(context.Background(), []string{dStr(saved["id"])}, modelInput{System: "JSON", Text: "test", Purpose: "analysis"}, false)
		if e != nil || result.InputTokens != 4 || result.OutputTokens != 3 || len(attempts) != 1 {
			t.Fatal(protocol, result, attempts, e)
		}
	}
	if info, e := os.Stat(path + ".key"); e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("vault key mode", e)
	}
	p := providerFixture(server.URL, "openai_chat")
	p["network_scope"] = "external"
	if _, e := validateProvider(p); e == nil {
		t.Fatal("external plaintext endpoint accepted")
	}
	p = providerFixture("http://169.254.169.254", "openai_chat")
	if _, e := validateProvider(p); e == nil {
		t.Fatal("metadata address accepted")
	}
	p = providerFixture("http://8.8.8.8", "openai_chat")
	p["timeout_seconds"] = 5
	validated, _ := validateProvider(p)
	start := time.Now()
	if _, e := a.upstream(context.Background(), validated, "models", "GET", nil); e == nil {
		t.Fatal("internal service contacted public IP")
	}
	if time.Since(start) > time.Second {
		t.Fatal("scope was not checked before dialing")
	}
}
func TestAITaskAtomicAdoptionRecoveryAndProvenance(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	s := h.(*HTTPServer)
	defer s.Close()
	wid := httpCreate(t, h)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body Object
		json.NewDecoder(r.Body).Decode(&body)
		messages := dList(body["messages"])
		input, _ := DecodeJSON([]byte(dStr(dObj(messages[len(messages)-1])["content"])))
		citation := dStr(dObj(dList(input["sources"])[0])["id"])
		proposal := analysisFixture(citation)
		raw, _ := repoBody(proposal)
		json.NewEncoder(w).Encode(Object{"choices": []any{Object{"message": Object{"content": raw}, "finish_reason": "stop"}}, "usage": Object{"prompt_tokens": 12, "completion_tokens": 24}})
	}))
	defer upstream.Close()
	provider, e := repo.saveProvider(context.Background(), "", -1, providerFixture(upstream.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	payload := Object{"kind": "analysis", "provider_ids": []any{provider["id"]}, "owner": "合成评审员"}
	body, _ := repoBody(payload)
	created := httpCall(h, "POST", "/api/v1/workspaces/"+wid+"/ai-tasks", body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "task-synthetic-1"})
	if created.Code != 202 {
		t.Fatal(created.Code, created.Body.String())
	}
	jobID := dStr(httpObject(t, created)["id"])
	job := awaitJob(t, s.ai, wid, jobID)
	if job["status"] != "ready" {
		t.Fatal(job["status"], job["error"], job["validation_error"])
	}
	again := httpCall(h, "POST", "/api/v1/workspaces/"+wid+"/ai-tasks", body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "task-synthetic-1"})
	if again.Code != 202 || dStr(httpObject(t, again)["id"]) != jobID || calls.Load() != 1 {
		t.Fatal("task replay rebilled")
	}
	public := httpCall(h, "GET", "/api/v1/workspaces/"+wid+"/ai-tasks/"+jobID, "", nil)
	if strings.Contains(public.Body.String(), "secret_ciphertext") || strings.Contains(public.Body.String(), "workspace_snapshot") || strings.Contains(public.Body.String(), "provider_configs") {
		t.Fatal("private task implementation leaked")
	}
	adoptPath := "/api/v1/workspaces/" + wid + "/ai-tasks/" + jobID + "/adoptions"
	adopt := httpCall(h, "POST", adoptPath, `{"owner":"合成评审员","reviewed":true}`, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "adoption-synthetic-1"})
	if adopt.Code != 200 {
		t.Fatal(adopt.Code, adopt.Body.String())
	}
	result := httpObject(t, adopt)
	ws, e := repo.Get(context.Background(), wid)
	if e != nil {
		t.Fatal(e)
	}
	for _, collection := range []string{"assets", "damage_scenarios", "scenarios", "security_goals", "requirements", "treatment_drafts", "ai_adoptions"} {
		if len(dList(ws[collection])) != 1 {
			t.Fatal(collection, ws[collection])
		}
	}
	if len(dList(ws["decisions"])) != 0 || len(dList(ws["evidence"])) != 0 || dBool(ws["method_confirmed"]) {
		t.Fatal("AI manufactured approval")
	}
	if dInt(ws["revision"]) != 1 {
		t.Fatal("not atomic revision")
	}
	exerciseAIDraftThroughHumanReviewAndReport(t, ws)
	replay := httpCall(h, "POST", adoptPath, `{"owner":"合成评审员","reviewed":true}`, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "adoption-synthetic-1"})
	if replay.Code != 200 {
		t.Fatal("adoption not idempotent", replay.Body.String())
	}
	cps, e := repo.checkpoints(context.Background(), wid)
	if e != nil || len(cps) != 1 {
		t.Fatal("missing rollback point", e)
	}
	restored, _, e := repo.restoreCheckpoint(context.Background(), wid, dStr(result["checkpoint_id"]), "restore-synthetic-1", "验证可还原", "test", 1)
	if e != nil {
		t.Fatal(e)
	}
	if dInt(restored["revision"]) != 2 {
		t.Fatal(restored)
	}
	ws, _ = repo.Get(context.Background(), wid)
	if len(dList(ws["assets"])) != 0 {
		t.Fatal("restoration did not restore")
	}
	cps, _ = repo.checkpoints(context.Background(), wid)
	if len(cps) != 2 {
		t.Fatal("restoration lost current state")
	}
}
func TestKnowledgeRealVectorsScopeAndInvalidation(t *testing.T) {
	repo, _ := repositoryTest(t)
	a := NewAIService(repo, nil)
	defer a.Close()
	ctx := context.Background()
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var p Object
		json.NewDecoder(r.Body).Decode(&p)
		data := []any{}
		for i := range dList(p["input"]) {
			data = append(data, Object{"index": i, "embedding": []any{1.0, 0.5, 0.1}})
		}
		json.NewEncoder(w).Encode(Object{"data": data})
	}))
	defer upstream.Close()
	provider, e := repo.saveProvider(ctx, "", -1, providerFixture(upstream.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	doc := Object{"name": "合成消息鉴权规范", "source": "原创测试规范", "edition": "v1", "content": "命令必须校验身份和完整性，拒绝非法消息。", "workspace_id": "", "tags": []any{}, "enabled": true}
	saved, e := repo.platformPut(ctx, "knowledge", "doc-test", -1, doc, "test")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.indexKnowledge(ctx, "doc-test", dStr(provider["id"]), false); e != nil {
		t.Fatal(e)
	}
	out, e := a.searchKnowledge(ctx, "", "消息鉴权", nil, dStr(provider["id"]), false, 8)
	if e != nil || out["mode"] != "hybrid" || len(httpObjectArray(out["items"])) != 1 || requests.Load() != 2 {
		t.Fatal(out, e, requests.Load())
	}
	doc["edition"], doc["content"] = "v2", "消息鉴权新增重放检测。"
	if _, e = repo.platformPut(ctx, "knowledge", "doc-test", int64(dInt(saved["revision"])), doc, "test"); e != nil {
		t.Fatal(e)
	}
	out, e = a.searchKnowledge(ctx, "", "消息鉴权", nil, dStr(provider["id"]), false, 8)
	if e != nil || out["mode"] != "lexical" || len(dList(out["warnings"])) == 0 {
		t.Fatal("stale vector reused", out, e)
	}
	doc["workspace_id"] = "private-project"
	if _, e = repo.platformPut(ctx, "knowledge", "doc-private", -1, doc, "test"); e != nil {
		t.Fatal(e)
	}
	if _, e = a.searchKnowledge(ctx, "", "消息", []string{"doc-private"}, "", false, 8); e == nil {
		t.Fatal("cross-project document disclosed")
	}
}
func TestAITaskCancellationAndRestartDoesNotRepeat(t *testing.T) {
	repo, _ := repositoryTest(t)
	ctx := context.Background()
	ws, _, e := repo.Create(ctx, "取消测试", "", "cancel-project-1")
	if e != nil {
		t.Fatal(e)
	}
	started := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
	}))
	defer upstream.Close()
	a := NewAIService(repo, nil)
	p, e := repo.saveProvider(ctx, "", -1, providerFixture(upstream.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	job, _, e := a.createJob(ctx, dStr(ws["id"]), "cancel-task-1", 0, Object{"kind": "review", "provider_ids": []any{p["id"]}}, "test")
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	if _, e = a.cancelJob(ctx, dStr(ws["id"]), dStr(job["id"])); e != nil {
		t.Fatal(e)
	}
	a.Close()
	job, _ = a.job(ctx, dStr(ws["id"]), dStr(job["id"]))
	if job["status"] != "cancelled" {
		t.Fatal(job)
	}
	job["status"] = "running"
	raw, _ := repoBody(job)
	repo.db.Exec("UPDATE ai_jobs SET body=? WHERE id=?", raw, job["id"])
	a2 := NewAIService(repo, nil)
	defer a2.Close()
	job, _ = a2.job(ctx, dStr(ws["id"]), dStr(job["id"]))
	if job["status"] != "interrupted" {
		t.Fatal("restart did not interrupt", job)
	}
}
func TestAIProposalRejectsUnsupportedWrites(t *testing.T) {
	ws := NewWorkspace("test", "")
	p := analysisFixture("project:test")
	p["evidence"] = []any{Object{"result": "pass"}}
	if _, _, e := validateAIProposal(ws, p, "analysis", "test", nil); e == nil {
		t.Fatal("fabricated evidence accepted")
	}
	delete(p, "evidence")
	dObj(dList(p["treatments"])[0])["customer_confirmation"] = "签署"
	if _, _, e := validateAIProposal(ws, p, "analysis", "test", nil); e == nil {
		t.Fatal("fabricated signature accepted")
	}
	delete(dObj(dList(p["treatments"])[0]), "customer_confirmation")
	if _, _, e := validateAIProposal(ws, p, "review", "test", nil); e == nil {
		t.Fatal("review changed work products")
	}
}

func TestAIExplicitFallbackAndCredentialsFailure(t *testing.T) {
	repo, _ := repositoryTest(t)
	a := NewAIService(repo, nil)
	defer a.Close()
	ctx := context.Background()
	var primaryStatus atomic.Int32
	primaryStatus.Store(503)
	var fallbackCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(primaryStatus.Load()))
		w.Write([]byte(`{"error":"secret-value-must-not-leak"}`))
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{}"},"finish_reason":"stop"}]}`))
	}))
	defer second.Close()
	p1, e := repo.saveProvider(ctx, "", -1, providerFixture(first.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	p2, e := repo.saveProvider(ctx, "", -1, providerFixture(second.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	in := modelInput{System: "test", Text: "synthetic", Purpose: "analysis"}
	got, attempts, e := a.generate(ctx, []string{dStr(p1["id"]), dStr(p2["id"])}, in, false)
	if e != nil || got.ProviderID != p2["id"] || len(attempts) != 2 || fallbackCalls.Load() != 1 {
		t.Fatal(e, attempts)
	}
	primaryStatus.Store(401)
	_, attempts, e = a.generate(ctx, []string{dStr(p1["id"]), dStr(p2["id"])}, in, false)
	if e == nil || len(attempts) != 1 || fallbackCalls.Load() != 1 || strings.Contains(e.Error(), "secret-value") {
		t.Fatal("auth error retried or leaked", e, attempts)
	}
}

func TestAIImageCreatesNativeEditableDiagramAndAssets(t *testing.T) {
	repo, _ := repositoryTest(t)
	ctx := context.Background()
	ws, _, e := repo.Create(ctx, "Synthetic image workflow", "", "image-project-1")
	if e != nil {
		t.Fatal(e)
	}
	graph := Object{"view_mode": "entities", "nodes": []any{Object{"id": "device", "name": "Synthetic device", "kind": "entity", "entityType": "device", "x": 50, "y": 50, "width": 220, "height": 130, "operating_system": Object{"family": "rtos", "name": "SyntheticOS"}, "children": []any{Object{"id": "sw", "name": "Controller", "kind": "software"}}}, Object{"id": "cloud", "name": "Synthetic service", "kind": "entity", "entityType": "cloud", "x": 550, "y": 50, "width": 220, "height": 130}}, "flows": []any{Object{"id": "telemetry", "name": "Telemetry", "source": "device", "target": "cloud", "protocol": "Wi-Fi", "communication": Object{"medium": "wireless", "technology": "wifi", "security_properties": []any{"integrity"}}}}}
	proposal := Object{"summary": "Synthetic native graph", "questions": []any{}, "assumptions": []any{}, "citations": []any{fmt.Sprintf("project:%s@0", ws["id"])}, "architecture": graph}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body Object
		json.NewDecoder(r.Body).Decode(&body)
		content := dList(dObj(dList(body["messages"])[1])["content"])
		if len(content) != 2 || dObj(content[1])["type"] != "image_url" {
			t.Error("image not transmitted in native protocol")
		}
		raw, _ := repoBody(proposal)
		json.NewEncoder(w).Encode(Object{"choices": []any{Object{"message": Object{"content": raw}, "finish_reason": "stop"}}})
	}))
	defer upstream.Close()
	p, e := repo.saveProvider(ctx, "", -1, providerFixture(upstream.URL, "openai_chat"), "test")
	if e != nil {
		t.Fatal(e)
	}
	a := NewAIService(repo, nil)
	defer a.Close()
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	task, _, e := a.createJob(ctx, dStr(ws["id"]), "image-task-1", 0, Object{"kind": "diagram", "provider_ids": []any{p["id"]}, "image_base64": base64.StdEncoding.EncodeToString(b.Bytes()), "image_mime": "image/png"}, "test")
	if e != nil {
		t.Fatal(e)
	}
	job := awaitJob(t, a, dStr(ws["id"]), dStr(task["id"]))
	if job["status"] != "ready" {
		t.Fatal(job["error"], job["validation_error"])
	}
	if _, _, e = a.adoptJob(ctx, dStr(ws["id"]), dStr(job["id"]), "adopt-image-1", "Synthetic reviewer", "test", 0); e != nil {
		t.Fatal(e)
	}
	current, _ := repo.Get(ctx, dStr(ws["id"]))
	if len(dList(dObj(current["architecture"])["nodes"])) != 2 || len(dList(current["assets"])) < 2 {
		t.Fatal("native DFD or assets absent")
	}
	updated := Clone(dObj(current["architecture"]))
	dObj(dList(updated["nodes"])[0])["x"] = 120
	if _, _, e = Apply(current, "replace-architecture", updated); e != nil {
		t.Fatal("native diagram not editable", e)
	}
	tables := Tables(current, nil, nil)
	found := false
	for _, table := range tables {
		if table["name"] == "辅助分析追溯" {
			found = true
		}
	}
	if !found {
		t.Fatal("AI provenance missing from report")
	}
}

func TestAIUnknownScoresStayPendingAndCannotBeApproved(t *testing.T) {
	ws := NewWorkspace("Synthetic pending scores", "")
	proposal := analysisFixture("synthetic")
	dObj(dObj(dList(proposal["damage_scenarios"])[0])["impact"])["safety"] = nil
	next, _, e := validateAIProposal(ws, proposal, "analysis", "Synthetic owner", nil)
	if e != nil {
		t.Fatal("unknown score must remain a usable draft", e)
	}
	scenario := dObj(dList(next["scenarios"])[0])
	if Risk(scenario, dObj(next["method"]))["level"] != nil {
		t.Fatal("missing score silently became assessed")
	}
	next["method_confirmed"] = true
	draft := Clone(dObj(dList(next["treatment_drafts"])[0]))
	delete(draft, "assessment_hash")
	delete(draft, "created_at")
	delete(draft, "draft")
	if _, _, e = Apply(next, "decide-treatment", draft); e == nil {
		t.Fatal("pending assessment was approved")
	}
}

func exerciseAIDraftThroughHumanReviewAndReport(t *testing.T, initial Object) {
	t.Helper()
	ws := Clone(initial)
	apply := func(action string, payload Object) {
		next, _, err := Apply(ws, action, payload)
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		ws = next
	}
	apply("define-item", Object{"purpose": "Synthetic command processing", "boundary": "Synthetic device and service", "environment": "Controlled test network", "assumptions": "Synthetic test only", "owner": "Synthetic engineer"})
	apply("replace-architecture", Object{"asset_generation": "catalog", "nodes": []any{Object{"id": "device", "name": "Synthetic device", "kind": "entity", "x": 0, "y": 0}}, "flows": []any{}})
	asset := dObj(dList(ws["assets"])[0])
	apply("record-asset", Object{"id": asset["id"], "name": asset["name"], "node_ids": []any{"device"}, "properties": asset["properties"]})
	apply("adopt-method", Clone(dObj(ws["method"])))
	decision := Clone(dObj(dList(ws["treatment_drafts"])[0]))
	for _, key := range []string{"draft", "created_at", "assessment_hash"} {
		delete(decision, key)
	}
	decision["residual_impact"], decision["residual_feasibility"], decision["residual_rationale"] = 1, 1, "Human reviewed synthetic control assessment"
	apply("decide-treatment", decision)
	if len(dList(ws["treatment_drafts"])) != 0 {
		t.Fatal("confirmed suggestion remained pending")
	}
	requirement := dObj(dList(ws["requirements"])[0])
	apply("record-evidence", Object{"requirement_id": requirement["id"], "result": "pass", "reference": "synthetic-test-record", "summary": "Synthetic validation performed by test fixture", "reviewer": "Synthetic reviewer"})
	decision["residual_status"] = "validated"
	apply("decide-treatment", decision)
	if !dBool(Review(ws)["ready"]) {
		t.Fatal("human workflow cannot close", Review(ws)["issues"])
	}
	apply("seal-baseline", Object{"name": "Synthetic reviewed baseline", "reviewer": "Synthetic reviewer", "note": "Test only"})
	file, err := Excel(ws, nil, nil)
	if err != nil || len(file) < 1000 {
		t.Fatal("report export", err)
	}
	html, err := Printable(ws, nil, nil)
	if err != nil || !strings.Contains(string(html), "辅助分析追溯") {
		t.Fatal("missing report provenance", err)
	}
}
