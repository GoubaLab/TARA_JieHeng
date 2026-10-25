> 公开源码版本：不附法规全文、历史资料、用户数据、二进制或前端构建产物。先按仓库根目录 README 构建；法规目录初始为空。

# Excel 导出与二次开发

界衡通过 Go 原生库 `github.com/xuri/excelize/v2` 生成标准 `.xlsx`。依赖固定为 **v2.11.0**，使用 BSD-3-Clause 许可证，要求 Go **1.25.0 或更高版本**。版本依据为 [Excelize 官方介绍](https://xuri.me/excelize/en/)及[该版本的 go.mod](https://github.com/qax-os/excelize/blob/v2.11.0/go.mod)；具体依赖与校验摘要保存在 `server/go.mod` 和 `server/go.sum`。

首次构建用 `go mod download` 获取公开依赖，然后正常构建后端。构建好的 Go 程序内置 Excel 能力，运行和导出时无需 Node、Python、Office 或 Codex 组件，也不联网获取组件。

```sh
cd next/server
go mod download
go build -o ../bin/jieheng ./cmd/jieheng
```

网页中的“导出 Excel”以及 `GET /api/v1/workspaces/{id}/reports?format=xlsx` 使用同一实现。指定 `baseline_id` 可导出冻结基线。原有 `JIEHENG_REPORT_NODE`、`JIEHENG_ARTIFACT_MODULES`、`JIEHENG_WORKBOOK_SCRIPT` 设置不再参与 Go 导出；`assurance/workbook.mjs` 只保留退役提示，不提供另一套渲染实现。

## 内容与排版

`server/internal/assurance/report.go` 的 `Tables` 是 HTML、JSON 工作产品与 Excel 共用的数据投影。Excel 沿用 **17 张工作表**及其顺序、列名和记录顺序，覆盖签批、更改记录、相关项、架构、资产、损害/威胁、方法、攻击路径、风险、处置、目标、声明、需求、标准引用等工作产品。修改导出不改项目数据，签批空白也不会自动填入人员。

`server/internal/assurance/report_excel.go` 负责 Excel 排版和文件验证：

- 第 2 行为表名，第 3 行为表说明；第 5 行为表头，第 6 行起为数据。
- 中文字体指定 Microsoft YaHei；查看设备缺少该字体时，由表格应用选择替代字体，文件不附带字体。
- 保留各表列宽、文字换行、顶部对齐、交替行底色和分隔线。
- 所有表固定冻结前 5 行，确保滚动时表头可见；超过 6 列的宽表同时冻结首列。
- 表说明在 A3 可直接查看，并附单元格批注。特别长的说明仍完整保留在 A3，批注会提示查看原单元格。
- 单行显示高度以 Excel 上限为界；内容不会因此截断，可展开单元格或在编辑栏查看完整文字。

## 文本与校验

所有内容（包括数字、时间、空值和以 `=`、`+`、`-`、`@` 开头的文字）均通过 [`SetCellStr`](https://xuri.me/excelize/en/cell.html) 写入并使用文本格式。不会为防注入额外插入可见的单引号；原始文字保持原样。导出的报告不包含公式、宏、外部工作簿连接或嵌入程序。

生成后验证 ZIP、工作簿入口与关系、准确的工作表名称和顺序、各表 XML、字符串单元格，并拒绝公式、宏二进制内容及外部关系。失败时返回错误，不提供部分文件。

Excel 单元格最多容纳 32,767 个 UTF-16 字符单元。超过限制时会明确指出工作表和单元格，请缩短对应内容或导出 HTML / JSON；系统不会静默截断。工作表行数、列数和列宽也会在生成时检查。

## 修改方式与验证

修改字段或新增工作产品时先调整 `Tables`；修改颜色、字体、冻结、列宽处理和批注时调整 `report_excel.go` 的 `reportExcelStyles` / `reportExcelSheet`。列宽由每张表的 `widths` 决定。不要改用公式写入任意用户文本。

```sh
cd next/server
go test ./internal/assurance -run 'TestExcel' -count=1
```

测试会实际生成并重新打开工作簿，核对 17 张表的数据、文本类型、公式样文字、中文样式、列宽、批注与冻结状态，以及损坏文件和不允许的活跃内容；不依赖私有工具或系统 Office 安装。
