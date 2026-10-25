# 界衡 · 威胁建模平台

Go / SQLite 与 React / Ant Design 实现的本地威胁建模工作台。源码、原创测试、接口契约和扩展说明公开，支持相关项、实体与数据流、资产、场景、风险评估、处置、需求、验证和冻结报告。

## 从源码运行

准备 Go（版本见 `next/server/go.mod`）、Node.js 22.12 或更新的兼容版本，以及 npm。在仓库根目录执行：

```sh
./next/tools/build-go.sh
npm ci --prefix next/client
npm run build --prefix next/client
./start.sh
```

打开 http://127.0.0.1:8790 。运行数据默认保存在 `next/var/assurance.sqlite3`；可通过 `--database` 和 `--port` 指定独立数据文件及端口。服务只监听回环地址。

公开仓库不附带法规/标准全文、历史参考资料、用户项目、运行数据库、本机二进制或编译后的前端。法规目录初始为空，建模和测试仍可运行。按 [法规接入说明](next/data/regulations/README.md) 在本地接入有权使用的资料；引用不代表符合性结论，应用不会自动联网获取正文。

## 开发与验证

```sh
(cd next/server && go test ./... && go vet ./...)
node --test next/tests/*.mjs next/tests/*.cjs
python3 next/tools/test_public_release.py
```

前端测试中的布线检查读取构建产物，因此先执行前端构建。Go 测试使用临时数据库与原创合成法规夹具，不依赖法规全文。

- [功能和开发说明](next/README.md)
- [领域与模块](next/ARCHITECTURE.md)
- [版本化 API](next/API_CONVENTIONS.md)
- [扩展流程、算法和报告](next/EXTENDING.md)
- [Go 迁移说明](next/GO_MIGRATION.md)

仓库目前未选定开源许可证。公开可见不表示已授予额外的使用、修改或再分发许可；第三方依赖遵循各自许可证。
