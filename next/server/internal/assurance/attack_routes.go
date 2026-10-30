package assurance

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Attack routes are directed communication candidates, never evidence of exploitability.
// Search is bounded; shortest means fewest data-flow hops within the stated search scope.
func attackRouteFingerprint(ws Object, route Object) string {
	graph := dObj(ws["architecture"])
	parts := []any{}
	for _, id := range dList(route["node_ids"]) {
		n, err := dFind(dList(graph["nodes"]), id)
		if err != nil {
			return "missing"
		}
		c := Clone(n)
		for _, k := range []string{"x", "y", "width", "height"} {
			delete(c, k)
		}
		parts = append(parts, c)
	}
	for _, id := range dList(route["flow_ids"]) {
		f, err := dFind(dList(graph["flows"]), id)
		if err != nil || f["channel_kind"] == "energy" {
			return "missing"
		}
		c := Clone(f)
		for _, k := range []string{"sourceSide", "targetSide", "labelOffset", "wireless_layout", "route_layout", "show_label", "emphasis"} {
			delete(c, k)
		}
		parts = append(parts, c)
	}
	return Digest(parts)
}
func attackRouteCurrent(ws Object, route Object) bool {
	return dBool(route["reviewed"]) && route["fingerprint"] == attackRouteFingerprint(ws, route)
}
func attackRouteTargetsScenario(ws, scenario, route Object) bool {
	nodes := dList(route["node_ids"])
	if len(nodes) == 0 {
		return false
	}
	target := nodes[len(nodes)-1]
	for _, assetID := range scenarioAssets(scenario) {
		asset, _ := dFind(dList(ws["assets"]), assetID)
		if dHas(asset["node_ids"], target) {
			return true
		}
	}
	return false
}
func validateAttackRoute(ws Object, route Object) error {
	ns, fs := dList(route["node_ids"]), dList(route["flow_ids"])
	if len(ns) != len(fs)+1 {
		return dError("路径节点与数据流数量不一致")
	}
	seen := map[string]bool{}
	for _, id := range ns {
		if seen[dStr(id)] {
			return dError("候选路径不能包含循环")
		}
		seen[dStr(id)] = true
	}
	for i, id := range fs {
		f, e := dFind(dList(dObj(ws["architecture"])["flows"]), id)
		if e != nil || f["source"] != ns[i] || f["target"] != ns[i+1] || f["channel_kind"] == "energy" {
			return dError("候选路径与当前数据流方向不一致，请重新查找")
		}
	}
	if route["fingerprint"] != attackRouteFingerprint(ws, route) {
		return dError("候选路径的架构依据已变更，请重新查找")
	}
	return nil
}
func clearPathRating(path Object) {
	path["feasibility"], path["rationale"] = nil, ""
	for _, key := range []string{"potential", "feasibility_method", "attack_vector"} {
		delete(path, key)
	}
}
func invalidateAttackRoutes(ws Object) {
	for _, s := range libRows(ws["scenarios"]) {
		for _, p := range libRows(s["paths"]) {
			if route := dObj(p["architecture_route"]); len(route) > 0 && (!attackRouteCurrent(ws, route) || !attackRouteTargetsScenario(ws, s, route)) {
				route["reviewed"] = false
				clearPathRating(p)
			}
		}
	}
}
func (s *HTTPServer) workspaceAttackRoutes(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "scenario_id", "entry_id", "target_id")
	if e != nil {
		return e
	}
	wid := r.PathValue("workspace_id")
	if e = s.readProject(r, wid); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), wid)
	if e != nil {
		return e
	}
	result, e := attackRouteCandidates(ws, q.Get("scenario_id"), q.Get("entry_id"), q.Get("target_id"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, result)
}
func attackRouteCandidates(ws Object, scenarioID, entryID, targetID string) (Object, error) {
	scenario, e := dFind(dList(ws["scenarios"]), scenarioID)
	if e != nil {
		return nil, dError("请选择关联威胁场景")
	}
	graph := dObj(ws["architecture"])
	nodes := map[string]Object{}
	adjacency := map[string][]Object{}
	entries, targets := []any{}, []any{}
	targetSet := map[string]bool{}
	for _, assetID := range scenarioAssets(scenario) {
		a, _ := dFind(dList(ws["assets"]), assetID)
		for _, id := range dList(a["node_ids"]) {
			targetSet[dStr(id)] = true
		}
	}
	for _, n := range libRows(graph["nodes"]) {
		if diagramEndpoint(graph, n) {
			nodes[dStr(n["id"])] = n
		}
	}
	ids := []string{}
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		n := nodes[id]
		option := Object{"id": id, "name": n["name"]}
		if n["kind"] == "external" || n["entityType"] == "external" || n["entityType"] == "actor" {
			entries = append(entries, option)
		}
		if targetSet[id] {
			targets = append(targets, option)
		}
	}
	if targetID != "" && !targetSet[targetID] {
		return nil, dError("目标必须是该威胁关联资产所在的实体")
	}
	if entryID != "" && nodes[entryID] == nil {
		return nil, dError("入口实体不存在")
	}
	for _, f := range libRows(graph["flows"]) {
		if f["channel_kind"] != "energy" && nodes[dStr(f["source"])] != nil && nodes[dStr(f["target"])] != nil {
			adjacency[dStr(f["source"])] = append(adjacency[dStr(f["source"])], f)
		}
	}
	for key := range adjacency {
		sort.Slice(adjacency[key], func(i, j int) bool { return dStr(adjacency[key][i]["id"]) < dStr(adjacency[key][j]["id"]) })
	}
	type state struct{ nodes, flows []any }
	queue := []state{}
	if entryID != "" {
		queue = append(queue, state{[]any{entryID}, nil})
	} else {
		for _, n := range entries {
			queue = append(queue, state{[]any{dObj(n)["id"]}, nil})
		}
	}
	items := []any{}
	signatures := map[string]bool{}
	minimum := map[string]int{}
	expanded := 0
	limited := false
	const maxDepth, maxStates, maxItems = 12, 10000, 20
	for len(queue) > 0 && expanded < maxStates && len(items) < maxItems {
		current := queue[0]
		queue = queue[1:]
		expanded++
		last := dStr(current.nodes[len(current.nodes)-1])
		reached := targetSet[last] && (targetID == "" || targetID == last)
		if reached && len(current.flows) > 0 {
			protocols, steps, protocolSequence := []string{}, []string{}, []string{}
			transitions := 0
			lastProtocol := ""
			for i, id := range current.flows {
				f, _ := dFind(dList(graph["flows"]), id)
				protocol := automationTechnology(f)
				if protocol == "" {
					protocol = "未确认协议"
				}
				protocols = append(protocols, protocol)
				if i == 0 || protocol != lastProtocol {
					protocolSequence = append(protocolSequence, protocol)
				}
				condition := "核实入口访问、授权及输入处理条件，并补充攻击动作"
				if i > 0 {
					condition = "核实上一节点受控后可发送此数据，并补充攻击动作"
					if protocol != lastProtocol {
						transitions++
						condition = "核实中间节点的跨协议转发与权限条件，并补充攻击动作"
					}
				}
				steps = append(steps, fmt.Sprintf("%d、%s → %s；%s（%s）。%s。", i+1, dStr(nodes[dStr(current.nodes[i])]["name"]), dStr(nodes[dStr(current.nodes[i+1])]["name"]), dStr(f["name"]), protocol, condition))
				lastProtocol = protocol
			}
			pair := dStr(current.nodes[0]) + "/" + last
			signature := pair + "/" + strings.Join(protocolSequence, ">")
			if !signatures[signature] {
				signatures[signature] = true
				if minimum[pair] == 0 {
					minimum[pair] = len(current.flows)
				}
				route := Object{"node_ids": current.nodes, "flow_ids": current.flows, "reviewed": false}
				route["fingerprint"] = attackRouteFingerprint(ws, route)
				items = append(items, Object{"id": Digest(route)[:16], "architecture_route": route, "name": dStr(nodes[dStr(current.nodes[0])]["name"]) + " → " + dStr(nodes[last]["name"]), "steps": strings.Join(steps, "\n"), "hops": len(current.flows), "protocols": protocols, "protocol_changes": transitions, "shortest": len(current.flows) == minimum[pair]})
			}
			continue
		}
		if len(current.flows) >= maxDepth {
			if len(adjacency[last]) > 0 {
				limited = true
			}
			continue
		}
		for _, f := range adjacency[last] {
			if !dHas(current.nodes, f["target"]) {
				if expanded+len(queue) >= maxStates {
					limited = true
					continue
				}
				queue = append(queue, state{append(append([]any{}, current.nodes...), f["target"]), append(append([]any{}, current.flows...), f["id"])})
			}
		}
	}
	limited = limited || len(queue) > 0
	allNodes := []any{}
	for _, id := range ids {
		allNodes = append(allNodes, Object{"id": id, "name": nodes[id]["name"]})
	}
	return Object{"base_revision": ws["revision"], "items": items, "entries": entries, "nodes": allNodes, "targets": targets, "truncated": limited, "max_hops": maxDepth, "max_candidates": maxItems, "selection": "fewest_hops_per_protocol_sequence", "notice": "仅沿有向数据流搜索，能量流不参与；同一入口和目标按协议序列保留最短候选。连通关系不证明攻击可行性，须核实每一步攻击动作和跨协议条件。"}, nil
}
