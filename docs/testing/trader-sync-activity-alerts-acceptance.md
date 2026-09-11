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

真实100活跃目标压力、公开时刻P95/P99、供应商静默漏推完整性和长期稳定性仍待验证。缺失Combo SELL maker recorded不能改名补格。Telegram 已于 2026-09-11 按明确授权完成下述单条直接 Sender 传输；不外推全实网数据库链或 SLO。

最终类型自审另以真实PG大整数重现前件JSON通用对象的float64舍入：9007199254740993变成9007199254740992。已将SQL显式jsonb生成[]byte并直接typed解析；形成同snapshot测试同时证明大整数/最近未形成前件/不可变cohort。该变更后仅重跑受影响真实消费者的定向race，前述完整矩阵是修复前运行，不混称同一源树。没有改变业务分类或配额。


## Task13 fix1：聚合边界与首次确认

修复基点 27b2caa0408aae97a73b4bd6dbe56f8142a0b469。管理员聚合 SQL 显式读取关联键、状态、时间与安全 gate/processing 标量；不选择 delivery/activity/attempt 全行或正文、备注、交易原件。测试同时检查源 SQL、生成 SQL 与真实 PG 三层分母。worker 结果 UTC 等待按 usable/clock_anomalies/missing 对账全部 attempt；摘要 oldest→firstStarted 与相邻批次同样区分，首批另列 no_predecessor，未冻结成员单列 waiting_members，不伪造批次起点。

Projector 累计指标为 kind=epoch，serviceEpoch 是本对象稳定 UUID；重连与同对象再次 Run 不重置，新对象归零并换 ID；in-flight 仍 gauge。recovery Swagger 与真实 JSON 的 startedAt/remainingMillis/elapsedMillis/clockSource 一致，保留旧顶层 snake_case。

首次 finality 是同一 Projector 时钟实例对该 source 连续可证的首次调用到首次 confirmed，不代表所有并行进程全局最早确认。源表 typed finality_timing 保存真实调用前/返回后 UTC 与同源 mono；完成区间不因后续版本/资料失败或重试重新计长。首次轮耗时、首次起点→首次完成及首轮结束→首次完成各有可用分母和 P95/P99，按 source 计，不按活动/owner 乘算。等待年龄只在当前匹配且有效时钟下计算。

启动只读取已提交 max(source.id) 截点；此前缺证据、跨实例未完成序列和任何丢失观察均保守 unavailable。一次写入失败会使该实例所有未完成观测失去连续资格，不能用后轮 confirmed 补称首次；已完成历史仍保留。实际确认结果先进入原 channel，metadata deadline 使用原返回时刻；随后在原 source worker 中最多五秒写库并由 workers.Wait 收尾。无新 leader/全局串行器或重试算法；每轮多一次 Begin、source 行锁读、更新和 Commit，失败/未知提交不重新 RPC，其耗时包括在 source_round 与总体中。

定向测试包括纯首次/跨 clock reducer、真实 Projector 两轮及版本失败顺序、已 waiting 丢失 confirmed、同实例再次 Run、截点失败和实际五秒取消/join；真实 PG 持久/幂等/大整数与安全聚合；实际普通/摘要及 403/null 恢复 pipeline→facade→gateway。最终 scoped race：tradersync 6.152s、activity 1.012s、store 8.866s、acceptance 35.630s、notification 5.066s、server 9.351s，exit0。未重跑未变容量和全部故障矩阵；profile info/recovery warning 仍存在，未全部断言（M1 deferred）。原静态扫描中间失败是越过生成 SELECT 扫到说明注释中的 wallet，修正扫描边界后通过，未删合法业务读取。

## 单条真实 Telegram 发送

用户明确授权固定三行纯文本、本人私聊 8815996650、一次且不重试；Bot 为 @test_bot_athena_bot（8945962939，此 ID 不是收件者）。实际 util Telegram + NewTelegramSender 五秒调用一次，1 个 HTTP 请求/1 次连接，HTTP200，messageID=4。Started 为 2026-09-11T04:57:39.307927395Z，返回为 04:57:40.031407655Z，原 Started mono 到返回 0.723480265 秒。准确授权/正文、原请求响应及 SHA 见[固定发送证据](evidence/trader-sync-telegram-send.json)。

这是直接 Sender 公网传输；没有 worker/PG，故 Outcome.Timing 两字段为 null，本次时间由探针在真实 Started 和 Send 返回旁路采样。结果只证明 Telegram 成功响应与本地返回，不证明用户设备送达、实际来源到消息的端到端 SLO或全实网持久链。未执行第二次发送或消费 updates。

## Task20：真实产品浏览器验收

2026-09-11；基点 `1f68ecfe57048d6d80acd2ee4982a05fcabaed51`。使用现有系统 Chrome / Playwright，通过双 tag 隔离入口加载实际 member/admin 构建、真实 bootstrap/realm/cookie/gateway、权限控制器、Trader Sync、Notification、PostgreSQL 和本地 HTTP/WSS/Telegram。没有启动正式交易模块，没有新增公网 Telegram 发送或消费公网 updates。

|部署|受控页面 fixture|真实链 live|原始运行目录（`.superpowers/trader-sync-acceptance/` 下）|
|---|---:|---:|---|
|根路径|33 PASS|10 PASS|`task20-root-fixtures-final5`、`task20-root-live-final4`|
|`/athena`|33 PASS|10 PASS|`task20-athena-fixtures-final`、`task20-athena-live-final`|

fixture33项中包含循环矩阵，不能把每张截图计作独立测试。Home、Add、Subscriptions、Subscription、Activity、Summary、管理员列表/详情/Service Status 均实际检查1440/1280/900/390宽度、两主题及文档无横向溢出。主题用实际 `html[data-theme]`、color-scheme和shell背景确认。触屏Combo、Tab弹窗循环、Escape焦点返回、原始数字复制、20 emoji备注、选择文字后实际五秒轮询保持选区、长50行→1行尾页→Previous的实际scrollY，以及摘要两套分页/返回来源均有浏览器断言。视觉抽查覆盖会员Activity既有截图和修后390深色Add、390浅色管理员列表；这不是每张截图的人工逐像素审查。

live通过三个独立cookie上下文实际完成同钱包A/B独立订阅和备注、Resolve/Create、来源形成activity、暂停/恢复、编辑备注/确认取消、绑定Bot update与草稿往返、撤权清正文/开放弹窗/旧读屏障、重新授权不自动恢复、空页及51条历史页的新活动提示。跨owner资源ID/batch为404；签名游标在错误owner上下文为400，正文固定 `invalid cursor or cursor context`；同owner同上下文尾页正常读取。管理员不能调用会员读取接口，也没有备注和活动正文入口。

创建响应故障发生在真实服务已提交之后：HTTP200正文只发送`{`，保留原Content-Length后断开。最终浏览器记录首次200头、`ERR_CONTENT_LENGTH_MISMATCH`、手动恢复的第二次200；同requestId重取原数据库ID，订阅数不增加。服务端确认token过期由隔离数据库控制过期；绑定离开期间草稿到期由浏览器Date固定到原服务端expiresAt之后验证，均不声称物理等待五分钟。Notification启动恢复屏障按实际runtime达到running后继续，未压缩其60秒安全时间。

普通failed/unknown/sent与摘要结果走真实Dispatcher/SDK/本地HTTP；sent缺started来自永久真实API样本的fixture展示。101 parts按实际pageSize50分页为50+50+1；mixed终态没有变回成功，all-sent用独立batch。51条订阅历史、101 Combo、极小金额及超大ID/金额为明确合成派生；原录制ABI/Task12、13 API原件不修改，不冒充实链事件或公网时效。

管理员概要由只读SQL独立计数校验；运行时6个delivery gauge与去重delivery ID的只读SQL一致，两次真实API中10个累计metric的serviceEpoch保持相同。SQL、两次API及比较值保存在`task20-root-final-admin-sql/audit.json`和`task20-athena-admin-sql/audit.json`。进程epoch不是数据库计时窗口，window展示使用明确合成样本，没有宣称SQL证明进程累计值或真实window/SLO。

浏览器验收发现并修复两类产品问题：订阅取消Modal立即卸载导致Escape后焦点不回触发按钮，改为保留关闭生命周期并仍在scope失效时移除；新增辅助文字、placeholder、选中按钮、危险按钮和管理员新状态标签的对比度不足，使用现有主题token及Trader Sync局部选择器修正。原始RED保留在`task20-root-live-r7`、`task20-root-fixtures-r10`、`task20-aa-stable-red`、`task20-aa-add-modal-red`、`task20-root-fixtures-final2`；最终两前缀通过同一产品构建。

AA证据包含新增页面的实际文字/placeholder、前景、祖先背景合成、未舍入ratio及阈值；普通文字4.5、大字3，取消了早期0.01容差。示例：Add辅助文字浅/深5.4254/6.9683，Current选中4.5261/6.2145，Monitoring浅色标签15.9990。相同token颜色角色复用，未声称全站无障碍认证。disabled控件单列`exempt`，加载过渡等待真实记录/动画结束；旧Services SERVING、既有Notification Yes/Active低对比样本仍保留为`outOfScope`，没有用disabled规则豁免或全局修改StatusTag。

可复现入口：先`yarn --cwd ui build`，再设置`ATHENA_TEST_PG_ADMIN_DSN`、`ATHENA_UI_DIST=$PWD/ui/dist/app`、独立`ATHENA_UI_E2E_DIR`及空或`/athena`的`ATHENA_UI_E2E_PATH_PREFIX`，执行：

~~~sh
go test -v -tags=integration,uiharness ./internal/tradersync/acceptance -run '^TestUIHarness$' -count=1 -timeout=30m
~~~

读取该目录`harness.json`，显式设置其中BaseURL为`ATHENA_UI_E2E_BASE_URL`、manifest绝对路径为`ATHENA_UI_E2E_MANIFEST`、PathPrefix为`ATHENA_UI_E2E_PATH_PREFIX`，分别执行`yarn --cwd ui test:e2e --project=ui-fixtures --output=<本轮独立绝对目录>`和`--project=live`。live十项顺序组成一次新库场景；重跑整套应新建harness。禁止运行期间重build资产；两份输入HTML必须与Go embed逐字相同。结束写入目录下`stop`并等待go test退出。没有显式目标时Playwright立即失败；普通integration构建不包含交互式TestUIHarness。

已验收产品build6.88秒、相关169项UI测试、产品/e2e TypeScript、adapter资产/base/meta测试均通过；未重复Task13完整容量或Task21全套race。11份本任务manifest对应随机库均已消失、HTTP/control端点均关闭，记录的10个UI_READY PID均不再存在；专用PG容器归整计划控制器处理。原始失败、每轮run/input SHA、截图/trace、清理核对位于上述scratch目录，详细索引见Task20报告。早期历史命令元数据与输入SHA覆盖不完整，未回填冒充事前记录；最终静态资产和提交文件另有完整hash清单。全部服务停止后，用补全未跟踪CSS输入的清单再build一次（4.20秒），219个dist资产路径/原字节SHA与浏览器已验收产物完全一致；见task20-dist-equivalence.json，历史输入清单未回填。
