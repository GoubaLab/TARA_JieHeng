package assurance

import (
	"math/big"
	"sort"
	"strings"
)

func hasTaraLinks(ws Object) bool {
	if len(dList(ws["damage_scenarios"])) > 0 || len(dList(ws["security_goals"])) > 0 {
		return true
	}
	for _, raw := range dList(ws["scenarios"]) {
		s := dObj(raw)
		for _, k := range []string{"asset_ids", "properties", "damage_ids"} {
			if _, ok := s[k]; ok {
				return true
			}
		}
		for _, p := range dList(s["paths"]) {
			if len(dObj(dObj(p)["potential"])) > 0 {
				return true
			}
		}
	}
	for _, raw := range dList(ws["requirements"]) {
		r := dObj(raw)
		if r["scenario_ids"] != nil || r["goal_ids"] != nil {
			return true
		}
	}
	for _, raw := range dList(ws["decisions"]) {
		if dObj(raw)["goal_ids"] != nil {
			return true
		}
	}
	return false
}

// Compare damage scenarios using the selected algorithm, including fractional
// coefficient totals. A high grade on a low-coefficient dimension is not a high total.
func damageImpactOrder(impact, method Object) (*big.Rat, bool) {
	if method["impact_aggregation"] == "coefficient_sum" {
		return coefficientImpact(impact, method)
	}
	level, ok := aggregateImpact(impact, method)
	if method["impact_aggregation"] == "coefficient_bands" {
		level, _, ok = bandedImpact(impact, method)
	}
	return big.NewRat(int64(level), 1), ok
}
func taraRecordCode(record Object) string {
	if s := dStr(record["source_code"]); s != "" {
		return s
	}
	return dStr(record["id"])
}
func linkedGoalText(ws Object, ids []any) string {
	parts := []string{}
	for _, id := range ids {
		g, e := dFind(dList(ws["security_goals"]), id)
		if e == nil {
			parts = append(parts, taraRecordCode(g)+" "+dStr(g["statement"]))
		}
	}
	return strings.Join(parts, "\n")
}
func linkedGoalsHash(ws Object, ids []any) string {
	goals := []any{}
	for _, id := range ids {
		g, e := dFind(dList(ws["security_goals"]), id)
		if e != nil {
			goals = append(goals, Object{"missing": id})
		} else {
			goals = append(goals, g)
		}
	}
	sort.Slice(goals, func(i, j int) bool { return Digest(goals[i]) < Digest(goals[j]) })
	return Digest(goals)
}
func goalDecisionStale(ws, decision Object) bool {
	ids := dList(decision["goal_ids"])
	if len(ids) == 0 {
		return false
	}
	if decision["goal_hash"] != nil {
		return decision["goal_hash"] != linkedGoalsHash(ws, ids)
	}
	// Earlier imported workspaces stored the approved goal text only.
	return dStr(decision["goal"]) != linkedGoalText(ws, ids)
}
func preserveGoalSnapshots(ws Object) {
	for _, raw := range dList(ws["decisions"]) {
		d := dObj(raw)
		ids := dList(d["goal_ids"])
		if len(ids) > 0 && d["goal_hash"] == nil {
			if goalDecisionStale(ws, d) {
				d["goal_hash"] = "unreviewed"
			} else {
				d["goal_hash"] = linkedGoalsHash(ws, ids)
			}
		}
	}
}

// Legacy single-scenario requirements retain their original fingerprint. Extended
// TARA verification also depends on the linked goals and assessment context.
func requirementEvidenceHash(ws, requirement Object) string {
	if requirement["goal_ids"] == nil && requirement["scenario_ids"] == nil {
		return Digest(requirement)
	}
	contexts := []any{}
	for _, id := range requirementScenarios(requirement) {
		s, e := dFind(dList(ws["scenarios"]), id)
		if e == nil {
			contexts = append(contexts, Object{"id": id, "assessment_hash": AssessmentHash(ws, s)})
		} else {
			contexts = append(contexts, Object{"missing": id})
		}
	}
	sort.Slice(contexts, func(i, j int) bool { return Digest(contexts[i]) < Digest(contexts[j]) })
	return Digest([]any{requirement, linkedGoalsHash(ws, dList(requirement["goal_ids"])), contexts})
}
func scenarioVerified(ws Object, scenarioID any) bool {
	count := 0
	for _, raw := range dList(ws["requirements"]) {
		r := dObj(raw)
		if !dHas(requirementScenarios(r), scenarioID) {
			continue
		}
		count++
		var latest Object
		for _, v := range dList(ws["evidence"]) {
			e := dObj(v)
			if e["requirement_id"] == r["id"] {
				latest = e
			}
		}
		if latest == nil || latest["result"] != "pass" || latest["requirement_hash"] != requirementEvidenceHash(ws, r) {
			return false
		}
	}
	return count > 0
}
