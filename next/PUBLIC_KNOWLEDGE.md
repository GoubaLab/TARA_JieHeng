# 公开知识维护与复核

知识维护遵循可追溯流程：**公开事实 → 原创分析 → 待复核 → 启用 → 检索 → 项目分析**。公开来源、CVE 编号和向量相似度都是参考证据，不会单独证明某个产品受影响。项目仍需确认组件、版本、配置、入口、攻击前提和现有措施，再形成自己的损害场景与风险结论。

## 内置案例

内置知识包位于 `next/data/public-knowledge/seed.json`，包含 8 条原创分析：

| 案例 | 建模关注点 |
| --- | --- |
| Uconnect / CVE-2015-5611 | 车载信息娱乐域到控制总线的信任边界 |
| QNX / CVE-2021-22156 | OS 内运行库、分配尺寸和静态依赖 |
| VxWorks IPnet / CVE-2019-12255 | RTOS 网络栈与应用鉴权的不同入口 |
| Siemens PLC / CVE-2020-15782 | 工程维护通信与控制器内存保护 |
| Hikvision / CVE-2021-36260 | 设备 Web 管理入口与命令处理 |
| Log4j / CVE-2021-44228 | 日志输入、远程查找与出站连接 |
| Heartbleed / CVE-2014-0160 | TLS 内存泄露和泄露后的密钥处置 |
| libwebp / CVE-2023-4863 | 图片解析与跨产品的组件依赖 |

每条均区分事实、适用条件、不适用情形、原创防护建议、验证启发和证据链接。内置案例用于跨行业分析参考，不构成漏洞扫描结果或适用于所有项目的风险清单。

## 内网和离线导入

企业内网不需要开放互联网出口即可使用知识导入。管理员可在获准联网的环境取得官方来源，保留来源和校验信息，再把 JSON 知识包通过企业批准的介质转入内网。

平台支持预览内置案例包和上传 JSON 知识包。导入后条目默认 `enabled=false`；审核人员核对并补齐内容，再通过 `reviewed:true` 的启用操作发布。启用新版本时，同一 `source_identity` 的旧版停用，已有引用和历史证据仍应保留。未经审核和未启用的知识不应参与检索或项目建议。

开发接入使用 `GET /knowledge-packs/public-cases` 获取内置包，使用 `/knowledge-imports` 导入知识包；实际服务前缀、权限和请求/回执结构以项目 OpenAPI 文档为准。公共包必须与企业私有项目、私有附件和内部知识分开管理。

## CISA KEV 收集工具

`next/tools/collect_public_knowledge.py` 只处理官方 KEV 格式，默认不联网，也不会导入数据库或修改项目。它支持：

- `--input`：读取离线 JSON；与 `--fetch-kev` 二选一。
- `--fetch-kev`：显式访问固定 CISA HTTPS 数据地址，拒绝重定向，不接受任意网址。
- `--output`：新的候选包文件。为保留每批结果，不覆盖已有输出。
- `--state`：本地每条 CVE 的游标文件。
- `--limit`：每批新建或变更候选的上限，默认 25，范围 1–50。

输入或响应不得超过 5 MB。工具校验目录数量、CVE ID、重复记录、字段长度、日期、URL、来源字段和已知枚举值；最终条目正文也会在转义后校验导入长度。输出还按字节预算分批，总包小于 1.9 MB，以适配 HTTP 导入上限；因此大文本批次可能未达到条数上限就进入下一批。新增的未知字段或不认识的枚举会要求适配器复核，失败时不会推进游标。空快照不会被解释为整个目录撤回。

在仓库根目录运行离线收集：

```sh
python3 next/tools/collect_public_knowledge.py \
  --input ./incoming/kev.json \
  --output ./out/kev-batch-001.json \
  --state ./var/kev-cursor.json \
  --limit 25
```

若回执中的 `pending` 或 `pending_reviews` 大于 0，继续使用同一输入和游标，指定新的输出文件名：

```sh
python3 next/tools/collect_public_knowledge.py \
  --input ./incoming/kev.json \
  --output ./out/kev-batch-002.json \
  --state ./var/kev-cursor.json \
  --limit 25
```

在线采集在企业批准的联网环境执行：

```sh
python3 next/tools/collect_public_knowledge.py \
  --fetch-kev \
  --output ./out/kev-online-001.json \
  --state ./var/kev-cursor.json \
  --limit 25
```

离线文件由使用者提供，工具标记为 `collection_mode=offline-input`，该模式不提供来源身份验证；联网模式则标记为固定官方 HTTPS 获取。首次接收离线文件时应核对传递渠道、源版本和摘要。

输出保留官方地址、`catalogVersion`、`dateReleased`、`captured_at`、规范化快照/记录摘要及 CC0 来源声明。每条候选明确标为“目录事实，尚未分析适用条件”，需要人员补齐产品版本、配置、前提、措施和验证。它不会生成攻击步骤或自动计算项目风险。

## 增量、变更和撤下

工具以 `cveID + 规范化内容 SHA-256` 去重。采集时间只保留在包级元数据，不参与条目正文；同一包版本和事实的重复采集保持相同条目内容。目录重新排序、字段顺序变化和重复 CWE 不会产生新候选；有实质内容变化则形成同一稳定 ID 的新版本素材。游标只推进已经输出或确认未变化的记录，超过批量上限的候选留待下一批。

候选文件先落盘，随后才更新游标；如果游标写入失败，重跑可能重复输出同一批，因此平台仍需依据稳定来源标识去重。保存输出包和游标时应作为一组备份。

旧条目在新快照中缺失时，输出顶层 `review_required`，保留旧游标和证据；不能仅凭一次缺失就自动认定撤回。每批最多输出 100 条复核通知，且通知部分不超过 60 KB；只有已经写入输出包的通知才推进通知游标，剩余数量保存在 `pending_reviews`。持续缺失的相同记录不会重复通知；记录重现后再次消失会重新通知。原始 CVE 游标和内容摘要始终保留。平台接受只有复核通知而没有新条目的包，将清单保存到导入回执审计，不会自动删除或停用知识。完全没有条目和通知的空包无需导入。审核人员需核对官方变化、确认是否撤下，并检查受影响的项目引用。

CVE Program 的撤回也要保留替代关系。例如官方将 `CVE-2023-5129` 标为 `CVE-2023-4863` 的重复项。它不能继续作为独立漏洞累加风险。[官方撤回记录](https://github.com/CVEProject/cvelistV5/blob/main/cves/2023/5xxx/CVE-2023-5129.json)

## 周期运行示例

以下示例适用于已有调度系统。将工作目录设为仓库根目录，每日执行：

```sh
batch_stamp=$(date -u +%Y%m%dT%H%M%SZ)
python3 next/tools/collect_public_knowledge.py \
  --input ./incoming/kev.json \
  --output "./out/kev-${batch_stamp}.json" \
  --state ./var/kev-cursor.json \
  --limit 25
```

由企业已有的数据传递流程更新 `incoming/kev.json`。定时任务只产生候选包；导入、审核和启用仍是独立操作。若使用 cron，把上述内容保存为本地任务脚本再调度，避免 cron 对 `%` 的特殊处理。同步失败、适配器版本变化或 `review_required` 非空应进入管理员的处理队列；没有新增候选不代表项目没有风险。

离线测试命令：

```sh
python3 -m unittest discover -s next/tools -p test_public_knowledge.py -v
```

测试使用虚构数据，覆盖重复、变更、批量上限、游标保留、缺失复核、坏输入不写游标、响应大小和重定向限制，不连接真实数据源。

## 来源许可边界

CISA 官方 [kev-data](https://github.com/cisagov/kev-data) 的 [LICENSE](https://github.com/cisagov/kev-data/blob/develop/LICENSE) 明确将 KEV 数据按 CC0 1.0 分发。许可不覆盖其第三方链接内容，不授权使用 CISA 标志，也不表示 CISA 认可本产品。KEV 的法规期限和取证分流字段有其适用对象，不能直接变成所有客户的项目要求。

CVE、CWE、CAPEC、ATT&CK 各有 MITRE 条款，复制或分发其内容时要保留规定的版权与许可；不能把这些源一律标为 CC0。[CVE 条款](https://www.cve.org/legal/termsofuse)、[CWE 条款](https://cwe.mitre.org/about/termsofuse.html)、[CAPEC 条款](https://capec.mitre.org/about/termsofuse.html)、[ATT&CK 条款](https://attack.mitre.org/resources/legal-and-branding/terms-of-use/)

内置 8 条案例包含原创分析、简要事实摘要和证据链接，不附厂商公告全文、补丁或附件。BlackBerry 与 Hikvision 相关案例以对应厂商 CNA 提交的官方 CVE 记录作为事实依据。链接内容的版权归各自权利人。当前收集器仅支持 CISA KEV，不采集或分发 NVD 内容。案例来源与许可范围见 [公开案例知识包](data/public-knowledge/README.md)。

后续接入其他来源前，应分别核实协议、许可、增量方式、撤回状态和版本范围表达；建立适配器与离线样例后再启用。CVE 数量和知识条目数量只是库存，实际质量来自适用条件、可追溯证据、复核记录以及在项目中的验证结果。
