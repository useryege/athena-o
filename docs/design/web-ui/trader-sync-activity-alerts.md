# Trader Sync：Activity Alerts 界面与交互设计

> 设计状态：已确认，正在按联合计划实施；实际进度与验证范围见文末。
>
> 关联：[业务需求](../../requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)、[后端设计](../trading/trader-sync-activity-alerts.md)、[完整 UI spec](../../superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)。后端设计此前已整体确认；本轮补充页面与读取契约，不重开已确认业务决定。

## 范围与现状

会员通过 Trader Sync 订阅人工挑选的低频目标，阅读逐条成交，管理备注和生命周期，理解监控与 Telegram 结果。桌面优先、手机完整可用。管理员只读订阅概要和运行状态。页面不执行交易，不提供重发、历史补查、收益持续刷新或逐订阅通知开关。

当前没有 Trader Sync 业务页面。现有 [member app](../../../ui/src/app/member/app.tsx)、[member routes](../../../ui/src/app/member/routes.tsx)、[member services](../../../ui/src/app/member/services.ts)分别注册导航、懒加载和类型化 service；[admin app](../../../ui/src/app/admin/app.tsx)及其 routes/services 为独立边界。新页面沿用[会员应用壳](member-application-shell.md)、[管理员应用壳](administrator-application-shell.md)和[共享样式](../../../ui/src/app/styles/shared.css)，不是整站改版。

[Notifications](../../../ui/src/app/member/pages/notifications.tsx)已有 Telegram 自助绑定与可见页 3 秒轮询；[Service Status](../../../ui/src/app/admin/pages/service-status.tsx)已有管理员运行入口和 10 秒轮询。新设计复用两者，不重复实现绑定状态机或新建运维首页。

## 关键决定

| 决定 | 理由及边界 |
| --- | --- |
| 桌面活动主区与右侧目标区同屏 | 对照跨目标活动与监控状态；手机折叠目标区，收起仍显示所选目标和异常数量。 |
| 独立添加页 | 身份、完整钱包、统计、六区间 P/L 与备注有充分阅读空间；返回恢复原筛选。 |
| 完整订阅列表和详情负责管理 | 主页侧栏只筛选/显示状态；暂停/恢复直接执行，取消确认一次。 |
| 独立活动详情与摘要批次页 | Telegram 与站内共用详情路径；事实、投递结果和原始证据分层。 |
| 状态每 5 秒刷新，新活动提示后点击载入 | 阅读、选中文本、焦点及滚动位置稳定；后台采集不依赖浏览器。 |
| 管理员概要独立，健康并入 Service Status | 沿用现有应用边界，互相链接；不复用会员正文查询。 |

比较过先选择目标的主页、分标签主页、添加宽抽屉及自动插入新活动；以上选择已逐节确认。具体流程、英文文案语义与验收以[UI spec](../../superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)为本轮完整基线。

## 页面和路由

以下都是各自应用内路径，部署前缀由现有 base path 处理，不能硬编码根路径。

| 应用 | 页面 | 路径与主要内容 |
| --- | --- | --- |
| member | 主页 | `/trader-sync`；跨目标活动、结算时间范围、当前订阅筛选、配额、监控和 Telegram 分别状态。 |
| member | 添加 | `/trader-sync/add`；输入、解析确认卡、六区间 P/L 默认 1Y、私有备注、明确创建。 |
| member | 列表/详情 | `/trader-sync/subscriptions`、`/:subscriptionId`；Current/Cancelled、六态管理、分页观察历史。 |
| member | 活动详情 | `/trader-sync/activities/:activityId`；成交、Combo、资料缺失、通知结果、证据。 |
| member | 摘要详情 | `/trader-sync/summaries/:batchId`；批次统计、所有分条、关联活动分别分页。 |
| member | 绑定 | 既有 `/notifications`；同会话返回保留添加草稿，token 过期仍须重新解析。 |
| admin | 概要 | `/trader-sync/subscriptions`、`/:subscriptionId`；用户/钱包/状态/健康/数量。 |
| admin | 健康 | 既有 `/service-status`；新增连接、积压、缺失及结果计数。 |

member Markets、admin System 增加 Trader Sync 导航。无当前 grant 不进入 member 页面；admin 身份不能读取 member 资源。Telegram 深链接经既有 [login-navigation](../../../ui/src/app/shared/login-navigation.ts)恢复站内 returnTo，保留有效查询参数和部署子路径，拒绝外部返回 URL。

## 流程与状态

### 添加与草稿

输入合法地址或受限 Profile URL → Resolve → 身份确认卡与六区间资料 → 备注 → Confirm subscription → 原主页定位 Preparing monitoring → 后端成功基线后 Monitoring。身份不能核准时禁止创建；辅助字段及曲线各自 unavailable，不造零。提示选择低频目标，不额外要求勾选。

Resolve 补充 owner 保存备注及 revision、现有未取消订阅和配额快照；Create 事务最终判定。备注按 20 个 Unicode code point；省略表示沿用、空串表示清空。输入改变废弃旧卡/token，另一个钱包不能误用上一钱包草稿。5 分钟 token 到期后重新解析，同目标草稿保留。

创建响应丢失重用 request_id 与原 payload。服务端先检查当前权限，再返回已提交幂等成功，然后才对未成功请求判断 token；不因 token 过期制造重复创建。进入 Notifications 再返回保留当前 owner 会话内草稿，换账号/退出/撤权清空，不向 URL 或持久浏览器存储写备注/token。

### 订阅与历史

Current 包含 pending_baseline、healthy、paused、interrupted、permission_disabled，均占 10 个名额；Cancelled 不占，不能恢复，重订阅新 ID。主页 All 活动包含取消历史，侧栏目标按 subscription_id 筛选，不合并同钱包各次订阅。

healthy/interrupted 可以暂停；paused/permission_disabled 可以手动恢复；所有未取消状态可取消，所有详情可编辑 owner-wallet 备注。生命周期写操作使用 revision，备注使用独立 noteRevision；较早响应不得覆盖已读到的较新状态或备注，正常观察概要仍可刷新。冲突读取最新值并保留本地备注草稿，不自动重放旧意图。暂停/取消明确旧队列仍可能送达；取消一次确认含身份、不可恢复、释放名额和保留历史。

观察时间线分页读取成功区间与中断，保留原因/未知边界/可能遗漏，不推测遗漏数量。自动恢复不删除中断，手动恢复新基线。当前备注和活动备注快照分开，编辑不追溯历史。

最近可靠观察来自当前generation持久化的区间健康检查点，不是最近成交或读取时刻，首个有效检查点前可缺。当前状态时间只取对应真实停止记录，不拼过去操作。物理中断起始未知保留Unknown，服务记录的区间停止边界单独解释；恢复仅表示同代后继成功区间重新生效。新代手动恢复另列，旧中断未记录同代恢复不表示当前仍故障。准备失败只展示准备原因，不冒充已监控后中断；次数是相关持久中断记录数，钟差不明不推算持续时长。

订阅保存成功创建所用确认卡的公开展示快照 targetDisplay；活动形成时复制 targetDisplaySnapshot。首行优先非空备注、其次可用名称、最后钱包识别，名称缺失字段仍 unavailable，确认/详情/通知提供完整钱包。名称、头像和可靠 Profile URL 保留原查询证据，按确认时资料展示；暂停/恢复不重新查询，重新订阅可取得新值，不追溯历史。没有已核验 canonical URL 时不生成 Profile 链接。

目标首行优先级只决定主标识，不取消已批准的公开名称展示：有备注时可用公开名称仍作为次级身份信息保留（活动行和详情均可读），不把DisplayName自动加@冒充已核准handle；钱包完整性要求不变。

### 成交及通知

活动详情先展示市场/组合、Outcome、BUY/SELL、目标、结算时间、成交额、份额及费用，再显示 Telegram，来源/时间展开阅读。数值以原始字符串及 decimals 精确处理，币种依证据，当前三 Exchange 为 pUSD。成交价仅本条 value÷shares，费用不入均价；舍入加近似标记，复制保留精确输入，分母不可用不计算。

Combo YES 为所有腿条件满足，NO 为整体合取的补集，不逐腿取反；仍是一笔成交。未知腿数不同于 0，已知 N/缺 M 才显示计数。市场链接必须可靠，缺资料保留真实 ID，晚补资料不生成/补发通知。

活动 notification_mode 固定 in_app_only/ordinary/summary；summary 区分 waiting/frozen/cancelled_before_freeze。普通及分条各有 pending/sending/sent/failed/unknown/cancelled。sending 只证明许可；缺 started_at 不能推断尚未调用，sent 缺起点仍是成功回执、相关时延不可判定。Telegram 接收不等于用户已读。尚无许可或结果时，相应时间与 messageId 可缺，不用零时间补齐。

原始证据区读取 sourceLocation 的 chainId、exchangeAddress、transactionHash、blockHash、blockNumber、logIndex，全部从原 source 保存值映射，不逐条补 RPC。价格及首次公开时间用独立 FieldEvidence；本期首次公开时刻不可观测，不以其他时间代填。活动存在 finalityAnomaly 时单列“Finality anomaly”标记，详情显示原因、检测时刻、原发布块和可缺冲突块；仅 removed 而替代块未知时明确缺失，不填零哈希或原块冒充。原成交、备注和通知结果保留，随原页刷新更新，不新增重发/删除入口。后续真实接收并确认同交易的不同区块时，可在同交易证据锁下仅补首次未知的冲突哈希；保留首次原因和检测时间，已有冲突哈希不覆盖，不恢复被隔离活动或生成新通知。

一个活动涉及的所有 parts 均可分页读取，并与整批统计分开。仅全部 sent 才称整批成功；每种其他状态保留计数。首期只给当前/最近 attempt 证据与总次数，后端保留完整 attempts，不新增尝试历史浏览器或管理员逐条读取。

## 读取、刷新和权限契约

活动主页及 member 详情可见时每 5 秒 single-flight；隐藏停止，恢复立即刷新。新活动提示后点击才替换列表；状态/资料原位更新。返回详情前列表时恢复筛选、游标和位置。失败保留最后数据及更新时间；首次加载失败不伪装空态。

为避免时间回退和新插入扰动阅读，本轮将原 `(recorded_at,id)` 活动分页草案修订为形成顺序 `id DESC`：bigint ID 在 owner account gate 内由持久 sequence（CACHE 1、正向、NO CYCLE）分配，不预取/回拨。读取在同 gate 取该 owner 已提交 max(id)，初始空为 0。它保证 owner 内形成顺序，非全局提交顺序；recorded_at 保留真实时钟供事实/指标使用，不人为改值。依据及限制见[UI spec 第 10 节](../../superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md#活动列表稳定读取协议)。

首次读取产生 snapshot，next_cursor 保留 snapshot；refresh_cursor 绑定本页固定成员、身份、筛选及签名。一条查询刷新本页所有行的资料/结果，并判断 snapshot 后符合条件的新活动 has_newer。用户点击后不带旧 cursor 取新 snapshot；筛选改变重建，非法或跨用户 cursor 拒绝。固定批次成员不参与新活动提示。默认 50、最多 100，前端使用 Previous/Next 与已访问页栈，不编造总数和页数。

新增 `ListSubscriptionHistory`、`GetSummaryBatch`、`ListSummaryParts`，并为 ListActivities 加 summary_batch_id、refresh_cursor、snapshot/has_newer/as_of 语义。其余补全包括 Resolve 草稿快照、显式通知模式、当前结果概要及管理员计数口径。嵌套 intervals、interruptions、parts、attempts 不无界内嵌，完整 API 边界见 UI spec 第 10 节。

服务端以权威 grant/owner 授权，所有会员读取与撤权同 gate。前端得知失权后 abort、提高 generation、清空数据/草稿；晚到旧响应不能恢复正文。无推送时发现撤权可等待下一次轮询，后台撤权提交后即拒绝新访问。admin 专用 SQL/DTO 从源头排除私有正文。

## 管理员概要与指标口径

订阅概要支持账户、规范钱包、状态和包含取消过滤，默认当前未取消。只能看用户身份、钱包、状态/时间、健康及数量，不提供订阅操作、备注、市场活动正文、消息或逐条投递。Service Status 提供连接新鲜度、目标/关系、raw/确认/投影积压、资料缺失、通知状态及异常，并互链。

所有概要携带 as_of。订阅活动数为该订阅全生命周期；Associated deliveries 为关联的 distinct 逻辑投递（普通消息/摘要部分），不是 attempt。一部分涉及多个订阅会在各行各计一次，行间不可相加；全局单独 distinct delivery 聚合。积压标明单位，当前 gauge、时间窗口和 service_epoch 累计分别说明，不混用。

运行指标的 kind 只取 gauge/window/epoch；epoch 项必须携带 serviceEpoch。Projector 累计项的 serviceEpoch 标识该计数实例的生命周期，同实例跨 Collector 重连保持稳定，新实例或计数重置时改变。它是 opaque string，不当作 Collector 连接序号解析，也不跨不同指标生产者合并累计值。

发送进程正在初始化或等待恢复预算时显示 Recovering，单列原因、该进程报告的剩余/已用等待及更新时间；fatal/stopped优先，恢复未完成不显示Running。recovery对象可缺，remainingMillis/elapsedMillis为十进制string，未知剩余保持Unavailable，合法0保留；浏览器不重新推算或解除恢复预算，也不把恢复等待归为Telegram网络耗时。raw队列与持久化中数量只在raw_observation_available可观测时展示，不可观测时不填0；按原10秒单飞读取更新，不另加计时或逐条请求。

## 组件与源码落点

| 预计位置或现有入口 | 职责 |
| --- | --- |
| 新增 `ui/src/app/member/pages/trader-sync/` | 六类页面与模块专属组件/hooks；页面负责流程，不解析链协议。 |
| 新增 `ui/src/app/member/trader-sync-service.ts` | 类型化会员读取/写入、精确字段及错误映射。 |
| 新增 `ui/src/app/admin/pages/trader-sync/`、`ui/src/app/admin/trader-sync-service.ts` | 安全概要及运行 DTO，与会员 service 隔离。 |
| member/admin app、routes、services | 入口、懒加载、权限及组合；路径与现有同层 service 习惯一致。 |
| [ResourceTable](../../../ui/src/app/components/resource-table.tsx) | 复用表格与 compactRender，省略 total/数字分页，外置游标控制器。 |
| [format](../../../ui/src/app/shared/format.ts)、[login-navigation](../../../ui/src/app/shared/login-navigation.ts) | UTC+8 明示、站内返回与部署前缀。 |
| Notifications、Service Status | 返回草稿及边界文案 / 运行概要区域。 |
| 后端 application types 与生成消费链 | 新增读取和 DTO 同步到 proto/gateway/apiclient/Swagger/UI；不手改生成代码。 |

复用 AppPage、Section、KeyValueGrid 与确认弹窗。Profile 的 P/L、Positions Value 和 Biggest Win 沿用供应商 `$` 展示符号，并说明未提供币种代码；不推定 USD/pUSD/USDC，Biggest Win 同符号是共用 formatter 的推断，已知零值保留（见[展示符号核验](../../requirements/polymarket-copy-trading/profile-pnl-contract-verification.md#2026-09-11-补充金额展示符号与币种边界)）。P/L 为创建前快照，使用独立 SVG/文字摘要组件，不为单图引入大型框架；绘图有限归一化与精确业务字符串分开。界面沿用英文、Heebo/系统字体、橙色主题及深浅模式。

## 验证与维护

实现验收覆盖 1440/1280、900 附近、390，键盘/触屏、焦点、AA 对比度和长内容；添加的六区间/缺失/过期/幂等，六态和观察历史，Combo/精度/所有消息结果，多部分与会员/admin 隔离，刷新/游标/晚响应撤权，Telegram 登录返回和子路径。完整矩阵见 UI spec 第 12 节。

设计阶段仅验证合成数据线框的桌面/手机、主题与交互；线框不包含业务 API、真实 Telegram 或完整可访问性验收。临时浏览器内容在 `.superpowers/`，不是长期依赖；被选结构与流程已写入文档。实现阶段的后端证据和限制见[验收记录](../../testing/trader-sync-activity-alerts-acceptance.md)，前端进度如下。

本书面 UI spec 已整体确认，[21项前后端联合实现计划](../../superpowers/plans/2026-09-10-trader-sync-activity-alerts.md)正在独立工作区逐项实施。共享权限矩阵及管理员授权行已实现十模块；Trader Sync 选择器仅提供 No access / Read & write，解析、编辑、克隆和提交对非法 READ 统一关闭为 NONE。会员类型化 service、可取消读取与精度基础，以及独立添加页、确认卡和 Notifications 返回草稿已通过对应任务审查。会员壳在退出、身份或 issuer 变化及 RW→NONE 时清空 Trader Sync 内存状态并隔离晚响应。订阅列表、详情、生命周期操作、独立备注版本与观察历史已通过任务审查；延迟响应保留已知较新事实，固定队列提示使用英文并保留用户 Unicode 备注。活动主页与目标侧栏已通过任务审查；可见时分别单飞轮询，活动刷新保留快照和有序成员，新活动由用户点击载入，分页和详情返回保留会话。Previous 在恢复页 DOM 提交后还原滚动位置，选中文本期间延后该行资料替换。活动和摘要详情、管理员概要及 Service Status 衔接仍由后续任务实施。真实浏览器、移动、键盘及完整前后端验收尚未完成，组件测试不替代这些结论。
