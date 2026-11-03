package assurance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDeleteAbsentOptionalCollectionsReturnsNotFoundWithoutMutation(t *testing.T) {
	for _, collection := range []string{"damage-scenarios", "security-goals"} {
		t.Run(collection, func(t *testing.T) {
			h, repo, _ := httpTestServer(t)
			id := httpCreate(t, h)
			before, err := repo.Get(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			response := httpCall(h, "DELETE", "/api/v1/workspaces/"+id+"/"+collection+"/missing", "", map[string]string{"If-Match": `"0"`, "Idempotency-Key": "delete-optional-0001"})
			if response.Code != 404 {
				t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
			}
			after, err := repo.Get(context.Background(), id)
			if err != nil || Digest(before) != Digest(after) {
				t.Fatalf("missing deletion changed workspace: %v", err)
			}
			_, count, err := repo.Events(context.Background(), id, 0, 100)
			if err != nil || count != 1 {
				t.Fatalf("missing deletion changed journal: %d %v", count, err)
			}
			checkpoints, err := repo.checkpoints(context.Background(), id)
			if err != nil || len(checkpoints) != 0 {
				t.Fatalf("missing deletion created recovery point: %v", err)
			}
			var receipts int
			if err := repo.db.QueryRow("SELECT count(*) FROM receipts WHERE workspace_id=? AND request_id=?", id, "delete-optional-0001").Scan(&receipts); err != nil || receipts != 0 {
				t.Fatalf("missing deletion saved a receipt: %d %v", receipts, err)
			}
		})
	}
}

func TestDeleteDerivedAssetRespectsCurrentGenerationMode(t *testing.T) {
	original := deleteFixture(t)
	asset := assetByName(original, "网关 · 功能与服务")
	before := Digest(original)
	if _, _, err := Apply(original, "delete-asset", Object{"id": asset["id"]}); err == nil {
		t.Fatal("automatic active source should protect asset")
	}
	graph := Clone(dObj(original["architecture"]))
	graph["asset_generation"] = "catalog"
	catalog, _, err := Apply(original, "replace-architecture", graph)
	if err != nil {
		t.Fatal(err)
	}
	deleted, _, err := Apply(catalog, "delete-asset", Object{"id": asset["id"]})
	if err != nil {
		t.Fatalf("catalog must allow unreferenced derived asset deletion: %v", err)
	}
	if _, err := dFind(dList(deleted["assets"]), asset["id"]); err == nil {
		t.Fatal("asset still present")
	}
	if len(dList(deleted["assets"])) != len(dList(catalog["assets"]))-1 {
		t.Fatal("other assets were removed")
	}
	saved, _, err := Apply(deleted, "replace-architecture", graph)
	if err != nil || len(dList(saved["assets"])) != len(dList(deleted["assets"])) {
		t.Fatalf("catalog save recreated deleted asset: %v", err)
	}
	if Digest(original) != before {
		t.Fatal("original workspace changed")
	}
}

func TestDeleteGoalRejectsAdoptedDraftAndDecisionReferences(t *testing.T) {
	ws, _, err := validateAIProposal(NewWorkspace("Draft dependency", ""), analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	requirement := dObj(dList(ws["requirements"])[0])
	ws, _, err = Apply(ws, "delete-requirement", Object{"id": requirement["id"]})
	if err != nil {
		t.Fatal(err)
	}
	goalID := dObj(dList(ws["security_goals"])[0])["id"]
	scenarioID := dObj(dList(ws["scenarios"])[0])["id"]
	before := Digest(ws)
	_, _, err = Apply(ws, "delete-goal", Object{"id": goalID})
	if err == nil || !strings.Contains(err.Error(), "处置草案") || !strings.Contains(err.Error(), "RS-001") || !strings.Contains(err.Error(), "合成篡改") || strings.Contains(err.Error(), dStr(scenarioID)) {
		t.Fatalf("draft reference must block deletion with a readable reference: %v", err)
	}
	if Digest(ws) != before {
		t.Fatal("failed deletion mutated working copy")
	}
	ws, _, err = Apply(ws, "adopt-method", Clone(DefaultMethod))
	if err != nil {
		t.Fatal(err)
	}
	draft := Clone(dObj(dList(ws["treatment_drafts"])[0]))
	for _, key := range []string{"draft", "created_at", "assessment_hash"} {
		delete(draft, key)
	}
	ws, _, err = Apply(ws, "decide-treatment", draft)
	if err != nil {
		t.Fatalf("rejected deletion must leave draft usable: %v", err)
	}
	_, _, err = Apply(ws, "delete-goal", Object{"id": goalID})
	if err == nil || !strings.Contains(err.Error(), "处置决策") || !strings.Contains(err.Error(), "RS-001") || strings.Contains(err.Error(), dStr(scenarioID)) {
		t.Fatalf("formal decision must still protect goal: %v", err)
	}
}

func TestDeleteDependencyLabelsUseProjectOrSourceNumbers(t *testing.T) {
	ws, _, err := validateAIProposal(NewWorkspace("Dependency labels", ""), analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	asset := dObj(dList(ws["assets"])[0])
	scenario := dObj(dList(ws["scenarios"])[0])
	requirement := dObj(dList(ws["requirements"])[0])
	for _, test := range []struct {
		action, expected string
		id               any
	}{
		{"delete-asset", "RS-001", asset["id"]},
		{"delete-scenario", "SR-001", scenario["id"]},
	} {
		_, _, err := Apply(ws, test.action, Object{"id": test.id})
		if err == nil || !strings.Contains(err.Error(), test.expected) {
			t.Fatalf("%s: %v", test.action, err)
		}
		for _, opaque := range []any{scenario["id"], requirement["id"]} {
			if strings.Contains(err.Error(), dStr(opaque)) {
				t.Fatalf("opaque id in dependency message: %v", err)
			}
		}
	}
	scenario["source_code"] = "TS-CUSTOM-21"
	_, _, err = Apply(ws, "delete-asset", Object{"id": asset["id"]})
	if err == nil || !strings.Contains(err.Error(), "TS-CUSTOM-21") {
		t.Fatalf("source code was not preserved: %v", err)
	}
}

func TestDeleteScenarioRemovesOnlyItsOwnTreatmentRecords(t *testing.T) {
	ws := NewWorkspace("Treatment cleanup", "")
	var err error
	var assetID string
	ws, assetID, err = Apply(ws, "record-asset", Object{"name": "command", "properties": []any{"integrity"}})
	if err != nil {
		t.Fatal(err)
	}
	scenarioIDs := []string{}
	for i := 0; i < 2; i++ {
		var id string
		ws, id, err = Apply(ws, "record-scenario", Object{"title": fmt.Sprintf("scenario %d", i), "asset_id": assetID, "property": "integrity", "damage": "damage", "threat": "threat", "paths": []any{}})
		if err != nil {
			t.Fatal(err)
		}
		scenarioIDs = append(scenarioIDs, id)
	}
	for _, collection := range []string{"decisions", "treatment_drafts"} {
		ws[collection] = []any{Object{"scenario_id": scenarioIDs[0], "treatment": "retain"}, Object{"scenario_id": scenarioIDs[1], "treatment": "retain"}}
	}
	before := Digest(ws)
	next, _, err := Apply(ws, "delete-scenario", Object{"id": scenarioIDs[0]})
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range []string{"decisions", "treatment_drafts"} {
		kept := dList(next[collection])
		if len(kept) != 1 || dObj(kept[0])["scenario_id"] != scenarioIDs[1] {
			t.Fatalf("incorrect %s cleanup", collection)
		}
	}
	if Digest(ws) != before {
		t.Fatal("scenario deletion mutated original")
	}
}

func TestRepositoryDeleteReplayRevisionAndFrozenReport(t *testing.T) {
	repo, _ := repositoryTest(t)
	ctx := context.Background()
	fixture := dFixture(t)
	ws := Clone(dObj(fixture["initial"]))
	var seal Object
	for _, raw := range dList(fixture["steps"]) {
		step := dObj(raw)
		if step["action"] == "seal-baseline" {
			seal = Clone(dObj(step["payload"]))
			break
		}
		ws = Clone(dObj(step["expected"]))
	}
	if seal == nil {
		t.Fatal("fixture must contain a reviewed baseline")
	}
	body, err := repoBody(ws)
	if err != nil {
		t.Fatal(err)
	}
	id := dStr(ws["id"])
	if _, err = repo.db.Exec("INSERT INTO workspaces(id,body) VALUES(?,?)", id, body); err != nil {
		t.Fatal(err)
	}
	revision := int64(dInt(ws["revision"]))
	result, _, err := repo.Change(ctx, id, revision, "delete-freeze-0001", "seal-baseline", seal, nil)
	if err != nil {
		t.Fatal(err)
	}
	baselineID := dStr(result["entity_id"])
	frozen, err := repo.Baseline(ctx, id, baselineID)
	if err != nil {
		t.Fatal(err)
	}
	reportBefore := string(mustHTML(t, frozen))
	result, _, err = repo.Change(ctx, id, revision+1, "delete-create-0001", "record-asset", Object{"name": "Temporary assessment asset", "properties": []any{"integrity"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assetID := result["entity_id"]
	before, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	retiredCode := RecordCodes(before)("asset", assetID)
	_, _, err = repo.Change(ctx, id, revision+1, "delete-stale-0001", "delete-asset", Object{"id": assetID}, nil)
	var revisionErr *RevisionError
	if !errors.As(err, &revisionErr) {
		t.Fatalf("stale deletion was not rejected: %v", err)
	}
	after, _ := repo.Get(ctx, id)
	if Digest(before) != Digest(after) {
		t.Fatal("stale deletion changed data")
	}
	checkpoints, err := repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 0 {
		t.Fatalf("stale deletion created recovery point: %v", err)
	}
	removed, replay, err := repo.Change(ctx, id, revision+2, "delete-asset-0001", "delete-asset", Object{"id": assetID}, nil)
	if err != nil || replay {
		t.Fatalf("delete failed: %v", err)
	}
	repeated, replay, err := repo.Change(ctx, id, revision+2, "delete-asset-0001", "delete-asset", Object{"id": assetID}, nil)
	if err != nil || !replay || Digest(removed) != Digest(repeated) {
		t.Fatalf("delete replay failed: %v", err)
	}
	checkpoints, err = repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 1 || checkpoints[0]["id"] != removed["checkpoint_id"] {
		t.Fatalf("replay duplicated or lost recovery point: %v %v", checkpoints, err)
	}
	checkpoint := checkpoints[0]
	if checkpoint["actor"] != "本地管理员" || !strings.Contains(dStr(checkpoint["name"]), "删除资产前") || !strings.Contains(dStr(checkpoint["name"]), retiredCode) || !strings.Contains(dStr(checkpoint["name"]), "Temporary assessment asset") {
		t.Fatalf("unreadable or unaudited checkpoint: %v", checkpoint)
	}
	stored, err := readObject(repo.db.QueryRow("SELECT body FROM workspace_checkpoints WHERE id=?", checkpoint["id"]))
	if err != nil || Digest(stored["workspace"]) != Digest(before) {
		t.Fatalf("recovery point did not capture the exact pre-deletion state: %v", err)
	}
	after, _ = repo.Get(ctx, id)
	if RecordCodes(after)("asset", assetID) != retiredCode {
		t.Fatal("retired number was removed")
	}
	var eventCount, receiptCount int
	if err = repo.db.QueryRow("SELECT count(*) FROM journal WHERE workspace_id=? AND action='delete-asset'", id).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err = repo.db.QueryRow("SELECT count(*) FROM receipts WHERE workspace_id=? AND request_id='delete-asset-0001'", id).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 || receiptCount != 1 {
		t.Fatalf("delete duplicated history: events=%d receipts=%d", eventCount, receiptCount)
	}
	result, _, err = repo.Change(ctx, id, revision+3, "delete-replace-0001", "record-asset", Object{"name": "Replacement assessment asset", "properties": []any{"integrity"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	after, _ = repo.Get(ctx, id)
	if RecordCodes(after)("asset", result["entity_id"]) == retiredCode {
		t.Fatal("retired number was reused")
	}
	if _, _, err = repo.restoreCheckpoint(ctx, id, dStr(removed["checkpoint_id"]), "delete-restore-0001", "恢复误删工作产品", "本地管理员", int64(dInt(after["revision"]))); err != nil {
		t.Fatal(err)
	}
	restored, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range []string{"assets", "scenarios", "requirements", "evidence", "decisions", "damage_scenarios", "security_goals"} {
		if Digest(restored[collection]) != Digest(before[collection]) {
			t.Fatalf("restoration did not recover %s", collection)
		}
	}
	stillFrozen, err := repo.Baseline(ctx, id, baselineID)
	if err != nil || Digest(stillFrozen) != Digest(frozen) || string(mustHTML(t, stillFrozen)) != reportBefore {
		t.Fatalf("deletion altered frozen baseline or report: %v", err)
	}
}

func TestRejectedAndRolledBackDeletesDoNotCreateRecoveryPoints(t *testing.T) {
	repo, _ := repositoryTest(t)
	ws, _, err := validateAIProposal(NewWorkspace("Atomic recovery", ""), analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := repoBody(ws)
	if err != nil {
		t.Fatal(err)
	}
	id := dStr(ws["id"])
	if _, err = repo.db.Exec("INSERT INTO workspaces(id,body) VALUES(?,?)", id, body); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), principalContextKey{}, Object{"id": "synthetic-reviewer", "role": "reviewer"})
	goalID := dObj(dList(ws["security_goals"])[0])["id"]
	requirementID := dObj(dList(ws["requirements"])[0])["id"]
	revision := int64(dInt(ws["revision"]))
	if _, _, err = repo.Change(ctx, id, revision, "protected-delete-001", "delete-goal", Object{"id": goalID}, nil); err == nil {
		t.Fatal("referenced goal deletion succeeded")
	}
	checkpoints, err := repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 0 {
		t.Fatalf("rejected deletion created recovery point: %v", err)
	}
	if _, err = repo.db.Exec("CREATE TRIGGER reject_test_workspace_update BEFORE UPDATE ON workspaces BEGIN SELECT RAISE(ABORT,'synthetic write failure'); END"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = repo.Change(ctx, id, revision, "rollback-delete-001", "delete-requirement", Object{"id": requirementID}, nil); err == nil {
		t.Fatal("injected write failure was not propagated")
	}
	checkpoints, err = repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 0 {
		t.Fatalf("rolled back deletion left recovery point: %v", err)
	}
	after, err := repo.Get(ctx, id)
	if err != nil || Digest(after) != Digest(ws) {
		t.Fatalf("failed deletion changed workspace: %v", err)
	}
	if _, err = repo.db.Exec("DROP TRIGGER reject_test_workspace_update"); err != nil {
		t.Fatal(err)
	}
	result, _, err := repo.Change(ctx, id, revision, "rollback-delete-001", "delete-requirement", Object{"id": requirementID}, nil)
	if err != nil {
		t.Fatalf("failed request reserved idempotency key: %v", err)
	}
	checkpoints, err = repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 1 || checkpoints[0]["id"] != result["checkpoint_id"] || checkpoints[0]["actor"] != "synthetic-reviewer" {
		t.Fatalf("principal not recorded in successful recovery point: %v %v", checkpoints, err)
	}
}

func TestDiscardTreatmentDraftHasOneRestorableRecoveryPoint(t *testing.T) {
	repo, _ := repositoryTest(t)
	ws, _, err := validateAIProposal(NewWorkspace("Draft recovery", ""), analysisFixture("synthetic"), "analysis", "tester", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := repoBody(ws)
	if err != nil {
		t.Fatal(err)
	}
	id := dStr(ws["id"])
	if _, err = repo.db.Exec("INSERT INTO workspaces(id,body) VALUES(?,?)", id, body); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	scenarioID := dObj(dList(ws["scenarios"])[0])["id"]
	revision := int64(dInt(ws["revision"]))
	result, _, err := repo.Change(ctx, id, revision, "discard-recovery-001", "discard-treatment-draft", Object{"scenario_id": scenarioID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	again, replay, err := repo.Change(ctx, id, revision, "discard-recovery-001", "discard-treatment-draft", Object{"scenario_id": scenarioID}, nil)
	if err != nil || !replay || Digest(again) != Digest(result) {
		t.Fatalf("draft discard replay failed: %v", err)
	}
	checkpoints, err := repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 1 || checkpoints[0]["id"] != result["checkpoint_id"] || !strings.Contains(dStr(checkpoints[0]["name"]), "放弃处置草案前 · RS-001") {
		t.Fatalf("draft recovery point missing or duplicated: %v %v", checkpoints, err)
	}
	after, err := repo.Get(ctx, id)
	if err != nil || len(dList(after["treatment_drafts"])) != 0 {
		t.Fatalf("draft was not removed: %v", err)
	}
	if _, _, err = repo.Change(ctx, id, revision+1, "discard-recovery-002", "discard-treatment-draft", Object{"scenario_id": scenarioID}, nil); err != ErrNotFound {
		t.Fatalf("missing draft discard must return not found: %v", err)
	}
	checkpoints, err = repo.checkpoints(ctx, id)
	if err != nil || len(checkpoints) != 1 {
		t.Fatalf("missing draft created another recovery point: %v", err)
	}
	if _, _, err = repo.restoreCheckpoint(ctx, id, dStr(result["checkpoint_id"]), "discard-restore-001", "恢复处置草案", "本地管理员", revision+1); err != nil {
		t.Fatal(err)
	}
	restored, err := repo.Get(ctx, id)
	if err != nil || Digest(restored["treatment_drafts"]) != Digest(ws["treatment_drafts"]) || Digest(restored["security_goals"]) != Digest(ws["security_goals"]) {
		t.Fatalf("draft and goal relations not restored: %v", err)
	}
}
