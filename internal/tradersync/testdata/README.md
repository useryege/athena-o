# 成交测试样本

[source_records.json](source_records.json)覆盖三 Exchange × BUY/SELL × maker/taker，共 12 条，其中 **11 条真实保存日志、1 条明确 synthetic**。每条 origin 保存原始文件、txHash 与 logIndex；log 的完整 address/topics/data/块/交易/索引直接取自[原始链上样本](../../../docs/requirements/polymarket-copy-trading/evidence/onchain-trade-data-sample.json)，没有用生产 decoder 输出回填预期。

真实数据来自该文件 allExchangeFills、combosLogSample 与 selectedFullReceipts。NegRisk SELL taker 来自 selectedFullReceipts 中交易 `0xad3078237fb3aef6d619d0763c3e6f86471b653482173a925ded3fa3c7ac24a4` 的 logIndex `0x14e`（334）；topics[2] 为 `0x33175349046ea26bbeb0a52189960d15d2df76b4`，topics[3] 为 NegRisk Exchange。negRiskTakerSellWindow 仅有窗口计数，不能当原始日志。

Combo SELL maker 在已存原始日志中没有足够样本，使用明确标注 synthetic 的 ABI 七字编码：side=1、position=90071992547409931234567890、makerAmount=7000000、takerAmount=2100000、fee=33000，builder/metadata=0；地址、块与交易定位均标明为测试构造，不能声称真实链验收。后续真实来源验收仍须补这个组合。

预期值按固定 ABI 手工读取七个 32 字节 word：BUY 的抵押币/份额为 maker/taker，SELL 反转；fee 独立。maker/taker 是交易角色，实际归属始终是 topics[2] 的资金钱包；主动单日志的 topics[3] 为 Exchange。大 Position ID 和金额保存十进制字符串，未经过浮点。角色标签不是 decoder 输入。

[runtime/](runtime/)保存三个固定实现及实际代理的原始十六进制 runtime，用于版本 fake 返回真实字节并核对生产 hash；来源与 SHA/Keccak 见[ABI provenance](../abi/provenance.json)。测试中的规范头、父哈希、成功回执、升级/重组故障以及单 tx 多自身日志变体均是定向构造，不声称已经逐历史样本证明其执行版本或最终性。

## Task 8 metadata evidence

`metadata.json` extracts the recorded two-leg migration calldata/results and the exact closed Gamma markets 3978602/3978659 from the saved protocol evidence. The tests require those literal calldata bytes, including the structured (not legacy) bytes32 condition passed to `getLegacyPositionId`. They do not use holdings or create additional trades. The original calls used a fixed block number; test candidate/parent headers are synthetic to exercise hash-scoped verification, not proof of historical upgrade absence.

`runtime/{combinatorial,binary}-module.hex` contains the approved deployment runtime bytes from the saved module evidence. Minimal callable ABIs, provenance, source hashes and interpretation/upgrade excerpts are recorded in `../abi/module-provenance.json`. The directory module 1/2 unit fixtures exercise exact matching and failure branches; they do not claim native Binary migration was verified. Full third-party Sourcify sources are not copied into the repository.
