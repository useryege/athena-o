# 关键操作事件目录

> 详细设计已于 2026-09-18 整体审阅通过，按用户要求暂不实施；[机器清单](event-catalog.json)与本表一致。[总体设计](../2026-09-18-key-operation-logs-design.md)。

共 **98 个采集位置、74 个动作编码**；其中 37 个 gRPC 写操作、61 个原生 HTTP／认证流程采集位置，包含失败专用位置及登录子动作，并非 98 个独立 URL。相同动作的不同身份供应商入口共用 actionCode，但每次请求只记录一次。

## 结果策略

- COMMIT：持久操作已确认才标记成功；标记提交事实后发生响应／缓存发布错误，保留已提交 effect。
- ACCEPTED：只承诺异步命令已受理，记录当时业务状态，不声明最终交易／投递成功。
- COMMAND：执行控制命令；服务确认本命令目标状态已持久达成则成功，明确接受继续处理则已受理；实际失败、拒绝、未知按采集契约映射。
- CONNECTION：CONNECTED／已确认 NOT_CONNECTED 为相应连接／断开目标达成；处理中为已受理，旧凭据仍可能存在等已知部分效果为 PARTIAL；无法核实则 UNKNOWN。
- MULTISTEP：保存钱包选择等多步操作；已提交部分移除、其后失败时记录 PARTIAL，全部确认才成功。
- INITIALIZATION_FAILURE：只采集初始化失败，不记录成功生成挑战或跳转。
- LOGIN／REGISTRATION／LOGOUT／AUTHORIZATION／REVEAL：按[采集与结果契约](collection-and-results.md)的具体提交点判定。

详情字段为日志标准化字段名，不等于直接拷贝原请求字段。缺少事实时省略；仅对动作适用的字段赋值。expectedRevision 是请求的预期版本，confirmedRevision 是已确认提交版本。数值保持十进制字符串；备注、提案正文、投票内容、密钥不进入详情。

## 详情字段类型

所有字段可缺失，缺失表示未观察到，不填造默认值。类型相同也只能用于本动作 allowedDetails 已列出的字段；状态枚举以该入口的既有业务类型为来源，并在 producer 显式映射为稳定代码。

| 类型 | 字段 | 约束 |
| --- | --- | --- |
| bool | accountCreated、accountResolved、beforeAvailable、confirmedOpen、cookieCleared、noActiveSession、noteChanged、requestedOpen、revocationConfirmed | 保留 false 与缺失的区别 |
| uint64 十进制字符串 | addedCount、blockNumber、chainId、confirmedCount、confirmedRevision、disputeCount、expectedRevision、expiresIn、gatewayCount、itemCount、proposalCount、removedCount、requestedCount、requestsPerKey、revision、stepCount | 不允许负值或浮点；expiresIn 单位为秒 |
| 规范标识字符串 | avatarPresetId、ballotId、batchId、cashOutId、codeHash、combinationId、deliveryId、displayId、planId、runId、stepId、subscriptionId、walletAddress、walletId | 遵循原对象格式，最多 128 字节；displayId 只取 API Key 展示 ID |
| 业务 slug 字符串 | newSlug、oldSlug、slug | 最多 256 字节；不把提案标题／正文当 slug |
| 枚举代码字符串 | attemptStatus、avatarKind、bindingStatus、confirmedTier、deliveryStatus、moduleKey、proofKind、provider、requestedTier、stage、state、walletType、warningCode | 最多 64 字节；不是任意错误正文 |
| 标识数组 | walletIds | 每项为规范 wallet ID，最多 100 项；缺省／截断由外层计数与 completeness 表达 |
| 字段代码数组 | changedFields | 最多 32 项，每项最多 64 字节，只表示变更字段名 |
| 权限对象 | requestedAccess、confirmedAccess | loginEnabled、apiKeyEnabled、profitSharingEnabled 三个 bool，revision 为 uint64 字符串，moduleAccess 为既有模块／权限枚举对数组；不接收任意额外键 |

权限对象在详情中展开为具体字段的 Change，枚举、布尔值和版本分别比较；不能把完整嵌套对象塞进只接受标量的 beforeValue／afterValue。没有同版本旧值时 beforeAvailable=false。

## gRPC 入口

| 动作编码 | 公共 RPC | 对象 | 允许详情 | 结果策略 | 源码 |
| --- | --- | --- | --- | --- | --- |
| account.access.update | `/account.AccountService/UpdateAccountAccess` | account | requestedAccess, expectedRevision, confirmedAccess, confirmedRevision, beforeAvailable | COMMIT | [UpdateAccountAccess](../../../../internal/server/account/account.go) |
| account.profile.update | `/account.AccountService/UpdateAccountProfile` | account | changedFields, expectedRevision, confirmedRevision | COMMIT | [UpdateAccountProfile](../../../../internal/server/account/account.go) |
| account.tier.update | `/account.AccountService/UpdateAccountTier` | account | requestedTier, confirmedTier, expectedRevision, confirmedRevision, beforeAvailable | COMMIT | [UpdateAccountTier](../../../../internal/server/account/account.go) |
| account.api_key.create | `/account.AccountService/CreateToken` | api_key | displayId, expiresIn | COMMIT | [CreateToken](../../../../internal/server/account/account.go) |
| account.api_key.delete | `/account.AccountService/DeleteToken` | api_key | displayId | COMMIT | [DeleteToken](../../../../internal/server/account/account.go) |
| managed_oo.block.scan | `/managedoo.ManagedOOService/ScanManagedOOBlock` | block | blockNumber, proposalCount, disputeCount | COMMIT | [ScanManagedOOBlock](../../../../internal/server/managedoo/managedoo.go) |
| system.module_access.update | `/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting` | module | moduleKey, requestedOpen, confirmedOpen, beforeAvailable | COMMIT | [UpdateModuleAccessSetting](../../../../internal/server/moduleaccess/server.go) |
| notification.binding_attempt.create | `/notification.NotificationService/CreateTelegramBindingAttempt` | binding_attempt | attemptStatus, revision | COMMIT | [CreateTelegramBindingAttempt](../../../../internal/server/notification/notification.go) |
| notification.binding_attempt.cancel | `/notification.NotificationService/DeleteTelegramBindingAttempt` | binding_attempt | attemptStatus | COMMIT | [DeleteTelegramBindingAttempt](../../../../internal/server/notification/notification.go) |
| notification.binding.delete | `/notification.NotificationService/DeleteTelegramBinding` | account | bindingStatus | COMMIT | [DeleteTelegramBinding](../../../../internal/server/notification/notification.go) |
| system.notification.test | `/notification.NotificationService/SendSystemNotificationTest` | notification_delivery | deliveryId, deliveryStatus | ACCEPTED | [SendSystemNotificationTest](../../../../internal/server/notification/notification.go) |
| profit_sharing.round.create | `/profitsharing.ProfitSharingService/CreateRound` | round | slug, revision, state | COMMIT | [CreateRound](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.round.update | `/profitsharing.ProfitSharingService/UpdateRound` | round | oldSlug, newSlug, revision, changedFields | COMMIT | [UpdateRound](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.round.open | `/profitsharing.ProfitSharingService/OpenRound` | round | slug, revision, state | COMMIT | [OpenRound](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.round.publish | `/profitsharing.ProfitSharingService/PublishRound` | round | slug, revision, state | COMMIT | [PublishRound](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.ballot.close | `/profitsharing.ProfitSharingService/CloseBallot` | round | slug, ballotId, state | COMMIT | [CloseBallot](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.proposal.update | `/profitsharing.ProfitSharingService/UpdateProposal` | proposal | slug, revision, changedFields | COMMIT | [UpdateProposal](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.proposal.submit | `/profitsharing.ProfitSharingService/SubmitProposal` | proposal | slug, revision, state | COMMIT | [SubmitProposal](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.proposal.reopen | `/profitsharing.ProfitSharingService/ReopenProposal` | proposal | slug, revision, state | COMMIT | [ReopenProposal](../../../../internal/server/profitsharing/profitsharing.go) |
| profit_sharing.vote.submit | `/profitsharing.ProfitSharingService/SubmitVote` | ballot | slug, ballotId | COMMIT | [SubmitVote](../../../../internal/server/profitsharing/profitsharing.go) |
| system.gateway_probe.create | `/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe` | probe_run | runId, requestsPerKey, gatewayCount, state | ACCEPTED | [RunEtherscanGatewayProbe](../../../../internal/server/servicestatus/etherscan_gateway_probe.go) |
| token.checkpoint.update | `/tokenapi.TokenOperationsService/UpdateChainCheckpoint` | chain | chainId, blockNumber | COMMIT | [UpdateChainCheckpoint](../../../../internal/server/tokenapi/tokenapi.go) |
| token.contract_blocklist.create | `/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry` | code_hash | codeHash | COMMIT | [CreateContractCodeBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| token.contract_blocklist.update | `/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry` | code_hash | codeHash, changedFields | COMMIT | [UpdateContractCodeBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| token.contract_blocklist.delete | `/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry` | code_hash | codeHash | COMMIT | [DeleteContractCodeBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| token.wallet_blocklist.create | `/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry` | wallet_address | walletAddress | COMMIT | [CreateWalletBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| token.wallet_blocklist.update | `/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry` | wallet_address | walletAddress, changedFields | COMMIT | [UpdateWalletBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| token.wallet_blocklist.delete | `/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry` | wallet_address | walletAddress | COMMIT | [DeleteWalletBlocklistEntry](../../../../internal/server/tokenapi/tokenapi.go) |
| trader_sync.subscription.create | `/tradersync.TraderSyncService/CreateSubscription` | subscription | subscriptionId, revision, state, noteChanged | COMMIT | [CreateSubscription](../../../../internal/server/tradersync/tradersync.go) |
| trader_sync.subscription.pause | `/tradersync.TraderSyncService/PauseSubscription` | subscription | subscriptionId, expectedRevision, confirmedRevision, state | COMMIT | [PauseSubscription](../../../../internal/server/tradersync/tradersync.go) |
| trader_sync.subscription.resume | `/tradersync.TraderSyncService/ResumeSubscription` | subscription | subscriptionId, expectedRevision, confirmedRevision, state | COMMIT | [ResumeSubscription](../../../../internal/server/tradersync/tradersync.go) |
| trader_sync.subscription.cancel | `/tradersync.TraderSyncService/CancelSubscription` | subscription | subscriptionId, expectedRevision, confirmedRevision, state | COMMIT | [CancelSubscription](../../../../internal/server/tradersync/tradersync.go) |
| trader_sync.target_note.update | `/tradersync.TraderSyncService/UpdateTargetNote` | wallet_address | walletAddress, expectedRevision, confirmedRevision, changedFields | COMMIT | [UpdateTargetNote](../../../../internal/server/tradersync/tradersync.go) |
| wallet.batch.create | `/wallet.WalletService/BatchCreateWallets` | wallet_batch | walletType, requestedCount, confirmedCount, walletIds | COMMIT | [BatchCreateWallets](../../../../internal/server/wallet/wallet.go) |
| wallet.batch.import | `/wallet.WalletService/BatchImportWallets` | wallet_batch | walletType, requestedCount, confirmedCount, walletIds | COMMIT | [BatchImportWallets](../../../../internal/server/wallet/wallet.go) |
| wallet.remark.update | `/wallet.WalletService/UpdateWalletRemark` | wallet | expectedRevision, confirmedRevision, changedFields | COMMIT | [UpdateWalletRemark](../../../../internal/server/wallet/wallet.go) |
| wallet.avatar_preset.update | `/wallet.WalletService/UpdateWalletAvatarPreset` | wallet | avatarPresetId, expectedRevision, confirmedRevision | COMMIT | [UpdateWalletAvatarPreset](../../../../internal/server/wallet/wallet.go) |

## 原生 HTTP／认证入口

表中的 `(purpose)` 用于说明 Google callback 的现有分派，不是 URL 的一部分。路径参数名是文档规范名称，实施时仍使用现有 handler 的实际参数名。

| 动作编码 | 入口 | 对象／策略 | 允许详情 | 采集位置 |
| --- | --- | --- | --- | --- |
| account.avatar.upload | `PUT /api/v1/account/{id}/avatar` | account / COMMIT | expectedRevision, confirmedRevision, avatarKind | [Upload](../../../../internal/server/accountavatarhttp/handler.go) |
| account.avatar.delete | `DELETE /api/v1/account/{id}/avatar` | account / COMMIT | expectedRevision, confirmedRevision, avatarKind | [Delete](../../../../internal/server/accountavatarhttp/handler.go) |
| wallet.avatar.upload | `PUT /api/v1/wallets/{id}/avatar` | wallet / COMMIT | expectedRevision, confirmedRevision, avatarKind | [Upload](../../../../internal/server/walletavatarhttp/handler.go) |
| wallet.avatar.delete | `DELETE /api/v1/wallets/{id}/avatar` | wallet / COMMIT | expectedRevision, confirmedRevision, avatarKind | [Delete](../../../../internal/server/walletavatarhttp/handler.go) |
| wallet.private_key.reveal | `POST /api/v1/wallets/{id}:revealPrivateKey` | wallet / REVEAL | 无 | [Reveal](../../../../internal/server/walletsecrethttp/handler.go) |
| worm.wallet_selection.replace | `PUT /api/v1/worm-trading/wallet-selection` | wallet_selection / MULTISTEP | walletIds, requestedCount, confirmedRevision, removedCount, addedCount | [replaceWormWalletSelection](../../../../internal/server/worm_wallet_selection.go) |
| worm.connection.connect | `POST /api/v1/worm-trading/wallet-connections/{id}` | wallet / CONNECTION | state, warningCode | [manageWormConnection](../../../../internal/server/worm_connection.go) |
| worm.connection.reconnect | `POST /api/v1/worm-trading/wallet-connections/{id}:reconnect` | wallet / CONNECTION | state, warningCode | [manageWormConnection](../../../../internal/server/worm_connection.go) |
| worm.connection.regenerate | `POST /api/v1/worm-trading/wallet-connections/{id}:regenerate` | wallet / CONNECTION | state, warningCode | [manageWormConnection](../../../../internal/server/worm_connection.go) |
| worm.connection.disconnect | `DELETE /api/v1/worm-trading/wallet-connections/{id}` | wallet / CONNECTION | state, warningCode | [manageWormConnection](../../../../internal/server/worm_connection.go) |
| worm.combination.create | `POST /api/v1/worm-trading/combinations` | combination / COMMIT | expectedRevision, confirmedRevision, itemCount, changedFields | [createWormCombination](../../../../internal/server/worm_combinations.go) |
| worm.combination.update | `PUT /api/v1/worm-trading/combinations/{id}` | combination / COMMIT | expectedRevision, confirmedRevision, itemCount, changedFields | [updateWormCombination](../../../../internal/server/worm_combinations.go) |
| worm.combination.delete | `DELETE /api/v1/worm-trading/combinations/{id}` | combination / COMMIT | expectedRevision, confirmedRevision, itemCount, changedFields | [deleteWormCombination](../../../../internal/server/worm_combinations.go) |
| worm.execution_plan.create | `POST /api/v1/worm-trading/execution-plans` | execution_plan / COMMIT | combinationId, planId, state, stepCount | [createWormExecutionPlan](../../../../internal/server/worm_execution_plans.go) |
| worm.execution.create | `POST /api/v1/worm-trading/executions` | execution / COMMIT | planId, runId, state | [createWormExecution](../../../../internal/server/worm_executions.go) |
| worm.execution.start | `POST /api/v1/worm-trading/executions/{runId}:start` | execution / COMMAND | expectedRevision, confirmedRevision, state | [startWormExecution](../../../../internal/server/worm_executions.go) |
| worm.execution.pause | `POST /api/v1/worm-trading/executions/{runId}:pause` | execution / COMMAND | expectedRevision, confirmedRevision, state | [pauseWormExecution](../../../../internal/server/worm_executions.go) |
| worm.execution.continue | `POST /api/v1/worm-trading/executions/{runId}:continue` | execution / COMMAND | expectedRevision, confirmedRevision, state | [continueWormExecution](../../../../internal/server/worm_executions.go) |
| worm.execution.terminate | `POST /api/v1/worm-trading/executions/{runId}:terminate` | execution / COMMAND | expectedRevision, confirmedRevision, state | [terminateWormExecution](../../../../internal/server/worm_executions.go) |
| worm.execution_step.reconcile | `POST /api/v1/worm-trading/executions/{runId}/steps/{stepId}:reconcile` | execution_step / COMMAND | runId, stepId, expectedRevision, confirmedRevision, state | [reconcileWormExecutionStep](../../../../internal/server/worm_executions.go) |
| worm.cash_out.create | `POST /api/v1/worm-trading/position-cash-outs` | cash_out / ACCEPTED | walletId, cashOutId, state, stage | [createWormPositionCashOut](../../../../internal/server/worm_position_cash_outs.go) |
| worm.cash_out.reconcile | `POST /api/v1/worm-trading/position-cash-outs/{cashOutId}:reconcile` | cash_out / COMMAND | expectedRevision, confirmedRevision, state, stage | [reconcileWormPositionCashOut](../../../../internal/server/worm_position_cash_outs.go) |
| worm.cash_out_batch.create | `POST /api/v1/worm-trading/position-cash-out-batches` | cash_out_batch / ACCEPTED | walletIds, requestedCount, batchId, state | [createWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| worm.cash_out_batch.cancel | `POST /api/v1/worm-trading/position-cash-out-batches/{batchId}:cancel` | cash_out_batch / COMMAND | expectedRevision, confirmedRevision, state | [mutateWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| worm.cash_out_batch.pause | `POST /api/v1/worm-trading/position-cash-out-batches/{batchId}:pause` | cash_out_batch / COMMAND | expectedRevision, confirmedRevision, state | [mutateWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| worm.cash_out_batch.continue | `POST /api/v1/worm-trading/position-cash-out-batches/{batchId}:continue` | cash_out_batch / COMMAND | expectedRevision, confirmedRevision, state | [mutateWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| worm.cash_out_batch.terminate | `POST /api/v1/worm-trading/position-cash-out-batches/{batchId}:terminate` | cash_out_batch / COMMAND | expectedRevision, confirmedRevision, state | [mutateWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| worm.cash_out_batch.check_status | `POST /api/v1/worm-trading/position-cash-out-batches/{batchId}:check-status` | cash_out_batch / COMMAND | expectedRevision, confirmedRevision, state | [mutateWormPositionCashOutBatch](../../../../internal/server/worm_position_cash_out_batches.go) |
| identity.login | `GET /auth/google/callback (login)` | account / LOGIN | stage, provider | [Callback](../../../../internal/googleoidc/handler.go) |
| identity.login | `POST /auth/phantom/verify` | account / LOGIN | stage, provider | [Verify](../../../../internal/phantomauth/handler.go) |
| identity.registration.submit | `POST /auth/registration` | account / REGISTRATION | stage, accountCreated, accountResolved | [create](../../../../internal/authregistration/handler.go) |
| identity.registration.cancel | `DELETE /auth/registration` | registration / COMMIT | stage | [delete](../../../../internal/authregistration/handler.go) |
| identity.logout | `GET /auth/logout` | account / LOGOUT | cookieCleared, revocationConfirmed, noActiveSession | [ServeHTTP](../../../../internal/server/logout/logout.go) |
| wallet.reveal.authorize | `GET /auth/google/callback (wallet.reveal.authorize)` | account / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [callback](../../../../internal/googleoidc/wallet_secret_reauth.go) |
| wallet.reveal.authorize | `POST /auth/wallet-secrets/solana/verify` | account / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [WalletSecretVerify](../../../../internal/phantomauth/wallet_secret_reauth.go) |
| wallet.reveal.authorize | `POST /auth/wallet-secrets/development` | account / AUTHORIZATION | stage, proofKind | [developmentWalletSecretLease](../../../../internal/server/wallet_secret.go) |
| worm.connection.authorize | `GET /auth/google/callback (worm.connection.authorize)` | account / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [callback](../../../../internal/googleoidc/worm_credential_reauth.go) |
| worm.connection.authorize | `POST /auth/worm-trading/solana/verify` | account / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [WormCredentialVerify](../../../../internal/phantomauth/worm_credential_reauth.go) |
| worm.connection.authorize | `POST /auth/worm-trading/development` | account / AUTHORIZATION | stage, proofKind | [developmentWormCredentialLease](../../../../internal/server/worm_connection.go) |
| worm.execution.authorize | `GET /auth/google/callback (worm.execution.authorize)` | execution / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [callback](../../../../internal/googleoidc/worm_execution_authorization.go) |
| worm.execution.authorize | `POST /auth/worm-trading/executions/{runId}/solana/verify` | execution / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [WormExecutionVerify](../../../../internal/phantomauth/worm_execution_authorization.go) |
| worm.execution.authorize | `POST /auth/worm-trading/executions/{runId}/development` | execution / AUTHORIZATION | stage, proofKind | [developmentWormExecutionAuthorization](../../../../internal/server/worm_execution_authorization.go) |
| worm.cash_out.authorize | `GET /auth/google/callback (worm.cash_out.authorize)` | cash_out / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [callback](../../../../internal/googleoidc/worm_position_cash_out_authorization.go) |
| worm.cash_out.authorize | `POST /auth/worm-trading/position-cash-outs/{cashOutId}/solana/verify` | cash_out / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [WormPositionCashOutVerify](../../../../internal/phantomauth/worm_position_cash_out_authorization.go) |
| worm.cash_out.authorize | `POST /auth/worm-trading/position-cash-outs/{cashOutId}/development` | cash_out / AUTHORIZATION | stage, proofKind | [developmentWormPositionCashOutAuthorization](../../../../internal/server/worm_position_cash_out_authorization.go) |
| worm.cash_out_batch.authorize | `GET /auth/google/callback (worm.cash_out_batch.authorize)` | cash_out_batch / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [callback](../../../../internal/googleoidc/worm_position_cash_out_batch_authorization.go) |
| worm.cash_out_batch.authorize | `POST /auth/worm-trading/position-cash-out-batches/{batchId}/solana/verify` | cash_out_batch / AUTHORIZATION | stage, proofKind, expectedRevision, confirmedRevision | [WormPositionCashOutBatchVerify](../../../../internal/phantomauth/worm_position_cash_out_batch_authorization.go) |
| worm.cash_out_batch.authorize | `POST /auth/worm-trading/position-cash-out-batches/{batchId}/development` | cash_out_batch / AUTHORIZATION | stage, proofKind | [developmentWormPositionCashOutBatchAuthorization](../../../../internal/server/worm_position_cash_out_batch_authorization.go) |
| identity.login（仅失败） | `GET /auth/google/login` | account / INITIALIZATION_FAILURE | stage, provider | [Login](../../../../internal/googleoidc/handler.go) |
| identity.login（仅失败） | `POST /auth/phantom/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [Challenge](../../../../internal/phantomauth/handler.go) |
| wallet.reveal.authorize（仅失败） | `GET /auth/wallet-secrets/google` | account / INITIALIZATION_FAILURE | stage, provider | [WalletSecretReauthentication](../../../../internal/googleoidc/wallet_secret_reauth.go) |
| wallet.reveal.authorize（仅失败） | `POST /auth/wallet-secrets/solana/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [WalletSecretChallenge](../../../../internal/phantomauth/wallet_secret_reauth.go) |
| worm.connection.authorize（仅失败） | `GET /auth/worm-trading/google` | account / INITIALIZATION_FAILURE | stage, provider | [WormCredentialReauthentication](../../../../internal/googleoidc/worm_credential_reauth.go) |
| worm.connection.authorize（仅失败） | `POST /auth/worm-trading/solana/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [WormCredentialChallenge](../../../../internal/phantomauth/worm_credential_reauth.go) |
| worm.execution.authorize（仅失败） | `GET /auth/worm-trading/executions/google` | account / INITIALIZATION_FAILURE | stage, provider | [WormExecutionAuthorization](../../../../internal/googleoidc/worm_execution_authorization.go) |
| worm.execution.authorize（仅失败） | `POST /auth/worm-trading/executions/{runId}/solana/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [WormExecutionChallenge](../../../../internal/phantomauth/worm_execution_authorization.go) |
| worm.cash_out.authorize（仅失败） | `GET /auth/worm-trading/position-cash-outs/google` | account / INITIALIZATION_FAILURE | stage, provider | [WormPositionCashOutAuthorization](../../../../internal/googleoidc/worm_position_cash_out_authorization.go) |
| worm.cash_out.authorize（仅失败） | `POST /auth/worm-trading/position-cash-outs/{cashOutId}/solana/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [WormPositionCashOutChallenge](../../../../internal/phantomauth/worm_position_cash_out_authorization.go) |
| worm.cash_out_batch.authorize（仅失败） | `GET /auth/worm-trading/position-cash-out-batches/google` | account / INITIALIZATION_FAILURE | stage, provider | [WormPositionCashOutBatchAuthorization](../../../../internal/googleoidc/worm_position_cash_out_batch_authorization.go) |
| worm.cash_out_batch.authorize（仅失败） | `POST /auth/worm-trading/position-cash-out-batches/{batchId}/solana/challenge` | account / INITIALIZATION_FAILURE | stage, provider | [WormPositionCashOutBatchChallenge](../../../../internal/phantomauth/worm_position_cash_out_batch_authorization.go) |
| identity.login | `POST /auth/registration (login child)` | account / LOGIN | stage, provider | [create](../../../../internal/authregistration/handler.go) |

## 明确排除

| 入口 | 原因 |
| --- | --- |
| `/tradersync.TraderSyncService/ResolveTarget` | 显式目标查询；POST 不等于用户写操作 |
| `POST /api/v1/worm-trading/executions/{runId}:heartbeat` | 执行协调心跳，不是用户主动动作 |
| `POST /api/v1/worm-trading/executions/{runId}:execute-next` | 已授权 Run 的自动推进；原业务执行记录继续保留 |
| `Get*/List*/Version、bootstrap、健康、reflection、日志查询` | 只读／运行协议，不是关键操作 |
| `登录跳转、challenge 生成、注册 GET／用户名可用性、成功的授权初始化` | 辅助步骤；初始化失败按目的动作单独记录 |
| `grpc-gateway proto 解码前拒绝、未匹配路由、资产下载` | 未形成可执行业务命令；不把运行／安全访问日志混入本功能 |

## 对象定位规则

- 已有对象由成功验证的请求 ID 定位；新对象优先使用已确认的响应 ID。创建失败且没有对象 ID 时只显示对象类型和请求摘要，不能生成虚构业务对象。
- 账户管理员操作的目标 UUID 保存到 targetAccountId；操作者仍是当前管理员。账户 API Key 的对象 ID 为公开展示 ID，并同时带所属账户。
- 钱包批量请求上限在当前 proto 中为 10，成功时记录全部返回钱包 ID；返回中的 privateKey 不接触日志编码。
- 轮次以当前／确认后的 slug 为标识，提案／投票以轮次及已确认对象标识定位，不记录分配比例正文、提案内容和投票选择。
- Worm 控制命令保留 run／plan／cashOut／batch ID 与已校验 commandId；自动 heartbeat／execute-next 不记录，用户点击的 reconcile／check-status 记录。
- Token 仅记录当前已注册接口，不开启 Token UI，也不扩展为其独立功能实施。

## 源码核对方式

设计阶段从当前公共 proto 提取所有非 Get/List/Version RPC，要求与本清单和 ResolveTarget 排除项恰好一致；逐项确认 source 存在且 symbol 在实际 Go 文件中定义。HTTP 对照实际注册、handler 和前端调用位置，执行阶段还要用真实请求验证覆盖，不能把静态清单检查当作运行验收。
