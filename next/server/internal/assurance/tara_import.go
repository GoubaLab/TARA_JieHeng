package assurance

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

var taraCodePattern = regexp.MustCompile(`(?:AS|DS|TS|AP|CSG|CSR|FR|IF|SEC-BMS)-[0-9]+`)

func taraRefs(s string) []any {
	out := []any{}
	for _, v := range taraCodePattern.FindAllString(s, -1) {
		out = append(out, v)
	}
	return dUnique(out)
}
func taraProps(s string) ([]any, error) {
	names := map[string]string{"机密性": "confidentiality", "完整性": "integrity", "可用性": "availability", "真实性": "authenticity", "授权性": "authorization", "授权": "authorization", "不可否认性": "non_repudiation", "不可抵赖性": "non_repudiation", "新鲜性": "freshness", "隐私性": "privacy"}
	out := []any{}
	for _, v := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune("、，,；; /\n", r) }) {
		key, ok := names[v]
		if !ok {
			return nil, dError("未识别的安全属性：" + v)
		}
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil, dError("安全属性为空")
	}
	return dUnique(out), nil
}

type taraRow struct {
	number int
	cells  []string
}

func (r taraRow) cell(column int) string {
	if column < 1 || column > len(r.cells) {
		return ""
	}
	return strings.TrimSpace(r.cells[column-1])
}
func (r taraRow) numberAt(column int) (int, error) {
	n, err := strconv.Atoi(r.cell(column))
	if err != nil {
		return 0, dError(fmt.Sprintf("第 %d 行第 %d 列不是整数", r.number, column))
	}
	return n, nil
}
func taraRows(rows []taraRow, prefix string) []taraRow {
	out := []taraRow{}
	p := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `[0-9]+$`)
	for _, r := range rows {
		if p.MatchString(r.cell(1)) {
			out = append(out, r)
		}
	}
	return out
}
func taraMethod() Object {
	return Object{"name": "导入 TARA · SFOP 系数分档与攻击潜力矩阵", "rationale": "按来源工作簿的方法配置：SFOP 原始取值 0/10/100/1000，S、F 系数 1，O、P 系数 0.1；影响值按 20/100/1000 分档，任一单维严重时综合影响不低于重大。攻击潜力五因子求和后按 10/14/20 分档。风险直接查项目矩阵。此处保留来源方法，不表示已完成项目批准。", "impact_aggregation": "coefficient_bands", "impact_scale": []any{0, 10, 100, 1000}, "impact_coefficients": Object{"safety": 1, "financial": 1, "operational": 0.1, "privacy": 0.1}, "impact_thresholds": []any{20, 100, 1000}, "severe_impact_floor": 2, "attack_potential_thresholds": []any{10, 14, 20}, "risk_calculation": "matrix", "matrix": []any{[]any{1, 1, 2, 3}, []any{1, 2, 3, 4}, []any{2, 3, 4, 5}, []any{3, 4, 5, 5}}, "feasibility_levels": methodLevels("极低", "低", "中", "高")}
}

// This adapter recognizes a documented column contract, not arbitrary prose.
// Unsupported methods/headers fail closed; original bytes and all sheets survive.
func buildTaraImport(payload Object) (Object, error) {
	for key := range payload {
		if key != "filename" && key != "content_base64" && key != "asset_links" {
			return nil, dError("不允许额外字段：" + key)
		}
	}
	name := dStr(payload["filename"])
	if filepath.Base(name) != name || len(name) > 200 || !strings.EqualFold(filepath.Ext(name), ".xlsx") {
		return nil, dError("请选择 XLSX 文件，文件名不得含路径")
	}
	data, err := base64.StdEncoding.DecodeString(dStr(payload["content_base64"]))
	if err != nil || len(data) == 0 || len(data) > 1_400_000 {
		return nil, dError("文件无效或超过 1.4 MB 导入限制")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, dError("文件不是有效的 XLSX")
	}
	var expanded uint64
	for _, f := range archive.File {
		expanded += f.UncompressedSize64
	}
	if expanded > 30_000_000 || len(archive.File) > 1000 {
		return nil, dError("工作簿解压后超出导入限制")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 30_000_000, UnzipXMLSizeLimit: 30_000_000})
	if err != nil {
		return nil, dError("无法读取工作簿")
	}
	defer book.Close()
	if len(book.GetSheetList()) > 40 {
		return nil, dError("工作簿最多支持 40 个表页")
	}
	sheets := []any{}
	byName := map[string][]taraRow{}
	cellCount := 0
	for _, sheet := range book.GetSheetList() {
		rows, e := book.GetRows(sheet)
		if e != nil {
			return nil, dError("无法读取表页：" + sheet)
		}
		if len(rows) > 5000 {
			return nil, dError("表页行数超出限制")
		}
		snapshot := []any{}
		formulas := []any{}
		list := []taraRow{}
		for i, row := range rows {
			if len(row) > 100 {
				return nil, dError("表页列数超出限制")
			}
			values := []any{}
			for j, value := range row {
				if len(value) > 40000 {
					return nil, dError("单元格内容过长")
				}
				values = append(values, value)
				if value != "" {
					cellCount++
				}
				cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
				formula, _ := book.GetCellFormula(sheet, cell)
				if formula != "" {
					formulas = append(formulas, Object{"cell": cell, "formula": formula, "cached_value": value})
				}
			}
			snapshot = append(snapshot, values)
			list = append(list, taraRow{i + 1, row})
		}
		merges := []any{}
		merged, _ := book.GetMergeCells(sheet)
		for _, m := range merged {
			merges = append(merges, m.GetStartAxis()+":"+m.GetEndAxis())
		}
		visible, _ := book.GetSheetVisible(sheet)
		sheets = append(sheets, Object{"name": sheet, "rows": snapshot, "formulas": formulas, "merges": merges, "visible": visible})
		byName[sheet] = list
	}
	contracts := map[string][]string{
		"资产与损害场景": {"资产编号", "资产", "类型", "分组", "关联功能", "安全属性"},
		"威胁场景":    {"TS", "威胁描述", "攻击原因", "STRIDE", "受损属性", "关联资产", "关联DS", "关联路径"},
		"攻击路径":    {"AP", "TS", "攻击者", "向量", "攻击步骤", "ET", "Ex", "KoI", "WoO", "Eq", "AP值", "可行性"},
		"风险评估与处置": {"TS", "影响", "可行性", "初始风险", "处置", "控制措施", "目标", "CAL", "残余可行性", "残余风险", "接受状态", "处置说明"},
		"网络安全目标":  {"目标编号", "目标内容", "关联TS", "最高风险", "CAL", "控制主题"},
		"网络安全需求":  {"需求编号", "所属目标", "需求内容", "验证方法", "分配专业", "CAL", "客户需求编号"},
	}
	for sheet, headers := range contracts {
		rows := byName[sheet]
		if len(rows) < 3 {
			return nil, dError("缺少表页或表头：" + sheet)
		}
		for i, h := range headers {
			if rows[2].cell(i+1) != h {
				return nil, dError(fmt.Sprintf("%s 第 3 行第 %d 列应为“%s”", sheet, i+1, h))
			}
		}
	}
	rules := ""
	for _, r := range byName["概述、方法与依据"] {
		rules += strings.Join(r.cells, " ") + "\n"
	}
	for _, expected := range []string{"IL = S + F + 0.1×O + 0.1×P", "0、10、100、1000", "20-99", "100-999", "AP 0-9高；10-13中；14-19低；≥20极低"} {
		if !strings.Contains(rules, expected) {
			return nil, dError("评估方法与此导入模板不一致，需要先核对映射：" + expected)
		}
	}
	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum)
	ws := NewWorkspace("TARA 导入", "")
	ws["method"] = taraMethod()
	ws["damage_scenarios"], ws["security_goals"] = []any{}, []any{}
	diagnostics := []any{}
	warn := func(code, field string, expected, actual any) {
		if fmt.Sprint(expected) != fmt.Sprint(actual) {
			diagnostics = append(diagnostics, Object{"code": code, "field": field, "source": expected, "calculated": actual})
		}
	}
	appendValidated := func(collection, model string, v Object) error {
		validated, e := dValidateModel(model, v)
		if e != nil {
			return dError(dStr(v["source_code"]) + ": " + e.Error())
		}
		ws[collection] = append(dList(ws[collection]), validated)
		return nil
	}
	for _, r := range taraRows(byName["资产与损害场景"], "AS-") {
		props, e := taraProps(r.cell(6))
		if e != nil {
			return nil, e
		}
		v := Object{"id": r.cell(1), "source_code": r.cell(1), "name": r.cell(2), "asset_type": r.cell(3), "group": r.cell(4), "function_refs": taraRefs(r.cell(5)), "properties": props, "description": "", "node_ids": []any{}, "reference": nil}
		if e = appendValidated("assets", "Asset", v); e != nil {
			return nil, e
		}
	}
	scale := map[int]int{0: 0, 10: 1, 100: 2, 1000: 3}
	for _, r := range taraRows(byName["资产与损害场景"], "DS-") {
		props, e := taraProps(r.cell(6))
		if e != nil {
			return nil, e
		}
		impact := Object{"rationale": "道路使用者后果：" + r.cell(5) + "；按来源方法独立评定 S/F/O/P。"}
		for i, key := range impactKeys {
			raw, e := r.numberAt(i + 7)
			if e != nil {
				return nil, e
			}
			level, ok := scale[raw]
			if !ok {
				return nil, dError(r.cell(1) + " 的 SFOP 取值不在方法定义中")
			}
			impact[key] = level
		}
		level, score, _ := bandedImpact(impact, EffectiveMethod(dObj(ws["method"])))
		warn(r.cell(1), "影响值", r.cell(11), score)
		warn(r.cell(1), "影响等级", r.cell(12), reportImpacts[level])
		if e = appendValidated("damage_scenarios", "DamageScenario", Object{"id": r.cell(1), "source_code": r.cell(1), "description": r.cell(4), "consequence": r.cell(5), "asset_ids": taraRefs(r.cell(2)), "function_refs": taraRefs(r.cell(3)), "properties": props, "impact": impact}); e != nil {
			return nil, e
		}
	}
	paths := map[string]Object{}
	pathThreat := map[string]string{}
	for _, r := range taraRows(byName["攻击路径"], "AP-") {
		potential := Object{}
		for i, key := range []string{"elapsed_time", "expertise", "knowledge", "opportunity", "equipment"} {
			n, e := r.numberAt(i + 6)
			if e != nil {
				return nil, e
			}
			potential[key] = n
		}
		path := Object{"id": r.cell(1), "source_code": r.cell(1), "name": r.cell(1) + " · " + r.cell(4), "actor": r.cell(3), "vector": r.cell(4), "steps": r.cell(5), "potential": potential, "rationale": fmt.Sprintf("来源攻击路径第 %d 行：ET=%v，Ex=%v，KoI=%v，WoO=%v，Eq=%v。", r.number, potential["elapsed_time"], potential["expertise"], potential["knowledge"], potential["opportunity"], potential["equipment"])}
		f, total, _ := pathFeasibility(path, dObj(ws["method"]))
		path["feasibility"] = f
		warn(r.cell(1), "AP值", r.cell(11), total)
		warn(r.cell(1), "可行性", r.cell(12), []string{"极低", "低", "中", "高"}[f-1])
		path, e := dValidateModel("AttackPath", path)
		if e != nil {
			return nil, e
		}
		if paths[r.cell(1)] != nil {
			return nil, dError("攻击路径编号重复")
		}
		paths[r.cell(1)] = path
		pathThreat[r.cell(1)] = r.cell(2)
	}
	seenPaths := map[string]bool{}
	for _, r := range taraRows(byName["威胁场景"], "TS-") {
		props, e := taraProps(r.cell(5))
		if e != nil {
			return nil, e
		}
		assets := taraRefs(r.cell(6))
		if len(assets) == 0 {
			return nil, dError("威胁场景未关联资产")
		}
		primary := ""
		var property any
		for _, id := range assets {
			a, err := dFind(dList(ws["assets"]), id)
			if err != nil {
				return nil, err
			}
			for _, p := range props {
				if dHas(a["properties"], p) {
					primary = dStr(id)
					property = p
					break
				}
			}
			if primary != "" {
				break
			}
		}
		if primary == "" {
			return nil, dError("威胁属性与资产属性无交集")
		}
		pp := []any{}
		for _, id := range taraRefs(r.cell(8)) {
			p := paths[dStr(id)]
			if p == nil || pathThreat[dStr(id)] != r.cell(1) {
				return nil, dError("威胁与攻击路径关联不一致")
			}
			seenPaths[dStr(id)] = true
			pp = append(pp, p)
		}
		if e = appendValidated("scenarios", "Scenario", Object{"id": r.cell(1), "source_code": r.cell(1), "title": r.cell(2), "asset_id": primary, "asset_ids": assets, "property": property, "properties": props, "damage_ids": taraRefs(r.cell(7)), "damage": "待关联损害场景", "threat": r.cell(2), "cause": r.cell(3), "stride": r.cell(4), "paths": pp}); e != nil {
			return nil, e
		}
	}
	if len(seenPaths) != len(paths) {
		return nil, dError("存在未关联到威胁场景的攻击路径")
	}
	for _, r := range taraRows(byName["网络安全目标"], "CSG-") {
		cal, e := strconv.Atoi(strings.TrimPrefix(r.cell(5), "CAL"))
		if e != nil {
			return nil, dError("目标 CAL 无效")
		}
		if e = appendValidated("security_goals", "SecurityGoal", Object{"id": r.cell(1), "source_code": r.cell(1), "statement": r.cell(2), "scenario_ids": taraRefs(r.cell(3)), "cal": cal, "theme": r.cell(6)}); e != nil {
			return nil, e
		}
	}
	for _, r := range taraRows(byName["网络安全需求"], "CSR-") {
		goals := taraRefs(r.cell(2))
		scenarios := []any{}
		for _, id := range goals {
			g, e := dFind(dList(ws["security_goals"]), id)
			if e != nil {
				return nil, e
			}
			scenarios = append(scenarios, dList(g["scenario_ids"])...)
		}
		scenarios = dUnique(scenarios)
		if len(scenarios) == 0 {
			return nil, dError("安全需求未关联目标或威胁")
		}
		cal, e := strconv.Atoi(strings.TrimPrefix(r.cell(6), "CAL"))
		if e != nil {
			return nil, dError("需求 CAL 无效")
		}
		if e = appendValidated("requirements", "RequirementDraft", Object{"id": r.cell(1), "source_code": r.cell(1), "scenario_id": scenarios[0], "scenario_ids": scenarios, "goal_ids": goals, "statement": r.cell(3), "verification": r.cell(4), "allocation": r.cell(5), "cal": cal, "customer_refs": taraRefs(r.cell(7))}); e != nil {
			return nil, e
		}
	}
	if value, present := payload["asset_links"]; present {
		if _, e := dValidateField(dl(dm("AssetLink", true), 0, 500, true), value); e != nil {
			return nil, e
		}
	}
	if len(dList(payload["asset_links"])) > 500 {
		return nil, dError("资产映射过多")
	}
	for _, raw := range dList(payload["asset_links"]) {
		link := dObj(raw)
		asset, e := dFind(dList(ws["assets"]), link["asset_id"])
		if e != nil {
			return nil, dError("资产映射引用不存在的资产")
		}
		ids, e := dValidateField(dl(ds(1, 200), 0, 200, true), link["node_ids"])
		if e != nil {
			return nil, e
		}
		asset["node_ids"] = dUnique(dList(ids))
	}
	if err = refreshTaraLinks(ws); err != nil {
		return nil, err
	}
	acceptance := map[string]taraRow{}
	for _, r := range taraRows(byName["风险接受声明"], "TS-") {
		acceptance[r.cell(1)] = r
	}
	feasibility := map[string]int{"极低": 1, "低": 2, "中": 3, "高": 4}
	treatments := map[string]string{"降低": "reduce", "规避": "avoid", "转移": "share", "接受": "retain"}
	for _, r := range taraRows(byName["风险评估与处置"], "TS-") {
		s, e := dFind(dList(ws["scenarios"]), r.cell(1))
		if e != nil {
			return nil, e
		}
		risk := Risk(s, dObj(ws["method"]))
		if risk["level"] == nil {
			return nil, dError("威胁场景评级不完整：" + r.cell(1))
		}
		impact := dInt(risk["impact"])
		cal, e := strconv.Atoi(strings.TrimPrefix(r.cell(8), "CAL"))
		if e != nil {
			return nil, dError("处置 CAL 无效")
		}
		s["cal"] = cal
		treatment, ok := treatments[r.cell(5)]
		if !ok {
			return nil, dError("未识别的处置方式")
		}
		resF, ok := feasibility[r.cell(9)]
		if !ok {
			return nil, dError("未识别的残余可行性")
		}
		warn(r.cell(1), "初始风险", r.cell(4), risk["level"])
		warn(r.cell(1), "影响", r.cell(2), fmt.Sprintf("%d %s", impact+1, reportImpacts[impact]))
		warn(r.cell(1), "可行性", r.cell(3), []string{"极低", "低", "中", "高"}[dInt(risk["feasibility"])-1])
		warn(r.cell(1), "残余目标风险", r.cell(10), calculatedRisk(impact, resF, EffectiveMethod(dObj(ws["method"]))))
		goals := taraRefs(r.cell(7))
		goalTexts := []string{}
		for _, id := range goals {
			g, e := dFind(dList(ws["security_goals"]), id)
			if e != nil {
				return nil, e
			}
			goalTexts = append(goalTexts, dStr(g["source_code"])+" "+dStr(g["statement"]))
		}
		ar := acceptance[r.cell(1)]
		resWhy := ar.cell(7)
		if resWhy == "" {
			resWhy = "来源设计目标：控制措施按需求落地并通过验证后，按“" + r.cell(9) + "”可行性评估。控制措施：" + r.cell(6)
		}
		decision := Object{"scenario_id": r.cell(1), "treatment": treatment, "rationale": r.cell(12), "owner": "", "goal": strings.Join(goalTexts, "\n"), "goal_ids": goals, "claim": r.cell(11) + "；以措施落地并通过验证为前提。", "controls": r.cell(6), "cal": cal, "residual_impact": impact, "residual_feasibility": resF, "residual_rationale": resWhy, "residual_status": "target", "acceptance": r.cell(11), "acceptance_rationale": ar.cell(7), "customer_confirmation": ar.cell(8), "confirmed_at": ar.cell(9), "declarant": ar.cell(10)}
		decision["assessment_hash"] = AssessmentHash(ws, s)
		decision["goal_hash"] = linkedGoalsHash(ws, goals)
		ws["decisions"] = append(dList(ws["decisions"]), decision)
	}
	if len(dList(ws["decisions"])) != len(dList(ws["scenarios"])) {
		return nil, dError("威胁场景与处置记录数量不一致")
	}
	for _, collection := range []string{"assets", "damage_scenarios", "scenarios", "security_goals", "requirements"} {
		seen := map[string]bool{}
		for _, raw := range dList(ws[collection]) {
			id := dStr(dObj(raw)["id"])
			if seen[id] {
				return nil, dError(collection + " 存在重复编号：" + id)
			}
			seen[id] = true
		}
		if len(seen) == 0 {
			return nil, dError(collection + " 为空")
		}
	}
	item := dObj(ws["item"])
	title := "TARA 导入项目"
	for _, r := range byName["相关项定义"] {
		switch r.cell(1) {
		case "相关项名称":
			title = r.cell(2)
		case "功能范围":
			item["purpose"] = r.cell(2)
		case "分析边界":
			item["boundary"] = r.cell(2)
		case "运行模式":
			item["environment"] = r.cell(2)
		}
	}
	item["assumptions"] = "项目采用零部件级分析范围。完整假设、未决输入、责任划分、方法说明及版本记录见 TARA 工作产品的原表。残余风险为设计目标，须验证并签署后生效。"
	item["domain"] = "automotive"
	ws["title"] = title + " · 完整 TARA"
	images := []any{}
	for _, f := range archive.File {
		if strings.HasPrefix(f.Name, "xl/media/") && (strings.HasSuffix(f.Name, ".jpeg") || strings.HasSuffix(f.Name, ".png")) {
			rc, e := f.Open()
			if e != nil {
				return nil, e
			}
			b, e := io.ReadAll(io.LimitReader(rc, 4_000_001))
			rc.Close()
			if e != nil || len(b) > 4_000_000 {
				return nil, dError("工作簿图像超出限制")
			}
			mime := "image/jpeg"
			if strings.HasSuffix(f.Name, ".png") {
				mime = "image/png"
			}
			images = append(images, Object{"name": filepath.Base(f.Name), "mime": mime, "content_base64": base64.StdEncoding.EncodeToString(b)})
		}
	}
	ws["tara_import"] = Object{"id": "tara_" + hash[:24], "filename": name, "sha256": hash, "imported_at": domainStamp(), "profile": "tara-chinese-v1", "sheets": sheets, "nonempty_cells": cellCount, "images": images, "original_base64": base64.StdEncoding.EncodeToString(data), "diagnostics": diagnostics}
	if len(diagnostics) > 0 {
		first := dObj(diagnostics[0])
		return nil, dError(fmt.Sprintf("计算结果与原表不一致：%s %s，原值 %v，计算值 %v（共 %d 项）", first["code"], first["field"], first["source"], first["calculated"], len(diagnostics)))
	}
	rulesN := EffectiveNumbering(nil)
	dObj(rulesN["prefixes"])["scenario"] = "TS"
	dObj(rulesN["prefixes"])["requirement"] = "CSR"
	ws["numbering"] = rulesN
	return ws, nil
}

func applyTaraImport(ws, payload Object) error {
	if len(dList(ws["damage_scenarios"])) > 0 || len(dList(ws["security_goals"])) > 0 || len(dList(ws["decisions"])) > 0 || ws["tara_import"] != nil || len(dList(ws["assets"])) > 0 || len(dList(ws["scenarios"])) > 0 || len(dList(ws["requirements"])) > 0 || len(dList(ws["evidence"])) > 0 || len(dList(ws["baselines"])) > 0 {
		return dError("完整 TARA 导入需要空白项目，以免覆盖现有工作产品")
	}
	incoming, err := buildTaraImport(payload)
	if err != nil {
		return err
	}
	for _, raw := range dList(incoming["assets"]) {
		for _, id := range dList(dObj(raw)["node_ids"]) {
			node, e := dFind(dList(dObj(ws["architecture"])["nodes"]), id)
			if e != nil || !diagramEndpoint(dObj(ws["architecture"]), node) {
				return dError("资产映射必须指向现有架构端点")
			}
		}
	}
	for _, k := range []string{"item", "assets", "damage_scenarios", "scenarios", "security_goals", "requirements", "decisions", "method", "method_confirmed", "numbering", "tara_import"} {
		ws[k] = incoming[k]
	}
	dObj(ws["architecture"])["asset_generation"] = "catalog"
	for _, raw := range dList(ws["decisions"]) {
		d := dObj(raw)
		s, _ := dFind(dList(ws["scenarios"]), d["scenario_id"])
		d["assessment_hash"] = AssessmentHash(ws, s)
	}
	return nil
}

func taraSummary(ws Object) Object {
	source := dObj(ws["tara_import"])
	paths := 0
	for _, s := range dList(ws["scenarios"]) {
		paths += len(dList(dObj(s)["paths"]))
	}
	return Object{"id": source["id"], "filename": source["filename"], "sha256": source["sha256"], "sheets": len(dList(source["sheets"])), "cells": source["nonempty_cells"], "assets": len(dList(ws["assets"])), "damages": len(dList(ws["damage_scenarios"])), "scenarios": len(dList(ws["scenarios"])), "paths": paths, "goals": len(dList(ws["security_goals"])), "requirements": len(dList(ws["requirements"])), "diagnostics": source["diagnostics"]}
}
