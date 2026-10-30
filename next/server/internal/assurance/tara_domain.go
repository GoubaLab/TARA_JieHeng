package assurance

import (
	"fmt"
	"math/big"
	"strings"
)

// Optional work-product fields leave legacy aggregates and their hashes intact.
func taraSchema(model string, schema map[string]dField) map[string]dField {
	if schema == nil {
		schema = map[string]dField{}
	}
	refs := func() dField { return dop(dl(ds(1, 200), 0, 500, false)) }
	switch model {
	case "Asset", "Scenario", "AttackPath", "Requirement", "RequirementDraft", "DamageScenario", "SecurityGoal":
		schema["source_code"] = dop(ds(0, 100))
	}
	switch model {
	case "Item":
		schema["functions"] = dop(dl(dm("ItemFunction", true), 0, 500, true))
	case "Architecture":
		schema["asset_generation"] = dop(de("automatic", "catalog"))
	case "Asset":
		schema["asset_type"], schema["group"], schema["function_refs"] = dop(ds(0, 200)), dop(ds(0, 200)), refs()
	case "Scenario":
		schema["asset_ids"], schema["damage_ids"] = refs(), refs()
		schema["properties"] = dop(dl(de(dProperties...), 1, 8, true))
		schema["stride"], schema["cause"], schema["cal"] = dop(ds(0, 500)), dop(ds(0, 10000)), dop(dn("int", 1, 4, true))
	case "AttackRoute":
		schema["node_ids"], schema["flow_ids"] = dl(ds(1, 80), 2, 13, true), dl(ds(1, 80), 1, 12, true)
		schema["fingerprint"], schema["reviewed"] = ds(1, 100), db(true, false)
	case "AttackPath":
		schema["architecture_route"] = dop(dm("AttackRoute", true))
		schema["feasibility_method"] = dop(de("attack_potential", "attack_vector"))
		schema["attack_vector"] = dop(de("network", "adjacent", "local", "physical"))
		schema["actor"], schema["vector"], schema["potential"] = dop(ds(0, 500)), dop(ds(0, 200)), dop(dm("AttackPotential", true))
	case "AssetLink":
		schema["asset_id"], schema["node_ids"] = ds(1, 200), dl(ds(1, 200), 0, 200, true)
	case "AttackPotential", "StandardAttackPotential":
		for _, key := range []string{"elapsed_time", "expertise", "knowledge", "opportunity", "equipment"} {
			schema[key] = dn("int", 0, 100, true)
		}
	case "Requirement", "RequirementDraft":
		schema["scenario_ids"], schema["goal_ids"], schema["customer_refs"] = refs(), refs(), refs()
		schema["cal"] = dop(dn("int", 1, 4, true))
	case "Decision":
		schema["controls"], schema["acceptance"], schema["acceptance_rationale"] = dop(ds(0, 10000)), dop(ds(0, 200)), dop(ds(0, 10000))
		schema["customer_confirmation"], schema["confirmed_at"], schema["declarant"] = dop(ds(0, 500)), dop(ds(0, 100)), dop(ds(0, 500))
		schema["goal_ids"], schema["cal"], schema["residual_status"] = refs(), dop(dn("int", 1, 4, true)), dop(de("target", "validated"))
	case "DamageScenario":
		schema["id"], schema["description"], schema["consequence"] = dsn(), ds(1, 10000), dsd("", 10000)
		schema["asset_ids"], schema["function_refs"], schema["properties"], schema["impact"] = dl(ds(1, 200), 1, 500, true), refs(), dl(de(dProperties...), 1, 8, true), dm("Impact", true)
	case "SecurityGoal":
		schema["id"], schema["statement"], schema["scenario_ids"], schema["cal"], schema["theme"] = dsn(), ds(1, 10000), dl(ds(1, 200), 1, 500, true), dop(dn("int", 1, 4, true)), dsd("", 1000)
	case "Method":
		schema["impact_aggregation"] = dop(de("maximum", "weighted_mean", "coefficient_sum", "coefficient_bands"))
		schema["impact_scale"] = dop(dl(dn("number", 0, 1000000, true), 4, 4, true))
		schema["impact_thresholds"] = dop(dl(dn("number", 0, 400000000, true), 3, 3, true))
		schema["severe_impact_floor"] = dop(dn("int", 0, 3, true))
		schema["attack_potential_thresholds"] = dop(dl(dn("int", 1, 500, true), 3, 3, true))
	}
	if len(schema) == 0 {
		return nil
	}
	return schema
}

func bandedImpact(impact, method Object) (int, any, bool) {
	scale := dList(method["impact_scale"])
	thresholds := dList(method["impact_thresholds"])
	if len(scale) != 4 || len(thresholds) != 3 {
		return 0, nil, false
	}
	sum := new(big.Rat)
	severe := false
	for _, key := range impactKeys {
		if impact[key] == nil {
			return 0, nil, false
		}
		level := dInt(impact[key])
		if level < 0 || level > 3 {
			return 0, nil, false
		}
		sum.Add(sum, new(big.Rat).Mul(decimalRat(scale[level]), decimalRat(dObj(method["impact_coefficients"])[key])))
		severe = severe || level == 3
	}
	level := 0
	for _, t := range thresholds {
		if sum.Cmp(decimalRat(t)) >= 0 {
			level++
		}
	}
	if severe && dInt(method["severe_impact_floor"]) > level {
		level = dInt(method["severe_impact_floor"])
	}
	return level, decimalNumber(sum), true
}
func pathFeasibility(path, method Object) (int, int, bool) {
	p := dObj(path["potential"])
	ts := dList(method["attack_potential_thresholds"])
	if path["feasibility_method"] == "attack_vector" {
		f, ok := map[string]int{"network": 4, "adjacent": 3, "local": 2, "physical": 1}[dStr(path["attack_vector"])]
		return f, 0, ok
	}
	if len(p) == 0 {
		if path["feasibility_method"] == "attack_potential" {
			return 0, 0, false
		}
		f := dInt(path["feasibility"])
		return f, 0, path["feasibility"] != nil && f >= 1 && f <= 4
	}
	if path["feasibility_method"] == "attack_potential" && validatePotential(p) != nil {
		return 0, 0, false
	}
	if len(ts) == 0 {
		ts = []any{14, 20, 25}
	}
	if len(ts) != 3 {
		return 0, 0, false
	}
	sum := 0
	for _, key := range []string{"elapsed_time", "expertise", "knowledge", "opportunity", "equipment"} {
		if p[key] == nil {
			return 0, 0, false
		}
		sum += dInt(p[key])
	}
	level := 4
	for _, t := range ts {
		if sum >= dInt(t) {
			level--
		}
	}
	return level, sum, true
}
func scenarioAssets(s Object) []any {
	if len(dList(s["asset_ids"])) > 0 {
		return dList(s["asset_ids"])
	}
	return []any{s["asset_id"]}
}
func scenarioProperties(s Object) []any {
	if len(dList(s["properties"])) > 0 {
		return dList(s["properties"])
	}
	return []any{s["property"]}
}
func requirementScenarios(r Object) []any {
	if len(dList(r["scenario_ids"])) > 0 {
		return dList(r["scenario_ids"])
	}
	return []any{r["scenario_id"]}
}

func preserveTaraFields(ws Object, action string, payload Object) Object {
	model, collection, key := "", "", "id"
	switch action {
	case "define-item":
		out := Clone(payload)
		if _, present := out["functions"]; !present {
			if f, present := dObj(ws["item"])["functions"]; present {
				out["functions"] = dClone(f)
			}
		}
		return out
	case "record-asset":
		model, collection = "Asset", "assets"
	case "record-scenario":
		model, collection = "Scenario", "scenarios"
	case "specify-requirement":
		model, collection = "Requirement", "requirements"
	case "decide-treatment":
		model, collection, key = "Decision", "decisions", "scenario_id"
	case "adopt-method":
		out := Clone(payload)
		for k := range taraSchema("Method", nil) {
			if k == "impact_aggregation" {
				continue
			}
			if _, ok := out[k]; !ok {
				if value, exists := dObj(ws["method"])[k]; exists {
					out[k] = dClone(value)
				}
			}
		}
		return out
	default:
		return payload
	}
	out := Clone(payload)
	for _, raw := range dList(ws[collection]) {
		old := dObj(raw)
		if old[key] != payload[key] {
			continue
		}
		for k := range taraSchema(model, nil) {
			if _, ok := out[k]; !ok {
				if value, present := old[k]; present {
					out[k] = dClone(value)
				}
			}
		}
	}
	return out
}

func refreshTaraLinks(ws Object) error {
	method := EffectiveMethod(dObj(ws["method"]))
	for _, raw := range dList(ws["damage_scenarios"]) {
		d := dObj(raw)
		for _, id := range dList(d["asset_ids"]) {
			if _, err := dFind(dList(ws["assets"]), id); err != nil {
				return dError("损害场景引用了不存在的资产")
			}
		}
	}
	for _, raw := range dList(ws["scenarios"]) {
		s := dObj(raw)
		if len(dList(s["asset_ids"])) > 0 && !dHas(s["asset_ids"], s["asset_id"]) {
			return dError("主资产须包含在全部关联资产中")
		}
		if len(dList(s["properties"])) > 0 && !dHas(s["properties"], s["property"]) {
			return dError("主安全属性须包含在全部受损属性中")
		}
		for _, id := range scenarioAssets(s) {
			if _, err := dFind(dList(ws["assets"]), id); err != nil {
				return dError("威胁场景引用了不存在的资产")
			}
		}
		if ids := dList(s["damage_ids"]); len(ids) > 0 {
			texts, why := []string{}, []string{}
			var highest *big.Rat
			var chosen Object
			complete := true
			for _, id := range ids {
				d, err := dFind(dList(ws["damage_scenarios"]), id)
				if err != nil {
					return dError("威胁场景引用了不存在的损害场景")
				}
				impact := dObj(d["impact"])
				rank, rated := damageImpactOrder(impact, method)
				if !rated || strings.TrimSpace(dStr(impact["rationale"])) == "" {
					complete = false
				}
				if rated && (highest == nil || rank.Cmp(highest) > 0) {
					highest = rank
					chosen = Clone(impact)
				}
				texts = append(texts, taraRecordCode(d)+" "+dStr(d["description"])+"\n道路使用者后果："+dStr(d["consequence"]))
				why = append(why, fmt.Sprintf("%s：%s", dStr(id), dStr(impact["rationale"])))
			}
			if !complete {
				chosen = Object{"safety": nil, "financial": nil, "operational": nil, "privacy": nil}
				why = append(why, "关联损害场景尚待完整评级，当前风险保持待评估")
			}
			chosen["rationale"] = strings.Join(why, "\n")
			s["impact"] = chosen
			s["damage"] = strings.Join(texts, "\n\n")
		}
		for _, p := range dList(s["paths"]) {
			path := dObj(p)
			if len(dObj(path["potential"])) > 0 || dBool(path["feasibility_method"]) {
				if f, _, ok := pathFeasibility(path, method); ok {
					path["feasibility"] = f
				} else {
					return dError("请完成所选方法的攻击可行性评估参数")
				}
			}
		}
	}
	for _, raw := range dList(ws["security_goals"]) {
		g := dObj(raw)
		for _, id := range dList(g["scenario_ids"]) {
			if _, err := dFind(dList(ws["scenarios"]), id); err != nil {
				return dError("安全目标引用了不存在的威胁场景")
			}
		}
	}
	for _, raw := range dList(ws["requirements"]) {
		r := dObj(raw)
		if len(dList(r["scenario_ids"])) > 0 && !dHas(r["scenario_ids"], r["scenario_id"]) {
			return dError("主威胁须包含在全部关联威胁中")
		}
		for _, id := range requirementScenarios(r) {
			if _, err := dFind(dList(ws["scenarios"]), id); err != nil {
				return dError("安全需求引用了不存在的威胁场景")
			}
		}
		for _, id := range dList(r["goal_ids"]) {
			if _, err := dFind(dList(ws["security_goals"]), id); err != nil {
				return dError("安全需求引用了不存在的目标")
			}
		}
	}
	for _, raw := range dList(ws["decisions"]) {
		decision := dObj(raw)
		for _, id := range dList(decision["goal_ids"]) {
			if _, err := dFind(dList(ws["security_goals"]), id); err != nil {
				return dError("处置引用了不存在的安全目标")
			}
		}
		if len(dList(decision["goal_ids"])) > 0 {
			decision["goal"] = linkedGoalText(ws, dList(decision["goal_ids"]))
		}
	}
	return nil
}
