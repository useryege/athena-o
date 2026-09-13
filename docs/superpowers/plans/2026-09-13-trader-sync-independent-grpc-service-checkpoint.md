# Trader Sync 独立服务：暂停交接

本记录保留2026-09-13 22:00（Asia/Shanghai）的暂停现场。用户随后已明确要求继续执行，Redis最小修复与保卷恢复、两次真实重启和最终Chrome验收已完成，整体审阅通过；最新状态以[验收报告](../../testing/trader-sync-independent-service-acceptance.md)和执行ledger为准。以下PID、地址和未完成项描述暂停时点，继续执行需重新核验。

## 已保存的实现

- 工作区 `/home/yege/work/athena/.worktrees/trader-sync-independent`，分支 `codex/trader-sync-independent`，基线 `3b1cd556`。原root `rf4`与其他worktree不是本次代码工作区。
- 独立Trader Sync gRPC/runtime、API facade、独立Notification、单活generation guard/原子事务适配器、schema工具、按实例运行器、全栈归属及TLS镜像均已实现。完整[12项计划](2026-09-13-trader-sync-independent-grpc-service.md)保留逐项进度；[验收报告](../../testing/trader-sync-independent-service-acceptance.md)记录全部成功、失败、限制和实施裁定。
- 最终业务修复 `176a46de` 完成旧epoch同步收尾的启动屏障、故障测试当前数据库隔离及两处文档修正；一次限定范围复审 Approved。真实RED/GREEN、关联三包race、独立构建通过。
- 最终镜像 `athena-trader-sync:final`，ID `sha256:24759bfbb97428bc78c52b8961d53c7b348b180797f7527fb9b5c03f5d139e46`；源码176a46de，实际TLS health、schema、非root、正常停止通过。
- 两跳/后端完整race、UI双base、首轮真实全栈Chrome、两方向stop/reset隔离均有通过证据。最终TS新binary受控验收35.256s通过。后续新全栈重启失败，不能把首轮旧run的smoke当作当前运行事实。

## 暂停现场

22:00:01只读核对已保存为证据目录的 `final-fix-runtime-pause-snapshot.json` / `final-fix-runtime-pause-report.md`。证据目录为本worktree `.superpowers/sdd/2026-09-13-trader-sync-independent-grpc-service/`，0600环境文件、详细日志和临时证据仍保留。

| 资源 | 暂停状态 |
| --- | --- |
| 独立 `ts-acceptance` | 运行中，health SERVING；TS `127.0.0.1:28122`，PID536539；supervisor535729，持久session94885；日志 `final-fix-ts-run.log`。 |
| TS 数据库/运行身份 | PG `127.0.0.1:56711`，数据库athena/OID16384；run `abce4e42-b3e0-44f0-9e09-1b14b84641b5`，generation3/epoch4。原账户、业务数据、配置hash、容器/volume IDs保留。 |
| 本次 `full-stack` | stopped，重启run `6f43e32e-f84a-4092-b68e-bf8138f4cc15` 在Redis启动时失败，session78615已exit2；三个容器停止、PG/Redis/MinIO持久volume及归属记录完整。UI24000/API28080此刻不作为可用入口。 |
| Telegram loopback fixture | 当前未运行，历史PID4126952/session35842及39131监听已不存在，退出原因无证据；暂停期间未重启。脚本/旧日志保留。 |
| 专用集成测试PG | `athena-ts-independent-tests-0ff7d993` 仍保留，端口62881；准确ID/标签/测试admin DSN由 `test-postgres.json` / `test-env.sh`记录，不是用户数据库。后续可能还需验证，因此未清理。 |

如需手动停止仍运行的独立TS，从本worktree执行 `make stop-instance INSTANCE=ts-acceptance`，保留数据。暂停操作没有执行该停止，也没有reset/删除数据卷。原root曾发生的旧脚本测试事故及恢复已写入验收报告；不要误把原root当作待清理或待重启实例。

## 待处理的准确问题

`full-stack` 正常stop后原配置run失败。重复准确命令 `docker container start d9e75b6aad544e55f1711624ce3518b265ee052d5d13df9165e81a7c27a8a4b0` 稳定得到Docker Desktop WSL bind源ENOENT；当前本worktree `.run/instances/full-stack/redis.conf` 仍存在且0600。容器标签、mount spec与state一致，数据卷未丢失。

`prepareAPIInfrastructure`每轮调用 `SaveSecret("redis.conf")` 原子替换文件，而容器被复用；同内容文件的inode变化是直接原因假设，尚需隔离RED确认。禁止用reset删除数据来回避持久重启问题。`redis-restart-fix-brief.md` 已准备，**尚未派发或实现**。

## 恢复顺序

1. 先核对本worktree git/state/实际进程，读取证据目录 `progress.md`、`final-review.md`、`redis-restart-fix-brief.md` 和上述pause报告；不重做已完成任务。
2. 按systematic-debugging/TDD，在新独立fixture复现Redis文件bind重启，实施最小修复并测试。优先对同内容普通文件保留inode及0600；变值仍原子写入，不能保留不安全symlink。检查真实两次停止/启动后的Redis数据及头像对象保留，完成限定范围独立审阅。
3. 重新验证本次坏Redis容器的精确归属/停止身份，保留原volume及secret，做受控容器恢复。不要新增历史兼容路径或删除任何volume，不动原root及其他实例。
4. 恢复本任务loopback fixture并记录新身份/session，使用证据目录 `task-12b-fullstack.env` 与Node24启动原full-stack；不重新seed。核对原数据库/账户/配置/数据保持，两次真实持久重启后验证公开接口及Chrome smoke，保留新运行会话和日志。
5. 更新文档与最终整体审阅。TS业务/协议未再变时无需重做已通过的两跳全矩阵或TLS矩阵；仅按新修复影响选择验证。当前整体审阅Pending，Ready to merge为No。
6. 全部必需项通过后才做最终资源收尾，并从worktree根目录按AGENTS发送唯一完成邮件；目前从未发送。未经后续授权不合并或发布。

全栈恢复命令和更细的身份约束见 `final-fix-runtime-pause-report.md`。继续时应触发有界Redis修复代理，再复用运行验收与审阅代理；暂停期间没有代理继续修改代码或操作环境。
