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
	if first("entity", "device") != "EN-001" || first("asset", generatedAsset(t, ws, "device")["id"]) != "AS-001" {
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
	if RecordCodes(ws)("asset", id) != "资产/0001" {
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
				if dList(raw)[0] == "资产/0001" {
					found = true
				}
			}
		}
	}
	if !found || Digest(ws) != before {
		t.Fatal("report codes missing or report mutated state")
	}
	if RecordCodes(original)("asset", id) != "AS-001" {
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
