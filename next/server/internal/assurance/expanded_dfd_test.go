package assurance

import "testing"

func expandedGraph() Object {
	return Object{"view_mode": "dfd", "show_numbers": false, "nodes": []any{
		Object{"id": "e", "name": "External", "kind": "entity", "entityType": "external", "x": 0, "y": 0},
		Object{"id": "p", "name": "Processing", "kind": "process_step", "x": 400, "y": 0},
		Object{"id": "d", "name": "Storage", "kind": "data_store", "x": 800, "y": 0},
		Object{"id": "b", "name": "Boundary", "kind": "boundary", "x": 350, "y": -100, "width": 700, "height": 400},
	}, "flows": []any{
		Object{"id": "in", "name": "Input", "source": "e", "target": "p", "emphasis": "critical"},
		Object{"id": "write", "name": "Write", "source": "p", "target": "d", "emphasis": "internal", "show_label": false},
		Object{"id": "power", "name": "Energy", "source": "e", "target": "p", "channel_kind": "energy"},
	}}
}
func TestExpandedDFDStableIdentityAssetsAndValidation(t *testing.T) {
	graph := expandedGraph()
	ws := saveGraph(t, NewWorkspace("Expanded", ""), graph)
	if len(dList(ws["assets"])) != 5 {
		t.Fatal("expected 3 nodes + 2 data flows, no energy or boundary asset")
	}
	for _, id := range []string{"p", "d"} {
		asset := generatedAsset(t, ws, id)
		dEqual(t, asset["node_ids"], []any{id})
	}
	if assetSource(generatedAsset(t, ws, "p"))["kind"] != "software" || assetSource(generatedAsset(t, ws, "d"))["kind"] != "store" {
		t.Fatal("DFD assets misclassified")
	}
	saved := saveGraph(t, ws, dObj(ws["architecture"]))
	dEqual(t, saved["assets"], ws["assets"])
	if dObj(dList(dObj(saved["architecture"])["nodes"])[1])["kind"] != "process_step" {
		t.Fatal("processing node converted on save")
	}
	asset := generatedAsset(t, ws, "d")
	payload := Clone(asset)
	delete(payload, "architecture_source")
	if _, _, err := Apply(ws, "record-asset", payload); err != nil {
		t.Fatal("cannot edit store asset", err)
	}
	for _, mutate := range []func(Object){
		func(g Object) { delete(g, "view_mode") },
		func(g Object) { dObj(dList(g["flows"])[1])["target"] = "b" },
		func(g Object) { dObj(dList(g["nodes"])[1])["operating_system"] = Object{} },
		func(g Object) { dObj(dList(g["flows"])[1])["channel_kind"] = "arbitrary" },
	} {
		bad := Clone(graph)
		mutate(bad)
		if _, err := ValidatePayload("replace-architecture", bad); err == nil {
			t.Fatal("invalid DFD accepted")
		}
	}
	moved := Clone(ws)
	dObj(moved["architecture"])["show_numbers"] = true
	for _, raw := range dList(dObj(moved["architecture"])["flows"]) {
		f := dObj(raw)
		f["show_label"] = true
		f["emphasis"] = "normal"
	}
	if AssessmentHash(moved, genericScenario()) != AssessmentHash(ws, genericScenario()) {
		t.Fatal("visual settings invalidated risk")
	}
	changed := Clone(ws)
	dObj(dList(dObj(changed["architecture"])["flows"])[0])["channel_kind"] = "energy"
	if AssessmentHash(changed, genericScenario()) == AssessmentHash(ws, genericScenario()) {
		t.Fatal("data/energy semantics omitted from assessment")
	}
}
