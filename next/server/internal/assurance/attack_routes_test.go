package assurance

import (
	"strings"
	"testing"
)

func routeFixture() Object {
	nodes := []any{}
	for _, id := range []string{"outside", "gateway", "relay", "ecu"} {
		n := Object{"id": id, "name": id, "kind": "entity", "entityType": "device"}
		if id == "outside" {
			n["entityType"] = "external"
		}
		nodes = append(nodes, n)
	}
	flows := []any{}
	add := func(id, source, target, protocol, kind string) {
		flows = append(flows, Object{"id": id, "name": id, "source": source, "target": target, "protocol": protocol, "channel_kind": kind})
	}
	add("eth", "outside", "gateway", "ETH", "data")
	add("can", "gateway", "ecu", "CAN", "data")
	add("wifi", "outside", "relay", "Wi-Fi", "data")
	add("usb", "relay", "gateway", "USB", "data")
	add("cycle", "gateway", "outside", "ETH", "data")
	add("power", "outside", "ecu", "12V", "energy")
	return Object{"revision": 3, "architecture": Object{"nodes": nodes, "flows": flows}, "assets": []any{Object{"id": "asset", "node_ids": []any{"ecu"}}}, "scenarios": []any{Object{"id": "threat", "asset_id": "asset", "paths": []any{}}}}
}
func TestAttackRoutesDirectedShortestProtocolAlternatives(t *testing.T) {
	ws := routeFixture()
	out, e := attackRouteCandidates(ws, "threat", "", "")
	if e != nil {
		t.Fatal(e)
	}
	rows := libRows(out["items"])
	if len(rows) != 2 || rows[0]["hops"] != 2 || rows[1]["hops"] != 3 || rows[0]["shortest"] != true || rows[1]["shortest"] != false {
		t.Fatal(out)
	}
	for _, r := range rows {
		route := dObj(r["architecture_route"])
		if e := validateAttackRoute(ws, route); e != nil {
			t.Fatal(e)
		}
		if dHas(route["flow_ids"], "power") || dHas(route["flow_ids"], "cycle") {
			t.Fatal("traversed energy or cycle")
		}
		if attackRouteCurrent(ws, route) {
			t.Fatal("candidate self approved")
		}
		if !strings.HasPrefix(dStr(r["steps"]), "1、") {
			t.Fatal("steps not numbered")
		}
	}
	reverse, e := attackRouteCandidates(ws, "threat", "ecu", "")
	if e != nil || len(dList(reverse["items"])) != 0 {
		t.Fatal(reverse, e)
	}
	if _, e := attackRouteCandidates(ws, "threat", "", "gateway"); e == nil {
		t.Fatal("unrelated target accepted")
	}
	if _, e := attackRouteCandidates(ws, "missing", "", ""); e == nil {
		t.Fatal("missing scenario accepted")
	}
}
func TestRouteEvidenceChangesInvalidateRatingButLayoutDoesNot(t *testing.T) {
	ws := routeFixture()
	out, _ := attackRouteCandidates(ws, "threat", "", "")
	route := Clone(dObj(dObj(dList(out["items"])[0])["architecture_route"]))
	route["reviewed"] = true
	p := Object{"id": "p", "steps": "1、检查入口\n2、检查协议转换", "architecture_route": route, "feasibility": 4, "feasibility_method": "attack_vector", "attack_vector": "network", "rationale": "证据"}
	dObj(dList(ws["scenarios"])[0])["paths"] = []any{p}
	dObj(dList(dObj(ws["architecture"])["nodes"])[0])["x"] = 900
	invalidateAttackRoutes(ws)
	if p["feasibility"] == nil {
		t.Fatal("layout invalidated assessment")
	}
	dObj(dList(dObj(ws["architecture"])["flows"])[0])["protocol"] = "USB"
	invalidateAttackRoutes(ws)
	if p["feasibility"] != nil || route["reviewed"] != false {
		t.Fatal("semantic change kept rating")
	}
	if validateAttackRoute(ws, route) == nil {
		t.Fatal("stale route accepted")
	}
}

func TestRequirementVerificationTracksAssessmentAfterEditingLegacyRecord(t *testing.T) {
	g := dFixture(t)
	steps := dList(g["steps"])
	ws := Clone(dObj(dObj(steps[len(steps)-1])["expected"]))
	requirements := libRows(ws["requirements"])
	if len(requirements) == 0 {
		t.Fatal("fixture requires requirement")
	}
	requirement := Clone(requirements[0])
	var err error
	ws, _, err = Apply(ws, "specify-requirement", requirement)
	if err != nil {
		t.Fatal(err)
	}
	requirement = dObj(dList(ws["requirements"])[0])
	if requirement["scenario_ids"] == nil || requirement["goal_ids"] == nil {
		t.Fatal("edited requirement lacks context")
	}
	ws, _, err = Apply(ws, "record-evidence", Object{"requirement_id": requirement["id"], "result": "pass", "reference": "TEST-001", "summary": "Synthetic verification", "reviewer": "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if dObj(dList(Review(ws)["requirements"])[0])["status"] != "pass" {
		t.Fatal("new evidence not current")
	}
	item := Clone(dObj(ws["item"]))
	item["assumptions"] = "Changed access assumptions"
	ws, _, err = Apply(ws, "define-item", item)
	if err != nil {
		t.Fatal(err)
	}
	if dObj(dList(Review(ws)["requirements"])[0])["status"] != "stale" {
		t.Fatal("changed analysis retained verification pass")
	}
	if _, _, err = Apply(ws, "seal-baseline", Object{"name": "invalid", "reviewer": "tester", "note": ""}); err == nil {
		t.Fatal("stale evidence allowed baseline")
	}
}
func TestAttackRouteHTTPReadContract(t *testing.T) {
	h, _, _ := httpTestServer(t)
	id := httpCreate(t, h)
	path := apiPrefix + "/workspaces/" + id + "/attack-route-candidates"
	for _, suffix := range []string{"", "?scenario_id=missing", "?unknown=1"} {
		res := httpCall(h, "GET", path+suffix, "", nil)
		if res.Code != 422 && res.Code != 400 {
			t.Fatal(res.Code, res.Body.String())
		}
	}
}

func TestLinkedDamageWithoutRationaleNeverProducesRisk(t *testing.T) {
	ws := NewWorkspace("rating integrity", "")
	apply := func(action string, body Object) string {
		var id string
		var err error
		ws, id, err = Apply(ws, action, body)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	a := apply("record-asset", Object{"name": "command", "properties": []any{"integrity"}})
	d := apply("record-damage", Object{"description": "unintended output", "asset_ids": []any{a}, "properties": []any{"integrity"}, "impact": Object{"safety": 1, "financial": 0, "operational": 1, "privacy": 0}})
	apply("record-scenario", Object{"title": "modified command", "asset_id": a, "property": "integrity", "damage": "unintended output", "damage_ids": []any{d}, "threat": "modified command executed", "paths": []any{Object{"name": "input", "steps": "1、send command", "feasibility": 2, "rationale": "restricted access"}}})
	if Risk(dObj(dList(ws["scenarios"])[0]), dObj(ws["method"]))["level"] != nil {
		t.Fatal("generated reference text bypassed missing damage rationale")
	}
}

func TestRouteTargetFollowsAssetLocation(t *testing.T) {
	ws := routeFixture()
	out, _ := attackRouteCandidates(ws, "threat", "", "")
	route := Clone(dObj(dObj(dList(out["items"])[0])["architecture_route"]))
	route["reviewed"] = true
	scenario := dObj(dList(ws["scenarios"])[0])
	path := Object{"id": "p", "architecture_route": route, "feasibility": 3, "rationale": "assessed"}
	scenario["paths"] = []any{path}
	if !attackRouteTargetsScenario(ws, scenario, route) {
		t.Fatal("associated target rejected")
	}
	dObj(dList(ws["assets"])[0])["node_ids"] = []any{"gateway"}
	invalidateAttackRoutes(ws)
	if path["feasibility"] != nil || route["reviewed"] != false {
		t.Fatal("moved asset kept unrelated route rating")
	}
	if attackRouteTargetsScenario(ws, scenario, route) {
		t.Fatal("unrelated target accepted")
	}
}

func TestRoutesDoNotAddLongerSameProtocolDetours(t *testing.T) {
	ws := routeFixture()
	graph := dObj(ws["architecture"])
	graph["nodes"] = append(dList(graph["nodes"]), Object{"id": "switch", "name": "switch", "kind": "entity", "entityType": "device"})
	graph["flows"] = append(dList(graph["flows"]), Object{"id": "eth1", "source": "outside", "target": "switch", "protocol": "ETH"}, Object{"id": "eth2", "source": "switch", "target": "gateway", "protocol": "ETH"})
	result, err := attackRouteCandidates(ws, "threat", "", "")
	if err != nil || len(dList(result["items"])) != 2 {
		t.Fatal("redundant same-protocol detour", result, err)
	}
}
