# Solana 基础信息补全与发行来源

状态：首版源分支已实现并验收，现已按用户授权集成到 `rf4`，并通过[本次集成验证](../../testing/rf4-branch-integration.md)。运行继续按用户要求暂停，现状见[设计总览](README.md)；本文描述程序运行时的行为。

关联：[业务需求](../../requirements/solana/README.md)、[发现流程](project-discovery.md)、[列表展示](../web-ui/solana-discovery.md)。本文件承接任务规格中的长期字段和证据规则，后续任务无需依赖聊天记录恢复这些约定。

## 名称与符号

名称与符号是发行方在链上保存的文字，可以为空、重复或修改。它们不是项目唯一身份、认证结果或发行时历史快照。Mint 始终是候选唯一标识。

使用 `getMultipleAccounts` 的 finalized/base64 快照，将每个 Mint 与其 canonical Metaplex Metadata PDA 配对查询。返回位置、数量及 context.slot 必须核对；观察 slot 不得早于初始化 slot。单个账户损坏不使同批其他候选失败。

| Mint / 指针情况 | 当前采用的元数据 |
| --- | --- |
| 原 Token Program | canonical Metaplex Metadata PDA |
| Token-2022，自指 MetadataPointer | 该 Mint 的 TokenMetadata 扩展 |
| Token-2022，指向 canonical Metaplex PDA | 对应 Metaplex Metadata |
| Token-2022，无显式 pointer，且无 TokenMetadata 扩展 | 尝试 canonical Metaplex PDA |
| Token-2022，有 TokenMetadata 扩展但无 pointer | 记 unavailable，不将缺少指针依据的扩展作为元数据，也不回退其他来源 |
| 指向其他账户或空目标 | 暂不支持，记 unavailable；不改用与指针冲突的数据 |

校验 owner、非 executable、初始化 Mint 类型、内嵌 Mint、padding、TLV 长度、重复扩展、字符串长度和 UTF-8。Token-2022 TLV 从 offset 166 开始，MetadataPointer type 18、TokenMetadata type 19；TokenMetadata 值内为 update authority 32 bytes、Mint 32 bytes 与 Borsh 字符串，不添加独立接口的 8-byte discriminator。Metaplex 使用 canonical PDA、MetadataV1 key 4 及内嵌 Mint；仅去掉字符串尾部 NUL padding。

纯空白名称/符号视为缺失；不接受数据库 text 无法保存的嵌入 NUL。URI 只用于验证链上结构，不访问它，不下载 Logo 或抓取网站。字段独立为空，不因为符号缺失就隐藏已有名称。

## 发行来源证据

发行来源来自成功初始化交易。已知发行指令必须位于该 Mint 初始化的调用祖先中，且程序、selector 和指定 Mint 账户同时匹配；只看到某程序出现在同一交易中不够。

| 来源代码 | 显示含义 | 程序地址 |
| --- | --- | --- |
| `pump_fun` | Pump.fun | `6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P` |
| `raydium_launchlab` | Raydium LaunchLab 协议 | `LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj` |
| `direct_token` | 顶层直接 Token 初始化 | 实际 Token / Token-2022 程序 |
| `unknown` | 未识别平台 | 有充分调用栈证据时保存直接父程序，否则为空 |

当前指令映射（账户下标从 0 开始）：

| 发行指令 | 8-byte selector | 目标 Mint 下标 |
| --- | --- | --- |
| Pump `create` | 24,30,200,40,5,28,7,119 | 0 |
| Pump `create_v2` | 214,144,76,236,95,139,49,180 | 0 |
| LaunchLab `initialize` | 175,175,109,31,13,152,155,237 | 6 |
| LaunchLab `initialize_v2` | 67,153,175,39,218,16,38,32 | 6 |
| LaunchLab `initialize_with_token_2022` | 37,190,126,222,44,154,171,17 | 6 |

按 innerInstructions 的 group.index 和 stackHeight 重建连续祖先链。兄弟调用不能互相归因；高度缺失或跳层时不能猜测直接父程序。顶层已知发行指令仍可由 group.index 与精确 Mint 账户证明祖先关系。多个相互冲突的已知平台命中时保留 unknown。

LaunchLab 的下标 6 是 base_mint，不把下标 7 的 quote_mint 误标为新发行币。协议来源不代表用户使用了 Raydium 官网或某第三方品牌。`direct_token` 不证明“手工创建”或“没有平台”。Token 标准、Metaplex 管理程序、update authority、手续费支付方、地址后缀与名称均不能作为平台归因依据。

新增候选在区块解析时写入来源；历史候选以原签名调用 finalized/jsonParsed `getTransaction`，复用相同解析器，并匹配 Mint、签名、Token 程序与初始化 slot。

## 公开字段合同

合同源为实现分支 `internal/server/solana/solana.proto`，列表接口不增加新的写操作。原初始化字段 1–10 保留，增加：

| 字段 | proto 序号 / 类型 | 含义 |
| --- | --- | --- |
| `name`、`symbol` | 11、12 / string | 当前观察到的名称、符号，分别允许为空 |
| `metadataStatus` | 13 / string | pending / ready / unavailable / error |
| `metadataSource` | 14 / string | token2022_on_mint / metaplex / 空 |
| `metadataAccount` | 15 / string | 已验证的元数据账户 |
| `metadataObservedSlot` | 16 / uint64 | finalized 账户观察 slot；未知为 0 |
| `metadataUpdatedAt` | 17 / int64 | 成功读取观察时间的 Unix 秒；未知为 0，失败保留上次值 |
| `issuanceSource` | 18 / string | pump_fun / raydium_launchlab / direct_token / unknown |
| `issuanceProgram` | 19 / string | 已知匹配的发行程序；未知时可证明的直接父程序 |
| `sourceStatus` | 20 / string | pending / identified / unrecognized / error |

元数据观察 slot/时间与发行 slot/blockTime 分开，不用后续查询覆盖初始化权限、付费人、签名或首次发现时间。API/UI 读取存储，不直接向节点查询。

## 首次补全状态与重试

| 状态 | 含义 | 后续行为 |
| --- | --- | --- |
| metadata `pending` | 尚未读取 | 服务运行时排队 |
| metadata `ready` | 至少名称或符号一项非空且已验证 | 完成本次首次补全，不周期更新；另一项仍可为空 |
| metadata `unavailable` | 当前没有可读元数据、两项均空或指针暂不支持 | 1 小时后重试 |
| metadata `error` | RPC、数据校验或解码失败 | 5 分钟后重试；更长 Retry-After 优先 |
| source `pending` | 历史记录尚未归因 | 查询原初始化交易 |
| source `identified` | 已证明支持的来源 | 完成，不重复查询 |
| source `unrecognized` | 已检查交易但没有支持的平台归因 | 完成，不重复查询 |
| source `error` | 交易不可得或初始化证据无法匹配 | 5 分钟后重试；更长 Retry-After 优先 |

metadata 与 source 分别维护状态和下次到期时间，已完成的一边不随另一边重试。数据库保存待处理记录，按到期时间、发现时间与 Mint 稳定排序；每批最多 10 个候选、20 个账户，同批原交易签名去重，批间等待 10 秒。

扫描和补全共用配置的节点预算、超时与 429 冷却。实际请求发出前统一准入，避免冷却期间囤积令牌并在到期时集中发送；网络调用仍可并行。数据库事务不等待远端请求，补全错误不改变扫描游标；错误不清空已有有效文字、来源与观察证据。

停止服务时停止两条循环，队列与游标保留；仅恢复服务后才继续处理。上述重试用于首次基础信息补全，与尚未实现的[活动驱动研究刷新](research-lifecycle-and-refresh.md)分开。

## 协议依据与验证

规则依据已在 2026-09-13 实现期间核对：[Solana getMultipleAccounts](https://solana.com/docs/rpc/http/getmultipleaccounts)、[Token-2022 扩展布局](https://github.com/solana-program/token-2022/blob/main/interface/src/extension/mod.rs)、[TokenMetadata 布局](https://github.com/solana-program/token-metadata/blob/main/interface/src/state.rs)、[Metaplex Metadata](https://github.com/metaplex-foundation/mpl-token-metadata/blob/main/clients/rust/src/generated/accounts/metadata.rs)、[Pump 固定版本 IDL](https://github.com/pump-fun/pump-public-docs/blob/f216b6724c6ede79d7cef9ce210b741f7e17e93b/idl/pump.json)、[LaunchLab 固定版本 IDL](https://github.com/raydium-io/raydium-idl/blob/e7e0c96fe77bcf6a020b84a44c47a722aac8e359/raydium_launchpad/raydium_launchpad.json)。

验证覆盖真实 USDC Metaplex 快照、真实 Pump 创建与三级未知 CPI 交易、错误 owner/Mint/长度/UTF-8、兄弟调用、LaunchLab base/quote 区分、失败保留、队列恢复和并发冷却。完整验证事实见[总览](README.md#验证事实)；本次文档整合没有新增协议支持。
