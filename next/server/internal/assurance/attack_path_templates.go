package assurance

import (
	"net/http"
	"sort"
	"strings"
)

const attackPathTemplateCatalogVersion = "jieheng.attack-path-templates/1"

// Templates define modeling entry points, prerequisites and control checks.
// Reachability, control status and risk ratings are evaluated in the project.
type attackPathTemplate struct {
	id, name, category, summary, entry, actor, objective      string
	prerequisites, steps, controls, properties, tags, signals []string
}

func builtinAttackPathTemplates() []attackPathTemplate {
	return []attackPathTemplate{
		{"AP-NETWORK-SERVICE", "网络服务输入处理缺陷", "网络与服务", "从可达网络服务进入，评估输入处理与执行边界。", "设备或服务的网络监听入口", "可访问目标网络服务的非授权主体", "影响服务进程或受保护状态",
			[]string{"确认监听服务、组件版本、配置及实际可达路径", "输入处理或权限边界是否存在缺陷，需要独立证据"},
			[]string{"从已确认的网络路径到达目标服务", "向相关解析或业务处理环节提交非预期输入", "若输入处理或隔离失效，进入超出允许范围的处理分支", "检查受保护状态是否因此被读取、改变或中断"},
			[]string{"服务最小化和访问控制", "输入边界校验与进程隔离", "组件补丁及故障恢复"}, []string{"integrity", "availability", "confidentiality"}, []string{"网络服务", "协议解析", "Linux", "RTOS", "QNX"}, []string{"network", "os"}},
		{"AP-API-AUTHORIZATION", "管理 API 越权访问", "身份与权限", "检查已登录主体能否越过对象、角色或租户的授权边界。", "管理接口或业务 API", "具备低权限账户或接口访问条件的主体", "读取或修改不属于其授权范围的对象",
			[]string{"确认目标确实提供相关 API 和受保护对象", "确认账户角色、对象归属和允许的业务操作"},
			[]string{"通过正常入口建立受限权限会话", "定位操作对象与服务端授权检查点", "尝试请求超出当前主体授权范围的对象或操作", "若服务端未正确约束主体与对象关系，触及受保护数据或管理功能"},
			[]string{"服务端逐对象授权", "角色和租户隔离", "敏感操作审计"}, []string{"authorization", "confidentiality", "integrity"}, []string{"API", "云平台", "对象授权", "管理接口"}, []string{"api", "cloud"}},
		{"AP-WIRELESS-PEER", "无线对端伪装与消息注入", "通信与接口", "沿无线接入、配对及消息接收链检查身份与完整性。", "Wi-Fi、蓝牙、NFC、UWB 或其他无线接收入口", "满足无线覆盖和接入前提的非授权主体", "使接收端接受非授权对端或消息",
			[]string{"确认无线范围、配对方式、连接状态和攻击者接近条件", "链路或应用层保护是否不足，需要核验配置与实现"},
			[]string{"到达无线接收范围并满足必要的接入条件", "参与或影响对端发现、配对或会话建立", "向业务接收端提供声称来自可信对端的消息", "若身份或消息完整性校验失效，业务处理接受非授权输入"},
			[]string{"配对与对端身份确认", "链路及应用消息完整性", "异常接入限制和密钥管理"}, []string{"authenticity", "integrity", "authorization"}, []string{"Wi-Fi", "蓝牙", "NFC", "UWB", "无线"}, []string{"wireless"}},
		{"AP-MESSAGE-REPLAY", "合法消息重放或乱序", "通信与接口", "检查合法消息离开原时间和状态上下文后是否仍被接受。", "已确认的数据流接收入口", "能够观察或重传相关消息的主体", "重复或改变业务状态转换",
			[]string{"能够取得适用的合法消息，并有发送或转发条件", "业务需要区分消息新鲜性，正常重试和幂等语义已明确"},
			[]string{"取得目标会话或状态中的合法消息", "在另一时间、序列或重连状态下重新提交消息", "若接收端不能关联序列、时间或会话上下文，消息再次被接受", "检查是否产生重复操作或非预期状态转换"},
			[]string{"序列和会话绑定", "时间窗口或等效新鲜性检查", "重试幂等与重启状态恢复"}, []string{"freshness", "integrity", "authenticity"}, []string{"重放", "时序", "数据流"}, []string{"flow", "freshness"}},
		{"AP-DEBUG-WRITE", "调试或维修接口未授权写入", "物理与维护", "沿调试入口、生命周期状态和写入能力分析权限边界。", "JTAG、DAP、SWD、UART 等维护入口", "能够接触设备或维护代理入口的主体", "改变固件、配置或运行状态",
			[]string{"接口实际用于调试或维护，并具备读写能力", "确认物理接近条件、设备生命周期状态与维护权限"},
			[]string{"接触物理接口或已确认的维护代理入口", "建立调试或维修会话并触达受保护操作", "若生命周期或维护授权检查不足，执行非授权写入", "检查程序、配置或运行状态是否偏离允许值"},
			[]string{"生产调试锁定和生命周期控制", "维修认证与最小权限", "敏感写入审计和恢复"}, []string{"integrity", "authorization", "confidentiality"}, []string{"JTAG", "DAP", "SWD", "UART", "维修"}, []string{"debug"}},
		{"AP-UPDATE-CONTENT", "软件更新包替换或篡改", "软件与更新", "从更新输入到安装执行，检查来源、内容及目标绑定。", "软件或固件更新入口", "能够影响更新包、分发路径或更新请求的主体", "使目标安装非授权软件内容",
			[]string{"确认实际更新通道和安装入口", "确认内容校验、签名信任及目标设备绑定的实现"},
			[]string{"在已确认的分发或接收环节影响更新内容", "向更新入口提供被替换或篡改的包", "若来源、完整性或目标绑定校验失效，内容进入安装阶段", "检查目标是否安装或执行了非授权软件"},
			[]string{"更新内容真实性和完整性", "目标设备与组件绑定", "安装失败恢复及密钥撤销"}, []string{"integrity", "authenticity", "authorization"}, []string{"OTA", "软件更新", "固件", "签名"}, []string{"update"}},
		{"AP-UPDATE-ROLLBACK", "旧版本回滚后恢复已知缺陷", "软件与更新", "检查有效但过期的更新内容是否突破版本授权策略。", "更新、刷写或恢复入口", "能提交历史有效软件版本的主体", "使目标运行不再获准的旧版本",
			[]string{"存在可用的旧版本内容和实际安装路径", "项目已定义允许版本及回滚、维修、恢复策略"},
			[]string{"取得目标可识别的历史软件版本", "通过更新或恢复入口提交旧版本", "若版本授权与防回滚状态检查不足，旧版本被接受", "检查已修复缺陷或已撤销功能是否重新暴露"},
			[]string{"版本授权和防回滚状态", "维修恢复的受控例外", "版本状态可信存储与审计"}, []string{"integrity", "freshness", "authorization"}, []string{"回滚", "版本", "恢复", "更新"}, []string{"update"}},
		{"AP-STORAGE-READ", "存储、备份或维修路径泄露数据", "数据与存储", "沿实际读出路径分析受保护数据及其副本的暴露。", "存储读接口、备份或维修读出路径", "能够接触存储或数据副本的非授权主体", "取得不应被其读取的敏感数据",
			[]string{"确认存储内容需要保密及实际读出路径", "确认运行、备份、维修和退役阶段的访问条件"},
			[]string{"接近存储接口或取得数据副本", "经运行、备份或维护路径请求读取", "若授权、密钥隔离或数据保护不足，取得敏感内容", "检查是否能够恢复或进一步使用泄露的数据"},
			[]string{"数据分类和最小访问权限", "静态数据保护及密钥分离", "备份和退役清除"}, []string{"confidentiality", "privacy"}, []string{"存储", "备份", "密钥", "隐私"}, []string{"store"}},
		{"AP-CREDENTIAL-REUSE", "凭据获取后冒用管理身份", "身份与权限", "将凭据暴露与后续会话、授权边界分开建模。", "凭据存储、分发及管理登录入口", "取得仍有效凭据的非授权主体", "以合法身份执行未获授权的管理操作",
			[]string{"存在凭据获取来源，并已单独核实取得条件", "凭据对目标仍有效，目标入口实际可达"},
			[]string{"从已确认的暴露路径取得凭据或令牌", "在目标入口建立或接管会话", "若额外身份确认、范围或有效期限制不足，凭据被接受", "使用取得的权限访问受保护数据或管理功能"},
			[]string{"凭据隔离和最小授权范围", "有效期、轮换与撤销", "敏感操作再认证及异常会话检测"}, []string{"authenticity", "authorization", "confidentiality"}, []string{"凭据", "令牌", "身份冒用", "账户"}, []string{"store", "cloud"}},
		{"AP-RESOURCE-EXHAUSTION", "异常负载耗尽处理资源", "可用性", "沿接收、排队和共享资源路径分析服务中断或实时性丧失。", "网络服务或数据流入口", "能够持续触达目标入口并提供负载的主体", "降低服务可用性或使关键处理超时",
			[]string{"确认负载可达性、处理开销和共享资源关系", "明确容量、响应时限、可接受降级和恢复条件"},
			[]string{"持续触达已确认的服务或消息入口", "使接收、解析、排队或会话资源进入异常负载状态", "若配额、隔离或超时控制不足，关键资源无法释放或分配", "检查关键服务是否中断、超时或不能恢复"},
			[]string{"入口配额与速率限制", "队列、内存和执行时间隔离", "背压、超时及故障恢复"}, []string{"availability"}, []string{"资源耗尽", "拒绝服务", "队列", "实时性"}, []string{"network", "flow"}},
		{"AP-SUPPLY-CHAIN", "依赖或构建制品被非授权替换", "软件与更新", "沿依赖引入、构建和交付链检查制品来源与授权。", "依赖仓库、构建输入或制品交付入口", "能够影响依赖、构建或制品分发环节的主体", "使受保护产品引入非授权组件或行为",
			[]string{"确认产品实际使用相关依赖和构建交付链", "确认可被影响的环节、权限以及制品验证措施"},
			[]string{"影响已确认的依赖来源或构建输入", "使替换内容进入构建或制品分发流程", "若版本固定、来源核验或构建权限控制不足，制品被接受", "检查非授权内容是否随发布进入目标产品"},
			[]string{"依赖锁定和来源验证", "构建权限隔离及制品签署", "SBOM、制品溯源和发布审核"}, []string{"integrity", "authenticity"}, []string{"供应链", "依赖", "SBOM", "构建"}, []string{"software", "os"}},
		{"AP-REMOVABLE-MEDIA", "可移动介质进入受保护处理流程", "物理与维护", "检查外接介质从识别、读取到数据或程序处理的边界。", "USB 或其他可移动介质接口", "能够提供或替换介质的主体", "让目标处理非授权内容或暴露数据",
			[]string{"确认接口确实接受介质内容，不仅用于供电", "确认介质读取、导入、维护或执行功能及访问条件"},
			[]string{"将介质接入目标允许的物理入口", "触发已确认的读取、导入或维护流程", "若内容校验、权限隔离或介质策略不足，非预期内容进入处理", "检查目标状态或受保护数据是否受影响"},
			[]string{"介质和设备类别限制", "导入内容校验及隔离处理", "执行授权、导出限制和审计"}, []string{"integrity", "confidentiality", "authorization"}, []string{"USB", "介质", "导入", "维护"}, []string{"usb"}},
		{"AP-BUS-INJECTION", "共享总线接入后伪造业务报文", "通信与接口", "沿总线接入和接收端处理检查报文来源与业务授权。", "CAN、CAN FD、FlexRay 或其他共享控制总线", "具备总线接入或受控桥接条件的主体", "使接收端执行非授权业务动作",
			[]string{"确认总线接入、桥接或已受控节点的前置条件", "明确报文用途、接收端状态检查与消息鉴别措施"},
			[]string{"通过已确认的物理或逻辑路径取得总线发送条件", "在适用的业务状态下提供声称来自合法节点的报文", "若来源、权限或状态一致性校验不足，接收端处理报文", "检查是否触发非授权动作或状态转换"},
			[]string{"跨域网关和报文白名单", "业务状态及发送权限校验", "消息真实性、完整性和新鲜性"}, []string{"integrity", "authenticity", "authorization", "freshness"}, []string{"CAN", "CAN FD", "FlexRay", "总线"}, []string{"bus"}},
		{"AP-LOCAL-PRIVILEGE", "低权限软件跨越执行权限边界", "软件与更新", "从已取得的受限执行位置分析本地权限提升路径。", "本地软件进程、IPC 或权限代理接口", "已取得有限本地执行能力的主体", "获得更高权限或访问受保护资源",
			[]string{"有限本地执行能力的取得必须另有证据或前置路径", "确认进程权限、IPC、共享资源及实际缺陷条件"},
			[]string{"从已确认的低权限执行位置访问本地接口", "触达权限代理、共享资源或特权组件的处理边界", "若授权或隔离失效，操作在更高权限上下文中执行", "检查受保护资源是否因此可被读取或修改"},
			[]string{"最小权限和进程隔离", "IPC 主体校验及特权操作约束", "权限配置和组件补丁"}, []string{"authorization", "integrity", "confidentiality"}, []string{"操作系统", "权限提升", "IPC", "隔离"}, []string{"os", "software"}},
		{"AP-DATA-EXPORT", "合法导出能力越过数据使用范围", "数据与存储", "检查数据从查询、导出到共享链路的授权与范围限制。", "查询、日志、备份或导出接口", "具有部分数据访问或导出能力的主体", "获得或传出超出授权范围的数据",
			[]string{"确认具体数据分类、导出功能和用户授权范围", "确认目标接收位置及业务允许的共享方式"},
			[]string{"通过已获准的查询或导出入口访问数据", "请求超出允许对象、字段、数量或使用目的的数据", "若授权、过滤或脱敏不足，输出包含受保护内容", "检查内容是否通过下载、日志或共享路径离开允许范围"},
			[]string{"对象和字段级授权", "导出范围限制与脱敏", "共享限制和数据访问审计"}, []string{"confidentiality", "privacy", "authorization"}, []string{"数据导出", "隐私", "日志", "备份"}, []string{"store", "cloud", "api"}},
		{"AP-CROSS-DOMAIN-INPUT", "跨信任域输入影响业务决策", "网络与服务", "把外部输入、边界转换和接收端决策串成可核验的路径。", "外部实体到内部处理的数据入口", "能够控制另一信任域中相关输入的主体", "使接收端形成不正确的业务决策",
			[]string{"确认两端确属不同信任主体及输入影响的业务功能", "确认实际可达路径和现有边界处理措施"},
			[]string{"从已确认的外部或不同信任域取得输入控制条件", "通过边界接口传入非预期但可被处理的数据", "若来源、语义或状态校验不足，数据参与受保护业务决策", "检查控制结果或业务状态是否偏离已定义的约束"},
			[]string{"边界身份及输入语义校验", "业务状态和合理性检查", "跨域最小权限及拒绝审计"}, []string{"integrity", "authenticity", "authorization"}, []string{"信任边界", "输入校验", "跨域"}, []string{"cross-domain"}},
	}
}

func (s *HTTPServer) attackPathTemplates(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	items, e := attackPathTemplateItems(nil, "", "")
	if e != nil {
		return e
	}
	return writeJSON(w, r, 200, Object{"version": attackPathTemplateCatalogVersion, "items": items})
}
func (s *HTTPServer) workspaceAttackPathTemplates(w http.ResponseWriter, r *http.Request) error {
	q, e := httpQuery(r, "asset_id", "property")
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
	items, e := attackPathTemplateItems(ws, q.Get("asset_id"), q.Get("property"))
	if e != nil {
		return e
	}
	setETag(w, ws)
	return writeJSON(w, r, 200, Object{"version": attackPathTemplateCatalogVersion, "base_revision": ws["revision"], "items": items})
}

func attackPathTemplateItems(ws Object, assetID, property string) ([]any, error) {
	if property != "" && !automationIn(property, dProperties...) {
		return nil, dError("安全属性无效")
	}
	var asset Object
	if assetID != "" {
		var e error
		asset, e = dFind(dList(ws["assets"]), assetID)
		if e != nil {
			return nil, ErrNotFound
		}
	}
	facts := attackPathTemplateFacts(ws, asset)
	propertyNames := map[string]string{"confidentiality": "机密性", "integrity": "完整性", "availability": "可用性", "authenticity": "真实性", "authorization": "授权", "non_repudiation": "不可抵赖性", "freshness": "新鲜性", "privacy": "隐私性"}
	items := []any{}
	for _, t := range builtinAttackPathTemplates() {
		score := 0
		reasons := []any{}
		for _, signal := range t.signals {
			if reason := facts[signal]; reason != "" {
				score += 25
				reasons = append(reasons, reason)
			}
		}
		if property != "" && automationIn(property, t.properties...) {
			score += 10
			reasons = append(reasons, "与当前关注的"+propertyNames[property]+"属性相关")
		} else if property == "" && asset != nil {
			for _, p := range t.properties {
				if dHas(asset["properties"], p) {
					score += 5
					reasons = append(reasons, "资产需要"+propertyNames[p]+"保护")
					break
				}
			}
		}
		if score > 95 {
			score = 95
		}
		items = append(items, Object{"id": t.id, "version": "1", "name": t.name, "category": t.category, "summary": t.summary, "entry_point": t.entry, "actor": t.actor, "objective": t.objective, "prerequisites": automationStrings(t.prerequisites), "steps": automationStrings(t.steps), "controls_to_check": automationStrings(t.controls), "properties": automationStrings(t.properties), "tags": automationStrings(t.tags), "source_code": "TPL:" + t.id + "@1", "match": Object{"score": score, "reasons": reasons}})
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := dObj(items[i]), dObj(items[j])
		sa, sb := dInt(dObj(a["match"])["score"]), dInt(dObj(b["match"])["score"])
		if sa != sb {
			return sa > sb
		}
		return dStr(a["id"]) < dStr(b["id"])
	})
	return items, nil
}

// Only the selected asset's exact, active architecture source supplies structural
// facts. A parent node alone cannot identify a software process, store or flow.
func attackPathTemplateFacts(ws, asset Object) map[string]string {
	facts := map[string]string{}
	source := assetSource(asset)
	if len(source) == 0 || source["status"] != "active" {
		return facts
	}
	graph := dObj(ws["architecture"])
	flows := append([]any{}, dList(graph["flows"])...)
	sort.SliceStable(flows, func(i, j int) bool { return dStr(dObj(flows[i])["id"]) < dStr(dObj(flows[j])["id"]) })
	nodes := map[string]Object{}
	for _, raw := range dList(graph["nodes"]) {
		n := dObj(raw)
		if n["kind"] != "boundary" {
			nodes[dStr(n["id"])] = n
		}
	}
	addFlow := func(f Object) {
		a, b := nodes[dStr(f["source"])], nodes[dStr(f["target"])]
		if f["channel_kind"] == "energy" || a == nil || b == nil {
			return
		}
		comm := dObj(f["communication"])
		tech := automationTechnology(f)
		facts["flow"] = "关联数据流「" + dStr(f["name"]) + "」"
		if comm["medium"] == "wireless" || automationIn(tech, "wifi", "bluetooth", "ble", "nfc", "uwb", "zigbee", "lora", "cellular", "4g", "5g") {
			facts["wireless"] = "数据流使用无线通信"
		}
		if automationIn(tech, "ethernet", "doip", "tcp", "udp", "http", "https", "mqtt", "wifi", "cellular", "4g", "5g") {
			facts["network"] = "数据流使用网络协议「" + dStr(f["protocol"]) + "」"
		}
		if automationIn(tech, "http", "https") {
			facts["api"] = "数据流使用 HTTP(S)，可核对 API 访问路径"
		}
		if automationIn(tech, "jtag", "dap", "swd", "uart") {
			facts["debug"] = "关联调试或串行接口，需确认维护用途"
		}
		if tech == "usb" {
			facts["usb"] = "数据流使用 USB，需确认设备类别"
		}
		if automationIn(tech, "can", "canfd", "flexray", "lin") {
			facts["bus"] = "数据流使用共享总线协议「" + dStr(f["protocol"]) + "」"
		}
		if automationUpdatePurpose(dStr(comm["purpose"])) {
			facts["update"] = "通信用途涉及软件或固件更新"
		}
		if dHas(comm["security_properties"], "freshness") {
			facts["freshness"] = "数据流需要新鲜性保护"
		}
		za, zb := automationZone(a), automationZone(b)
		if automationExternal(a) != automationExternal(b) || (za != "" && zb != "" && !strings.EqualFold(za, zb)) {
			facts["cross-domain"] = "数据流连接了外部主体或不同信任域"
		}
	}
	kind, id := dStr(source["kind"]), dStr(source["id"])
	if kind == "flow" {
		for _, raw := range flows {
			f := dObj(raw)
			if f["id"] == id && automationSameSet(dList(asset["node_ids"]), []any{f["source"], f["target"]}) {
				addFlow(f)
				break
			}
		}
		return facts
	}
	var target, parent Object
	for _, raw := range dList(graph["nodes"]) {
		n := dObj(raw)
		want := "entity"
		if n["kind"] == "process_step" {
			want = "software"
		}
		if n["kind"] == "data_store" {
			want = "store"
		}
		if n["id"] == id && kind == want && diagramEndpoint(graph, n) && automationSameSet(dList(asset["node_ids"]), []any{n["id"]}) {
			target, parent = n, n
			break
		}
		for _, childRaw := range dList(n["children"]) {
			c := dObj(childRaw)
			if c["id"] == id && c["kind"] == kind && automationSameSet(dList(asset["node_ids"]), []any{n["id"]}) {
				target, parent = c, n
				break
			}
		}
		if target != nil {
			break
		}
	}
	if target == nil {
		return facts
	}
	if kind == "store" {
		facts["store"] = "关联存储「" + dStr(target["name"]) + "」"
		return facts
	}
	if kind == "software" {
		facts["software"] = "关联软件或处理过程「" + dStr(target["name"]) + "」"
	}
	os := dObj(parent["operating_system"])
	if family := dStr(os["family"]); family != "" && family != "unknown" && family != "bare_metal" {
		facts["os"] = "关联实体的操作系统为「" + family + "」"
	}
	if kind == "entity" && (target["entityType"] == "cloud" || target["entityType"] == "service") {
		facts["cloud"] = "关联云服务或服务实体"
	}
	// Parent connections do not prove that an internal process exposes that service.
	if kind == "entity" {
		for _, raw := range flows {
			f := dObj(raw)
			if f["source"] == id || f["target"] == id {
				addFlow(f)
			}
		}
	}
	return facts
}
