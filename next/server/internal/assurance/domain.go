package assurance

import "sort"

var DefaultMethod = Object{
	"name":      "组织示例方法 · 4 × 4",
	"rationale": "示例配置，须经项目负责人确认。影响取 S/F/O/P 四维最高等级；可行性由评估人员基于路径证据判定。此矩阵不是 ISO/SAE 21434 规定的固定算法。",
	"matrix":    []any{[]any{1, 1, 1, 1}, []any{1, 2, 2, 3}, []any{2, 3, 3, 4}, []any{3, 4, 5, 5}},
}

func NewWorkspace(title, description string) Object {
	item, _ := dValidateModel("Item", Object{})
	architecture, _ := dValidateModel("Architecture", Object{})
	return Object{"id": domainIdentity("ws"), "title": title, "description": description, "revision": 0, "created_at": domainStamp(), "updated_at": domainStamp(), "item": item, "architecture": architecture, "assets": []any{}, "scenarios": []any{}, "decisions": []any{}, "requirements": []any{}, "evidence": []any{}, "method": Clone(DefaultMethod), "method_confirmed": false, "baselines": []any{}, "reference_imports": Object{}, "regulation_refs": []any{}}
}
func Risk(scenario, method Object) Object {
	method = EffectiveMethod(method)
	impact := dObj(scenario["impact"])
	paths := dList(scenario["paths"])
	complete := dBool(impact["rationale"]) && len(paths) > 0
	maxImpact, ratedImpact := aggregateImpact(impact, method)
	var bandScore any
	if method["impact_aggregation"] == "coefficient_bands" {
		maxImpact, bandScore, ratedImpact = bandedImpact(impact, method)
	}
	var impactValue any = maxImpact
	var calculationImpact any = maxImpact
	coefficient := method["impact_aggregation"] == "coefficient_sum"
	if coefficient {
		total, completeImpact := coefficientImpact(impact, method)
		ratedImpact = completeImpact
		if completeImpact {
			impactValue, calculationImpact = decimalNumber(total), total
		}
	}
	complete = complete && ratedImpact
	for _, v := range paths {
		p := dObj(v)
		_, _, rated := pathFeasibility(p, method)
		if !rated || !dBool(p["rationale"]) || !dBool(p["steps"]) {
			complete = false
		}
	}
	if !complete {
		result := Object{"impact": nil, "feasibility": nil, "level": nil, "paths": []any{}}
		if coefficient {
			result["score"] = nil
		}
		return result
	}
	rated := []any{}
	maxFeasibility, maxRisk := 0, 0
	for _, v := range paths {
		p := dObj(v)
		feasibility, potential, _ := pathFeasibility(p, method)
		level := calculatedRisk(calculationImpact, feasibility, method)
		pathResult := Object{"id": p["id"], "level": level}
		if len(dObj(p["potential"])) > 0 {
			pathResult["attack_potential"], pathResult["feasibility"] = potential, feasibility
		}
		if coefficient {
			_, pathResult["score"] = coefficientRisk(calculationImpact, feasibility, method)
		}
		rated = append(rated, pathResult)
		if level > maxRisk {
			maxRisk = level
		}
		if feasibility > maxFeasibility {
			maxFeasibility = feasibility
		}
	}
	result := Object{"impact": impactValue, "feasibility": maxFeasibility, "level": maxRisk, "paths": rated}
	if bandScore != nil {
		result["impact_score"] = bandScore
	}
	if coefficient {
		_, result["score"] = coefficientRisk(calculationImpact, maxFeasibility, method)
	}
	return result
}
func AssessmentHash(ws, scenario Object) string {
	nodes, flows := []any{}, []any{}
	for _, v := range dList(dObj(ws["architecture"])["nodes"]) {
		node := dObj(v)
		semantic := Clone(node)
		for _, key := range []string{"x", "y", "width", "height", "entityType", "children", "attributes"} {
			delete(semantic, key)
		}
		if node["kind"] == "entity" {
			entityType := dStr(node["entityType"])
			if entityType == "external" {
				semantic["kind"] = "external"
			} else {
				semantic["kind"] = "process"
			}
			if entityType != "" && entityType != "ecu" && entityType != "external" {
				semantic["entityType"] = entityType
			}
		}
		if value, present := node["operating_system"]; present {
			if normalized, meaningful := semanticOptional("OperatingSystem", value); meaningful {
				semantic["operating_system"] = normalized
			} else {
				delete(semantic, "operating_system")
			}
		}
		for key, field := range map[string]string{"children": "id", "attributes": "name"} {
			if values := dList(node[key]); len(values) > 0 {
				copied := dList(dClone(values))
				sort.SliceStable(copied, func(i, j int) bool { return dStr(dObj(copied[i])[field]) < dStr(dObj(copied[j])[field]) })
				semantic[key] = copied
			}
		}
		nodes = append(nodes, semantic)
	}
	for _, v := range dList(dObj(ws["architecture"])["flows"]) {
		flow := Clone(dObj(v))
		for _, key := range []string{"sourceSide", "targetSide", "labelOffset", "wireless_layout", "route_layout", "emphasis", "show_label"} {
			delete(flow, key)
		}
		if value, present := flow["communication"]; present {
			if normalized, meaningful := semanticOptional("Communication", value); meaningful {
				flow["communication"] = normalized
			} else {
				delete(flow, "communication")
			}
		}
		flows = append(flows, flow)
	}
	asset, _ := dFind(dList(ws["assets"]), scenario["asset_id"])
	asset = Clone(asset)
	delete(asset, "architecture_source")
	item := Clone(dObj(ws["item"]))
	if item["domain"] == "automotive" {
		delete(item, "domain")
	}
	inputs := []any{scenario, semanticMethod(dObj(ws["method"])), item, asset, Object{"nodes": nodes, "flows": flows}}
	if len(dList(scenario["asset_ids"])) > 0 {
		linked := []any{}
		for _, id := range scenarioAssets(scenario) {
			a, _ := dFind(dList(ws["assets"]), id)
			a = Clone(a)
			delete(a, "architecture_source")
			linked = append(linked, a)
		}
		inputs = append(inputs, linked)
	}
	return Digest(inputs)
}
func Review(ws Object) Object {
	issues := []any{}
	issue := func(stage, text string, ref any) {
		issues = append(issues, Object{"stage": stage, "message": text, "reference": ref})
	}
	item := dObj(ws["item"])
	for _, key := range []string{"purpose", "boundary", "environment", "assumptions", "owner"} {
		if !dBool(item[key]) {
			issue("definition", "补全功能、边界、运行环境、假设和责任人", nil)
			break
		}
	}
	if !dBool(ws["method_confirmed"]) {
		issue("assessment", "确认组织采用的风险评估方法", nil)
	}
	architecture := dObj(ws["architecture"])
	assets, scenarios := dList(ws["assets"]), dList(ws["scenarios"])
	if len(dList(architecture["nodes"])) == 0 || len(assets) == 0 {
		issue("architecture", "建立系统架构并识别受保护资产", nil)
	}
	for _, v := range dList(architecture["nodes"]) {
		node := dObj(v)
		if node["kind"] == "store" {
			issue("architecture", "旧数据存储“"+dStr(node["name"])+"”尚未归属实体", node["id"])
		}
	}
	for _, v := range assets {
		asset := dObj(v)
		if assetSource(asset)["status"] == "missing" {
			issue("architecture", "资产“"+dStr(asset["name"])+"”的架构来源已移除，请核对资产及风险影响", asset["id"])
		}
		if len(dList(asset["node_ids"])) == 0 {
			issue("architecture", "资产“"+dStr(asset["name"])+"”尚未关联架构组件", asset["id"])
		}
		for _, property := range dList(asset["properties"]) {
			found := false
			for _, raw := range scenarios {
				s := dObj(raw)
				if dHas(scenarioAssets(s), asset["id"]) && dHas(scenarioProperties(s), property) {
					found = true
					break
				}
			}
			if !found {
				issue("assessment", "资产“"+dStr(asset["name"])+"”的 "+dStr(property)+" 尚无风险场景", asset["id"])
			}
		}
	}
	if len(scenarios) == 0 {
		issue("assessment", "尚未建立损害场景、威胁及攻击路径", nil)
	}
	requirementReviews := []any{}
	for _, v := range dList(ws["requirements"]) {
		requirement := dObj(v)
		var latest Object
		for _, e := range dList(ws["evidence"]) {
			evidence := dObj(e)
			if evidence["requirement_id"] == requirement["id"] {
				latest = evidence
			}
		}
		status := "missing"
		var latestID any
		if latest != nil {
			latestID = latest["id"]
			if latest["requirement_hash"] == requirementEvidenceHash(ws, requirement) {
				status = dStr(latest["result"])
			} else {
				status = "stale"
			}
		}
		draft := !dBool(requirement["allocation"]) || !dBool(requirement["verification"])
		if draft {
			issue("treatment", "需求草稿尚未明确分配对象或验证方法", requirement["id"])
		}
		if status != "pass" {
			issue("verification", "安全需求缺少有效的通过验证记录", requirement["id"])
		}
		requirementReviews = append(requirementReviews, Object{"requirement_id": requirement["id"], "status": status, "draft": draft, "latest_evidence_id": latestID})
	}
	evaluations := []any{}
	for _, v := range scenarios {
		s := dObj(v)
		result := Risk(s, dObj(ws["method"]))
		var decision Object
		for _, raw := range dList(ws["decisions"]) {
			d := dObj(raw)
			if d["scenario_id"] == s["id"] {
				decision = d
				break
			}
		}
		stale := decision != nil && (decision["assessment_hash"] != AssessmentHash(ws, s) || goalDecisionStale(ws, decision))
		residualStatus := decision["residual_status"]
		if residualStatus == "validated" && !scenarioVerified(ws, s["id"]) {
			residualStatus = "target"
		}
		var residual any
		var residualScore any
		if result["level"] == nil {
			issue("assessment", "评估缺少影响等级、路径或判定依据", s["id"])
		}
		if decision == nil {
			issue("treatment", "尚未记录风险处置决策", s["id"])
		} else if stale {
			issue("treatment", "评估依据已变化，需要重新确认处置决策", s["id"])
		} else {
			if residualStatus == "target" {
				issue("verification", "残余风险为设计目标，尚未完成控制验证与接受确认", s["id"])
			}
			if !dBool(decision["owner"]) {
				issue("treatment", "处置责任人尚未填写", s["id"])
			}
			if decision["residual_impact"] == nil || decision["residual_feasibility"] == nil || !dBool(decision["residual_rationale"]) {
				issue("treatment", "尚未独立评估残余风险并记录依据", s["id"])
			} else {
				method := EffectiveMethod(dObj(ws["method"]))
				residual = calculatedRisk(decision["residual_impact"], dInt(decision["residual_feasibility"]), method)
				if method["impact_aggregation"] == "coefficient_sum" {
					_, residualScore = coefficientRisk(decision["residual_impact"], dInt(decision["residual_feasibility"]), method)
				}
			}
			if decision["treatment"] == "reduce" {
				hasRequirement := false
				for _, raw := range dList(ws["requirements"]) {
					if dHas(requirementScenarios(dObj(raw)), s["id"]) {
						hasRequirement = true
						break
					}
				}
				if !dBool(decision["goal"]) || !hasRequirement {
					issue("treatment", "降低风险需要安全目标和分配到组件的安全需求", s["id"])
				}
			} else if !dBool(decision["claim"]) {
				issue("treatment", "保留、分担或规避决策需要网络安全声明", s["id"])
			}
		}
		evaluation := Clone(result)
		evaluation["scenario_id"] = s["id"]
		evaluation["decision_stale"] = stale
		evaluation["residual"] = residual
		if decision != nil && decision["residual_status"] != nil {
			evaluation["residual_status"] = residualStatus
		}
		if dObj(ws["method"])["impact_aggregation"] == "coefficient_sum" {
			evaluation["residual_score"] = residualScore
		}
		evaluations = append(evaluations, evaluation)
	}
	stages := []any{}
	blocked := false
	for _, key := range []string{"definition", "architecture", "assessment", "treatment", "verification"} {
		for _, v := range issues {
			if dObj(v)["stage"] == key {
				blocked = true
			}
		}
		stages = append(stages, Object{"key": key, "complete": !blocked})
	}
	return Object{"ready": len(issues) == 0, "issues": issues, "evaluations": evaluations, "requirements": requirementReviews, "stages": stages}
}
func dUpsert(ws Object, collection string, value Object, prefix string) (string, error) {
	values := dList(ws[collection])
	if dBool(value["id"]) {
		if _, err := dFind(values, value["id"]); err != nil {
			return "", err
		}
		for i, v := range values {
			if dObj(v)["id"] == value["id"] {
				values[i] = value
			}
		}
	} else {
		value["id"] = domainIdentity(prefix)
		values = append(values, value)
	}
	ws[collection] = values
	return dStr(value["id"]), nil
}
func Apply(original Object, action string, payload Object) (Object, string, error) {
	ws := Clone(original)
	preserveGoalSnapshots(ws)
	if action != "seal-baseline" {
		ensureRecordNumbers(ws)
	}
	entityID := dStr(ws["id"])
	var err error
	fail := func(message string) (Object, string, error) { return nil, "", dError(message) }
	if action == "import-tara" {
		if err := applyTaraImport(ws, payload); err != nil {
			return nil, "", err
		}
		entityID = dStr(dObj(ws["tara_import"])["id"])
	} else if action == "replace-architecture" {
		payload = Clone(payload)
		if _, present := payload["asset_generation"]; !present {
			if mode, ok := dObj(ws["architecture"])["asset_generation"]; ok {
				payload["asset_generation"] = mode
			}
		}
		stores, parents := map[string]bool{}, map[string]any{}
		for _, v := range dList(dObj(ws["architecture"])["nodes"]) {
			node := dObj(v)
			if node["kind"] == "store" {
				stores[dStr(node["id"])] = true
			}
		}
		for _, v := range dList(payload["nodes"]) {
			node := dObj(v)
			kind := node["kind"]
			if kind == nil || kind == "entity" || kind == "process" || kind == "external" {
				for _, c := range dList(node["children"]) {
					child := dObj(c)
					if stores[dStr(child["id"])] && child["kind"] == "store" {
						parents[dStr(child["id"])] = node["id"]
					}
				}
			}
		}
		for _, v := range dList(payload["flows"]) {
			flow := dObj(v)
			for _, key := range []string{"source", "target"} {
				if parent, ok := parents[dStr(flow[key])]; ok {
					flow[key] = parent
				}
			}
		}
		graph, e := ValidatePayload(action, payload)
		if e != nil {
			return nil, "", e
		}
		ids := map[string]bool{}
		for _, v := range dList(graph["nodes"]) {
			node := dObj(v)
			if diagramEndpoint(graph, node) {
				ids[dStr(node["id"])] = true
			}
		}
		for _, v := range dList(ws["assets"]) {
			asset := dObj(v)
			mapped := []any{}
			for _, id := range dList(asset["node_ids"]) {
				if parent, ok := parents[dStr(id)]; ok {
					id = parent
				}
				mapped = append(mapped, id)
			}
			asset["node_ids"] = dUnique(mapped)
			for _, id := range dList(asset["node_ids"]) {
				if !ids[dStr(id)] && len(assetSource(asset)) == 0 {
					return fail("该实体已关联资产，请先调整资产关联；迁移旧数据存储时须保留其内部对象标识")
				}
			}
		}
		ws["architecture"] = graph
		syncArchitectureAssets(ws)
	} else if action == "apply-reference-selection" {
		return nil, "", ErrReferenceDataUnavailable
	} else {
		payload = preserveTaraFields(ws, action, payload)
		value, e := ValidatePayload(action, payload)
		if e != nil {
			return nil, "", e
		}
		switch action {
		case "define-item":
			ws["item"] = value
		case "record-damage":
			entityID, err = dUpsert(ws, "damage_scenarios", value, "damage")
		case "record-goal":
			entityID, err = dUpsert(ws, "security_goals", value, "goal")
		case "record-asset":
			ids := map[string]bool{}
			for _, v := range dList(dObj(ws["architecture"])["nodes"]) {
				node := dObj(v)
				if diagramEndpoint(dObj(ws["architecture"]), node) || node["kind"] == "process" || node["kind"] == "external" {
					ids[dStr(node["id"])] = true
				}
			}
			for _, id := range dList(value["node_ids"]) {
				if !ids[dStr(id)] {
					return fail("资产必须关联已有实体，不能直接关联内部软件、数据存储或信任边界")
				}
			}
			if dBool(value["id"]) {
				for _, s := range dList(ws["scenarios"]) {
					scenario := dObj(s)
					if scenario["asset_id"] == value["id"] && !dHas(value["properties"], scenario["property"]) {
						return fail("已有风险场景依赖该安全属性")
					}
				}
			}
			if previous, findErr := dFind(dList(ws["assets"]), value["id"]); findErr == nil && len(assetSource(previous)) > 0 {
				source := Clone(assetSource(previous))
				overrides := dList(source["overrides"])
				for _, field := range []string{"name", "description", "node_ids", "properties"} {
					if Digest(value[field]) != Digest(previous[field]) {
						overrides = append(overrides, field)
					}
				}
				source["overrides"] = dUnique(overrides)
				value["architecture_source"] = source
			}
			entityID, err = dUpsert(ws, "assets", value, "asset")
		case "record-scenario":
			asset, e := dFind(dList(ws["assets"]), value["asset_id"])
			if e != nil {
				return nil, "", e
			}
			if !dHas(asset["properties"], value["property"]) {
				return fail("风险场景的安全属性必须属于关联资产")
			}
			entityID, err = dUpsert(ws, "scenarios", value, "scenario")
		case "decide-treatment":
			scenario, e := dFind(dList(ws["scenarios"]), value["scenario_id"])
			if e != nil {
				return nil, "", e
			}
			if Risk(scenario, dObj(ws["method"]))["level"] == nil || !dBool(ws["method_confirmed"]) {
				return fail("请先完成风险评估并确认评估方法")
			}
			if (value["residual_impact"] == nil) != (value["residual_feasibility"] == nil) {
				return fail("残余风险的影响和可行性必须同时填写")
			}
			if value["residual_impact"] != nil && !dBool(value["residual_rationale"]) {
				return fail("请填写残余风险的独立判定依据")
			}
			if value["residual_impact"] != nil {
				normalized, validationErr := validateResidualImpact(value["residual_impact"], EffectiveMethod(dObj(ws["method"])))
				if validationErr != nil {
					return nil, "", validationErr
				}
				value["residual_impact"] = normalized
			}

			if value["residual_status"] == "validated" && !scenarioVerified(ws, scenario["id"]) {
				return fail("请先完成全部关联需求在当前目标与评估依据下的有效验证")
			}

			value["assessment_hash"] = AssessmentHash(ws, scenario)
			if len(dList(value["goal_ids"])) > 0 {
				value["goal_hash"] = linkedGoalsHash(ws, dList(value["goal_ids"]))
			}
			value["decided_at"] = domainStamp()
			decisions := []any{}
			for _, d := range dList(ws["decisions"]) {
				if dObj(d)["scenario_id"] != value["scenario_id"] {
					decisions = append(decisions, d)
				}
			}
			ws["decisions"] = append(decisions, value)
			entityID = dStr(value["scenario_id"])
		case "specify-requirement":
			if _, e := dFind(dList(ws["scenarios"]), value["scenario_id"]); e != nil {
				return nil, "", e
			}
			entityID, err = dUpsert(ws, "requirements", value, "req")
		case "select-regulations":
			ws["regulation_refs"] = value["references"]
		case "record-evidence":
			if dBool(value["id"]) {
				return fail("验证记录不可覆盖，请新增复测记录")
			}
			requirement, e := dFind(dList(ws["requirements"]), value["requirement_id"])
			if e != nil {
				return nil, "", e
			}
			if !dBool(requirement["allocation"]) || !dBool(requirement["verification"]) {
				return fail("请先补全需求草稿的分配对象和验证方法")
			}
			value["requirement_hash"] = requirementEvidenceHash(ws, requirement)
			value["recorded_at"] = domainStamp()
			entityID, err = dUpsert(ws, "evidence", value, "evidence")
		case "configure-numbering":
			ws["numbering"] = value
		case "adopt-method":
			ws["method"] = value
			ws["method_confirmed"] = true
		case "seal-baseline":
			if !dBool(Review(ws)["ready"]) {
				return fail("仍有未完成的工作产品，不能建立已评审基线")
			}
			value["id"] = domainIdentity("baseline")
			value["revision"] = ws["revision"]
			value["sealed_at"] = domainStamp()
			value["content_hash"] = Digest(ws)
			ws["baselines"] = append(dList(ws["baselines"]), value)
			entityID = dStr(value["id"])
		}
		if err != nil {
			return nil, "", err
		}
	}
	if hasTaraLinks(ws) {
		if err := refreshTaraLinks(ws); err != nil {
			return nil, "", err
		}
	}
	if action != "seal-baseline" {
		ensureRecordNumbers(ws)
	}
	ws["revision"] = dInt(ws["revision"]) + 1
	ws["updated_at"] = domainStamp()
	return ws, entityID, nil
}
