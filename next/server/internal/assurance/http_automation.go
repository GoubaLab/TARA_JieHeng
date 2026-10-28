package assurance

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
)

func (s *HTTPServer) automationPlan(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, AutomationPlan(ws))
}
func (s *HTTPServer) adoptAutomation(w http.ResponseWriter, r *http.Request) error {
	rev, e := expectedRevision(r)
	if e != nil {
		return e
	}
	key, e := requestKey(r)
	if e != nil {
		return e
	}
	p, e := platformPayload(w, r, map[string]dField{"plan_hash": ds(64, 64), "candidate_ids": dl(ds(1, 150), 1, 50, false), "owner": ds(1, 200), "reviewed": db(false, true)})
	if e != nil {
		return e
	}
	if p["reviewed"] != true {
		return dError("请确认已审阅匹配条件、适用前提与待确认项")
	}
	out, e := s.repository.adoptAutomation(r.Context(), r.PathValue("workspace_id"), key, actorID(r), rev, p, s.catalog)
	if e != nil {
		return e
	}
	setETag(w, out)
	return writeJSON(w, r, 201, out)
}
func (r *Repository) adoptAutomation(ctx context.Context, workspaceID, key, actor string, expected int64, p Object, catalog *Catalog) (Object, error) {
	if e := validateIdempotency(key); e != nil {
		return nil, e
	}
	fp := Digest(Object{"action": "adopt-automation", "payload": p, "revision": expected})
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	receipt, e := readReceipt(ctx, tx, workspaceID, key, fp)
	if e != nil || receipt != nil {
		return receipt, e
	}
	ws, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM workspaces WHERE id=?", workspaceID))
	if e != nil {
		return nil, e
	}
	if int64(dInt(ws["revision"])) != expected {
		return nil, &RevisionError{Current: int64(dInt(ws["revision"]))}
	}
	plan := AutomationPlan(ws)
	if plan["plan_hash"] != p["plan_hash"] {
		return nil, &ConflictError{Detail: "模型或规则已变化，请刷新候选并重新审阅"}
	}
	ids := []string{}
	for _, v := range dList(p["candidate_ids"]) {
		ids = append(ids, dStr(v))
	}
	proposal, e := AutomationProposal(ws, ids)
	if e != nil {
		return nil, e
	}
	next, counts, e := validateAIProposal(ws, proposal, "analysis", dStr(p["owner"]), catalog)
	if e != nil {
		return nil, e
	}
	cp, e := checkpointInTx(ctx, tx, ws, "采用自动分析候选前", actor)
	if e != nil {
		return nil, e
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	snapshots := []any{}
	for _, raw := range dList(plan["candidates"]) {
		candidate := dObj(raw)
		if selected[dStr(candidate["id"])] {
			snapshots = append(snapshots, Clone(candidate))
		}
	}
	next["automation_adoptions"] = append(dList(next["automation_adoptions"]), Object{"id": Identity("automation"), "actor": actor, "owner": p["owner"], "at": Stamp(), "base_revision": expected, "rule_version": plan["rule_version"], "plan_hash": plan["plan_hash"], "candidate_ids": p["candidate_ids"], "candidates": snapshots, "checkpoint_id": cp["id"]})
	result := Object{"workspace_id": workspaceID, "revision": next["revision"], "checkpoint_id": cp["id"], "counts": counts}
	if e = persistRecoveredWorkspace(ctx, tx, next, key, fp, "adopt-automation", Object{"actor": actor, "owner": p["owner"], "candidate_ids": p["candidate_ids"], "rule_version": plan["rule_version"], "checkpoint_id": cp["id"]}, result); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return result, nil
}
func (s *HTTPServer) publicKnowledgePack(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	raw, e := os.ReadFile(filepath.Join(s.root, "next", "data", "public-knowledge", "seed.json"))
	if e != nil {
		return fail(503, "公开案例包未安装", "请安装公开案例种子文件，或导入自行整理的知识包")
	}
	pack, e := DecodeJSON(raw)
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, pack)
}
func (s *HTTPServer) importKnowledgePack(w http.ResponseWriter, r *http.Request) error {
	if e := s.manageKnowledge(r, ""); e != nil {
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
	out, e := s.repository.importKnowledgePack(r.Context(), p, key, actorID(r))
	if e != nil {
		return e
	}
	return writeJSON(w, r, 201, out)
}
