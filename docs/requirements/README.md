# 需求目标设计

`docs/requirements/` 保存 ATHENA 长期维护的业务需求与目标行为。这里回答“为什么做、为谁做、系统应表现成什么样”，不提前规定组件拆分、数据库表、RPC 字段或其他实现方案。

新任务遵循原版 Superpowers，按其 `brainstorming`、`writing-plans` 等技能开展工作，入口见 [Superpowers 开发工作流](../developer-guide/superpowers-development.md)。复杂任务的规格与实现计划默认放在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/`；本目录承接其中需要长期维护的业务知识，不另外设置需求、技术设计和实现的审批流程。

聊天记录不是跨任务事实来源。会影响后续工作的业务决定应同步到对应长期文档，并按需链接任务规格。明确区分已经决定的内容、仍在讨论的问题和当前实现，不能因工作流切换而将未决业务标记为已确认。

## 状态

已有需求状态继续描述内容的确定程度，维护时按事实更新；它们不是启动设计或实现的审批开关，也不替代 Superpowers 的工作流程。

| 状态 | 含义 |
| --- | --- |
| `讨论中` | 目标、范围、业务规则或验收行为仍有待决定的内容 |
| `已确认` | 文档记录的业务决定已经获得确认；不表示相关代码已经实现 |

业务规则、权限、状态、范围或可观察行为变化时，更新相应内容及待决问题；如果原状态已不能准确反映确定程度，同时更新状态。措辞和排版调整不改变业务决定。

## 内容范围

按能力需要记录：

- 背景、目标和非目标；
- 使用者、角色与权限边界；
- 业务术语、规则和必须保持的不变量；
- 主流程、关键状态变化、异常与边界场景；
- 对外可见的输入、输出和成功标准；
- 已确认决定与待确认问题，以及未决问题的影响。

需求文档可以描述现状问题和外部约束；后端组件、数据模型、事务、并发或基础设施等长期技术知识写入 [`docs/design/`](../design/README.md)，任务中的方案讨论保存在对应 Superpowers 规格中。

## 组织与维护

- 按稳定业务域和能力组织文档，不按日期、聊天、任务、Issue、PR 或版本建立一次性文件。
- 一个大型能力可以按独立业务主题拆成多份需求；各文档按实际情况维护状态。
- 新文档从[需求模板](template.md)开始，并在本索引或所属业务域索引中登记。
- 已有对应需求和技术设计时互相链接；相关 Superpowers 规格与计划按需链接。没有独立技术设计时如实说明，不为满足流程而补造文档。
- 不删除仍然有效的业务规则；发生实质变化时直接更新目标内容，不把聊天或提交历史复制进正文。

## 文档地图

| 业务域 | 入口 | 状态 | 关联技术设计 |
| --- | --- | --- | --- |
| Trader Sync | [产品需求](polymarket-copy-trading/README.md) | `已确认`（Activity Alerts 业务及完整 UI 设计已确认） | [后端设计](../design/trading/trader-sync-activity-alerts.md)（已实现）；[UI 设计](../design/web-ui/trader-sync-activity-alerts.md)（已实现）；[验收记录](../testing/trader-sync-activity-alerts-acceptance.md) |
| Token Intelligence | [Token 业务设计与研究资料](token/README.md) | `讨论中`（其中部分独立需求已确认） | [Token Intelligence 当前及目标技术设计](../design/README.md) |
| Solana Intelligence | [Solana 项目发现与研究](solana/README.md) | 首版发现与列表已实现并验收；研究规则延后 | [项目发现](../design/solana-intelligence/project-discovery.md)、[列表页](../design/web-ui/solana-discovery.md)、[后续研究](../design/solana-intelligence/research-lifecycle-and-refresh.md) |
