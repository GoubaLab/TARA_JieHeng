package assurance

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func httpGenericBody(t *testing.T, value Object) string {
	t.Helper()
	data, err := CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHTTPGenericModelPersistsAndUsesSelectedRiskAlgorithm(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	workspaceID := httpCreate(t, h)
	base := "/api/v1/workspaces/" + workspaceID
	revision := 0
	write := func(method, path string, payload Object) Object {
		t.Helper()
		headers := map[string]string{"If-Match": fmt.Sprintf(`"%d"`, revision), "Idempotency-Key": fmt.Sprintf("generic-flow-%03d", revision)}
		response := httpCall(h, method, base+path, httpGenericBody(t, payload), headers)
		want := 200
		if method == http.MethodPost {
			want = 201
		}
		if response.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, response.Code, response.Body.String())
		}
		revision++
		if response.Header().Get("ETag") != fmt.Sprintf(`"%d"`, revision) {
			t.Fatal("mutation ETag does not follow revision")
		}
		return httpObject(t, response)
	}
	write("PUT", "/item", Object{"domain": "robotics", "purpose": "原创机器人通信测试", "boundary": "设备与服务", "environment": "测试网络", "assumptions": "本地合成数据", "owner": "测试负责人"})
	architecture := genericArchitecture(t)
	dObj(dList(architecture["nodes"])[0])["entityType"] = "device"
	dObj(dList(architecture["nodes"])[0])["operating_system"] = Object{"family": "rtos", "name": "测试实时系统", "version": "2.1", "architecture": "arm64", "notes": "独立测试描述"}
	dObj(dList(architecture["nodes"])[1])["entityType"] = "cloud"
	dObj(dList(architecture["flows"])[0])["communication"] = Object{"medium": "wireless", "technology": "BLE", "range": "near_field", "purpose": "传输状态消息", "security_properties": []any{"integrity", "authenticity"}}
	write("PUT", "/architecture", architecture)
	method := EffectiveMethod(DefaultMethod)
	method["name"] = "项目自定义方法"
	method["rationale"] = "按原创测试锚点和相对权重评估"
	method["impact_aggregation"] = "weighted_mean"
	method["impact_weights"] = Object{"safety": 0, "financial": 25, "operational": 75, "privacy": 0}
	method["risk_calculation"] = "product"
	dObj(method["impact_dimensions"])["operational"] = "业务连续性"
	dObj(dList(method["risk_levels"])[1])["description"] = "项目二级风险判定说明"
	write("PUT", "/assessment-method", method)
	asset := write("POST", "/assets", Object{"name": "状态数据", "properties": []any{"integrity"}, "node_ids": []any{"device"}})
	scenario := write("POST", "/scenarios", Object{"title": "状态消息异常", "asset_id": asset["entity_id"], "property": "integrity", "damage": "测试状态不一致", "threat": "测试消息被更改", "impact": Object{"safety": nil, "financial": 1, "operational": 2, "privacy": nil, "rationale": "零权重维度不适用；其余依据项目锚点"}, "paths": []any{Object{"name": "消息测试路径", "steps": "本地合成步骤", "feasibility": 2, "rationale": "测试可行性依据"}}})
	review := httpCall(h, "GET", base+"/review", "", nil)
	if review.Code != 200 {
		t.Fatal(review.Body.String())
	}
	evaluation := dObj(dList(httpObject(t, review)["evaluations"])[0])
	if dInt(evaluation["impact"]) != 2 || dInt(evaluation["level"]) != 2 {
		t.Fatalf("expected ceil(1*.25+2*.75)=2 and product 6 => risk 2: %v", evaluation)
	}
	write("PUT", "/scenarios/"+dStr(scenario["entity_id"])+"/treatment", Object{"treatment": "retain", "rationale": "测试保留决定", "owner": "测试负责人", "claim": "测试范围内接受残余风险", "residual_impact": 0, "residual_feasibility": 4, "residual_rationale": "独立残余风险依据"})
	review = httpCall(h, "GET", base+"/review", "", nil)
	evaluation = dObj(dList(httpObject(t, review)["evaluations"])[0])
	if dInt(evaluation["residual"]) != 2 || dBool(evaluation["decision_stale"]) {
		t.Fatalf("residual must use product 4 => risk 2, not matrix 1: %v", evaluation)
	}
	before := httpCall(h, "GET", base, "", nil)
	if before.Code != 200 {
		t.Fatal(before.Body.String())
	}
	stored := httpObject(t, before)
	if dObj(stored["item"])["domain"] != "robotics" {
		t.Fatal("domain did not persist")
	}
	graph := dObj(stored["architecture"])
	dEqual(t, dObj(dList(graph["nodes"])[0])["operating_system"], dObj(dList(architecture["nodes"])[0])["operating_system"])
	dEqual(t, dObj(dList(graph["flows"])[0])["communication"], dObj(dList(architecture["flows"])[0])["communication"])
	dEqual(t, stored["method"], method)
	database, err := repo.Get(context.Background(), workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	dEqual(t, database, stored)
	// A stale editor cannot overwrite persisted OS/communication or its history.
	stale := httpCall(h, "PUT", base+"/architecture", httpGenericBody(t, architecture), map[string]string{"If-Match": `"2"`, "Idempotency-Key": "generic-stale-write"})
	if stale.Code != 412 || stale.Header().Get("ETag") != fmt.Sprintf(`"%d"`, revision) {
		t.Fatalf("stale save must fail with current ETag: %d %s", stale.Code, stale.Body.String())
	}
	after := httpCall(h, "GET", base, "", nil)
	if before.Body.String() != after.Body.String() {
		t.Fatal("stale save altered project")
	}
	_, count, err := repo.Events(context.Background(), workspaceID, 0, 100)
	if err != nil || count != revision+1 {
		t.Fatalf("unexpected history after rejected save: %d %v", count, err)
	}
}

func TestHTTPGenericModelRejectsInvalidConfigurationsAtomically(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := "/api/v1/workspaces/" + id
	badWeight := Clone(DefaultMethod)
	badWeight["impact_aggregation"] = "weighted_mean"
	badWeight["impact_weights"] = Object{"safety": 0, "financial": 0, "operational": 0, "privacy": 0}
	badThresholds := Clone(DefaultMethod)
	badThresholds["risk_calculation"] = "product"
	badThresholds["product_thresholds"] = []any{3, 6, 6, 12}
	badInteger := Clone(DefaultMethod)
	badInteger["impact_weights"] = Object{"safety": 25.0}
	badLevels := Clone(DefaultMethod)
	badLevels["risk_levels"] = methodLevels("低", "高")
	badOS := genericArchitecture(t)
	dObj(dList(badOS["nodes"])[0])["operating_system"] = Object{"family": "invalid"}
	badOwner := genericArchitecture(t)
	node := dObj(dList(badOwner["nodes"])[0])
	node["kind"] = "boundary"
	node["entityType"] = nil
	node["operating_system"] = Object{}
	badCommunication := genericArchitecture(t)
	dObj(dList(badCommunication["flows"])[0])["communication"] = Object{"medium": "wireless", "security_properties": []any{"encrypted"}}
	badUnknown := genericArchitecture(t)
	dObj(dList(badUnknown["flows"])[0])["communication"] = Object{"infer_security": true}
	for i, test := range []struct {
		path  string
		value Object
	}{{"item", Object{"domain": "unknown"}}, {"assessment-method", badWeight}, {"assessment-method", badThresholds}, {"assessment-method", badInteger}, {"assessment-method", badLevels}, {"architecture", badOS}, {"architecture", badOwner}, {"architecture", badCommunication}, {"architecture", badUnknown}} {
		response := httpCall(h, "PUT", base+"/"+test.path, httpGenericBody(t, test.value), map[string]string{"If-Match": `"0"`, "Idempotency-Key": fmt.Sprintf("generic-invalid-%03d", i)})
		if response.Code != 422 || response.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatalf("invalid %s must return 422: %d %s", test.path, response.Code, response.Body.String())
		}
	}
	ws, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if dInt(ws["revision"]) != 0 {
		t.Fatal("invalid configuration changed project")
	}
	_, count, err := repo.Events(context.Background(), id, 0, 100)
	if err != nil || count != 1 {
		t.Fatal("invalid configuration changed history")
	}
}
