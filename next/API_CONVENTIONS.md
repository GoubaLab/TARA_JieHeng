# 界衡 API 开发约定

规范版本：v1。机器可读契约为 `server/openapi.json`（OpenAPI 3.1.1），运行时通过 `/api/v1/openapi.json` 取得；本地阅读入口为 `/api/v1/docs`。运行服务不依赖外部文档站点。

## 资源和方法

所有业务地址使用 `/api/v1` 前缀，资源用复数名词和小写连字符。GET 只读；POST 创建记录；PUT 替换一个明确资源。资产、场景、需求以路径 ID 标识，客户端不通过任意动作字符串选择服务端逻辑。请求只包含该资源的数据。

- `/workspaces`：项目创建与分页查询。
- `/workspaces/{id}/item`、`/architecture`、`/assessment-method`：相关项、架构和评估方法。
- `/workspaces/{id}/assets`、`/scenarios`、`/requirements`：工作产品集合；更新使用对应 `/{record_id}`。
- `/workspaces/{id}/scenarios/{scenario_id}/attack-feasibility`：单条攻击路径的独立可行性评级。
- `/workspaces/{id}/scenarios/{scenario_id}/treatment`：风险处置决策。
- `/workspaces/{id}/evidence`、`/baselines`：追加验证记录、冻结基线；已存在的记录不可覆盖。
- `/workspaces/{id}/reference-imports`：自有资料导入扩展入口；当前无可用参考资料，POST 返回 422。
- `/workspaces/{id}/regulation-references`：项目适用法规引用。
- `/workspaces/{id}/review`、`/events`、`/reports`：完整性检查、变更记录和导出。
- `/regulations`、`/catalog`、`/references/{key}`：只读的法规版本、参考资料目录和引用详情。

具体路径、参数、请求和响应字段以 OpenAPI 为准。JSON 字段以 `snake_case` 为主；既有工作产品中的 `entityType`、`sourceSide`、`targetSide`、`labelOffset` 保留以避免破坏架构数据。接口版本与工作产品格式版本分别管理。

## 写入一致性

写入发送 `Content-Type: application/json`、`X-Workspace-Client: jieheng` 和 `Idempotency-Key`。后者为 8–100 个可见 ASCII 字符，在一次操作及其网络重试间保持不变；新的操作使用新键。项目内写入还必须携带 `If-Match`，值为读取项目时获得的 ETag，例如 `"8"`。

成功重放优先于修订检查：已经成功的请求因网络中断重试时返回原结果，不再次提交。相同幂等键对应不同动作或内容返回 409。首次请求的修订落后返回 412，缺少修订条件返回 428。创建项目使用独立的幂等回执。

项目内容、修订、变更日志、请求回执和新建基线在同一数据库事务中提交。任何校验失败均回滚。评审基线保留当时的快照；重新验证以新记录追加。规范化 JSON 保持内容摘要兼容；数值表示或默认字段补全不会单独导致已有决策与证据失效。

示例：

```http
PUT /api/v1/workspaces/ws_example/regulation-references
Content-Type: application/json
X-Workspace-Client: jieheng
If-Match: "8"
Idempotency-Key: b320291b-39d1-44df-800b-c26b8a445731

{"references":["reg:gb-44496-2024:4.1.1"]}
```

新增记录成功返回 201 及资源 Location，替换成功返回 200；工作区写入结果包含 `workspace_id`、`revision`、`entity_id`。读取完整项目和写入结果带 ETag。集合分页采用 `offset` / `limit`，上限 100；响应包含总数及分页值。日志列表使用 `entries`，项目、知识和条款列表使用 `items`；法规版本目录使用 `documents` 并报告 `total_clauses`。

## 校验和错误

请求拒绝未知字段、重复 JSON 键、非法类型、非有限数值、无效引用和超过 2 MB 的实际请求体。整数等级不能由字符串或布尔值冒充。实体关系、风险矩阵、来源、需求和证据约束在服务端执行。客户端检查用于交互提示，不能替代服务端校验。

业务接口错误采用 RFC 9457 的 `application/problem+json`，包含 `type`、`title`、`status`、`detail`、`instance` 和 `request_id`。相同请求标识同时写入 `X-Request-ID` 响应头。内部异常不向客户端返回堆栈或数据库细节。

| 状态码 | 含义 |
| --- | --- |
| 400 | JSON、参数格式或必要的请求标识错误 |
| 403 | 本地应用标识或来源边界不匹配 |
| 404 | 资源或引用不存在 |
| 405 | 资源不支持该方法 |
| 409 | 同一幂等键对应不同操作内容 |
| 412 | 项目修订已变化 |
| 413 | 请求体超过限制 |
| 415 | 写入不是 JSON 媒体类型 |
| 422 | 字段或工作产品规则校验失败 |
| 428 | 缺少 If-Match |
| 500 | 内部错误；根据 request_id 排查 |

## 本地运行边界

服务只监听回环地址，验证 Host 及写入 Origin，不开放跨域写入。`X-Workspace-Client` 是本地应用边界标识，不是身份认证；本机模式不提供身份认证。企业模式启用 HTTPS 反向代理入口、会话登录、账户角色与项目成员检查；不能把该标识当作企业认证。审批电子签名尚未提供。

报告和 PDF 以实际媒体类型返回；法规原件使用 inline 并支持物理页码定位；文件范围与缓存条件遵循 HTTP 的 206、304、416 响应，OpenAPI 单独声明。JSON/HTML/Excel 保留引用版本和来源核对边界；引用本身不构成标准符合性结论。静态画布导出继续在浏览器完成。

依据：[OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html)、[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457)、[HTTP 条件请求](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match)。

## 可配置领域与方法

`item.domain` 可选通用、汽车、机器人、物联网、充电或互联网领域。`architecture.nodes[].operating_system` 为实体操作系统；`architecture.flows[].communication` 将传输方式、技术、范围、可选用途和安全属性分开保存。范围不由协议名称自动推断。

`assessment-method` 为每项目配置，支持最高维度/加权平均/系数求和、矩阵/乘积阈值算法，评分名称和依据可编辑。语义变更使原决策需要复核；纯默认字段补全保持旧摘要兼容。详细字段、上限和公式见 OpenAPI 及 [二次开发指南](EXTENDING.md)。

目录当前为空。旧参考返回 `withdrawn:true,citable:false` 的空占位，不含原文；校验只允许同一项目记录原字段保留或删除旧键，禁止给新记录或其他字段引用。此检查与写入在同一事务进行。独立法规 `reg:` 条款仍可按实际存在的版本与条款引用。

系数求和仅与 product 配对，`impact_coefficients` 默认 `{safety:1,financial:1,operational:0.1,privacy:0.1}`，总分不归一化，风险分值不加 1。普通方法的 `product_thresholds` 仍为 1–15 的递增整数；系数法可用 0–4800 的递增小数。`residual_impact` 依方法限制：普通方法整数0–3，系数法允许小数且最多为3×系数和。

### 连线布局（适用于所有通信方式）

`Flow.sourceSide` / `targetSide` 可为 `top/right/bottom/left` 或 `null`（自动择短）。只调整所属实体的连接边，不改变 `source` / `target`、流向或安全属性。

`Flow.route_layout` 为可选对象，`waypoint` 使用 `LabelOffset` 的有限坐标范围（±10000），表示画布中的手工途经点；编辑器吸附至 10 单位网格，重新进行直角避障布线，合并共线段。空对象清除手工途经点，保留连接边；将两侧同时设为 `null` 可恢复自动择边。连线调整和标签位置均不参与评估摘要。架构保存沿用修订前提与幂等规则。

`Flow.wireless_layout` 已弃用但继续接受并保留，兼容旧客户端读写。独立无线卡片及 `source_offset` / `target_offset` 不再绘制；仅在新的布局字段缺省时兼容读取旧途经点和连接边。新编辑器统一使用实体间连线，并在线路标签上显示协议及无线图标。界面与静态 SVG 使用相同路线，导出没有编辑控制点和动画。

### 保存架构自动同步资产

`PUT /api/v1/workspaces/{workspace_id}/architecture` 在同一个事务内保存图和同步资产，响应新增可选 `asset_sync`，包含 `created`、`updated`、`missing`、`total`。同一幂等键重放返回原摘要；修订冲突或图校验失败均不产生部分资产。

实体生成“功能与服务”资产，内部软件生成软件资产，内部存储生成存储数据资产，数据流生成传输数据资产。信任边界、标签、图形控制点、操作系统名称本身不额外生成资产；操作系统资料仍是实体上下文。实体与软件建议完整性、可用性；存储和传输数据建议机密性、完整性、可用性。数据流明确选定安全属性时优先使用用户值。这些是可编辑的项目起始值，不代表完成风险评估。

服务器按项目与图内稳定标识匹配生成资产；重命名、移动子对象或调整连线不会重复生成。只读 `AssetRecord.architecture_source` 记录来源类别、对象 ID、状态、上次建议值及手工覆盖字段，不接受在资产写入请求中伪造。通过资产接口修改名称、说明、关联或安全属性后，后续架构同步保留相应字段。已有风险场景依赖的属性不会被自动删除。

来源被移除时保留资产和已有风险，标记 `missing` 并清理失效实体关联，评审提示待核对；来源恢复后沿用原资产 ID。手工资产继续沿用原有删除实体关联检查。来源追踪元数据不参与风险决策摘要，纯图形移动不会使决策失效。存量项目在下次显式保存架构时同步，不在应用启动或只读加载时改写。

## 项目编号规则

`GET /workspaces/{id}/numbering` 返回有效规则，`PUT` 完整替换规则并遵守 `If-Match` 与 `Idempotency-Key`。请求为 `prefixes`（10 种类型）、`separator`（`-` / `_` / `.` / `/`）、`padding`（2–6）。前缀支持 Unicode 文字和数字，可包含下划线、短横线，最长 12 字符，各类型前缀不能重复（不区分大小写）。默认类型依次为 entity=EN、boundary=TB、software=SW、store=DS、flow=DF、asset=AS、scenario=RS、path=AP、requirement=SR、evidence=EV。

`Workspace.numbering_state` 为只读类型→内部 ID→序号登记；客户端不可指定序号。攻击路径 ID 只在场景内唯一，其登记键为 `UTF8字节长度(scenario_id) + ":" + scenario_id + path_id`。读旧项目不写库；第一次非冻结操作先固定旧记录序号，再为新增记录顺序分配，删除保留登记不复用。只修改格式不会改变业务 ID、风险计算摘要或引用。画布、清单、HTML 与 Excel 显示编号一致；JSON 保留内部 ID 和编号配置/登记以支持复现。冻结基线始终按冻结快照规则导出。

## 展开 DFD

`Architecture.view_mode` 缺省为 `entities`，既有实体及内部组成规则不变。`dfd` 模式可使用独立 `process_step` 与 `data_store` 节点，允许与实体互联；信任边界仍不能作为连接端点。旧 `process`、`external`、`store` 的迁移规则不变。处理过程生成软件类资产，独立存储生成存储数据资产，均保持稳定来源 ID 与用户编辑。

`Flow.channel_kind` 可为 `data`（缺省）或 `energy`；电能流独立绘制，不生成传输数据资产。`emphasis` 为 `normal` / `critical` / `internal`，仅用于图中色彩强调。`show_label`（缺省 true）控制标签可见性；隐藏仍保留名称与报告明细。`Architecture.show_numbers` 控制画布编号前缀显示。显示设置不改变风险摘要，数据流/电能流类别属于语义变更。


## 完整 TARA 工作簿导入

`POST /workspaces/{id}/tara-import-previews` 接收 `{filename, content_base64, asset_links?}`，只核对，不增加修订。`POST /workspaces/{id}/tara-imports` 使用相同内容，并携带 `If-Match`、`Idempotency-Key` 与 `X-Workspace-Client`；成功返回 201、ETag、Location 和 `import_summary`。失败原子回滚，相同请求幂等重放。导入仅允许空白工作产品项目，可先建立 DFD 并设置 `architecture.asset_generation="catalog"`。

当前适配版本 `tara-chinese-v1` 支持约定的中文表结构和已核对的 SFOP 原始取值分档、五因子攻击潜力矩阵方法；不是任意 Excel 的自动识别器。限制为原文件 1.4 MB、解压 30 MB、40 个表页；不执行公式或宏。未知表头、无效引用、未知方法和计算不一致返回 422。用户数据仅保存到所选本地数据库，事件记录只存来源摘要和计数。

`GET /workspaces/{id}/tara-imports/{resource_id}` 返回来源表页、图像、合并范围、公式与缓存文本；`.../source` 返回字节一致的原始 XLSX。工作区普通读取不包含原文件字节。项目报告追加全部来源表页，完整原始排版与嵌入图仍可从原文件取得。

`damage-scenarios`、`security-goals` 提供 POST 创建与 `/{resource_id}` GET/PUT。完整模型见 OpenAPI。场景支持多个资产、安全属性、损害引用；需求支持多个目标与威胁。业务编号 `source_code` 与内部 ID 独立。修改关联损害后，风险重新计算；攻击路径的 `potential` 五因子存在时按项目阈值计算可行性。

`coefficient_bands` 使用 `impact_scale`、`impact_coefficients`、`impact_thresholds`、`severe_impact_floor`；仅允许风险矩阵。`attack_potential_thresholds` 以三个递增下限把攻击可行性从 4 降至 1。其他项目原有 coefficient_sum 总分直接参与风险计算的行为保持兼容。

导入时方法待确认、负责人和签署空项保持空，残余评估为 `residual_status="target"`，不得视为通过验证。改成 `validated` 前必须存在全部关联需求的有效通过记录；评审完整性仍独立检查。

目标关联处置保存服务端 `goal_hash`，调用者不在输入中提交此字段。目标修改要求处置复核。扩展 TARA 的 `EvidenceRecord.requirement_hash` 同时绑定关联目标与评估上下文；普通旧需求保持原摘要兼容。后续复测失败或依据改变时，有效 `residual_status` 回到 target，评审不通过。


## 企业平台与辅助分析接口

完整字段以 OpenAPI 为准。企业模式使用 `jieheng_session` Cookie（Secure/HttpOnly/SameSite=Strict，8 小时）；本机模式 `/auth/session` 返回本地管理员。所有写操作仍要求 `X-Workspace-Client: jieheng`，企业模式核对 HTTPS Host/Origin 和当前成员权限。

- `/model-providers`：GET/POST 服务；PUT `/{resource_id}` 条件替换。POST 以 Idempotency-Key 重放，PUT 以 If-Match 防覆盖。读取永不返回凭据；api_key 仅写入。`/export` 和 `/model-provider-imports` 为无凭据配置交换；导入停用。
- `/model-providers/{resource_id}/model-discoveries`、`/connection-tests`：主动查询/测试（后者只发合成测试内容）。这些显式操作可能消耗供应商用量，不作为可无限重放的资源创建。
- `/prompt-templates`：内置和组织模板；组织模板新增/条件替换，历史版本由服务端保留，任务固定模板版本。
- `/knowledge-documents`：组织/项目文本资料，GET 支持 workspace_id，POST/PUT 分别用幂等键/版本；`/knowledge-searches` 以 POST 提交查询，读取检索结果，可选调用向量服务。
- `/workspaces/{workspace_id}/ai-tasks`：GET 最新 100 个任务；POST 带 If-Match 与 Idempotency-Key，返回 202 和 Location，轮询 GET `/{resource_id}`。结果记录状态、步骤、来源、模型、Token、草案和校验结果，不含内部项目快照或凭据。
- 任务 `/{resource_id}/cancellations` 可重复取消；`/retries` 带版本/新幂等键创建新任务，原任务保留；`/adoptions` 带版本/幂等键及 `{owner,reviewed:true}`，原子采用并返回还原点。任务不修改验证、签署或冻结基线。
- `/workspaces/{workspace_id}/quality`：确定性缺口与覆盖检查；`/checkpoints` GET/POST，创建需 If-Match 与 Idempotency-Key；`/restorations` POST 同样需版本与幂等键，提交 checkpoint_id/reason，返回新修订。
- `/auth/session` GET、`/auth/sessions` POST 登录、`/auth/logout` POST 注销；`/enterprise/users` 管理账号，PUT 需 If-Match；项目 `/members/{resource_id}` PUT 设置成员角色或 none。账户建立以唯一账号名约束，登录、账户建立、成员设置、主动模型测试和检索不提供 Idempotency-Key 重放语义。
- `/platform-events` 返回最近 200 条管理记录，仅管理员可见。项目历史继续使用原有分页接口。

任务状态：queued → running → ready / needs_revision / failed；可取消为 cancelled，重启遗留为 interrupted，成功采用为 adopted。知识索引成功为 completed。来源变化或旧修订拒绝采用，412 要刷新重新分析，409 表示状态或幂等冲突。队列满为 429；格式错误 422；企业未登录 401、权限不足 403、非项目成员 404。服务不会因刷新页面自动创建或重试模型任务。

自动分析计划与采用接口见 [AUTOMATION.md](AUTOMATION.md)。公开知识包通过 `POST /knowledge-imports` 导入为停用待复核版本；修改导入资料并启用时，`PUT /knowledge-documents/{id}` 须提交 `reviewed:true`，同源旧启用版在同一事务内停用。导入保留来源、版本、内容指纹、CVE 和证据链接，重导不覆盖人工编辑。

## 项目基本信息

`PUT /api/v1/workspaces/{id}/profile` 更新项目名称与说明，请求体为 `{title, description}`。名称必填，最长 200 字符；说明可为空，最长 4,000 字符。接口要求项目写权限、`If-Match` 和 `Idempotency-Key`，变更写入修订与审计。该操作不修改工作产品、评分依据或冻结基线。

## 分步评级与功能引用

`item.functions` 保存功能编号、名称与具体内容，编号不可重复。旧客户端省略时保留现有目录；已被资产或损害场景引用的功能不能移除。`function_refs` 中的新增编号必须在相关项功能目录或已解析的来源范围定义中存在。

`PUT /workspaces/{id}/scenarios/{scenario_id}/attack-feasibility` 接收 `path_id`、`method` 和非空 `rationale`。`attack_potential` 方法要求全部五项 `potential` 档位；`attack_vector` 方法要求 `attack_vector`，两类参数不可混用。服务器计算可行性，不接受手填结果。附录 G 示例档位在 `StandardAttackPotential` 契约中列明，AP 分档保留项目配置。

未评级的攻击路径允许保存为草稿，但任何路径缺少评级、步骤或依据时，场景风险保持 `null`，不得确认处置。修改路径步骤、攻击者、入口或名称会清除原评级，包括导入路径的旧评级。既有导入工作簿及自定义 `AttackPotential` 数据保留，不自动转换评级结果。

损害场景及安全目标新增 `damage` / `goal` 编号类型，默认前缀 `DM` / `CG`。旧编号配置省略这两项时使用默认值，已有序号保持稳定。
