package assurance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Synthetic, non-customer fixture exercises the public import contract.
func taraFixture(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheets := map[string]map[int][]any{
		"概述、方法与依据": {1: {"IL = S + F + 0.1×O + 0.1×P；0、10、100、1000；20-99；100-999；AP 0-9高；10-13中；14-19低；≥20极低"}},
		"相关项定义":    {4: {"相关项名称", "合成测试产品"}, 5: {"功能范围", "FR-001"}, 6: {"运行模式", "运行"}, 7: {"分析边界", "组件"}},
		"资产与损害场景":  {3: {"资产编号", "资产", "类型", "分组", "关联功能", "安全属性"}, 4: {"AS-001", "测试命令", "数据", "消息", "FR-001", "完整性"}, 5: {"AS-002", "测试日志", "数据", "记录", "FR-002", "完整性、不可否认性"}, 7: {"DS-001", "AS-001、AS-002", "FR-001", "测试损害", "测试后果", "完整性", 10, 10, 0, 0, 20, "中等"}},
		"威胁场景":     {3: {"TS", "威胁描述", "攻击原因", "STRIDE", "受损属性", "关联资产", "关联DS", "关联路径"}, 4: {"TS-001", "测试篡改", "权限错误", "T", "完整性", "AS-001、AS-002", "DS-001", "AP-001"}},
		"攻击路径":     {3: {"AP", "TS", "攻击者", "向量", "攻击步骤", "ET", "Ex", "KoI", "WoO", "Eq", "AP值", "可行性"}, 4: {"AP-001", "TS-001", "测试攻击者", "本地", "连接、发送、触发", 2, 2, 2, 2, 2, 10, "中"}},
		"风险评估与处置":  {3: {"TS", "影响", "可行性", "初始风险", "处置", "控制措施", "目标", "CAL", "残余可行性", "残余风险", "接受状态", "处置说明"}, 4: {"TS-001", "2 中等", "中", 3, "降低", "鉴权", "CSG-01", "CAL2", "低", 2, "条件接受", "需验证"}},
		"网络安全目标":   {3: {"目标编号", "目标内容", "关联TS", "最高风险", "CAL", "控制主题"}, 4: {"CSG-01", "防止非授权修改", "TS-001", 3, "CAL2", "鉴权"}},
		"网络安全需求":   {3: {"需求编号", "所属目标", "需求内容", "验证方法", "分配专业", "CAL", "客户需求编号"}, 4: {"CSR-001", "CSG-01", "检查身份", "负向用例", "软件", "CAL2", "SEC-BMS-001"}},
		"风险接受声明":   {4: {"TS", "影响", "处置", "初始风险", "残余风险", "接受结论", "残余依据", "客户确认", "确认日期", "声明人"}, 5: {"TS-001", "中等", "降低", 3, 2, "条件接受", "控制经验证后"}},
		"额外附录":     {1: {"任意补充内容", "=1+1"}},
	}
	for name, rows := range sheets {
		f.NewSheet(name)
		for n, values := range rows {
			if e := f.SetSheetRow(name, fmt.Sprintf("A%d", n), &values); e != nil {
				t.Fatal(e)
			}
		}
	}
	f.DeleteSheet("Sheet1")
	b, e := f.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func taraPayload(data []byte) Object {
	return Object{"filename": "synthetic.xlsx", "content_base64": base64.StdEncoding.EncodeToString(data)}
}
func TestTaraFullImportRoundTripAndRecalculation(t *testing.T) {
	data := taraFixture(t)
	original := NewWorkspace("test", "")
	ws, _, e := Apply(original, "import-tara", taraPayload(data))
	if e != nil {
		t.Fatal(e)
	}
	if len(dList(ws["assets"])) != 2 || len(dList(ws["damage_scenarios"])) != 1 || len(dList(ws["security_goals"])) != 1 || len(dList(dObj(ws["tara_import"])["sheets"])) != 10 {
		t.Fatal("lost work products")
	}
	source := dObj(ws["tara_import"])
	got, _ := base64.StdEncoding.DecodeString(dStr(source["original_base64"]))
	if !bytes.Equal(got, data) || source["sha256"] != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("source changed")
	}
	if len(dList(source["diagnostics"])) > 0 {
		t.Fatal(source["diagnostics"])
	}
	review := Review(ws)
	evaluation := dObj(dList(review["evaluations"])[0])
	if dInt(evaluation["level"]) != 3 || dInt(evaluation["residual"]) != 2 || dBool(evaluation["decision_stale"]) || dBool(review["ready"]) {
		t.Fatal(evaluation, review["ready"])
	}
	if _, _, e = Apply(ws, "import-tara", taraPayload(data)); e == nil {
		t.Fatal("overwrite accepted")
	}
	// Moving/resaving a catalog graph must not generate duplicate assets.
	saved, _, e := Apply(ws, "replace-architecture", dObj(ws["architecture"]))
	if e != nil || len(dList(saved["assets"])) != 2 {
		t.Fatal("catalog resave", e)
	}
	damage := Clone(dObj(dList(ws["damage_scenarios"])[0]))
	dObj(damage["impact"])["safety"] = 3
	changed, _, e := Apply(ws, "record-damage", damage)
	if e != nil {
		t.Fatal(e)
	}
	ev := dObj(dList(Review(changed)["evaluations"])[0])
	if dInt(ev["level"]) != 5 || !dBool(ev["decision_stale"]) {
		t.Fatal("linked damage did not invalidate decision", ev)
	}
	scenario := Clone(dObj(dList(ws["scenarios"])[0]))
	dObj(dObj(dList(scenario["paths"])[0])["potential"])["equipment"] = 12
	changed, _, e = Apply(ws, "record-scenario", scenario)
	if e != nil {
		t.Fatal(e)
	}
	ev = dObj(dList(Review(changed)["evaluations"])[0])
	if dInt(ev["feasibility"]) != 1 || !dBool(ev["decision_stale"]) {
		t.Fatal("AP factors not applied", ev)
	}
	// An edit to a secondary asset is part of the assessment fingerprint.
	asset := Clone(dObj(dList(ws["assets"])[1]))
	asset["description"] = "updated secondary asset"
	changed, _, e = Apply(ws, "record-asset", asset)
	if e != nil {
		t.Fatal(e)
	}
	if !dBool(dObj(dList(Review(changed)["evaluations"])[0])["decision_stale"]) {
		t.Fatal("secondary asset ignored")
	}
	report, e := Excel(ws, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	book, e := excelize.OpenReader(bytes.NewReader(report))
	if e != nil {
		t.Fatal(e)
	}
	defer book.Close()
	count := 0
	for _, name := range book.GetSheetList() {
		if strings.HasPrefix(name, "原表") {
			count++
		}
	}
	if count != 10 {
		t.Fatal("source sheets missing from report", count)
	}
	if len(dList(original["assets"])) != 0 {
		t.Fatal("original mutated")
	}
}
func TestTaraRejectsBrokenWorkbooksAtomically(t *testing.T) {
	for _, tc := range []struct{ name, sheet, cell, value string }{{"header", "威胁场景", "A3", "wrong"}, {"reference", "威胁场景", "G4", "DS-999"}, {"score", "风险评估与处置", "D4", "1"}, {"method", "概述、方法与依据", "A1", "different method"}, {"path", "攻击路径", "B4", "TS-999"}, {"potential", "攻击路径", "G4", "999999"}} {
		t.Run(tc.name, func(t *testing.T) {
			f, e := excelize.OpenReader(bytes.NewReader(taraFixture(t)))
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			f.SetCellStr(tc.sheet, tc.cell, tc.value)
			b, _ := f.WriteToBuffer()
			original := NewWorkspace("test", "")
			before := Digest(original)
			if _, _, e := Apply(original, "import-tara", taraPayload(b.Bytes())); e == nil {
				t.Fatal("invalid input accepted")
			}
			if Digest(original) != before {
				t.Fatal("partial write")
			}
		})
	}
}
func TestTaraHTTPImportContract(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	path := "/api/v1/workspaces/" + id
	data := taraFixture(t)
	payload, _ := json.Marshal(taraPayload(data))
	p := httpCall(h, "POST", path+"/tara-import-previews", string(payload), nil)
	if p.Code != 200 {
		t.Fatal(p.Code, p.Body.String())
	}
	before, _ := repo.Get(context.Background(), id)
	if dInt(before["revision"]) != 0 {
		t.Fatal("preview wrote data")
	}
	noPrecondition := httpCall(h, "POST", path+"/tara-imports", string(payload), nil)
	if noPrecondition.Code != 428 {
		t.Fatal(noPrecondition.Code)
	}
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "tara-import-test-01"}
	created := httpCall(h, "POST", path+"/tara-imports", string(payload), headers)
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body.String())
	}
	replay := httpCall(h, "POST", path+"/tara-imports", string(payload), headers)
	if replay.Code != 201 || replay.Body.String() != created.Body.String() {
		t.Fatal("import replay")
	}
	location := created.Header().Get("Location")
	resource := httpCall(h, "GET", location, "", nil)
	if resource.Code != 200 || strings.Contains(resource.Body.String(), "original_base64") {
		t.Fatal("resource projection")
	}
	source := httpCall(h, "GET", location+"/source", "", nil)
	if source.Code != 200 || !bytes.Equal(source.Body.Bytes(), data) {
		t.Fatal("source changed")
	}
	get := httpCall(h, "GET", path, "", nil)
	if get.Code != 200 || strings.Contains(get.Body.String(), "original_base64") {
		t.Fatal("workspace projection")
	}
	resource = httpCall(h, "GET", path+"/damage-scenarios/DS-001", "", nil)
	if resource.Code != 200 {
		t.Fatal("damage GET")
	}
	if httpCall(h, "GET", path+"/tara-imports/wrong/source", "", nil).Code != 404 {
		t.Fatal("wrong source id")
	}
}
func TestLocalTaraWorkbook(t *testing.T) {
	path := os.Getenv("TARA_IMPORT_VERIFY_FILE")
	if path == "" {
		t.Skip("optional private workbook check")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	ws, e := buildTaraImport(Object{"filename": filepath.Base(path), "content_base64": base64.StdEncoding.EncodeToString(data)})
	if e != nil {
		t.Fatal(e)
	}
	source := dObj(ws["tara_import"])
	if len(dList(source["diagnostics"])) != 0 {
		t.Fatal(source["diagnostics"])
	}
	for _, key := range []string{"assets", "damage_scenarios", "scenarios", "security_goals", "requirements", "decisions"} {
		if len(dList(ws[key])) == 0 {
			t.Fatal(key, len(dList(ws[key])))
		}
	}
	if len(dList(source["sheets"])) == 0 {
		t.Fatal("sheets")
	}
	_, e = Excel(ws, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("source sheets, work products and calculation parity verified")
}

func TestTaraThresholdEdgesAndValidationGate(t *testing.T) {
	method := taraMethod()
	for _, tc := range []struct{ score, level int }{{9, 4}, {10, 3}, {13, 3}, {14, 2}, {19, 2}, {20, 1}} {
		f, total, ok := pathFeasibility(Object{"potential": Object{"elapsed_time": tc.score, "expertise": 0, "knowledge": 0, "opportunity": 0, "equipment": 0}}, method)
		if !ok || total != tc.score || f != tc.level {
			t.Fatal(tc, f, total)
		}
	}
	// An operational-only severe score must retain the source's severity floor.
	impact := Object{"safety": 0, "financial": 0, "operational": 3, "privacy": 0}
	level, score, ok := bandedImpact(impact, method)
	if !ok || level != 2 || dInt(score) != 100 {
		t.Fatal(level, score)
	}
	ws, _, e := Apply(NewWorkspace("test", ""), "import-tara", taraPayload(taraFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	ws["method_confirmed"] = true
	decision := Clone(dObj(dList(ws["decisions"])[0]))
	delete(decision, "assessment_hash")
	delete(decision, "goal_hash")
	decision["owner"] = "test reviewer"
	decision["residual_status"] = "validated"
	if _, _, e := Apply(ws, "decide-treatment", decision); e == nil {
		t.Fatal("unverified target accepted")
	}
	decision["residual_status"] = "target"
	if _, _, e := Apply(ws, "decide-treatment", decision); e != nil {
		t.Fatal(e)
	}
}

func TestTaraReferencesWithoutImportedCollections(t *testing.T) {
	ws, _, err := Apply(NewWorkspace("synthetic", ""), "import-tara", taraPayload(taraFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	delete(ws, "damage_scenarios")
	delete(ws, "security_goals")
	delete(ws, "tara_import")
	ws["decisions"], ws["requirements"] = []any{}, []any{}
	s := dObj(dList(ws["scenarios"])[0])
	delete(s, "damage_ids")
	for _, tc := range []struct {
		name   string
		change func(Object)
	}{
		{"secondary asset", func(s Object) { s["asset_ids"] = []any{s["asset_id"], "missing"} }},
		{"damage", func(s Object) { s["damage_ids"] = []any{"missing"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := Clone(s)
			tc.change(payload)
			if _, _, e := Apply(ws, "record-scenario", payload); e == nil {
				t.Fatal("unresolved relationship accepted")
			}
		})
	}
	req := Object{"scenario_id": s["id"], "statement": "test", "allocation": "team", "verification": "test", "scenario_ids": []any{s["id"], "missing"}}
	if _, _, e := Apply(ws, "specify-requirement", req); e == nil {
		t.Fatal("unresolved secondary scenario accepted")
	}
}
func TestTaraCoefficientDamageSelectionUsesWeightedTotal(t *testing.T) {
	ws, _, e := Apply(NewWorkspace("synthetic", ""), "import-tara", taraPayload(taraFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	ws["method"] = coefficientMethod()
	dObj(ws["method"])["attack_potential_thresholds"] = []any{10, 14, 20}
	d := dObj(dList(ws["damage_scenarios"])[0])
	d["impact"] = Object{"safety": 0, "financial": 0, "operational": 3, "privacy": 0, "rationale": "test"}
	other := Clone(d)
	other["id"], other["source_code"] = "DS-002", "DS-002"
	other["impact"] = Object{"safety": 2, "financial": 0, "operational": 0, "privacy": 0, "rationale": "test"}
	ws["damage_scenarios"] = append(dList(ws["damage_scenarios"]), other)
	s := dObj(dList(ws["scenarios"])[0])
	s["damage_ids"] = []any{"DS-001", "DS-002"}
	if e := refreshTaraLinks(ws); e != nil {
		t.Fatal(e)
	}
	risk := Risk(s, dObj(ws["method"]))
	if dInt(risk["impact"]) != 2 {
		t.Fatalf("selected raw max dimension instead of total: %v", risk)
	}
	s["damage_ids"] = []any{"DS-002", "DS-001"}
	if e := refreshTaraLinks(ws); e != nil {
		t.Fatal(e)
	}
	if dInt(Risk(s, dObj(ws["method"]))["impact"]) != 2 {
		t.Fatal("order-dependent impact")
	}
}
func TestTaraGoalsAndEvidenceRequireCurrentContext(t *testing.T) {
	ws, _, e := Apply(NewWorkspace("synthetic", ""), "import-tara", taraPayload(taraFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	ws["method_confirmed"] = true
	evidence := Object{"requirement_id": "CSR-001", "result": "pass", "reference": "synthetic://test-run", "summary": "synthetic verification", "reviewer": "test reviewer"}
	ws, _, e = Apply(ws, "record-evidence", evidence)
	if e != nil {
		t.Fatal(e)
	}
	decision := Clone(dObj(dList(ws["decisions"])[0]))
	delete(decision, "assessment_hash")
	delete(decision, "goal_hash")
	decision["owner"] = "test reviewer"
	decision["residual_status"] = "validated"
	ws, _, e = Apply(ws, "decide-treatment", decision)
	if e != nil {
		t.Fatal(e)
	}
	if !scenarioVerified(ws, "TS-001") {
		t.Fatal("valid evidence not recognized")
	}
	g := Clone(dObj(dList(ws["security_goals"])[0]))
	g["statement"] = "changed security goal"
	changed, _, e := Apply(ws, "record-goal", g)
	if e != nil {
		t.Fatal(e)
	}
	review := Review(changed)
	ev := dObj(dList(review["evaluations"])[0])
	if !dBool(ev["decision_stale"]) || ev["residual_status"] != "target" || dObj(dList(review["requirements"])[0])["status"] != "stale" {
		t.Fatal("goal changes left approvals current", review)
	}
	if !strings.Contains(dStr(dObj(dList(changed["decisions"])[0])["goal"]), "changed security goal") {
		t.Fatal("copied goal text stale")
	}
	if _, _, e = Apply(changed, "decide-treatment", decision); e == nil {
		t.Fatal("old passing evidence accepted after goal edit")
	}
	// A later failed verification must also revoke the effective validated state.
	evidence["result"] = "fail"
	changed, _, e = Apply(ws, "record-evidence", evidence)
	if e != nil {
		t.Fatal(e)
	}
	if dObj(dList(Review(changed)["evaluations"])[0])["residual_status"] != "target" {
		t.Fatal("failed retest still displays validated")
	}
	// Moving a node is presentation-only and must preserve the verification context.
	graph := Clone(dObj(ws["architecture"]))
	graph["equal_spacing"] = true
	changed, _, e = Apply(ws, "replace-architecture", graph)
	if e != nil || !scenarioVerified(changed, "TS-001") {
		t.Fatal("layout edit invalidated evidence", e)
	}
}
func TestTaraCatalogModeSurvivesLegacyArchitectureWrite(t *testing.T) {
	ws, _, e := Apply(NewWorkspace("synthetic", ""), "import-tara", taraPayload(taraFixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	graph := Clone(dObj(ws["architecture"]))
	delete(graph, "asset_generation")
	next, _, e := Apply(ws, "replace-architecture", graph)
	if e != nil || dObj(next["architecture"])["asset_generation"] != "catalog" || len(dList(next["assets"])) != 2 {
		t.Fatal("legacy save reset asset mode", e)
	}
}

func TestTaraExportRetainsEditedAssetMetadata(t *testing.T) {
	ws, _, err := Apply(NewWorkspace("test", ""), "import-tara", taraPayload(taraFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	asset := dObj(dList(ws["assets"])[0])
	asset["asset_type"], asset["group"], asset["function_refs"] = "软件", "更新分组", []any{"FR-NEW"}
	data, err := Excel(ws, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	rows, err := book.GetRows("资产识别")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if len(row) == 10 && row[6] == dStr(asset["source_code"]) {
			if row[7] != "软件" || row[8] != "更新分组" || row[9] != "FR-NEW" {
				t.Fatal(row)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("edited asset metadata missing from export")
	}
	delete(ws, "tara_import")
	delete(ws, "damage_scenarios")
	delete(ws, "security_goals")
	if len(taraReportTables(ws)) != 6 {
		t.Fatal("hand-authored extended work products missing")
	}
}
