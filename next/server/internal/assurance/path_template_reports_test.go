package assurance

import (
	"context"
	"strings"
	"testing"
)

func TestTemplatePathSaveEditAndReportProvenance(t *testing.T) {
	repo, _ := repositoryTest(t)
	ctx := context.Background()
	created := repoTestCreate(t, repo)
	wid := dStr(created["id"])
	_, _, e := repo.Change(ctx, wid, 0, "preset-test-asset", "record-asset", Object{"name": "Synthetic control data", "properties": []any{"integrity"}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	ws, _ := repo.Get(ctx, wid)
	asset := dObj(dList(ws["assets"])[0])
	path := Object{"id": "path-synthetic-preset", "name": "Synthetic editable preset", "source_code": "TPL:AP-SYNTHETIC@1", "actor": "Synthetic actor", "vector": "Synthetic interface", "steps": "前提：入口可达\n1. 合成流程\n目标：合成结果", "feasibility": nil, "rationale": "", "reference": nil}
	payload := Object{"title": "Synthetic preset scenario", "asset_id": asset["id"], "property": "integrity", "damage": "Synthetic outcome", "threat": "Synthetic threat", "paths": []any{path}}
	_, _, e = repo.Change(ctx, wid, 1, "preset-test-scenario", "record-scenario", payload, nil)
	if e != nil {
		t.Fatal(e)
	}
	ws, _ = repo.Get(ctx, wid)
	scenario := dObj(dList(ws["scenarios"])[0])
	saved := dObj(dList(scenario["paths"])[0])
	if saved["source_code"] != path["source_code"] || saved["vector"] != path["vector"] || saved["feasibility"] != nil {
		t.Fatal("preset did not persist as unrated editable path")
	}
	oldHash := AssessmentHash(ws, scenario)
	edit := Clone(scenario)
	dObj(dList(edit["paths"])[0])["steps"] = "工程师补充实际入口与步骤"
	_, _, e = repo.Change(ctx, wid, 2, "preset-test-edit", "record-scenario", edit, nil)
	if e != nil {
		t.Fatal(e)
	}
	ws, _ = repo.Get(ctx, wid)
	scenario = dObj(dList(ws["scenarios"])[0])
	saved = dObj(dList(scenario["paths"])[0])
	if saved["steps"] != "工程师补充实际入口与步骤" || saved["source_code"] != path["source_code"] || oldHash == AssessmentHash(ws, scenario) {
		t.Fatal("editing lost origin or assessment did not change")
	}
	before := Digest(ws)
	tables := Tables(ws, nil, nil)
	found := false
	for _, table := range tables {
		if table["name"] == "风险评估" {
			found = true
			columns := table["columns"].([]string)
			rows := table["rows"].([][]any)
			if len(rows) != 1 || len(rows[0]) != len(columns) || columns[len(columns)-3] != "路径来源编号 / 版本" || rows[0][len(columns)-3] != path["source_code"] {
				t.Fatal("missing template report provenance", table)
			}
		}
	}
	if !found || Digest(ws) != before {
		t.Fatal("report missing or mutates workspace")
	}
	html, e := Printable(ws, nil, nil)
	if e != nil || !strings.Contains(string(html), dStr(path["source_code"])) || !strings.Contains(string(html), "Synthetic actor") {
		t.Fatal("HTML missing origin", e)
	}
	// An expanded TARA report keeps project path numbering separate from shared template origin.
	ws["damage_scenarios"] = []any{Object{"id": "synthetic-damage", "description": "Synthetic", "asset_ids": []any{asset["id"]}, "properties": []any{"integrity"}, "impact": Object{}}}
	for _, table := range taraReportTables(ws) {
		if table["name"] == "攻击潜力工作产品" {
			rows := table["rows"].([][]any)
			columns := table["columns"].([]string)
			expected := RecordCodes(ws)("path", pathNumberKey(dStr(scenario["id"]), dStr(path["id"])))
			if len(rows) != 1 || rows[0][0] != expected || rows[0][len(columns)-1] != path["source_code"] {
				t.Fatal("template origin used as unique project path code", table)
			}
		}
	}
}
