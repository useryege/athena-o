# 钱包数据展示：Nansen 接入

> 设计状态：后端接入草案已形成，页面设计 v1 已于 2026-09-14 获用户确认，[实施计划](../../superpowers/plans/2026-09-14-token-wallet-analytics.md)已整理。API Key 和供应商请求已验证；ATHENA 业务接口、缓存表和正式页面尚未实现。

有效业务范围见[钱包交易与战绩需求](../../requirements/token/wallet-trading-performance.md)。具体接口、缓存、排序、权限和验证方案集中维护在[后端设计草案](../../superpowers/specs/2026-09-14-token-wallet-analytics-backend-design.md)，本页记录长期边界与当前差距。

实施计划包含 10 项任务，先完成后端映射、持久化、供应商访问、排序分页、缓存协调和公开接口及验证，再实现页面请求层、已确认 v1 与整体验收。计划补齐了按 generation 保存请求完成结果、同次刷新去重、逐币反向排序的 snapshot 引用、实际环境变量白名单及钱包专用浏览器验收入口；这些仍是待实施工程安排。

## 目标边界

- 在 API Server 中组合独立的 Nansen 查询适配模块，负责请求、字段转换、结果缓存、排序分页和调用记录；没有独立采集或分析后台任务。
- 数据来自 Nansen，采用供应商金额与统计含义；不实现本地交易解析、成本账本、基础币筛选、转账分类、多跳还原或盈亏重算。胜率暂缓。
- 三个数据区域独立取得与失败；Token 查看权限在缓存命中时同样有效。Key 仅由服务端环境加载，用户共用额度和相同条件的结果。
- 使用现有 API PostgreSQL pool 保存可重建结果与调用记录；API 是资源 owner，adapter 显式借用、只操作自己的表。每份成功结果前 10 分钟有效、最多保留 24 小时；读取和失败不续期。
- 新查询不依赖旧 `athena-token-api` 的链配置、EVM registry 或项目扫描。原项目发现与研究的服务不因此改变。

## 当前源码与差距

[API Server](../../../internal/server/athena-server.go)已提供账户上下文、权限和服务注册；[权限表](../../../internal/server/authz.go)需要新增方法登记。[旧 Token API 启动](../../../cmd/athena-token-api/commands/athena-token-api.go)仍包含链配置和 EVM registry；[会员菜单](../../../ui/src/app/member/app.tsx)仍是禁用的 Token 入口。

草案中的 `internal/tokenwalletanalytics/`、公开 RPC、缓存表和环境变量注入均为目标实现。逐币两组返回的合并、全局排序及稳定分页已写明设计，但尚未完成真实大集合验证；供应商原始字段与已核对限制见[样本报告](../../requirements/token/nansen-api-validation-2026-09-13.md)。

## 页面设计衔接

[页面提案 v1](../../requirements/token/wallet-analytics-page-proposal.md)使用保存的 Nansen 原始样本组织三部分展示，并提供桌面／手机预览、合约筛选、排序、已保存交易分页及模拟状态。用户于 2026-09-14 反馈“这一版本很好，通过”，本版作为正式页面的设计依据，具体范围见[确认记录](../../requirements/token/previews/wallet-analytics-v1-approval.json)。沿用全站已确认的 Nansen 深色主题；预览不发起 API 请求，也不构成真实缓存、权限或额度验证。

## 依赖与验证

服务边界按[服务开发规范](../../developer-guide/service-development-standards.md) SDS-R1／R2 的协议适配与 API housekeeping 条款处理；共享 pool、配置隔离和停止归属按 R3–R7 记录在草案中。本阶段仅验证文档、源码对应和保存样本；R8 所需的实现、生成、数据库及真实页面证据将在实现后补充。

后续沿用“后端设计 → 页面设计 → 后端实现 → 页面实现与验收”的既定顺序。实现完成后，本页应转为实际源码与验证说明，不把本草案状态写成已交付。
