# Solana 新项目发现设计

> 当前方案：第一步采用标准 HTTP RPC 的 finalized 区块扫描，独立服务持久发现候选并补齐名称、符号及可确认的发行来源，ATHENA 列表查看。更多平台解析、流式订阅和项目研究留待下一步。
>
> 实现与运行状态见[设计总览](README.md)：首版源分支已验收并集成到 `rf4`。2026-09-17 新实例已纳入十一应用统一运行器，完成当前主网 v1 交易只读兼容和真实后台连续性验收；旧 profile 的暂停与数据保留仍是原现场事实，不限制新实例授权。证据见[全栈验收](../../testing/full-stack-access-acceptance.md)。本文不表示程序此刻在线。
>
> 关联需求：[Solana 项目研究](../../requirements/solana/README.md)。资料核对日期：2026-09-13。

## 范围与现状

本文落实第一步“发现并查看样本”。[持续研究与刷新调度](research-lifecycle-and-refresh.md)保留未来方向，本步没有项目研究、活动监控和每小时资料刷新任务。

现有 [Token 扫描器](../../../internal/token/discovery/application/chain_processor.go) 和 [EVM 候选识别器](../../../internal/token/adapters/evm/candidate_inspector.go) 处理 EVM 合约候选；它们不能直接解释为 Solana 新项目发现能力。已有 [Solana 钱包登录](../identity-access/solana-wallet-authentication.md) 也不承担项目发现。

## 已核对的链上事实

- Solana 的普通 SPL 代币以 Mint 账户标识；Token Program 和 Token Extension Program（Token-2022）是两类主要 Token 程序。Mint、持币账户及其权限具有不同含义，Mint authority 不必等于创建或付费账户。来源：[Solana Tokens](https://solana.com/docs/tokens)。
- Mint 初始化指令包括 `InitializeMint`、`InitializeMint2`。后续对已有 Mint 执行铸币是增加供应量，不能据此判定又发行了一个新项目。来源：[创建 Mint](https://solana.com/docs/tokens/basics/create-mint)、[铸币](https://solana.com/docs/tokens/basics/mint-tokens)。
- 程序内部调用的指令由交易元数据的 `innerInstructions` 表达。首版解析覆盖顶层及 CPI 内部初始化；只检查顶层可能遗漏平台创建路径。来源：[RPC JSON 结构](https://solana.com/docs/rpc/json-structures)。

新 Mint 是候选事实。是否属于研究范围内的普通代币，需要独立业务定义；不能仅凭供应量或 decimals 就断定项目类型。

## 方案比较

| 方式 | 能力与收益 | 限制及适用判断 |
| --- | --- | --- |
| 标准节点 RPC：日志订阅与交易解析、区块扫描 | 按 Token 程序发现成功的 Mint 初始化，可覆盖选定程序内的不同发行途径；历史区块可用于补查 | 需自行解析、去重及证明覆盖；订阅两大 Token 程序仍可能有大量无关交易，吞吐和补查成本需实测 |
| 发行平台创建指令或事件 | 对 Pump.fun、Raydium LaunchLab 等平台识别发行来源、创建角色及发行阶段，项目语义更明确 | 覆盖限于已支持的平台与版本；不能承诺覆盖其他平台或直接创建的代币 |
| Yellowstone / Geyser 交易流 | 服务端可按账户筛选交易并传送执行数据，可作为持续监听的候选接入方式 | 需要支持该服务的节点或提供方；过滤权限、资源额度、回放窗口及恢复方式需核验，不能因使用 gRPC 就承诺低延迟或不丢数据 |

前两行中的业务识别可以使用相同的节点数据；第三行属于传输方式，不额外改变项目纳入范围。

节点接口依据为 [logsSubscribe](https://solana.com/docs/rpc/websocket/logssubscribe)、[getTransaction](https://solana.com/docs/rpc/http/gettransaction)、[getBlock](https://solana.com/docs/rpc/http/getblock)。平台依据为 [Pump Program](https://github.com/pump-fun/pump-public-docs/blob/main/docs/PUMP_PROGRAM_README.md)、[LaunchLab](https://docs.raydium.io/user-flows/launchlab-overview)。交易流及过滤依据为 [Yellowstone 官方项目](https://github.com/rpcpool/yellowstone-grpc#filters-for-streamed-data)。上述收益判断是面向本项目的推论，尚无本地性能验证。

## 前期方案比较结论与本次取舍

首版采用成功初始化新 Mint 作为候选事实，Token Program 和 Token-2022 顶层/CPI 均处理。HTTP finalized 区块扫描提供持久发现闭环；名称/符号与 Pump.fun、Raydium LaunchLab 归因按用户后续确认纳入，暂不叠加 WebSocket 或 Yellowstone。官方方案比较仍保留，用户查看真实样本后再决定是否需要扩展能力。

## 影响设计的接口边界

- `logsSubscribe` 的 `mentions` 每个订阅只支持一个地址，返回签名、执行错误和日志。日志文本不能单独证明目标 Mint 已成功初始化；应核对程序身份、实际指令及交易执行结果。来源：[logsSubscribe](https://solana.com/docs/rpc/websocket/logssubscribe)。
- `blockSubscribe` 被标记为不稳定接口，节点需开启相应配置；不能假设已有 RPC 支持。来源：[blockSubscribe](https://solana.com/docs/rpc/websocket/blocksubscribe)。
- 补查使用 `getBlocks`、`getBlock` 时，应区分跳过的 slot、暂未取得的数据和节点历史缺口；不能把所有 slot 都当作必有区块。交易版本、地址查找表和内部指令也必须纳入解析验证。来源：[getBlocks](https://solana.com/docs/rpc/http/getblocks)、[跳过 slot 的接口说明](https://solana.com/docs/rpc/deprecated/getconfirmedblocks)、[RPC JSON 结构](https://solana.com/docs/rpc/json-structures)。
- `processed` 数据可能回滚；首版已确定仅采纳 `finalized` 数据，并通过主网 genesis hash 校验网络。未来如为降低延迟引入更低确认等级，需要另外设计回滚及重复证据处理；不自动复制 EVM 当前“不考虑重组”的规则。来源：[RPC commitment](https://solana.com/docs/rpc)。
- `programSubscribe` 是账户变化通知。将首次见到的账户变化直接解释为新发行，可能混淆存量账户更新与创建；这是基于接口含义的限制判断。来源：[programSubscribe](https://solana.com/docs/rpc/websocket/programsubscribe)。
- Yellowstone 的普通交易流提供执行后的数据；执行前的 deshred 流缺少执行状态及内部指令等证据，不能独立证明发行成功。本文建议不依赖执行前推送。来源：[Yellowstone deshred 说明](https://github.com/rpcpool/yellowstone-grpc#deshred-transactions)。

## 创建与首笔交易可能重叠

Pump 官方仓库中的 SDK 使用文档提供 `createV2AndBuyInstructions`，将创建和初始买入组装到交易中；LaunchLab 产品流程也包含可选首买。来源：[Pump 创建与首买](https://github.com/pump-fun/pump-fun-skills/blob/main/create-coin/SKILL.md)、[LaunchLab 创建模式](https://docs.raydium.io/user-flows/launchlab-overview)。

系统首次取得成功创建交易时，项目可能已经有首笔买入。已确认 Solana 项目持续研究，因此创建交易内或之后出现买入/Swap 都不结束研究，也无需人为忽略首买或改用下一笔交易作为停止条件。发现流程仍需完整解析创建交易并保留顺序与证据；若发现与认可的交易活动产生重复研究需求，建议按[刷新调度建议](research-lifecycle-and-refresh.md)合并，具体去重语义随活动清单细化。

## 当前服务与数据边界（SDS-R1、R2、R6、R7）

`internal/solanadiscovery` 是唯一业务 owner，拥有解析、RPC 适配、扫描生命周期、`solana_discovery` schema 和查询。独立入口 `cmd/athena-solana-discovery` 不导入 API/UI/其他业务实现；API 仅通过生成的 gRPC 契约代理查询。

扫描 Mainnet Beta 并核对 genesis hash；读取 finalized 范围的 getBlocks，然后并发 getBlock（jsonParsed/full、legacy/v0/v1、`maxSupportedTransactionVersion=1`）。补全读取 getTransaction 使用同一最大版本 1。2026-09-17 真实起点区块包含 v1 交易，旧上限 0 返回 RPC -32015 并按既有“失败不推进”规则阻断整个范围；修订后相同区块返回并由现有解析器处理。该调整只覆盖当前主网只读版本契约，不升级发送、签名或 SDK 交易协议，也不是历史兼容 shim。首次从 head 向前32 slots开始，显式 start-slot 仅在无检查点时生效。范围按slot提交，每个范围内候选和检查点同一事务落库。Mint唯一，重复发现保留首次证据；失败块/损坏已识别初始化不推进范围。提供方返回的跳过slot依其getBlocks结果处理，不宣称自行证明账本完整性。版本依据见[Solana 官方交易版本说明](https://solana.com/developers/cookbook/transactions/versions)。

发现时保存 Mint、Token 程序、交易签名、手续费支付方、初始化权限、精度、slot、blockTime 和 discoveredAt；两个时间分别表示链上时间和首次保存时间。初始化交易同时解析 issuanceSource、issuanceProgram、sourceStatus。所有资产尚未分类，不主动抓取链外 metadata。

### 首次信息补全

发现流程和补全循环属于同一独立服务，名称/符号使用 finalized 账户快照，历史来源用原初始化交易。它们共用节点预算，候选入库不依赖补全成功；状态、字段合同、来源证据与重试规则统一维护在[基础信息补全与发行来源](candidate-metadata.md)，不依赖任务规格恢复长期知识。

项目与账户权限共用已配置 PostgreSQL 数据库的独立业务表：项目位于 `solana_discovery` schema，账户表在现有 public schema。独立进程拥有连接池，账户 adapter 仅借用读取，不能关闭该池。项目提交事务不跨 RPC。数据库仍是共享故障域，API 与服务必须配置到同一账户数据库；独立开发预览使用自己的数据库，避免干扰另一 checkout。

## 接口与授权（SDS-R2、R4、R6）

`internal/server/solana/solana.proto` 生成 `SolanaService.ListProjects` 与 `GetDiscoveryStatus`，对应 `GET /api/v1/solana/projects` 和 `/api/v1/solana/status`。列表 page 从1开始，默认25项，最多100项，按 `slot DESC, mint ASC` 稳定排序，query 最多 128 个 Unicode 字符，可按Mint（大小写敏感）及名称/符号（不区分大小写）字面子串查询，`%`/`_` 不作为通配符。状态含起点、检查点、最新观察的finalized slot、成功时间、候选数和错误；状态值为 `starting`、`catching_up`、`current`、`error`。API 仅查询持久化结果，不对节点发补全请求。

公共 API 验证 Solana READ，从认证上下文取账户ID。内部客户端设置10秒deadline、独立Bearer token和唯一 `x-athena-account-id`，不接收公共请求传入的身份字段。业务 RPC 再验证token和规范账户UUID，每次重读持久权限，要求登录有效且Solana READ；管理员无隐式业务访问。撤权后的后续请求不能靠API缓存继续读取。

新 `solana` 授权最高 READ；当前完整账户权限矩阵精确为七项（包含前端隐藏但仍参与聚合的 Token），新／现有普通成员默认 NONE。该七模块账户矩阵与六个环境访问键是两套独立条件：环境开关不包含 Wallet 或 Token，也不改变账户 revision 或授予 Solana READ。账户初始化及必要升级 SQL、生成客户端和 UI 一同更新。健康 RPC 只说明进程／查询入口可用，节点扫描健康以 `GetDiscoveryStatus` 为准；不把监听端口等同于扫描追平。

## 独立运行与配置（SDS-R3、R4、R5）

当前入口：`make solana-discovery-build`、`make solana-discovery-run INSTANCE=<name>`、`make solana-discovery-stop INSTANCE=<name>`。run/stop 别名委托统一实例 owner；managed 模式按选择创建 PostgreSQL 并先执行账户及 Solana `schema up`／`verify`，external 模式只读 verify，扫描不依赖 API 在线。Solana 已纳入十一应用默认图，也可单独正向选择。旧 `hack/solana-local.sh` 的 profile 仅由原 owner 停止，状态与数据不迁入新实例。

历史旧 `solana-preview` profile 曾将 `ATHENA_SOLANA_PREVIEW_POSTGRES_DSN` 同时提供给 Solana 存储和 API 的 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`，并由旧启动链先准备账户 schema。该 profile 只用于解释保留现场，当前 checkout 已封闭其 start，不能作为现行启动入口；旧状态仅由原 owner 停止。当前预览应使用统一 `run-services` 和独立 `INSTANCE`，按所选服务准备 schema 与依赖。

|配置|默认/行为|
|---|---|
|ATHENA_SOLANA_DISCOVERY_RPC_URL|https://api.mainnet-beta.solana.com；只接受HTTP(S)|
|ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN|未配置时回退 ATHENA_ACCOUNT_STATE_POSTGRES_DSN，两者均缺失则启动报错；须与 API 账户库一致|
|ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN|独立生产部署显式提供；统一本地实例未配置时由当前 owner 生成、以 0600 文件持久保存并在重启中复用，显式值与已保存值冲突时拒绝启动|
|ATHENA_SOLANA_DISCOVERY_LISTEN_ADDRESS / PORT|127.0.0.1 / 8112|
|ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS|API客户端默认127.0.0.1:8112|
|ATHENA_SOLANA_DISCOVERY_START_SLOT|0；首次取head向前32slots，重启沿用持久起点|
|ATHENA_SOLANA_DISCOVERY_RANGE_SIZE|4，允许1..32|
|ATHENA_SOLANA_DISCOVERY_CONCURRENCY|1，允许1..32|
|ATHENA_SOLANA_DISCOVERY_REQUESTS_PER_SECOND|1，允许1..1000；短时令牌突发最多4|
|ATHENA_SOLANA_DISCOVERY_REQUEST_TIMEOUT|15s，每个RPC有界|
|ATHENA_SOLANA_DISCOVERY_POLL_INTERVAL|2s，追平后等待；积压时持续推进|

参数为初始预算，不是容量SLO。请求失败指数退避，最多30秒；429存在Retry-After时等待至少该时长，始终可取消。错误与积压可查询。终止信号取消扫描和节点请求，gRPC最多5秒优雅等待后强制停止，扫描最多5秒收尾，进程关闭自有资源。[Solana 局部运行说明](../../developer-guide/running-locally.md#solana-discovery-local)记录环境准备；当前停机与恢复边界见[总览](README.md#当前运行状态与恢复边界)。

## 验证记录（SDS-R8）

首版发现及信息补全已经过解析器/数据库/权限/API/页面隔离验证，以及真实主网数据、桌面/手机页面与重启恢复验收。验证摘要与原始记录位置见[设计总览](README.md#验证事实)。历史补全队列未全部处理，当前按用户要求停机；功能验收通过不等于已追平链头或全部记录已补齐。

公共节点实测 getBlock 返回约17MB，节点会返回429；即使降低请求次数，也不能仅据QPS推断容量。官方还声明公共端点有数据量额度、限额会变化，不适合作为生产节点；见[Solana公共RPC说明](https://solana.com/docs/references/clusters)。首版预览优先逐块持久保存真实样本，显示catching_up或error，不宣称追平。若后续要求持续低延迟全范围发现，需要相应容量的RPC或更有针对性的发现数据源。
