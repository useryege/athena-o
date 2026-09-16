# 两个 BSC 索引器与 Sports 删除清理验收记录

实施开始：2026-09-16 01:52:25 UTC。基线 `rf4@264d0dc1fd524dc23b236f939da1d343572bd030`，根工作区干净。
实施工作区 `/home/yege/work/athena/.worktrees/module-removal`，分支 `codex/module-removal-cleanup`。

## 当前状态

| 层面 | 状态 |
| --- | --- |
| 代码清理 | T1–T6 完成；四程序、三个公共服务及页面、权限、生成产物、启动和部署入口已清理 |
| 运行退役 | 两台 BSC 主机实查目标已不存在，Gateway 保留；本地默认旧生产者已退出，新版保留能力通过验证后恢复原停机状态 |
| 数据删除 | 本地默认实例 Sports 两库已按 OID 删除并复查；两台 BSC 主机的历史独占实例和卷本轮核验不存在，并非本轮执行 DROP |
| 范围调整 | 用户明确要求跳过 `47.245.181.189`；不发布、不删除、不继续检查，不作为本轮阻塞 |
| 环境收尾 | 本任务 14 个容器已停止，16 个端口已释放；19 个原有容器保持原运行／停止状态，普通验收卷和证据保留 |

## T1 基线

证据目录：实施工作区 `.superpowers/module-removal/`。`baseline.json` 记录版本与工具；`source-inventory.txt`（2663 行）、`preserved-inventory.txt`（666 行）、`sports-ui-consumers.txt`（8 行）记录引用；`deployment-locators.txt` 保存旧部署定位与停止边界。Docker、进程和端口快照独立保存。

Go 1.27.1、Node 24.14.1（与 ui/.nvmrc 一致）、Yarn 1.22.22、PostgreSQL 客户端 16.15、Docker 29.6.1；rg/Python 与现有生成工具可用，目标工作区复用已安装的只读工具与 UI 依赖。

基线 `go test ./internal/accountaccess ./internal/devruntime ./internal/migration ./internal/notification/store -count=1` 退出 0，前三包有测试；Notification store 无普通测试文件，只计构建，后续使用独立 PostgreSQL 集成测试。日志 `logs/t1-baseline.log`。

现有 make run 实际六程序：Trader Sync、API、Notification、Wallet、Profit Sharing、UI。Worm 现有运行入口为生产 Compose 的两个服务，T6 另建隔离 project 验证，不扩展十二应用编排。

sports-models/sports-market-card 仅由待删 Sports 页面及专属测试使用。共享 Polymarket、BSC/EVM、Worm 体育市场语义保留；Worm 权限、签名、再认证、交易与通知维持。允许旧名称保留的位置：历史迁移、通知退役六精确来源、删除回归断言和明确历史的文档/证据。

## 环境归属

任务前本机已有 ATHENA 停止实例和其他项目运行容器。默认 `full-stack` 实例只在本轮已授权的权限迁移和 Sports 历史库退役范围内维护，最后恢复原有停止状态；其他实例和项目保持原样。普通验收数据保留与历史四库永久退役分别记录，详见下文。

## T2 通知维护

实现默认只读、显式 --apply、总期限与每次 5 秒操作期限、100 条批次、独占 session advisory lock 和精确六来源。pending 取消与发送许可共享 delivery 行锁，sending/终态/发送尝试保留。

独立 PostgreSQL：容器 `athena-module-removal-20260916-tests`（ID `1ee650507fde`），loopback `127.0.0.1:61752`，同名数据卷，owner 标签为当前实施工作区；fixture 只创建随机 athena_test 库。退役工具与存储集成测试通过，CLI 普通测试和构建通过。日志：`t2-red.log`、`t2-cli-red.log`、`t2-sqlc.log`、`t2-green.log`（包含已修复的 fixture Topic 缺失）、`t2-cli-green.log`。未调用 Telegram，未取得 sender 身份。

## T3 后端与账户

四个程序/实现、三个 API facade、旧公共客户端、权限映射与业务注册已删除。追加 `000004_remove_sports_access.sql`，保留 000001–000003 原文；proto 2/3/7 和旧名称 reserved，其余编号不变。空库与升级库 catalog 校验通过，Sports 行清理、八项授权、Worm/Wallet 权限、login/API Key、Pending 和迁移失败原子回滚/重入均通过数据库测试。

运行日志：`t3-go-retry.log`、`t3-store-retry.log`、`t3-integration.log`（初次旧模块数量断言失配已在重验修正）、`t3-build.log`。schema、txgate 与 migrate 命令集成通过；API、账户、迁移、服务命令测试及五个消费者构建通过。

sqlc、account-state-schema-contract、protogen、clientgen 已从源生成。protogen 使用任务独立 GOPATH，首次第三方 proto 搜索路径失败已修正后成功；31 个只有 gzip 压缩字节差异的产物在解压描述符与剩余源码均逐字一致后恢复原文件，清单 `generator-compression-only.txt`。所有日志保留；未手改生成逻辑。

## T4 前端

三个页面、菜单、权限、专属服务与样式已移除；混合市场场景保留 Radar、Managed OO 和全部 Worm 场景。新增旧路由未找到、无旧请求、管理员八权限和七条 Worm 路由场景。T4 五个 Jest suites / 16 tests、lint 与生产 UI build 通过；完整 Playwright 和真实环境结果见 T6。证据 `t4-tests-retry.log`、`t4-lint-final.log`、`t4-build.log`。

## T5 启动与部署

现有六程序编排保留，数据库准备不再创建 Sports 两库；Compose、索引器构建/部署入口及精确配置清理。账户 schema 维护根据 `io.athena.account-state.consumer: "true"` 标签识别实际消费者（含 profile 服务），核对各自 DSN 与 authority 一致，拒绝缺失或不同 DSN。仅继承 env_file 中 DSN 不视为消费者；现场额外 Solana 等使用者必须同样声明。维护前记录运行项，停止并核验退出，迁移与 verify 后显式恢复原先运行的消费者；profile 状态在同一远程执行上下文中保留。

runtime/migration/CLI 测试、实际 Compose 解析、部署脚本模拟及改动脚本 ShellCheck 均通过：`t5-runtime-correct.log`、`t5-compose.log`、`t5-deploy-final.log`、`t5-shellcheck-final.log`。两次手动验证命令误写旧 CLI 路径的 setup failure 已改为实际 `cmd/athena-local-runtime` 重验；产品代码无对应失败。审阅修正后的重验为 `t6-deploy-review-green.log`、`t6-compose-review-green.log`、`t6-shellcheck-review.log`。专属历史产物实际删除结果见 T8。

## T6 测试与真实本地验收

以下日志均位于实施工作区 `.superpowers/module-removal/logs/`，浏览器报告位于 `.tmp/athena-ui-acceptance/`。最终运行的源码行为与已提交一致，后续仅补全文档和现场证据。

| 验证 | 结果与证据 |
| --- | --- |
| 计划列出的 Go 保留模块、API、账户、通知、Wallet、runtime、migration、工具回归 | 通过，`t6-go-unit.log`；Wallet 包包含实际 gRPC wire、认证、取消和 deadline 回归 |
| 独立真实 PostgreSQL 集成 | accountstate/schema/store/txgate、Notification store、退役 CLI 全通过，`t6-go-integration.log`；fixture 使用本任务隔离 PG，未连接生产库 |
| 全部 cmd 与退役工具构建 | 通过，`t6-go-build.log`；Worm 业务包显示 `[no test files]` 仅计构建，业务证据由页面场景、Wallet wire 和下述真实读取补齐 |
| 完整 Jest | 37 suites、417 tests 通过，`t6-ui-jest-all.log`；修复旧 11 项数量断言，保留 Trader Sync READ 拒绝／READ_WRITE 通过语义 |
| 最终 lint／生产 build | lint 通过，`t6-ui-lint-delivery.log`；完整浏览器入口重建生产 UI 通过 |
| 完整隔离浏览器 | `2026-09-16T02-41-58-832Z-36209e76/report.md`：passed，cleanup passed；root 与 `/athena` 各 491 fixtures、10 live，共 1002 项 |
| 原生 make run 空实例 smoke | `2026-09-16T02-27-26-551Z-b3715201/report.md`：passed；真实 member/admin bootstrap 已核验 |
| 原生已有权限升级 smoke | `2026-09-16T02-38-56-288Z-1cc557aa/report.md`：passed；同一测试实例旧 v3 的两账户 11 项权限，经正常启动升级为八项，UUID 及 Worm/Wallet 权限保留 |
| 独立生产 Compose smoke | `2026-09-16T02-30-04-927Z-b94e38eb/report.md`：passed；明确使用 `http://localhost:61770`，未混接原生实例 |

隔离浏览器的 fixtures 使用模拟 API；live 使用真实产品组件和临时 PostgreSQL，链、资料与 Telegram 为本地替身。真实 smoke 验证应用壳与服务链路，不能单独证明 Worm 交易。本轮没有真实下单、Cash Out、平仓或发送测试通知。Worm 钱包选择／连接、组合、授权／预览、执行和 Cash Out 状态通过隔离 UI 覆盖；真实环境仅执行读取。

Worm 独立 Compose project 为 `athena-module-removal-20260916`，使用新构建的 `athena:module-removal-20260916` 及独立 Trader Sync 镜像。三个 external 卷分别为该 project 前缀的 `-compose-postgres`、`-compose-redis`、`-compose-minio`，创建前核对不存在，均有 owner 标签。完整解析配置、独立 env、最小本地覆盖与镜像构建记录在 `.superpowers/module-removal/compose/`；未直接使用 `.env.prod`。API 的 disabled-auth 仍只监听容器 loopback，本地专用代理承接 host loopback 61770；未放宽产品监听约束。所有业务地址、存储及内部凭据均指向本 project；本地 Telegram 替身禁止外发，不接管原环境 poller。

`compose/member-status-markets.json` 记录 running，`member-events-markets.json` 获得真实上游 20 条赛事；`member-status-trading.json` 中 RPC、genesis、USDC 验证成功，credential store ready。无用户交易凭据，`wormApiStatus=unknown` 如实保留，不宣称已验证真实下单能力。`compose/real-browser-reads.json` 与三张 `worm-*-real.png` 记录 Assets、Combinations、Executions 页面及仅 GET 的真实请求，无页面异常。

原生实例使用 `INSTANCE=module-removal-20260916`、UI 61761／API 61760／Trader Sync 61762／Notification 61763／Wallet 61764／Profit Sharing 61765。启动日志 `local-run*.log`、`native-upgrade-run.log`；替身缺少 Bot 元数据方法造成的初次启动失败已修正后重验，未修改产品来绕过校验。旧 v3 fixture 仅写本任务数据库，SQL 与前后结果为 `native-upgrade-fixture.sql`、`native-upgrade-before.log`、`native-upgrade-after.log`。

完整浏览器前两轮证据保留：`2026-09-16T02-19-09-767Z-621f10b5` 暴露旧 Sports 场景清单和旧权限数量；`2026-09-16T02-29-56-158Z-edeb4104` 为 490 通过、1 个已有 Profile 弹窗动画时序断言失败。已清理专属场景，以保留模块维持共享错误布局覆盖；两个按钮的位置改为同一帧读取，未放宽布局断言。最终完整重跑通过。

独立审阅发现并验证修复四个 P2：维护 advisory lock 占用单连接池容量（改为 Hijack 专用物理连接，`pool_max_conns=1` 红绿用例）；继承 DSN 造成误停非消费者（改显式标签）；SSH 上下文丢失 profile 恢复名单（同上下文保存并显式恢复）；残留 11 项权限单测。证据包括 `t6-review-red.log`／`t6-review-green.log`、部署审阅重验和 `t6-models-review-red.log`／完整 Jest。无未解决审阅项。

生成器仅按真实源变更运行；T6 `make clidocsgen` 同步命令说明并保留人工本地运行章节。手册见[退役操作手册](../operator-manual/module-removal-retirement.md)。服务规范证据：SDS-R2/R3 为公共接口精确删除及独立构建，R4 为八权限与 Worm wire/UI 授权回归，R5 为追加原子迁移及真实 PG 并发验证，R6 为两套归属明确的本地环境，R7/R8 为有界工具、逐环境身份清单、真实验收和精确收尾；没有新增服务边界耦合。

## T7–T8 逐环境退役与实际数据处理

统一结果为 `.superpowers/module-removal/retirement/environment-results.json`；原始只读 inventory、资源身份、后续步骤均保留，未导出业务历史数据或建立数据库归档。

| 环境 | 旧程序／自动拉起 | 数据与专属资源 | 保留及结果 |
| --- | --- | --- | --- |
| `47.245.183.140`，普通交易索引器 | 实查无目标进程、容器、部署目录和自启动引用 | Docker 容器／卷均为 0，无专属镜像；历史独占 `bsc_inbound` 实例本轮核验不存在，未执行 DROP／删卷 | Gateway PID 1402650，6776，gRPC Health SERVING；主机、Gateway、PostgreSQL 基础镜像保留 |
| `47.254.154.128`，Swap 索引器 | 实查无目标进程、容器、部署目录和自启动引用 | Docker 容器／卷均为 0，无专属镜像；历史独占 `bsc_swap` 实例本轮核验不存在，未执行 DROP／删卷 | Gateway PID 755335，6776，gRPC Health SERVING；主机、Gateway、PostgreSQL 基础镜像保留 |
| `47.245.181.189`，原主生产线索 | **用户明确跳过** | 无发布或删除；保留指示前只读证据，不再处理 | 不写作已退役，不列为本轮阻塞 |
| 本地原根工作区默认 `full-stack` | 原运行器记录为 stopped，逐 PID/start ticks 核验旧消费者已退出；无 Sports／BSC 生产者在线 | `sports_live` OID 17155、`sports_history` OID 17156 已 DROP 并逐库确认不存在；两个 BSC 库原本不存在；共享 PG 卷保留 | 新 authority up/verify、新 API/Wallet/Worm 读取通过；三个原有基础设施恢复 stopped |
| 本机其他历史集成／worktree 实例 | 只读记录归属，保持原状态 | 其他任务普通测试库／卷不属于本轮历史业务库退役目标，保持原样 | 最终容器状态与任务前记录一致 |

两台索引器机器 ID 分别为 `20e4754167104b38bf6177a9e165c48c`、`0a60688187424f40944d13d7c290dd62`。证据为各 IP 的 `-inventory.json`、`-residuals.json`、`-gateway-health-verified.json`。首次 grpcurl reflection 请求不支持，随后提供标准 health proto 明确调用 Health/Check，返回 SERVING；不能把 reflection 失败写成 Gateway 故障。

本地默认实例归属 `/home/yege/work/athena`，namespace `ac252d3201cc3c6d872334f8e6a1bcf5`。PostgreSQL 容器 `c2385c5962b1d7bde1d01684161aac3a7d6c80710f3bd4e31e22c19be7dfc375`，system identifier `7685665987292844070`，目标两库 owner 均为 `athena`。`local-root-full-stack-databases.json` 与 `local-default/readonly-inventory.json` 先记录只读清单；删除前重新核对全部卷消费者、实例、OID、owner 和连接，无强踢连接、FORCE 或整卷删除。

`local-default/result.json` 记录最终 completed、cleanup 无错误。新版本在 61780–61783 临时端口使用原默认实例自己的 PostgreSQL、Redis、MinIO 与 Wallet 密钥：全部账户库消费者退出后 up→verify，运行维护工具只读和 apply，六来源 pending=0、sending=0、cancelled=0、counts_verified=true。Notification 原本停止且账本无待发送数据，因此没有启动或接管原 poller；共享投递／发送尝试仍为 0，原有 topic 1 条保留。新 member/admin bootstrap、八权限、Wallet、Worm 状态／行情、旧 API 404 均通过后才删库。

默认实例初次只读盘点时 Worm 两库的业务表为空；Worm Trading 无历史连接／凭据，使用仅本次验证的临时加密配置，没有创建或轮换业务凭据。实际同步产生的 Worm 行情与价格历史保留。Wallet、Worm Trading 表计数未改变，其他数据库名称/OID/owner 与删除前一致，账户和 Trader Sync 数据保留。原 `rf4` 源码未切换；该默认账户库已升级至 schema 4，后续访问应使用本清理分支的新版本，旧 schema authority 不能继续管理它。

维护脚本前三次在删除前正确停止：缺少 realm 请求头导致 anonymous、临时端口占用、把旧 API 404 文本当 JSON 解析。修正脚本后重新核对身份执行，未为验证改动产品认证规则；三次均恢复原有基础设施停止状态，未 DROP。失败结果与最终结果并存于 `local-default/` 和 `logs/local-default-retirement-*.log`。

本地专属产物清理：核对 Go build path、文件 SHA256、实际 `/proc/*/exe` 使用者、镜像入口与全部运行／停止容器引用后，删除原根目录 `dist/athena-bsc-transaction-indexer`、`dist/athena-bsc-swap-indexer` 及 6 个无消费者的索引器专属镜像。精确对象与结果为 `local-exclusive-artifacts.json`／`local-exclusive-artifacts-result.json`。共享 athena、Trader Sync、PostgreSQL 等镜像未删除；源码历史和其他 worktree 不清理。

## T9 环境收尾与交付

`cleanup-verification.json` 为最终通过的归属与退出核验，`standalone-cleanup.json` 记录独立 PG 和 Telegram 替身。没有因清理本任务环境执行 run-reset、prune、down --remove-orphans 或删除普通验收卷。

| 环境 | 停止入口与验证 | 保留内容 |
| --- | --- | --- |
| 原生实例 `module-removal-20260916` | 在实施 worktree 执行 `make stop INSTANCE=module-removal-20260916`；state=stopped，原 PID/start ticks 不再运行，61760–61765 关闭；`logs/local-stop-final.log` | 该 namespace 三个数据卷、运行器记录、日志 |
| Compose project `athena-module-removal-20260916` | 同 worktree 执行 `bash .superpowers/module-removal/compose/compose.sh stop -t 30`；全部容器停止、61770 关闭；`logs/compose-stop-final.log` | 三个明确命名的 external 卷、新版镜像、配置、截图与报告 |
| 集成测试 PostgreSQL | `docker stop -t 30 1ee650507fde74ccbec2cf6ee11eb1c1cb8dbd3eeac23f6eb3ee9319ea2efdfe`；容器停止，61752 关闭 | 同名 `athena-module-removal-20260916-tests` 数据卷 |
| 独立 Telegram 替身 | 核对 worktree 与命令后正常终止 PID 827980；61759 关闭 | 脚本及日志 |
| 浏览器 harness | 最终 report cleanup=passed，测试进程退出，任务 harness 无运行容器 | `.tmp/athena-ui-acceptance/` 所有成功与失败报告／截图／trace |
| 默认实例临时维护 | 四个临时新版进程正常退出，61780–61786 关闭；原 PG/Redis/MinIO 三容器恢复 stopped | 除已授权删除的两个 Sports 库外，原数据库、共享卷与配置保留 |

仍保留运行的环境只有原有其他项目容器和远端 Gateway，均非本任务新启动；具体 ID、名称和状态在 `cleanup-verification.json` 的 `original_containers_preserved`，两台远端地址和 unit 见前表。它们不需要本任务执行停止。`47.245.181.189` 按用户指示完全排除。

阶段本地提交：`5ebe1d15` 基线、`6245c8c3` 通知工具、`b2496710` 后端／权限、`8ec5b84b` UI、`bb71f79c` runtime／部署、`e196dc4f` 单连接池修正、`888580b9` 消费者维护修正、`b724d60e` 完整验收场景与手册。最终文档另以收尾提交保存；未 push、创建 PR 或合并。worktree 与证据保留供审阅。

并行工作保护：任务期间原根工作区 `rf4` 新增 `c841579e`、`4a777391` 两个 Worm Markets 后续规划提交；最终根工作区干净，HEAD 为 `4a7773910603d149aab1f31c7eb75fa722806d65`。本清理分支仍基于原批准的 `264d0dc1`，没有合并、回退或覆盖这些规划，也未把后续 Worm Markets 删除扩大为本任务操作。以后整合两个分支时需要保留这组后续规划。

`verified-artifacts.json` 保存二进制 SHA256 与镜像 ID。运行二进制构建于 02:22 UTC，内嵌版本为 `bb71f79c.dirty`（包含尚未提交的通知锁修正），后续产品 Go/UI 行为未变化；最终部署标签和恢复逻辑以 `888580b9` 为准。证据区分实际构建版本与最终提交，不伪称镜像内版本为文档 HEAD。

按用户调整后的本轮范围，没有未完成实施／验收或未解决阻塞。跳过主机、原有其他任务环境、后续十二应用编排、访问开关及 Token 重构均不在本轮完成声明内。任务累计超过 600 秒，最终结果邮件按 AGENTS.md 在收尾完成后发送一次，通知执行结果另保存在证据目录。
