package assurance

import (
	"strings"
	"testing"
)

func automationFixture(t *testing.T) Object {
	t.Helper()
	ws := NewWorkspace("Synthetic deterministic analysis", "")
	graph := Object{"view_mode": "entities", "nodes": []any{
		Object{"id": "device", "name": "Synthetic controller", "kind": "entity", "entityType": "device", "x": 100, "y": 100, "operating_system": Object{"family": "linux", "name": "Synthetic Linux", "version": "1.0"}, "children": []any{Object{"id": "store", "name": "Synthetic configuration", "kind": "store"}}},
		Object{"id": "peer", "name": "Synthetic external peer", "kind": "entity", "entityType": "external", "x": 600, "y": 100},
		Object{"id": "visual-boundary", "name": "Visual only", "kind": "boundary", "x": 80, "y": 80, "width": 300, "height": 300},
	}, "flows": []any{
		Object{"id": "update", "name": "Synthetic update", "source": "peer", "target": "device", "protocol": "Wi-Fi", "communication": Object{"medium": "wireless", "technology": "wifi", "purpose": "firmware update", "security_properties": []any{"integrity", "availability", "freshness"}}},
		Object{"id": "maintenance", "name": "Synthetic maintenance", "source": "peer", "target": "device", "protocol": "JTAG", "communication": Object{"medium": "wired", "technology": "JTAG"}},
	}}
	next, _, err := Apply(ws, "replace-architecture", graph)
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func automationCandidate(t *testing.T, plan Object, rule, subjectID string) Object {
	t.Helper()
	for _, raw := range dList(plan["candidates"]) {
		c := dObj(raw)
		if c["rule_id"] == rule && (subjectID == "" || dObj(c["subject"])["id"] == subjectID) {
			return c
		}
	}
	t.Fatalf("missing rule %s for %s", rule, subjectID)
	return nil
}
func automationSelectableIDs(plan Object) []string {
	ids := []string{}
	for _, raw := range dList(plan["candidates"]) {
		c := dObj(raw)
		if dBool(c["selectable"]) {
			ids = append(ids, dStr(c["id"]))
		}
	}
	return ids
}

func TestAutomationRulesOriginalPlanAndUnratedAdoption(t *testing.T) {
	ws := automationFixture(t)
	before := Digest(ws)
	plan := AutomationPlan(ws)
	rules := map[string]bool{}
	for _, raw := range dList(plan["candidates"]) {
		c := dObj(raw)
		rules[dStr(c["rule_id"])] = true
		if !dBool(c["selectable"]) || len(dList(c["matched_conditions"])) == 0 || len(dList(c["prerequisites"])) == 0 || len(dList(c["controls_to_check"])) == 0 {
			t.Fatal("missing review context", c)
		}
		for _, ref := range dList(c["asset_ids"]) {
			a, err := dFind(dList(ws["assets"]), ref)
			if err != nil || !dHas(a["properties"], c["property"]) || assetSource(a)["id"] != dObj(c["subject"])["id"] {
				t.Fatal("inaccurate asset association", c, a, err)
			}
		}
	}
	if len(rules) != 8 {
		t.Fatalf("got %d original rules: %v", len(rules), rules)
	}
	proposal, err := AutomationProposal(ws, automationSelectableIDs(plan))
	if err != nil {
		t.Fatal(err)
	}
	if Digest(ws) != before {
		t.Fatal("pure analysis mutated workspace")
	}
	for _, key := range []string{"assets", "security_goals", "treatments", "citations"} {
		if len(dList(proposal[key])) != 0 {
			t.Fatalf("invented %s", key)
		}
	}
	next, counts, err := validateAIProposal(ws, proposal, "analysis", "Synthetic reviewer", nil)
	if err != nil {
		t.Fatal("proposal incompatible with native domain", err)
	}
	if dInt(counts["scenarios"]) != len(dList(plan["candidates"])) {
		t.Fatal(counts)
	}
	for _, raw := range dList(next["scenarios"]) {
		s := dObj(raw)
		if !strings.HasPrefix(dStr(s["source_code"]), "AUTO:ac_") {
			t.Fatal("missing persistent dedup provenance", s)
		}
		if s["cal"] != nil || Risk(s, dObj(next["method"]))["level"] != nil {
			t.Fatal("invented rating", s)
		}
		for _, key := range impactKeys {
			if dObj(s["impact"])[key] != nil {
				t.Fatal("invented impact")
			}
		}
		for _, p := range dList(s["paths"]) {
			if dObj(p)["feasibility"] != nil {
				t.Fatal("invented feasibility")
			}
		}
	}
	after := AutomationPlan(next)
	if dInt(dObj(after["summary"])["selectable"]) != 0 || dInt(dObj(after["summary"])["already_covered"]) != len(dList(plan["candidates"])) {
		t.Fatal("repeat adoption not deduplicated", after["summary"])
	}
	if _, err := AutomationProposal(next, automationSelectableIDs(plan)); err == nil {
		t.Fatal("reapplied existing work products")
	}
}

func TestAutomationRulesStablePureAndGeometryIndependent(t *testing.T) {
	ws := automationFixture(t)
	initial := AutomationPlan(ws)
	for i := 0; i < 20; i++ {
		if Digest(initial) != Digest(AutomationPlan(ws)) {
			t.Fatal("nondeterministic plan")
		}
	}
	moved := Clone(ws)
	moved["revision"] = dInt(ws["revision"]) + 10
	moved["updated_at"] = "different timestamp"
	for _, raw := range dList(dObj(moved["architecture"])["nodes"]) {
		n := dObj(raw)
		n["x"], n["y"], n["width"], n["height"] = 999, 555, 700, 900
	}
	for _, raw := range dList(dObj(moved["architecture"])["flows"]) {
		f := dObj(raw)
		f["sourceSide"], f["targetSide"] = "top", "bottom"
		f["labelOffset"] = Object{"x": 30, "y": -20}
		f["route_layout"] = Object{"waypoint": Object{"x": 50, "y": 80}}
		f["wireless_layout"] = Object{"source_offset": Object{"x": 10, "y": 20}}
	}
	graph := dObj(moved["architecture"])
	graph["show_numbers"], graph["equal_spacing"] = false, false
	movedPlan := AutomationPlan(moved)
	if initial["plan_hash"] != movedPlan["plan_hash"] {
		t.Fatal("geometry changed semantic plan hash")
	}
	if movedPlan["base_revision"] == initial["base_revision"] {
		t.Fatal("revision omitted")
	}
	changed := Clone(ws)
	dObj(dObj(dList(dObj(changed["architecture"])["nodes"])[0])["operating_system"])["version"] = "2.0"
	changedPlan := AutomationPlan(changed)
	if initial["plan_hash"] == changedPlan["plan_hash"] {
		t.Fatal("semantic OS change omitted from hash")
	}
	oldCandidate := automationCandidate(t, initial, "OS-NETWORK-SURFACE", "device")
	newCandidate := automationCandidate(t, changedPlan, "OS-NETWORK-SURFACE", "device")
	if oldCandidate["id"] != newCandidate["id"] {
		t.Fatal("candidate identity changed with attribute update")
	}
	for _, key := range []string{"nodes", "flows"} {
		rows := dList(graph[key])
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	if movedPlan["plan_hash"] != AutomationPlan(moved)["plan_hash"] {
		t.Fatal("model order changed semantic plan")
	}
}

func TestAutomationRulesDoNotInferFromVisualBoundaryOrUnknownFacts(t *testing.T) {
	ws := NewWorkspace("Synthetic no match", "")
	ws["architecture"] = Object{"nodes": []any{
		Object{"id": "a", "kind": "entity", "name": "A", "x": 1, "y": 1, "operating_system": Object{"family": "unknown"}},
		Object{"id": "b", "kind": "entity", "name": "B", "x": 999, "y": 999, "operating_system": Object{"family": "bare_metal"}},
		Object{"id": "box", "kind": "boundary", "name": "Trust rectangle", "x": 0, "y": 0, "width": 300, "height": 300},
	}, "flows": []any{
		Object{"id": "data", "name": "CAN", "source": "a", "target": "b", "protocol": "CAN FD"},
		Object{"id": "energy", "name": "Not data", "source": "a", "target": "b", "protocol": "JTAG", "channel_kind": "energy", "communication": Object{"medium": "wireless", "security_properties": []any{"availability"}}},
	}}
	if len(dList(AutomationPlan(ws)["candidates"])) != 0 {
		t.Fatal("invented applicability from visual position, unknown OS or energy flow")
	}
	nodes := dList(dObj(ws["architecture"])["nodes"])
	dObj(nodes[0])["attributes"] = []any{Object{"name": "trust_zone", "value": "service"}}
	dObj(nodes[1])["attributes"] = []any{Object{"name": "信任域", "value": "maintenance"}}
	c := automationCandidate(t, AutomationPlan(ws), "DF-TRUST-INPUT", "data")
	if dBool(c["selectable"]) {
		t.Fatal("selected without assets")
	}
	if len(dList(AutomationPlan(ws)["candidates"])) != 1 {
		t.Fatal("unexpected explicit trust match")
	}
}

func TestAutomationRulesAssetAssociationAndMissingProperty(t *testing.T) {
	ws := automationFixture(t)
	plan := AutomationPlan(ws)
	wireless := automationCandidate(t, plan, "DF-WIRELESS-INTEGRITY", "update")
	ws["assets"] = []any{}
	missing := AutomationPlan(ws)
	if dInt(dObj(missing["summary"])["selectable"]) != 0 {
		t.Fatal("selected without assets")
	}
	if _, err := AutomationProposal(ws, []string{dStr(wireless["id"])}); err == nil {
		t.Fatal("missing asset adoption succeeded")
	}
	ws["assets"] = []any{Object{"id": "manual-a", "name": "Endpoint only", "node_ids": []any{"device"}, "properties": []any{"integrity"}}}
	if dBool(automationCandidate(t, AutomationPlan(ws), "DF-WIRELESS-INTEGRITY", "update")["selectable"]) {
		t.Fatal("flow incorrectly linked to one endpoint asset")
	}
	ws["assets"] = []any{Object{"id": "manual-a", "name": "Transmission", "node_ids": []any{"device", "peer"}, "properties": []any{"integrity"}}}
	c := automationCandidate(t, AutomationPlan(ws), "DF-WIRELESS-INTEGRITY", "update")
	if !dBool(c["selectable"]) || dList(c["asset_ids"])[0] != "manual-a" {
		t.Fatal("exact manual flow association absent", c)
	}
	fresh := automationCandidate(t, AutomationPlan(ws), "DF-FRESHNESS", "update")
	if dBool(fresh["selectable"]) {
		t.Fatal("asset does not declare required freshness")
	}
	second := Clone(dObj(dList(ws["assets"])[0]))
	second["id"] = "manual-b"
	ws["assets"] = append(dList(ws["assets"]), second)
	ambiguous := automationCandidate(t, AutomationPlan(ws), "DF-WIRELESS-INTEGRITY", "update")
	if dBool(ambiguous["selectable"]) || len(dList(ambiguous["asset_options"])) != 2 {
		t.Fatal("ambiguous manual assets selected arbitrarily", ambiguous)
	}
}

func TestAutomationRulesSelectionAndExistingHumanWork(t *testing.T) {
	ws := automationFixture(t)
	c := automationCandidate(t, AutomationPlan(ws), "DF-WIRELESS-INTEGRITY", "update")
	id := dStr(c["id"])
	proposal, err := AutomationProposal(ws, []string{id, id})
	if err != nil || len(dList(proposal["scenarios"])) != 1 {
		t.Fatal("duplicate selection generated repeated records", err)
	}
	for _, ids := range [][]string{{}, {"unknown"}, {id, "unknown"}} {
		if _, err := AutomationProposal(ws, ids); err == nil {
			t.Fatal("invalid selection accepted", ids)
		}
	}
	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = id
	}
	if _, err := AutomationProposal(ws, tooMany); err == nil {
		t.Fatal("unbounded adoption")
	}
	next, _, err := validateAIProposal(ws, proposal, "analysis", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	human := dObj(dList(next["scenarios"])[0])
	delete(human, "source_code")
	before := Digest(next)
	covered := automationCandidate(t, AutomationPlan(next), "DF-WIRELESS-INTEGRITY", "update")
	if !dBool(covered["already_covered"]) {
		t.Fatal("equivalent human work not recognised")
	}
	if _, err := AutomationProposal(next, []string{id}); err == nil || Digest(next) != before {
		t.Fatal("human work overwritten")
	}
	human["title"], human["threat"] = "Different human analysis", "A distinct manually confirmed threat"
	related := automationCandidate(t, AutomationPlan(next), "DF-WIRELESS-INTEGRITY", "update")
	if !dBool(related["selectable"]) || len(dList(related["related_scenario_ids"])) != 1 {
		t.Fatal("broad same-STRIDE match incorrectly suppresses distinct finding", related)
	}
	human["source_code"] = "AUTO:" + id
	if !dBool(automationCandidate(t, AutomationPlan(next), "DF-WIRELESS-INTEGRITY", "update")["already_covered"]) {
		t.Fatal("human edit lost stable provenance")
	}
}

func TestAutomationRulesReuseExistingDamageAndDoNotClaimVulnerability(t *testing.T) {
	ws := automationFixture(t)
	c := automationCandidate(t, AutomationPlan(ws), "OS-NETWORK-SURFACE", "device")
	id := dStr(c["id"])
	proposal, err := AutomationProposal(ws, []string{id})
	if err != nil {
		t.Fatal(err)
	}
	damageOnly := Clone(proposal)
	damageOnly["scenarios"], damageOnly["requirements"] = []any{}, []any{}
	next, _, err := validateAIProposal(ws, damageOnly, "analysis", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := AutomationProposal(next, []string{id})
	if err != nil || len(dList(second["damage_scenarios"])) != 0 {
		t.Fatal("existing damage duplicated", err)
	}
	s := dObj(dList(second["scenarios"])[0])
	if dList(s["damage_ids"])[0] != dObj(dList(next["damage_scenarios"])[0])["id"] {
		t.Fatal("wrong damage reuse")
	}
	if !strings.Contains(dStr(s["threat"]), "CVE 适用性需核对组件、版本和配置") {
		t.Fatal("OS rule omitted applicability qualification")
	}
	if _, _, err := validateAIProposal(next, second, "analysis", "", nil); err != nil {
		t.Fatal("existing damage reference not accepted", err)
	}
}

func TestAutomationRulesDoNotTreatActorOrConflictingZonesAsExternal(t *testing.T) {
	if automationExternal(Object{"kind": "entity", "entityType": "actor"}) {
		t.Fatal("an actor is not necessarily outside the trust boundary")
	}
	n := Object{"attributes": []any{Object{"name": "trust_zone", "value": "operational"}, Object{"name": "信任域", "value": "maintenance"}}}
	if automationZone(n) != "" {
		t.Fatal("conflicting trust attributes treated as certain")
	}
}

func TestAutomationRulesChildStoreRequiresExactSource(t *testing.T) {
	ws := automationFixture(t)
	var exactStore Object
	for _, raw := range dList(ws["assets"]) {
		a := dObj(raw)
		if assetSource(a)["kind"] == "store" && assetSource(a)["id"] == "store" {
			exactStore = Clone(a)
			break
		}
	}
	if exactStore == nil {
		t.Fatal("fixture has no exact storage asset")
	}
	dObj(dList(dObj(dList(dObj(ws["architecture"])["nodes"])[0])["children"])[0])["name"] = "Synthetic key store"
	ws["assets"] = []any{Object{"id": "manual-logs", "name": "Synthetic logs", "node_ids": []any{"device"}, "properties": []any{"confidentiality"}}}
	before := Digest(ws)
	candidate := automationCandidate(t, AutomationPlan(ws), "DS-CONFIDENTIALITY", "store")
	if dBool(candidate["selectable"]) || len(dList(candidate["asset_ids"])) != 0 || len(dList(candidate["asset_options"])) != 0 {
		t.Fatal("parent-only log asset was associated with internal key storage", candidate)
	}
	if _, err := AutomationProposal(ws, []string{dStr(candidate["id"])}); err == nil {
		t.Fatal("accepted internal storage analysis without exact source association")
	}
	if Digest(ws) != before {
		t.Fatal("rejected analysis changed workspace")
	}
	ws["assets"] = append(dList(ws["assets"]), exactStore)
	exact := automationCandidate(t, AutomationPlan(ws), "DS-CONFIDENTIALITY", "store")
	if !dBool(exact["selectable"]) || len(dList(exact["asset_ids"])) != 1 || dList(exact["asset_ids"])[0] != exactStore["id"] {
		t.Fatal("exact child storage association was not retained", exact)
	}
}
