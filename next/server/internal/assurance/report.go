package assurance

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

var reportProperties = map[string]string{"confidentiality": "机密性", "integrity": "完整性", "availability": "可用性", "authenticity": "真实性", "authorization": "授权", "non_repudiation": "不可抵赖性", "freshness": "新鲜性", "privacy": "隐私性"}
var reportImpacts = []string{"可忽略", "中等", "重大", "严重"}
var reportFeasibility = []string{"很低", "低", "中", "高"}
var reportTreatments = map[string]string{"reduce": "降低", "avoid": "规避", "share": "分担", "retain": "保留"}
var reportActions = map[string]string{"open-workspace": "建立工作区", "define-item": "更新相关项定义", "replace-architecture": "保存系统架构", "record-asset": "记录资产", "record-scenario": "更新风险场景", "decide-treatment": "确认风险处置", "specify-requirement": "更新安全需求", "record-evidence": "记录验证证据", "configure-numbering": "更新编号规则", "adopt-method": "确认评估方法", "seal-baseline": "建立评审基线", "select-regulations": "选择项目法规条款", "apply-reference-selection": "批量选用参考知识", "import-tara": "导入完整 TARA", "record-damage": "更新损害场景", "record-goal": "更新安全目标"}

func reportImpact(v any) string {
	if v == nil {
		return "待评估"
	}
	i := libInt(v)
	if i < 0 || i >= len(reportImpacts) {
		return "待评估"
	}
	return reportImpacts[i]
}
func reportFeasible(v any) string {
	if v == nil {
		return "待评估"
	}
	i := libInt(v) - 1
	if i < 0 || i >= len(reportFeasibility) {
		return "待评估"
	}
	return reportFeasibility[i]
}
func reportIndex(rows any, key string) map[string]Object {
	out := map[string]Object{}
	for _, v := range libRows(rows) {
		out[libStr(v[key])] = v
	}
	return out
}

// Tables projects the aggregate into independent, static report work products.
func Tables(ws Object, journal []Object, catalog *Catalog) []Object {
	code := RecordCodes(ws)
	review := Review(ws)
	architecture := libObj(ws["architecture"])
	item := libObj(ws["item"])
	method := EffectiveMethod(libObj(ws["method"]))
	impactLabel := func(value any) string { return reportLevel(method, "impact_levels", value, 0) }
	feasibilityLabel := func(value any) string { return reportLevel(method, "feasibility_levels", value, 1) }
	coefficient := method["impact_aggregation"] == "coefficient_sum"
	impactValue := func(value any) string {
		if coefficient && value != nil {
			return libStr(value)
		}
		return impactLabel(value)
	}
	dimensions := libObj(method["impact_dimensions"])
	assets := reportIndex(ws["assets"], "id")
	nodes := reportIndex(architecture["nodes"], "id")
	decisions := reportIndex(ws["decisions"], "scenario_id")
	evaluations := reportIndex(review["evaluations"], "scenario_id")
	output := []Object{}
	add := func(name string, columns []string, rows [][]any, widths []int, note string) {
		if rows == nil {
			rows = [][]any{}
		}
		output = append(output, Object{"name": name, "columns": columns, "rows": rows, "widths": widths, "note": note})
	}
	add("签批封面", []string{"项目", "内容"}, [][]any{{"项目名称", ws["title"]}, {"项目说明", ws["description"]}, {"报告修订", ws["revision"]}, {"负责人", item["owner"]}, {"编制签字", ""}, {"复核签字", ""}, {"批准签字", ""}}, []int{24, 95}, "界衡网络安全工作产品报告")
	rows := [][]any{}
	for i := len(journal) - 1; i >= 0; i-- {
		e := journal[i]
		action := libStr(e["action"])
		label := reportActions[action]
		if label == "" {
			label = action
		}
		rows = append(rows, []any{e["revision"], e["at"], label, libStr(libObj(e["body"])["entity_id"])})
	}
	add("更改记录", []string{"修订", "时间（UTC）", "操作", "对象"}, rows, []int{12, 30, 30, 42}, "")
	pathCount := 0
	for _, s := range libRows(ws["scenarios"]) {
		pathCount += len(libList(s["paths"]))
	}
	status := "待完善"
	if libBool(review["ready"]) {
		status = "完整性检查通过"
	}
	add("威胁分析风险评估概述", []string{"项目", "数量 / 状态"}, [][]any{{"受保护资产", len(libList(ws["assets"]))}, {"风险场景", len(libList(ws["scenarios"]))}, {"攻击路径", pathCount}, {"安全需求", len(libList(ws["requirements"]))}, {"验证记录", len(libList(ws["evidence"]))}, {"待完成检查", len(libList(review["issues"]))}, {"当前状态", status}}, []int{32, 70}, "")
	rows = [][]any{}
	for i, row := range libList(method["matrix"]) {
		values := []any{impactLabel(i)}
		for j, value := range libList(row) {
			if method["risk_calculation"] == "product" {
				value = calculatedRisk(i, j+1, method)
			}
			values = append(values, value)
		}
		rows = append(rows, values)
	}
	methodColumns := []string{"方法 / 影响等级", feasibilityLabel(1) + "可行性", feasibilityLabel(2) + "可行性", feasibilityLabel(3) + "可行性", feasibilityLabel(4) + "可行性"}
	if coefficient {
		methodColumns = []string{"风险等级", "风险分值范围", "等级名称", "判定锚点", "计算规则"}
		rows = [][]any{}
		thresholds := libList(method["product_thresholds"])
		for i, level := range libRows(method["risk_levels"]) {
			interval := "[0, " + libStr(thresholds[0]) + "]"
			if i > 0 && i < 4 {
				interval = "(" + libStr(thresholds[i-1]) + ", " + libStr(thresholds[i]) + "]"
			}
			if i == 4 {
				interval = "> " + libStr(thresholds[3])
			}
			rows = append(rows, []any{i + 1, interval, level["label"], level["description"], "SFOP总分 × 可行性"})
		}
	}
	add("威胁分析风险评估说明", methodColumns, rows, []int{28, 22, 22, 22, 22}, reportMethodNote(method))
	rows = [][]any{}
	for _, pair := range [][2]string{{"purpose", "功能与用途"}, {"boundary", "边界与接口"}, {"environment", "运行环境"}, {"assumptions", "假设与依赖"}, {"lifecycle", "生命周期阶段"}, {"owner", "责任人"}} {
		rows = append(rows, []any{pair[1], item[pair[0]]})
	}
	rows = append(rows, []any{"评估方法依据", method["rationale"]})
	domain := libStr(libFirst(item["domain"], "automotive"))
	rows = append(rows, []any{"应用领域", map[string]string{"general": "通用系统", "automotive": "汽车", "robotics": "机器人", "iot": "物联网", "charging": "充电基础设施", "internet": "互联网"}[domain]})
	add("相关项定义", []string{"内容项", "说明"}, rows, []int{26, 110}, "")
	rows = [][]any{}
	for _, node := range libRows(architecture["nodes"]) {
		kind := libStr(libFirst(node["kind"], "process"))
		entityType := libStr(node["entityType"])
		if entityType == "" {
			entityType = "ecu"
			if kind == "external" {
				entityType = "external"
			}
		}
		nodeType := map[string]string{"store": "旧数据存储（待归属）", "boundary": "信任边界", "process_step": "处理过程", "data_store": "数据存储"}[kind]
		if kind == "entity" || kind == "process" || kind == "external" {
			nodeType = map[string]string{"ecu": "控制实体", "external": "外部实体", "service": "服务实体", "device": "设备", "application": "应用", "cloud": "云服务", "actor": "参与者"}[entityType]
		}
		details := []string{libStr(node["description"])}
		if kind == "entity" || kind == "process" || kind == "external" {
			details = append(details, reportOS(node["operating_system"]))
		}
		if libBool(node["reference"]) {
			details = append(details, "来源："+reportReference(node["reference"], catalog))
		}
		for _, a := range libRows(node["attributes"]) {
			details = append(details, libStr(a["name"])+"："+libStr(a["value"]))
		}
		rows = append(rows, []any{code(nodeNumberKind(node), node["id"]), nodeType, node["name"], "", "", strings.Join(details, "\n")})
		for _, child := range libRows(node["children"]) {
			desc := libStr(child["description"])
			if libBool(child["reference"]) {
				desc += "\n来源：" + reportReference(child["reference"], catalog)
			}
			rows = append(rows, []any{code(libStr(child["kind"]), child["id"]), map[string]string{"software": "内部软件", "store": "内部数据存储"}[libStr(child["kind"])], child["name"], node["name"], "", desc})
		}
	}
	for _, flow := range libRows(architecture["flows"]) {
		flowType := "数据流"
		if flow["channel_kind"] == "energy" {
			flowType = "电能流（非数据流）"
		}
		rows = append(rows, []any{code("flow", flow["id"]), flowType, flow["name"], nodes[libStr(flow["source"])]["name"], nodes[libStr(flow["target"])]["name"], reportCommunication(flow)})
	}
	add("数据流分析", []string{"标识", "类型", "名称", "所属实体 / 起点", "终点", "属性 / 协议 / 说明"}, rows, []int{38, 24, 28, 28, 28, 70}, "静态架构快照。实体模式中软件与存储归属实体；展开 DFD 模式中处理过程与数据存储独立展示。电能流不作为传输数据资产。")
	rows = [][]any{}
	assetColumns := []string{"资产标识", "资产名称", "描述", "安全属性", "关联组件", "来源引用"}
	assetWidths := []int{38, 28, 65, 28, 35, 28}
	extendedAssets := false
	for _, a := range libRows(ws["assets"]) {
		if a["source_code"] != nil || a["asset_type"] != nil || a["group"] != nil || len(libList(a["function_refs"])) > 0 {
			extendedAssets = true
			break
		}
	}
	if extendedAssets {
		assetColumns = append(assetColumns, "来源编号", "资产类型", "资产分组", "关联功能")
		assetWidths = append(assetWidths, 22, 22, 28, 35)
	}
	for _, a := range libRows(ws["assets"]) {
		properties, nodeNames := []string{}, []string{}
		for _, p := range libList(a["properties"]) {
			properties = append(properties, reportProperties[libStr(p)])
		}
		for _, n := range libList(a["node_ids"]) {
			nodeNames = append(nodeNames, libStr(nodes[libStr(n)]["name"]))
		}
		row := []any{code("asset", a["id"]), a["name"], a["description"], strings.Join(properties, "、"), strings.Join(nodeNames, "、"), reportReference(a["reference"], catalog)}
		if extendedAssets {
			row = append(row, a["source_code"], a["asset_type"], a["group"], libJoin(a["function_refs"], "、"))
		}
		rows = append(rows, row)
	}
	add("资产识别", assetColumns, rows, assetWidths, "")
	rows = [][]any{}
	for _, s := range libRows(ws["scenarios"]) {
		imp := libObj(s["impact"])
		rows = append(rows, []any{code("scenario", s["id"]), s["damage"], impactLabel(imp["safety"]), impactLabel(imp["financial"]), impactLabel(imp["operational"]), impactLabel(imp["privacy"]), imp["rationale"]})
	}
	add("影响性判定", []string{"场景标识", "损害场景", libStr(dimensions["safety"]), libStr(dimensions["financial"]), libStr(dimensions["operational"]), libStr(dimensions["privacy"]), "判定依据"}, rows, []int{38, 70, 18, 18, 18, 18, 75}, "")
	rows = [][]any{}
	for _, s := range libRows(ws["scenarios"]) {
		rows = append(rows, []any{code("scenario", s["id"]), s["title"], assets[libStr(s["asset_id"])]["name"], reportProperties[libStr(s["property"])], s["threat"], reportReferences(s["references"], catalog)})
	}
	add("威胁判定", []string{"场景标识", "风险场景", "资产", "安全属性", "威胁场景", "知识引用"}, rows, []int{38, 35, 28, 18, 80, 35}, "")
	rows = [][]any{}
	for _, s := range libRows(ws["scenarios"]) {
		ev := evaluations[libStr(s["id"])]
		levels := reportIndex(ev["paths"], "id")
		for _, p := range libRows(s["paths"]) {
			var level any = "待评估"
			var score any = "待评估"
			if v, ok := levels[libStr(p["id"])]; ok {
				level = v["level"]
				if v["score"] != nil {
					score = v["score"]
				}
			}
			row := []any{code("scenario", s["id"]), code("path", pathNumberKey(libStr(s["id"]), libStr(p["id"]))), p["name"], p["steps"], impactValue(ev["impact"]), feasibilityLabel(p["feasibility"])}
			if coefficient {
				row = append(row, score)
			}
			rows = append(rows, append(row, level, p["rationale"], reportReference(p["reference"], catalog)))
		}
	}
	riskColumns := []string{"场景标识", "路径标识", "攻击路径", "步骤与前提", "综合影响", "可行性", "风险值", "可行性依据", "来源引用"}
	riskWidths := []int{38, 38, 35, 80, 18, 18, 14, 80, 28}
	if coefficient {
		riskColumns = []string{"场景标识", "路径标识", "攻击路径", "步骤与前提", "SFOP总分", "可行性", "风险分值", "风险等级", "可行性依据", "来源引用"}
		riskWidths = []int{38, 38, 35, 80, 18, 18, 18, 14, 80, 28}
	}
	add("风险评估", riskColumns, rows, riskWidths, "")
	add("附件1 资产分类", []string{"安全属性", "识别提示"}, [][]any{{"机密性", "信息不被未授权主体获取"}, {"完整性", "信息或功能不被未授权修改"}, {"可用性", "授权主体能够在需要时使用"}, {"真实性", "数据或实体来源可确认"}, {"授权", "操作满足权限规则"}, {"不可抵赖性", "行为可关联到对应主体"}}, []int{24, 95}, "")
	rows = [][]any{}
	for _, s := range libRows(ws["scenarios"]) {
		d := decisions[libStr(s["id"])]
		ev := evaluations[libStr(s["id"])]
		treatment, owner, rationale, residualImpact, residualFeasibility, residualRationale, decisionState := "待决策", "", "", "待评估", "待评估", "", "待处置"
		if len(d) > 0 {
			treatment = reportTreatments[libStr(d["treatment"])]
			owner = libStr(d["owner"])
			rationale = libStr(d["rationale"])
			residualImpact = impactValue(d["residual_impact"])
			residualFeasibility = feasibilityLabel(d["residual_feasibility"])
			residualRationale = libStr(d["residual_rationale"])
			decisionState = "已记录"
			if ev["residual_status"] == "target" {
				decisionState = "设计目标，待验证与接受确认"
			}
		}
		if libBool(ev["decision_stale"]) {
			decisionState = "依据已变化，待复核"
		}
		var residual any = "待评估"
		if ev["residual"] != nil {
			residual = ev["residual"]
		}
		row := []any{code("scenario", s["id"]), treatment, owner, rationale, residualImpact, residualFeasibility}
		if coefficient {
			var score any = "待评估"
			if ev["residual_score"] != nil {
				score = ev["residual_score"]
			}
			row = append(row, score)
		}
		rows = append(rows, append(row, residual, residualRationale, decisionState))
	}
	residualColumns := []string{"场景标识", "处置策略", "责任人", "决策依据", "残余影响", "残余可行性", "残余风险", "独立评估依据", "决策状态"}
	residualWidths := []int{38, 18, 22, 65, 18, 20, 18, 75, 28}
	if coefficient {
		residualColumns = []string{"场景标识", "处置策略", "责任人", "决策依据", "残余SFOP总分", "残余可行性", "残余风险分值", "残余风险等级", "独立评估依据", "决策状态"}
		residualWidths = []int{38, 18, 22, 65, 18, 20, 18, 18, 75, 28}
	}
	add("残余风险评估", residualColumns, rows, residualWidths, "")
	add("附件2 威胁分类", []string{"分析视角", "含义"}, [][]any{{"身份仿冒", "冒充实体或来源"}, {"数据篡改", "未经授权修改"}, {"行为抵赖", "行为缺少可追溯证据"}, {"信息泄露", "未经授权获取信息"}, {"拒绝服务", "破坏可用性"}, {"权限提升", "获取未经授权的能力"}}, []int{24, 95}, "通用分析视角，仅供辅助识别，不替代项目场景分析。")
	rows = [][]any{}
	for _, d := range libRows(ws["decisions"]) {
		if libBool(d["goal"]) {
			rows = append(rows, []any{code("scenario", d["scenario_id"]), d["goal"], d["owner"]})
		}
	}
	add("网络安全目标", []string{"场景标识", "目标", "责任人"}, rows, []int{38, 110, 24}, "")
	rows = [][]any{}
	for _, d := range libRows(ws["decisions"]) {
		if libBool(d["claim"]) {
			rows = append(rows, []any{code("scenario", d["scenario_id"]), d["claim"], d["rationale"], d["owner"]})
		}
	}
	add("网络安全声明", []string{"场景标识", "声明", "处置依据", "责任人"}, rows, []int{38, 95, 80, 24}, "")
	rows = [][]any{}
	for _, req := range libRows(ws["requirements"]) {
		logs := []string{}
		for _, e := range libRows(ws["evidence"]) {
			if e["requirement_id"] != req["id"] {
				continue
			}
			logs = append(logs, code("evidence", e["id"])+" · "+libStr(e["reference"])+"："+map[string]string{"pass": "通过", "fail": "失败", "pending": "待验证"}[libStr(e["result"])]+"；"+libStr(e["summary"])+"；复核人 "+libStr(e["reviewer"]))
		}
		rows = append(rows, []any{code("requirement", req["id"]), code("scenario", req["scenario_id"]), req["statement"], req["allocation"], req["verification"], strings.Join(logs, "\n"), reportReferences(req["standard_refs"], catalog), reportReference(req["reference"], catalog)})
	}
	add("网络安全需求", []string{"需求标识", "场景标识", "需求陈述", "分配对象", "验证方法", "验证记录", "标准引用", "知识引用"}, rows, []int{38, 38, 90, 30, 85, 100, 35, 28}, "")
	refSet := map[string]bool{}
	for _, v := range libList(ws["regulation_refs"]) {
		refSet[libStr(v)] = true
	}
	for _, req := range libRows(ws["requirements"]) {
		for _, v := range libList(req["standard_refs"]) {
			refSet[libStr(v)] = true
		}
	}
	refs := []string{}
	for key := range refSet {
		refs = append(refs, key)
	}
	sort.Strings(refs)
	rows = [][]any{}
	for _, key := range refs {
		entry := Object{"title": key, "description": ""}
		if catalog != nil {
			if found, err := catalog.Get(key); err == nil {
				entry = found
			} else {
				entry["description"] = "引用条目当前不可用，请核对资料库"
			}
		}
		relatedIDs := []string{}
		for _, req := range libRows(ws["requirements"]) {
			if libContains(req["standard_refs"], key) {
				relatedIDs = append(relatedIDs, code("requirement", req["id"]))
			}
		}
		related := strings.Join(relatedIDs, "、")
		state := "项目引用，待符合性确认"
		if related != "" {
			state = "已关联需求，待符合性确认"
		}
		source := ""
		if libBool(entry["withdrawn"]) {
			state = "来源已撤下，仅保留历史引用标识；不能据此确认符合性"
			entry["description"] = ""
		}
		if strings.HasPrefix(key, "reg:") && !libBool(entry["withdrawn"]) {
			if entry["document_coverage"] == "partial" {
				state += "；原文尚待完整核验"
			}
			facts := libObj(entry["facts"])
			parts := []string{}
			for _, label := range []string{"正文来源", "原件说明", "原文核对说明", "版本收录说明"} {
				if libBool(facts[label]) {
					parts = append(parts, label+"："+libStr(facts[label]))
				}
			}
			source = strings.Join(parts, "\n")
			pages := libList(entry["source_pages"])
			if len(pages) == 0 && libBool(entry["page"]) {
				pages = []any{entry["page"]}
			}
			if len(pages) > 0 {
				source += "\nPDF 物理页码：" + libJoin(pages, "、")
			}
		}
		rows = append(rows, []any{key, entry["title"], entry["description"], related, state, source})
	}
	add("网络安全标准", []string{"引用编号", "条目 / 版本", "内容", "关联需求", "覆盖状态", "来源与核对说明"}, rows, []int{28, 50, 100, 50, 30, 80}, "")
	if len(libRows(ws["automation_adoptions"])) > 0 {
		rows = [][]any{}
		for _, a := range libRows(ws["automation_adoptions"]) {
			rows = append(rows, []any{a["id"], a["at"], a["owner"], a["base_revision"], a["rule_version"], libJoin(a["candidate_ids"], "、"), a["plan_hash"], a["checkpoint_id"]})
		}
		add("自动化分析追溯", []string{"记录", "采用时间", "责任人", "依据修订", "规则版本", "候选编号", "计划摘要", "还原点"}, rows, []int{30, 26, 20, 12, 30, 65, 70, 30}, "确定性规则生成待评估工作产品；匹配结果不等同于实际漏洞或已完成风险评估。")
	}
	if len(libRows(ws["ai_adoptions"])) > 0 {
		rows = [][]any{}
		for _, a := range libRows(ws["ai_adoptions"]) {
			usage := libObj(a["usage"])
			sources := []string{}
			for _, source := range libRows(a["sources"]) {
				sources = append(sources, libStr(source["id"])+" · "+libStr(source["name"])+" · "+libStr(source["edition"])+" · "+libStr(source["source"])+" · "+libStr(source["content_hash"]))
			}
			rows = append(rows, []any{a["job_id"], a["at"], a["owner"], a["base_revision"], a["prompt_name"], a["prompt_revision"], usage["model"], libJoin(a["citations"], "、"), strings.Join(sources, "\n"), a["checkpoint_id"]})
		}
		add("辅助分析追溯", []string{"任务", "采用时间", "责任人", "依据修订", "提示词", "提示词版本", "模型", "引用片段", "来源及内容摘要哈希", "还原点"}, rows, []int{30, 26, 20, 12, 30, 12, 30, 50, 90, 30}, "AI 草案经平台校验和工作人员采用。采用不代表已完成验证、风险接受或法规符合性确认。")
	}
	if len(libRows(ws["treatment_drafts"])) > 0 {
		rows = [][]any{}
		for _, d := range libRows(ws["treatment_drafts"]) {
			rows = append(rows, []any{code("scenario", d["scenario_id"]), reportTreatments[libStr(d["treatment"])], d["rationale"], d["goal"], d["controls"], d["owner"], "待人工确认"})
		}
		add("处置待确认建议", []string{"场景", "建议方式", "依据", "目标", "措施", "责任人", "状态"}, rows, []int{20, 15, 70, 70, 70, 25, 20}, "仅供评审的建议，不作为已批准的风险处置或验证记录。")
	}
	return append(output, taraReportTables(ws)...)
}

func Printable(ws Object, journal []Object, catalog *Catalog) ([]byte, error) {
	if err := reportCheckReferences(ws, catalog); err != nil {
		return nil, err
	}
	var sections strings.Builder
	escape := func(v any) string { return html.EscapeString(libStr(v)) }
	for _, table := range Tables(ws, journal, catalog) {
		sections.WriteString("<section><h2>" + escape(table["name"]) + "</h2><p>" + strings.ReplaceAll(escape(table["note"]), "\n", "<br>") + "</p><table><thead><tr>")
		for _, column := range libList(table["columns"]) {
			sections.WriteString("<th>" + escape(column) + "</th>")
		}
		sections.WriteString("</tr></thead><tbody>")
		for _, row := range libList(table["rows"]) {
			sections.WriteString("<tr>")
			for _, cell := range libList(row) {
				sections.WriteString("<td>" + strings.ReplaceAll(escape(cell), "\n", "<br>") + "</td>")
			}
			sections.WriteString("</tr>")
		}
		sections.WriteString("</tbody></table></section>")
	}
	issues := ""
	for _, issue := range libRows(Review(ws)["issues"]) {
		issues += "<li>" + escape(issue["message"]) + "</li>"
	}
	if issues == "" {
		issues = "<li>当前完整性检查通过</li>"
	}
	return []byte(fmt.Sprintf(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>%s — 界衡报告</title><style>body{font:14px/1.7 system-ui;color:#234737;margin:40px}h1{font-size:32px}h2{border-bottom:1px solid #ccd6c3;padding:12px 0}table{width:100%%;border-collapse:collapse;table-layout:auto}th,td{text-align:left;vertical-align:top;border-bottom:1px solid #dee5d7;padding:12px;overflow-wrap:anywhere;font-size:12px}th{background:#e8efdf}section{break-before:page;margin-top:40px}p{color:#78826d;font-size:12px}@media print{@page{size:A3 landscape;margin:15mm}body{margin:0}thead{display:table-header-group}}}</style><h1>%s</h1><p>网络安全工作产品报告 · 修订 %s</p><h2>评审待办</h2><ul>%s</ul>%s</html>`, escape(ws["title"]), escape(ws["title"]), escape(ws["revision"]), issues, sections.String())), nil
}
