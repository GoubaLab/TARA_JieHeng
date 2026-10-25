> 公开源码版本：不附法规全文、历史资料、用户数据、二进制或前端构建产物。先按仓库根目录 README 构建；法规目录初始为空。

# 模块与领域设计

前端 React / Ant Design，后端 Go / net/http，持久化 SQLite（modernc.org/sqlite）。领域逻辑、资料读取、事务和静态报告在 Go 中执行；Excel 使用 Go 原生 Excelize 输出。前后端只通过版本化工作产品 API 通信。

| 模块 | 职责 |
| --- | --- |
| `server/cmd/jieheng/main.go` | 参数、路径、回环监听、优雅关闭 |
| `domain.go` / `domain_validation.go` | 工作产品、严格校验、完整性、摘要失效与业务操作 |
| `domain_method.go` | 方法默认值、四维最高值/加权平均/系数求和、统一初始/残余风险算法 |
| `canonical.go` | 数值保真的 JSON 及兼容旧摘要的规范编码 |
| `repository.go` | SQLite 事务、修订、幂等回执、变更日志与冻结基线 |
| `catalog.go` / `automation.go` | 当前为空目录、撤下引用占位与选用禁用；自有资料扩展点 |
| `regulations.go` | 版本、条款、来源摘要、父子引用关系和原件访问 |
| `http.go` / `http_resources.go` | 资源接口、条件请求、媒体类型、同源边界和问题响应 |
| `report.go` / `report_method.go` | 17 类工作产品、方法及元数据的静态投影 |
| `report_excel.go` | Go 原生 Excel 生成及 ZIP/工作簿结构验证 |
| `client/src/Workbench.jsx` | 导航、总览、清单、待办与项目上下文 |
| `editors.jsx` / `AssessmentSettings.jsx` | 记录编辑、评分算法与判定依据设置 |
| `method-config.js` | 前端默认显示和风险预览；最终计算以 Go 为准 |
| `diagram.js` / `architecture-view.js` | 画布交互、实体表示、静态 SVG 导出 |
| `communication.js` | 通信技术、OS 与图形标识预置 |
| `regulations.jsx` | 版本目录、条款选择和来源查看 |

后端表包含工作区、日志、请求回执、基线和创建回执。每次写入在同一事务内检查成功回执、项目修订、对象存在性、引用来源与业务规则，再原子保存结果。传给 prepare 回调的 current 是事务内工作区副本，撤下引用校验不额外读库，也不产生检查与写入竞态。

工作区含 Item、Architecture、Asset、Scenario、Decision、Requirement、Evidence、Method、Baseline 及法规引用。通用领域与稳定接口键分离，汽车是一个领域上下文。实体拥有 OS、属性和内部软件/存储；数据流连接不同实体，通信元数据独立于几何位置。SFOP 四维与可行性为现有评分骨架，显示标签与算法可配置，新增任意维度需要领域迁移。

读取不改写旧数据。可选 OS/通信/方法默认值在摘要中归一，不因补默认字段改变旧决策；真实语义变化会使受影响处置需要复核。位置、尺寸、端点侧向、标签偏移和项目法规勾选不参与评估摘要。基线保存完整工作产品快照，不被后续修改覆盖；报告可投影当前版本或冻结版本。

旧处理/外部节点兼容为实体；旧独立存储必须明确选择归属，再在事务中映射流与资产。自环、缺失归属或非法引用拒绝整次保存。旧参考键仅允许同一记录同一字段保留或删除，不向新对象传播，也不返回撤下原文。业务记录正文与基线不自动删除，运行数据库不打包。

当前法规目录独立保留、按 manifest 校验来源与条款，收录范围限定到指定版本。图形动画仅表达方向，静态 SVG/HTML/Excel 均无流动动画；文字和完整明细避免仅靠颜色或画布卡片展示。

接口细节见 [API_CONVENTIONS.md](API_CONVENTIONS.md)，版本契约见 [server/openapi.json](server/openapi.json)，扩展步骤见 [EXTENDING.md](EXTENDING.md)。本地工作台尚无认证、角色、租户隔离、动态插件或远程协同；不能直接作为互联网多租户服务暴露。
