# 板块访问开关：简化方案

> 状态：范围、默认／重启规则于 2026-09-15 确认，2026-09-17 按[修订技术方案](../../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)完成实施，并以代码版本 `6b2db2ec8ca5d488a672e68785eea3272af4f50c` 完成十一应用、双 realm、根路径与 `/athena` 前缀的真实验收。证据见[全栈启动与访问控制验收记录](../../testing/full-stack-access-acceptance.md)。本文记录当前长期契约，不表示人工最终验收通过。
>
> 原[整组运行控制需求](business-group-control.md)及 D01–D03 技术设计继续暂停。访问开关不停止服务或后台任务。Market Radar、Managed OO、Profit Sharing 的新增内部鉴权、Actor 协议和账户连接池不在本轮范围；Token 接入继续延期。

## 1. 目标与当前实现

管理员可分别设置 Trader Sync、Solana、Market Radar、Managed OO、Profit Sharing、Worm Trading 是否开放新的用户业务请求。关闭后，API Server 在原身份、角色、模块权限或业务资格检查之外执行板块准入；进程、后台扫描、已受理工作和业务通知继续按原规则运行。

| 部分 | 当前职责 |
| --- | --- |
| 管理员后台 | Service Status 的 Module Access 第四页签显示六项设置、最后修改人和时间，并提交明确的 OPEN／CLOSED 值 |
| API Server | 对登记的公共 gRPC、HTTP 和原始 Trading 路径执行统一准入；核心、健康和开关管理入口不受业务开关阻断 |
| PostgreSQL | `athena_module_access_setting` 持久保存六个固定键；缺行等同 CLOSED，读取失败按暂不可用关闭处理 |
| 会员壳 | 可见时 5 秒单飞刷新；关闭后中止请求、卸载业务正文并保留菜单与 URL；重新开放后重新读取 |
| 业务服务 | 不感知访问开关，不新增分布式启停、代次、成员确认或收尾协议 |

固定键为 `trader_sync`、`solana`、`market_radar`、`managed_oo`、`profit_sharing`、`worm`。`worm` 只对应 Worm Trading；Worm Markets 已退役。Token 保留产品板块和账户权限，但不在六键、管理员设置或本轮十一应用中。

HTTP 沿现有 `encoding/json` 契约返回 snake_case 字段和数值枚举：OPEN 为 `1`，CLOSED 为 `2`；审计时间为 RFC3339Nano 字符串。接口源为 [`moduleaccess.proto`](../../../internal/server/moduleaccess/moduleaccess.proto)：会员读取 `/api/v1/module-access-states`，管理员读取和更新 `/api/v1/admin/module-access-settings`。不提供 camelCase、字符串枚举或双格式兼容。

## 2. 默认、持久化与并发

- 新环境未配置的六项全部 CLOSED；缺行不能默认为开放。
- 管理员写入按数据库最后提交值生效，保存 `updated_by_account_id`、安全用户名投影与 RFC3339Nano `updated_at`。
- `make run`、服务重启和同一实例重新启动不重置设置，也不伪造首次修改审计；账户 access revision 不因板块开关改变。
- 设置保存成功后，尚未经过准入点的新请求读取新值；已经通过准入的请求自然完成。不会取消已受理任务或撤回外部动作。
- 无法确认设置时返回专用不可用错误，不能冒充管理员主动 CLOSED。关闭与服务不健康分别展示和诊断。

真实验收覆盖全新实例默认关闭、逐项开放与关闭、第二次重启后六项值和完整审计保持、核心接口未被拦截，以及管理员／会员边界。最终测试实例保留混合设置仅作为持久化证据，不是产品默认值。

## 3. 访问与部署边界

- **部署网络边界已确认（2026-09-17）**：服务器访问规则禁止外部用户直连业务服务端口，外部用户请求统一经过 API Server，见[服务开发规范](../../developer-guide/service-development-standards.md#deployment-network-boundary)。本轮没有远端部署、外网端口扫描或服务器规则实测。
- 会员、管理员和原本允许的 API Key 业务调用都不绕过板块开关；管理员管理开关本身始终可用。开放访问不授予账户模块权限或业务资格。
- 登录、账户、Wallet、Notification、Service Status、Etherscan 管理和模块设置属于核心入口。健康状态、账户权限与板块访问是三个独立维度。
- 后台扫描、采集、计算、恢复和通知继续运行，关闭访问不会减少供应商调用或费用。实际验收在 Solana 设置保持 CLOSED 时观察到成功检查点推进，Notification poller 同时持续前进。
- Market Radar、Managed OO、Profit Sharing 的内部业务端口依赖部署网络隔离；本轮没有新增其服务侧内部鉴权。Solana、Worm Trading 既有内部 Bearer、可信账户身份和账户权限重读保持不变。

该边界落实 SDS-R2、R6、R7 的统一外部准入和失败分类；它不声称本轮完成所有服务的内部零信任改造。

## 4. 用户体验与交易边界

页面将关闭解释为“该板块暂未开放”，不误报登录失效或服务停机。正常联网时最多 5 秒隐藏正文；关闭产生新的访问代次，中止旧请求并阻止晚响应回填。重新开放后按原身份和权限重新读取，不恢复旧浏览器任务。

Worm Trading 的目录、组合、Preview、Run、Cash Out、历史及二次验证统一受 `worm` 控制。已经获得准入并由后台推进的工作继续；仍需浏览器发起或授权的下一步必须等重新开放。重新开放不自动补发交易、恢复驱动或重放未知修改。

## 5. 范围外与验证依据

本期不新增 Runtime Control、成员注册、心跳租约、运行代次、后台暂停、任务取消、通知暂停、跨组启停或停止生效确认。BSC、Sports、Worm Markets 不恢复；Token 的目标服务、访问键和 UI 接入等其重构完成后另行设计。

最终实现以[修订技术方案](../../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)和[验收记录](../../testing/full-stack-access-acceptance.md)为准；2026-09-15 的[原接口设计](../../superpowers/specs/2026-09-15-business-access-control-design.md)保留批准时点事实，但过时的 RPC 数量、页面状态和“尚未实施”描述不再作为现行依据。真实验收覆盖根路径 94 次和前缀 92 次 API 请求、双 realm Chrome smoke、实际 GUI 关闭／开放闭环、重启持久化及后台连续性；未验证事项按验收记录保留。
