package assurance

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestAutomationHTTPAtomicReviewReplayAndRestore(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	created := repoTestCreate(t, repo)
	_, _, e := repo.Change(context.Background(), dStr(created["id"]), 0, "synthetic-diagram", "replace-architecture", dObj(automationFixture(t)["architecture"]), nil)
	if e != nil {
		t.Fatal(e)
	}
	ws, e := repo.Get(context.Background(), dStr(created["id"]))
	if e != nil {
		t.Fatal(e)
	}
	wid := dStr(ws["id"])
	path := "/api/v1/workspaces/" + wid
	planResponse := httpCall(h, "GET", path+"/automation-plan", "", nil)
	if planResponse.Code != 200 {
		t.Fatal(planResponse.Body.String())
	}
	plan := httpObject(t, planResponse)
	ids := automationSelectableIDs(plan)
	if len(ids) == 0 {
		t.Fatal("no candidates")
	}
	payload := Object{"plan_hash": plan["plan_hash"], "candidate_ids": []any{ids[0]}, "owner": "Synthetic reviewer", "reviewed": false}
	headers := map[string]string{"If-Match": fmt.Sprintf(`"%d"`, dInt(ws["revision"])), "Idempotency-Key": "synthetic-adoption"}
	body, _ := repoBody(payload)
	response := httpCall(h, "POST", path+"/automation-adoptions", body, headers)
	if response.Code != 422 {
		t.Fatal("unreviewed draft accepted", response.Code, response.Body.String())
	}
	payload["reviewed"] = true
	payload["plan_hash"] = strings.Repeat("a", 64)
	body, _ = repoBody(payload)
	response = httpCall(h, "POST", path+"/automation-adoptions", body, headers)
	if response.Code != 409 {
		t.Fatal("stale plan accepted", response.Code, response.Body.String())
	}
	cps, _ := repo.checkpoints(context.Background(), wid)
	if len(cps) != 0 {
		t.Fatal("failed adoption mutated checkpoints")
	}
	payload["plan_hash"] = plan["plan_hash"]
	body, _ = repoBody(payload)
	response = httpCall(h, "POST", path+"/automation-adoptions", body, headers)
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	result := httpObject(t, response)
	replay := httpCall(h, "POST", path+"/automation-adoptions", body, headers)
	if replay.Code != 201 || Digest(httpObject(t, replay)) != Digest(result) {
		t.Fatal("non-idempotent adoption")
	}
	next, _ := repo.Get(context.Background(), wid)
	if dInt(next["revision"]) != dInt(ws["revision"])+1 || len(dList(next["scenarios"])) != 1 || len(dList(next["automation_adoptions"])) != 1 || len(dList(next["ai_adoptions"])) != 0 {
		t.Fatal("not atomic or mislabeled")
	}
	found := false
	for _, table := range Tables(next, nil, nil) {
		if table["name"] == "自动化分析追溯" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing report provenance")
	}
	cps, _ = repo.checkpoints(context.Background(), wid)
	if len(cps) != 1 {
		t.Fatal("duplicate checkpoint")
	}
	headers["Idempotency-Key"] = "synthetic-stale-adoption"
	response = httpCall(h, "POST", path+"/automation-adoptions", body, headers)
	if response.Code != 412 {
		t.Fatal("stale revision accepted")
	}
	_, _, e = repo.restoreCheckpoint(context.Background(), wid, dStr(result["checkpoint_id"]), "synthetic-restore", "Undo synthetic adoption", "test", int64(dInt(next["revision"])))
	if e != nil {
		t.Fatal(e)
	}
	restored, _ := repo.Get(context.Background(), wid)
	if len(dList(restored["scenarios"])) != 0 || len(dList(restored["automation_adoptions"])) != 0 {
		t.Fatal("restore left generated records")
	}
}
