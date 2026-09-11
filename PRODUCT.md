# ATHENA 产品上下文

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

ATHENA 有独立的普通会员与管理员应用。会员使用被授予权限的业务模块；管理员管理账户授权及各模块允许的运行概要。两种身份的页面、会话和业务数据边界分别维护。

Trader Sync 的第一阶段能力是 Activity Alerts。用户人工选择低频 Polymarket 交易者，及时查看目标正在交易的市场、Outcome、方向和公开事实，再自行判断是否前往市场操作。管理员在本功能中只查看安全概要，不查看用户私有备注、完整活动或逐条消息。

## Product Purpose

ATHENA 是面向区块链与预测市场的情报分析平台，通过 Web UI、API 和通知服务提供市场与链上数据。本文件为界面工作提供上下文，业务细则仍以相关长期需求和获批 spec 为准。

Trader Sync 第一阶段提供目标确认、独立订阅、实时成交活动和 Telegram 私聊提醒。未来 Copy Trading 的具体业务另行讨论，Activity Alerts 不执行交易、签名或钱包操作。

## Operating Context

- 用户于本轮 UI 设计中确认：Trader Sync 桌面优先，手机完整可用。手机须覆盖从 Telegram 进入详情、查看通知状态与管理订阅的流程。这不是对其他模块新增移动端改造要求。
- 已确认的主页以跨目标活动为主，桌面同时展示目标与监控状态，手机折叠目标栏。用户通过独立添加页解析和审阅目标，再明确确认订阅；备注、暂停、恢复、取消与中断记录集中在完整订阅列表和目标详情页管理。
- 活动使用独立详情，与 Telegram 深链接共用；摘要批次有独立页面展示全部分条。可见页每 5 秒更新状态，新活动提示后点击载入，保持阅读位置。管理员订阅概要独立，运行健康并入现有 Service Status。
- 首期按 10 名用户、每人最多 10 个未取消订阅设计，覆盖最多 100 个不同目标。后台监控不依赖浏览器持续打开。
- Telegram 绑定已有独立会员自助入口；管理员使用独立管理应用。新增界面应与这些实际入口衔接。

## Capabilities and Constraints

- 每位用户拥有独立订阅、备注、活动和通知资格；私有备注最多 20 个 Unicode code point，历史保留形成时快照。
- 地址或 Profile URL 均先解析并展示确认卡。身份不能核准时禁止创建；辅助资料缺失明确标为不可用，仍可确认。确认卡包含六个已定义 P/L 区间，默认 1Y；创建后不持续刷新收益。
- 基线建立后立即生效，按链上结算时间划界。断线、重启及故障期间的遗漏不补查，已有中断说明保留。
- 每条合格成交形成独立持久活动；用户级滚动 60 秒内前 10 条逐条提醒，第 11 条起按已确认规则汇总。消息未知结果不自动重发，摘要分条分别显示结果。
- 暂停/取消保留旧通知队列；撤权、解绑或重绑终止未取得发送许可的旧资格。重新获权后用户逐个恢复订阅。
- 显示结算时间、监控健康、资料缺失和通知结果，不把“无新活动”“监控失效”“投递未知”混为同一状态。
- 后端、会员与管理员页面、Notifications 返回态和 Service Status 集成已经实现。受控浏览器覆盖根路径与 `/athena`、桌面/手机、深浅主题、权限撤销、owner 隔离及关键管理流程；资料查询与运行结论以实际[验收记录](docs/testing/trader-sync-activity-alerts-acceptance.md)为准。
- 现有证据不等于生产 SLO：协议样本包含 11 条 recorded 与 1 条明确 synthetic，100 个真正活跃实网目标、公开时刻 P95/P99、供应商静默漏推完整性和长期稳定仍需外部环境验证。

## Brand Commitments

产品名称为 ATHENA；本模块名为 Trader Sync，第一阶段能力名为 Activity Alerts。现有界面的主题、组件和语言事实以实际 React/CSS 及界面设计文档为依据；本任务未授权整站换品牌或重做其他模块。

## Evidence on Hand

- [平台说明](README.md)
- [业务需求](docs/requirements/polymarket-copy-trading/target-trade-monitoring-notifications.md)
- [已确认后端 spec](docs/superpowers/specs/2026-09-10-trader-sync-activity-alerts-design.md)
- [已整体确认 UI spec](docs/superpowers/specs/2026-09-10-trader-sync-activity-alerts-ui-design.md)
- [长期 UI 设计](docs/design/web-ui/trader-sync-activity-alerts.md)
- [前后端联合实现计划](docs/superpowers/plans/2026-09-10-trader-sync-activity-alerts.md)
- [最终验收记录](docs/testing/trader-sync-activity-alerts-acceptance.md)
- [会员应用壳](docs/design/web-ui/member-application-shell.md)、[管理员应用壳](docs/design/web-ui/administrator-application-shell.md)
- [共享样式实现](ui/src/app/styles/shared.css)、[会员入口](ui/src/app/member/app.tsx)、[管理员入口](ui/src/app/admin/app.tsx)

## Product Principles

- 优先让用户理解目标实际交易了什么，以及可前往哪个市场查看。
- 数据缺失、监控中断和消息未知必须如实表达，不伪造完整性或成功。
- 用户隔离、管理员最小可见范围和既有授权决定落实到实际界面与接口。
- 页面设计与后端契约共同形成可验收的用户流程，已有技术设计可根据有依据的接口缺口修订。
