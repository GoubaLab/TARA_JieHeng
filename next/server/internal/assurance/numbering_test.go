package assurance

import (
	"context"
	"fmt"
	"testing"
)

func TestNumberingStableAcrossFormattingReorderDeletionAndReports(t *testing.T) {
	ws := saveGraph(t, NewWorkspace("编号示例", ""), assetGraph(t))
	original := Clone(ws)
	first := RecordCodes(ws)
	if first("entity", "device") != "EN-001" || first("asset", generatedAsset(t, ws, "device")["id"]) != "EN-001" {
		t.Fatal("default numbering missing")
	}
	rules := EffectiveNumbering(nil)
	dObj(rules["prefixes"])["asset"] = "资产"
	rules["separator"] = "/"
	rules["padding"] = 4
	var err error
	ws, _, err = Apply(ws, "configure-numbering", rules)
	if err != nil {
		t.Fatal(err)
	}
	id := generatedAsset(t, ws, "device")["id"]
	if RecordCodes(ws)("asset", id) != "EN/0001" {
		t.Fatal("custom code not applied")
	}
	dEqual(t, ws["assets"], original["assets"])
	if AssessmentHash(ws, genericScenario()) != AssessmentHash(original, genericScenario()) {
		t.Fatal("format change invalidated risk")
	}
	graph := Clone(dObj(ws["architecture"]))
	nodes := dList(graph["nodes"])
	graph["nodes"] = []any{nodes[1], nodes[0]}
	ws = saveGraph(t, ws, graph)
	if RecordCodes(ws)("entity", "device") != "EN/0001" {
		t.Fatal("sorting shifted numbers")
	}
	graph["flows"] = []any{}
	graph["nodes"] = []any{nodes[1]}
	ws = saveGraph(t, ws, graph)
	newNode := Clone(dObj(nodes[0]))
	newNode["id"] = "replacement"
	newNode["children"] = []any{}
	graph["nodes"] = append(dList(graph["nodes"]), newNode)
	ws = saveGraph(t, ws, graph)
	if RecordCodes(ws)("entity", "replacement") != "EN/0003" {
		t.Fatal("deleted sequence was reused")
	}
	graph["nodes"] = append(dList(graph["nodes"]), nodes[0])
	ws = saveGraph(t, ws, graph)
	if RecordCodes(ws)("entity", "device") != "EN/0001" {
		t.Fatal("restored identity changed")
	}
	before := Digest(ws)
	tables := Tables(ws, nil, libraryCatalog(t))
	found := false
	for _, table := range tables {
		if table["name"] == "资产识别" {
			for _, raw := range dList(table["rows"]) {
				if dList(raw)[0] == "EN/0001" {
					found = true
				}
			}
		}
	}
	if !found || Digest(ws) != before {
		t.Fatal("report codes missing or report mutated state")
	}
	if RecordCodes(original)("asset", id) != "EN-001" {
		t.Fatal("old snapshot numbering changed")
	}
}
func TestNumberingPathScopeAndLegacyRead(t *testing.T) {
	ws := NewWorkspace("Legacy", "")
	ws["scenarios"] = []any{Object{"id": "场景", "paths": []any{Object{"id": "same"}}}, Object{"id": "other", "paths": []any{Object{"id": "same"}}}}
	before := Digest(ws)
	code := RecordCodes(ws)
	if pathNumberKey("场景", "same") != "6:场景same" || code("path", pathNumberKey("场景", "same")) != "AP-001" || code("path", pathNumberKey("other", "same")) != "AP-002" {
		t.Fatal("path identities collide")
	}
	if before != Digest(ws) {
		t.Fatal("read projection mutated legacy state")
	}
	rules := EffectiveNumbering(nil)
	for _, bad := range []string{"<x>", "has space", "", "-ABC", "1234567890123", "en"} {
		body := Clone(rules)
		dObj(body["prefixes"])["asset"] = bad
		if _, err := ValidatePayload("configure-numbering", body); err == nil {
			t.Fatalf("accepted invalid/duplicate prefix %q", bad)
		}
	}
	for _, padding := range []any{1, 7, 2.5, "3"} {
		body := Clone(rules)
		body["padding"] = padding
		if _, err := ValidatePayload("configure-numbering", body); err == nil {
			t.Fatal("invalid padding accepted")
		}
	}
	body := Clone(rules)
	body["numbering_state"] = Object{}
	if _, err := ValidatePayload("configure-numbering", body); err == nil {
		t.Fatal("client sequence metadata accepted")
	}
}
func TestHTTPNumberingAtomicRevisionAndIdempotency(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := "/api/v1/workspaces/" + id
	before, _ := repo.Get(context.Background(), id)
	read := httpCall(h, "GET", base+"/numbering", "", nil)
	readRules, decodeErr := DecodeJSON(read.Body.Bytes())
	if read.Code != 200 || decodeErr != nil || dObj(readRules["prefixes"])["asset"] != "AS" {
		t.Fatalf("default resource: %s", read.Body.String())
	}
	after, _ := repo.Get(context.Background(), id)
	dEqual(t, before, after)
	headers := map[string]string{"If-Match": fmt.Sprintf(`"%d"`, dInt(before["revision"])), "Idempotency-Key": "numbering-custom-test"}
	rules := EffectiveNumbering(nil)
	dObj(rules["prefixes"])["asset"] = "ZC"
	response := httpCall(h, "PUT", base+"/numbering", httpGenericBody(t, rules), headers)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	replay := httpCall(h, "PUT", base+"/numbering", httpGenericBody(t, rules), headers)
	if replay.Code != 200 || replay.Body.String() != response.Body.String() {
		t.Fatal("idempotent replay failed")
	}
	saved, _ := repo.Get(context.Background(), id)
	dEqual(t, saved["numbering"], rules)
	headers["Idempotency-Key"] = "numbering-stale-test"
	if stale := httpCall(h, "PUT", base+"/numbering", httpGenericBody(t, rules), headers); stale.Code != 412 {
		t.Fatal("stale update accepted")
	}
	headers["If-Match"] = fmt.Sprintf(`"%d"`, dInt(saved["revision"]))
	headers["Idempotency-Key"] = "numbering-invalid-test"
	rules["separator"] = ""
	if bad := httpCall(h, "PUT", base+"/numbering", httpGenericBody(t, rules), headers); bad.Code != 422 {
		t.Fatal("invalid rules accepted")
	}
	after, _ = repo.Get(context.Background(), id)
	dEqual(t, saved, after)
}

func TestAssetDisplayCodesFollowDFDAndPreserveExplicitBusinessCodes(t *testing.T) {
	ws := saveGraph(t, NewWorkspace("资产来源编号", ""), assetGraph(t))
	ws["assets"] = append(dList(ws["assets"]),
		Object{"id": "manual", "name": "人工资产", "properties": []any{"integrity"}},
		Object{"id": "imported", "name": "导入资产", "source_code": "CUSTOM-42", "properties": []any{"integrity"}},
	)
	rules := EffectiveNumbering(nil)
	for kind, prefix := range map[string]string{"entity": "ECU", "software": "APP", "store": "DATA", "flow": "TX", "asset": "ZC"} {
		dObj(rules["prefixes"])[kind] = prefix
	}
	ws["numbering"] = rules
	frozen := Clone(ws)
	before := Digest(ws)
	frozenReport := Digest(Tables(frozen, nil, libraryCatalog(t)))
	codes := RecordCodes(ws)
	for sourceID, expected := range map[string]string{"device": "ECU-001", "cloud": "ECU-002", "app": "APP-001", "store": "DATA-001", "flow": "TX-001"} {
		asset := generatedAsset(t, ws, sourceID)
		if got := codes("asset", asset["id"]); got != expected {
			t.Fatalf("source %s: got %s, want %s", sourceID, got, expected)
		}
	}
	if codes("asset", "manual") != "ZC-006" || codes("asset", "imported") != "CUSTOM-42" {
		t.Fatal("manual numbering or imported business code lost")
	}
	for _, table := range Tables(ws, nil, libraryCatalog(t)) {
		if table["name"] != "资产识别" {
			continue
		}
		rows := dList(table["rows"])
		if dList(rows[0])[0] != "ECU-001" || dList(rows[len(rows)-1])[0] != "CUSTOM-42" {
			t.Fatal("report asset codes differ from DFD/business codes")
		}
	}
	if Digest(ws) != before {
		t.Fatal("reading display codes or reports rewrote the workspace")
	}
	flow := generatedAsset(t, ws, "flow")
	flow["source_code"] = "RADIO-LINK-7"
	if RecordCodes(ws)("asset", flow["id"]) != "RADIO-LINK-7" {
		t.Fatal("generated asset ignored its explicit business code")
	}
	dObj(dObj(ws["numbering"])["prefixes"])["flow"] = "LINK"
	currentGraph := dObj(ws["architecture"])
	dObj(dList(currentGraph["flows"])[0])["name"] = "新链路名称"
	dObj(dList(dObj(dList(currentGraph["nodes"])[0])["children"])[0])["kind"] = "store"
	if RecordCodes(frozen)("asset", generatedAsset(t, frozen, "flow")["id"]) != "TX-001" ||
		RecordCodes(frozen)("asset", generatedAsset(t, frozen, "app")["id"]) != "APP-001" ||
		Digest(frozen) != before || Digest(Tables(frozen, nil, libraryCatalog(t))) != frozenReport {
		t.Fatal("later graph, business-code or rule edits changed frozen display/report identity")
	}
}

func TestAssetSourceNumbersSurviveMissingSourcesAndSoftwareStoreReclassification(t *testing.T) {
	ws := saveGraph(t, NewWorkspace("来源变更", ""), assetGraph(t))
	assetID := generatedAsset(t, ws, "app")["id"]
	graph := Clone(dObj(ws["architecture"]))
	device := dObj(dList(graph["nodes"])[0])
	dObj(dList(device["children"])[0])["kind"] = "store"
	legacy := Clone(ws)
	legacy["architecture"] = graph
	if RecordCodes(legacy)("asset", assetID) != "DS-002" {
		t.Fatal("current child kind was not used for display")
	}
	ws = saveGraph(t, ws, graph)
	if generatedAsset(t, ws, "app")["id"] != assetID || RecordCodes(ws)("asset", assetID) != "DS-002" {
		t.Fatal("software/store reclassification changed identity or display mapping")
	}
	device["children"] = []any{dList(device["children"])[1]}
	ws = saveGraph(t, ws, graph)
	if assetSource(generatedAsset(t, ws, "app"))["status"] != "missing" || RecordCodes(ws)("asset", assetID) != "DS-002" {
		t.Fatal("removed source lost its reserved DFD number")
	}
	// Reusing a removed child's ID for an entity must not link its old asset
	// to an unrelated object from another source namespace.
	dObj(ws["architecture"])["nodes"] = append(dList(dObj(ws["architecture"])["nodes"]), Object{"id": "app", "kind": "entity", "name": "新实体"})
	if RecordCodes(ws)("asset", assetID) != "DS-002" {
		t.Fatal("source lookup crossed child/entity identity namespaces")
	}
	orphan := generatedAsset(t, ws, "app")
	orphan["architecture_source"] = Object{"kind": "software", "id": "unknown-source", "status": "missing"}
	if got := RecordCodes(ws)("asset", assetID); got != "AS-002" {
		t.Fatalf("unresolved source fallback: %s", got)
	}
}

func TestExpandedDFDAssetsUseProcessAndStoreDisplayKinds(t *testing.T) {
	ws := NewWorkspace("展开 DFD 来源", "")
	ws["architecture"] = Object{"view_mode": "dfd", "nodes": []any{
		Object{"id": "process", "kind": "process_step", "name": "采样"},
		Object{"id": "memory", "kind": "data_store", "name": "参数"},
	}, "flows": []any{}}
	ws["assets"] = []any{
		Object{"id": "process-asset", "architecture_source": Object{"kind": "software", "id": "process"}},
		Object{"id": "memory-asset", "architecture_source": Object{"kind": "store", "id": "memory"}},
	}
	code := RecordCodes(ws)
	if code("asset", "process-asset") != "SW-001" || code("asset", "memory-asset") != "DS-001" {
		t.Fatal("expanded DFD source kinds differ from asset display")
	}
}
