# 两个 BSC 索引器与 Sports 删除清理验收记录

实施开始：2026-09-16 01:52:25 UTC。基线 `rf4@264d0dc1fd524dc23b236f939da1d343572bd030`，根工作区干净。
实施工作区 `/home/yege/work/athena/.worktrees/module-removal`，分支 `codex/module-removal-cleanup`。

## 当前状态

| 层面 | 状态 |
| --- | --- |
| 代码清理 | T1–T2 完成，T3 开始 |
| 运行退役 | 尚未执行，历史 IP 与默认卷不视为现场事实 |
| 数据删除 | 尚未执行；四库直接删除策略已批准 |

## T1 基线

证据目录：实施工作区 `.superpowers/module-removal/`。`baseline.json` 记录版本与工具；`source-inventory.txt`（2663 行）、`preserved-inventory.txt`（666 行）、`sports-ui-consumers.txt`（8 行）记录引用；`deployment-locators.txt` 保存旧部署定位与停止边界。Docker、进程和端口快照独立保存。

Go 1.27.1、Node 24.14.1（与 ui/.nvmrc 一致）、Yarn 1.22.22、PostgreSQL 客户端 16.15、Docker 29.6.1；rg/Python 与现有生成工具可用，目标工作区复用已安装的只读工具与 UI 依赖。

基线 `go test ./internal/accountaccess ./internal/devruntime ./internal/migration ./internal/notification/store -count=1` 退出 0，前三包有测试；Notification store 无普通测试文件，只计构建，后续使用独立 PostgreSQL 集成测试。日志 `logs/t1-baseline.log`。

现有 make run 实际六程序：Trader Sync、API、Notification、Wallet、Profit Sharing、UI。Worm 现有运行入口为生产 Compose 的两个服务，T6 另建隔离 project 验证，不扩展十二应用编排。

sports-models/sports-market-card 仅由待删 Sports 页面及专属测试使用。共享 Polymarket、BSC/EVM、Worm 体育市场语义保留；Worm 权限、签名、再认证、交易与通知维持。允许旧名称保留的位置：历史迁移、通知退役六精确来源、删除回归断言和明确历史的文档/证据。

## 环境归属

任务前本机已有 ATHENA 停止实例和其他项目运行容器；全部保持原样。任务新增环境与其停止结果后续逐项记录。普通验收数据保留与历史四库永久退役分别记录。

## T2 通知维护

实现默认只读、显式 --apply、总期限与每次 5 秒操作期限、100 条批次、独占 session advisory lock 和精确六来源。pending 取消与发送许可共享 delivery 行锁，sending/终态/发送尝试保留。

独立 PostgreSQL：容器 `athena-module-removal-20260916-tests`（ID `1ee650507fde`），loopback `127.0.0.1:61752`，同名数据卷，owner 标签为当前实施工作区；fixture 只创建随机 athena_test 库。退役工具与存储集成测试通过，CLI 普通测试和构建通过。日志：`t2-red.log`、`t2-cli-red.log`、`t2-sqlc.log`、`t2-green.log`（包含已修复的 fixture Topic 缺失）、`t2-cli-green.log`。未调用 Telegram，未取得 sender 身份。
