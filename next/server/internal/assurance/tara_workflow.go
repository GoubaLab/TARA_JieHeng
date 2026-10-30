package assurance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var attackPotentialValues = map[string][]int{
	"elapsed_time": {0, 1, 4, 17, 19},
	"expertise":    {0, 3, 6, 8},
	"knowledge":    {0, 3, 7, 11},
	"opportunity":  {0, 1, 4, 10},
	"equipment":    {0, 4, 7, 9},
}

func validatePotential(p Object) error {
	for key, allowed := range attackPotentialValues {
		if p[key] == nil {
			return dError("攻击潜力必须完成全部五项选择")
		}
		found := false
		for _, value := range allowed {
			if dInt(p[key]) == value {
				found = true
			}
		}
		if !found {
			return dError(key + ": 请选择攻击潜力分级准则中的有效档位")
		}
	}
	return nil
}

func workflowValidate(model string, value Object) error {
	if model == "StandardAttackPotential" {
		return validatePotential(value)
	}
	if model == "AttackPath" {
		if value["feasibility_method"] == "attack_potential" {
			if value["attack_vector"] != nil {
				return dError("攻击潜力法不能同时提交攻击向量评级参数")
			}
			if e := validatePotential(dObj(value["potential"])); e != nil {
				return e
			}
		}
		if value["feasibility_method"] == "attack_potential" && len(dObj(value["potential"])) == 0 {
			return dError("攻击潜力法必须选择全部五项因子")
		}
		if value["feasibility_method"] == "attack_vector" && (value["attack_vector"] == nil || len(dObj(value["potential"])) > 0) {
			return dError("攻击向量法必须选择主要攻击向量，不能同时提交五因子")
		}
	}
	if model == "Item" {
		seen := map[string]bool{}
		for _, raw := range dList(value["functions"]) {
			f := dObj(raw)
			id := dStr(f["id"])
			if seen[id] {
				return dError("功能编号重复：" + id)
			}
			seen[id] = true
		}
	}
	return nil
}

var functionRange = regexp.MustCompile(`(FR-[0-9]+)(?:\.\.(FR-[0-9]+))?\s+([^；;\n]+)`)

func functionIDs(ws Object) map[string]bool {
	ids := map[string]bool{}
	for _, raw := range dList(dObj(ws["item"])["functions"]) {
		ids[dStr(dObj(raw)["id"])] = true
	}
	for _, match := range functionRange.FindAllStringSubmatch(dStr(dObj(ws["item"])["purpose"]), -1) {
		start, _ := strconv.Atoi(strings.TrimPrefix(match[1], "FR-"))
		end := start
		if match[2] != "" {
			end, _ = strconv.Atoi(strings.TrimPrefix(match[2], "FR-"))
		}
		if end < start || end-start > 500 {
			continue
		}
		width := len(strings.TrimPrefix(match[1], "FR-"))
		for n := start; n <= end; n++ {
			ids[fmt.Sprintf("FR-%0*d", width, n)] = true
		}
	}
	return ids
}

func validateFunctionLinks(original, ws Object, action string, value Object) error {
	if action != "record-asset" && action != "record-damage" && action != "define-item" {
		return nil
	}
	ids := functionIDs(ws)
	if action == "define-item" {
		oldIDs := functionIDs(original)
		for _, collection := range []string{"assets", "damage_scenarios"} {
			for _, raw := range dList(ws[collection]) {
				for _, ref := range dList(dObj(raw)["function_refs"]) {
					if oldIDs[dStr(ref)] && !ids[dStr(ref)] {
						return dError("功能仍被引用，不能移除：" + dStr(ref))
					}
				}
			}
		}
		return nil
	}
	collection := "assets"
	if action == "record-damage" {
		collection = "damage_scenarios"
	}
	old, _ := dFind(dList(original[collection]), value["id"])
	for _, ref := range dList(value["function_refs"]) {
		if !ids[dStr(ref)] && !dHas(old["function_refs"], ref) {
			return dError("请先在相关项定义中建立功能内容：" + dStr(ref))
		}
	}
	return nil
}

func applyAttackFeasibility(original, payload Object) (Object, string, error) {
	v, err := dValidateModel("AttackFeasibility", payload)
	if err != nil {
		return nil, "", err
	}
	ws := Clone(original)
	s, err := dFind(dList(ws["scenarios"]), v["scenario_id"])
	if err != nil {
		return nil, "", err
	}
	p, err := dFind(dList(s["paths"]), v["path_id"])
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(dStr(p["steps"])) == "" {
		return nil, "", dError("请先完成攻击路径的步骤与前提")
	}
	switch v["method"] {
	case "attack_potential":
		if err = validatePotential(dObj(v["potential"])); err != nil {
			return nil, "", err
		}
		if v["attack_vector"] != nil {
			return nil, "", dError("攻击潜力法不接受攻击向量评级参数")
		}
		p["potential"] = v["potential"]
		delete(p, "attack_vector")
	case "attack_vector":
		if v["potential"] != nil {
			return nil, "", dError("攻击向量法不接受攻击潜力参数")
		}
		if v["attack_vector"] == nil {
			return nil, "", dError("请选择主要攻击向量")
		}
		delete(p, "potential")
		p["attack_vector"] = v["attack_vector"]
	}
	p["feasibility_method"] = v["method"]
	p["rationale"] = v["rationale"]
	f, _, ok := pathFeasibility(p, EffectiveMethod(dObj(ws["method"])))
	if !ok {
		return nil, "", dError("攻击可行性参数不完整")
	}
	p["feasibility"] = f
	// Reuse scenario validation and recalculation; the repository records the stage action.
	body := Object{}
	for key := range taraSchema("Scenario", dSchema("Scenario")) {
		if x, present := s[key]; present {
			body[key] = x
		}
	}
	return Apply(ws, "record-scenario", body)
}

func clearChangedPathRatings(original, scenario Object) {
	oldScenario, _ := dFind(dList(original["scenarios"]), scenario["id"])
	for _, raw := range dList(scenario["paths"]) {
		path := dObj(raw)
		old, _ := dFind(dList(oldScenario["paths"]), path["id"])
		if len(old) == 0 {
			continue
		}
		changed := false
		for _, key := range []string{"name", "steps", "actor", "vector"} {
			if dStr(old[key]) != dStr(path[key]) {
				changed = true
			}
		}
		if changed {
			path["feasibility"] = nil
			path["rationale"] = ""
			delete(path, "potential")
			delete(path, "feasibility_method")
			delete(path, "attack_vector")
		}
	}
}
