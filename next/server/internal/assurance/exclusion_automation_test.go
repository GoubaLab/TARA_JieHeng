package assurance

import (
	"strings"
	"testing"
)

func TestAutomationRespectsExcludedPropertyAndKeepsOtherCoverage(t *testing.T) {
	ws := automationFixture(t)
	before := AutomationPlan(ws)
	old := automationCandidate(t, before, "DF-WIRELESS-INTEGRITY", "update")
	asset := attackTemplateAsset(t, ws, "flow", "update")
	asset["excluded_properties"] = []any{Object{"property": "integrity", "reason": "Synthetic scope decision"}}
	// A manual asset on the same endpoints must not bypass the exact flow scope.
	ws["assets"] = append(dList(ws["assets"]), Object{"id": "manual", "name": "Different data", "node_ids": dClone(asset["node_ids"]), "properties": []any{"integrity"}})
	plan := AutomationPlan(ws)
	if plan["plan_hash"] == before["plan_hash"] {
		t.Fatal("scope decision omitted from plan revision")
	}
	for _, raw := range dList(plan["candidates"]) {
		c := dObj(raw)
		if dObj(c["subject"])["id"] == "update" && c["property"] == "integrity" {
			t.Fatal("excluded exact source generated a candidate", c)
		}
		for _, rawOption := range dList(c["asset_options"]) {
			a, _ := dFind(dList(ws["assets"]), dObj(rawOption)["id"])
			if assetPropertyExcluded(a, c["property"]) {
				t.Fatal("excluded property remained selectable")
			}
		}
	}
	if !dBool(automationCandidate(t, plan, "DF-AVAILABILITY", "update")["selectable"]) || !dBool(automationCandidate(t, plan, "DF-FRESHNESS", "update")["selectable"]) {
		t.Fatal("excluding integrity removed other applicable coverage")
	}
	if _, err := AutomationProposal(ws, []string{dStr(old["id"])}); err == nil {
		t.Fatal("stale excluded candidate was adopted")
	}
	proposal, err := AutomationProposal(ws, automationSelectableIDs(plan))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := validateAIProposal(ws, proposal, "analysis", "Synthetic reviewer", nil); err != nil {
		t.Fatal("remaining automatic candidates are not valid", err)
	}
}

func TestTemplateRecommendationsDoNotRecommendExcludedScope(t *testing.T) {
	ws := automationFixture(t)
	asset := attackTemplateAsset(t, ws, "flow", "update")
	asset["excluded_properties"] = []any{Object{"property": "integrity", "reason": "Synthetic scope"}}
	if _, err := attackPathTemplateItems(ws, dStr(asset["id"]), "integrity"); err == nil || !strings.Contains(err.Error(), "不适用") {
		t.Fatal("explicit excluded property query accepted", err)
	}
	items, err := attackPathTemplateItems(ws, dStr(asset["id"]), "")
	if err != nil {
		t.Fatal(err)
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-WIRELESS-PEER")["match"])["score"]) != 0 {
		t.Fatal("wireless template recommended only excluded/undeclared properties")
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-RESOURCE-EXHAUSTION")["match"])["score"]) == 0 {
		t.Fatal("applicable availability template was suppressed")
	}
	for _, p := range dList(asset["properties"]) {
		if p != "integrity" {
			asset["excluded_properties"] = append(dList(asset["excluded_properties"]), Object{"property": p, "reason": "Synthetic scope"})
		}
	}
	items, err = attackPathTemplateItems(ws, dStr(asset["id"]), "")
	if err != nil || len(items) != 16 {
		t.Fatal("catalog browsing must remain available", err)
	}
	for _, raw := range items {
		if dInt(dObj(dObj(raw)["match"])["score"]) != 0 {
			t.Fatal("fully excluded asset still has matched recommendations")
		}
	}
}

func TestAIAdoptionCannotOverrideAssetScope(t *testing.T) {
	ws := automationFixture(t)
	asset := attackTemplateAsset(t, ws, "flow", "update")
	asset["excluded_properties"] = []any{Object{"property": "integrity", "reason": "Synthetic scope decision"}}
	proposal := Object{"summary": "Synthetic suggestion", "assumptions": []any{}, "questions": []any{}, "citations": []any{}, "scenarios": []any{Object{"id": "new-threat", "title": "Tampered input", "asset_id": asset["id"], "property": "integrity", "damage": "Unexpected output", "threat": "Modified request is accepted", "paths": []any{}}}}
	before := Digest(ws)
	if _, _, err := validateAIProposal(ws, proposal, "analysis", "Synthetic reviewer", nil); err == nil || !strings.Contains(err.Error(), "不适用") {
		t.Fatal("AI adopted a threat for excluded property", err)
	}
	if Digest(ws) != before {
		t.Fatal("rejected AI proposal changed the project")
	}
	proposal["scenarios"] = []any{}
	proposal["assets"] = []any{Object{"id": "new-asset", "name": "Model asset", "properties": []any{"integrity"}, "excluded_properties": []any{Object{"property": "integrity", "reason": "Model decision"}}}}
	if _, _, err := validateAIProposal(ws, proposal, "analysis", "Synthetic reviewer", nil); err == nil || !strings.Contains(err.Error(), "AI 不得") {
		t.Fatal("AI supplied an unapproved exclusion", err)
	}
}
