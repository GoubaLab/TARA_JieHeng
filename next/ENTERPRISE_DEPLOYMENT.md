# 企业本地与内网部署

本版本支持单企业、单实例的本机部署和内网团队工作台。Go 服务始终监听回环地址；企业模式通过同一主机上的 HTTPS 反向代理提供访问。SQLite 保存项目、任务、知识和身份；没有公网多租户、分布式任务队列、自动高可用或 SSO。

## 构建与两种模式

在仓库根目录执行：

```sh
./next/tools/build-go.sh
npm ci --prefix next/client
npm run build --prefix next/client
```

生成 `next/bin/jieheng` 与 `next/bin/jieheng-maintenance`。离线运行不需要 npm 或外网；构建依赖需由企业提前准备或使用内部镜像。标准正文及私有知识不随公开代码分发。

个人本机模式使用 `./start.sh` 或 `./next/bin/jieheng --root . --database next/var/assurance.sqlite3`。此模式没有账号认证，不要将它反向代理到内网或公网。

企业模式首次初始化：使用企业管理流程创建仅部署账号可读的 UTF-8 初始口令文件（权限 0600，12–72 字节），执行：

```sh
./next/bin/jieheng --root . --database next/var/enterprise.sqlite3 \
  --bootstrap-admin administrator --bootstrap-password-file /secure/initial-password.txt
```

初始化完成退出；妥善移除初始口令文件。已有账户时禁止再次初始化，不会覆盖账户。然后运行：

```sh
./next/bin/jieheng --root . --database next/var/enterprise.sqlite3 --port 8790 \
  --enterprise --public-origin https://tara.example.internal
```

`example.internal` 为示例域名，请替换成企业入口。配置必须与 HTTPS Host 和 Origin 一致。浏览器必须使用受信任证书，不建议通过浏览器忽略证书错误。

同主机 Nginx 示例（证书由企业 PKI 提供；部署前执行 Nginx 自身配置检查）：

```nginx
server {
    listen 443 ssl;
    server_name tara.example.internal;
    ssl_certificate /etc/tara/tls/fullchain.pem;
    ssl_certificate_key /etc/tara/tls/private.key;
    client_max_body_size 2m;
    location / {
        proxy_pass http://127.0.0.1:8790;
        proxy_set_header Host $host;
        proxy_read_timeout 200s;
        proxy_send_timeout 200s;
    }
}
```

若使用非标准 HTTPS 端口，Host 应保留该端口并与 `--public-origin` 相符。不支持将代理放在另一台主机后直接连接此回环监听器；不要自行把监听地址改成公网。可由 systemd/企业进程管理器管理服务，设置独立低权限运行账号、工作目录、持久数据目录与 `UMask=0077`，用 SIGTERM 正常退出，保留退出日志。不要从配置中关闭 TLS 验证。

## 账户与项目权限

| 角色 | 范围 |
| --- | --- |
| 管理员 | 账户与角色、模型配置、组织提示词/知识、项目成员及所有项目 |
| 评审员 | 已加入项目的分析写入、验证、建立基线和还原 |
| 分析员 | 已加入项目的建模、分析、任务、知识及工作产品写入；不能建立基线或还原 |
| 只读 | 已加入项目的查看、检索与报告读取 |

项目权限取账户角色和项目成员角色中的较低权限。非成员看不到该项目；创建项目的分析人员自动加入自己创建的项目。角色或口令修改会撤销该账号会话；停用账号立即拒绝后续访问，最后一个启用管理员不能被停用或降级。

口令以 bcrypt 保存，会话令牌只保存哈希；Cookie 为 Secure、HttpOnly、SameSite=Strict，有效期 8 小时。同一账号名连续 5 次失败后限制 15 分钟。写入同时核对来源与应用标识，`X-Workspace-Client` 不是身份凭据。证据复核人与基线评审人使用已登录账号身份，平台保留操作记录；这不等同于具有法律效力的数字签名。

登录审计和业务审计保存在本地数据库，管理员可在模型服务的「管理记录」中查看平台操作，项目更改在各自历史中查看。文件系统管理员仍可修改数据库，本版本没有独立防篡改审计存储。

## 备份、核验与还原

每个数据库都有同名 `.key` 凭据主密钥。密钥丢失时无法恢复模型凭据；如已存在加密凭据，程序拒绝生成新钥匙掩盖丢失。项目库与主密钥属于敏感运营数据，不要进入 Git、公开截图或通用源码包。

运行中的数据库可用 SQLite 一致性快照备份到**尚不存在的新目录**：

```sh
./next/bin/jieheng-maintenance --action backup \
  --source next/var/enterprise.sqlite3 --destination /secure/backups/tara-20261028
./next/bin/jieheng-maintenance --action verify --source /secure/backups/tara-20261028
./next/bin/jieheng-maintenance --action restore \
  --source /secure/backups/tara-20261028 --destination /secure/restore/tara-20261028
```

父目录需预先建立。备份包含 `assurance.sqlite3`、`assurance.sqlite3.key` 和 SHA-256 清单，目标目录 0700、文件 0600；核验包含 SQLite integrity_check。校验哈希用于检测损坏，不是签名认证。法规文件、应用构建与反向代理配置需按企业流程另行备份；它们不在数据库快照中。

还原只写入新目录，不覆盖运行数据库；恢复后的会话清空，需重新登录。先用还原文件在独立端口核对项目、模型凭据、知识索引和报告，确认后停止原服务，再修改 `--database` 切换。保留旧库和密钥，便于回退。程序启动会把遗留 AI 任务标为中断，不自动重复调用模型。

项目内「还原点」便于撤销误操作；它随数据库一起丢失，因此不能代替以上独立备份。建议企业定义备份周期、保留期限、异机副本和定期恢复演练。

## 交付验收与边界

先完成源码测试和构建，再核对 HTTPS 登录、非成员隔离、只读拒写、角色变更、备份恢复、并发版本冲突、任务取消/重启，以及人工建模到静态报告的完整链条。模型接入验收另见 [辅助分析](AI_ASSISTANCE.md)。真实项目外发由企业配置和每次任务范围控制，应用不预置外部模型账号。

当前仍需由企业实施主机运维、监控告警、证书生命周期、出口策略和容量压测。SSO/MFA、审批流、合格电子签名、多实例、高可用和大规模向量服务不在当前版本支持范围内。
