package assurance

import "fmt"

var numberPrefixes = Object{"entity": "EN", "boundary": "TB", "software": "SW", "store": "DS", "flow": "DF", "asset": "AS", "scenario": "RS", "path": "AP", "requirement": "SR", "evidence": "EV"}

func EffectiveNumbering(value Object) Object {
	result := Object{"prefixes": Clone(numberPrefixes), "separator": "-", "padding": 3}
	for key := range result {
		if v, ok := value[key]; ok {
			result[key] = dClone(v)
		}
	}
	return result
}
func pathNumberKey(scenarioID, pathID string) string {
	return fmt.Sprintf("%d:%s%s", len(scenarioID), scenarioID, pathID)
}
func nodeNumberKind(node Object) string {
	if node["kind"] == "boundary" {
		return "boundary"
	}
	if node["kind"] == "process_step" {
		return "software"
	}
	if node["kind"] == "store" || node["kind"] == "data_store" {
		return "store"
	}
	return "entity"
}

// Keep retired assignments: deleting or sorting records must not reuse or shift numbers.
// This registry is server-owned and separate from immutable relationship IDs.
func ensureRecordNumbers(ws Object) {
	state := dObj(ws["numbering_state"])
	if state == nil {
		state = Object{}
	}
	next := map[string]int{}
	for kind := range numberPrefixes {
		assignments := dObj(state[kind])
		if assignments == nil {
			assignments = Object{}
		}
		state[kind] = assignments
		for _, value := range assignments {
			if n := dInt(value); n > next[kind] {
				next[kind] = n
			}
		}
	}
	add := func(kind, id string) {
		assignments := dObj(state[kind])
		if id == "" || assignments == nil {
			return
		}
		if dInt(assignments[id]) == 0 {
			next[kind]++
			assignments[id] = next[kind]
		}
	}
	graph := dObj(ws["architecture"])
	for _, raw := range dList(graph["nodes"]) {
		node := dObj(raw)
		add(nodeNumberKind(node), dStr(node["id"]))
		for _, v := range dList(node["children"]) {
			child := dObj(v)
			add(dStr(child["kind"]), dStr(child["id"]))
		}
	}
	for _, raw := range dList(graph["flows"]) {
		add("flow", dStr(dObj(raw)["id"]))
	}
	for _, pair := range [][2]string{{"assets", "asset"}, {"scenarios", "scenario"}, {"requirements", "requirement"}, {"evidence", "evidence"}} {
		for _, raw := range dList(ws[pair[0]]) {
			row := dObj(raw)
			add(pair[1], dStr(row["id"]))
			if pair[1] == "scenario" {
				for _, p := range dList(row["paths"]) {
					add("path", pathNumberKey(dStr(row["id"]), dStr(dObj(p)["id"])))
				}
			}
		}
	}
	ws["numbering_state"] = state
}

// RecordCodes projects legacy snapshots without writing to them.
func RecordCodes(ws Object) func(string, any) string {
	copy := Clone(ws)
	ensureRecordNumbers(copy)
	state := dObj(copy["numbering_state"])
	rules := EffectiveNumbering(dObj(ws["numbering"]))
	return func(kind string, id any) string {
		n := dInt(dObj(state[kind])[dStr(id)])
		if n == 0 {
			return dStr(id)
		}
		return fmt.Sprintf("%s%s%0*d", dStr(dObj(rules["prefixes"])[kind]), dStr(rules["separator"]), dInt(rules["padding"]), n)
	}
}
