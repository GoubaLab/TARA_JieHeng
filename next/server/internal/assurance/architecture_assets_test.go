package assurance

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func assetGraph(t *testing.T) Object {
	t.Helper()
	graph := genericArchitecture(t)
	node := dObj(dList(graph["nodes"])[0])
	node["children"] = []any{Object{"id": "app", "name": "控制应用", "kind": "software", "description": "控制逻辑"}, Object{"id": "store", "name": "参数", "kind": "store", "description": "配置记录"}}
	dObj(dList(graph["flows"])[0])["communication"] = Object{"medium": "wireless", "technology": "Wi-Fi", "security_properties": []any{"authenticity", "integrity"}}
	graph, err := ValidatePayload("replace-architecture", graph)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}
func generatedAsset(t *testing.T, ws Object, sourceID string) Object {
	t.Helper()
	for _, raw := range dList(ws["assets"]) {
		a := dObj(raw)
		if assetSource(a)["id"] == sourceID {
			return a
		}
	}
	t.Fatalf("missing asset for %s", sourceID)
	return nil
}
func saveGraph(t *testing.T, ws, graph Object) Object {
	t.Helper()
	updated, _, err := Apply(ws, "replace-architecture", graph)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}
func TestArchitectureAutomaticallyCreatesDistinctAssetsAndPreservesIdentity(t *testing.T) {
	original := NewWorkspace("自动资产", "")
	before := Digest(original)
	graph := assetGraph(t)
	graph["nodes"] = append(dList(graph["nodes"]), Object{"id": "boundary", "name": "边界", "kind": "boundary", "x": 0, "y": 0, "width": 800, "height": 400})
	ws := saveGraph(t, original, graph)
	if len(dList(ws["assets"])) != 5 {
		t.Fatalf("wanted 2 entities + software + store + flow: %d", len(dList(ws["assets"])))
	}
	if Digest(original) != before {
		t.Fatal("mutated previous workspace")
	}
	flow := generatedAsset(t, ws, "flow")
	dEqual(t, flow["properties"], []any{"authenticity", "integrity"})
	dEqual(t, flow["node_ids"], []any{"device", "cloud"})
	store := generatedAsset(t, ws, "store")
	dEqual(t, store["node_ids"], []any{"device"})
	dEqual(t, store["properties"], []any{"confidentiality", "integrity", "availability"})
	again := saveGraph(t, ws, graph)
	dEqual(t, again["assets"], ws["assets"])
	graph = Clone(dObj(ws["architecture"]))
	dObj(dList(graph["nodes"])[0])["x"] = 50
	graph = dObj(dClone(graph))
	again = saveGraph(t, again, graph)
	dEqual(t, again["assets"], ws["assets"])
	scenario := genericScenario()
	scenario["asset_id"] = generatedAsset(t, ws, "device")["id"]
	if AssessmentHash(again, scenario) != AssessmentHash(ws, scenario) {
		t.Fatal("visual graph move invalidated generated-asset assessment")
	}
	dObj(dList(graph["nodes"])[0])["name"] = "重命名设备"
	renamed := saveGraph(t, again, graph)
	if generatedAsset(t, renamed, "device")["name"] != "重命名设备 · 功能与服务" {
		t.Fatal("rename not synchronized")
	}
	if generatedAsset(t, renamed, "device")["id"] != generatedAsset(t, ws, "device")["id"] {
		t.Fatal("rename changed asset ID")
	}
}
func TestArchitectureSyncPreservesManualEditsAndScenarioProperties(t *testing.T) {
	ws := saveGraph(t, NewWorkspace("手工保护", ""), assetGraph(t))
	asset := generatedAsset(t, ws, "flow")
	id := asset["id"]
	edit := Object{"id": id, "name": "自定义资产", "description": "人工说明", "node_ids": asset["node_ids"], "properties": []any{"authenticity"}, "reference": nil}
	var err error
	ws, _, err = Apply(ws, "record-asset", edit)
	if err != nil {
		t.Fatal(err)
	}
	graph := Clone(dObj(ws["architecture"]))
	flow := dObj(dList(graph["flows"])[0])
	flow["name"] = "自定义资产"
	flow["description"] = "新图说明"
	dObj(flow["communication"])["security_properties"] = []any{"confidentiality"}
	ws = saveGraph(t, ws, graph)
	a := generatedAsset(t, ws, "flow")
	for _, k := range []string{"name", "description", "properties"} {
		dEqual(t, a[k], edit[k])
	}
	if !dHas(assetSource(a)["overrides"], "properties") {
		t.Fatal("manual override not retained")
	}
	// A later source value equal to an override must not silently re-enable synchronization.
	defaults := dObj(assetSource(a)["defaults"])
	a["name"] = defaults["name"]
	flow["name"] = "再次改名"
	ws = saveGraph(t, ws, graph)
	if generatedAsset(t, ws, "flow")["name"] != defaults["name"] {
		t.Fatal("manual name overwritten after coincident defaults")
	}
	// Untouched generated properties can evolve, but existing scenarios keep their property.
	ws = saveGraph(t, NewWorkspace("已有风险", ""), assetGraph(t))
	a = generatedAsset(t, ws, "flow")
	ws["scenarios"] = []any{Object{"id": "risk", "asset_id": a["id"], "property": "authenticity"}}
	graph = Clone(dObj(ws["architecture"]))
	dObj(dObj(dList(graph["flows"])[0])["communication"])["security_properties"] = []any{"confidentiality"}
	ws = saveGraph(t, ws, graph)
	a = generatedAsset(t, ws, "flow")
	if !dHas(a["properties"], "authenticity") || !dHas(a["properties"], "confidentiality") {
		t.Fatal("lost a risk dependency or new property")
	}
	if dObj(dList(ws["scenarios"])[0])["asset_id"] != a["id"] {
		t.Fatal("risk disconnected")
	}
}
func TestArchitectureRemovalAndRestorationRetainsAssetsAndRisks(t *testing.T) {
	graph := assetGraph(t)
	ws := saveGraph(t, NewWorkspace("移除保留", ""), graph)
	originalID := generatedAsset(t, ws, "device")["id"]
	ws["scenarios"] = []any{Object{"id": "risk", "asset_id": originalID, "property": "integrity", "impact": Object{}, "paths": []any{}}}
	ws = saveGraph(t, ws, Object{"nodes": []any{}, "flows": []any{}})
	if len(dList(ws["assets"])) != 5 || len(dList(ws["scenarios"])) != 1 {
		t.Fatal("removed project records")
	}
	for _, raw := range dList(ws["assets"]) {
		a := dObj(raw)
		if assetSource(a)["status"] != "missing" || len(dList(a["node_ids"])) != 0 {
			t.Fatal("missing source not marked")
		}
	}
	issues := dList(Review(ws)["issues"])
	found := false
	for _, raw := range issues {
		if strings.Contains(dStr(dObj(raw)["message"]), "架构来源已移除") {
			found = true
		}
	}
	if !found {
		t.Fatal("review omitted missing sources")
	}
	ws = saveGraph(t, ws, graph)
	if generatedAsset(t, ws, "device")["id"] != originalID {
		t.Fatal("restore created duplicate")
	}
	dEqual(t, generatedAsset(t, ws, "device")["node_ids"], []any{"device"})
	if assetSource(generatedAsset(t, ws, "device"))["status"] != "active" {
		t.Fatal("restore not active")
	}
}
func TestArchitectureChildMoveKeepsAssetAndManualAssetsUntouched(t *testing.T) {
	ws := saveGraph(t, NewWorkspace("子对象移动", ""), assetGraph(t))
	oldID := generatedAsset(t, ws, "app")["id"]
	manual := Object{"id": "manual", "name": "手工资产", "description": "keep", "node_ids": []any{}, "properties": []any{"integrity"}, "reference": nil}
	ws["assets"] = append(dList(ws["assets"]), manual)
	graph := Clone(dObj(ws["architecture"]))
	a, b := dObj(dList(graph["nodes"])[0]), dObj(dList(graph["nodes"])[1])
	children := dList(a["children"])
	a["children"] = children[1:]
	b["children"] = children[:1]
	ws = saveGraph(t, ws, graph)
	asset := generatedAsset(t, ws, "app")
	if asset["id"] != oldID {
		t.Fatal("moving child replaced asset")
	}
	dEqual(t, asset["node_ids"], []any{"cloud"})
	saved, _ := dFind(dList(ws["assets"]), "manual")
	dEqual(t, saved, manual)
}
func TestArchitectureAssetSynchronizationIsAtomicAndIdempotentOverHTTP(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	path := "/api/v1/workspaces/" + id + "/architecture"
	graph := assetGraph(t)
	body := httpGenericBody(t, graph)
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "auto-assets-save"}
	first := httpCall(h, "PUT", path, body, headers)
	if first.Code != 200 {
		t.Fatalf("save %d %s", first.Code, first.Body.String())
	}
	result := httpObject(t, first)
	dEqual(t, dObj(result["asset_sync"])["created"], 5)
	replay := httpCall(h, "PUT", path, body, headers)
	if replay.Code != 200 || replay.Body.String() != first.Body.String() {
		t.Fatal("receipt replay changed")
	}
	ws, _ := repo.Get(context.Background(), id)
	if len(dList(ws["assets"])) != 5 || dInt(ws["revision"]) != 1 {
		t.Fatal("duplicate save")
	}
	digest := Digest(ws)
	invalid := Clone(graph)
	dObj(dList(invalid["flows"])[0])["target"] = "missing"
	bad := httpCall(h, "PUT", path, httpGenericBody(t, invalid), map[string]string{"If-Match": `"1"`, "Idempotency-Key": "auto-assets-invalid"})
	if bad.Code != 422 {
		t.Fatal(bad.Code)
	}
	unchanged, _ := repo.Get(context.Background(), id)
	if Digest(unchanged) != digest {
		t.Fatal("partial graph/asset mutation")
	}
	stale := httpCall(h, "PUT", path, body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "auto-assets-stale"})
	if stale.Code != 412 {
		t.Fatal(stale.Code)
	}
	second := httpCall(h, "PUT", path, body, map[string]string{"If-Match": `"1"`, "Idempotency-Key": "auto-assets-again"})
	if second.Code != 200 {
		t.Fatal(second.Code)
	}
	dEqual(t, httpObject(t, second)["asset_sync"], Object{"created": 0, "updated": 0, "missing": 0, "total": 5})
	// API clients cannot forge origin metadata to turn a manual asset into a generated one.
	spoof := httpCall(h, "POST", "/api/v1/workspaces/"+id+"/assets", `{"name":"spoof","properties":["integrity"],"architecture_source":{"kind":"flow","id":"flow"}}`, map[string]string{"If-Match": `"2"`, "Idempotency-Key": "auto-assets-spoof"})
	if spoof.Code != 422 {
		t.Fatal(fmt.Sprintf("metadata write accepted: %d", spoof.Code))
	}
}
