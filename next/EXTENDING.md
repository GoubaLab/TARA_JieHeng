> 公开源码版本：不附法规全文、历史资料、用户数据、二进制或前端构建产物。先按仓库根目录 README 构建；法规目录初始为空。

# 二次开发指南

## 产品边界

界衡是通用威胁建模工作台，汽车领域以 ISO/SAE 21434 工作产品为流程参考。当前提供同一条可追溯链：范围 → 实体/资产 → 损害/威胁/攻击路径 → 风险与处置 → 需求/验证 → 冻结基线/报告。选择互联网、物联网、机器人、充电设施或通用产品，只改变领域上下文，不自动宣称覆盖该领域的标准，也不自动修改风险算法。

当前版本支持直接配置领域、实体类型、通信技术、OS、评分名称/依据、SFOP 权重/系数、风险矩阵及乘积阈值。增加流程步骤、任意数量评分维度或新的报告模板仍需修改源码；尚无运行时插件安装、可执行公式、动态流程设计器。以下接口和扩展点用于可复现的二次开发。

## 开发与约束

后端 Go 1.25+，前端 React / Ant Design。数据库为 SQLite，现有内容和冻结基线保留原始表示。运行 `./next/tools/build-go.sh`、`npm run build --prefix next/client`、`./start.sh`。完整资源契约见 `server/openapi.json`，运行时 `/api/v1/docs`；更新契约后运行 `python3 next/tools/build-api-docs.py` 生成离线文档。

接口原则：资源名复数、HTTP 方法表示操作、`/api/v1` 版本前缀；请求字段默认 snake_case；已有画布几何字段兼容保留。写入检查修订、幂等键和同源边界，不在前端代替后端校验。每次变更在事务中保存工作区、变更记录和回执。增加不兼容结构时用显式迁移或新主版本，禁止静默重写冻结基线。

## 一条完整调用链

1. `POST /api/v1/workspaces` 创建项目，返回工作区与 ETag；请求头含 `X-Workspace-Client: jieheng`、独立 `Idempotency-Key`。
2. `PUT /api/v1/workspaces/{id}/item` 设置目的、边界、环境、假设、负责人及 `domain`。项目内写入携带上次 ETag 作为 `If-Match`，每个新操作使用新的幂等键；网络重试复用原键和原内容。
3. `PUT .../architecture` 整体保存实体和数据流。实体持有 `operating_system`、属性及软件/存储子对象；数据流关联两个不同实体。不要在客户端臆造后端资源 ID 冲突处理。
4. `PUT .../assessment-method` 设置方法；`POST .../assets`、`POST .../scenarios` 建立分析。评分计算仅在后端，前端矩阵是预览。
5. `PUT .../scenarios/{scenario_id}/treatment` 记录处置和独立残余风险；`POST .../requirements`、`POST .../evidence` 建立验证追溯。
6. `GET .../review` 获取完整性与失效状态；`POST .../baselines` 建立冻结基线；`GET .../reports?format=html|json|xlsx` 导出当前版本，带 `baseline_id` 则导出冻结版本。

422 返回字段或业务校验问题；412 需要重新读取、合并用户意图再提交，不能静默覆盖；409 说明同一个幂等键被用于不同内容。所有错误包含请求标识。

## 通信、实体与操作系统

`Item.domain`：`general / automotive / robotics / iot / charging / internet`。旧数据省略时按汽车领域解释，读取不补写。

`Node.entityType`：`ecu / external / service / device / application / cloud / actor`。只有实体可带 OS：

```json
{"family":"linux","name":"Yocto Linux","version":"5.x","architecture":"ARM64","notes":"产品维护的发行版本"}
```

family 还支持 `unknown / bare_metal / rtos / qnx / windows / android / ios / macos / other`。`name` 可写 FreeRTOS、Zephyr 或自有 OS；类型表示家族，版本/架构分字段记录。裸机和未知不是同一状态。实体内部软件与数据存储不作为数据流端点。

```json
{"medium":"wireless","technology":"UWB","range":"local","purpose":"测距","security_properties":["integrity","authenticity"]}
```

上述对象位于 `Flow.communication`；原 `Flow.protocol` 留作上层协议或通道。medium 为 `unspecified / wired / wireless / logical`；range 为 `unspecified / near_field / local / wide_area`。技术是开放文本，不用新后端版本即可填写新协议。用途也是可选文本。不会根据技术名称推断其安全能力或自动勾选安全属性。Wi-Fi、蓝牙与 NFC 可选择不同范围；DoIP 和 Ethernet 可在技术与上层协议中分层记录。

前端预置及视觉标识集中在 `client/src/communication.js`；扩展技术建议在这里维护别名、显示名称和中性 SVG 标记。已有协议识别仅供展示，用户确认后才写入新字段。画布 `diagram.js` 可有方向动画；静态导出 `architecture-view.js` 不允许动画或脚本，且应保留文字含义，不能只依赖颜色。

## 评分方法

`domain_method.go` 是影响合并、默认值与风险分级的唯一业务实现。每个项目保存自己的 method，基线冻结当时的方法。

- SFOP 各维度 0–3；最高值法要求四项完整。加权平均法用整数权重 0–100，相对归一后向上取整；零权重维度可不评分，总权重必须大于零。
- `coefficient_sum` 模式直接计算 `Σ(各维等级×impact_coefficients)`，默认系数 S/F=1，O/P=0.1，不归一、不取整。此模式仅允许 product，风险分值=原始总分×可行性（不加1），四个阈值允许有限小数0–4800。相等阈值归入低一档。零系数维度可缺评分，总系数须大于零，比较按十进制语义计算。初始影响及风险分值分别出现在评估的 impact 与 score；残余总分允许小数，不能超过 3×系数和。
- 攻击可行性 1–4，越高越容易；风险 1–5。矩阵为单调的 4 × 4 整数矩阵。
- 普通最高值/加权平均模式的乘积法分值 `(impact + 1) * feasibility`，范围 1–16；四个严格递增整数阈值限定 1–15，分段为闭合上限，默认 `[3,6,9,12]`。
- 场景取各路径最高风险；残余风险输入独立，但使用相同分级规则。缺失必要评分或依据时保持待评估。
- 维度显示名称、4 个影响级、4 个可行性级、5 个风险级的名称与判定条件均可配置。显示名不会改变稳定键 `safety / financial / operational / privacy` 的接口含义。

如果需要新增算法，依次修改 `domain_validation.go`、`domain_method.go`、`report_method.go`、OpenAPI 和 `AssessmentSettings.jsx`；补边界、缺失值、初始/残余一致性及基线测试。禁止把输入公式直接交给 eval 或 shell。新增任意数量维度需要新的领域结构与迁移设计，当前 UI 不声称支持。

## 流程与报告扩展

增加步骤需要同步修改：

| 扩展点 | 文件 / 责任 |
| --- | --- |
| 工作产品、命令、校验和失效关系 | `server/internal/assurance/domain.go`、`domain_validation.go` |
| 事务与审计回执 | `repository.go`；业务变化使用同一 Change 事务 |
| API 资源及文档 | `http.go`、`http_resources.go`、`server/openapi.json` |
| 导航、待办与编辑 | `client/src/api.js`、`Workbench.jsx`、独立表单组件 |
| 静态报告投影 | `report.go`、`report_method.go`；JSON、HTML、Excel 保持事实一致 |
| Excel 样式和工作表 | `report_excel.go`（Excelize）；改变表数时同步结构检查 |

建议先定义新工作产品与依赖图，再实现后端规则，最后接页面。不要只加导航却让评审检查或报告漏项。报告来源应从工作区或冻结基线投影，禁止绕过基线去读当前活数据。导出不能包含动态流动效果；画布静态 SVG 与报告表格分别验证。

## 自有资料与法规

当前发行版不含历史非公开参考资料，参考目录为空。旧引用只返回 `withdrawn:true,citable:false` 的空占位；新引用禁止使用，同一记录原字段可原样保留或删除，不能挪用到新记录。旧项目与冻结基线不自动删改；业务内容由项目所有者管理，运行数据库不进入交付包。

如接入自有资料，在 `catalog.go` 增加独立数据适配器与稳定命名空间，记录来源、版本、授权范围和摘要；不要恢复已退休的数据目录。`automation.go` 当前返回空候选并明确拒绝选用操作；需要自有候选时，服务端重新解析条目并在事务中核对所选资源，避免客户端提交伪造正文。不得自动编造评分、处置或验证结论。

法规独立目录使用 `data/regulations/manifest.json`；新增版本必须独立 ID、唯一条款键、有效父子关系、SHA-256 及来源/覆盖说明。不覆盖旧版本，不把勾选条款当作符合性结论。当前 GB/T 44464 转载本仍标记 partial，详见目录说明。

## 验证与交付

运行 Go 测试/race/vet、前端测试、真实报表检查和离线打包门禁。打包采用路径白名单，并扫描撤下资料的文件、标识和正文指纹；指纹只存 hash。备份、运行数据库、项目数据、旧发布包均不交付。对于需要开放测试源码的开发交付，应保留原创测试，不分发退休夹具。企业模式已具备会话认证、组织角色和项目成员权限；回环工作台标识仍不是登录体系。多租户、SSO/MFA 与集群部署须另行实现，交付配置见 ENTERPRISE_DEPLOYMENT.md。
