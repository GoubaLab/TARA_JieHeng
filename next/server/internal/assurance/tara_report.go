package assurance

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func taraReportTables(ws Object) []Object {
	if ws["tara_import"] == nil && !hasTaraLinks(ws) {
		return nil
	}
	out := []Object{}
	codes := RecordCodes(ws)
	code := func(kind string, r Object) string {
		if c := dStr(r["source_code"]); c != "" && !strings.HasPrefix(c, "TPL:") {
			return c
		}
		return codes(kind, r["id"])
	}
	refs := func(kind, collection string, ids []any) string {
		parts := []string{}
		for _, id := range ids {
			row, e := dFind(dList(ws[collection]), id)
			if e != nil {
				parts = append(parts, dStr(id)+"（引用缺失）")
				continue
			}
			title := dStr(row["name"])
			if title == "" {
				title = dStr(row["title"])
			}
			if title == "" {
				title = dStr(row["description"])
			}
			if title == "" {
				title = dStr(row["statement"])
			}
			parts = append(parts, code(kind, row)+" · "+title)
		}
		return strings.Join(parts, "\n")
	}

	add := func(name string, columns []string, rows [][]any, note string) {
		widths := make([]int, len(columns))
		for i := range widths {
			widths[i] = 28
		}
		out = append(out, Object{"name": name, "columns": columns, "rows": rows, "widths": widths, "note": note})
	}
	method := EffectiveMethod(dObj(ws["method"]))
	scale := dList(method["impact_scale"])
	rows := [][]any{}
	for _, raw := range dList(ws["damage_scenarios"]) {
		d := dObj(raw)
		impact := dObj(d["impact"])
		row := []any{code("damage", d), refs("asset", "assets", dList(d["asset_ids"])), libJoin(d["function_refs"], "、"), d["description"], d["consequence"], taraPropertyNames(d["properties"])}
		for _, key := range impactKeys {
			v := impact[key]
			if method["impact_aggregation"] == "coefficient_bands" && len(scale) == 4 && v != nil {
				v = scale[dInt(v)]
			}
			row = append(row, v)
		}
		rank, ok := damageImpactOrder(impact, method)
		if !ok {
			row = append(row, "待评估", "待评估")
		} else if method["impact_aggregation"] == "coefficient_sum" {
			row = append(row, decimalNumber(rank), "总分直接参与风险计算")
		} else if method["impact_aggregation"] == "coefficient_bands" {
			level, score, _ := bandedImpact(impact, method)
			row = append(row, score, reportLevel(method, "impact_levels", level, 0))
		} else {
			row = append(row, "", reportLevel(method, "impact_levels", int(rank.Num().Int64()), 0))
		}

		row = append(row, impact["rationale"])
		rows = append(rows, row)
	}
	add("损害场景工作产品", []string{"编号", "关联资产", "关联功能", "损害场景", "道路使用者后果", "安全属性", "S", "F", "O", "P", "影响值", "影响等级", "评级依据"}, rows, "")
	rows = [][]any{}
	for _, raw := range dList(ws["scenarios"]) {
		s := dObj(raw)
		rows = append(rows, []any{code("scenario", s), s["title"], refs("asset", "assets", scenarioAssets(s)), taraPropertyNames(scenarioProperties(s)), refs("damage", "damage_scenarios", dList(s["damage_ids"])), s["stride"], s["cause"], s["cal"]})
	}
	add("威胁完整追溯", []string{"编号", "威胁场景", "全部关联资产", "全部受损属性", "关联损害", "STRIDE", "攻击原因", "CAL"}, rows, "")
	templateOrigins, assessmentMethods := false, false
	for _, scenario := range libRows(ws["scenarios"]) {
		for _, path := range libRows(scenario["paths"]) {
			if dBool(path["feasibility_method"]) {
				assessmentMethods = true
			}
			if strings.HasPrefix(libStr(path["source_code"]), "TPL:") {
				templateOrigins = true
			}
		}
	}
	recordCode := RecordCodes(ws)
	rows = [][]any{}
	for _, raw := range dList(ws["scenarios"]) {
		s := dObj(raw)
		for _, pRaw := range dList(s["paths"]) {
			p := dObj(pRaw)
			potential := dObj(p["potential"])
			f, total, _ := pathFeasibility(p, method)
			var potentialTotal any = ""
			if len(potential) > 0 {
				potentialTotal = total
			}
			pathCode := code("path", p)
			if dStr(p["source_code"]) == "" || strings.HasPrefix(dStr(p["source_code"]), "TPL:") {
				pathCode = recordCode("path", pathNumberKey(dStr(s["id"]), dStr(p["id"])))
			}
			row := []any{pathCode, code("scenario", s), p["actor"], p["vector"], p["steps"], potential["elapsed_time"], potential["expertise"], potential["knowledge"], potential["opportunity"], potential["equipment"], potentialTotal, reportLevel(method, "feasibility_levels", f, 1)}
			if templateOrigins {
				row = append(row, p["source_code"])
			}
			if assessmentMethods {
				row = append(row, map[string]string{"attack_potential": "攻击潜力 · 五因子", "attack_vector": "主要攻击向量"}[dStr(p["feasibility_method"])], p["attack_vector"], p["rationale"])
			}
			rows = append(rows, row)
		}
	}
	pathColumns := []string{"AP", "TS", "攻击者", "向量", "步骤", "ET", "Ex", "KoI", "WoO", "Eq", "AP值", "可行性"}
	if templateOrigins {
		pathColumns = append(pathColumns, "路径来源编号 / 版本")
	}
	if assessmentMethods {
		pathColumns = append(pathColumns, "评级方法", "主要攻击向量", "评级依据")
	}
	add("攻击潜力工作产品", pathColumns, rows, "")
	rows = [][]any{}
	for _, raw := range dList(ws["security_goals"]) {
		g := dObj(raw)
		rows = append(rows, []any{code("goal", g), g["statement"], refs("scenario", "scenarios", dList(g["scenario_ids"])), g["cal"], g["theme"]})
	}
	add("独立安全目标", []string{"编号", "目标", "关联威胁", "CAL", "控制主题"}, rows, "")
	rows = [][]any{}
	for _, raw := range dList(ws["requirements"]) {
		r := dObj(raw)
		rows = append(rows, []any{code("requirement", r), refs("goal", "security_goals", dList(r["goal_ids"])), refs("scenario", "scenarios", requirementScenarios(r)), r["statement"], r["verification"], r["allocation"], r["cal"], libJoin(r["customer_refs"], "、")})
	}
	add("需求完整追溯", []string{"编号", "目标", "全部关联威胁", "需求", "验证方法", "分配", "CAL", "客户需求"}, rows, "")
	rows = [][]any{}
	evaluations := reportIndex(Review(ws)["evaluations"], "scenario_id")
	for _, raw := range dList(ws["decisions"]) {
		d := dObj(raw)
		state := evaluations[dStr(d["scenario_id"])]["residual_status"]
		if dBool(evaluations[dStr(d["scenario_id"])]["decision_stale"]) {
			state = "依据变化，待复核"
		}
		rows = append(rows, []any{refs("scenario", "scenarios", []any{d["scenario_id"]}), d["controls"], refs("goal", "security_goals", dList(d["goal_ids"])), d["cal"], state, d["acceptance"], d["acceptance_rationale"], d["customer_confirmation"], d["confirmed_at"], d["declarant"]})
	}
	add("接受条件与确认", []string{"威胁", "控制措施", "安全目标", "CAL", "残余评估状态", "原接受结论", "残余依据", "客户确认", "确认日期", "声明人"}, rows, "target 为设计目标，不能作为验证通过或签署接受的证据。")
	source := dObj(ws["tara_import"])
	for index, raw := range dList(source["sheets"]) {
		sheet := dObj(raw)
		sourceRows := dList(sheet["rows"])
		maxCols := 0
		for _, r := range sourceRows {
			if len(dList(r)) > maxCols {
				maxCols = len(dList(r))
			}
		}
		columns := []string{"Excel 行号"}
		for i := 1; i <= maxCols; i++ {
			col, _ := excelize.ColumnNumberToName(i)
			columns = append(columns, col)
		}
		data := [][]any{}
		for i, r := range sourceRows {
			row := append([]any{i + 1}, dList(r)...)
			for len(row) < len(columns) {
				row = append(row, "")
			}
			data = append(data, row)
		}
		name := fmt.Sprintf("原表%02d %s", index+1, dStr(sheet["name"]))
		runes := []rune(name)
		if len(runes) > 31 {
			name = string(runes[:31])
		}
		add(name, columns, data, "导入来源原表快照；编辑后的工作产品见前方项目表。来源："+dStr(source["filename"])+"；SHA-256："+dStr(source["sha256"]))
	}
	return out
}
func taraPropertyNames(v any) string {
	out := []string{}
	for _, p := range dList(v) {
		out = append(out, reportProperties[dStr(p)])
	}
	return strings.Join(out, "、")
}

// Only native opaque identifiers are replaced in presentation text; stored records
// and imported source tables retain their original bytes and revision fingerprints.
func reportTextFormatter(ws Object) func(string) string {
	pairs := []string{}
	codes := RecordCodes(ws)
	for _, pair := range [][2]string{{"assets", "asset"}, {"damage_scenarios", "damage"}, {"scenarios", "scenario"}, {"security_goals", "goal"}, {"requirements", "requirement"}} {
		for _, r := range libRows(ws[pair[0]]) {
			id := dStr(r["id"])
			if len(id) < 16 || !strings.Contains(id, "_") {
				continue
			}
			code := dStr(r["source_code"])
			if code == "" {
				code = codes(pair[1], id)
			}
			pairs = append(pairs, id, code)
		}
	}
	return strings.NewReplacer(pairs...).Replace
}
