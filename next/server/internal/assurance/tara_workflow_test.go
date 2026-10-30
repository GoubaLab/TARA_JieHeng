package assurance

import (
	"fmt"
	"strings"
	"testing"
)

func TestAttackFeasibilityStageRejectsInvalidScoresAndReplays(t *testing.T) {
	h, _, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := apiPrefix + "/workspaces/" + id
	imported := httpCall(h, "POST", base+"/tara-imports", httpGenericBody(t, taraPayload(taraFixture(t))), map[string]string{"If-Match": `"0"`, "Idempotency-Key": "workflow-import-001"})
	if imported.Code != 201 {
		t.Fatal(imported.Code, imported.Body.String())
	}
	path := base + "/scenarios/TS-001/attack-feasibility"
	valid := Object{"path_id": "AP-001", "method": "attack_potential", "potential": Object{"elapsed_time": 4, "expertise": 3, "knowledge": 3, "opportunity": 0, "equipment": 4}, "rationale": "按路径的前提、访问条件和所需资源分别判定。"}
	original := httpObject(t, httpCall(h, "GET", base, "", nil))
	for i, mutate := range []func(Object){
		func(v Object) { delete(v, "method") },
		func(v Object) { delete(v, "rationale") },
		func(v Object) { v["rationale"] = "  " },
		func(v Object) { delete(dObj(v["potential"]), "expertise") },
		func(v Object) { dObj(v["potential"])["elapsed_time"] = 999999 },
		func(v Object) { dObj(v["potential"])["equipment"] = 99 },
		func(v Object) { dObj(v["potential"])["knowledge"] = 1.5 },
		func(v Object) { v["feasibility"] = 4 },
		func(v Object) { v["attack_vector"] = "network" },
	} {
		body := Clone(valid)
		mutate(body)
		r := httpCall(h, "PUT", path, httpGenericBody(t, body), map[string]string{"If-Match": `"1"`, "Idempotency-Key": fmt.Sprintf("workflow-invalid-%02d", i)})
		if r.Code != 422 {
			t.Fatal(i, r.Code, r.Body.String())
		}
	}
	if got := httpObject(t, httpCall(h, "GET", base, "", nil)); Digest(got) != Digest(original) {
		t.Fatal("invalid rating altered workspace")
	}
	headers := map[string]string{"If-Match": `"1"`, "Idempotency-Key": "workflow-rate-001"}
	r := httpCall(h, "PUT", path, httpGenericBody(t, valid), headers)
	if r.Code != 200 || r.Header().Get("ETag") != `"2"` {
		t.Fatal(r.Code, r.Body.String())
	}
	replay := httpCall(h, "PUT", path, httpGenericBody(t, valid), headers)
	if replay.Code != 200 || replay.Body.String() != r.Body.String() {
		t.Fatal("rating replay failed")
	}
	stored := httpObject(t, httpCall(h, "GET", base, "", nil))
	s := dObj(dList(stored["scenarios"])[0])
	p := dObj(dList(s["paths"])[0])
	if dInt(p["feasibility"]) != 2 || p["feasibility_method"] != "attack_potential" {
		t.Fatal(p)
	}
	// The imported custom 10/14/20 bands remain authoritative for this project.
	if Digest(stored["method"]) != Digest(original["method"]) || Digest(stored["tara_import"]) != Digest(original["tara_import"]) {
		t.Fatal("rating changed method or source")
	}
	if dObj(dList(Review(stored)["evaluations"])[0])["decision_stale"] != true {
		t.Fatal("changed assessment did not invalidate decision")
	}
	// A changed attack path invalidates a completed rating on the server as well.
	s = Clone(s)
	p = dObj(dList(s["paths"])[0])
	p["steps"] = "新增前置条件和实施步骤"
	body := Object{}
	for k := range taraSchema("Scenario", dSchema("Scenario")) {
		if v, ok := s[k]; ok {
			body[k] = v
		}
	}
	changed, _, err := Apply(stored, "record-scenario", body)
	if err != nil {
		t.Fatal(err)
	}
	changedPath := dObj(dList(dObj(dList(changed["scenarios"])[0])["paths"])[0])
	if changedPath["feasibility"] != nil || Risk(dObj(dList(changed["scenarios"])[0]), dObj(changed["method"]))["level"] != nil {
		t.Fatal("changed path retained risk")
	}
	if stale := httpCall(h, "PUT", path, httpGenericBody(t, valid), map[string]string{"If-Match": `"1"`, "Idempotency-Key": "workflow-stale-001"}); stale.Code != 412 {
		t.Fatal("stale rating accepted", stale.Code)
	}
}

func TestAttackPotentialStandardMappingAndVectors(t *testing.T) {
	for _, test := range []struct {
		potential    Object
		total, level int
	}{
		{Object{"elapsed_time": 0, "expertise": 0, "knowledge": 0, "opportunity": 0, "equipment": 0}, 0, 4},
		{Object{"elapsed_time": 4, "expertise": 3, "knowledge": 3, "opportunity": 0, "equipment": 4}, 14, 3},
		{Object{"elapsed_time": 17, "expertise": 3, "knowledge": 0, "opportunity": 0, "equipment": 0}, 20, 2},
		{Object{"elapsed_time": 17, "expertise": 8, "knowledge": 0, "opportunity": 0, "equipment": 0}, 25, 1},
	} {
		f, total, ok := pathFeasibility(Object{"feasibility_method": "attack_potential", "potential": test.potential}, EffectiveMethod(DefaultMethod))
		if !ok || f != test.level || total != test.total {
			t.Fatal(f, total, ok)
		}
	}
	for vector, want := range map[string]int{"network": 4, "adjacent": 3, "local": 2, "physical": 1} {
		f, _, ok := pathFeasibility(Object{"feasibility_method": "attack_vector", "attack_vector": vector}, DefaultMethod)
		if !ok || f != want {
			t.Fatal(vector, f)
		}
	}
	if _, _, ok := pathFeasibility(Object{"feasibility_method": "attack_potential", "feasibility": 4}, DefaultMethod); ok {
		t.Fatal("missing factors counted as complete")
	}
}

func TestFunctionalCatalogPreservesContentAndRejectsDanglingReferences(t *testing.T) {
	ws := NewWorkspace("控制系统", "")
	item := Clone(dObj(ws["item"]))
	item["functions"] = []any{Object{"id": "FN-1", "name": "安全启动", "description": "在启动阶段校验固件完整性"}}
	ws, _, err := Apply(ws, "define-item", item)
	if err != nil {
		t.Fatal(err)
	}
	asset := Object{"name": "启动映像", "properties": []any{"integrity"}, "function_refs": []any{"missing"}}
	if _, _, e := Apply(ws, "record-asset", asset); e == nil {
		t.Fatal("unknown function accepted")
	}
	asset["function_refs"] = []any{"FN-1"}
	ws, _, err = Apply(ws, "record-asset", asset)
	if err != nil {
		t.Fatal(err)
	}
	legacyItem := Clone(item)
	delete(legacyItem, "functions")
	kept, _, err := Apply(ws, "define-item", legacyItem)
	if err != nil || len(dList(dObj(kept["item"])["functions"])) != 1 {
		t.Fatal("legacy item edit lost catalog", err)
	}
	item["functions"] = []any{}
	if _, _, e := Apply(ws, "define-item", item); e == nil {
		t.Fatal("referenced function removed")
	}
	item["functions"] = []any{Object{"id": "F", "name": "a"}, Object{"id": "F", "name": "b"}}
	if _, _, e := Apply(ws, "define-item", item); e == nil {
		t.Fatal("duplicate function accepted")
	}
	ranged := Clone(dObj(ws["item"]))
	ranged["purpose"] = "FR-019..FR-024 唤醒：硬线、网络与定时唤醒"
	ws["item"] = ranged
	if !functionIDs(ws)["FR-021"] || functionIDs(ws)["FR-025"] {
		t.Fatal("range definition did not resolve correctly")
	}
}

func TestTaraSeparatedStagesCompleteBeforeRiskAndReport(t *testing.T) {
	ws := NewWorkspace("分步评估", "")
	apply := func(action string, body Object) string {
		t.Helper()
		var id string
		var err error
		ws, id, err = Apply(ws, action, body)
		if err != nil {
			t.Fatal(action, err)
		}
		return id
	}
	apply("adopt-method", Clone(DefaultMethod))
	apply("define-item", Object{"functions": []any{Object{"id": "FN-001", "name": "控制命令校验", "description": "接收控制命令前验证来源和完整性"}}})
	assetID := apply("record-asset", Object{"name": "控制命令", "properties": []any{"integrity"}, "function_refs": []any{"FN-001"}})
	damageID := apply("record-damage", Object{"description": "非预期控制动作", "consequence": "功能中断", "asset_ids": []any{assetID}, "function_refs": []any{"FN-001"}, "properties": []any{"integrity"}, "impact": Object{}})
	scenarioID := apply("record-scenario", Object{"title": "伪造控制命令", "asset_id": assetID, "property": "integrity", "damage": "非预期控制动作", "damage_ids": []any{damageID}, "threat": "未经授权的命令被执行", "paths": []any{}})
	scenario := func() Object { return dObj(dList(ws["scenarios"])[0]) }
	if Risk(scenario(), dObj(ws["method"]))["level"] != nil {
		t.Fatal("missing impact and path produced risk")
	}
	damage := Clone(dObj(dList(ws["damage_scenarios"])[0]))
	damage["impact"] = Object{"safety": 1, "financial": 1, "operational": 2, "privacy": 0, "rationale": "各维度按照损害后果和场景判定"}
	apply("record-damage", damage)
	body := Clone(scenario())
	body["paths"] = []any{Object{"id": "p", "name": "发送伪造命令", "steps": "取得邻接网络访问，再发送控制命令"}}
	apply("record-scenario", body)
	if Risk(scenario(), dObj(ws["method"]))["level"] != nil {
		t.Fatal("unrated path produced risk")
	}
	decision := Object{"scenario_id": scenarioID, "treatment": "retain", "rationale": "已评审风险", "owner": "负责人", "claim": "范围内接受"}
	if _, _, err := Apply(ws, "decide-treatment", decision); err == nil {
		t.Fatal("unrated risk accepted")
	}
	apply("rate-attack-path", Object{"scenario_id": scenarioID, "path_id": "p", "method": "attack_vector", "attack_vector": "adjacent", "rationale": "攻击需要进入同一受限网络"})
	if dInt(Risk(scenario(), dObj(ws["method"]))["level"]) != 3 {
		t.Fatal("complete assessment not calculated")
	}
	apply("decide-treatment", decision)
	if RecordCodes(ws)("damage", damageID) != "DM-001" {
		t.Fatal("damage lacks stable business code")
	}
	reportText := fmt.Sprint(Tables(ws, nil, nil))
	if !strings.Contains(reportText, "接收控制命令前验证来源和完整性") || !strings.Contains(reportText, "主要攻击向量") || !strings.Contains(reportText, "各维度按照损害后果和场景判定") {
		t.Fatal("report lost workflow evidence")
	}
	workProducts := fmt.Sprint(taraReportTables(ws))
	if !strings.Contains(workProducts, "DM-001") || !strings.Contains(workProducts, "AP-001") || strings.Contains(workProducts, damageID) || strings.Contains(workProducts, scenarioID) {
		t.Fatal("report must display stable business references")
	}
	body = Clone(scenario())
	dObj(dList(body["paths"])[0])["steps"] = "需要物理接触后才能发送命令"
	apply("record-scenario", body)
	if Risk(scenario(), dObj(ws["method"]))["level"] != nil {
		t.Fatal("changed path retained prior rating")
	}
	if _, _, err := Apply(ws, "decide-treatment", decision); err == nil {
		t.Fatal("changed path allowed decision")
	}
}
