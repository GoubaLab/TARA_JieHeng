package assurance

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type dRequired struct{}
type dField struct {
	kind       string
	fallback   any
	nullable   bool
	min, max   int
	values     []string
	model      string
	item       *dField
	strict     bool
	omitAbsent bool
}

func ds(min, max int) dField {
	return dField{kind: "string", fallback: dRequired{}, min: min, max: max}
}
func dsd(value string, max int) dField { f := ds(0, max); f.fallback = value; return f }
func dsn() dField                      { f := ds(0, 0); f.nullable = true; f.fallback = nil; return f }
func de(values ...string) dField       { return dField{kind: "enum", fallback: dRequired{}, values: values} }
func ded(value any, values ...string) dField {
	f := de(values...)
	f.fallback = value
	f.nullable = value == nil
	return f
}
func dn(kind string, min, max int, strict bool) dField {
	return dField{kind: kind, fallback: dRequired{}, min: min, max: max, strict: strict}
}
func dnd(kind string, value any, min, max int, strict bool) dField {
	f := dn(kind, min, max, strict)
	f.fallback = value
	f.nullable = value == nil
	return f
}
func dl(item dField, min, max int, required bool) dField {
	f := dField{kind: "list", item: &item, min: min, max: max, fallback: []any{}}
	if required {
		f.fallback = dRequired{}
	}
	return f
}
func dm(model string, required bool) dField {
	f := dField{kind: "model", model: model, fallback: func() any { v, _ := dValidateModel(model, Object{}); return v }}
	if required {
		f.fallback = dRequired{}
	}
	return f
}
func db(value bool, strict bool) dField { return dField{kind: "bool", fallback: value, strict: strict} }
func dop(field dField) dField           { field.omitAbsent = true; return field }

var dProperties = []string{"confidentiality", "integrity", "availability", "authenticity", "authorization", "non_repudiation", "freshness", "privacy"}
var domainStamp = Stamp
var domainIdentity = Identity

func dSchema(model string) map[string]dField {
	switch model {
	case "WorkspaceProfile":
		return map[string]dField{"title": ds(1, 200), "description": ds(0, 4000)}
	case "NumberPrefixes":
		fields := map[string]dField{}
		for key := range numberPrefixes {
			fields[key] = ds(1, 12)
			if key == "damage" || key == "goal" {
				fields[key] = dsd(dStr(numberPrefixes[key]), 12)
			}
		}
		return fields
	case "Numbering":
		return map[string]dField{"prefixes": dm("NumberPrefixes", true), "separator": de("-", "_", ".", "/"), "padding": dn("int", 2, 6, true)}
	case "ItemFunction":
		return map[string]dField{"id": ds(1, 100), "name": ds(1, 300), "description": dsd("", 6000)}
	case "AttackFeasibility":
		return map[string]dField{"scenario_id": ds(1, 200), "path_id": ds(1, 200), "method": de("attack_potential", "attack_vector"), "potential": dop(dm("StandardAttackPotential", true)), "attack_vector": dop(de("network", "adjacent", "local", "physical")), "rationale": ds(1, 6000)}
	case "Item":
		return map[string]dField{"purpose": dsd("", 10000), "boundary": dsd("", 10000), "environment": dsd("", 10000), "assumptions": dsd("", 10000), "owner": dsd("", 200), "lifecycle": dsd("概念阶段", 100), "domain": dop(de("general", "automotive", "robotics", "iot", "charging", "internet"))}
	case "OperatingSystem":
		return map[string]dField{"family": ded("unknown", "unknown", "bare_metal", "rtos", "linux", "qnx", "windows", "android", "ios", "macos", "other"), "name": dsd("", 200), "version": dsd("", 200), "architecture": dsd("", 100), "notes": dsd("", 4000)}
	case "Communication":
		return map[string]dField{"medium": ded("unspecified", "unspecified", "wired", "wireless", "logical"), "technology": dsd("", 200), "range": ded("unspecified", "unspecified", "near_field", "local", "wide_area"), "purpose": dsd("", 4000), "security_properties": dl(de(dProperties...), 0, 8, false)}
	case "EntityAttribute":
		return map[string]dField{"name": ds(1, 200), "value": dsd("", 4000)}
	case "EntityChild":
		return map[string]dField{"id": ds(1, 80), "name": ds(1, 200), "kind": de("software", "store"), "description": dsd("", 4000), "reference": dsn()}
	case "Node":
		return map[string]dField{"id": ds(1, 80), "name": ds(1, 200), "kind": ded("entity", "entity", "process", "external", "store", "boundary", "process_step", "data_store"), "entityType": ded(nil, "ecu", "external", "service", "device", "application", "cloud", "actor"), "children": dl(dm("EntityChild", true), 0, 200, false), "attributes": dl(dm("EntityAttribute", true), 0, 100, false), "x": dn("float", -10000, 10000, false), "y": dn("float", -10000, 10000, false), "width": dnd("float", 180, 60, 3000, false), "height": dnd("float", 80, 40, 3000, false), "description": dsd("", 4000), "reference": dsn(), "operating_system": dop(dm("OperatingSystem", true))}
	case "LabelOffset":
		return map[string]dField{"x": dnd("float", 0, -10000, 10000, true), "y": dnd("float", 0, -10000, 10000, true)}
	case "RouteLayout":
		return map[string]dField{"waypoint": dop(dm("LabelOffset", true))}
	case "WirelessLayout":
		return map[string]dField{"source_offset": dop(dm("LabelOffset", true)), "target_offset": dop(dm("LabelOffset", true)), "source_side": dop(de("top", "right", "bottom", "left")), "target_side": dop(de("top", "right", "bottom", "left")), "waypoint": dop(dm("LabelOffset", true))}
	case "Flow":
		return map[string]dField{"id": ds(1, 80), "name": ds(1, 200), "source": ds(0, 0), "target": ds(0, 0), "sourceSide": ded(nil, "top", "right", "bottom", "left"), "targetSide": ded(nil, "top", "right", "bottom", "left"), "labelOffset": dm("LabelOffset", false), "protocol": dsd("", 200), "description": dsd("", 4000), "communication": dop(dm("Communication", true)), "wireless_layout": dop(dm("WirelessLayout", true)), "route_layout": dop(dm("RouteLayout", true)), "channel_kind": dop(de("data", "energy")), "emphasis": dop(de("normal", "critical", "internal")), "show_label": dop(db(true, true))}
	case "Architecture":
		return map[string]dField{"nodes": dl(dm("Node", true), 0, 200, false), "flows": dl(dm("Flow", true), 0, 400, false), "equal_spacing": db(true, false), "view_mode": dop(de("entities", "dfd")), "show_numbers": dop(db(true, true))}
	case "Asset":
		return map[string]dField{"id": dsn(), "name": ds(1, 200), "description": dsd("", 6000), "node_ids": dl(ds(0, 0), 0, 0, false), "properties": dl(de(dProperties...), 1, 0, true), "reference": dsn()}
	case "Impact":
		return map[string]dField{"safety": dnd("int", nil, 0, 3, true), "financial": dnd("int", nil, 0, 3, true), "operational": dnd("int", nil, 0, 3, true), "privacy": dnd("int", nil, 0, 3, true), "rationale": dsd("", 6000)}
	case "AttackPath":
		f := ds(0, 0)
		f.fallback = func() any { return domainIdentity("path") }
		return map[string]dField{"id": f, "name": ds(1, 500), "steps": dsd("", 10000), "feasibility": dnd("int", nil, 1, 4, true), "rationale": dsd("", 6000), "reference": dsn()}
	case "Scenario":
		return map[string]dField{"id": dsn(), "title": ds(1, 300), "asset_id": ds(0, 0), "property": de(dProperties...), "damage": ds(1, 10000), "threat": ds(1, 10000), "impact": dm("Impact", false), "paths": dl(dm("AttackPath", true), 0, 100, false), "references": dl(ds(0, 0), 0, 50, false)}
	case "Decision":
		return map[string]dField{"scenario_id": ds(0, 0), "treatment": de("reduce", "avoid", "share", "retain"), "rationale": ds(1, 8000), "owner": ds(1, 200), "goal": dsd("", 8000), "claim": dsd("", 8000), "residual_impact": dnd("number", nil, 0, 1200, true), "residual_feasibility": dnd("int", nil, 1, 4, true), "residual_rationale": dsd("", 8000)}
	case "Requirement", "RequirementDraft":
		m := map[string]dField{"id": dsn(), "scenario_id": ds(0, 0), "statement": ds(1, 10000), "allocation": ds(1, 500), "verification": ds(1, 8000), "standard_refs": dl(ds(0, 0), 0, 500, false), "reference": dsn()}
		if model == "RequirementDraft" {
			m["allocation"] = dsd("", 500)
			m["verification"] = dsd("", 8000)
		}
		return m
	case "RegulationSelection":
		return map[string]dField{"references": dl(ds(1, 250), 0, 2000, false)}
	case "Evidence":
		return map[string]dField{"id": dsn(), "requirement_id": ds(0, 0), "result": de("pass", "fail", "pending"), "reference": ds(1, 2000), "summary": ds(1, 10000), "reviewer": ds(1, 200)}
	case "Method":
		return map[string]dField{"name": ds(1, 200), "rationale": ds(1, 6000), "matrix": dl(dl(dn("int", 1, 5, true), 4, 4, true), 4, 4, true), "impact_aggregation": dop(de("maximum", "weighted_mean", "coefficient_sum")), "impact_weights": dop(dm("ImpactWeights", true)), "impact_coefficients": dop(dm("ImpactCoefficients", true)), "impact_dimensions": dop(dm("ImpactDimensions", true)), "impact_levels": dop(dl(dm("LevelDefinition", true), 4, 4, true)), "feasibility_levels": dop(dl(dm("LevelDefinition", true), 4, 4, true)), "risk_levels": dop(dl(dm("LevelDefinition", true), 5, 5, true)), "risk_calculation": dop(de("matrix", "product")), "product_thresholds": dop(dl(dn("number", 0, 4800, true), 4, 4, true))}
	case "ImpactWeights":
		return map[string]dField{"safety": dnd("int", 25, 0, 100, true), "financial": dnd("int", 25, 0, 100, true), "operational": dnd("int", 25, 0, 100, true), "privacy": dnd("int", 25, 0, 100, true)}
	case "ImpactCoefficients":
		return map[string]dField{"safety": dnd("number", 1, 0, 100, true), "financial": dnd("number", 1, 0, 100, true), "operational": dnd("number", 0.1, 0, 100, true), "privacy": dnd("number", 0.1, 0, 100, true)}
	case "ImpactDimensions":
		return map[string]dField{"safety": dLabel("安全"), "financial": dLabel("财务"), "operational": dLabel("运行"), "privacy": dLabel("隐私")}
	case "LevelDefinition":
		return map[string]dField{"label": ds(1, 100), "description": dsd("", 2000)}
	case "Baseline":
		return map[string]dField{"name": ds(1, 200), "reviewer": ds(1, 200), "note": ds(1, 10000)}
	case "ReferenceSelection":
		return map[string]dField{"asset_id": ds(1, 100), "reference": ds(0, 0), "relationship_keys": dl(ds(0, 0), 1, 100, true), "include_controls": db(true, true)}
	}
	return platformModel(model)
}
func dTrim(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}
func dValidateField(field dField, value any) (any, error) {
	if value == nil {
		if field.nullable {
			return nil, nil
		}
		return nil, dError("不得为空")
	}
	switch field.kind {
	case "number":
		check := field
		check.kind, check.strict = "float", true
		normalized, err := dValidateField(check, value)
		if err != nil {
			return nil, err
		}
		// Preserve integer-vs-float identity for legacy work products while new
		// decimal inputs are normalized to their persisted finite JSON value.
		n := normalized.(float64)
		switch v := value.(type) {
		case json.Number:
			if !strings.ContainsAny(string(v), ".eE") {
				return int(n), nil
			}
		default:
			kind := reflect.ValueOf(value).Kind()
			if kind >= reflect.Int && kind <= reflect.Uint64 {
				return int(n), nil
			}
		}
		return n, nil
	case "string":
		s, ok := value.(string)
		if !ok {
			return nil, dError("必须为字符串")
		}
		s = dTrim(s)
		n := utf8.RuneCountInString(s)
		if n < field.min || (field.max > 0 && n > field.max) {
			return nil, dError("文本长度不符合要求")
		}
		return s, nil
	case "enum":
		s, ok := value.(string)
		if !ok {
			return nil, dError("选项无效")
		}
		for _, allowed := range field.values {
			if s == allowed {
				return s, nil
			}
		}
		return nil, dError("选项无效")
	case "int":
		var n int64
		switch v := value.(type) {
		case json.Number:
			if strings.ContainsAny(string(v), ".eE") {
				return nil, dError("必须为整数")
			}
			var err error
			n, err = v.Int64()
			if err != nil {
				return nil, dError("整数超出范围")
			}
		default:
			rv := reflect.ValueOf(value)
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n = rv.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if rv.Uint() > math.MaxInt64 {
					return nil, dError("整数超出范围")
				}
				n = int64(rv.Uint())
			default:
				return nil, dError("必须为整数")
			}
		}
		if n < int64(field.min) || n > int64(field.max) {
			return nil, dError("整数超出范围")
		}
		return int(n), nil
	case "float":
		var n float64
		var err error
		switch v := value.(type) {
		case json.Number:
			n, err = v.Float64()
		case float64:
			n = v
		case float32:
			n = float64(v)
		case string:
			if field.strict {
				return nil, dError("必须为数值")
			}
			n, err = strconv.ParseFloat(dTrim(v), 64)
		case bool:
			if field.strict {
				return nil, dError("必须为数值")
			}
			if v {
				n = 1
			}
		default:
			rv := reflect.ValueOf(value)
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n = float64(rv.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n = float64(rv.Uint())
			default:
				return nil, dError("必须为数值")
			}
		}
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < float64(field.min) || n > float64(field.max) {
			return nil, dError("数值超出范围或不是有限数")
		}
		return n, nil
	case "bool":
		if b, ok := value.(bool); ok {
			return b, nil
		}
		if field.strict {
			return nil, dError("必须为布尔值")
		}
		s := ""
		switch v := value.(type) {
		case string:
			s = strings.ToLower(v)
		case json.Number:
			n, err := v.Float64()
			if err == nil && (n == 0 || n == 1) {
				s = strconv.Itoa(int(n))
			}
		case int:
			s = strconv.Itoa(v)
		case float64:
			if v == 0 || v == 1 {
				s = strconv.Itoa(int(v))
			}
		}
		switch s {
		case "0", "off", "f", "false", "n", "no":
			return false, nil
		case "1", "on", "t", "true", "y", "yes":
			return true, nil
		}
		return nil, dError("必须为布尔值")
	case "model":
		v, ok := value.(map[string]any)
		if !ok {
			return nil, dError("必须为对象")
		}
		return dValidateModel(field.model, v)
	case "list":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return nil, dError("必须为列表")
		}
		if rv.Len() < field.min || (field.max > 0 && rv.Len() > field.max) {
			return nil, dError("列表长度不符合要求")
		}
		out := make([]any, rv.Len())
		for i := range out {
			v, err := dValidateField(*field.item, rv.Index(i).Interface())
			if err != nil {
				return nil, dError(fmt.Sprintf("%d: %s", i, err))
			}
			out[i] = v
		}
		return out, nil
	}
	return nil, dError("未知校验类型")
}
func dValidateModel(model string, payload Object) (Object, error) {
	schema := taraSchema(model, dSchema(model))
	if schema == nil {
		return nil, dError("未知领域模型")
	}
	input := Clone(payload)
	if model == "Node" {
		kind, exists := input["kind"]
		if !exists {
			kind = "entity"
		}
		if kind == "process" || kind == "external" {
			input["kind"] = "entity"
			if _, ok := input["entityType"]; !ok {
				if kind == "external" {
					input["entityType"] = "external"
				} else {
					input["entityType"] = "ecu"
				}
			}
		} else if kind == "entity" {
			if _, ok := input["entityType"]; !ok {
				input["entityType"] = "ecu"
			}
		}
	}
	for key := range input {
		if _, ok := schema[key]; !ok {
			return nil, dError(key + ": 不允许额外字段")
		}
	}
	result := Object{}
	for key, field := range schema {
		value, present := input[key]
		if !present {
			if field.omitAbsent {
				continue
			}
			if _, required := field.fallback.(dRequired); required {
				return nil, dError(key + ": 必填字段")
			}
			if factory, ok := field.fallback.(func() any); ok {
				result[key] = factory()
			} else {
				result[key] = dClone(field.fallback)
			}
			continue
		}
		normalized, err := dValidateField(field, value)
		if err != nil {
			return nil, dError(key + ": " + err.Error())
		}
		result[key] = normalized
	}
	if err := workflowValidate(model, result); err != nil {
		return nil, err
	}
	switch model {
	case "NumberPrefixes":
		seen := map[string]bool{}
		pattern := regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}_-]{0,11}$`)
		for _, v := range result {
			prefix := dStr(v)
			key := strings.ToLower(dCasefold(prefix))
			if !pattern.MatchString(prefix) {
				return nil, dError("编号前缀须以文字或数字开头，仅可包含文字、数字、下划线和短横线，最长 12 个字符")
			}
			if seen[key] {
				return nil, dError("不同类型的编号前缀不能重复（不区分大小写）")
			}
			seen[key] = true
		}
	case "Node":
		if result["kind"] == "entity" && result["entityType"] == nil {
			return nil, dError("实体必须选择实体类型")
		}
		if result["kind"] != "entity" && (len(dList(result["children"])) > 0 || len(dList(result["attributes"])) > 0 || result["entityType"] != nil) {
			return nil, dError("只有实体可以包含软件、数据存储和下级属性")
		}
		if _, present := result["operating_system"]; present && result["kind"] != "entity" {
			return nil, dError("只有实体可以配置操作系统")
		}
		seen := map[string]bool{}
		for _, a := range dList(result["attributes"]) {
			name := dCasefold(dStr(dObj(a)["name"]))
			if seen[name] {
				return nil, dError("同一实体的属性名称不能重复")
			}
			seen[name] = true
		}
	case "Architecture":
		seen, entities := map[string]bool{}, map[string]bool{}
		identifiers := []string{}
		for _, n := range dList(result["nodes"]) {
			node := dObj(n)
			identifiers = append(identifiers, dStr(node["id"]))
			for _, c := range dList(node["children"]) {
				identifiers = append(identifiers, dStr(dObj(c)["id"]))
			}
			if node["kind"] == "store" {
				return nil, dError("数据存储必须归属实体，请先为旧数据存储选择所属实体")
			}
			if (node["kind"] == "process_step" || node["kind"] == "data_store") && result["view_mode"] != "dfd" {
				return nil, dError("处理过程和独立数据存储仅用于展开 DFD 模式")
			}
			if diagramEndpoint(result, node) {
				entities[dStr(node["id"])] = true
			}
		}
		for _, f := range dList(result["flows"]) {
			flow := dObj(f)
			identifiers = append(identifiers, dStr(flow["id"]))
			if !entities[dStr(flow["source"])] || !entities[dStr(flow["target"])] || flow["source"] == flow["target"] {
				return nil, dError("连接必须使用当前模式的两个不同端点；实体模式连接实体，展开 DFD 可连接处理过程和独立数据存储；信任边界不能作为端点")
			}
		}
		for _, id := range identifiers {
			if seen[id] {
				return nil, dError("图内标识不能重复")
			}
			seen[id] = true
		}
	case "Scenario":
		seen := map[string]bool{}
		for _, p := range dList(result["paths"]) {
			id := dStr(dObj(p)["id"])
			if seen[id] {
				return nil, dError("攻击路径标识不能重复")
			}
			seen[id] = true
		}
	case "RegulationSelection":
		result["references"] = dUnique(dList(result["references"]))
	case "Communication":
		result["security_properties"] = dUnique(dList(result["security_properties"]))
	case "Method":
		if result["impact_aggregation"] == "coefficient_bands" {
			if result["risk_calculation"] != "matrix" || len(dList(result["impact_scale"])) != 4 || len(dList(result["impact_thresholds"])) != 3 {
				return nil, dError("系数分档法需要四级原始取值、三个影响阈值和风险矩阵")
			}
		}
		for _, key := range []string{"impact_scale", "impact_thresholds", "attack_potential_thresholds"} {
			values := dList(result[key])
			for i := 1; i < len(values); i++ {
				if decimalRat(values[i]).Cmp(decimalRat(values[i-1])) <= 0 {
					return nil, dError(key + " 必须严格递增")
				}
			}
		}
		matrix := dList(result["matrix"])
		for i := 0; i < 4; i++ {
			for j := 0; j < 4; j++ {
				if (j < 3 && dInt(dList(matrix[i])[j]) > dInt(dList(matrix[i])[j+1])) || (i < 3 && dInt(dList(matrix[i])[j]) > dInt(dList(matrix[i+1])[j])) {
					return nil, dError("影响或可行性增加时，风险值不得降低")
				}
			}
		}
		effective := EffectiveMethod(result)
		coefficient := effective["impact_aggregation"] == "coefficient_sum"
		if coefficientTotal(effective).Sign() <= 0 {
			return nil, dError("至少需要一个非零影响系数")
		}
		if coefficient {
			if effective["risk_calculation"] != "product" {
				return nil, dError("系数总分法必须使用总分乘可行性的风险计算方式")
			}
		}
		if effective["impact_aggregation"] == "weighted_mean" {
			total := 0
			for _, key := range impactKeys {
				total += dInt(dObj(effective["impact_weights"])[key])
			}
			if total == 0 {
				return nil, dError("加权平均至少需要一个非零影响权重")
			}
		}
		thresholds := dList(effective["product_thresholds"])
		if !coefficient {
			for _, threshold := range thresholds {
				if _, err := dValidateField(dn("int", 1, 15, true), threshold); err != nil {
					return nil, dError("普通等级算法的风险阈值必须为1至15的整数")
				}
			}
		}
		for i := 1; i < len(thresholds); i++ {
			if decimalRat(thresholds[i]).Cmp(decimalRat(thresholds[i-1])) <= 0 {
				return nil, dError("乘积风险阈值必须严格递增")
			}
		}
	case "ReferenceSelection":
		if !regexp.MustCompile(`^systems:[0-9]+$`).MatchString(dStr(result["reference"])) {
			return nil, dError("请选择组件参考条目")
		}
		values := dList(result["relationship_keys"])
		if len(dUnique(values)) != len(values) {
			return nil, dError("请选择不重复的关联知识条目")
		}
		for _, key := range values {
			if !strings.HasPrefix(dStr(key), "relationships:") {
				return nil, dError("只允许选择组件关联知识条目")
			}
		}
	}
	return result, nil
}
func dUnique(values []any) []any {
	out := []any{}
	seen := map[string]bool{}
	for _, v := range values {
		key := dStr(v)
		if !seen[key] {
			seen[key] = true
			out = append(out, v)
		}
	}
	return out
}

// Unicode full folding is generated from the same Python Unicode data as the
// legacy validator; ASCII and ordinary Unicode lowercasing cover other runes.
func dCasefold(s string) string {
	var b strings.Builder
	for _, r := range s {
		if v, ok := domainCasefold[r]; ok {
			b.WriteString(v)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func ValidatePayload(action string, payload Object) (Object, error) {
	models := map[string]string{"update-workspace-profile": "WorkspaceProfile", "configure-numbering": "Numbering", "define-item": "Item", "replace-architecture": "Architecture", "record-asset": "Asset", "record-scenario": "Scenario", "decide-treatment": "Decision", "specify-requirement": "Requirement", "select-regulations": "RegulationSelection", "record-evidence": "Evidence", "adopt-method": "Method", "seal-baseline": "Baseline", "apply-reference-selection": "ReferenceSelection"}
	model, ok := models[action]
	if action == "record-damage" {
		model, ok = "DamageScenario", true
	}
	if action == "record-goal" {
		model, ok = "SecurityGoal", true
	}
	if !ok {
		return nil, dError("未知领域操作")
	}
	return dValidateModel(model, payload)
}
