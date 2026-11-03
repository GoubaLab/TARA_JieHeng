package assurance

import (
	"fmt"
	"testing"
)

func TestPropertyExclusionsPreserveOmissionAndRequireExplicitClearing(t *testing.T) {
	h, _, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := apiPrefix + "/workspaces/" + id
	write := func(method, path, body string, rev int, key string) Object {
		t.Helper()
		w := httpCall(h, method, base+path, body, map[string]string{"If-Match": fmt.Sprintf(`"%d"`, rev), "Idempotency-Key": key})
		if w.Code != 200 && w.Code != 201 {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return httpObject(t, w)
	}
	created := write("POST", "/assets", `{"name":"command","properties":["integrity"],"excluded_properties":[{"property":"integrity","reason":"outside scope"}]}`, 0, "exclusion-create")
	assetID := dStr(created["entity_id"])
	write("PUT", "/assets/"+assetID, `{"name":"command updated","properties":["integrity"]}`, 1, "exclusion-preserve")
	saved := httpObject(t, httpCall(h, "GET", base+"/assets/"+assetID, "", nil))
	if !assetPropertyExcluded(saved, "integrity") || dStr(dObj(dList(saved["excluded_properties"])[0])["reason"]) != "outside scope" {
		t.Fatalf("omitted field lost exclusions: %v", saved)
	}
	write("PUT", "/assets/"+assetID, `{"name":"command updated","properties":["integrity"],"excluded_properties":[]}`, 2, "exclusion-clear")
	saved = httpObject(t, httpCall(h, "GET", base+"/assets/"+assetID, "", nil))
	if len(dList(saved["excluded_properties"])) != 0 {
		t.Fatal("explicit clearing failed")
	}
	forbidden := httpCall(h, "PATCH", base+"/assets/"+assetID, `{}`, nil)
	if forbidden.Code != 405 || forbidden.Header().Get("Allow") != "GET, HEAD, PUT, DELETE" {
		t.Fatalf("Allow header mismatch: %d %s", forbidden.Code, forbidden.Header().Get("Allow"))
	}
}

func TestPropertyExclusionsCannotBeBypassedThroughThreatWrites(t *testing.T) {
	ws := NewWorkspace("scope consistency", "")
	var assetID, otherID string
	var err error
	ws, assetID, err = Apply(ws, "record-asset", Object{"name": "command", "properties": []any{"integrity", "availability"}, "excluded_properties": []any{Object{"property": "integrity", "reason": "outside scope"}}})
	if err != nil {
		t.Fatal(err)
	}
	ws, otherID, err = Apply(ws, "record-asset", Object{"name": "other", "properties": []any{"integrity", "availability"}})
	if err != nil {
		t.Fatal(err)
	}
	before := Digest(ws)
	threat := Object{"title": "forged command", "asset_id": assetID, "property": "integrity", "damage": "unintended output", "threat": "forged command accepted"}
	if _, _, err = Apply(ws, "record-scenario", threat); err == nil {
		t.Fatal("excluded property accepted in new threat")
	}
	if Digest(ws) != before {
		t.Fatal("rejected creation changed workspace")
	}
	threat["property"] = "availability"
	var scenarioID string
	ws, scenarioID, err = Apply(ws, "record-scenario", threat)
	if err != nil {
		t.Fatalf("nonexcluded property rejected: %v", err)
	}
	changed := Clone(dObj(dList(ws["scenarios"])[0]))
	changed["id"], changed["property"] = scenarioID, "integrity"
	if _, _, err = Apply(ws, "record-scenario", changed); err == nil {
		t.Fatal("excluded property accepted in edited threat")
	}
	changed["asset_id"] = otherID
	changed["asset_ids"] = []any{otherID, assetID}
	changed["properties"] = []any{"integrity"}
	if _, _, err = Apply(ws, "record-scenario", changed); err == nil {
		t.Fatal("excluded secondary asset bypassed scope")
	}
}
