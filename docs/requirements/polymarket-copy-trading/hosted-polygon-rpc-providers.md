# Trader Sync 托管 Polygon RPC 服务与费用调研

> 调研日期：2026-09-09。
>
> 用户决定：当前暂不自建 Polygon PoS 节点，先调研第三方托管 RPC。
>
> 性质：供应商公开资料与预算比较；尚未选定供应商、套餐或采集架构，不构成已确认技术设计或采购授权。
>
> 关联资料：[Activity Alerts 需求](target-trade-monitoring-notifications.md)、[链上成交可行性与实际样本](onchain-trade-data-feasibility.md)。

## 结论

Alchemy、QuickNode、Chainstack、Infura、dRPC、Chainnodes 都提供 Polygon PoS RPC，可作为 HTTP 日志查询与 WebSocket 订阅的候选。当前低频目标活动监控可以从共享 RPC 服务开始评估，现有需求没有体现必须购买独享节点或固定高吞吐产品的理由。

付费候选推荐优先比较 **Chainstack 与 dRPC**：前者提供免费额度和约 49 美元的固定月费方案，后者按量计费简单，常用调用折合每百万次 6 美元。Chainnodes 的免费请求额度也适合开发阶段，但官方明确免费资源在繁忙时可能降优先级。此建议依据费用、计费方式和公开限制，不代表已证明任何一家在实际部署区域更快、更稳定。

## 开发阶段免费方案建议（2026-09-10 复核）

针对用户提出的开发阶段免费使用需求，建议首选 **Chainnodes Core**，将 **Chainstack Developer** 保留为第二选择，**dRPC Free** 作为临时补充查询候选。本节是推荐，用户尚未确认具体供应商，也没有形成自动故障切换设计。

- **Chainnodes Core**：每月 12.5M 次请求，标称 25 RPS，支持 HTTP/WSS，单次日志查询区块跨度上限 20,000。免费额度与较宽的补查范围适合开发期间频繁启停后的恢复；仍受响应大小等限制，免费资源繁忙时可能提前限流。WSS 每条推送与 HTTP 共用请求额度和 RPS。[官方限制](https://www.chainnodes.org/docs/FAQs/rate_limits)
- **Chainstack Developer**：每月 3M RU、25 RPS，单次 `eth_getLogs` 最多 100 块，适合近期小范围查询；停机较久后的补查需要更多分段。要保持零 RPC 费用，应确认免费套餐的 extra usage 已关闭；额度耗尽后服务会停止，不应假设默认配置绝不会收费。[价格](https://chainstack.com/pricing/)、[查询限制](https://docs.chainstack.com/docs/limits)、[账单设置](https://docs.chainstack.com/docs/manage-your-billing#manage-the-extra-usage-setting)
- **dRPC Free**：文档列 210M CU/30 天，但依赖公共节点，免费查询超时 2 秒、最多 10,000 条日志，并可能动态限流，因此暂不作为优先开发入口。免费账户层的额度不能未经核实套到任意匿名公共 RPC URL。[免费层限制](https://drpc.org/docs/howitworks/ratelimiting)

沿用下文每 5 秒查询共享高度加 5 组日志的预算，30 天约 3.11M 次基础调用，占 Chainnodes 免费额度约四分之一，尚未计 WSS 推送、补查等额外用量。推荐依据是免费额度和开发便利性，尚无实际部署区域的连接质量排名。

## 价格口径与套餐

以下金额均为美元，按公开标准价格，不含税费、服务器、数据库、应用网络流量或其他 API 成本。`M` 表示百万。不同供应商的 CU、credits、RU 不可直接比较；它们不是相同单位，也不等于同样数量的 RPC 调用。

| 服务商 | 免费或试用 | 入门付费方案（月付） | 年付与超额 | 官方来源 |
| --- | --- | --- | --- | --- |
| dRPC | 210M CU / 30 天，使用公共节点；按 20 CU/次约 10.5M 次 | 预充值、按量扣费；0.30 美元/M CU，即常用方法 6 美元/M 次 | 免费与付费节点分层，预算不将免费额度抵扣付费流量；最低充值额未核实 | [套餐](https://drpc.org/docs/pricing/requests)、[方法单价](https://drpc.org/docs/pricing/compute-units) |
| Chainstack | Developer：3M RU/月，25 RPS；允许额外用量计费 | Growth：49 美元/月，20M RU，250 RPS | Growth 年付页面折算约 40 美元/月；Developer 超额 20 美元/M RU，Growth 超额 15 美元/M RU | [价格](https://chainstack.com/pricing/) |
| Chainnodes | Core：12.5M 次/月，25 RPS | Developer：50 美元/月，25M 次，50 RPS | 年付标示优惠 20%；Team 250 美元/月含 125M 次；未确认自动超额单价 | [价格](https://www.chainnodes.org/pricing) |
| Alchemy | Free：30M CU/月 | PAYG：无平台月费；前 300M CU 按 0.45 美元/M，之后按 0.40 美元/M | PAYG 按全部用量收费，不抵扣 Free 的 30M CU | [价格](https://www.alchemy.com/pricing)、[PAYG 计算示例](https://www.alchemy.com/docs/reference/pay-as-you-go-pricing-faq) |
| QuickNode | 1 个月试用，10M credits，15 RPS；不是长期免费套餐 | Build：49 美元/月，80M credits，50 RPS | 当前年付页面折算约 34 美元/月；Build 超额 0.62 美元/M credits | [价格](https://www.quicknode.com/pricing) |
| Infura | 新客户 Core：3M credits/日，500 credits/秒 | Developer：50 美元/月，15M credits/日，4,000 credits/秒 | Team：225 美元/月，75M credits/日；未确认统一自动超额单价和公开年付价 | [价格](https://www.infura.io/pricing)、[当前套餐文档](https://docs.infura.io/get-started/pricing/) |

年付数字是官方页面的月均展示，可能包含促销和取整，不是随时可取消的月付价格。本次未进入结账页核实精确年账单。

Infura 官网同一页面的 FAQ 仍写免费 6M credits/日、2,000 credits/秒，与套餐表冲突；本报告采用当前套餐文档明确给新客户的 3M/日、500/秒。没有用供应商对竞争对手的宣传比较表作为计费依据。

Chainstack Developer 无须升级 Growth 即可启用付费超额。管理员可在账单设置开启 extra usage；关闭时额度耗尽会停止服务。默认开关和绑定付款方式的具体步骤未登录核实。[官方免费套餐说明](https://chainstack.com/best-free-tier-rpcs-web3-builders-2026/)、[账单设置](https://docs.chainstack.com/docs/manage-your-billing#manage-the-extra-usage-setting)

## 日志与订阅的计费差异

| 服务商 | `eth_blockNumber` | `eth_getLogs` | WebSocket 日志推送 | 官方依据 |
| --- | --- | --- | --- | --- |
| dRPC | 20 CU | 20 CU | 建立订阅 20 CU；每条通知 20 CU | [方法费用](https://drpc.org/docs/pricing/compute-units)、[EVM 订阅](https://drpc.org/docs/pricing/subscriptions/evm) |
| Chainstack Global Node | 1 RU | 近期 1 RU；历史 2 RU | 每条 push 计一次请求；近期 Global Node 按 1 RU | [RU 规则](https://docs.chainstack.com/docs/request-units) |
| Chainnodes | 1 次请求 | 1 次请求 | 每条订阅响应计一次请求，同时占用 RPS | [计量与限额](https://www.chainnodes.org/docs/FAQs/rate_limits) |
| Alchemy | 10 CU | 60 CU | 0.04 CU/字节；例如约 1,000 字节为 40 CU；订阅动作另计 10 CU | [方法与推送费用](https://www.alchemy.com/docs/reference/compute-unit-costs) |
| QuickNode | 20 credits | 20 credits | 每条响应计入用量；Polygon 单条推送具体 credits 未确认 | [API credits](https://www.quicknode.com/api-credits)、[WSS 计量说明](https://support.quicknode.com/articles/9223695305-understanding-my-method-call-breakdown-in-metrics-tab) |
| Infura | 80 credits | 255 credits | `logs` 按块计 300 credits；订阅动作 5 credits；空匹配块及多个订阅如何计量未核实 | [方法费用](https://docs.infura.io/get-started/pricing/credit-cost/) |

Chainstack 的历史分界为距 tip 至少 127 块，`getLogs` 根据 `fromBlock` 判断；一次 HTTP 响应有多条日志仍按请求计量。不同节点产品可能使用不同规则，不能把 Global Node 的 1 RU 套到 archive Trader Node 等产品。

HTTP 轮询即使返回空数组也消耗调用额度。WebSocket 不是只在建立连接时收费；只订阅目标日志可能随低频活动减少用量，但订阅全链区块或宽范围事件会增加推送。断线恢复仍需要 HTTP 补查，不能仅用实时推送数量估算完整费用。

## 对本项目有影响的限制

| 服务商 | 日志补查限制 | 订阅及其他重要限制 | 官方依据 |
| --- | --- | --- | --- |
| dRPC | 免费节点最多返回 10,000 条日志，超时 2 秒；付费 Polygon 的统一区块跨度上限未核实 | 免费节点可动态限流；支持 Polygon `logs` 及 `newHeads`；付费连接/订阅数上限未核实 | [限制](https://drpc.org/docs/howitworks/ratelimiting)、[Polygon 日志](https://drpc.org/docs/polygon-api/eventlogs/eth_getLogs)、[订阅](https://drpc.org/docs/polygon-api/subscriptions/eth_subscribe) |
| Chainstack | 免费每次 100 块，Growth 每次 10,000 块；官方建议 Polygon 每次不超过 3,000 块 | 文档列 500 个并发 WSS 连接，但额度归属范围不明确；空闲连接 1 小时超时 | [限制](https://docs.chainstack.com/docs/limits)、[Polygon 建议](https://docs.chainstack.com/reference/getlogs)、[WSS](https://docs.chainstack.com/docs/handle-real-time-data-using-websockets-with-javascript-and-python) |
| Chainnodes | `toBlock-fromBlock ≤ 20,000`；HTTP 响应最大 30MB | Core 最多 25 个 WSS 连接，付费连接数为 RPS 的 100 倍；推送共享请求限额；Core 不保证标称额度与 RPS | [限制](https://www.chainnodes.org/docs/FAQs/rate_limits)、[Polygon 订阅](https://www.chainnodes.org/docs/polygon/eth_subscribe) |
| Alchemy | 当前 Polygon 方法专页：Free 每次 10 块，付费区块跨度不设固定上限，响应最大 150MB | Free 每 app 100 个 WSS 连接，其他套餐 2,000 个；每连接最多 1,000 个订阅 | [Polygon 日志](https://www.alchemy.com/docs/chains/polygon-pos/polygon-po-s-api-endpoints/eth-get-logs)、[订阅限制](https://www.alchemy.com/docs/reference/subscription-api) |
| QuickNode | 试用每次 5 块，付费每次 10,000 块 | 常规 Build 等套餐的 Polygon WSS 连接/订阅上限未核实 | [Polygon 日志](https://www.quicknode.com/docs/polygon/eth_getLogs)、[连接协议](https://www.quicknode.com/docs/polygon/endpoints) |
| Infura | 最多 10,000 条结果、10 秒查询时间；未给出统一固定区块跨度 | 按日计量，UTC 零点重置；日额度耗尽可能停止服务并断开 WSS | [Polygon 日志](https://docs.infura.io/reference/polygon-pos/json-rpc-methods/eth_getlogs/)、[订阅](https://docs.infura.io/reference/polygon-pos/json-rpc-methods/subscription-methods/eth_subscribe/)、[额度行为](https://docs.infura.io/how-to/avoid-rate-limiting/) |

Alchemy 部分旧官方页面仍写付费 Polygon 每次 2,000 块，本报告采用当前 Polygon 方法专页的口径；端点实际限制仍是采购与使用约束。以上单次区块跨度不能被解释成只能查询最近这些区块；历史保留深度、结果数量和单次跨度是不同条件。

上一轮样本已经证明指定目标钱包及四钱包 OR 过滤可行，但没有证明无限大的地址/topics 数组都能被这些供应商接受。目标分组数还受过滤参数、响应体、事件版本和补查分段影响。

## 同一轮询工作量下的预算

为便于比较，采用以下明确假设，不作为正式采集设计：

- 运行 30 天，每 5 秒一轮，共 518,400 轮。
- 每轮共享一次 `eth_blockNumber`，对 `G` 个实际日志查询组各执行一次近期 `eth_getLogs`。
- `G` 是请求分组数，不是钱包数、ATHENA 用户数或订阅条数。多个目标可以 OR 过滤；同一目标被多个用户订阅可以共享采集结果。
- 每轮请求数为 `1+G`，因此月调用数为 `518,400 × (1+G)`。JSON-RPC batch 不自动把其中多个调用合成一次计费。
- 表内不含 WSS、时间/确认状态查询、回执、重试、故障补查、重组重扫和备用供应商流量，也不含 Polymarket 市场资料 API 与应用基础设施。

| 实际日志查询组 G | 月 RPC 调用数 | dRPC 付费 | Chainstack Growth | Chainnodes Developer | Alchemy PAYG | QuickNode Build | Infura 适配套餐 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | 1,036,800 | 6.22 美元 | 49 美元 | 50 美元 | 16.33 美元 | 49 美元 | Developer：50 美元 |
| 5 | 3,110,400 | 18.66 美元 | 49 美元 | 50 美元 | 72.32 美元 | 49 美元 | Team：225 美元 |
| 10 | 5,702,400 | 34.21 美元 | 49 美元 | 50 美元 | 141.49 美元 | 70.11 美元 | Team：225 美元 |

上表是指定套餐下的计算结果，不是每家所有免费、促销或混合计费选项中的最低价：

- Chainnodes Core 的 12.5M 免费请求额度在数量上覆盖全部三个场景，但免费资源不保证标称额度与 RPS。
- dRPC 免费层的 210M CU 在数量上也覆盖三个场景，但使用公共节点，与付费服务的资源和限制不同。
- Chainstack Developer 的免费额度覆盖 G=1；按官网 Developer 超额 20 美元/M RU 计算，G=5、G=10 分别约 2.21、54.05 美元。付费超额不改变 Developer 每次 100 块等套餐限制；G=10 时 Growth 49 美元反而更便宜。
- Alchemy Free 的 30M CU 不足以覆盖 G=1 的 36.288M CU，PAYG 不能先减 30M。G=5、10 分别消耗 160.704M、316.224M CU。
- QuickNode 对应 20.736M、62.208M、114.048M credits；最后一个场景包含 Build 超额费。
- Infura 对应每日 5.7888M、23.4144M、45.4464M credits；后两个场景超过 Developer 的 15M/日。这里选 Team 标准套餐作比较，没有假定额外 credits 包的结算方式。

若仅为 dRPC 的这些基础调用增加 20% 用量预留，结果约为每月 7.46、22.39、41.06 美元。**20% 只是预算假设，不能保证覆盖故障恢复等额外用量**。如果每轮还需要查最终确认高度或其他区块信息，应把对应调用直接纳入基础公式，而不是长期隐藏在预留中。

每 5 秒轮询只是请求调度周期，不等于 5 秒内完成确认或通知。供应商额度、真实端到端时延和产品已有时效目标仍需分别判断。

## 建议与尚未确定的信息

1. **开发阶段可以先使用免费层。** Chainnodes Core、Chainstack Developer、dRPC Free 都有可用额度；重点区别是日志补查限制、共享资源与付费后的变化。
2. **希望较低付费起点时，比较 dRPC 按量和 Chainstack Developer 超额。** 不需要立即承诺年付。费用取决于实际分组和采集方式，不能宣称某一家在所有工作量下最便宜。
3. **希望固定月费、保留较多补查与吞吐余量时，优先考虑 Chainstack Growth 49 美元/月。** Chainnodes Developer 50 美元/月也是计量简单的候选。按量和固定方案的取舍不代表性能排名。
4. **暂不需要独享节点、高吞吐附加包或一次性购买全部候选服务。** 备用供应商是否需要及其用量属于后续技术设计，不是本次已确认投入。

尚未确定实际部署区域的连接质量、全平台唯一目标数量、目标 OR 分组上限、正常与恢复时的调用构成，以及采用轮询还是推送结合补查。因此可先用约 50 美元/月作为入门付费 RPC 的预算量级，但不能将其承诺为任何用户规模下的费用上限。

本次仅查阅官方公开定价和文档、进行费用复算并维护调研记录；未注册账号、购买套餐、联系销售、部署节点或修改后端业务源码，也没有新增或运行测试。
