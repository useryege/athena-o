# Solana 名称、符号与发行来源验收

日期：2026-09-13；验收工作树：`/home/yege/work/athena/.worktrees/solana-discovery`，分支 `codex/solana-discovery`。

## 交付内容与范围

在原有独立 Solana 发现模块中补齐名称、符号和发行来源；保存链上元数据账户、观察 slot、观察时间及各自补全状态。新增候选在扫描时保存发行来源，历史候选通过初始化交易补查。链上名称支持 Token-2022 自指 TokenMetadata 和 canonical Metaplex Metadata；发行协议支持 Pump.fun、Raydium LaunchLab，并区分直接 Token 初始化与未识别平台。

API 与列表展示相同持久化字段，支持名称/符号/Mint 查询、复制、链上链接与展开详情。首次信息补全在后台持续进行，缺失和错误分别重试，不启动研究规则、活动监听或研究生命周期调度。

实现提交：`8b71a886`（后端）、`0a7a26c6`（API/UI/文档）、`5e9802f7`（共享限速冷却修正）。设计与任务范围见[设计](../../superpowers/specs/2026-09-13-solana-metadata-design.md)、[计划](../../superpowers/plans/2026-09-13-solana-metadata.md)。

## 自动验证

| 验证 | 结果与证据 |
| --- | --- |
| 后端专用 PostgreSQL + race | `go test -race -json ./internal/solanadiscovery ./cmd/athena-solana-discovery/commands -count=1`，100 个测试/子测试通过、0 失败、0 跳过；使用任务独有端口 55619，每例创建并删除测试库，未向预览库写入测试样本 |
| 429 恢复与共享限速 | 评审发现冷却期间可能积压已消费令牌；先复现 7 请求在 19.231µs 内集中放行，再统一可取消准入。定向并发、预算、429 与取消测试连续 3 轮 race 通过，实际 RPC 保持并行 |
| API 契约及授权 | `go test ./internal/solanadiscovery/rpcservice ./internal/server/solana -count=1` 通过；所有新增字段及空观察时间均验证 |
| UI 行为 | Solana 页面 Jest 7/7 通过；包括名称、符号、来源、独立缺失状态、脚本文本安全、详情、搜索回第一页、分页与复制链接 |
| UI 静态检查 | 完整 `yarn lint` 通过，含配置测试、TypeScript 和 ESLint |
| 生成与构建 | 隔离 GOPATH 执行 `make protogen` 成功，只保留本轮 Solana 生成契约和 Swagger 改动；服务与 API 独立构建通过；最终预览从修正后的源代码启动 |

服务仍为 Solana 数据唯一 owner（SDS-R1），API/proto 消费一致（SDS-R2），独立入口构建启动（SDS-R3），Solana READ 与可信内部身份保持（SDS-R4），可取消进程退出（SDS-R5），远端请求不跨存储事务、候选和游标原子提交（SDS-R6），独立预览配置与依赖归属明确（SDS-R7），本记录区分隔离测试与实际环境证据（SDS-R8）。

## 真实主网与持久化验证

首次升级前保存快照：起点 `446684678`、已处理 slot `446686907`、至少 `2066` 条候选。升级及再次重启后原起点不变，候选和进度继续增长；2026-09-13 21:07 左右为 `2130` 条候选、已处理 slot `446686973`。

实际旧候选：

| 字段 | 值 |
| --- | --- |
| Mint | `s8z8Xg4b21SnUaucPKEzZ1rCmPFkLxYu2okNHidpump` |
| 名称 / 符号 | `Anita maxyn` / `brotha` |
| 发行来源 / 程序 | `pump_fun` / `6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P` |
| 发行 slot | `446684707` |
| 元数据来源 | `token2022_on_mint`，账户为该 Mint |
| 元数据观察 slot | `446709437` |
| 状态 | metadata `ready`、source `identified` |

独立请求主网 finalized 账户（slot `446708104`）取得的名称/符号，与后台自动入库及 API 返回一致。原发行签名、slot、blockTime、discoveredAt 保留；最终限速修正版再次重启后名称、符号、来源、观察 slot 与观察时间保持。新增候选也持续入库，来源在初次保存时即为 identified 或 unrecognized。

该验证不向扫描结果导入 USDC 等历史样本；Metaplex 使用实际 USDC 主网账户快照作离线解析 fixture。不会把所有新 Mint 当作普通代币，实际候选包含头寸 NFT。

后台队列尚未全部处理：21:08 左右，2130 条记录中 metadata 为 ready 22、unavailable 134、pending 1974；source 为 identified 27、unrecognized 193、pending 1910。该统计仅是当时运行快照，不表示这些数量固定。未处理记录继续后台补全，不能将其描述为全部历史数据已经补完。

原始本地证据位于 `.tmp/solana-metadata/`：`before-restart.json`、`pump-account-live.json`、`pump-expected.json`、`live-verification.json`、`final-live-verification.json`。测试与评审归档在 `review-and-test-evidence/`。

## 浏览器与权限验收

系统 Chrome 实际打开 `http://127.0.0.1:14000/solana`；未拦截或伪造业务响应。桌面 1440×900 和手机 390×844 通过以下断言：

- 名称、符号、Pump.fun 来源和链上详情与真实 API 一致。
- Mint 搜索、名称/符号不区分大小写搜索、手动刷新正常。
- Mint 复制内容正确，Solscan 链接与原地址一致。
- 手机文档宽度 390px，与视口相同；表格自身 clientWidth 360px、scrollWidth 1435px，可横向查看。
- console error 和 page error 均为 0；截图人工检查通过。

业务页面证据为 `.tmp/solana-metadata/browser/report.json`、桌面/手机截图和 `trace.zip`。正式 smoke 只验证会员/管理员应用壳，与上述业务检查分开：最终运行 `.tmp/athena-ui-acceptance/2026-09-13T13-07-36-958Z-6081771e/`，两项测试通过、退出 0。两个 realm 的 bootstrap 均为 200；成员有 Solana READ，管理员业务列表请求仍为 403。

## 环境恢复与交付运行状态

第一次重启时，共享 PostgreSQL/Redis/Minio 已被另一个本地流程删除；Docker 事件时间约 20:55:47–49，早于本轮预览重启。原持久卷仍存在。尝试按原 helper 恢复时，同名容器已被另一个流程重新创建，helper 因名称冲突退出；未删除、替换任何现有容器或数据卷。核对原 `athena-local-postgres-data` 卷、2066 候选及原游标后，重新启动本 worktree 的预览并完成真实重验。

最终保留运行：

- 页面 `http://127.0.0.1:14000/solana`；API 18080；Solana gRPC 18112。
- 数据库 `athena_solana_preview`，位于原本地 PostgreSQL；Redis DB 13。
- worktree `/home/yege/work/athena/.worktrees/solana-discovery`。
- 持续执行会话 `11166`；Goreman PID `4026448`（启动 ticks `29283301`），归属已通过 `/proc` 确认。
- 运行日志 `.tmp/solana-metadata/preview-final.log`；早期失败与恢复日志保留在同目录。
- 停止本预览：从该 worktree 执行 `make stop ATHENA_RUN_PROFILE=solana-preview`。该 profile 只停止自己的进程，保留借用的基础设施和数据。

公共 RPC 容量仍有限，页面如实显示 catching_up；本次不宣称已经追平链头、覆盖所有发行平台或全部历史补全已结束。元数据是观察时发行方自述的信息，不代表平台或项目认证。
