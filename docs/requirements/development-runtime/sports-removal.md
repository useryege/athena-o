# Sports 板块删除需求与 Worm 保留决定

> 需求状态：2026-09-16 已确认。用户先明确“worm markets和worm trading 不要删除”，随后确认“历史数据直接删除”。本轮删除范围为 Sports Live、Sports History、World Cup Corners；Worm Markets／Trading 及其配套功能和数据全部保留。
>
> 实现状态：尚未实施。当前只维护设计，未删除代码、配置、进程、数据库或远端资源。历史数据决定不代表已经执行删除。

## 1. 删除范围

| 对象 | 删除功能 | 现有主要入口 |
| --- | --- | --- |
| Sports Live | 实时赛事同步、价格历史、比分与价格告警、查询和页面 | [命令](../../../cmd/athena-sports-live)、[模块](../../../internal/sportslive)、[设计](../../design/market-intelligence/sports-live.md) |
| Sports History | 历史赛事同步、启动刷新、手动刷新、价格历史、查询和页面 | [命令](../../../cmd/athena-sports-history)、[模块](../../../internal/sportshistory)、[设计](../../design/market-intelligence/sports-history.md) |
| World Cup Corners | 世界杯角球统计、筛选、查询、静态数据和页面 | [API 内嵌业务](../../../internal/server/worldcupcorners)、[页面](../../../ui/src/app/member/pages/world-cup-corners.tsx) |

从本地与生产目标服务、会员导航、页面、管理员权限及板块访问开关中移除，不保留隐藏页面或兼容业务接口。这里是两个独立程序和一个 API 内嵌业务。

## 2. 配套清理

- 删除两个命令、业务模块、后台任务、服务健康注册、API facade、proto、生成客户端和专属测试；World Cup Corners 的接口及静态数据一并移除。
- 删除 `/sports-live`、`/sports-history`、`/world-cup-corners` 页面及导航、路由、专属组件和客户端。混合组件按实际使用者保留。
- 清理 `sports_live`、`sports_history`、`world_cup_corners` 三类权限定义、默认授权、已有失效授权、SQL 和 schema 契约。Worm 的两类权限保留。
- 删除 Sports 专属通知生产和配置，按精确来源取消未发送消息；保留共享 Notification 的发送记录及真实结果，不处理 Worm 通知。
- 清理 SQL／迁移源码、sqlc 和其他生成入口；同步生产 Compose、本地运行器、命令分发、Makefile、数据库初始化、环境变量、预检及使用说明。
- 保留服务在缺少 Sports 配置和数据库时仍可构建、初始化和运行。未来实施须包含消费者、生成产物和受影响的验证。

## 3. 历史数据：直接删除，已确认

在对应旧服务退出、没有残留连接和资源归属核对完成后，直接删除 `sports_live`、`sports_history` 数据库及确认仅供其使用的存储；不设置导出备份或归档保留步骤。World Cup Corners 的源码静态数据随功能删除。

实际名称以目标环境核对为准。若数据库位于共享 PostgreSQL 容器或卷中，只删除两个明确数据库，保留容器、卷和其他库。正常 `make run`／`make stop` 不执行该永久数据清理；删除作为一次性退役步骤独立记录。

这项决定仅覆盖删除对象的数据。账户、Wallet、Worm 两个数据库及共享通知账本不在直接删除清单内。

## 4. Worm 与共用能力保留

- Worm Markets 和 Worm Trading 全部保留：服务、页面、查询、交易、组合、执行、平仓、告警、权限、部署和配置，以及 `worm_markets`／`worm_trading` 数据库。
- 保留 Wallet／账户中的 Worm 专属签名、凭据、再认证、钱包选择、连接与授权接口，以及 `util/worm` 等仍使用的工具。不得沿旧草案清理这些内容。
- 保留 API、UI、登录、账户、Wallet、Notification、Etherscan Manager／Gateway 和其余业务；ATHENA 钱包、私钥、头像、资产及外部订单／持仓不因本次删除而处置。
- Polymarket、BSC／EVM、Solana、市场模型和通用 UI 按实际消费者保留。Worm 自身使用的体育市场数据不属于 Sports 板块删除范围。
- v21 的 Sports 页面转为退役记录；Market Radar／Managed OO、v22 Worm／共用界面和全部已确认视觉规则继续有效。
- Worm 回到保留服务与访问控制目标范围；运行清单已补回两个进程，访问设计已补齐共用 Worm 开关、全部用户入口与交易处理边界，尚未实施。Token 详细接入仍延期。

## 5. 完成标准与决定记录

实施完成需要分别给出：仓库及生成入口删除、旧进程退役、三类权限清理、Sports 旧队列取消、两个专属数据库直接删除，以及 Worm／核心能力回归证据。代码删除、实例退出和数据删除分别验收。

2026-09-15 曾确认 Worm 与 Sports 一起删除，[原 R19](business-group-control.md#confirmed)保留该时点事实。2026-09-16 用户明确保留 Worm，覆盖此前 Worm 删除决定；随后确认剩余删除对象的历史数据直接删除。

具体顺序与边界见[删除与清理配套设计](../../superpowers/specs/2026-09-16-module-removal-cleanup-design.md)。2026-09-16 用户再次确认删除范围；配套自查已补齐追加迁移和 schema 契约、专属部署产物清理、失败与中断重入，尚未实施。Worm 的新增启动／访问接入已由对应设计补齐，尚未实施；保留其现有能力与数据的决定不变。遵守[服务开发规范](../../developer-guide/service-development-standards.md)的适用规则；本轮仅以文档链接、范围及格式检查验证设计维护。

2026-09-16 已按用户优先顺序整理[删除清理实施计划](../../superpowers/plans/2026-09-16-module-removal-cleanup.md)，覆盖仓库清理、本地验收、现场退役与直接删库、最终核验。当前仅完成规划，尚未执行删除。
