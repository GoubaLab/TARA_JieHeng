> 公开源码版本：不附法规全文、历史资料、用户数据、二进制或前端构建产物。先按仓库根目录 README 构建；法规目录初始为空。

# 界衡 API 开发约定

规范版本：v1。机器可读契约为 `server/openapi.json`（OpenAPI 3.1.1），运行时通过 `/api/v1/openapi.json` 取得；本地阅读入口为 `/api/v1/docs`。运行服务不依赖外部文档站点。

## 资源和方法

所有业务地址使用 `/api/v1` 前缀，资源用复数名词和小写连字符。GET 只读；POST 创建记录；PUT 替换一个明确资源。资产、场景、需求以路径 ID 标识，客户端不通过任意动作字符串选择服务端逻辑。请求只包含该资源的数据。

- `/workspaces`：项目创建与分页查询。
- `/workspaces/{id}/item`、`/architecture`、`/assessment-method`：相关项、架构和评估方法。
- `/workspaces/{id}/assets`、`/scenarios`、`/requirements`：工作产品集合；更新使用对应 `/{record_id}`。
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

项目内容、修订、变更日志、请求回执和新建基线在同一数据库事务中提交。任何校验失败均回滚。评审基线保留当时的快照；重新验证以新记录追加。Go 读写保留旧 JSON 内容摘要的语义，已有决策和证据不会因为语言更换而失效。

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

服务只监听回环地址，验证 Host 及写入 Origin，不开放跨域写入。`X-Workspace-Client` 是本地应用边界标识，不是身份认证；本版本仍是单机工作台。多人身份、角色权限和审批签名属于后续独立功能，不能把这层标识当作企业认证。

报告和 PDF 以实际媒体类型返回；法规原件使用 inline 并支持物理页码定位；文件范围与缓存条件遵循 HTTP 的 206、304、416 响应，OpenAPI 单独声明。JSON/HTML/Excel 保留引用版本和来源核对边界；引用本身不构成标准符合性结论。静态画布导出继续在浏览器完成。

依据：[OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html)、[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457)、[HTTP 条件请求](https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match)。

## 可配置领域与方法

`item.domain` 可选通用、汽车、机器人、物联网、充电或互联网领域。`architecture.nodes[].operating_system` 为实体操作系统；`architecture.flows[].communication` 将传输方式、技术、范围、可选用途和安全属性分开保存。范围不由协议名称自动推断。

`assessment-method` 为每项目配置，支持最高维度/加权平均/系数求和、矩阵/乘积阈值算法，评分名称和依据可编辑。语义变更使原决策需要复核；纯默认字段补全保持旧摘要兼容。详细字段、上限和公式见 OpenAPI 及 [二次开发指南](EXTENDING.md)。

目录当前为空。旧参考返回 `withdrawn:true,citable:false` 的空占位，不含原文；校验只允许同一项目记录原字段保留或删除旧键，禁止给新记录或其他字段引用。此检查与写入在同一事务进行。独立法规 `reg:` 条款仍可按实际存在的版本与条款引用。

系数求和仅与 product 配对，`impact_coefficients` 默认 `{safety:1,financial:1,operational:0.1,privacy:0.1}`，总分不归一化，风险分值不加 1。普通方法的 `product_thresholds` 仍为 1–15 的递增整数；系数法可用 0–4800 的递增小数。`residual_impact` 依方法限制：普通方法整数0–3，系数法允许小数且最多为3×系数和。
