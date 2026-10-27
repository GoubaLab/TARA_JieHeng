package assurance

// Assets are derived from project architecture, without consulting a catalog or
// inferring damage, threats, scores or safety decisions.
func architectureAssetKey(kind, id string) string {
	if kind == "software" || kind == "store" {
		kind = "child"
	}
	return kind + ":" + id
}
func assetSource(asset Object) Object { return dObj(asset["architecture_source"]) }
func assetText(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}
func diagramEndpoint(graph, node Object) bool {
	return node["kind"] == "entity" || (graph["view_mode"] == "dfd" && (node["kind"] == "process_step" || node["kind"] == "data_store"))
}
func architectureAssetCandidates(graph Object) []Object {
	result := []Object{}
	add := func(kind string, item Object, nodes []any, name, description string, properties []any) {
		result = append(result, Object{"name": assetText(name, 200), "description": assetText(description, 6000), "node_ids": dUnique(nodes), "properties": dUnique(properties), "reference": nil, "architecture_source": Object{"kind": kind, "id": item["id"], "status": "active"}})
	}
	for _, raw := range dList(graph["nodes"]) {
		node := dObj(raw)
		if !diagramEndpoint(graph, node) {
			continue
		}
		kind, suffix := "entity", " · 功能与服务"
		properties := []any{"integrity", "availability"}
		if node["kind"] == "process_step" {
			kind, suffix = "software", " · 处理过程"
		}
		if node["kind"] == "data_store" {
			kind, suffix = "store", " · 存储数据"
			properties = []any{"confidentiality", "integrity", "availability"}
		}
		add(kind, node, []any{node["id"]}, dStr(node["name"])+suffix, dStr(node["description"]), properties)
		for _, childRaw := range dList(node["children"]) {
			child := dObj(childRaw)
			kind := dStr(child["kind"])
			properties := []any{"integrity", "availability"}
			suffix := " · 软件"
			if kind == "store" {
				properties = []any{"confidentiality", "integrity", "availability"}
				suffix = " · 存储数据"
			}
			add(kind, child, []any{node["id"]}, dStr(child["name"])+suffix, dStr(child["description"]), properties)
		}
	}
	for _, raw := range dList(graph["flows"]) {
		flow := dObj(raw)
		if flow["channel_kind"] == "energy" {
			continue
		}
		properties := dList(dObj(flow["communication"])["security_properties"])
		if len(properties) == 0 {
			properties = []any{"confidentiality", "integrity", "availability"}
		}
		add("flow", flow, []any{flow["source"], flow["target"]}, dStr(flow["name"])+" · 传输数据", dStr(flow["description"]), properties)
	}
	return result
}

func syncArchitectureAssets(ws Object) {
	if dObj(ws["architecture"])["asset_generation"] == "catalog" {
		return
	}
	assets := dList(ws["assets"])
	bySource := map[string]Object{}
	ids := map[string]bool{}
	active := map[string]bool{}
	for _, raw := range dList(dObj(ws["architecture"])["nodes"]) {
		n := dObj(raw)
		if diagramEndpoint(dObj(ws["architecture"]), n) {
			ids[dStr(n["id"])] = true
		}
	}
	for _, raw := range assets {
		asset := dObj(raw)
		source := assetSource(asset)
		if len(source) > 0 {
			bySource[architectureAssetKey(dStr(source["kind"]), dStr(source["id"]))] = asset
		}
	}
	for _, candidate := range architectureAssetCandidates(dObj(ws["architecture"])) {
		source := assetSource(candidate)
		key := architectureAssetKey(dStr(source["kind"]), dStr(source["id"]))
		active[key] = true
		asset := bySource[key]
		if asset == nil {
			candidate["id"] = "asset_arch_" + Digest([]any{ws["id"], key})[:24]
			asset = candidate
			assets = append(assets, asset)
		}
		oldSource := assetSource(asset)
		defaults := dObj(oldSource["defaults"])
		overrides := dList(oldSource["overrides"])
		nextDefaults := Object{}
		for _, field := range []string{"name", "description", "node_ids", "properties"} {
			desired := dClone(candidate[field])
			if field == "properties" {
				// A graph edit cannot delete a property used by an existing risk scenario.
				for _, raw := range dList(ws["scenarios"]) {
					s := dObj(raw)
					if s["asset_id"] == asset["id"] {
						desired = dUnique(append(dList(desired), s["property"]))
					}
				}
			}
			if !dHas(overrides, field) && (len(defaults) == 0 || Digest(asset[field]) == Digest(defaults[field]) || (field == "node_ids" && oldSource["status"] == "missing")) {
				asset[field] = dClone(desired)
			}
			nextDefaults[field] = dClone(desired)
		}
		asset["architecture_source"] = Object{"kind": source["kind"], "id": source["id"], "status": "active", "defaults": nextDefaults, "overrides": dClone(overrides)}
	}
	for _, raw := range assets {
		asset := dObj(raw)
		source := assetSource(asset)
		if len(source) == 0 {
			continue
		}
		if !active[architectureAssetKey(dStr(source["kind"]), dStr(source["id"]))] {
			source["status"] = "missing"
		}
		// Keep the asset and its risk references when its graphical source disappears.
		// Drop only dangling entity links, so the graph remains valid and editable.
		linked := []any{}
		for _, id := range dList(asset["node_ids"]) {
			if ids[dStr(id)] {
				linked = append(linked, id)
			}
		}
		asset["node_ids"] = linked
	}
	ws["assets"] = assets
}
func architectureAssetSummary(before, after Object) Object {
	old := map[string]Object{}
	for _, raw := range dList(before["assets"]) {
		a := dObj(raw)
		old[dStr(a["id"])] = a
	}
	created, updated, missing := 0, 0, 0
	for _, raw := range dList(after["assets"]) {
		a := dObj(raw)
		if len(assetSource(a)) == 0 {
			continue
		}
		previous := old[dStr(a["id"])]
		if previous == nil {
			created++
		} else if Digest(previous) != Digest(a) {
			updated++
		}
		if assetSource(a)["status"] == "missing" {
			missing++
		}
	}
	return Object{"created": created, "updated": updated, "missing": missing, "total": len(dList(after["assets"]))}
}
