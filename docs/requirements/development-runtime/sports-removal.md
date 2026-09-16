# Sports 板块删除需求与 Worm 边界

> 需求状态：2026-09-16 Sports 删除与专属历史数据直接删除已确认。最新决定另行删除 Worm Markets，仅保留 Trading，见[Worm 删除与保留需求](worm-markets-removal.md)；该决定覆盖本文早先双服务保留表述。本文件继续只负责 Sports 范围。
>
> 实现状态（2026-09-16）：代码、协议、八权限契约、页面与运行配置已删除；空库/升级、通知维护及真实本地保留能力验证已有证据。数据库与运行退役分别记录在[验收记录](../../testing/module-removal-cleanup-acceptance.md)；主生产机按用户后续指示跳过。

## 1. 删除范围

| 对象 | 删除功能 | 现有主要入口 |
| --- | --- | --- |
| Sports Live | 实时赛事同步、价格历史、比分与价格告警、查询和页面 | 命令（历史路径 `cmd/athena-sports-live`，基线 `264d0dc1`）、模块（历史路径 `internal/sportslive`，基线 `264d0dc1`）、[设计](../../design/market-intelligence/sports-live.md) |
| Sports History | 历史赛事同步、启动刷新、手动刷新、价格历史、查询和页面 | 命令（历史路径 `cmd/athena-sports-history`，基线 `264d0dc1`）、模块（历史路径 `internal/sportshistory`，基线 `264d0dc1`）、[设计](../../design/market-intelligence/sports-history.md) |
| World Cup Corners | 世界杯角球统计、筛选、查询、静态数据和页面 | API 内嵌业务（历史路径 `internal/server/worldcupcorners`，基线 `264d0dc1`）、页面（历史路径 `ui/src/app/member/pages/world-cup-corners.tsx`，基线 `264d0dc1`） |

从本地与生产目标服务、会员导航、页面、管理员权限及板块访问开关中移除，不保留隐藏页面或兼容业务接口。这里是两个独立程序和一个 API 内嵌业务。

## 2. 配套清理

- 删除两个命令、业务模块、后台任务、服务健康注册、API facade、proto、生成客户端和专属测试；World Cup Corners 的接口及静态数据一并移除。
- 删除 `/sports-live`、`/sports-history`、`/world-cup-corners` 页面及导航、路由、专属组件和客户端。混合组件按实际使用者保留。
- 清理 `sports_live`、`sports_history`、`world_cup_corners` 三类权限定义、默认授权、已有失效授权、SQL 和 schema 契约。Trading 权限保留；Markets 权限由独立重构清理，两项迁移需协调最终约束。
- 删除 Sports 专属通知生产和配置，按精确来源取消未发送消息；保留共享 Notification 的发送记录及真实结果，不处理 Worm 通知。
- 清理 SQL／迁移源码、sqlc 和其他生成入口；同步生产 Compose、本地运行器、命令分发、Makefile、数据库初始化、环境变量、预检及使用说明。
- 保留服务在缺少 Sports 配置和数据库时仍可构建、初始化和运行。未来实施须包含消费者、生成产物和受影响的验证。

## 3. 历史数据：直接删除，已确认

在对应旧服务退出、没有残留连接和资源归属核对完成后，直接删除 `sports_live`、`sports_history` 数据库及确认仅供其使用的存储；不设置导出备份或归档保留步骤。World Cup Corners 的源码静态数据随功能删除。

实际名称以目标环境核对为准。若数据库位于共享 PostgreSQL 容器或卷中，只删除两个明确数据库，保留容器、卷和其他库。正常 `make run`／`make stop` 不执行该永久数据清理；删除作为一次性退役步骤独立记录。

本文件的删库清单只含 Sports 两库。账户、Wallet、`worm_trading` 和共享通知账本保留；`worm_markets` 直接删除已由独立需求确认，不混入本文件的执行范围。

## 4. Worm 与共用能力保留

- 保留 Worm Trading 的服务、页面、交易、组合、执行、平仓、权限、配置和 `worm_trading` 数据；Markets 的按需交易目录归入 Trading，其余 Markets 能力及专属数据独立退役。
- 保留 Wallet／账户中的 Worm 专属签名、凭据、再认证、钱包选择、连接与授权接口，以及 `util/worm` 等仍使用的工具。不得沿旧草案清理这些内容。
- 保留 API、UI、登录、账户、Wallet、Notification、Etherscan Manager／Gateway 和其余业务；ATHENA 钱包、私钥、头像、资产及外部订单／持仓不因本次删除而处置。
- Polymarket、BSC／EVM、Solana、市场模型和通用 UI 按实际消费者保留。Worm 自身使用的体育市场数据不属于 Sports 板块删除范围。
- v21 的 Sports 页面转为退役记录；Market Radar／Managed OO、v22 Worm／共用界面和全部已确认视觉规则继续有效。
- 运行目标只保留 Trading，`worm` 访问标识与交易处理规则继续适用；相关目标设计尚未实施，Token 详细接入仍延期。

## 5. 完成标准与决定记录

实施完成需要分别给出：仓库及生成入口删除、旧进程退役、三类权限清理、Sports 旧队列取消、两个专属数据库直接删除，以及 Worm／核心能力回归证据。代码删除、实例退出和数据删除分别验收。

2026-09-15 曾确认 Worm 与 Sports 一起删除，[原 R19](business-group-control.md#confirmed)保留该时点事实。2026-09-16 用户明确保留 Worm，覆盖此前 Worm 删除决定；随后确认剩余删除对象的历史数据直接删除。

具体顺序与边界见[删除与清理配套设计](../../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)。2026-09-16 用户再次确认删除范围；追加迁移、schema 契约、专属部署入口清理及失败／中断重入现已实施和验证，逐环境执行证据见验收记录。Worm 的新增全栈启动／统一访问开关仍未实施，属于后续任务；现有能力与数据继续保留。服务边界、事务、有限超时与验证遵守[服务开发规范](../../developer-guide/service-development-standards.md)的适用规则。

2026-09-16 已按用户优先顺序整理[删除清理实施计划](../../superpowers/plans/2026-09-16-module-removal-cleanup.md)，覆盖仓库清理、本地验收、现场退役与直接删库、最终核验。当前执行状态以验收记录为准。

2026-09-16 最新决定：用户要求删除 Markets、选择专属数据直接删除并采用方案 A；按需市场查询归 Trading。此前两者保留的确认记录仅保留历史含义，以[最新需求](worm-markets-removal.md)为准。
