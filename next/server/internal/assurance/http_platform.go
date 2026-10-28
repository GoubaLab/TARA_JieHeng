package assurance

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *HTTPServer) platformRoutes() {
	for _, v := range []struct {
		method, path string
		fn           endpoint
	}{
		{"GET", "attack-path-templates", s.attackPathTemplates}, {"GET", "workspaces/{workspace_id}/attack-path-templates", s.workspaceAttackPathTemplates},
		{"GET", "model-providers", s.providers}, {"POST", "model-providers", s.saveProvider}, {"GET", "model-providers/export", s.exportProviders}, {"POST", "model-provider-imports", s.importProviders}, {"PUT", "model-providers/{resource_id}", s.saveProvider}, {"POST", "model-providers/{resource_id}/model-discoveries", s.discoverModels}, {"POST", "model-providers/{resource_id}/connection-tests", s.testProvider},
		{"GET", "prompt-templates", s.prompts}, {"POST", "prompt-templates", s.savePrompt}, {"PUT", "prompt-templates/{resource_id}", s.savePrompt},
		{"GET", "knowledge-documents", s.knowledgeDocuments}, {"POST", "knowledge-documents", s.saveKnowledge}, {"GET", "knowledge-documents/{resource_id}", s.knowledgeDocument}, {"PUT", "knowledge-documents/{resource_id}", s.saveKnowledge}, {"POST", "knowledge-searches", s.knowledgeSearch},
		{"GET", "knowledge-packs/public-cases", s.publicKnowledgePack}, {"POST", "knowledge-imports", s.importKnowledgePack},
		{"GET", "workspaces/{workspace_id}/automation-plan", s.automationPlan}, {"POST", "workspaces/{workspace_id}/automation-adoptions", s.adoptAutomation},
		{"GET", "platform-events", s.platformEvents}, {"GET", "workspaces/{workspace_id}/quality", s.quality}, {"GET", "workspaces/{workspace_id}/checkpoints", s.checkpoints}, {"POST", "workspaces/{workspace_id}/checkpoints", s.checkpoint}, {"POST", "workspaces/{workspace_id}/restorations", s.restoreCheckpoint},
		{"GET", "workspaces/{workspace_id}/ai-tasks", s.aiTasks}, {"POST", "workspaces/{workspace_id}/ai-tasks", s.createAITask}, {"GET", "workspaces/{workspace_id}/ai-tasks/{resource_id}", s.aiTask}, {"POST", "workspaces/{workspace_id}/ai-tasks/{resource_id}/cancellations", s.cancelAITask}, {"POST", "workspaces/{workspace_id}/ai-tasks/{resource_id}/adoptions", s.adoptAITask}, {"POST", "workspaces/{workspace_id}/ai-tasks/{resource_id}/retries", s.retryAITask},
	} {
		s.route(v.method+" "+apiPrefix+"/"+v.path, v.fn)
	}
}
func (s *HTTPServer) providers(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	rows, e := s.repository.platformList(r.Context(), "provider")
	if e != nil {
		return e
	}
	out := []Object{}
	for _, p := range rows {
		o := publicProvider(p)
		o["health"] = s.ai.providerHealth(dStr(p["id"]))
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool { return dInt(out[i]["priority"]) < dInt(out[j]["priority"]) })
	return writeJSON(w, r, 200, Object{"items": out})
}
func resourceCreation(r *http.Request, kind string, p Object) (string, string, error) {
	key, e := requestKey(r)
	if e != nil {
		return "", "", e
	}
	return kind + "_" + Digest(Object{"key": key})[:20], Digest(p), nil
}
func (s *HTTPServer) saveProvider(w http.ResponseWriter, r *http.Request) error {
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	id := r.PathValue("resource_id")
	rev := int64(-1)
	fp := ""
	if r.Method == "PUT" {
		rev, e = expectedRevision(r)
	} else {
		id, fp, e = resourceCreation(r, "provider", p)
		if e == nil {
			if old, x := s.repository.platformGet(r.Context(), "provider", id); x == nil {
				if old["_create_fingerprint"] != fp {
					return &ConflictError{Detail: "同一请求标识对应不同配置"}
				}
				setETag(w, old)
				return writeJSON(w, r, 200, publicProvider(old))
			}
		}
	}
	if e != nil {
		return e
	}
	out, e := s.repository.saveProvider(withCreation(r.Context(), fp), id, rev, p, actorID(r))
	if e != nil {
		return e
	}
	setETag(w, out)
	w.Header().Set("Location", apiPrefix+"/model-providers/"+id)
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	return writeJSON(w, r, status, out)
}
func providerConfig(p Object) Object {
	o := Object{}
	for key := range platformModel("AIProvider") {
		if key != "api_key" && key != "clear_key" {
			if v, exists := p[key]; exists && v != nil {
				o[key] = v
			}
		}
	}
	return o
}
func (s *HTTPServer) exportProviders(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.platformList(r.Context(), "provider")
	if e != nil {
		return e
	}
	out := []Object{}
	for _, p := range rows {
		out = append(out, providerConfig(p))
	}
	w.Header().Set("Content-Disposition", `attachment; filename="model-providers.json"`)
	return writeJSON(w, r, 200, Object{"schema": "jieheng.providers/1", "credentials_included": false, "providers": out})
}
func (s *HTTPServer) importProviders(w http.ResponseWriter, r *http.Request) error {
	body, e := readPayload(w, r)
	if e != nil {
		return e
	}
	if body["schema"] != "jieheng.providers/1" {
		return dError("配置文件版本不受支持")
	}
	rows := dList(body["providers"])
	if len(rows) < 1 || len(rows) > 50 {
		return dError("一次导入 1–50 个模型服务")
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	fp := Digest(body)
	prepared := []Object{}
	for i, v := range rows {
		p := Clone(dObj(v))
		if dBool(p["api_key"]) {
			return dError("配置导入不接受凭据，请导入后单独配置")
		}
		p["enabled"] = false
		validated, e := validateProvider(p)
		if e != nil {
			return e
		}
		delete(validated, "api_key")
		delete(validated, "clear_key")
		validated["id"] = fmt.Sprintf("provider_%s_%d", Digest(Object{"key": key})[:16], i)
		validated["revision"], validated["created_at"], validated["updated_at"], validated["_create_fingerprint"] = 0, Stamp(), Stamp(), fp
		prepared = append(prepared, validated)
	}
	tx, e := s.repository.db.BeginTx(r.Context(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, p := range prepared {
		old, err := readObject(tx.QueryRowContext(r.Context(), "SELECT body FROM platform_resources WHERE kind='provider' AND id=?", p["id"]))
		if err == nil {
			if old["_create_fingerprint"] != fp {
				return &ConflictError{Detail: "同一导入标识对应不同内容"}
			}
			continue
		}
		if err != ErrNotFound {
			return err
		}
		raw, _ := repoBody(p)
		if _, e = tx.ExecContext(r.Context(), "INSERT INTO platform_resources(kind,id,body) VALUES('provider',?,?)", p["id"], raw); e != nil {
			return e
		}
	}
	if e = platformEvent(r.Context(), tx, actorID(r), "import-provider-configs", key, Object{"count": len(prepared), "credentials_included": false}); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	return writeJSON(w, r, 201, Object{"count": len(prepared), "enabled": false, "message": "配置已导入为停用状态，请填写凭据并测试连接后启用"})
}
func (s *HTTPServer) discoverModels(w http.ResponseWriter, r *http.Request) error {
	p, e := s.repository.platformGet(r.Context(), "provider", r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	rows, e := s.ai.modelList(r.Context(), p)
	if e != nil {
		return fail(502, "模型发现失败", e.Error())
	}
	return writeJSON(w, r, 200, Object{"models": rows})
}
func (s *HTTPServer) testProvider(w http.ResponseWriter, r *http.Request) error {
	body, e := readPayload(w, r)
	if e != nil {
		return e
	}
	role := dStr(body["role"])
	if role == "" {
		role = "analysis"
	}
	if role != "analysis" && role != "vision" && role != "embedding" {
		return dError("测试角色无效")
	}
	p, e := s.repository.platformGet(r.Context(), "provider", r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	if !dBool(dObj(p["models"])[role]) {
		return dError("请先配置所需模型")
	}
	start := time.Now()
	result := modelResult{}
	if role == "embedding" {
		_, e = s.ai.embed(r.Context(), p, []string{"模型连接测试"})
	} else {
		input := modelInput{System: "Return only JSON: {\"ok\":true}", Text: "This is a connection test. No project information is included.", Purpose: role}
		testConfig := Clone(p)
		testConfig["max_output_tokens"] = 256
		payload, model := generationBody(testConfig, input)
		response, err := s.ai.upstream(r.Context(), p, "generation", "POST", payload)
		e = err
		if e == nil {
			result, e = parseGeneration(p, response)
			result.Model = model
		}
	}
	s.ai.recordCall(p, result, time.Since(start), e)
	if e != nil {
		return fail(502, "连接测试失败", e.Error())
	}
	return writeJSON(w, r, 200, Object{"ok": true, "latency_ms": time.Since(start).Milliseconds(), "role": role, "model": dObj(p["models"])[role], "note": "连接成功；视觉识别准确性须用实际图片任务验证"})
}
func (s *HTTPServer) prompts(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.platformList(r.Context(), "prompt")
	if e != nil {
		return e
	}
	out := aiDefaultPrompts()
	for _, row := range rows {
		out = append(out, publicResource(row))
	}
	return writeJSON(w, r, 200, Object{"items": out, "contract_version": aiContractVersion})
}
func (s *HTTPServer) savePrompt(w http.ResponseWriter, r *http.Request) error {
	return s.savePlatformResource(w, r, "prompt", "AIPrompt")
}
func (s *HTTPServer) saveKnowledge(w http.ResponseWriter, r *http.Request) error {
	return s.savePlatformResource(w, r, "knowledge", "KnowledgeDocument")
}
func (s *HTTPServer) savePlatformResource(w http.ResponseWriter, r *http.Request, kind, model string) error {
	body, e := readPayload(w, r)
	if e != nil {
		return e
	}
	p, e := dValidateModel(model, body)
	if e != nil {
		return e
	}
	id := r.PathValue("resource_id")
	rev := int64(-1)
	fp := ""
	if kind == "knowledge" {
		if scope := dStr(p["workspace_id"]); scope != "" {
			if _, e = s.repository.Get(r.Context(), scope); e != nil {
				return e
			}
		}
		if e = s.manageKnowledge(r, dStr(p["workspace_id"])); e != nil {
			return e
		}
		if id != "" {
			old, e := s.repository.platformGet(r.Context(), kind, id)
			if e != nil {
				return e
			}
			if e = s.manageKnowledge(r, dStr(old["workspace_id"])); e != nil {
				return e
			}
			if old["workspace_id"] != p["workspace_id"] {
				return dError("资料所属项目不可直接变更，请另建资料版本")
			}
		}
	}
	if r.Method == "PUT" {
		rev, e = expectedRevision(r)
	} else {
		id, fp, e = resourceCreation(r, kind, body)
		if e == nil {
			if old, x := s.repository.platformGet(r.Context(), kind, id); x == nil {
				if old["_create_fingerprint"] != fp {
					return &ConflictError{Detail: "同一请求标识对应不同内容"}
				}
				return writeJSON(w, r, 200, publicResource(old))
			}
		}
	}
	if e != nil {
		return e
	}
	out, e := s.repository.platformPut(withCreation(r.Context(), fp), kind, id, rev, p, actorID(r))
	if e != nil {
		return e
	}
	setETag(w, out)
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	return writeJSON(w, r, status, publicResource(out))
}
func publicResource(p Object) Object { o := Clone(p); delete(o, "_create_fingerprint"); return o }
func (s *HTTPServer) knowledgeDocuments(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "workspace_id")
	if e != nil {
		return e
	}
	scope := q.Get("workspace_id")
	if e = s.readProject(r, scope); e != nil {
		return e
	}
	rows, e := s.repository.platformList(r.Context(), "knowledge")
	if e != nil {
		return e
	}
	indexes, e := s.repository.platformList(r.Context(), "knowledge_index")
	if e != nil {
		return e
	}
	indexMap := map[string]Object{}
	for _, i := range indexes {
		indexMap[dStr(i["id"])] = i
	}
	out := []Object{}
	for _, doc := range rows {
		if dStr(doc["workspace_id"]) != "" && doc["workspace_id"] != scope {
			continue
		}
		index := indexMap[dStr(doc["id"])]
		o := knowledgeIndexPublic(doc, index)
		if index != nil {
			p, err := s.repository.platformGet(r.Context(), "provider", dStr(index["provider_id"]))
			if err != nil || !dBool(p["enabled"]) || providerSignature(p) != index["signature"] {
				o["index_state"] = "stale"
			}
		}
		o["characters"] = len([]rune(dStr(o["content"])))
		delete(o, "content")
		out = append(out, publicResource(o))
	}
	return writeJSON(w, r, 200, Object{"items": out})
}
func (s *HTTPServer) knowledgeDocument(w http.ResponseWriter, r *http.Request) error {
	p, e := s.repository.platformGet(r.Context(), "knowledge", r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	if e = s.readProject(r, dStr(p["workspace_id"])); e != nil {
		return e
	}
	setETag(w, p)
	return writeJSON(w, r, 200, publicResource(p))
}
func (s *HTTPServer) knowledgeSearch(w http.ResponseWriter, r *http.Request) error {
	p, e := platformPayload(w, r, map[string]dField{"workspace_id": dsd("", 100), "query": ds(1, 4000), "document_ids": dl(ds(1, 100), 0, 100, false), "embedding_provider_id": dsd("", 100), "allow_external": db(false, true)})
	if e != nil {
		return e
	}
	scope := dStr(p["workspace_id"])
	if e = s.readProject(r, scope); e != nil {
		return e
	}
	ids := []string{}
	for _, id := range dList(p["document_ids"]) {
		ids = append(ids, dStr(id))
	}
	out, e := s.ai.searchKnowledge(r.Context(), scope, dStr(p["query"]), ids, dStr(p["embedding_provider_id"]), dBool(p["allow_external"]), 8)
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) platformEvents(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.platformAudit(r.Context())
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"items": rows})
}
func (s *HTTPServer) quality(w http.ResponseWriter, r *http.Request) error {
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, Quality(ws))
}
func (s *HTTPServer) checkpoints(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.repository.checkpoints(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"items": rows})
}
func (s *HTTPServer) checkpoint(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	p, e := platformPayload(w, r, map[string]dField{"name": ds(1, 200)})
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	out, e := s.repository.createCheckpoint(r.Context(), r.PathValue("workspace_id"), dStr(p["name"]), actorID(r), rev, key)
	if e != nil {
		return e
	}
	return writeJSON(w, r, 201, out)
}
func (s *HTTPServer) restoreCheckpoint(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	p, e := platformPayload(w, r, map[string]dField{"checkpoint_id": ds(1, 100), "reason": ds(1, 2000)})
	if e != nil {
		return e
	}
	out, _, e := s.repository.restoreCheckpoint(r.Context(), r.PathValue("workspace_id"), dStr(p["checkpoint_id"]), key, dStr(p["reason"]), actorID(r), rev)
	if e != nil {
		return e
	}
	setETag(w, out)
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) aiTasks(w http.ResponseWriter, r *http.Request) error {
	rows, e := s.ai.jobs(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"items": rows})
}
func (s *HTTPServer) aiTask(w http.ResponseWriter, r *http.Request) error {
	job, e := s.ai.job(r.Context(), r.PathValue("workspace_id"), r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, publicJob(job, true))
}
func (s *HTTPServer) createAITask(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	p, e := readPayload(w, r)
	if e != nil {
		return e
	}
	job, _, e := s.ai.createJob(r.Context(), r.PathValue("workspace_id"), key, rev, p, actorID(r))
	if e != nil {
		return e
	}
	w.Header().Set("Location", apiPrefix+"/workspaces/"+r.PathValue("workspace_id")+"/ai-tasks/"+dStr(job["id"]))
	return writeJSON(w, r, 202, job)
}
func (s *HTTPServer) cancelAITask(w http.ResponseWriter, r *http.Request) error {
	out, e := s.ai.cancelJob(r.Context(), r.PathValue("workspace_id"), r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) adoptAITask(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	p, e := platformPayload(w, r, map[string]dField{"owner": ds(1, 200), "reviewed": db(false, true)})
	if e != nil {
		return e
	}
	if p["reviewed"] != true {
		return dError("请确认已审阅草案及待确认项")
	}
	out, _, e := s.ai.adoptJob(r.Context(), r.PathValue("workspace_id"), r.PathValue("resource_id"), key, dStr(p["owner"]), actorID(r), rev)
	if e != nil {
		return e
	}
	setETag(w, out)
	return writeJSON(w, r, 200, out)
}
func (s *HTTPServer) retryAITask(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	job, e := s.ai.job(r.Context(), r.PathValue("workspace_id"), r.PathValue("resource_id"))
	if e != nil {
		return e
	}
	if job["status"] == "running" || job["status"] == "queued" {
		return dError("任务仍在执行，可先取消")
	}
	p, e := platformPayload(w, r, map[string]dField{"instructions": dsd("", 12000)})
	if e != nil {
		return e
	}
	for _, raw := range dList(job["providers"]) {
		previous := dObj(raw)
		current, err := s.repository.platformGet(r.Context(), "provider", dStr(previous["id"]))
		if err != nil {
			return err
		}
		if Digest(publicProvider(current)) != Digest(previous) {
			return &ConflictError{Detail: "模型服务配置已变化，请新建任务并重新确认发送目标"}
		}
	}
	request := Clone(dObj(job["request"]))
	if instructions, ok := p["instructions"]; ok {
		request["instructions"] = instructions
	}
	out, _, e := s.ai.createJob(r.Context(), r.PathValue("workspace_id"), key, rev, request, actorID(r))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 202, out)
}

type creationContextKey struct{}

func withCreation(ctx context.Context, fp string) context.Context {
	return context.WithValue(ctx, creationContextKey{}, fp)
}
func creationFingerprint(ctx context.Context) string {
	v, _ := ctx.Value(creationContextKey{}).(string)
	return v
}

var _ = strings.TrimSpace

func platformPayload(w http.ResponseWriter, r *http.Request, schema map[string]dField) (Object, error) {
	raw, e := readPayload(w, r)
	if e != nil {
		return nil, e
	}
	out := Object{}
	for key := range raw {
		if _, ok := schema[key]; !ok {
			return nil, dError("未知字段：" + key)
		}
	}
	for key, f := range schema {
		value, ok := raw[key]
		if !ok {
			value = resolveDefault(f.fallback)
		}
		value, e = dValidateField(f, value)
		if e != nil {
			return nil, dError("字段 " + key + " 无效：" + e.Error())
		}
		out[key] = value
	}
	return out, nil
}
