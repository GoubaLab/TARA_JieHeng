package assurance

// Deletions remove a record from the working copy only. Revisions, the journal
// and frozen baselines keep the history; retired display numbers are never reused.
var deletionCollections = map[string]string{"delete-asset": "assets", "delete-damage": "damage_scenarios", "delete-scenario": "scenarios", "delete-requirement": "requirements", "delete-goal": "security_goals"}

func assetPropertyExcluded(asset Object, property any) bool {
	for _, raw := range dList(asset["excluded_properties"]) {
		if dObj(raw)["property"] == property {
			return true
		}
	}
	return false
}

// An exclusion documents why a protected property is out of scope. It must name a
// declared property and cannot coexist with a threat scenario on the same property.
func validateAssetExclusions(ws, asset Object) error {
	seen := map[string]bool{}
	for _, raw := range dList(asset["excluded_properties"]) {
		property := dStr(dObj(raw)["property"])
		if seen[property] {
			return dError("同一安全属性只能有一条不适用说明")
		}
		seen[property] = true
		if !dHas(asset["properties"], property) {
			return dError("不适用说明只能针对该资产声明的安全属性")
		}
		for _, s := range dList(ws["scenarios"]) {
			scenario := dObj(s)
			if dHas(scenarioAssets(scenario), asset["id"]) && dHas(scenarioProperties(scenario), property) {
				return dError("资产 " + workProductDependencyLabel(ws, "asset", asset) + " 的" + reportProperties[property] + "已有威胁场景 " + workProductDependencyLabel(ws, "scenario", scenario) + "，不能同时存在不适用说明与威胁；请先撤销不适用说明或调整威胁")
			}
		}
	}
	return nil
}

// Dependency errors use the same source or project numbering as work-product lists.
func workProductDependencyLabel(ws Object, kind string, record Object) string {
	code := dStr(record["source_code"])
	if code == "" && dStr(record["id"]) != "" {
		code = RecordCodes(ws)(kind, record["id"])
		if code == dStr(record["id"]) {
			code = ""
		}
	}
	name := ""
	for _, key := range []string{"name", "title", "description", "statement"} {
		if name = dStr(record[key]); name != "" {
			break
		}
	}
	name = assetText(name, 80)
	if code != "" && name != "" {
		return code + " · " + name
	}
	if code != "" {
		return code
	}
	if name != "" {
		return name
	}
	return "未编号记录"
}

func deleteRecord(ws Object, collection, id string) error {
	rows := dList(ws[collection])
	if _, err := dFind(rows, id); err != nil {
		return ErrNotFound
	}
	label := func(kind string, record Object) string { return workProductDependencyLabel(ws, kind, record) }
	switch collection {
	case "assets":
		asset, _ := dFind(rows, id)
		if dObj(ws["architecture"])["asset_generation"] != "catalog" && assetSource(asset)["status"] == "active" {
			return dError("该资产由系统架构派生，请在架构中移除来源对象，或改用资产目录模式")
		}
		for _, raw := range dList(ws["scenarios"]) {
			if dHas(scenarioAssets(dObj(raw)), id) {
				return dError("威胁场景 " + label("scenario", dObj(raw)) + " 仍引用该资产，请先调整或删除该场景")
			}
		}
		for _, raw := range dList(ws["damage_scenarios"]) {
			if dHas(dObj(raw)["asset_ids"], id) {
				return dError("损害场景 " + label("damage", dObj(raw)) + " 仍引用该资产，请先调整或删除该损害场景")
			}
		}
	case "damage_scenarios":
		for _, raw := range dList(ws["scenarios"]) {
			if dHas(dObj(raw)["damage_ids"], id) {
				return dError("威胁场景 " + label("scenario", dObj(raw)) + " 仍引用该损害场景，请先调整或删除该场景")
			}
		}
	case "scenarios":
		for _, raw := range dList(ws["requirements"]) {
			if dHas(requirementScenarios(dObj(raw)), id) {
				return dError("安全需求 " + label("requirement", dObj(raw)) + " 仍关联该威胁场景，请先调整或删除该需求")
			}
		}
		for _, raw := range dList(ws["security_goals"]) {
			if dHas(dObj(raw)["scenario_ids"], id) {
				return dError("安全目标 " + label("goal", dObj(raw)) + " 仍关联该威胁场景，请先调整或删除该目标")
			}
		}
		for _, key := range []string{"decisions", "treatment_drafts"} {
			if ws[key] == nil {
				continue
			}
			kept := []any{}
			for _, raw := range dList(ws[key]) {
				if dObj(raw)["scenario_id"] != id {
					kept = append(kept, raw)
				}
			}
			ws[key] = kept
		}
	case "requirements":
		for _, raw := range dList(ws["evidence"]) {
			if dObj(raw)["requirement_id"] == id {
				return dError("该需求已有验证记录，为保留审计痕迹不能删除；可修改需求内容或建立新需求")
			}
		}
	case "security_goals":
		for _, raw := range dList(ws["requirements"]) {
			if dHas(dObj(raw)["goal_ids"], id) {
				return dError("安全需求 " + label("requirement", dObj(raw)) + " 仍关联该安全目标，请先调整或删除该需求")
			}
		}
		for _, dependency := range []struct{ collection, title string }{{"decisions", "处置决策"}, {"treatment_drafts", "处置草案"}} {
			for _, raw := range dList(ws[dependency.collection]) {
				decision := dObj(raw)
				if dHas(decision["goal_ids"], id) {
					scenario, _ := dFind(dList(ws["scenarios"]), decision["scenario_id"])
					return dError("威胁场景 " + label("scenario", scenario) + " 的" + dependency.title + "仍关联该安全目标，请先调整" + dependency.title)
				}
			}
		}
	}
	kept := []any{}
	for _, raw := range rows {
		if dObj(raw)["id"] != id {
			kept = append(kept, raw)
		}
	}
	ws[collection] = kept
	return nil
}
