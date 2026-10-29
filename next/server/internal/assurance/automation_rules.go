package assurance

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const automationRuleVersion = "jieheng.automation/1"

// Review rules match recorded model facts and identify assessment topics.
// Protection properties identify requirements; control status is assessed separately.
type automationRule struct {
	id, title, property, stride, threat, damage, steps, verification string
	prerequisites, controls                                          []string
}

func automationRules() []automationRule {
	return []automationRule{
		{"DF-TRUST-INPUT", "跨信任域输入完整性复核", "integrity", "Tampering",
			"若接收端信任来自不同信任域的输入且校验不足，输入可能被替换或构造，从而影响处理结果。",
			"若关键输入被错误接纳，可能产生不正确的处理或控制结果；实际业务后果需确认。",
			"确认跨域输入的可达入口；在授权测试环境提交异常或未经授权的数据；检查接收端是否拒绝并记录。",
			"使用正常、格式异常、越权与被修改的输入进行边界测试，记录拒绝结果和审计证据。",
			[]string{"接收方使用该输入影响业务处理或设备行为", "输入可被不同信任主体控制；需核实实际访问条件"},
			[]string{"发送方身份与授权校验", "边界输入的长度、格式、范围和状态校验", "校验失败时的安全处置与审计"}},
		{"DF-WIRELESS-INTEGRITY", "无线通道完整性复核", "integrity", "Tampering",
			"若无线链路或应用层未有效确认消息来源与完整性，具备相应接入条件的主体可能注入或修改传输数据。",
			"若无线数据被错误接纳，可能导致接收端处理结果偏离预期；影响程度待业务分析。",
			"确认无线接入和配对条件；核对消息鉴别覆盖范围；在隔离测试环境验证被修改或伪造的消息是否被拒绝。",
			"核对配对、链路和应用鉴别配置，以篡改消息和未授权发送方用例验证拒绝行为。",
			[]string{"无线接口在目标生命周期阶段可达", "接收端依赖该数据；控制实施情况尚需核对"},
			[]string{"配对或接入身份校验", "消息完整性与来源鉴别", "密钥配置及异常消息拒绝策略"}},
		{"IF-DEBUG-ACCESS", "调试接口写入权限复核", "integrity", "Elevation of Privilege",
			"若调试或维护接口在目标生命周期仍可达且权限控制不足，接口使用者可能修改程序、配置或运行状态。",
			"若未授权修改通过维护接口进入，可能破坏设备功能或配置；需确认接口实际写入能力。",
			"确认接口引脚或逻辑入口与生命周期状态；核对允许的调试命令；在授权设备验证未授权写入的拒绝行为。",
			"分别检查开发、生产和维修状态，记录身份校验、写入限制和生命周期锁定的测试结果。",
			[]string{"接口实际支持调试、维护或写入，不仅是普通串行数据", "攻击者能够接触物理或代理入口"},
			[]string{"生产状态的调试权限与生命周期锁定", "维护身份、授权和会话限制", "敏感写入的审计与恢复"}},
		{"OS-NETWORK-SURFACE", "联网操作系统攻击面复核", "integrity", "Elevation of Privilege",
			"若联网实体的网络服务、解析组件或权限隔离存在缺陷，可达服务可能成为修改系统状态的入口；CVE 适用性需核对组件、版本和配置。",
			"若服务被利用或权限隔离失效，可能导致系统配置或执行状态被未授权修改；实际后果需确认。",
			"建立操作系统和网络服务清单；核实实际版本、配置与可达性；按最小权限及输入处理要求验证服务边界。",
			"核对软件物料及维护版本、服务监听范围和权限，引用实测或公告适用性核验记录，不以 OS 名称直接判定漏洞。",
			[]string{"实体确实运行所登记的操作系统", "数据流对应实际启用的服务或协议栈", "漏洞适用性须另行核实组件、版本、配置及前置条件"},
			[]string{"服务最小化及访问范围", "服务账户权限与进程隔离", "组件版本清单、更新与漏洞适用性核验"}},
		{"DS-CONFIDENTIALITY", "存储数据读取权限复核", "confidentiality", "Information Disclosure",
			"若存储中包含受保护数据且读取路径的授权或保护不足，未授权主体可能读取数据副本或明文内容。",
			"若受保护数据被读取，可能造成信息或隐私泄露；数据类别和实际后果待确认。",
			"确认存储内容和读出路径；核对运行、备份、维修及退役阶段权限；用未授权主体验证读取限制。",
			"记录数据分类与访问矩阵，验证未授权读取被拒绝，检查备份、密钥分离和退役清除要求。",
			[]string{"存储实际包含需要保密的数据", "存在可被未授权主体接触的读取、备份或物理路径"},
			[]string{"数据分类和最小读取权限", "必要的静态数据保护及独立密钥管理", "备份、维修和退役阶段保护"}},
		{"DF-FRESHNESS", "数据流新鲜性复核", "freshness", "Tampering",
			"若接收端需要消息新鲜性而未能区分旧消息与当前消息，重复或延迟到达的数据可能被当作当前有效输入。",
			"若过期输入被错误接纳，可能产生与当前状态不一致的操作；时间容限和业务后果待确认。",
			"确认允许的时延和重试语义；记录合法消息；在隔离环境验证重复、乱序及过期消息的处理。",
			"使用重复、乱序、过期和重连后的消息用例，按已确认的时间窗口及状态要求核对接收结果。",
			[]string{"已声明 freshness 保护需求", "业务确实区分新旧消息，正常重试语义须保留"},
			[]string{"序列或状态关联及重复检测", "可信时间窗口或等价新鲜性机制", "重连、重启和计数状态恢复策略"}},
		{"DF-AVAILABILITY", "数据流可用性复核", "availability", "Denial of Service",
			"若关键数据流受到异常负载、连续错误或连接中断，且资源隔离与恢复不足，接收端可能不能及时提供服务。",
			"若必要数据无法按时到达或处理，可能导致服务降级或中断；容忍时限与后果需确认。",
			"确认流量上限、时限和降级要求；在隔离测试环境模拟超额负载或断连；观察资源隔离及恢复。",
			"使用约定负载边界、异常输入与断连恢复用例，测量响应、资源上限及降级恢复结果。",
			[]string{"已声明 availability 保护需求", "需明确时限、容量和可接受降级状态"},
			[]string{"输入限额、队列及资源隔离", "超时、背压和重试上限", "故障降级与恢复验证"}},
		{"SW-UPDATE-INTEGRITY", "软件更新内容完整性复核", "integrity", "Tampering",
			"若传输用途包含软件更新且接收端对内容、目标或版本授权校验不足，非预期更新可能进入安装流程。",
			"若错误或未授权软件被安装，可能改变功能或降低既有保护；实际后果待确认。",
			"确认该通道承载更新包或安装命令；核对内容来源与目标版本约束；验证篡改、错误目标和中断更新的拒绝或恢复。",
			"执行内容篡改、错误目标、未授权版本和更新中断用例，记录校验及恢复结果。",
			[]string{"用途字段确实指向软件或固件更新", "安装入口和更新授权边界需人工核实"},
			[]string{"更新内容来源和完整性验证", "目标设备与版本授权检查", "中断恢复及失败回退策略"}},
	}
}

// AutomationPlan is a deterministic, read-only analysis of recorded facts. It
// intentionally does not infer trust membership from a boundary rectangle.
func AutomationPlan(ws Object) Object {
	rules := automationRules()
	byRule := map[string]automationRule{}
	for _, rule := range rules {
		byRule[rule.id] = rule
	}
	graph := dObj(ws["architecture"])
	nodes := map[string]Object{}
	for _, raw := range dList(graph["nodes"]) {
		n := dObj(raw)
		if n["kind"] != "boundary" {
			nodes[dStr(n["id"])] = n
		}
	}
	candidates := []any{}
	add := func(ruleID string, subject Object, facts []any) {
		rule := byRule[ruleID]
		id := "ac_" + Digest([]any{ruleID, subject["kind"], subject["id"]})[:24]
		options, selected := automationAssets(ws, subject, rule.property)
		title := assetText(dStr(subject["name"])+" · "+rule.title, 300)
		existing, related := automationCoverage(ws, id, title, rule, selected)
		warnings := []any{"待核对攻击前提与控制实施情况。"}
		if len(options) == 0 {
			warnings = append(warnings, "缺少具有对应安全属性的关联资产，请保存架构或完善资产。")
		}
		if len(options) > 0 && len(selected) == 0 {
			warnings = append(warnings, "匹配到多个手工资产，请明确资产的架构来源。")
		}
		if len(existing) > 0 {
			warnings = append(warnings, "已有关联分析，跳过重复创建。")
		}
		candidates = append(candidates, Object{
			"id": id, "rule_id": rule.id, "rule_version": automationRuleVersion,
			"title": title, "subject": Clone(subject), "matched_conditions": facts,
			"prerequisites": automationStrings(rule.prerequisites), "controls_to_check": automationStrings(rule.controls),
			"property": rule.property, "stride": rule.stride, "asset_options": options, "asset_ids": selected,
			"selectable": len(selected) > 0 && len(existing) == 0, "already_covered": len(existing) > 0,
			"existing_scenario_ids": existing, "related_scenario_ids": related, "warnings": warnings,
		})
	}
	incident := map[string][]any{}
	for _, raw := range dList(graph["flows"]) {
		f := dObj(raw)
		source, target := nodes[dStr(f["source"])], nodes[dStr(f["target"])]
		if f["channel_kind"] == "energy" || source == nil || target == nil || f["source"] == f["target"] {
			continue
		}
		comm := dObj(f["communication"])
		tech := automationTechnology(f)
		subject := Object{"kind": "flow", "id": f["id"], "name": f["name"], "node_ids": []any{f["source"], f["target"]}}
		prefix := "architecture.flows[" + dStr(f["id"]) + "]"
		baseFacts := []any{automationFact(prefix+".source", "exists", f["source"], "已记录的发送端"), automationFact(prefix+".target", "exists", f["target"], "已记录的接收端")}
		zoneS, zoneT := automationZone(source), automationZone(target)
		if (zoneS != "" && zoneT != "" && !strings.EqualFold(zoneS, zoneT)) || automationExternal(source) != automationExternal(target) {
			facts := append(append([]any{}, baseFacts...), automationFact("endpoints.trust_context", "different", Object{"source_zone": zoneS, "target_zone": zoneT, "source_external": automationExternal(source), "target_external": automationExternal(target)}, "连接两端的信任域或外部实体身份不同"))
			add("DF-TRUST-INPUT", subject, facts)
		}
		wireless := comm["medium"] == "wireless" || automationIn(tech, "wifi", "bluetooth", "ble", "nfc", "uwb", "zigbee", "lora", "cellular", "4g", "5g")
		if wireless {
			add("DF-WIRELESS-INTEGRITY", subject, append(append([]any{}, baseFacts...), automationFact(prefix+".communication", "wireless", Object{"medium": comm["medium"], "technology": tech}, "通信介质或协议为无线")))
		}
		if automationIn(tech, "jtag", "dap", "swd", "uart") {
			add("IF-DEBUG-ACCESS", subject, append(append([]any{}, baseFacts...), automationFact(prefix+".protocol", "debug_or_serial", tech, "接口类型为调试或串行接口，实际用途待确认")))
		}
		for property, ruleID := range map[string]string{"freshness": "DF-FRESHNESS", "availability": "DF-AVAILABILITY"} {
			if dHas(dList(comm["security_properties"]), property) {
				add(ruleID, subject, append(append([]any{}, baseFacts...), automationFact(prefix+".communication.security_properties", "contains", property, "数据流已设置该安全属性")))
			}
		}
		if automationUpdatePurpose(dStr(comm["purpose"])) {
			add("SW-UPDATE-INTEGRITY", subject, append(append([]any{}, baseFacts...), automationFact(prefix+".communication.purpose", "update_keyword", comm["purpose"], "用途包含更新标识，实际承载内容待核实")))
		}
		if wireless || automationIn(tech, "ethernet", "doip", "tcp", "udp", "http", "https", "mqtt") {
			incident[dStr(source["id"])] = append(incident[dStr(source["id"])], f["id"])
			incident[dStr(target["id"])] = append(incident[dStr(target["id"])], f["id"])
		}
	}
	for _, node := range nodes {
		kind := dStr(node["kind"])
		if automationExternal(node) {
			continue
		}
		subject := Object{"kind": "node", "id": node["id"], "name": node["name"], "node_ids": []any{node["id"]}, "node_kind": kind}
		os := dObj(node["operating_system"])
		if automationIn(dStr(os["family"]), "linux", "rtos", "qnx", "windows", "android", "ios", "macos") && len(incident[dStr(node["id"])]) > 0 {
			links := automationSortedValues(incident[dStr(node["id"])])
			add("OS-NETWORK-SURFACE", subject, []any{automationFact("architecture.nodes["+dStr(node["id"])+"].operating_system", "known_family", Clone(os), "已登记操作系统类型"), automationFact("architecture.flows", "connected_network", links, "实体存在已记录的网络或无线数据流")})
		}
		if kind == "data_store" || kind == "store" {
			add("DS-CONFIDENTIALITY", subject, []any{automationFact("architecture.nodes["+dStr(node["id"])+"].kind", "equals", kind, "图元明确为数据存储；数据分类待核实")})
		}
		for _, raw := range dList(node["children"]) {
			child := dObj(raw)
			if child["kind"] != "store" {
				continue
			}
			childSubject := Object{"kind": "child", "id": child["id"], "name": child["name"], "node_ids": []any{node["id"]}, "owner_id": node["id"], "child_kind": "store"}
			add("DS-CONFIDENTIALITY", childSubject, []any{automationFact("architecture.nodes["+dStr(node["id"])+"].children["+dStr(child["id"])+"].kind", "equals", "store", "实体内部明确登记了存储子项")})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return dStr(dObj(candidates[i])["id"]) < dStr(dObj(candidates[j])["id"]) })
	selectable, covered, missing := 0, 0, 0
	for _, raw := range candidates {
		c := dObj(raw)
		if dBool(c["selectable"]) {
			selectable++
		}
		if dBool(c["already_covered"]) {
			covered++
		}
		if len(dList(c["asset_ids"])) == 0 {
			missing++
		}
	}
	return Object{"base_revision": ws["revision"], "rule_version": automationRuleVersion,
		"plan_hash": Digest(Object{"rules": automationRuleVersion, "workspace": automationSemanticSnapshot(ws), "candidates": candidates}),
		"summary":   Object{"total": len(candidates), "selectable": selectable, "already_covered": covered, "missing_assets": missing}, "candidates": candidates}
}

// AutomationProposal only produces new, unrated analysis records linked to
// existing assets. Caller checks the plan revision/hash before transactional
// adoption through validateAIProposal; existing human work is never overwritten.
func AutomationProposal(ws Object, candidateIDs []string) (Object, error) {
	if len(candidateIDs) == 0 || len(candidateIDs) > 50 {
		return nil, dError("请选择 1–50 个自动分析候选")
	}
	plan := AutomationPlan(ws)
	byID := map[string]Object{}
	for _, raw := range dList(plan["candidates"]) {
		c := dObj(raw)
		byID[dStr(c["id"])] = c
	}
	byRule := map[string]automationRule{}
	for _, r := range automationRules() {
		byRule[r.id] = r
	}
	selected := map[string]bool{}
	ids := []string{}
	for _, id := range candidateIDs {
		c := byID[id]
		if c == nil {
			return nil, dError("自动分析候选不存在或已不适用：" + id)
		}
		if !dBool(c["selectable"]) {
			return nil, dError("自动分析候选不可采纳，请核对资产关联或已有场景：" + id)
		}
		if !selected[id] {
			selected[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	proposal := Object{"summary": fmt.Sprintf("生成 %d 项分析草案，等待评估与复核。", len(ids)),
		"assumptions": []any{"分析依据为已保存的实体、接口、操作系统和安全属性。", "攻击前提、控制实施情况和业务后果待核实。"},
		"questions":   []any{"评估损害影响与攻击可行性，并填写评分依据。", "确认 CAL、关联安全目标，再由负责人完成风险处置和验证。"},
		"citations":   []any{}, "assets": []any{}, "damage_scenarios": []any{}, "scenarios": []any{}, "security_goals": []any{}, "requirements": []any{}, "treatments": []any{}}
	for _, id := range ids {
		c := byID[id]
		rule := byRule[dStr(c["rule_id"])]
		assetIDs := dList(c["asset_ids"])
		code := "AUTO:" + id
		damageID := automationExistingDamage(ws, code, rule.damage, assetIDs, rule.property)
		if damageID == "" {
			damageID = "auto_damage_" + id
			proposal["damage_scenarios"] = append(dList(proposal["damage_scenarios"]), Object{"id": damageID, "source_code": code, "description": rule.damage, "consequence": "业务后果、影响对象和运行条件待确认。", "asset_ids": dClone(assetIDs), "properties": []any{rule.property}, "impact": automationUnknownImpact()})
		}
		scenarioID := "auto_scenario_" + id
		proposal["scenarios"] = append(dList(proposal["scenarios"]), Object{
			"id": scenarioID, "source_code": code, "title": c["title"], "asset_id": assetIDs[0], "asset_ids": dClone(assetIDs),
			"property": rule.property, "properties": []any{rule.property}, "damage": rule.damage, "damage_ids": []any{damageID}, "threat": rule.threat,
			"stride": rule.stride, "cause": "规则 " + rule.id + "（" + automationRuleVersion + "）；适用前提与控制状态待确认。",
			"impact": automationUnknownImpact(), "paths": []any{Object{"name": "待验证路径：" + rule.title, "steps": rule.steps, "feasibility": nil, "rationale": "入口可达性、所需资源和可行性等级待评估。", "actor": "具备入口访问条件的主体（待确认）", "vector": assetText(dStr(dObj(c["subject"])["name"]), 200)}}})
		proposal["requirements"] = append(dList(proposal["requirements"]), Object{
			"id": "auto_requirement_" + id, "source_code": code, "scenario_id": scenarioID, "scenario_ids": []any{scenarioID},
			"statement": "待确认适用范围后落实：" + strings.Join(rule.controls, "；") + "。", "allocation": assetText(dStr(dObj(c["subject"])["name"])+"（分配责任边界待确认）", 500), "verification": rule.verification, "standard_refs": []any{}})
		proposal["questions"] = append(dList(proposal["questions"]), assetText(dStr(c["title"])+"："+strings.Join(rule.prerequisites, "；"), 2000))
	}
	return proposal, nil
}

func automationFact(field, operator string, value any, explanation string) Object {
	return Object{"field": field, "operator": operator, "value": value, "explanation": explanation}
}
func automationStrings(values []string) []any {
	result := []any{}
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
func automationIn(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func automationExternal(n Object) bool {
	return n["kind"] == "external" || n["entityType"] == "external"
}
func automationZone(n Object) string {
	zone := ""
	for _, raw := range dList(n["attributes"]) {
		a := dObj(raw)
		key := strings.ToLower(strings.TrimSpace(dStr(a["name"])))
		if !automationIn(key, "trust_zone", "trust zone", "信任域", "信任区") {
			continue
		}
		value := strings.TrimSpace(dStr(a["value"]))
		if value == "" {
			continue
		}
		// Conflicting aliases are unknown, not evidence of a particular boundary.
		if zone != "" && !strings.EqualFold(zone, value) {
			return ""
		}
		zone = value
	}
	return zone
}
func automationTechnology(f Object) string {
	value := strings.TrimSpace(dStr(dObj(f["communication"])["technology"]))
	if value == "" {
		value = dStr(f["protocol"])
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' || r == '_' || r == '/' {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
	switch value {
	case "eth", "以太网":
		return "ethernet"
	case "bt", "蓝牙":
		return "bluetooth"
	case "jtagdap":
		return "jtag"
	case "usbdebug":
		return "dap"
	}
	return value
}
func automationUpdatePurpose(value string) bool {
	for _, keyword := range []string{"软件更新", "固件更新", "软件升级", "固件升级", "刷写"} {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if automationIn(token, "ota", "update", "firmware", "bootloader") {
			return true
		}
	}
	return false
}
func automationSortedValues(values []any) []any {
	result := append([]any{}, values...)
	sort.Slice(result, func(i, j int) bool { return dStr(result[i]) < dStr(result[j]) })
	return result
}
func automationSameSet(a, b []any) bool {
	return Digest(automationSortedValues(dUnique(a))) == Digest(automationSortedValues(dUnique(b)))
}

func automationAssets(ws, subject Object, property string) ([]any, []any) {
	preferred, manual := []any{}, []any{}
	for _, raw := range dList(ws["assets"]) {
		a := dObj(raw)
		if dStr(a["id"]) == "" || !dHas(dList(a["properties"]), property) || !automationSameSet(dList(a["node_ids"]), dList(subject["node_ids"])) {
			continue
		}
		source := assetSource(a)
		option := Object{"id": a["id"], "name": a["name"], "properties": dClone(a["properties"])}
		if len(source) == 0 {
			// A parent-node link cannot identify a particular internal store.
			// For child subjects require the exact architecture source instead
			// of associating another asset (for example logs) with key storage.
			if subject["kind"] != "child" {
				manual = append(manual, option)
			}
			continue
		}
		if source["status"] == "missing" || source["id"] != subject["id"] {
			continue
		}
		match := subject["kind"] == "flow" && source["kind"] == "flow"
		if subject["kind"] == "child" {
			match = source["kind"] == subject["child_kind"]
		}
		if subject["kind"] == "node" {
			want := "entity"
			if subject["node_kind"] == "process_step" {
				want = "software"
			}
			if subject["node_kind"] == "data_store" || subject["node_kind"] == "store" {
				want = "store"
			}
			match = source["kind"] == want
		}
		if match {
			preferred = append(preferred, option)
		}
	}
	options := append(preferred, manual...)
	sort.Slice(options, func(i, j int) bool { return dStr(dObj(options[i])["id"]) < dStr(dObj(options[j])["id"]) })
	// Multiple manually linked assets do not establish which business asset the
	// rule concerns; do not make an arbitrary choice by name or list order.
	if len(preferred) == 1 {
		return options, []any{dObj(preferred[0])["id"]}
	}
	if len(preferred) == 0 && len(manual) == 1 {
		return options, []any{dObj(manual[0])["id"]}
	}
	return options, []any{}
}
func automationCoverage(ws Object, candidateID, title string, rule automationRule, assetIDs []any) ([]any, []any) {
	existing, related := []any{}, []any{}
	for _, raw := range dList(ws["scenarios"]) {
		s := dObj(raw)
		if s["source_code"] == "AUTO:"+candidateID {
			existing = append(existing, s["id"])
			continue
		}
		if !automationSameSet(scenarioAssets(s), assetIDs) || !dHas(scenarioProperties(s), rule.property) {
			continue
		}
		if strings.TrimSpace(dStr(s["title"])) == title || strings.TrimSpace(dStr(s["threat"])) == rule.threat {
			existing = append(existing, s["id"])
		} else if strings.EqualFold(dStr(s["stride"]), rule.stride) {
			related = append(related, s["id"])
		}
	}
	return automationSortedValues(existing), automationSortedValues(related)
}
func automationExistingDamage(ws Object, code, description string, assetIDs []any, property string) string {
	ids := []any{}
	for _, raw := range dList(ws["damage_scenarios"]) {
		d := dObj(raw)
		if automationSameSet(dList(d["asset_ids"]), assetIDs) && dHas(dList(d["properties"]), property) && (d["source_code"] == code || d["description"] == description) {
			ids = append(ids, d["id"])
		}
	}
	if len(ids) == 0 {
		return ""
	}
	return dStr(automationSortedValues(ids)[0])
}
func automationUnknownImpact() Object {
	return Object{"safety": nil, "financial": nil, "operational": nil, "privacy": nil, "rationale": "损害后果和影响等级待评估。"}
}
func automationSemanticSnapshot(ws Object) Object {
	graph := dObj(ws["architecture"])
	nodes, flows := []any{}, []any{}
	for _, raw := range dList(graph["nodes"]) {
		n := Clone(dObj(raw))
		for _, key := range []string{"x", "y", "width", "height"} {
			delete(n, key)
		}
		for _, key := range []string{"children", "attributes"} {
			values := dList(n[key])
			sort.Slice(values, func(i, j int) bool { return Digest(values[i]) < Digest(values[j]) })
			n[key] = values
		}
		nodes = append(nodes, n)
	}
	for _, raw := range dList(graph["flows"]) {
		f := Clone(dObj(raw))
		for _, key := range []string{"sourceSide", "targetSide", "labelOffset", "wireless_layout", "route_layout", "emphasis", "show_label"} {
			delete(f, key)
		}
		if comm, present := f["communication"]; present {
			dObj(comm)["security_properties"] = automationSortedValues(dList(dObj(comm)["security_properties"]))
		}
		flows = append(flows, f)
	}
	sortRows := func(rows []any) []any {
		sort.Slice(rows, func(i, j int) bool { return dStr(dObj(rows[i])["id"]) < dStr(dObj(rows[j])["id"]) })
		return rows
	}
	result := Object{"id": ws["id"], "item": dClone(ws["item"]), "method": dClone(ws["method"]), "architecture": Object{"nodes": sortRows(nodes), "flows": sortRows(flows), "view_mode": graph["view_mode"], "asset_generation": graph["asset_generation"]}}
	for _, key := range []string{"assets", "damage_scenarios", "scenarios", "security_goals", "requirements"} {
		rows := []any{}
		for _, raw := range dList(ws[key]) {
			row := Clone(dObj(raw))
			for _, field := range []string{"created_at", "updated_at"} {
				delete(row, field)
			}
			for _, field := range []string{"node_ids", "asset_ids", "properties", "damage_ids", "scenario_ids", "goal_ids"} {
				if value, ok := row[field]; ok {
					row[field] = automationSortedValues(dList(value))
				}
			}
			if source, ok := row["architecture_source"]; ok {
				s := dObj(source)
				row["architecture_source"] = Object{"id": s["id"], "kind": s["kind"], "status": s["status"]}
			}
			rows = append(rows, row)
		}
		result[key] = sortRows(rows)
	}
	return result
}
