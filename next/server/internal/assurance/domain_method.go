package assurance

import "sort"

var impactKeys = []string{"safety", "financial", "operational", "privacy"}

func dLabel(value string) dField { f := ds(1, 100); f.fallback = value; return f }

func methodLevels(labels ...string) []any {
	levels := make([]any, len(labels))
	for i, label := range labels {
		levels[i] = Object{"label": label, "description": ""}
	}
	return levels
}

// EffectiveMethod adds presentation and algorithm defaults without changing the
// persisted method. Legacy methods retain their original serialized identity.
func EffectiveMethod(method Object) Object {
	result := Clone(DefaultMethod)
	for key, value := range method {
		result[key] = dClone(value)
	}
	defaults := methodDefaults()
	for key, value := range defaults {
		if _, present := result[key]; !present {
			result[key] = dClone(value)
		}
	}
	for _, key := range []string{"impact_weights", "impact_dimensions", "impact_coefficients"} {
		complete := Clone(dObj(defaults[key]))
		for field, value := range dObj(result[key]) {
			complete[field] = dClone(value)
		}
		result[key] = complete
	}
	// Numeric spelling does not change coefficient semantics (1 and 1.0 agree).
	for key, value := range dObj(result["impact_coefficients"]) {
		number := decimalRat(value)
		if number.IsInt() {
			dObj(result["impact_coefficients"])[key] = int(number.Num().Int64())
		} else {
			dObj(result["impact_coefficients"])[key] = decimalNumber(number)
		}
	}
	if result["impact_aggregation"] == "coefficient_sum" {
		for i, value := range dList(result["product_thresholds"]) {
			number := decimalRat(value)
			if number.IsInt() {
				dList(result["product_thresholds"])[i] = int(number.Num().Int64())
			}
		}
	}
	return result
}

func methodDefaults() Object {
	return Object{
		"impact_aggregation":  "maximum",
		"impact_weights":      Object{"safety": 25, "financial": 25, "operational": 25, "privacy": 25},
		"impact_coefficients": Object{"safety": 1, "financial": 1, "operational": 0.1, "privacy": 0.1},
		"impact_dimensions":   Object{"safety": "安全", "financial": "财务", "operational": "运行", "privacy": "隐私"},
		"impact_levels":       methodLevels("可忽略", "中等", "重大", "严重"),
		"feasibility_levels":  methodLevels("很低", "低", "中", "高"),
		"risk_levels":         methodLevels("很低", "低", "中", "高", "很高"),
		"risk_calculation":    "matrix",
		"product_thresholds":  []any{3, 6, 9, 12},
	}
}

func semanticMethod(method Object) Object {
	result := Clone(method)
	effective := EffectiveMethod(method)
	for key, value := range methodDefaults() {
		if _, present := result[key]; !present {
			continue
		}
		if Digest(effective[key]) == Digest(value) {
			delete(result, key)
		} else {
			result[key] = effective[key]
		}
	}
	return result
}

func semanticOptional(model string, value any) (any, bool) {
	input, ok := value.(map[string]any)
	if !ok {
		return value, true
	}
	defaults, _ := dValidateModel(model, Object{})
	complete := Clone(defaults)
	for key, v := range input {
		complete[key] = dClone(v)
	}
	if model == "Communication" {
		properties := dUnique(dList(complete["security_properties"]))
		sort.Slice(properties, func(i, j int) bool { return dStr(properties[i]) < dStr(properties[j]) })
		complete["security_properties"] = properties
	}
	return complete, Digest(complete) != Digest(defaults)
}

func aggregateImpact(impact, method Object) (int, bool) {
	weighted := method["impact_aggregation"] == "weighted_mean"
	weights := dObj(method["impact_weights"])
	maximum, sum, total := 0, 0, 0
	for _, key := range impactKeys {
		weight := 1
		if weighted {
			weight = dInt(weights[key])
		}
		if weight == 0 {
			continue
		}
		if impact[key] == nil {
			return 0, false
		}
		level := dInt(impact[key])
		if level > maximum {
			maximum = level
		}
		sum += level * weight
		total += weight
	}
	if weighted {
		if total == 0 {
			return 0, false
		}
		return (sum + total - 1) / total, true
	}
	return maximum, true
}

func calculatedRisk(impact any, feasibility int, method Object) int {
	if method["impact_aggregation"] == "coefficient_sum" {
		level, _ := coefficientRisk(impact, feasibility, method)
		return level
	}
	if method["risk_calculation"] == "product" {
		product := (dInt(impact) + 1) * feasibility
		for i, threshold := range dList(method["product_thresholds"]) {
			if product <= dInt(threshold) {
				return i + 1
			}
		}
		return 5
	}
	return dInt(dList(dList(method["matrix"])[dInt(impact)])[feasibility-1])
}
