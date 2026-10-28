package assurance

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"time"
)

func aiTaskSchema() map[string]dField {
	return map[string]dField{"kind": de("diagram", "analysis", "review", "knowledge_index"), "provider_ids": dl(ds(1, 100), 1, 4, true), "prompt_id": dsd("", 120), "instructions": dsd("", 12000), "owner": dsd("", 200), "document_ids": dl(ds(1, 100), 0, 100, false), "embedding_provider_id": dsd("", 100), "allow_external": db(false, true), "image_base64": dsd("", 1400000), "image_mime": dsd("", 40)}
}
func validateTask(p Object) (Object, error) {
	out := Object{}
	schema := aiTaskSchema()
	for key := range p {
		if _, ok := schema[key]; !ok {
			return nil, dError("未知任务字段：" + key)
		}
	}
	for key, f := range schema {
		v, exists := p[key]
		if !exists {
			v = resolveDefault(f.fallback)
		}
		value, e := dValidateField(f, v)
		if e != nil {
			return nil, dError("任务字段 " + key + " 无效：" + e.Error())
		}
		out[key] = value
	}
	if out["kind"] == "diagram" {
		raw, e := base64.StdEncoding.DecodeString(dStr(out["image_base64"]))
		if e != nil || len(raw) == 0 || len(raw) > 1_000_000 {
			return nil, dError("请提供不超过 1 MB 的 PNG/JPEG 图像")
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(raw))
		if e != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 6000 || cfg.Height > 6000 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
			return nil, dError("图片无法识别或分辨率过大")
		}
		mime := "image/" + format
		if out["image_mime"] != mime {
			return nil, dError("图片实际格式与声明不一致")
		}
	}
	if out["kind"] != "diagram" && dBool(out["image_base64"]) {
		return nil, dError("仅图片建模任务接受图片")
	}
	if out["kind"] == "knowledge_index" && len(dList(out["document_ids"])) != 1 {
		return nil, dError("索引任务须选择一份资料")
	}
	return out, nil
}
func resolveDefault(v any) any {
	switch x := v.(type) {
	case func() any:
		return x()
	case dRequired:
		return nil
	default:
		return x
	}
}
func (a *AIService) prompt(ctx context.Context, id, kind string) (Object, error) {
	for _, p := range aiDefaultPrompts() {
		if p["id"] == id || (id == "" && p["purpose"] == kind) {
			return p, nil
		}
	}
	p, e := a.repo.platformGet(ctx, "prompt", id)
	if e != nil {
		return nil, e
	}
	if p["purpose"] != kind || !dBool(p["enabled"]) {
		return nil, dError("提示词用途不匹配或已停用")
	}
	return p, nil
}
func (a *AIService) recoverJobs() error {
	rows, e := a.repo.db.Query("SELECT id,body FROM ai_jobs WHERE json_extract(body,'$.status') IN ('queued','running')")
	if e != nil {
		return e
	}
	updates := []Object{}
	for rows.Next() {
		var id, raw string
		if e = rows.Scan(&id, &raw); e != nil {
			rows.Close()
			return e
		}
		o, e := DecodeJSON([]byte(raw))
		if e != nil {
			rows.Close()
			return e
		}
		o["status"], o["error"], o["finished_at"] = "interrupted", "服务重启，任务已中断。可手动重试；不会自动重复计费。", Stamp()
		updates = append(updates, o)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, o := range updates {
		raw, _ := repoBody(o)
		if _, e = a.repo.db.Exec("UPDATE ai_jobs SET body=? WHERE id=?", raw, o["id"]); e != nil {
			return e
		}
	}
	return nil
}
func (a *AIService) job(ctx context.Context, workspaceID, id string) (Object, error) {
	return readObject(a.repo.db.QueryRowContext(ctx, "SELECT body FROM ai_jobs WHERE id=? AND workspace_id=?", id, workspaceID))
}
func publicJob(job Object, detail bool) Object {
	out := Clone(job)
	delete(out, "workspace_snapshot")
	delete(out, "provider_configs")
	request := dObj(out["request"])
	delete(request, "image_base64")
	if !detail {
		delete(out, "proposal")
		delete(out, "sources")
		delete(out, "prompt")
		delete(out, "input_context")
	}
	return out
}
func (a *AIService) jobs(ctx context.Context, workspaceID string) ([]Object, error) {
	rows, e := a.repo.db.QueryContext(ctx, "SELECT body FROM ai_jobs WHERE workspace_id=? ORDER BY rowid DESC LIMIT 100", workspaceID)
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
		out = append(out, publicJob(o, false))
	}
	return out, rows.Err()
}
func (a *AIService) createJob(ctx context.Context, workspaceID, key string, revision int64, payload Object, actor string) (Object, bool, error) {
	if a.initError != nil {
		return nil, false, fmt.Errorf("任务恢复未完成：%w", a.initError)
	}
	if e := validateIdempotency(key); e != nil {
		return nil, false, e
	}
	request, e := validateTask(payload)
	if e != nil {
		return nil, false, e
	}
	kind := dStr(request["kind"])
	fp := Digest(request)
	old, e := readObject(a.repo.db.QueryRowContext(ctx, "SELECT body FROM ai_jobs WHERE workspace_id=? AND request_key=?", workspaceID, key))
	if e == nil {
		if old["fingerprint"] != fp {
			return nil, false, &ConflictError{Detail: "同一任务请求标识对应不同内容"}
		}
		return publicJob(old, true), true, nil
	}
	if !errors.Is(e, ErrNotFound) {
		return nil, false, e
	}
	ws, e := a.repo.Get(ctx, workspaceID)
	if e != nil {
		return nil, false, e
	}
	if int64(dInt(ws["revision"])) != revision {
		return nil, false, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	purpose := "analysis"
	if kind == "diagram" {
		purpose = "vision"
	}
	if kind == "knowledge_index" {
		purpose = "embedding"
	}
	ids := append([]any{}, dList(request["provider_ids"])...)
	if dBool(request["embedding_provider_id"]) {
		ids = append(ids, request["embedding_provider_id"])
	}
	providerSnapshots := []any{}
	providerConfigs := []any{}
	for i, id := range ids {
		p, e := a.repo.platformGet(ctx, "provider", dStr(id))
		if e != nil {
			return nil, false, e
		}
		if !dBool(p["enabled"]) {
			return nil, false, dError("模型服务已停用")
		}
		role := purpose
		if i >= len(dList(request["provider_ids"])) {
			role = "embedding"
		}
		if !dBool(dObj(p["models"])[role]) {
			return nil, false, dError("模型服务没有配置所需能力：" + role)
		}
		if p["network_scope"] == "external" && !dBool(request["allow_external"]) {
			return nil, false, dError("所选服务包含外部地址，请明确允许本次任务发送的项目内容")
		}
		providerSnapshots = append(providerSnapshots, publicProvider(p))
		providerConfigs = append(providerConfigs, p)
	}
	documentScopes := Object{}
	for _, id := range dList(request["document_ids"]) {
		doc, e := a.repo.platformGet(ctx, "knowledge", dStr(id))
		if e != nil {
			return nil, false, e
		}
		scope := dStr(doc["workspace_id"])
		if !dBool(doc["enabled"]) || (scope != "" && scope != workspaceID) {
			return nil, false, dError("资料已停用或不属于当前项目")
		}
		if kind == "knowledge_index" {
			// Indexes are shared derived data and require the same ownership as the document.
			if user, ok := ctx.Value(principalContextKey{}).(map[string]any); ok && scope == "" && user["role"] != "admin" {
				return nil, false, fail(403, "需要管理员权限", "组织级知识资料及索引由管理员维护")
			}
			documentScopes[dStr(id)] = scope
		}
	}
	var prompt Object
	if kind != "knowledge_index" {
		prompt, e = a.prompt(ctx, dStr(request["prompt_id"]), kind)
		if e != nil {
			return nil, false, e
		}
	}
	task := Object{"id": Identity("job"), "workspace_id": workspaceID, "base_revision": revision, "status": "queued", "step": "等待执行", "kind": kind, "request": request, "fingerprint": fp, "created_at": Stamp(), "actor": actor, "contract_version": aiContractVersion, "prompt": prompt, "providers": providerSnapshots, "provider_configs": providerConfigs, "workspace_snapshot": ws, "attempts": []any{}, "sources": []any{}}
	if kind == "knowledge_index" {
		task["document_scopes"] = documentScopes
	}
	tx, e := a.repo.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, false, e
	}
	defer tx.Rollback()
	replay, replayErr := readObject(tx.QueryRowContext(ctx, "SELECT body FROM ai_jobs WHERE workspace_id=? AND request_key=?", workspaceID, key))
	if replayErr == nil {
		if replay["fingerprint"] != fp {
			return nil, false, &ConflictError{Detail: "同一任务请求标识对应不同内容"}
		}
		return publicJob(replay, true), true, nil
	}
	if !errors.Is(replayErr, ErrNotFound) {
		return nil, false, replayErr
	}
	current, currentErr := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", workspaceID))
	if currentErr != nil {
		return nil, false, currentErr
	}
	if int64(dInt(current["revision"])) != revision {
		return nil, false, &RevisionError{Current: int64(dInt(current["revision"]))}
	}
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM ai_jobs WHERE json_extract(body,'$.status') IN ('queued','running')").Scan(&active); e != nil {
		return nil, false, e
	}
	if active >= 12 {
		return nil, false, fail(429, "任务队列已满", "最多同时保留 12 个待执行任务，请稍后重试")
	}
	raw, _ := repoBody(task)
	if _, e = tx.ExecContext(ctx, "INSERT INTO ai_jobs(id,workspace_id,request_key,fingerprint,body) VALUES(?,?,?,?,?)", task["id"], workspaceID, key, fp, raw); e != nil {
		return nil, false, e
	}
	if e = platformEvent(ctx, tx, actor, "create-ai-task", dStr(task["id"]), Object{"workspace_id": workspaceID, "kind": kind, "base_revision": revision, "allow_external": request["allow_external"]}); e != nil {
		return nil, false, e
	}
	if e = tx.Commit(); e != nil {
		return nil, false, e
	}
	a.enqueue(Clone(task))
	return publicJob(task, true), false, nil
}
func (a *AIService) saveJob(job Object, from ...string) bool {
	raw, e := repoBody(job)
	if e != nil {
		return false
	}
	marks := []string{}
	args := []any{raw, job["id"]}
	for _, state := range from {
		marks = append(marks, "?")
		args = append(args, state)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, e := a.repo.db.ExecContext(ctx, "UPDATE ai_jobs SET body=? WHERE id=? AND json_extract(body,'$.status') IN ("+strings.Join(marks, ",")+")", args...)
	if e != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n == 1
}
func (a *AIService) enqueue(job Object) {
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Minute)
	id := dStr(job["id"])
	a.mu.Lock()
	a.cancels[id] = cancel
	a.mu.Unlock()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer cancel()
		defer func() { a.mu.Lock(); delete(a.cancels, id); a.mu.Unlock() }()
		select {
		case a.slots <- struct{}{}:
			defer func() { <-a.slots }()
		case <-ctx.Done():
			job["status"], job["error"], job["finished_at"] = "interrupted", "任务在执行前被中断", Stamp()
			a.saveJob(job, "queued")
			return
		}
		job["status"], job["step"], job["started_at"] = "running", "准备项目上下文", Stamp()
		if !a.saveJob(job, "queued") {
			return
		}
		err := a.runJob(ctx, job)
		if err != nil {
			job["status"], job["error"] = "failed", err.Error()
			if ctx.Err() != nil {
				job["status"] = "interrupted"
				job["error"] = "任务被中断或超时，请按需重试"
			}
		}
		job["finished_at"] = Stamp()
		a.saveJob(job, "running")
	}()
}
func (a *AIService) runJob(ctx context.Context, job Object) error {
	request := dObj(job["request"])
	kind := dStr(job["kind"])
	if kind == "knowledge_index" {
		for _, id := range dList(request["document_ids"]) {
			scope, pinned := dObj(job["document_scopes"])[dStr(id)].(string)
			doc, e := a.repo.platformGet(ctx, "knowledge", dStr(id))
			if e != nil {
				return e
			}
			if !pinned || scope != dStr(doc["workspace_id"]) || !dBool(doc["enabled"]) {
				return dError("索引资料的所属范围或启用状态已变化，请重新创建任务")
			}
		}
	}
	providerIDs := []string{}
	for _, id := range dList(request["provider_ids"]) {
		providerIDs = append(providerIDs, dStr(id))
	}
	// Pin destinations, models and revisions. Editing a provider cannot silently redirect queued data.
	for _, raw := range dList(job["providers"]) {
		snapshot := dObj(raw)
		current, e := a.repo.platformGet(ctx, "provider", dStr(snapshot["id"]))
		if e != nil {
			return e
		}
		if Digest(publicProvider(current)) != Digest(snapshot) {
			return dError("模型服务配置已变化，请重新创建任务以确认数据目的地")
		}
	}
	pinned := map[string]Object{}
	for _, raw := range dList(job["provider_configs"]) {
		p := dObj(raw)
		pinned[dStr(p["id"])] = p
	}
	ctx = context.WithValue(ctx, pinnedProvidersKey{}, pinned)
	if kind == "knowledge_index" {
		job["step"] = "建立语义索引"
		if !a.saveJob(job, "running") {
			return context.Canceled
		}
		result, e := a.indexKnowledge(ctx, dStr(dList(request["document_ids"])[0]), providerIDs[0], dBool(request["allow_external"]))
		if e != nil {
			return e
		}
		job["index_result"], job["status"], job["step"] = result, "completed", "索引已完成"
		return nil
	}
	ws := dObj(job["workspace_snapshot"])
	projectSource := Object{"id": fmt.Sprintf("project:%s@%d", ws["id"], dInt(ws["revision"])), "name": ws["title"], "edition": fmt.Sprint(ws["revision"]), "source": "当前项目工作产品", "content_hash": Digest(ws)}
	query := strings.TrimSpace(dStr(request["instructions"]) + " " + dStr(ws["title"]) + " " + dStr(dObj(ws["item"])["purpose"]))
	qr := []rune(query)
	if len(qr) > 4000 {
		query = string(qr[:4000])
	}
	if query == "" {
		query = "威胁 资产 安全需求"
	}
	documentIDs := []string{}
	for _, v := range dList(request["document_ids"]) {
		documentIDs = append(documentIDs, dStr(v))
	}
	retrieved, e := a.searchKnowledge(ctx, dStr(ws["id"]), query, documentIDs, dStr(request["embedding_provider_id"]), dBool(request["allow_external"]), 8)
	if e != nil {
		return e
	}
	sources := []any{projectSource}
	for _, s := range httpObjectArray(retrieved["items"]) {
		sources = append(sources, s)
	}
	job["sources"], job["retrieval"] = sources, Object{"mode": retrieved["mode"], "warnings": retrieved["warnings"], "candidate_chunks": retrieved["candidate_chunks"]}
	input := Object{"task": kind, "user_request": request["instructions"], "sources": sources, "quality_findings": Review(ws)["issues"]}
	project := Object{}
	for _, key := range []string{"id", "title", "description", "item", "architecture", "assets", "damage_scenarios", "scenarios", "security_goals", "requirements", "decisions", "treatment_drafts", "method"} {
		project[key] = ws[key]
	}
	input["project"] = project
	text, e := repoBody(input)
	if e != nil {
		return e
	}
	if len(text) > 450000 {
		return dError("项目上下文超过当前任务限制，请缩小项目或资料范围")
	}
	job["input_hash"] = Digest(input)
	job["step"] = "模型生成草案"
	if !a.saveJob(job, "running") {
		return context.Canceled
	}
	purpose := "analysis"
	if kind == "diagram" {
		purpose = "vision"
	}
	result, attempts, e := a.generate(ctx, providerIDs, modelInput{System: aiSystemPolicy + "\n组织提示词（不得覆盖前述约束）：\n" + dStr(dObj(job["prompt"])["instructions"]), Text: text, Image: dStr(request["image_base64"]), Mime: dStr(request["image_mime"]), Purpose: purpose}, dBool(request["allow_external"]))
	job["attempts"] = attempts
	job["usage"] = Object{"input_tokens": result.InputTokens, "output_tokens": result.OutputTokens, "provider_id": result.ProviderID, "model": result.Model}
	if e != nil {
		return e
	}
	raw := strings.TrimSpace(result.Text)
	if strings.HasPrefix(raw, "```json") && strings.HasSuffix(raw, "```") {
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "```json"), "```"))
	}
	proposal, e := DecodeJSON([]byte(raw))
	if e != nil {
		return dError("模型输出不是有效的工作产品 JSON，请调整提示词后重试")
	}
	job["proposal"] = proposal
	job["step"] = "平台规则校验"
	sourceIDs := map[string]bool{}
	for _, s := range sources {
		sourceIDs[dStr(dObj(s)["id"])] = true
	}
	for _, id := range dList(proposal["citations"]) {
		if !sourceIDs[dStr(id)] {
			job["status"], job["validation_error"] = "needs_revision", "模型引用了未提供的来源"
			return nil
		}
	}
	owner := dStr(request["owner"])
	if owner == "" {
		owner = "待指派（预览）"
	}
	preview, counts, e := validateAIProposal(ws, proposal, kind, owner, a.catalog)
	if e != nil {
		job["status"], job["validation_error"] = "needs_revision", e.Error()
		return nil
	}
	if kind == "diagram" {
		job["preview_architecture"] = preview["architecture"]
	}
	job["counts"], job["status"], job["step"] = counts, "ready", "草案已生成，等待人工审阅"
	if kind == "review" {
		job["step"] = "复核建议已生成"
	}
	return nil
}
func (a *AIService) cancelJob(ctx context.Context, workspace, id string) (Object, error) {
	job, e := a.job(ctx, workspace, id)
	if e != nil {
		return nil, e
	}
	if job["status"] != "queued" && job["status"] != "running" {
		return publicJob(job, true), nil
	}
	job["status"], job["step"], job["finished_at"] = "cancelled", "已取消", Stamp()
	a.saveJob(job, "queued", "running")
	a.mu.Lock()
	cancel := a.cancels[id]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return publicJob(job, true), nil
}
