# Go lint 后续分批修复方案

状态：首批分润提案错误传播修复已获用户批准并实施，见[验收记录](../../testing/profit-sharing-submit-error.md)；随后批次仍待各自确认，未实施。依据为[分类报告](../../testing/go-lint-triage.md)与[完整诊断清单](../../testing/evidence/go-lint-triage-20260913.json)；1,684是本轮快照，不是固定验收阈值。

## 首批（已实施）：仅修复分润提案的错误传播

**目标：** 提交提案的查询失败必须进入现有错误处理，零行映射为 `ErrRevisionConflict`，普通查询错误保留cause；正常提交与重新打开行为保持不变。

**范围：** 修改 `internal/profitsharing/store/operations.go`，新增同目录 `operations_integration_test.go`；补充本批验收记录。无公共API、数据库结构、sqlc查询或生成文件变化，不夹带其他lint清理。

### 1. 先建立真实写路径的失败回归

- 测试使用 `//go:build integration` 和现有 `internal/testutil/pgtest`；调用 `pgtest.New(t, Migrations(), "migrations")`，通过 `NewSQLStore(db.Pool)` 操作。不能用 `NewSQLStoreWithQuerier` 或复制业务分支的假实现替代事务路径。
- 准备任务专属本地PostgreSQL，临时端口只绑定loopback，设置 `ATHENA_TEST_PG_ADMIN_DSN` 且 `sslmode=disable`；`pgtest` 为每例创建并清理独立数据库。仅清理本批创建的容器/数据库，不使用业务数据库或删除现有数据卷。
- 每个fixture创建一个 collecting round、5个有效UUID participant、一个revision=1的draft proposal和5条完整item；每条责任非空、份额2,000，总计10,000。作者为其中一个participant。
- `TestSQLStoreSubmitProposalNoRows`：在测试数据库的proposal表安装仅对 `NEW.status='submitted'` 生效的BEFORE UPDATE行触发器，返回NULL。实际提交必须返回nil proposal和 `errors.Is(err, ErrRevisionConflict)`；数据库状态、revision和items不变。旧实现应在这个断言失败，保留红日志。
- `TestSQLStoreSubmitProposalQueryError`：另一独立fixture的触发器抛出 `P0001`、固定消息 `profit-sharing submit probe`。要求nil proposal，`errors.As` 可取得原 `*pgconn.PgError`，错误含提交状态上下文，数据保持draft；不能只断言非nil错误，因为旧实现可能仅在Commit时失败。

### 2. 最小实现

将submit分支读取参与者的局部错误改名，避免遮蔽后面需要传播的错误：

```go
participants, listErr := queries.ListProfitSharingParticipants(ctx, round.ID)
if listErr != nil {
    return fmt.Errorf("list participants for profit sharing round %q: %w", slug, listErr)
}
```

其余列表读取、`changed, err = queries.SubmitProfitSharingProposal(...)`、分支外的ErrNoRows映射及withTx逻辑保留。无需为这条修复扩展SQLStore接口或改写事务框架。

### 3. 正常路径与验收

- `TestSQLStoreSubmitProposalSuccess`：无故障触发器时提交成功，返回正确proposal和5条items；状态submitted、revision递增、submitted_at设置。
- `TestSQLStoreReopenProposalSuccess`：对已提交fixture重新打开，状态draft、revision递增、submitted_at清空，items保持。
- `TestSQLStoreSubmitProposalRevisionConflict`：revision不匹配仍返回原冲突错误，状态不变。
- 先确认任务专属数据库就绪，再执行 `go test -tags=integration -count=1 ./internal/profitsharing/store -run 'TestSQLStore(Submit|Reopen)Proposal'`，要求全部执行且通过，无跳过；再执行 `go test ./internal/profitsharing/...`。
- 对受影响代码格式和lint检查，再用原配置完整复扫并按文件/规则/文本比较；要求G0554消失、无未解释的新诊断，不要求其他存量清零。运行必要的独立审查和差异检查，记录数据库归属与清理结果。

## 随后批次：分开规划和验收

以下为明确的分组边界与处理方向；每批实施前先用其失败场景锁定实际收益，业务决策项不混入低风险清理。

| 顺序 | 诊断及根因 | 处理方式 | 验收 |
| --- | --- | --- | --- |
| 2：取消/错误分类 | G0073只识别直接取消；G0051只识别直接stale permit；G1497拨号不接受请求ctx。 | Trader Sync复用逐叶纯取消分类并保留混合故障；stale permit采用 `errors.Is`；API客户端同一net.Dialer改用 `DialContext`，保留超时和代理设置。每个子项先形成真实调用点的失败测试再改。 | 包装/直接取消、混合Join故障；包装stale不重试与普通错误仍重试；受控慢拨号取消；相关包测试，无网络外部依赖。 |
| 3：测试可靠性 | G0001、G0005、G0006、G0014–G0032的测试写入/服务启动/解析诊断，以及G0560–G0564的测试上下文。 | 明确报告夹具写入、解析项数及服务启动错误；测试请求绑定测试ctx。负例断言预期拒绝原因，过大响应中主动断连按原设计处理。 | 让夹具故障不能冒充业务拒绝；定向测试及已拥有资源清理。JSON解码错误即使已有值断言也应直接报告。 |
| 4：确定冗余与格式 | G0553、G0555、G0215；goimports/gofumpt/whitespace共311条。 | 三项局部冗余单独小批；格式仅处理扫描命中的262个手写文件，按模块分批，不全仓随手重写。保留指令、注释和表达式语义。 | server错误路径、Cash Out成功/失败/状态转换；格式差异与受影响包验证，再与完整lint基线比较。真实E2E前置服务按所需场景准备，不因格式任务运行外部限流探针。 |
| 5：等价写法候选 | perfsprint、gocritic等大批建议；部分可影响作用域或输出。 | 先做无格式参数的纯字面错误、整数/布尔输出、常量等可证明等价项，再按子规则处理；错误文案内容不改，短声明需查外层变量/defer，接口参数不得自动删除。 | 精确输出、边界值、作用域/错误传播回归；分批lint新增为0，不以自动修复成功代替验证。 |
| 6：API/工具弃用 | gomodguard配置；G1432/G1468 Redis NX；G1456重复Close；G1458–G1460测试gRPC；G1501/G1502 Redis范围查询。 | gomodguard_v2迁移保留三项模块禁止规则和替代建议；Redis改用现有版本推荐接口并保留Nil/TTL/区间语义；gRPC按Canceled状态处理重复关闭，测试客户端改NewClient并验证懒连接。 | 配置校验及三项禁止规则反例；NX竞争/TTL/错误、范围边界/同分/负序号；首次和重复Close、连接与资源清理。工具和依赖版本不为清lint整体升级。 |

## 保留与需另行决定的项目

- 保留G0122的正则返回长度契约、G1375/G1428的单调时钟比较、G0071逐叶取消分类，以及HTML引用和当前命令接受集合。若以后要求清除其提示，用有依据的局部说明，不全局关闭规则。
- 156条错误文案、业务哨兵翻译中的底层cause、提交结果未知时的多错误公开策略，涉及对外语义。默认保留；若要求改变，先确认错误身份与状态码契约。
- G0011取消时回滚先验证连接释放与主错误保留；三个telemetry/health `Start()` 的ctx贯穿和邮件拨号的ctx接口，需明确是否要求新增启动取消能力。默认暂缓接口改造，不能仅换Background后宣称增加了取消能力。
- 未使用字段/参数先核对默认构建以外的消费者、接口约束与初始化副作用；不根据单次静态报告直接裁剪业务模型。当前Redis设计以仓库实际配置为准，不引入历史兼容分支。

首批的实施、真实数据库回归及 lint 对比结果见[验收记录](../../testing/profit-sharing-submit-error.md)。其他批次继续按分类独立规划和验收，不因首批获批而自动扩大修改范围。
