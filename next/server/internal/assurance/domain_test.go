package assurance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func dFixture(t *testing.T) Object {
	t.Helper()
	g, err := DecodeJSON([]byte(domainGoldenFixture))
	if err != nil {
		t.Fatal(err)
	}
	return g
}
func dFixed(t *testing.T) {
	t.Helper()
	oldStamp, oldIdentity := domainStamp, domainIdentity
	counter := 0
	domainStamp = func() string { return "2026-10-25T12:34:56.123456+00:00" }
	domainIdentity = func(prefix string) string { counter++; return fmt.Sprintf("%s_%016x", prefix, counter) }
	t.Cleanup(func() { domainStamp = oldStamp; domainIdentity = oldIdentity })
}
func dEqual(t *testing.T, got, want any) {
	t.Helper()
	a, err := CanonicalJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalJSON(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("Python parity mismatch\ngot: %s\nwant: %s", a, b)
	}
}
func TestDomainPythonCanonicalGolden(t *testing.T) {
	for i, raw := range dList(dFixture(t)["digests"]) {
		row := dObj(raw)
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			encoded, err := CanonicalJSON(row["value"])
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != row["canonical"] {
				t.Fatalf("encoding got %s want %s", encoded, row["canonical"])
			}
			if Digest(row["value"]) != row["digest"] {
				t.Fatal("digest mismatch")
			}
		})
	}
}
func TestDomainPythonValidationGolden(t *testing.T) {
	for i, raw := range dList(dFixture(t)["validations"]) {
		row := dObj(raw)
		t.Run(fmt.Sprintf("%d_%s", i, dStr(row["action"])), func(t *testing.T) {
			dFixed(t)
			payload := dObj(row["payload"])
			before := Digest(payload)
			value, err := ValidatePayload(dStr(row["action"]), payload)
			if dBool(row["invalid"]) {
				if err == nil {
					t.Fatalf("accepted invalid payload %#v", payload)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				dEqual(t, value, row["expected"])
			}
			if Digest(payload) != before {
				t.Fatal("validation mutated caller")
			}
		})
	}
}

// Historical Python snapshots predate generated assets and display numbering. Compare their original
// fields after projecting out this intentional extension; dedicated architecture
// tests exercise synchronization, persistence and risk dependencies end-to-end.
func legacyAssetProjection(ws Object) Object {
	result := Clone(ws)
	delete(result, "numbering_state")
	delete(result, "numbering")
	assets := []any{}
	for _, raw := range dList(result["assets"]) {
		if len(assetSource(dObj(raw))) == 0 {
			assets = append(assets, raw)
		}
	}
	result["assets"] = assets
	return result
}
func TestDomainPythonFullFlowGolden(t *testing.T) {
	dFixed(t)
	g := dFixture(t)
	ws := NewWorkspace("Golden 项目", "测试历史兼容")
	dEqual(t, ws, g["initial"])
	dEqual(t, Review(ws), g["initial_review"])
	for i, raw := range dList(g["steps"]) {
		step := dObj(raw)
		action := dStr(step["action"])
		before := Digest(ws)
		if before != step["before_digest"] {
			t.Fatalf("step %d %s previous digest mismatch", i, action)
		}
		updated, id, err := Apply(ws, action, dObj(step["payload"]))
		if err != nil {
			t.Fatalf("step %d %s: %v", i, action, err)
		}
		if Digest(ws) != before {
			t.Fatalf("step %d mutated prior state", i)
		}
		if id != step["entity_id"] {
			t.Fatalf("step %d returned %s want %s", i, id, step["entity_id"])
		}
		updated = legacyAssetProjection(updated)
		dEqual(t, updated, step["expected"])
		dEqual(t, Review(updated), step["review"])
		if Digest(updated) != step["digest"] {
			t.Fatal("workspace digest mismatch")
		}
		persisted, _ := CanonicalJSON(updated)
		ws, err = DecodeJSON(persisted)
		if err != nil {
			t.Fatal(err)
		}
		if Digest(ws) != step["digest"] {
			t.Fatal("persistence changed numeric identity")
		}
	}
}
func TestDomainLegacyMigrationGoldenAndRollback(t *testing.T) {
	dFixed(t)
	g := dFixture(t)
	legacy := dObj(g["legacy"])
	before := Digest(legacy)
	updated, _, err := Apply(legacy, "replace-architecture", dObj(g["migration_payload"]))
	if err != nil {
		t.Fatal(err)
	}
	dEqual(t, legacyAssetProjection(updated), g["migrated"])
	dEqual(t, Review(legacyAssetProjection(updated)), g["migration_review"])
	if Digest(legacy) != before {
		t.Fatal("legacy input mutated")
	}
	bad := Clone(dObj(g["migration_payload"]))
	dObj(dList(bad["flows"])[0])["target"] = "ecu"
	if _, _, err := Apply(legacy, "replace-architecture", bad); err == nil {
		t.Fatal("migration self-loop accepted")
	}
	if Digest(legacy) != before {
		t.Fatal("failed migration mutated legacy state")
	}
}
func TestDomainJSONBoundaryAndFloatRoundtrip(t *testing.T) {
	for _, raw := range []string{`{"a":1,"a":2}`, `{"a":{"owner":"x","owner":"y"}}`, `{"x":1} {}`, `[]`, `null`, `{"x":NaN}`, `{"x":1e999}`, `{"x":-1e999}`} {
		if _, err := DecodeJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	ws, err := DecodeJSON([]byte(`{"integer":1,"float":1.0,"zero":-0.0,"big":1267650600228229401496703205376}`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := CanonicalJSON(ws)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"float": 1.0`) || !strings.Contains(string(encoded), `"zero": -0.0`) {
		t.Fatal(string(encoded))
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := CanonicalJSON(v); err == nil {
			t.Fatal("non-finite encoded")
		}
	}
	if _, ok := ws["integer"].(json.Number); !ok {
		t.Fatal("numbers must preserve lexical identity")
	}
}
func TestDomainValidationLimitsAndIsolatedDefaults(t *testing.T) {
	a := NewWorkspace("a", "")
	b := NewWorkspace("b", "")
	dObj(a["item"])["owner"] = "x"
	if dObj(b["item"])["owner"] != "" {
		t.Fatal("shared defaults")
	}
	for _, payload := range []Object{{"name": strings.Repeat("😀", 201), "properties": []any{"integrity"}}, {"name": "ok", "properties": []any{}}, {"name": "ok", "properties": []any{"unknown"}}} {
		if _, err := ValidatePayload("record-asset", payload); err == nil {
			t.Fatal("invalid asset accepted")
		}
	}
	if _, err := ValidatePayload("record-asset", Object{"name": strings.Repeat("😀", 200), "properties": []any{"integrity"}}); err != nil {
		t.Fatal("Unicode length is not byte length", err)
	}
	refs := make([]any, 2001)
	for i := range refs {
		refs[i] = "reg:test:1"
	}
	if _, err := ValidatePayload("select-regulations", Object{"references": refs}); err == nil {
		t.Fatal("length limit must precede deduplication")
	}
}

func TestDomainSemanticFailuresLeaveOriginalUntouched(t *testing.T) {
	dFixed(t)
	fixture := dFixture(t)
	steps := dList(fixture["steps"])
	complete := dObj(dObj(steps[7])["expected"])
	scenario := dObj(dList(complete["scenarios"])[0])
	asset := Clone(dObj(dList(complete["assets"])[0]))
	asset["properties"] = []any{"confidentiality"}
	changedScenario := Clone(scenario)
	changedScenario["property"] = "confidentiality"
	decision := Clone(dObj(dObj(steps[5])["payload"]))
	decision["residual_feasibility"] = nil
	evidence := Clone(dObj(dObj(steps[7])["payload"]))
	evidence["id"] = dObj(dList(complete["evidence"])[0])["id"]
	cases := []struct {
		action  string
		payload Object
	}{
		{"record-asset", asset}, {"record-scenario", changedScenario}, {"decide-treatment", decision}, {"record-evidence", evidence},
		{"record-asset", Object{"name": "内部存储", "node_ids": []any{"not-an-entity"}, "properties": []any{"integrity"}}},
	}
	for _, c := range cases {
		before := Digest(complete)
		if _, _, err := Apply(complete, c.action, c.payload); err == nil {
			t.Fatalf("accepted invalid %s", c.action)
		}
		if Digest(complete) != before {
			t.Fatalf("failed %s mutated workspace", c.action)
		}
	}
	blank := NewWorkspace("未完成", "")
	if _, _, err := Apply(blank, "seal-baseline", Object{"name": "基线", "reviewer": "人", "note": "验证"}); err == nil {
		t.Fatal("sealed incomplete workspace")
	}
	draft := Clone(complete)
	dObj(dList(draft["requirements"])[0])["allocation"] = ""
	if _, _, err := Apply(draft, "record-evidence", dObj(dObj(steps[7])["payload"])); err == nil {
		t.Fatal("recorded evidence on an incomplete draft")
	}
}
