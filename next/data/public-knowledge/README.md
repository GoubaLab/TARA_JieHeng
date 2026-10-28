> 公开源码版本：不附法规全文、历史资料、用户数据、二进制或前端构建产物。先按仓库根目录 README 构建；法规目录初始为空。

# 公开案例知识包

`seed.json` 是基于公开一手证据重新编写的 8 条分析案例，不含之前平台的私有知识库、项目、用户或附件。包标识为 `jieheng-public-cases`，版本为 `2026.10.28`，格式为 `jieheng.knowledge-pack/1`。

每条内容分为事实、适用条件、不适用情形、原创建模与防护建议、原创验证启发和证据链接。涉及汽车、QNX/RTOS、工业控制、摄像机、互联网服务和多媒体组件。这里的 CVE 是证据索引；CVE、弱点、攻击模式、损害场景和项目风险是不同对象。

## 使用方式与审核

- 导入后应处于待审核且未启用状态；只有经过审核、确认适用条件的条目才能用于项目检索与建议。
- 新版本按 `pack_id + entry.id` 识别同一条目；`version` 和内容变化表示修订，不应复制成新案例，也不应悄悄覆盖已批准的项目结论。
- `edition` 是本次原创分析版本，不是厂商固件版本或外部源版本。原始 CVE 更新时间、源文件提交、抓取时间和内容摘要应由后续采集记录单独保存。
- 版本范围、组件实际存在、配置、入口可达性和供应商回补必须逐项确认。未知不是“不受影响”，也不是“已确认受影响”。
- 包内防护与验证内容是工程建议，和来源明确报告的事实分开书写。它们不宣称自动满足 ISO/SAE 21434 或其他标准。
- CVSS、KEV 和 SSVC 是不同证据信号，不能直接替代项目的 SFOP、损害场景或风险计算。

## 来源与许可边界

本包分发的是原创中文分析、公开事实的简要概括和证据链接；没有复制厂商公告、论文、网页截图或 CVE 原始记录全文。原创贡献遵循本项目适用的许可；链接指向的内容仍归各自权利人，项目许可不替外部来源重新授权。

### CVE Program

逐条核对使用 [CVE Program 官方 CVE List V5](https://github.com/CVEProject/cvelistV5)。CVE 内容由相应 CNA 提供，关联来源包括 BlackBerry、Siemens、Hikvision、Apache、Chrome 等。CVE 数据版权归 The MITRE Corporation 等适用权利人所有；使用规则见 [CVE Terms of Use](https://www.cve.org/legal/termsofuse)。该条款允许复制、派生及分发，并要求在复制 CVE 内容时保留 MITRE 的版权标识和许可。

本包的稳定标识不是 CVE Program 的官方分类或认证。以后如直接分发 CVE 原始记录或原文片段，必须随包收录对应 MITRE 版权标识和完整许可，不能只保留网页链接。

### CISA KEV

CISA 官方 [kev-data 仓库](https://github.com/cisagov/kev-data) 的 [LICENSE](https://github.com/cisagov/kev-data/blob/develop/LICENSE) 明确采用 **CC0 1.0 Universal**；[CISA 官网对应许可](https://www.cisa.gov/sites/default/files/licenses/kev/license.txt)是其同源许可地址。官方说明同时指出，KEV 中第三方链接受其各自条款约束，CC0 不授权使用 CISA 标志，也不表示 CISA 背书。

本种子包没有复制 KEV 全库。后续采集器可将官方 KEV 的必要元数据转为候选记录，保留 CVE ID、来源 URL、目录版本与采集时间，按来源做差异审核。KEV 的加入、修改或移除都应留痕；缺少 KEV 记录不等于没有被利用。KEV 期限不应自动变成所有客户的合规期限。

### 厂商通报与其他知识源

- 厂商通报全文、补丁、附件和需要账号的资料不随本包分发，仅保留链接。BlackBerry 和 Hikvision 部分页面本次不能直接读取；对应事实由官方 CNA 提交的 CVE 记录核对，不能把链接存在当作已读取页面。
- [Siemens SSA-434534](https://cert-portal.siemens.com/productcert/html/ssa-434534.html)具有厂商自己的使用条款，不能套用 CISA 的 CC0。
- [CWE](https://cwe.mitre.org/about/termsofuse.html)、[CAPEC](https://capec.mitre.org/about/termsofuse.html)、[ATT&CK](https://attack.mitre.org/resources/legal-and-branding/terms-of-use/)均有各自 MITRE 许可与署名要求。本包未打包这些库的正文或分类数据。
- NVD 的当前 API 使用条款页面本次未成功取到正文，因此本包不使用 NVD 正文，不将其所有内容笼统标为公有领域。后续接入前应核验 [API 条款](https://nvd.nist.gov/developers/terms-of-use)及上游内容权利。

## 证据更新

本次核对日期：2026-10-28。仅定点读取 8 条案例 CVE 及 1 条重复/撤回样例，没有全量下载漏洞库。

建议每次更新先保存来源版本与哈希、生成变更候选，审核后形成新知识版本。上游撤回、修正版本范围或改变利用状态时，应暂停受影响推荐并提示重新审核，保留已导出报告所引用的历史版本。已撤回的 `CVE-2023-5129` 被官方记录标为 `CVE-2023-4863` 的重复项，说明必须保留撤回及替代关系，不能机械按 CVE 数量积累条目。
