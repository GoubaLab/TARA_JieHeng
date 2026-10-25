package assurance

import (
	"errors"
	"testing"
)

func TestDomainWithdrawnReferenceSelection(t *testing.T) {
	ws := NewWorkspace("原创测试", "")
	before := Digest(ws)
	if _, _, err := Apply(ws, "apply-reference-selection", Object{}); !errors.Is(err, ErrReferenceDataUnavailable) {
		t.Fatalf("withdrawn source must be unavailable: %v", err)
	}
	if Digest(ws) != before {
		t.Fatal("unavailable selection changed project")
	}
}

func genericScenario() Object {
	return Object{"id": "scenario_local", "asset_id": "asset_local", "impact": Object{"safety": 0, "financial": 1, "operational": 2, "privacy": 3, "rationale": "原创测试依据"}, "paths": []any{Object{"id": "path_local", "feasibility": 3, "steps": "测试步骤", "rationale": "测试依据"}}}
}

func TestDomainConfigurableRiskAndResidual(t *testing.T) {
	scenario := genericScenario()
	if got := Risk(scenario, DefaultMethod); got["impact"] != 3 || got["level"] != 5 {
		t.Fatalf("legacy risk changed: %v", got)
	}
	method := Clone(DefaultMethod)
	method["impact_aggregation"] = "weighted_mean"
	method["risk_calculation"] = "product"
	method["impact_weights"] = Object{"safety": 0, "financial": 25, "operational": 75, "privacy": 0}
	dObj(scenario["impact"])["safety"] = nil
	dObj(scenario["impact"])["privacy"] = nil
	if got := Risk(scenario, method); got["impact"] != 2 || got["level"] != 3 {
		t.Fatalf("weighted ceil/product incorrect: %v", got)
	}
	dObj(scenario["impact"])["operational"] = nil
	if Risk(scenario, method)["level"] != nil {
		t.Fatal("missing active dimension received a score")
	}
	dObj(scenario["impact"])["operational"] = 2
	ws := NewWorkspace("原创通用场景", "")
	ws["method"], ws["method_confirmed"], ws["scenarios"] = method, true, []any{scenario}
	updated, _, err := Apply(ws, "decide-treatment", Object{"scenario_id": "scenario_local", "treatment": "retain", "rationale": "测试处置", "owner": "测试人员", "claim": "测试声明", "residual_impact": 2, "residual_feasibility": 3, "residual_rationale": "独立测试依据"})
	if err != nil {
		t.Fatal(err)
	}
	evaluation := dObj(dList(Review(updated)["evaluations"])[0])
	if evaluation["residual"] != 3 || evaluation["decision_stale"] != false {
		t.Fatalf("residual must use product algorithm: %v", evaluation)
	}
	// Thresholds are inclusive upper bounds, including exact boundary values.
	effective := EffectiveMethod(Object{"risk_calculation": "product"})
	for _, test := range []struct{ impact, feasibility, want int }{{0, 3, 1}, {0, 4, 2}, {1, 3, 2}, {2, 3, 3}, {2, 4, 4}, {3, 4, 5}} {
		if got := calculatedRisk(test.impact, test.feasibility, effective); got != test.want {
			t.Fatalf("product(%d,%d)=%d", test.impact, test.feasibility, got)
		}
	}
}

func TestDomainMethodConfigurationValidation(t *testing.T) {
	for _, extra := range []Object{
		{"impact_aggregation": "weighted_mean", "impact_weights": Object{"safety": 0, "financial": 0, "operational": 0, "privacy": 0}},
		{"impact_weights": Object{"privacy": -1}}, {"impact_weights": Object{"privacy": 101}}, {"impact_weights": Object{"privacy": 1.5}},
		{"product_thresholds": []any{1, 2, 2, 4}}, {"product_thresholds": []any{0, 2, 3, 4}}, {"product_thresholds": []any{1, 2, 3, 16}},
		{"product_thresholds": []any{1, 2, 3}}, {"risk_calculation": "eval"}, {"impact_aggregation": "sum"},
		{"impact_levels": []any{}}, {"impact_dimensions": Object{"safety": " "}}, {"formula": "execute()"},
	} {
		payload := Clone(DefaultMethod)
		for key, value := range extra {
			payload[key] = value
		}
		if _, err := ValidatePayload("adopt-method", payload); err == nil {
			t.Fatalf("accepted invalid configuration: %v", extra)
		}
	}
	method := EffectiveMethod(DefaultMethod)
	before := Digest(DefaultMethod)
	validated, err := ValidatePayload("adopt-method", method)
	if err != nil {
		t.Fatal(err)
	}
	if dObj(validated["impact_weights"])["safety"] != 25 {
		t.Fatal("unexpected default weight")
	}
	dObj(method["impact_weights"])["safety"] = 99
	if Digest(DefaultMethod) != before || dObj(EffectiveMethod(DefaultMethod)["impact_weights"])["safety"] != 25 {
		t.Fatal("effective method changed shared defaults")
	}
}

func genericArchitecture(t *testing.T) Object {
	t.Helper()
	value, err := ValidatePayload("replace-architecture", Object{"nodes": []any{Object{"id": "device", "name": "测试设备", "x": 0, "y": 0}, Object{"id": "cloud", "name": "测试服务", "x": 300, "y": 0}}, "flows": []any{Object{"id": "flow", "name": "测试消息", "source": "device", "target": "cloud", "protocol": "自定义协议"}}})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestDomainGenericMetadataAndHashCompatibility(t *testing.T) {
	ws := NewWorkspace("通用测试", "")
	ws["architecture"] = genericArchitecture(t)
	scenario := genericScenario()
	base := AssessmentHash(ws, scenario)
	defaults := Clone(ws)
	dObj(defaults["item"])["domain"] = "automotive"
	defaults["method"] = EffectiveMethod(dObj(defaults["method"]))
	architecture := dObj(defaults["architecture"])
	dObj(dList(architecture["nodes"])[0])["operating_system"] = Object{}
	dObj(dList(architecture["flows"])[0])["communication"] = Object{}
	validated, err := ValidatePayload("replace-architecture", architecture)
	if err != nil {
		t.Fatal(err)
	}
	defaults["architecture"] = validated
	if AssessmentHash(defaults, scenario) != base {
		t.Fatal("new empty defaults invalidated legacy decision")
	}
	for _, mutate := range []func(Object){
		func(w Object) { dObj(w["item"])["domain"] = "robotics" },
		func(w Object) { dObj(dList(dObj(w["architecture"])["nodes"])[0])["entityType"] = "device" },
		func(w Object) {
			dObj(dList(dObj(w["architecture"])["nodes"])[0])["operating_system"] = Object{"family": "linux"}
		},
		func(w Object) {
			dObj(dList(dObj(w["architecture"])["flows"])[0])["communication"] = Object{"purpose": "状态同步"}
		},
		func(w Object) { dObj(w["method"])["risk_calculation"] = "product" },
		func(w Object) { dObj(w["method"])["impact_dimensions"] = Object{"safety": "人员安全"} },
	} {
		changed := Clone(ws)
		mutate(changed)
		if AssessmentHash(changed, scenario) == base {
			t.Fatal("semantic change did not invalidate decision")
		}
	}
	first, second := Clone(ws), Clone(ws)
	dObj(dList(dObj(first["architecture"])["flows"])[0])["communication"] = Object{"security_properties": []any{"integrity", "authenticity"}}
	dObj(dList(dObj(second["architecture"])["flows"])[0])["communication"] = Object{"security_properties": []any{"authenticity", "integrity"}}
	if AssessmentHash(first, scenario) != AssessmentHash(second, scenario) {
		t.Fatal("security-property order changed meaning")
	}
}

func TestDomainGenericMetadataBoundaries(t *testing.T) {
	for _, kind := range []string{"device", "application", "cloud", "actor", "ecu", "external", "service"} {
		architecture := genericArchitecture(t)
		node := dObj(dList(architecture["nodes"])[0])
		node["entityType"] = kind
		node["operating_system"] = Object{"family": "rtos", "name": "测试系统"}
		flow := dObj(dList(architecture["flows"])[0])
		flow["communication"] = Object{"medium": "wireless", "technology": "BLE", "range": "near_field", "purpose": "状态同步"}
		got, err := ValidatePayload("replace-architecture", architecture)
		if err != nil {
			t.Fatal(err)
		}
		if len(dList(dObj(dObj(dList(got["flows"])[0])["communication"])["security_properties"])) != 0 {
			t.Fatal("security properties were inferred")
		}
	}
	for _, mutate := range []func(Object){
		func(a Object) {
			n := dObj(dList(a["nodes"])[0])
			n["kind"] = "boundary"
			n["entityType"] = nil
			n["operating_system"] = Object{}
		},
		func(a Object) { dObj(dList(a["nodes"])[0])["operating_system"] = Object{"family": "unknown-os"} },
		func(a Object) { dObj(dList(a["flows"])[0])["communication"] = Object{"medium": "radio"} },
		func(a Object) {
			dObj(dList(a["flows"])[0])["communication"] = Object{"security_properties": []any{"encrypted"}}
		},
	} {
		a := genericArchitecture(t)
		mutate(a)
		if _, err := ValidatePayload("replace-architecture", a); err == nil {
			t.Fatal("invalid generic metadata accepted")
		}
	}
	legacy, err := ValidatePayload("define-item", Object{})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := legacy["domain"]; present {
		t.Fatal("legacy item gained default domain")
	}
	if _, err := ValidatePayload("define-item", Object{"domain": "arbitrary"}); err == nil {
		t.Fatal("unknown domain accepted")
	}
}
