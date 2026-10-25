package assurance

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func coefficientMethod() Object {
	m := EffectiveMethod(DefaultMethod)
	m["impact_aggregation"], m["risk_calculation"] = "coefficient_sum", "product"
	return m
}

func coefficientScenario() Object {
	s := genericScenario()
	s["impact"] = Object{"safety": 3, "financial": 2, "operational": 1, "privacy": 1, "rationale": "原创系数法依据"}
	dObj(dList(s["paths"])[0])["feasibility"] = 2
	return s
}

func TestDomainCoefficientRawScoreResidualAndReport(t *testing.T) {
	method, scenario := coefficientMethod(), coefficientScenario()
	result := Risk(scenario, method)
	if result["impact"] != 5.2 || result["score"] != 10.4 || result["level"] != 4 {
		t.Fatalf("coefficient total/product must remain raw: %v", result)
	}
	path := dObj(dList(result["paths"])[0])
	if path["score"] != 10.4 || path["level"] != 4 {
		t.Fatal("path raw score missing")
	}
	ws := NewWorkspace("原创系数法项目", "")
	ws["method"], ws["method_confirmed"], ws["scenarios"] = method, true, []any{scenario}
	payload := Object{"scenario_id": scenario["id"], "treatment": "retain", "rationale": "原创处置理由", "owner": "测试人员", "claim": "测试声明", "residual_impact": 2.5, "residual_feasibility": 2, "residual_rationale": "独立残余评估"}
	next, _, err := Apply(ws, "decide-treatment", payload)
	if err != nil {
		t.Fatal(err)
	}
	evaluation := dObj(dList(Review(next)["evaluations"])[0])
	if evaluation["residual"] != 2 || evaluation["residual_score"] != 5.0 || dBool(evaluation["decision_stale"]) {
		t.Fatalf("residual score must be 2.5*2=5: %v", evaluation)
	}
	tables := Tables(next, nil, nil)
	if len(tables) != 17 {
		t.Fatal("report table count changed")
	}
	for _, table := range tables {
		columns := libList(table["columns"])
		for _, row := range libList(table["rows"]) {
			if len(libList(row)) != len(columns) {
				t.Fatalf("report row/column mismatch: %s", table["name"])
			}
		}
	}
	riskRow := libList(libList(tables[9]["rows"])[0])
	residualRow := libList(libList(tables[11]["rows"])[0])
	if riskRow[4] != "5.2" || riskRow[6] != 10.4 || riskRow[7] != 4 {
		t.Fatalf("report truncated SFOP total: %v", riskRow)
	}
	if residualRow[4] != "2.5" || residualRow[6] != 5.0 || residualRow[7] != 2 {
		t.Fatalf("report truncated residual total: %v", residualRow)
	}
	encoded, _ := CanonicalJSON(tables)
	for _, text := range []string{"SFOP总分", "风险分值", "不归一化、不取整", "直接使用原始总分，不加1"} {
		if !strings.Contains(string(encoded), text) {
			t.Errorf("report missing %s", text)
		}
	}
	// Persist/reload changes numeric Go types, never the grade or source digest.
	raw, _ := CanonicalJSON(next)
	restored, err := DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	dEqual(t, Review(restored), Review(next))
}

func TestDomainCoefficientDecimalBoundariesAndMissingDimensions(t *testing.T) {
	method := coefficientMethod()
	method["impact_coefficients"] = Object{"safety": 0.1, "financial": 0.1, "operational": 0, "privacy": 0}
	method["product_thresholds"] = []any{0.3, 0.6, 0.9, 1.2}
	scenario := coefficientScenario()
	scenario["impact"] = Object{"safety": 1, "financial": 2, "operational": nil, "privacy": nil, "rationale": "零系数维度不适用"}
	dObj(dList(scenario["paths"])[0])["feasibility"] = 1
	result := Risk(scenario, method)
	if result["impact"] != 0.3 || result["score"] != 0.3 || result["level"] != 1 {
		t.Fatalf("0.1+0.2 must equal threshold 0.3: %v", result)
	}
	dObj(dList(scenario["paths"])[0])["feasibility"] = 3
	result = Risk(scenario, method)
	if result["score"] != 0.9 || result["level"] != 3 {
		t.Fatalf("0.3*3 must equal threshold 0.9: %v", result)
	}
	dObj(scenario["impact"])["financial"] = nil
	if Risk(scenario, method)["level"] != nil {
		t.Fatal("positive-coefficient dimension missing but received risk")
	}
	method["impact_coefficients"] = Object{"safety": 100, "financial": 100, "operational": 100, "privacy": 100}
	scenario["impact"] = Object{"safety": 3, "financial": 3, "operational": 3, "privacy": 3, "rationale": "上限测试"}
	dObj(dList(scenario["paths"])[0])["feasibility"] = 4
	method["product_thresholds"] = []any{0, 1, 2, 4800}
	result = Risk(scenario, method)
	if result["impact"] != 1200.0 || result["score"] != 4800.0 || result["level"] != 4 {
		t.Fatalf("maximum score or inclusive last threshold incorrect: %v", result)
	}
}

func TestDomainCoefficientValidationAndAtomicResidualBounds(t *testing.T) {
	for _, extra := range []Object{
		{"risk_calculation": "matrix"},
		{"impact_coefficients": Object{"safety": 0, "financial": 0, "operational": 0, "privacy": 0}},
		{"impact_coefficients": Object{"safety": -0.1}}, {"impact_coefficients": Object{"safety": 100.1}},
		{"impact_coefficients": Object{"safety": math.NaN()}}, {"impact_coefficients": Object{"safety": math.Inf(1)}},
		{"impact_coefficients": Object{"safety": "0.1"}}, {"impact_coefficients": Object{"safety": true}},
		{"product_thresholds": []any{0.1, 0.2, 0.2, 1}}, {"product_thresholds": []any{-0.1, 0.2, 0.3, 1}},
		{"product_thresholds": []any{0.1, 0.2, 0.3, 4800.1}}, {"product_thresholds": []any{0.1, 0.2, 0.3, math.Inf(1)}},
	} {
		m := coefficientMethod()
		for key, value := range extra {
			m[key] = value
		}
		if _, err := ValidatePayload("adopt-method", m); err == nil {
			t.Fatalf("invalid coefficient configuration accepted: %v", extra)
		}
	}
	valid := coefficientMethod()
	valid["product_thresholds"] = []any{0, 0.1, 0.3, 4800}
	if _, err := ValidatePayload("adopt-method", valid); err != nil {
		t.Fatal(err)
	}
	for _, thresholds := range []any{[]any{0, 1, 2, 3}, []any{0.1, 2, 3, 4}, []any{3, 6, 9, 4800}} {
		ordinary := Clone(DefaultMethod)
		ordinary["product_thresholds"] = thresholds
		if _, err := ValidatePayload("adopt-method", ordinary); err == nil {
			t.Fatal("ordinary method accepted coefficient thresholds")
		}
	}
	ws := NewWorkspace("原子边界测试", "")
	ws["method"], ws["method_confirmed"], ws["scenarios"] = coefficientMethod(), true, []any{coefficientScenario()}
	before := Digest(ws)
	payload := Object{"scenario_id": "scenario_local", "treatment": "retain", "rationale": "理由", "owner": "负责人", "claim": "声明", "residual_impact": 6.6000000001, "residual_feasibility": 2, "residual_rationale": "独立依据"}
	if _, _, err := Apply(ws, "decide-treatment", payload); err == nil {
		t.Fatal("residual exceeded 3*(1+1+.1+.1)=6.6")
	}
	if Digest(ws) != before {
		t.Fatal("rejected residual mutated aggregate")
	}
	payload["residual_impact"] = 6.6
	if _, _, err := Apply(ws, "decide-treatment", payload); err != nil {
		t.Fatalf("inclusive decimal maximum rejected: %v", err)
	}
	ws["method"] = Clone(DefaultMethod)
	for _, value := range []any{2.5, 3.1, 4, json.Number("2.0")} {
		payload["residual_impact"] = value
		if _, _, err := Apply(ws, "decide-treatment", payload); err == nil {
			t.Fatalf("ordinary residual accepted %v", value)
		}
	}
}

func TestDomainCoefficientDefaultHashCompatibility(t *testing.T) {
	ws := NewWorkspace("原摘要", "")
	ws["architecture"] = genericArchitecture(t)
	scenario := coefficientScenario()
	before := AssessmentHash(ws, scenario)
	method := Clone(dObj(ws["method"]))
	method["impact_coefficients"] = Object{"safety": 1.0, "financial": 1, "operational": 0.1, "privacy": 0.1}
	validated, err := ValidatePayload("adopt-method", method)
	if err != nil {
		t.Fatal(err)
	}
	ws["method"] = validated
	if AssessmentHash(ws, scenario) != before {
		t.Fatal("explicit default coefficients invalidated legacy decision")
	}
	dObj(validated["impact_coefficients"])["operational"] = 0.2
	if AssessmentHash(ws, scenario) == before {
		t.Fatal("real coefficient change did not invalidate decision")
	}
}
