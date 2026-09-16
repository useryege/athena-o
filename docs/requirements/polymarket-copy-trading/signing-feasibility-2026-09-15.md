# Polymarket 专用签名能力：可行性调研

> 调研日期：2026-09-15。结论为协议与离线签名可实现；不是 Wallet 正式功能、真实账户接入或交易链路验收。本轮沿用已确认需求，实验代码为临时验证用途。

## 结论

**现有 Wallet 的 EVM 私钥和 Go 技术栈可以实现 Polymarket 专用签名。** 官方协议公开了认证与订单的签名载荷，官方 SDK 也接受外部签名器。ATHENA 可以在 Wallet 内完成受限签名，通过既有独立交易服务边界返回签名结果，继续保持私钥由 Wallet 保管。

本次用临时生成、未入金的测试身份做了独立对照：官方 SDK 构造载荷并由 Viem 签名，Go 使用项目现有 `go-ethereum v1.17.2` 计算并签名，**26 组载荷的摘要、原始签名、提交用签名编码和恢复的签名地址全部一致**。另有 **146 组字段改动检查**通过：改动已签名内容后，原签名不再对应原签名身份。

该结论不依赖申请 Builder 凭据。Builder／Relayer 接入条件、真实用户账户控制关系及交易可用性仍属于后续接入验证。

## 验证了什么

| 类别 | 本次验证 | 结果与范围 |
| --- | --- | --- |
| CLOB 身份认证 | 1 组 `ClobAuth` EIP-712 载荷 | Go 与 SDK 签名一致；未调用生成或派生 API 凭据接口 |
| 订单签名 | 24 组：V2 普通交易合约、V2 Neg Risk 合约和 V3 合约各覆盖 4 种签名类型及买／卖两个方向 | EOA、Proxy、Safe 的直接订单签名，以及 Deposit Wallet 的嵌套签名和 ERC-7739 编码一致；合成订单不证明实际市场或账户受支持 |
| Deposit Wallet 链上操作签名 | 1 组 ERC-20 `approve` 的 `Batch` 载荷 | 官方构造器使用本地模拟的 nonce，Go 与 SDK 的批次签名一致；没有部署或提交授权 |
| 外部签名器适配 | 官方 workflow 经 `getAddress`、`signTypedData` 回调取得订单签名 | 接口可接外部签名能力；实际 gRPC 适配及 Wallet 权限边界尚未实现 |
| 字段绑定 | 订单金额、token、maker、chainId、交易合约及 domain version，共 144 次；认证 nonce 和批次 nonce 各 1 次 | 146 次均改变摘要且原签名无法恢复为原身份；这不是已实现签名服务的越权拒绝测试 |

直接签名为 65 字节，本次 Deposit Wallet 订单的最终编码为 317 字节。不能将 Deposit Wallet 简化为「把普通 EOA 签名放进同一个字段」。智能合约的实际 `isValidSignature`、CLOB 服务端接受和链上最终执行均未验证。

证据：[SDK 结果](evidence/signing-feasibility-2026-09-15/sdk-result.json)、[Go 对照结果](evidence/signing-feasibility-2026-09-15/go-result.json)、[合成签名向量](evidence/signing-feasibility-2026-09-15/signed-vectors.json)、[版本与源码记录](evidence/signing-feasibility-2026-09-15/source-review.json)。向量使用临时身份，不来自用户 Wallet；私钥未复制进长期证据。

## 官方依据与接入差异

- [官方认证协议](https://docs.polymarket.com/getting-started/api#authentication)：钱包为 `ClobAuth` 签名，再请求 CLOB 凭据；请求认证和订单授权是两层不同机制。
- [官方订单说明](https://docs.polymarket.com/trading/place-orders)：普通账户使用 Exchange 的 EIP-712 Order；Deposit Wallet 另需 `TypedDataSign` 和 ERC-7739 编码。
- 固定 SDK 源码版本为 [`983a10a7579c95043d4099f60873ff7ea817e5a0`](https://github.com/Polymarket/ts-sdk/tree/983a10a7579c95043d4099f60873ff7ea817e5a0)，包版本为 `0.10.0`。[Signer 接口](https://github.com/Polymarket/ts-sdk/blob/983a10a7579c95043d4099f60873ff7ea817e5a0/packages/client/src/types.ts)支持外部签名；[Exchange 编码](https://github.com/Polymarket/ts-sdk/blob/983a10a7579c95043d4099f60873ff7ea817e5a0/packages/client/src/exchange.ts)给出了可移植的实现。
- 当前 SDK 同时存在 V2／V3 订单签名域及按资产选择合约的路径，见[合约选择](https://github.com/Polymarket/ts-sdk/blob/983a10a7579c95043d4099f60873ff7ea817e5a0/packages/client/src/actions/orders/context.ts)。实施前须核准目标普通／Neg Risk 市场实际使用的资产、合约及 domain version，不能把文档某一个 V2 示例硬编码成所有市场的规则。
- 当前 SDK 的[通用授权集合](https://github.com/Polymarket/ts-sdk/blob/983a10a7579c95043d4099f60873ff7ea817e5a0/packages/client/src/actions/approvals.ts)还包含 V3、多个 adapter、自动领取及 Perps 等对象。ATHENA 必须按本期已确认操作限定授权对象，不能直接采用整套默认集合。标准长期额度的既有决定保持有效；发现新的合约需求不等于已确认额外授权范围。

本次检索也遇到已关闭的[社区认证问题报告](https://github.com/Polymarket/ts-sdk/issues/73)。其作者的推断不能替代协议或证明平台当前故障；当前官方文档和固定版本 SDK 仍以签名 EOA 完成 CLOB 认证。真实账户接入必须核对认证身份与实际交易账户，不按该问题描述自行改写认证协议。

## ATHENA 需要补齐的能力

[现有密钥处理](../../../internal/wallet/keys.go)已经使用与本次实验相同的 EVM 密钥格式和加密库；[Wallet 服务](../../../internal/wallet/service.go)具备归属查询、私钥解密及本地签名模式。[现有 RPC](../../../internal/wallet/wallet.proto)的业务签名仍为 Worm 专用，尚无 Polymarket 签名接口。

建议沿用已确认的服务分工：独立交易服务处理账户映射、行情、构单、凭据、提交与结果；Wallet 验证可信用户／服务身份、钱包归属、账户控制关系及操作范围后签名。该建议符合 `SDS-R1`、`SDS-R2`、`SDS-R4`，具体 RPC、状态与失败处理仍须按[服务开发规范](../../developer-guide/service-development-standards.md)形成正式契约。

Wallet 的目标签名能力至少区分认证、订单、必要链上准备／领取三类。金额、方向、token、网络、合约、版本、nonce／时效及用户已确认内容应绑定到本次操作；接口不能退化为调用方提供任意摘要就签名。实验中的通用签名代码只用于算法对照，不能直接作为该接口实现。

**下一阶段是实现与接入验证。** 还需明确各账户类型支持范围，完成专用接口和外部签名器适配，并验证真实 CLOB 凭据、授权、订单接受、成交及领取。Proxy／Safe 的链上准备、真实账户资格和已确认的官网入金同账户链路亦未由本次实验覆盖。

## 执行与环境记录

- Go 验证命令使用 `go run -mod=readonly`，退出码 0；SDK 脚本退出码 0。具体路径及脚本摘要记录于源码证据。
- SDK 校验期间禁用实际 `fetch`，批次 nonce 仅由本地替身提供；没有发送用户密钥、创建真实凭据或账户、提交订单、授权、领取或划转。
- 临时脚本及依赖位于 `.superpowers/polymarket-signing-spike-y0qAyH/`，不接入产品。项目 Go／前端依赖未修改，没有启动 ATHENA 服务、容器或长期进程，无运行环境需要保留。

[返回平台契约校验](manual-trading-contract-verification.md) · [返回总体设计](../../design/trading/polymarket-manual-trading.md)
