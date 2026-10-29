package assurance

import (
	"strings"
	"testing"
)

// This fixture is authored for the local product; it contains no imported library records.
func reportWorkspace(t *testing.T) Object {
	t.Helper()
	ws := NewWorkspace("<script>通用核对项目</script>", "=1+1")
	item, err := ValidatePayload("define-item", Object{"domain": "robotics", "purpose": "同步设备状态", "boundary": "测试设备与测试服务", "environment": "测试网络", "assumptions": "仅测试数据", "owner": "测试人员"})
	if err != nil {
		t.Fatal(err)
	}
	ws["item"] = item
	architecture := genericArchitecture(t)
	node := dObj(dList(architecture["nodes"])[0])
	node["entityType"] = "device"
	node["operating_system"] = Object{"family": "linux", "name": "测试发行版", "version": "1", "architecture": "arm64", "notes": "原创测试环境"}
	node["children"] = []any{Object{"id": "software_local", "kind": "software", "name": "状态应用", "description": "本地合成示例", "reference": "systems:987654321"}}
	dObj(dList(architecture["nodes"])[1])["entityType"] = "cloud"
	dObj(dList(architecture["flows"])[0])["communication"] = Object{"medium": "wireless", "technology": "BLE", "range": "near_field", "purpose": "状态同步", "security_properties": []any{"integrity"}}
	ws["architecture"], err = ValidatePayload("replace-architecture", architecture)
	if err != nil {
		t.Fatal(err)
	}
	method := EffectiveMethod(DefaultMethod)
	method["risk_calculation"] = "product"
	method["impact_aggregation"] = "weighted_mean"
	dObj(method["impact_dimensions"])["safety"] = "人员影响"
	dObj(dList(method["impact_levels"])[2])["label"] = "显著影响"
	dObj(dList(method["impact_levels"])[2])["description"] = "原创影响判定锚点"
	ws["method"], ws["method_confirmed"] = method, true
	ws["assets"] = []any{Object{"id": "asset_local", "name": "状态记录", "description": "原创测试资产", "node_ids": []any{"device"}, "properties": []any{"integrity"}, "reference": "systems:987654321"}}
	scenario := genericScenario()
	scenario["title"] = "伪造状态消息"
	scenario["property"] = "integrity"
	scenario["damage"] = "测试状态与实际不一致"
	scenario["threat"] = "测试发送方提供错误状态"
	scenario["references"] = []any{}
	path := dObj(dList(scenario["paths"])[0])
	path["name"] = "测试消息路径"
	path["reference"] = nil
	ws["scenarios"] = []any{scenario}
	ws["regulation_refs"] = []any{"gb:987654321"}
	return ws
}

func TestReportsGenericMetadataAlgorithmsAndWithdrawnReferences(t *testing.T) {
	c := libraryCatalog(t)
	ws := reportWorkspace(t)
	before := Digest(ws)
	tables := Tables(ws, nil, c)
	if len(tables) != 17 {
		t.Fatal("expected 17 work-product tables")
	}
	encoded, err := CanonicalJSON(tables)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{"机器人", "Linux", "arm64", "通信介质：无线", "BLE", "近场", "状态同步", "完整性", "人员影响", "显著影响", "原创影响判定锚点", "向上取整", "阈值上限", "3、6、9、12", "引用已失效", "符合性依据需重新核对"} {
		if !strings.Contains(text, want) {
			t.Errorf("report missing %q", want)
		}
	}
	methodRows := libList(tables[3]["rows"])
	libraryEqual(t, methodRows[0], []any{"可忽略", 1, 1, 1, 2})
	if Digest(ws) != before {
		t.Fatal("report changed persisted data")
	}
}
