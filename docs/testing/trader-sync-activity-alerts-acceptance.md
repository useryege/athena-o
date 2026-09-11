# Trader Sync 指标、故障与容量验收

2026-09-11；实现基点 3b0272924732046df977c9a2486b83d72e5151c8。本页记录限定证据，不是生产 SLO、公开时间或 UI 验收声明。

## 可重复执行的组件链

[acceptance fixture](../../internal/tradersync/acceptance/fixtures_test.go) 用隔离 PostgreSQL，实际 target Resolve/Create、Collector/WSS Session、SourceRPC/VersionVerifier、Projector、普通/摘要 store、Dispatcher/WorkSource、Telegram SDK。HTTP/WSS 仅提供协议响应，业务资格/版本/预算没有测试副本。三文件均有 integration build tag；普通 go test 不连接 DB。测试环境设置 ATHENA_TEST_PG_ADMIN_DSN 后运行：

~~~sh
go test -tags=integration ./internal/tradersync/acceptance -count=1
go test -race -p 2 -tags=integration ./internal/accountstate/... ./internal/notification/... ./internal/tradersync/... ./internal/server/... ./util/telegram/... ./cmd/athena-notification/commands -count=1
~~~

正式运行显式专用 GOPATH、GOPROXY=off、GOSUMDB=off，只用已有缓存；pgtest 随机库退出清理，不重置业务数据库。fixture JSON 导出可设置 ATHENA_TASK13_GATEWAY_FIXTURE_DIR 到每次运行独立目录，避免混淆运行。原协议 fixture 为[source_records.json](../../internal/tradersync/testdata/source_records.json)及其同目录 provenance/runtime：11 recorded + 1 明确 synthetic Combo SELL maker 不变。容量改写钱包、交易/区块定位、header/receipt/状态与时间，始终标为录制 ABI 载荷派生的合成链；不是这些历史交易的真实确认。

最终独立 acceptance 全命令103.437s PASS；相关 integration/race 全命令 exit0（notification116.240s、tradersync210.520s、acceptance100.264s、trader-store121.130s、server10.025s）。最后新增已有 gateway 低频/暂停文案断言单独 race2.806s PASS。没有签名、sendTransaction/sendRawTransaction、范围历史扫描；未知 RPC 或额外 Telegram 方法使 fixture 立即失败。真实 trader Service 未注入交易/签名执行器，不启动交易模块。

## 最终容量和时间

A 为10 owner各10互异 wallet；B 为10 owner共享10 wallet。每场景100关系/100活动/100 sent，每 owner10活动及独立备注/chat；A100 source、B10 source。13活动摘要场景为10 ordinary +3摘要成员、1 part，合计11 logical delivery/attempt/HTTP，分母未被成员联结放大。

以下为最后一次非 race acceptance 真实 runtime DB聚合值，单位秒。UTC 差值不宣称具备跨进程 monotonic 保证；样本没有独立公开区间，因此公开→站内 P95/P99全部不可判定。普通 unknown/default 均保留非 burst 与总体，没有把慢样本剔除。Burst 是形成时严格前件/私聊竞争分类，不能据结果修改，未知分组覆盖率如实列出。

|场景/组|活动数|无ACK|received→recorded P95/P99 UTC|recorded→ACK P95/P99 UTC|
|---|---:|---:|---|---|
|100目标/all|100|0|2.374660 / 2.381668|9.186315 / 9.205664|
|100目标/ordinary_nonburst|61|0|2.367637 / 2.379705|9.178521 / 9.191579|
|100目标/ordinary_default|15|0|2.379150 / 2.382256|9.190989 / 9.194294|
|100目标/ordinary_unclassified|46|0|2.258417 / 2.273401|5.992269 / 6.246975|
|100目标/ordinary_burst|39|0|2.374822 / 2.379260|9.189782 / 9.207952|
|10目标/all|100|0|0.544311 / 0.557145|11.066414 / 11.150631|
|10目标/ordinary_nonburst|51|0|0.516625 / 0.538293|8.104454 / 8.158864|
|10目标/ordinary_default|11|0|0.479711 / 0.487829|2.569680 / 2.911627|
|10目标/ordinary_unclassified|40|0|0.517907 / 0.539598|8.119842 / 8.161244|
|10目标/ordinary_burst|49|0|0.552145 / 0.560869|11.119324 / 11.160125|

最终 A unknown46%、B40%；这部分仍保留普通非burst。队列 snapshot 仅覆盖形成 SQL 时已提交记录，不宣称纳入尚未落库/未提交或内存队列。A/B shared SourceRPC finalized 读取分别4/3，未为每候选独立轮询；metadata 部分不可用重试会真实增加其他确认调用，不能从成本中扣掉。

|场景|raw/关系|RPC chain/headerHash/headerNumber/code/slot/receipt/logs|WSS subscribe/unsubscribe/push|Profile相关HTTP合计|sendMessage|
|---|---|---|---|---:|---:|
|100目标|100/100|1/497/245/431/66/232/33|16/14/100|909|100|
|10目标|10/100|1/47/32/41/6/22/3|2/0/10|911|100|

成本是该有限运行实际协议调用，不是长期计费/吞吐算例。每个 getLogs 均精确已知 blockHash 的升级证据，历史范围调用0；HTTP构造 metadata 与公开资料请求仍计成本。Setup每个关系都实际 Resolve，因此共享wallet场景没有把100次确认卡请求隐去。原始每方法/每选择器计数随 fixture 日志可复现。

429 场景1活动、5 attempts全部保留，最终failed/noACK1，最后观测无ACK年龄15.213751s；timeout场景1活动、1 unknown/noACK，年龄5.076767s，实际 Sender mono5.000275438s。两者仍ordinary_default与总表，成功ACK分位缺失而非0。13活动摘要场景全sent/noACK0，summary oldest→start/60秒miss见实际 runtime；本测试只有一批，不据此证明相邻批次。真实相邻窗口/四源/70parts见下面单边界矩阵及既有 Task11 钟差记录。

## 故障与证据边界

|验证|具体测试入口|范围|
|---|---|---|
|403/null积压但WSS健康，旧epoch成功候选继续|acceptance TestActualConfirmationFailureKeepsWSSAndClosedEpochCandidate|完整本地组件链；真正HTTP失败|
|版本读失败重试、removed、pause新代|acceptance对应 TestActualVersionReadFailureIsRetriedWithoutDiscardingRaw / TestActualRemovedCannotBeClearedByOriginalReceipt / TestActualPauseAndNewGenerationCannotReviveOriginalCandidate|同raw不丢弃/不复活|
|429/五次上限/五秒未知|acceptance TestActualSenderFailuresRetainAllAttemptEvidence|真实SDK/HTTP/PG，attempt全部保留|
|注册ACK/同秒/原始写失败|CollectorCommittedRegistrationPrecedesACK、BaselineBoundary、CollectorPersistenceFailure 测试|真实Session/PG单边界，不称完整发送链|
|baseline/epoch未知提交|store baselines_integration_test.go 三个UnknownCommit/UnknownRollback测试|真实PG提交/读回|
|checkpoint服务端COMMIT成功、客户端回执未知|TestCheckpointServerCommittedButAcknowledgementLost|wire截断真正CommandComplete，独立连接核对原参数时间到微秒，epoch/interval不重复|
|深度重组/不完整receipt|confirmation_test.go、source_rpc_test.go|fake canonical与真实JSON-RPC分别标识|
|撤权/重绑/许可竞争|subscriptions_integration_test.go、notification/store/attempts_integration_test.go|真实account gate与资格墓碑|
|ACK落库失败/缺起点|TestWorkerPersistsPermitBeforeSendAndRetriesOnlyResult、TestWorkerPersistsSenderReturnBeforeLocalStartBookkeeping、SummaryMissingStart|单次HTTP、同permit结果补记，sent缺start不造值|
|摘要长内容/各part混合结果|summary_test.go、store/summaries_integration_test.go|完整成员/无截断/独立结果|
|未来窗口/四源/跨owner|TestSummaryDispatcherMixedSourcesAndNextHead|真实Dispatcher/WorkSource/PG/HTTP的70parts组合，非100活跃实网|
|sender停止确认/长RetryAfter/UTC跳变|notification/recovery_integration_test.go|全部历史预算与monotonic屏障；受控clock不是物理等待120秒|
|提示文案/API|TestTraderSyncGatewayRealCredentialsAndOwnerPrivacy|真实gateway精确usageNotice与queueNotice，非UI|

原110 checkpoint测试在健康前置未成立时没有触发COMMIT截断，14.222s FAIL保留；改为先等待真实合格checkpoint再选下一次，115及140成功。不是用回滚冒充未知成功提交。Task11曾观察UTC与mono差3.772637s，紧窗口失约原事实保留，不能把其负测试通过写成60秒目标达成。

## 观测契约

详细行为见[长期设计](../design/trading/trader-sync-activity-alerts.md#形成证据与时效观测)和[通知恢复/结果](../design/notifications/account-telegram-notifications.md#实际结果时间与恢复进度)。真实 Started mono、Sender返回/worker处理时间分开；begin/pool与advisory分开；数据库事务总时长/冻结/渲染日志在锁外；恢复状态由实际sender进程提供。有限runtime名字不含owner/wallet/note/payload。无ACK/未冻结和各终态年龄保留；sender返回不是服务端或设备ACK。

真实 recovery gateway 原响应包含缺recovery、unknown remaining、合法0/55000string、cancelled/fatal/stopped优先；旧顶层snake_case与新recoverycamelCase保持真实形状。TraderSync runtime样本是真实facade/DB读取，认证principal为明确测试注入，另有既有真实credentials测试。暂存原件在计划scratch，永久保留本页摘要及可运行fixture；临时随机DB由pgtest回收、隔离PG容器由控制器整计划结束清理。

## 真实来源与未完成的实网证明

历史补证37次Chainstack WSS RPC（15成功/22 Archive拒绝）+3连接：8完整头实际geth重算hash一致、8receipt覆盖11recorded；canonicalByNumber和历史code/slot缺证据仍在。当前HTTP前置6RPC+1TCP成功chain/latest/finalized，但当次latest hash code/slot失败，不能泛化永久不支持。正式本次只读实网使用生产 liveFilters/DialSession 的 scratch 观测 overlay（不是原样生产二进制，观测写盘可能影响耗时）。两100地址OR均ACK，其中8历史已见公开钱包、92明确合成补位。20.000380881秒mono观察（UTC22.019882585秒）取得同钱包3条Core推送。首条在真实SourceRPC/ConfirmReceived/VersionVerifier/DecodeOwnTrade通过：tx 0x237b9792266bea7875342aab59869330b9679c5dd0eccc1237311b3a66c793d8，block93596639/hash0x8d1b7d450478409aaf32237031953bd2cfbe96788103fb12a6b74e7f00c7d222，logIndex344；真实ParentHash0x7b536b92c870b1b89632bab3c159e42ab50c9bc80c23461156f795e39571f0f4。Core候选/父hash runtime均21037字节、Keccak0xa08da89bbac2063dfa6a705e70314d218d40fb2b2a6405442297c241fcd58401；版本polygon137-core-v2-ccc0596074f4，BUY本金2300000、份额5000000、fee0。币种/精度来自匹配的固定registry，不声称新取getter证据。

本次13JSON-RPC（WSS5+HTTP8）、16ProfileHTTP，另计WSS握手1与HTTP TCP10，均有界成功，无重试/扫描/provider切换。HTTP body143336字节，WS应用发送14564/接收3913字节，不包含TLS计费。没有DB/活动形成/Telegram，不是100真实活跃吞吐或旧历史缺口解除。NegRisk/Combo本轮0推送，未捕获缺失Combo SELL maker。

两个公开钱包0x31e5d54aded22aa7cd80dbe9e33102abe2504879与0x07db5765beba17be154cbfbd6324a15f3c81fa4f，16HTTP均200；六区间金额全unavailable：五个非ALL区间reference_unknown、ALL金额rounding_unknown。只有ALL曲线available，原始点/HTTP200不能充作官网展示一致性；无新币种、参考时刻、官方舍入规则证明。没有独立公开时间。观测overlay重建的396包/2351输入核对与相同binary SHA保留为临时受控证据，其限度没有覆盖编译器全部传递工具。

真实100活跃目标压力、公开时刻P95/P99、供应商静默漏推完整性和长期稳定性仍待验证。缺失Combo SELL maker recorded不能改名补格。Telegram仍待明确测试接收者授权；本页不冒称已外发成功。

最终类型自审另以真实PG大整数重现前件JSON通用对象的float64舍入：9007199254740993变成9007199254740992。已将SQL显式jsonb生成[]byte并直接typed解析；形成同snapshot测试同时证明大整数/最近未形成前件/不可变cohort。该变更后仅重跑受影响真实消费者的定向race，前述完整矩阵是修复前运行，不混称同一源树。没有改变业务分类或配额。
