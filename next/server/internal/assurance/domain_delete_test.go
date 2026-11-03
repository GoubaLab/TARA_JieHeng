package assurance

import (
	"fmt"
	"strings"
	"testing"
)

func deleteFixture(t *testing.T) Object {
	t.Helper()
	ws := NewWorkspace("删除与豁免", "")
	apply := func(action string, payload Object) Object {
		t.Helper()
		next, _, err := Apply(ws, action, payload)
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		ws = next
		return ws
	}
	apply("replace-architecture", Object{"nodes": []any{
		Object{"id": "gw", "name": "网关", "kind": "entity", "entityType": "ecu", "x": 0, "y": 0, "children": []any{Object{"id": "st", "name": "密钥", "kind": "store"}}},
		Object{"id": "ext", "name": "诊断仪", "kind": "entity", "entityType": "external", "x": 300, "y": 0},
	}, "flows": []any{}})
	apply("record-asset", Object{"name": "手工资产", "properties": []any{"integrity", "availability"}, "node_ids": []any{"gw"}})
	return ws
}
func assetByName(ws Object, name string) Object {
	for _, raw := range dList(ws["assets"]) {
		if strings.HasPrefix(dStr(dObj(raw)["name"]), name) {
			return dObj(raw)
		}
	}
	return nil
}
func reviewMessages(ws Object) string {
	parts := []string{}
	for _, raw := range dList(Review(ws)["issues"]) {
		parts = append(parts, dStr(dObj(raw)["message"]))
	}
	return strings.Join(parts, "\n")
}
func TestAssetPropertyExclusionsRemoveCoverageIssues(t *testing.T) {
	ws := deleteFixture(t)
	ext := assetByName(ws, "诊断仪")
	before := reviewMessages(ws)
	if !strings.Contains(before, "诊断仪 · 功能与服务”的 integrity") {
		t.Fatalf("expected coverage issue before exclusion:\n%s", before)
	}
	for _, raw := range dList(Review(ws)["issues"]) {
		i := dObj(raw)
		if i["reference"] == ext["id"] && i["stage"] == "threats" && i["property"] == nil {
			t.Fatal("coverage issue must carry the property for prefilled handling")
		}
	}
	payload := Object{"id": ext["id"], "name": ext["name"], "description": ext["description"], "node_ids": ext["node_ids"], "properties": ext["properties"]}
	payload["excluded_properties"] = []any{Object{"property": "integrity", "reason": "外部诊断设备不在分析边界内"}, Object{"property": "availability", "reason": "由售后流程保障"}}
	next, _, err := Apply(ws, "record-asset", payload)
	if err != nil {
		t.Fatal(err)
	}
	after := reviewMessages(next)
	if strings.Contains(after, "诊断仪 · 功能与服务”的") {
		t.Fatalf("excluded properties still reported:\n%s", after)
	}
	quality := Quality(next)
	for _, raw := range dList(quality["uncovered_assets"]) {
		if dObj(raw)["id"] == ext["id"] {
			t.Fatal("fully excluded asset should count as covered in quality view")
		}
	}
	if !strings.Contains(string(mustHTML(t, next)), "外部诊断设备不在分析边界内") {
		t.Fatal("report must disclose the exclusion reason")
	}
	payload["excluded_properties"] = []any{Object{"property": "confidentiality", "reason": "x"}}
	if _, _, err = Apply(ws, "record-asset", payload); err == nil {
		t.Fatal("exclusion of undeclared property accepted")
	}
	payload["excluded_properties"] = []any{Object{"property": "integrity", "reason": "a"}, Object{"property": "integrity", "reason": "b"}}
	if _, _, err = Apply(ws, "record-asset", payload); err == nil {
		t.Fatal("duplicate exclusion accepted")
	}
	payload["excluded_properties"] = []any{Object{"property": "integrity", "reason": ""}}
	if _, _, err = Apply(ws, "record-asset", payload); err == nil {
		t.Fatal("empty reason accepted")
	}
	// A scenario on the property blocks the exclusion.
	scenario := Object{"title": "伪造诊断仪", "asset_id": ext["id"], "property": "integrity", "damage": "损害", "threat": "威胁", "impact": Object{}, "paths": []any{}, "references": []any{}}
	withScenario, _, err := Apply(ws, "record-scenario", scenario)
	if err != nil {
		t.Fatal(err)
	}
	payload["excluded_properties"] = []any{Object{"property": "integrity", "reason": "边界外"}}
	if _, _, err = Apply(withScenario, "record-asset", payload); err == nil || !strings.Contains(err.Error(), "已有威胁场景") {
		t.Fatalf("exclusion with existing scenario accepted: %v", err)
	}
	// Omitting the field keeps the digest of untouched assets stable.
	if _, present := assetByName(ws, "手工资产")["excluded_properties"]; present {
		t.Fatal("absent exclusions must not be materialised")
	}
}
func mustHTML(t *testing.T, ws Object) []byte {
	t.Helper()
	body, err := Printable(ws, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestDeletionsRespectReferencesAndKeepNumbers(t *testing.T) {
	ws := deleteFixture(t)
	originalDigest := Digest(ws)
	manual := assetByName(ws, "手工资产")
	derived := assetByName(ws, "网关 · 功能与服务")
	if _, _, err := Apply(ws, "delete-asset", Object{"id": derived["id"]}); err == nil || !strings.Contains(err.Error(), "架构派生") {
		t.Fatalf("derived asset deletion must be refused: %v", err)
	}
	if _, _, err := Apply(ws, "delete-asset", Object{"id": "missing"}); err != ErrNotFound {
		t.Fatalf("missing id should be not found: %v", err)
	}
	damage, _, err := Apply(ws, "record-damage", Object{"description": "损害", "asset_ids": []any{manual["id"]}, "properties": []any{"integrity"}, "impact": Object{"safety": 1, "financial": 1, "operational": 1, "privacy": 0, "rationale": "依据"}})
	if err != nil {
		t.Fatal(err)
	}
	damageID := dStr(dObj(dList(damage["damage_scenarios"])[0])["id"])
	if _, _, err = Apply(damage, "delete-asset", Object{"id": manual["id"]}); err == nil || !strings.Contains(err.Error(), "损害场景") {
		t.Fatalf("asset referenced by damage must be refused: %v", err)
	}
	scenario, _, err := Apply(damage, "record-scenario", Object{"title": "威胁", "asset_id": manual["id"], "property": "integrity", "asset_ids": []any{manual["id"]}, "properties": []any{"integrity"}, "damage_ids": []any{damageID}, "damage": "损害", "threat": "威胁", "impact": Object{}, "paths": []any{}, "references": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	scenarioID := dStr(dObj(dList(scenario["scenarios"])[0])["id"])
	if _, _, err = Apply(scenario, "delete-damage", Object{"id": damageID}); err == nil || !strings.Contains(err.Error(), "威胁场景") {
		t.Fatalf("damage referenced by scenario must be refused: %v", err)
	}
	requirement, _, err := Apply(scenario, "specify-requirement", Object{"scenario_id": scenarioID, "statement": "需求", "allocation": "网关", "verification": "测试", "standard_refs": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	requirementID := dStr(dObj(dList(requirement["requirements"])[0])["id"])
	if _, _, err = Apply(requirement, "delete-scenario", Object{"id": scenarioID}); err == nil || !strings.Contains(err.Error(), "安全需求") {
		t.Fatalf("scenario referenced by requirement must be refused: %v", err)
	}
	evidence, _, err := Apply(requirement, "record-evidence", Object{"requirement_id": requirementID, "result": "pass", "reference": "R-1", "summary": "通过", "reviewer": "李工"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = Apply(evidence, "delete-requirement", Object{"id": requirementID}); err == nil || !strings.Contains(err.Error(), "验证记录") {
		t.Fatalf("requirement with evidence must be refused: %v", err)
	}
	// Without evidence the requirement can go, then the scenario (with its decision), damage and asset.
	noReq, _, err := Apply(requirement, "delete-requirement", Object{"id": requirementID})
	if err != nil {
		t.Fatal(err)
	}
	if len(dList(noReq["requirements"])) != 0 {
		t.Fatal("requirement not removed")
	}
	noReq["decisions"] = []any{Object{"scenario_id": scenarioID, "treatment": "retain"}}
	noReq["treatment_drafts"] = []any{Object{"scenario_id": scenarioID}}
	noScenario, id, err := Apply(noReq, "delete-scenario", Object{"id": scenarioID})
	if err != nil || id != scenarioID {
		t.Fatalf("scenario deletion: %v %s", err, id)
	}
	if len(dList(noScenario["scenarios"])) != 0 || len(dList(noScenario["decisions"])) != 0 || len(dList(noScenario["treatment_drafts"])) != 0 {
		t.Fatal("scenario deletion must drop its decision and draft")
	}
	noDamage, _, err := Apply(noScenario, "delete-damage", Object{"id": damageID})
	if err != nil {
		t.Fatal(err)
	}
	noAsset, _, err := Apply(noDamage, "delete-asset", Object{"id": manual["id"]})
	if err != nil {
		t.Fatal(err)
	}
	if assetByName(noAsset, "手工资产") != nil {
		t.Fatal("manual asset not removed")
	}
	codes := RecordCodes(noAsset)
	if codes("asset", manual["id"]) != "AS-004" {
		t.Fatalf("retired number lost: %s", codes("asset", manual["id"]))
	}
	replacement, _, err := Apply(noAsset, "record-asset", Object{"name": "新资产", "properties": []any{"integrity"}, "node_ids": []any{"gw"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := RecordCodes(replacement)("asset", assetByName(replacement, "新资产")["id"]); got != "AS-005" {
		t.Fatalf("deleted number must not be reused, got %s", got)
	}
	if Digest(ws) != originalDigest {
		t.Fatal("original workspace mutated by deletions")
	}
}
func TestHTTPDeleteRoutes(t *testing.T) {
	h, _, _ := httpTestServer(t)
	key := httpCreate(t, h)
	base := "/api/v1/workspaces/" + key
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "delete-flow-00001"}
	w := httpCall(h, "POST", base+"/assets", `{"name":"资产","properties":["integrity"]}`, headers)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	assetID := fmt.Sprint(httpObject(t, w)["entity_id"])
	headers["If-Match"], headers["Idempotency-Key"] = `"1"`, "delete-flow-00002"
	w = httpCall(h, "DELETE", base+"/assets/missing", "", headers)
	if w.Code != 404 {
		t.Fatalf("missing delete %d %s", w.Code, w.Body.String())
	}
	w = httpCall(h, "DELETE", base+"/assets/"+assetID, `{"id":"other"}`, headers)
	if w.Code != 422 {
		t.Fatalf("mismatched body id %d %s", w.Code, w.Body.String())
	}
	w = httpCall(h, "DELETE", base+"/assets/"+assetID, "", headers)
	if w.Code != 200 || httpObject(t, w)["entity_id"] != assetID {
		t.Fatalf("delete %d %s", w.Code, w.Body.String())
	}
	replay := httpCall(h, "DELETE", base+"/assets/"+assetID, "", headers)
	if replay.Code != 200 {
		t.Fatalf("idempotent replay %d %s", replay.Code, replay.Body.String())
	}
	if get := httpCall(h, "GET", base+"/assets/"+assetID, "", nil); get.Code != 404 {
		t.Fatal("deleted asset still readable")
	}
	events := httpObject(t, httpCall(h, "GET", base+"/events", "", nil))
	found := false
	for _, raw := range dList(events["entries"]) {
		if dObj(raw)["action"] == "delete-asset" {
			found = true
		}
	}
	if !found {
		t.Fatal("deletion must be journaled")
	}
}
