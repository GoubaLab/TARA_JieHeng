package assurance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func checkpointInTx(ctx context.Context, tx *sql.Tx, ws Object, name, actor string) (Object, error) {
	cp := Object{"id": Identity("checkpoint"), "workspace_id": ws["id"], "name": name, "created_at": Stamp(), "actor": actor, "revision": ws["revision"], "sha256": Digest(ws), "workspace": ws}
	raw, e := repoBody(cp)
	if e != nil {
		return nil, e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO workspace_checkpoints(id,workspace_id,body) VALUES(?,?,?)", cp["id"], ws["id"], raw)
	return cp, e
}
func publicCheckpoint(cp Object) Object { o := Clone(cp); delete(o, "workspace"); return o }
func (r *Repository) checkpoints(ctx context.Context, workspaceID string) ([]Object, error) {
	rows, e := r.db.QueryContext(ctx, "SELECT body FROM workspace_checkpoints WHERE workspace_id=? ORDER BY rowid DESC LIMIT 100", workspaceID)
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
		cp, e := DecodeJSON([]byte(raw))
		if e != nil {
			return nil, e
		}
		out = append(out, publicCheckpoint(cp))
	}
	return out, rows.Err()
}
func (r *Repository) createCheckpoint(ctx context.Context, workspaceID, name, actor string, expected int64, key string) (Object, error) {
	if e := validateIdempotency(key); e != nil {
		return nil, e
	}
	fp := Digest(Object{"action": "create-checkpoint", "name": name, "expected_revision": expected})
	if _, e := dValidateField(ds(1, 200), name); e != nil {
		return nil, e
	}
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	replay, e := readReceipt(ctx, tx, workspaceID, key, fp)
	if e != nil || replay != nil {
		return replay, e
	}
	ws, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", workspaceID))
	if e != nil {
		return nil, e
	}
	if int64(dInt(ws["revision"])) != expected {
		return nil, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	cp, e := checkpointInTx(ctx, tx, ws, name, actor)
	if e != nil {
		return nil, e
	}
	if e = platformEvent(ctx, tx, actor, "create-checkpoint", dStr(cp["id"]), Object{"workspace_id": workspaceID, "revision": expected}); e != nil {
		return nil, e
	}
	raw, _ := repoBody(publicCheckpoint(cp))
	if _, e = tx.ExecContext(ctx, "INSERT INTO receipts(workspace_id,request_id,fingerprint,body) VALUES(?,?,?,?)", workspaceID, key, fp, raw); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return publicCheckpoint(cp), nil
}
func readReceipt(ctx context.Context, tx *sql.Tx, workspaceID, key, fp string) (Object, error) {
	var existing, raw string
	e := tx.QueryRowContext(ctx, "SELECT fingerprint,body FROM receipts WHERE workspace_id=? AND request_id=?", workspaceID, key).Scan(&existing, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if existing != fp {
		return nil, &ConflictError{Detail: "同一请求标识对应不同操作"}
	}
	return DecodeJSON([]byte(raw))
}
func persistRecoveredWorkspace(ctx context.Context, tx *sql.Tx, next Object, key, fp, action string, detail, result Object) error {
	raw, e := repoBody(next)
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE workspaces SET body=? WHERE id=?", raw, next["id"]); e != nil {
		return e
	}
	if e = repoEvent(ctx, tx, next, action, detail); e != nil {
		return e
	}
	raw, e = repoBody(result)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO receipts(workspace_id,request_id,fingerprint,body) VALUES(?,?,?,?)", next["id"], key, fp, raw)
	return e
}
func (a *AIService) adoptJob(ctx context.Context, workspaceID, id, key, owner, actor string, expected int64) (Object, bool, error) {
	if e := validateIdempotency(key); e != nil {
		return nil, false, e
	}
	if _, e := dValidateField(ds(1, 200), owner); e != nil {
		return nil, false, dError("请填写此次采用建议的责任人")
	}
	fp := Digest(Object{"action": "adopt-ai-proposal", "id": id, "owner": owner})
	tx, e := a.repo.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, false, e
	}
	defer tx.Rollback()
	receipt, e := readReceipt(ctx, tx, workspaceID, key, fp)
	if e != nil || receipt != nil {
		return receipt, receipt != nil, e
	}
	ws, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", workspaceID))
	if e != nil {
		return nil, false, e
	}
	if int64(dInt(ws["revision"])) != expected {
		return nil, false, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	job, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM ai_jobs WHERE id=? AND workspace_id=?", id, workspaceID))
	if e != nil {
		return nil, false, e
	}
	if job["status"] != "ready" || job["kind"] == "review" {
		return nil, false, dError("该任务没有可采用的工作产品草案")
	}
	if int64(dInt(job["base_revision"])) != expected {
		return nil, false, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	// A stale or withdrawn source requires a fresh task; cached citations are not silently trusted.
	for _, raw := range dList(job["sources"]) {
		source := dObj(raw)
		if key := dStr(source["regulation_key"]); key != "" {
			if a.catalog == nil || a.catalog.Regulations == nil {
				return nil, false, dError("引用法规不可用，请核对后重新分析")
			}
			clause, e := a.catalog.Get(key)
			if e != nil || dBool(clause["withdrawn"]) {
				return nil, false, dError("引用法规不可用，请核对后重新分析")
			}
			// Keep the source mapping identical to searchKnowledge so edition and source changes count too.
			doc := Object{"id": key, "name": clause["title"], "source": dObj(clause["facts"])["正文来源"], "edition": dObj(clause["facts"])["版本"], "content": clause["description"], "workspace_id": workspaceID}
			chunks := knowledgeChunks(doc)
			if len(chunks) == 0 || chunks[0]["content_hash"] != source["content_hash"] {
				return nil, false, dError("引用法规版本已变化，请重新分析")
			}
			continue
		}
		docID := dStr(source["document_id"])
		if docID == "" {
			continue
		}
		doc, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind='knowledge' AND id=?", docID))
		if e != nil || !dBool(doc["enabled"]) {
			return nil, false, dError("引用资料已停用或不存在，请重新分析")
		}
		chunks := knowledgeChunks(doc)
		if len(chunks) == 0 || chunks[0]["content_hash"] != source["content_hash"] {
			return nil, false, dError("引用资料版本已变化，请重新分析")
		}
	}
	next, counts, e := validateAIProposal(ws, dObj(job["proposal"]), dStr(job["kind"]), owner, a.catalog)
	if e != nil {
		return nil, false, e
	}
	cp, e := checkpointInTx(ctx, tx, ws, "采用 AI 草案前", actor)
	if e != nil {
		return nil, false, e
	}
	adoptions := dList(next["ai_adoptions"])
	adoptions = append(adoptions, Object{"job_id": id, "actor": actor, "owner": owner, "at": Stamp(), "base_revision": job["base_revision"], "prompt_name": dObj(job["prompt"])["name"], "prompt_revision": dObj(job["prompt"])["revision"], "usage": job["usage"], "citations": dObj(job["proposal"])["citations"], "sources": job["sources"], "checkpoint_id": cp["id"]})
	next["ai_adoptions"] = adoptions
	result := Object{"workspace_id": workspaceID, "revision": next["revision"], "job_id": id, "checkpoint_id": cp["id"], "counts": counts}
	if e = persistRecoveredWorkspace(ctx, tx, next, key, fp, "adopt-ai-proposal", Object{"job_id": id, "actor": actor, "owner": owner, "checkpoint_id": cp["id"], "counts": counts}, result); e != nil {
		return nil, false, e
	}
	job["step"] = "草案已采用"
	job["status"], job["adopted_at"], job["adopted_revision"], job["adopted_by"], job["checkpoint_id"] = "adopted", Stamp(), next["revision"], actor, cp["id"]
	raw, _ := repoBody(job)
	if _, e = tx.ExecContext(ctx, "UPDATE ai_jobs SET body=? WHERE id=?", raw, id); e != nil {
		return nil, false, e
	}
	if e = tx.Commit(); e != nil {
		return nil, false, e
	}
	return result, false, nil
}
func (r *Repository) restoreCheckpoint(ctx context.Context, workspaceID, id, key, reason, actor string, expected int64) (Object, bool, error) {
	if e := validateIdempotency(key); e != nil {
		return nil, false, e
	}
	if _, e := dValidateField(ds(1, 2000), reason); e != nil {
		return nil, false, dError("请填写还原原因")
	}
	fp := Digest(Object{"action": "restore-checkpoint", "id": id, "reason": reason})
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, false, e
	}
	defer tx.Rollback()
	receipt, e := readReceipt(ctx, tx, workspaceID, key, fp)
	if e != nil || receipt != nil {
		return receipt, receipt != nil, e
	}
	ws, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", workspaceID))
	if e != nil {
		return nil, false, e
	}
	if int64(dInt(ws["revision"])) != expected {
		return nil, false, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	cp, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspace_checkpoints WHERE id=? AND workspace_id=?", id, workspaceID))
	if e != nil {
		return nil, false, e
	}
	snapshot := dObj(cp["workspace"])
	if Digest(snapshot) != cp["sha256"] {
		return nil, false, dError("还原点完整性校验失败")
	}
	undo, e := checkpointInTx(ctx, tx, ws, "还原前自动保存", actor)
	if e != nil {
		return nil, false, e
	}
	next := Clone(snapshot)
	next["id"], next["revision"], next["updated_at"], next["method_confirmed"] = workspaceID, expected+1, Stamp(), false
	next["baselines"] = ws["baselines"] // Frozen history is immutable and remains discoverable.
	result := Object{"workspace_id": workspaceID, "revision": expected + 1, "checkpoint_id": id, "undo_checkpoint_id": undo["id"]}
	if e = persistRecoveredWorkspace(ctx, tx, next, key, fp, "restore-checkpoint", Object{"checkpoint_id": id, "actor": actor, "reason": reason, "undo_checkpoint_id": undo["id"]}, result); e != nil {
		return nil, false, e
	}
	if e = tx.Commit(); e != nil {
		return nil, false, e
	}
	return result, false, nil
}
func Quality(ws Object) Object {
	review := Review(ws)
	issues := append([]any{}, dList(review["issues"])...)
	covered := map[string]bool{}
	for _, s := range dList(ws["scenarios"]) {
		for _, id := range scenarioAssets(dObj(s)) {
			covered[dStr(id)] = true
		}
	}
	uncovered := []any{}
	for _, asset := range dList(ws["assets"]) {
		a := dObj(asset)
		if !covered[dStr(a["id"])] {
			uncovered = append(uncovered, Object{"id": a["id"], "name": a["name"]})
			issues = append(issues, Object{"stage": "assessment", "reference": a["id"], "detail": "资产尚未关联威胁场景或不适用说明：" + dStr(a["name"]), "code": "asset_coverage"})
		}
	}
	flowQuestions := []any{}
	for _, flow := range dList(dObj(ws["architecture"])["flows"]) {
		f := dObj(flow)
		if f["channel_kind"] != "energy" && dStr(f["protocol"]) == "" && dStr(dObj(f["communication"])["technology"]) == "" {
			flowQuestions = append(flowQuestions, Object{"id": f["id"], "name": f["name"]})
		}
	}
	return Object{"ready": review["ready"], "issues": issues, "uncovered_assets": uncovered, "communication_to_confirm": flowQuestions, "counts": Object{"assets": len(dList(ws["assets"])), "covered_assets": len(covered), "scenarios": len(dList(ws["scenarios"])), "requirements": len(dList(ws["requirements"])), "evidence": len(dList(ws["evidence"]))}, "note": fmt.Sprintf("修订 %d 的确定性检查；覆盖提醒不代替工程评审。", dInt(ws["revision"]))}
}
