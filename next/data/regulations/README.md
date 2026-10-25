# 本地法规资料接入

本公开源码版本不附带任何法规或标准正文、PDF、Formex ZIP、用户上传资料或预生成条款库。`manifest.json` 与 `manifest.sample.json` 均为可运行的空目录；空库不会生成示例法规或隐式联网下载。

## 目录契约

`manifest.json` 使用 `schema_version: 1`，`documents` 为数组。无资料时保持空数组。接入文档时，条目须包含独立 `id`、`code`、`title`、`edition`、`source_url`、`coverage` 和 `clause_count`。正文列表通过 `clauses_file` 引用目录内文件，并填写其 `clauses_sha256`；原件通过 `artifacts` 声明目录内路径、媒体类型与 SHA-256。

覆盖状态仅有 `metadata_only`、`partial`、`complete_for_edition`。仅元数据不可包含可引用正文；完整版本必须有正文和 `coverage_evidence`。不得把未取得、未核对或仅部分提取的材料声明为完整版本。

条款列表每项至少包含 `key`（`reg:{document_id}:{locator}`）、`document_id`、`number`、`title`、`body` 和 `parent`。父级必须属于同一版本且不存在循环。物理页码用 `page` / `page_end`，印刷页码用 `print_page`；多原件版本可使用 `source_file` 与 `source_pages` 明确定位。具体约束见 `server/internal/assurance/regulations.go` 与原创测试。

版本或修改单使用独立文档 ID。不要覆盖原始版本，也不要根据引用自动判断符合性。加载器校验所有声明文件的路径、摘要及条款关系；缺失或损坏的已声明文件会报错，不会静默变成空库。

## 可选离线构建器

保留的 `build_*.py` 是针对指定版式与来源摘要编写的解析代码，正文从调用者自行准备的本地文件读取，未内嵌法规全文。Python 仅用于可选资料处理，不是服务运行依赖；相关构建器使用 `pypdf` 或 `pdfplumber`，按脚本导入要求安装。

GB 44495、GB 44496 构建器接收命令行 PDF 路径；GB/T 44464 构建器读取指定本地 PDF；UN 构建器需要其脚本列明的本地原件与工作目录。不同来源或版本应先独立核对，再调整解析器、摘要和覆盖证据，不能为通过校验直接替换摘要。构建器生成待核对条款与清单片段，不会自动更新运行目录。

接入资料前自行确认使用及分发权限。`sources/` 和生成的正文 JSON 已在公开仓库的忽略规则中排除；不要将个人、客户资料或未获分发许可的全文提交到公开仓库。
