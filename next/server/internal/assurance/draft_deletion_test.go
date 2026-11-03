package assurance

import (
	"context"
	"fmt"
	"testing"
)

func TestDiscardUnratedDraftPreservesOtherProducts(t *testing.T) {
	ws, _, err := validateAIProposal(NewWorkspace("draft review", ""), analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	dObj(dList(ws["scenarios"])[0])["paths"] = []any{}
	before := Digest(ws)
	draft := dObj(dList(ws["treatment_drafts"])[0])
	scenarioID := dStr(draft["scenario_id"])
	if Risk(dObj(dList(ws["scenarios"])[0]), dObj(ws["method"]))["level"] != nil {
		t.Fatal("fixture must remain unassessed")
	}
	next, id, err := Apply(ws, "discard-treatment-draft", Object{"scenario_id": scenarioID})
	if err != nil || id != scenarioID {
		t.Fatal(id, err)
	}
	if len(dList(next["treatment_drafts"])) != 0 {
		t.Fatal("draft not discarded")
	}
	for _, key := range []string{"scenarios", "security_goals", "requirements", "decisions", "evidence", "baselines"} {
		if Digest(next[key]) != Digest(ws[key]) {
			t.Fatalf("discard changed %s", key)
		}
	}
	if Digest(ws) != before {
		t.Fatal("source mutated")
	}
	requirement := dObj(dList(next["requirements"])[0])
	next, _, err = Apply(next, "delete-requirement", Object{"id": requirement["id"]})
	if err != nil {
		t.Fatal(err)
	}
	goal := dObj(dList(next["security_goals"])[0])
	next, _, err = Apply(next, "delete-goal", Object{"id": goal["id"]})
	if err != nil {
		t.Fatalf("discard did not release dependency: %v", err)
	}
	if _, _, err = Apply(next, "discard-treatment-draft", Object{"scenario_id": scenarioID}); err != ErrNotFound {
		t.Fatal("missing draft must be not found", err)
	}
}
func TestDiscardDraftHTTPRevisionAndReplay(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	wid := httpCreate(t, h)
	ws, err := repo.Get(context.Background(), wid)
	if err != nil {
		t.Fatal(err)
	}
	ws, _, err = validateAIProposal(ws, analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := repoBody(ws)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.db.Exec("UPDATE workspaces SET body=? WHERE id=?", body, wid); err != nil {
		t.Fatal(err)
	}
	scenarioID := dStr(dObj(dList(ws["treatment_drafts"])[0])["scenario_id"])
	path := apiPrefix + "/workspaces/" + wid + "/scenarios/" + scenarioID + "/treatment-draft"
	headers := map[string]string{"If-Match": fmt.Sprintf(`"%d"`, dInt(ws["revision"])), "Idempotency-Key": "discard-draft-0001"}
	result := httpCall(h, "DELETE", path, "", headers)
	if result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	replay := httpCall(h, "DELETE", path, "", headers)
	if replay.Code != 200 || replay.Body.String() != result.Body.String() {
		t.Fatal("idempotency failed")
	}
	headers["Idempotency-Key"] = "discard-draft-0002"
	if stale := httpCall(h, "DELETE", path, "", headers); stale.Code != 412 {
		t.Fatal("stale revision accepted", stale.Code)
	}
	headers["If-Match"] = fmt.Sprintf(`"%d"`, dInt(ws["revision"])+1)
	if missing := httpCall(h, "DELETE", path, "", headers); missing.Code != 404 {
		t.Fatal("missing draft status", missing.Code)
	}
}
