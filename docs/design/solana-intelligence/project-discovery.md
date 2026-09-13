# Solana 新项目发现方案调研

> 设计状态：设计中；当前为能力核对与方案比较，尚未选定接入方式，尚未实现。
>
> 关联需求：[Solana 项目研究](../../requirements/solana/README.md)。资料核对日期：2026-09-13。

## 范围与现状

本文回答如何从发行阶段发现 Solana 新代币，为独立研究板块建立起点。生命周期采用[持续研究与刷新调度](research-lifecycle-and-refresh.md)：发行后持续监控，活动按每项目一小时限频触发刷新；研究资料与完整服务架构另随需求细化。

现有 [Token 扫描器](../../../internal/token/discovery/application/chain_processor.go) 和 [EVM 候选识别器](../../../internal/token/adapters/evm/candidate_inspector.go) 处理 EVM 合约候选；它们不能直接解释为 Solana 新项目发现能力。已有 [Solana 钱包登录](../identity-access/solana-wallet-authentication.md) 也不承担项目发现。

## 已核对的链上事实

- Solana 的普通 SPL 代币以 Mint 账户标识；Token Program 和 Token Extension Program（Token-2022）是两类主要 Token 程序。Mint、持币账户及其权限具有不同含义，Mint authority 不必等于创建或付费账户。来源：[Solana Tokens](https://solana.com/docs/tokens)。
- Mint 初始化指令包括 `InitializeMint`、`InitializeMint2`。后续对已有 Mint 执行铸币是增加供应量，不能据此判定又发行了一个新项目。来源：[创建 Mint](https://solana.com/docs/tokens/basics/create-mint)、[铸币](https://solana.com/docs/tokens/basics/mint-tokens)。
- 程序内部调用的指令由交易元数据的 `innerInstructions` 表达。发现解析应覆盖顶层及 CPI 内部初始化；只检查顶层可能遗漏平台创建路径。这是根据交易结构作出的设计建议。来源：[RPC JSON 结构](https://solana.com/docs/rpc/json-structures)。

新 Mint 是候选事实。是否属于研究范围内的普通代币，需要独立业务定义；不能仅凭供应量或 decimals 就断定项目类型。

## 方案比较

| 方式 | 能力与收益 | 限制及适用判断 |
| --- | --- | --- |
| 标准节点 RPC：日志订阅与交易解析、区块扫描 | 按 Token 程序发现成功的 Mint 初始化，可覆盖选定程序内的不同发行途径；历史区块可用于补查 | 需自行解析、去重及证明覆盖；订阅两大 Token 程序仍可能有大量无关交易，吞吐和补查成本需实测 |
| 发行平台创建指令或事件 | 对 Pump.fun、Raydium LaunchLab 等平台识别发行来源、创建角色及发行阶段，项目语义更明确 | 覆盖限于已支持的平台与版本；不能承诺覆盖其他平台或直接创建的代币 |
| Yellowstone / Geyser 交易流 | 服务端可按账户筛选交易并传送执行数据，可作为持续监听的候选接入方式 | 需要支持该服务的节点或提供方；过滤权限、资源额度、回放窗口及恢复方式需核验，不能因使用 gRPC 就承诺低延迟或不丢数据 |

前两行中的业务识别可以使用相同的节点数据；第三行属于传输方式，不额外改变项目纳入范围。

节点接口依据为 [logsSubscribe](https://solana.com/docs/rpc/websocket/logssubscribe)、[getTransaction](https://solana.com/docs/rpc/http/gettransaction)、[getBlock](https://solana.com/docs/rpc/http/getblock)。平台依据为 [Pump Program](https://github.com/pump-fun/pump-public-docs/blob/main/docs/PUMP_PROGRAM_README.md)、[LaunchLab](https://docs.raydium.io/user-flows/launchlab-overview)。交易流及过滤依据为 [Yellowstone 官方项目](https://github.com/rpcpool/yellowstone-grpc#filters-for-streamed-data)。上述收益判断是面向本项目的推论，尚无本地性能验证。

## 当前建议，尚未确认

建议以“成功初始化新 Mint”作为最早候选发现事实，解析选定 Token 程序的顶层和内部指令；平台解析补充发行背景。新项目是否正式进入研究，由待确认的资产纳入规则决定。

实时订阅与历史补查共同处理发现。使用 HTTP 区块扫描作为覆盖与恢复依据时，需要保存处理进度并校验数据完整性；不把一次订阅连接成功视为已完整覆盖。实时通道选标准 WebSocket 还是 Yellowstone，由目标时效、节点权限、吞吐及历史能力决定。平台首版清单也未选定。

这沿用“链上发现后研究”的业务思路，具体发行证据按 Solana 定义。尚未选择购买提供方服务或部署自有节点。

## 影响设计的接口边界

- `logsSubscribe` 的 `mentions` 每个订阅只支持一个地址，返回签名、执行错误和日志。日志文本不能单独证明目标 Mint 已成功初始化；应核对程序身份、实际指令及交易执行结果。来源：[logsSubscribe](https://solana.com/docs/rpc/websocket/logssubscribe)。
- `blockSubscribe` 被标记为不稳定接口，节点需开启相应配置；不能假设已有 RPC 支持。来源：[blockSubscribe](https://solana.com/docs/rpc/websocket/blocksubscribe)。
- 补查使用 `getBlocks`、`getBlock` 时，应区分跳过的 slot、暂未取得的数据和节点历史缺口；不能把所有 slot 都当作必有区块。交易版本、地址查找表和内部指令也必须纳入解析验证。来源：[getBlocks](https://solana.com/docs/rpc/http/getblocks)、[跳过 slot 的接口说明](https://solana.com/docs/rpc/deprecated/getconfirmedblocks)、[RPC JSON 结构](https://solana.com/docs/rpc/json-structures)。
- `processed` 数据可能回滚；发现采用何种 commitment、何时作为确定事实以及分叉如何处理，需要另行决定。不能自动复制 EVM 当前“不考虑重组”的规则。来源：[RPC commitment](https://solana.com/docs/rpc)。
- `programSubscribe` 是账户变化通知。将首次见到的账户变化直接解释为新发行，可能混淆存量账户更新与创建；这是基于接口含义的限制判断。来源：[programSubscribe](https://solana.com/docs/rpc/websocket/programsubscribe)。
- Yellowstone 的普通交易流提供执行后的数据；执行前的 deshred 流缺少执行状态及内部指令等证据，不能独立证明发行成功。本文建议不依赖执行前推送。来源：[Yellowstone deshred 说明](https://github.com/rpcpool/yellowstone-grpc#deshred-transactions)。

## 创建与首笔交易可能重叠

Pump 官方仓库中的 SDK 使用文档提供 `createV2AndBuyInstructions`，将创建和初始买入组装到交易中；LaunchLab 产品流程也包含可选首买。来源：[Pump 创建与首买](https://github.com/pump-fun/pump-fun-skills/blob/main/create-coin/SKILL.md)、[LaunchLab 创建模式](https://docs.raydium.io/user-flows/launchlab-overview)。

系统首次取得成功创建交易时，项目可能已经有首笔买入。已确认 Solana 项目持续研究，因此创建交易内或之后出现买入/Swap 都不结束研究，也无需人为忽略首买或改用下一笔交易作为停止条件。发现流程仍需完整解析创建交易并保留顺序与证据；若发现与认可的交易活动产生重复研究需求，建议按[刷新调度建议](research-lifecycle-and-refresh.md)合并，具体去重语义随活动清单细化。

## 后续设计需要解决的事项

- 发现起点、资产纳入、平台覆盖、链上角色归因与活动触发清单。
- 目标网络、初始历史区间、时效目标、节点配额与历史保留能力。
- 项目身份、重复发现、交易顺序、确认等级、分叉及持续覆盖模型。
- 独立服务职责、进程、接口、持久化、事务、鉴权、配置和停止边界。

服务边界设计须按 [SDS-R1 至 SDS-R8](../../developer-guide/service-development-standards.md)形成独立运行和验证方案；当前没有新增进程、RPC 契约或运行命令。仅本文档修改适用 SDS-R7、SDS-R8 的事实记录与静态验证要求。

## 验证状态

本轮核对官方文档与既有源码入口，未连接实际 Solana 节点采样，未验证提供方容量或实现目标流程。后续验证应覆盖直接初始化、CPI 初始化、失败交易、同笔创建与买入、重复通知、断线补查、Token-2022、版本化交易及不完整历史；具体预期由已确认规则决定。
