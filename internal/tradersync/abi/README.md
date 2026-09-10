# 固定 Exchange ABI 与运行代码证据

[`provenance.json`](provenance.json)记录原始来源 URL、时间、commit、响应 SHA256、提取 ABI 的 SHA256 和实际 runtime Keccak256。三个 JSON 仅保留固定 Sourcify ABI 中的 OrderFilled / OrdersMatched / Upgraded 事件，逐项结构一致；Core 与 NegRisk 共用事件 ABI，但部署 immutable 不同，必须使用各自 runtime hash。实现 runtime 保存在[测试资源](../testdata/runtime/)，不是待生成代码。

Core/NegRisk 的五份固定官方源文件与 Sourcify sources 逐字一致，官方 commit 为 `ccc0596074f4dfd62c944fbca4de252893b82b4b`。Combo 使用固定实现 `0x641b40ec414a076b9e79e703fc7bf4ebec248bb7`，没有用 latest 地址替换旧来源。Sourcify 的 match 是供应商验证证据，本任务没有独立重编译。

## 实际代理与升级语义

保存的 dRPC 请求/完整响应限定已知块 `0x1d64636dcdfa966bb1c6564f0ff3f4322bfc23150c2579f737eb36c9bb5028ad`（93550754）。61 字节代理的[完整反汇编](evidence/proxy-disassembly.txt)只从固定 EIP1967 槽 SLOAD，再 DELEGATECALL；唯一条件跳转按执行成功位返回或回滚。PUSH32 内容不是指令；整个实际 runtime 没有 SSTORE 或管理员选择器路径。因此代理自身没有另一条静默改 implementation 槽的分支。

同一固定块的槽和实现完整代码响应也已保存，代码与固定 Sourcify runtime 逐字相同。[固定 UUPS 源码](evidence/UUPSUpgradeable.sol)的 upgradeToAndCall 在第 88 行发 Upgraded、第 89 行才写槽、第 96 行才可委托新实现；Exchange 仅覆写授权为 onlyOwner。委托实现仍可修改代理 storage，不能从“代理无 SSTORE”推断未来未知实现安全。首次从已核准实现升级必须发事件，所以即使同块升级离开后又返回，完整候选块 Upgraded 查询也会使核验保守拒绝。

这些是固定块补证，没有验证该块 finality、父块或候选执行历史。仅 dRPC 账户端点成功，没有第二个独立供应商交叉成功；此前 Chainstack HTTPS Cloudflare1010 与同 WSS 的 Archive 拒绝保留在执行资料中，不当作空代码或无升级。运行时仍须验证链 137、已知头与实际 ParentHash、父/候选部署代码及槽/实现、完整候选块升级结果，缺任一项保持 unverified。静态 registry ID 仅供已核验管线选择解码器，不能单独证明日志可接纳。
