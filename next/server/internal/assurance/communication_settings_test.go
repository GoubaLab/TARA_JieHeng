package assurance

import (
	"context"
	"strings"
	"testing"
)

func communicationPresetFixture() Object {
	return Object{"presets": []any{Object{"id": "uplink", "name": "广域遥测", "protocol": "MQTT/TLS", "communication": Object{"medium": "wireless", "technology": "5G/NR", "frequency_band": "n78", "range": "wide_area"}}}}
}
func TestCommunicationSettingsHTTPAndSnapshotIsolation(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := "/api/v1/workspaces/" + id
	empty := httpCall(h, "GET", base+"/communication-settings", "", nil)
	if empty.Code != 200 || len(dList(httpObject(t, empty)["presets"])) != 0 {
		t.Fatal(empty.Body.String())
	}
	before, _ := repo.Get(context.Background(), id)
	body := httpGenericBody(t, communicationPresetFixture())
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "save-communication-preset"}
	first := httpCall(h, "PUT", base+"/communication-settings", body, headers)
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body.String())
	}
	again := httpCall(h, "PUT", base+"/communication-settings", body, headers)
	if again.Code != 200 || first.Body.String() != again.Body.String() {
		t.Fatal("preset write is not idempotent")
	}
	stale := httpCall(h, "PUT", base+"/communication-settings", body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "stale-preset"})
	if stale.Code != 412 {
		t.Fatal("stale preset overwritten", stale.Code)
	}
	after, _ := repo.Get(context.Background(), id)
	dEqual(t, before["architecture"], after["architecture"])
	dEqual(t, before["assets"], after["assets"])
	if dInt(after["revision"]) != 1 {
		t.Fatal("wrong revision")
	}
	read := httpCall(h, "GET", base+"/communication-settings", "", nil)
	if read.Code != 200 || !strings.Contains(read.Body.String(), "n78") {
		t.Fatal(read.Body.String())
	}
	ws := reportWorkspace(t)
	original := Digest(ws)
	hash := AssessmentHash(ws, genericScenario())
	changed, _, err := Apply(ws, "configure-communication", communicationPresetFixture())
	if err != nil || Digest(ws) != original || AssessmentHash(changed, genericScenario()) != hash {
		t.Fatal("preset edit changed assessment", err)
	}
	dEqual(t, changed["scenarios"], ws["scenarios"])
}
func TestCommunicationFrequencyValidationAndReports(t *testing.T) {
	for _, v := range []any{"", "  ", nil} {
		c, err := dValidateModel("Communication", Object{"medium": "wireless", "frequency_band": v})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := c["frequency_band"]; ok {
			t.Fatal("empty frequency persisted")
		}
	}
	for _, medium := range []string{"wired", "logical", "unspecified"} {
		if _, err := dValidateModel("Communication", Object{"medium": medium, "frequency_band": "433 MHz"}); err == nil {
			t.Fatal("non-wireless frequency accepted")
		}
	}
	duplicate := communicationPresetFixture()
	duplicate["presets"] = append(dList(duplicate["presets"]), dList(duplicate["presets"])[0])
	if _, err := ValidatePayload("configure-communication", duplicate); err == nil {
		t.Fatal("duplicate ids accepted")
	}
	unsafe := communicationPresetFixture()
	dObj(dObj(dList(unsafe["presets"])[0])["communication"])["security_properties"] = []any{"integrity"}
	if _, err := ValidatePayload("configure-communication", unsafe); err == nil {
		t.Fatal("preset assigned security scope")
	}
	ws := reportWorkspace(t)
	graph := Clone(dObj(ws["architecture"]))
	flow := dObj(dList(graph["flows"])[0])
	flow["communication"] = Object{"medium": "wireless", "technology": "5G/NR", "frequency_band": " n78 ", "security_properties": []any{"integrity"}}
	saved, _, err := Apply(ws, "replace-architecture", graph)
	if err != nil {
		t.Fatal(err)
	}
	c := dObj(dObj(dList(dObj(saved["architecture"])["flows"])[0])["communication"])
	if c["frequency_band"] != "n78" {
		t.Fatal(c)
	}
	text, _ := CanonicalJSON(Tables(saved, nil, libraryCatalog(t)))
	if !strings.Contains(string(text), "频段：n78") {
		t.Fatal("frequency missing from report tables")
	}
	if automationTechnology(Object{"communication": Object{"technology": "4G/LTE"}}) != "4g" || automationTechnology(Object{"communication": Object{"technology": "5G/NR"}}) != "5g" {
		t.Fatal("cellular network aliases missing")
	}
}
