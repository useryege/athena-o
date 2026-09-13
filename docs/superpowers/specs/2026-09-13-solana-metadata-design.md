# Solana 候选名称、符号与发行来源

日期：2026-09-13。状态：用户已确认补充名称、符号和发行来源，按本文实施。

## 目标与范围

在已有独立 Solana 发现模块及 `/solana` 列表中，为新增和已保存的 Mint 补充名称、符号、发行来源。继续保留初始化交易、Mint 地址、Token 程序和发现时间等证据。候选可能是普通代币、头寸 NFT 或其他 Mint；本轮不引入项目筛选、研究规则、交易监控或完整研究刷新调度。

## 数据来源

名称与符号读取 finalized 链上账户快照：传统 Token 使用 canonical Metaplex Metadata PDA；Token-2022 优先使用有效自指 MetadataPointer 的 TokenMetadata 扩展，也支持指向 canonical Metaplex PDA。指向其他账户的 pointer 暂不支持，不擅自采用冲突元数据。核对 owner、mint、账户类型和长度，验证 UTF-8，去掉 Metaplex 尾部 NUL padding，不抓取 URI、Logo 或其他链外地址。保存 metadataSource、metadataAccount、metadataObservedSlot、metadataUpdatedAt；这些字段描述观察时刻，不声称是发行时名称。

发行来源通过成功的 Mint 初始化交易判断：已知祖先发行指令同时匹配程序地址、selector 和该 Mint 的指定账户。先支持 Pump.fun、Raydium LaunchLab；顶层 Token 初始化显示“直接 Token 程序初始化”；其他显示“未识别平台”，尽可能保留可证明的直接父程序。按 stackHeight 重建 CPI 调用栈，禁止依据地址后缀、名称、任意同交易程序出现或 metadata 管理者猜测平台。LaunchLab 来源表示协议，不代表某个网页品牌。

## 补全、状态和限速

扫描在初始化时解析发行来源，持久化候选与游标仍保持原子性。独立后台补全循环读取待处理记录；已有记录通过原初始化签名查询 getTransaction 补来源，通过 getMultipleAccounts 批量补名称符号。每批最多 10 个候选、20 个账户；同批交易签名去重。扫描和补全共用配置的节点请求预算与超时，429 尊重 Retry-After。补全失败不得推进或阻塞扫描游标，不得覆盖已有有效名称和来源。

metadataStatus：pending（待补全）、ready（至少一项有效文字）、unavailable（目前无可读取元数据或不支持的指针）、error（读取失败）。sourceStatus：pending、identified、unrecognized、error。名称和符号分别允许为空。已验证的字段不自动周期更新；缺失元数据 1 小时后重试，失败 5 分钟后重试；来源已识别或已确认未识别不重复读取交易。该调度仅补全首次信息，不是未来研究生命周期调度。

数据库升级保留当前全部候选和扫描游标，重启后继续待处理补全。补全状态来自存储，API 和 UI 不直接访问节点。查询同时支持 Mint 的区分大小写子串与名称、符号的不区分大小写子串。

## 字段合同

Go Project 新增 Name、Symbol、MetadataStatus、MetadataSource、MetadataAccount、IssuanceSource、IssuanceProgram、SourceStatus（string），MetadataObservedSlot（uint64）、MetadataUpdatedAt（time.Time）。公开 protobuf 接着现有字段增加 name=11、symbol=12、metadataStatus=13、metadataSource=14、metadataAccount=15、metadataObservedSlot=16、metadataUpdatedAt=17、issuanceSource=18、issuanceProgram=19、sourceStatus=20。公开时间为 Unix 秒，空时间为 0。

metadataSource 使用 token2022_on_mint / metaplex / 空；issuanceSource 使用 pump_fun / raydium_launchlab / direct_token / unknown。未知显示说明，不伪造名称或把 Token 程序显示成发售平台。

## 列表与验收

沿用 ATHENA 表格：首列呈现名称、符号与可复制 Mint；发行来源独立一列；Token 程序、元数据出处与观察时间放在展开详情。无名称、补全中、读取失败及未识别来源均有可区分文案。长文字换行或省略，完整地址可查看与复制，手机只在表格容器横向滚动。名称是发行方自述，界面不暗示官方认证。

验收覆盖真实主网 Token-2022 样本、Metaplex 二进制 fixture、已存记录补全、新候选来源、失败重试、重启保留、查询、API 授权和桌面/手机 UI。复用 worktree 的 solana-preview 环境，不修改其他环境或重置数据。按 SDS-R1/R2/R3/R4/R5/R6/R7/R8 保持模块所有权、同一 API 合同、独立进程、现有 READ 授权、短事务、独立预览及验证证据。

## 协议依据

- [Solana getMultipleAccounts](https://solana.com/docs/rpc/http/getmultipleaccounts)
- [Token-2022 扩展布局](https://github.com/solana-program/token-2022/blob/main/interface/src/extension/mod.rs)
- [TokenMetadata 字段布局](https://github.com/solana-program/token-metadata/blob/main/interface/src/state.rs)
- [Metaplex Metadata](https://github.com/metaplex-foundation/mpl-token-metadata/blob/main/clients/rust/src/generated/accounts/metadata.rs)
- [Pump 官方 IDL](https://github.com/pump-fun/pump-public-docs/blob/f216b6724c6ede79d7cef9ce210b741f7e17e93b/idl/pump.json)
- [Raydium 官方 Launchpad IDL](https://github.com/raydium-io/raydium-idl/blob/e7e0c96fe77bcf6a020b84a44c47a722aac8e359/raydium_launchpad/raydium_launchpad.json)
