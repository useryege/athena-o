# BSC 索引器与 Sports 退役操作手册

本手册用于 2026-09-16 已批准的精确删除。代码提交不代表现场完成；每个环境的实际状态见[验收记录](../testing/module-removal-cleanup-acceptance.md)。不导出业务数据，不建立历史归档，保留清单、日志与验证证据。

## 版本与边界

必须使用同时包含后端、八权限契约、前端与部署清理的一致版本。保留两个 Worm 程序/数据库、独立权限、钱包/签名/再认证/交易/Cash Out/通知、共享账户和通知账本、Gateway、其他业务及基础设施。仅删除确认属于两索引器和 Sports 的对象；Worm 的体育分类不属于删除范围。

构建并记录 `athena`、独立 Trader Sync、`athena-account-state-migrate` 与 `tools/retire-sports-notifications` 的 commit、文件摘要、镜像 ID。先完成计划 T6 的真实本地验证。生产配置使用现场原有密钥，禁止为了更新调用重置 secrets 的 Make 包装；禁止全 project `down --remove-orphans`、`run-reset` 或 Docker prune。现有 `prod-deploy-remote`/`deploy` 是带整栈重建副作用的入口，不用于本任务。

## 每环境清单

在 `.superpowers/module-removal/retirement/` 保存独立 JSON，记录 UTC 时间、SSH 主机指纹/机器 ID、实际目录、版本和 project。`docker ps -a --no-trunc` 后 inspect 精确容器，保存 ID、标签、挂载、重启策略、镜像 ID；卷消费者必须包括停止容器。读取 systemd、cron、进程启动时间/可执行文件以及业务端口。历史 IP 和默认卷仅为线索。

数据库只读核对：连接实际配置指向的服务，读取 `pg_control_system()`、`pg_postmaster_start_time()`、`pg_database` 中真实库名/OID/owner，以及 `pg_stat_activity` 的用户、application、client 和 PID；同时列出全部保留库。连接有界、statement timeout 5 秒。保存覆盖库名对应的 DSN 身份，不能仅查默认名称。

两台历史索引器位于 `47.245.183.140:/opt/athena-bsc-transaction-indexer` 与 `47.254.154.128:/opt/athena-bsc-swap-indexer`。必须独立保存同机 Gateway 的 unit、PID、监听和健康/可用性基线；主机及 Gateway 不退役。清单发现对象已不存在时记录“本轮核验不存在”，不能记为本轮执行了删除。

## 维护顺序与失败停止点

1. 单环境只有一个操作者；每项修改前重新读取机器、容器 ID/标签、数据库实例/OID、卷消费者，与清单一致才继续。身份变化或消费者不明则暂停该对象，继续独立工作。
2. 记录业务暂停前实际运行的消费者。Compose 用 `io.athena.account-state.consumer: "true"` 明确标记账户库使用者，再对照实际进程与同库连接。当前 server、notification、trader-sync 已标记；现场 Solana 等额外使用者必须同样标记或作为外部消费者精确停止。仅从 env_file 继承 DSN 不说明使用该库。`--profile '*' config --format json` 用于发现 profile 服务；恢复只显式启动原先运行项。
3. 撤下旧 API/UI 权限写入版本，按精确 owner/ID 停止全部 Sports 和索引器生产者，移除其自动拉起来源。单次正常退出预算 30 秒。保留原 owner 状态和证据，不按端口杀进程。
4. 全部实际账户库使用者退出后，使用新 schema authority 执行 `up`、`verify`，成功后启动一致新版本。000004 为追加迁移，删除三权限与约束收紧在一个事务。禁止旧代码继续写入，迁移失败不跳过 verify。
5. Sports 生产者全部退出后，设置该环境实际 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，先运行维护工具默认只读，再运行 `retire-sports-notifications --apply --timeout=5m`。工具数据库级互斥、每批100行、锁等待2秒/语句5秒；输出数据库/OID/owner/实例启动时间和计数。仅六个精确来源的 pending 转 cancelled；sending 等真实结果，retryable 回 pending 后下一批取消，终态和尝试账本保留。不接管 poller，不发送 Telegram，不重发 unknown。
6. 工具退出 0 且 `status=completed`、`counts_verified=true`、pending=0、sending=0 才通过通知门槛。默认只读成功不代表清理完成。超时/中断后记录非零结果，通过 Notification 原恢复流程处理停止 sender，再核对身份重入；不得人工改 sending 伪造完成。
7. 核验新登录/账户/Wallet/Notification/Worm及同机Gateway，旧API不再提供业务。真实验证仅安全读取，资产副作用以隔离测试证据为准。任何保留能力失败，停止后续删除并记录已经发生的变化。

## 精确数据删除

共享 PostgreSQL 只删除核对的四业务库。每次从非目标维护库连接，重新核对实例身份、真实名称/OID/owner、`pg_stat_activity` 无未知连接；设置 lock timeout 5 秒、statement timeout 60 秒。用 psql 标识符变量引用实际已核对库名：

```sql
SET lock_timeout = '5s';
SET statement_timeout = '60s';
-- actual_database 由已复核清单赋值，不能使用近似名称或默认值猜测。
DROP DATABASE :"actual_database";
```

不得 `WITH (FORCE)` 或强踢未知连接。每删一库立即重查 catalog，保留共享卷与 Worm/Wallet/账户等所有保留库。

独占索引器实例：确认卷内无保留数据库、所有挂载消费者均已列明且退出，停止/移除精确程序与专属 PostgreSQL 容器，再次核对卷无消费者，才删该独占卷。无需重启旧实例补 DROP。每项 Docker 操作预算60秒；锁定/未退出对象保持失败状态，不扩大信号或删除范围。

数据删除之后才清理已确认专属二进制、env、Compose、部署临时包。目录只在内容逐项核实清空后删除；专属镜像必须无任何保留消费者，混合 `PROD_IMAGE` 与 PostgreSQL 基础镜像保留。失败不恢复已删数据，不重新拉起旧业务。

## 结束与证据

逐环境分别记录代码版本、旧程序退出、通知计数、四库/独占卷是否不存在、残留文件、保留服务结果和未完成项。没有访问权限/凭据、无法定位或资源身份改变时，明确具体对象与阻塞，不把其他环境结果代替该环境。

普通本地验收收尾与历史数据永久退役分开记录。由原 worktree 的同一 `INSTANCE` 执行 `make stop`；Compose 按本轮记录的 project/文件/env 精确 `stop -t 30`。停止本轮替身和预览，验证进程/端口/容器退出，保留测试卷、截图、报告、日志。借用的用户/其他任务环境保持原样。
