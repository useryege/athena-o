# Go lint 分类分析

日期：2026-09-13。承接 [ESLint 修复](eslint-config-matching.md)。下文 1,684 条统计与分类 JSON 是最初只读分析的历史快照，当时未修改 Go 源码、模块、生成文件或 lint 配置。后续首批已修复 `G0554`；当前完整复扫为 1,683 条，新增 0，仅目标诊断消失，见[修复验收](profit-sharing-submit-error.md)。

## 分类时的扫描基线

在 `.worktrees/eslint-go-lint` 的冻结源码上使用 Go 1.27.1、golangci-lint 2.13.2（Go 1.27.1 编译），执行：

```bash
GOTOOLCHAIN=local GOFLAGS=-mod=readonly golangci-lint run \
  --timeout=10m --modules-download-mode=readonly \
  --output.json.path=<证据目录>/go-lint.json \
  --output.text.path=<证据目录>/go-lint.log ./...
```

扫描完整结束，原生退出码1，原因是发现诊断；没有类型加载、分析器崩溃或超时失败。结果为 **1,684条、385个文件**。与第三方依赖修复后的1,684条按文件、规则、诊断文本及重复数量对比，新增0、消失0。保留原配置、默认构建条件和测试分析；没有 `--new`、`--fix` 或规则降级，额外 `integration` 标签不在本次扫描范围。扫描前后802个 Go/模块/配置文件哈希一致。

| 规则 | 数量 | 分类要点 |
| --- | ---: | --- |
| perfsprint | 802 | 格式化简化：error-format 720、string-format 70、integer-format 9、concat-loop 2、bool-format 1；属于候选清理，不是已测得的性能瓶颈。 |
| goimports | 198 | 导入分组和格式。 |
| staticcheck | 177 | 156条错误文案大小写、9条API弃用，其余12条简化/常量建议。 |
| gocritic | 161 | 107条局部变量再赋值建议；另有作用域、条件、HTML引用等需辨别语义的提示。 |
| gofumpt | 110 | 格式。 |
| revive | 61 | 43条未使用参数；另有2条刻意检查单调时钟的时间比较等。 |
| errorlint | 51 | 46条错误包装格式、5条直接错误比较。 |
| unused | 40 | 默认构建没有读取的私有字段或调用的函数，删除前仍须查消费者与初始化副作用。 |
| errcheck | 32 | 10条事务回滚；其余22条来自测试的响应写入、服务启动或解析。 |
| thelper | 18 | `testing.TB` 参数命名，不是缺少 `Helper()` 的诊断。 |
| unparam | 13 | 固定实参/未使用返回值，先核对接口与调用约束。 |
| noctx | 11 | 生产监听、测试HTTP调用、邮件拨号的上下文接口。 |
| usestdlibvars | 4 | HTTP状态常量。 |
| whitespace | 3 | 空白格式。 |
| ineffassign | 2 | 一项实际错误传播缺陷、一项无效初值。 |
| govet | 1 | 已知非nil错误再次判非nil的冗余条件。 |

此外，工具提示 `gomodguard` 已弃用并建议 `gomodguard_v2`；这条配置警告不计入1,684条。Go格式三规则合计311条，分布于262个文件，不应与全部存量诊断混作一次业务修复。

## 首批已修复：分润提案提交错误被遮蔽

历史诊断 `G0554` 的根因为 submit 分支的 `participants, err :=` 新建内层错误变量，提交查询向它赋值，分支外却检查外层变量，绕过 `pgx.ErrNoRows → ErrRevisionConflict` 映射及查询错误包装。

[当前实现](../../internal/profitsharing/store/operations.go#L285)已将参与者列表错误改名为 `listErr`，使提交查询和后续检查使用同一个 `err`。真实 PostgreSQL 故障注入在旧代码上复现了两种后果：零行时返回由零值构造的 proposal；语句异常时只在 Commit 返回事务失败，丢失原查询错误。行锁及前置检查使正常并发下零行不易发生，受控复现不等于已有业务事故。

本次新增真实 SQLStore 集成回归，覆盖上述两种失败及正常提交、重新打开、版本冲突。完整复扫从 1,684 条降为 1,683 条，仅目标 `ineffassign` 消失；其余诊断未修改。测试环境、红绿证据和验收边界见[修复验收记录](profit-sharing-submit-error.md)。历史分类 JSON 与下面各表仍保留原快照，不作为当前未解决数量。

另外三项确定的代码冗余为 `G0553`（server重复判错）、`G0555`（Cash Out providerState初值被覆盖）、`G0215`（当前调用不可能进入的状态改写分支）；它们不等同于三个运行故障。

## 需要定向验证的可靠性项

- `G0073`：Trader Sync关停只接受直接取消错误，包装的纯取消可能被报告成失败。应复用逐叶取消分类，并保留 `errors.Join(取消, 清理故障)` 的真实故障；不能直接改成一次 `errors.Is`。
- `G0051`：Notification当前SQLStore直接返回 stale permit，现有比较可命中；只有错误被包装时才会额外重试。属于条件风险，不能记成已经发生的重复发送。
- `G1497`：API客户端使用不接收请求ctx的 `Transport.Dial`，已有30秒拨号超时；取消时拨号仍可能继续，应单独验证 `DialContext` 的改法。
- `G0011`：finality的defer回滚使用可取消ctx，应核对取消/连接释放契约。其他9处回滚采用后台上下文，成功提交后的 `ErrTxClosed` 是预期结果。
- `G0018/G0029/G0032`：部分负例HTTP夹具只要求调用返回错误，夹具写入失败可能混入预期拒绝，形成条件性假阳性。过大响应中客户端主动关闭可能是正确行为，不能一律把写失败判为测试失败。

这些项来自110条错误处理、运行/逻辑及API诊断的逐项或同调用契约核查；没有把未运行的故障场景写成已通过或已复现。

## 保留语义，避免错误的批量修复

- `G0122`：正则成功匹配固定返回5项，nil检查已经满足后续索引契约，不应随意增加改变接受集合的长度判定。
- `G1375/G1428`：`t == t.Round(0)` 用于识别单调时钟分量，改为 `Equal` 会破坏计时证据。
- `G0071`：取消错误按每个叶子分类是为了保留混合的清理失败，不能用任意叶子匹配替代。
- `G0093/G0094/G0116/G0117`：生成HTML属性，`%q` 使用Go字符串转义，不能视为HTML引用的等价替代。
- `G0102`：Telegram命令的 `ToLower` 比较与 `EqualFold` 接受的Unicode集合不同。本机无外部依赖probe确认 `/ſtart` 在前者不匹配、后者匹配；HTML反斜杠输出也不同。probe只说明建议不是普遍等价转换，不证明当前业务已有异常输入。
- 多处 `%w: %v` 保持一个业务错误身份，同时将底层原因作为文本；全量换 `%w` 会改变 `errors.Is` 可见身份。注册/授权、Worm错误域及提交结果不确定性应保留现有分类，扩展cause链另行决策。
- 156条错误字符串提示包含Google/Phantom等专名和对外文案，不机械小写。未使用字段/参数、短声明建议和字符串简化也要先核对消费者、作用域及精确输出。

## 完整分类及后续顺序

持久化的[逐项分类JSON](evidence/go-lint-triage-20260913.json)将全部1,684个诊断映射到82个处理组，每个ID恰好出现一次，包含原诊断、路径行号、原因、源码证据、建议与验收方式。状态含义如下：

| 状态 | 数量 | 含义 |
| --- | ---: | --- |
| confirmed | 4 | 1项结果风险和3项确定冗余，不能解释为4项业务故障。 |
| conditional | 34 | 需要包装错误、取消、写失败等条件才可能影响行为，尚未全部实测。 |
| intentional | 66 | 当前来源或调用契约支持保留现有语义。 |
| intentional_or_review | 156 | 错误文案/专名需单独检查，不自动修改。 |
| unverified | 107 | API/消费者/参数等尚不足以判定修改收益与边界，暂缓。 |
| cleanup_candidate | 1,317 | 格式或惯用写法候选；不代表每项都已证明等价或可自动修复。 |

分润提案错误传播已作为首批单独修复；后续可分开处理可验证的取消/错误分类、测试可靠性、纯格式、等价写法及API弃用，尚未授权实施。三个telemetry/health启动接口的ctx改造、错误cause公开策略和消费者不明的字段裁剪列为待决，不夹带实施。当前设计不为假设的旧Redis部署保留兼容分支。

## 验证证据与局限

原始本机证据位于 `.tmp/eslint-go-lint/`，目录被Git忽略，清理后需重新扫描；完整分类JSON已随文档保留。

- [原始扫描](../../.tmp/eslint-go-lint/go-lint.log)、[JSON](../../.tmp/eslint-go-lint/go-lint.json)、[退出码](../../.tmp/eslint-go-lint/go-lint-exit-code.txt)、[进程警告](../../.tmp/eslint-go-lint/go-lint-process.log)、[版本与命令](../../.tmp/eslint-go-lint/scan-metadata.json)。
- [历史对比](../../.tmp/eslint-go-lint/go-lint-comparison.json)、[错误处理核查](../../.tmp/eslint-go-lint/triage-errors.md)、[运行/逻辑核查](../../.tmp/eslint-go-lint/triage-runtime.md)、[语义差异probe](../../.tmp/eslint-go-lint/suggestion-semantics.log)。

原分类阶段未重新扫描漏洞，未运行Go行为测试、数据库、Redis、SMTP或开发服务；该历史分析不代替后续修复的失败复现和回归。完成通知是整个已批准任务结束后的单独动作。

独立[分类审查](../../.tmp/eslint-go-lint/task-2-review.md)与[最终交付审查](../../.tmp/eslint-go-lint/final-review.md)均通过；原分类任务回写后9个任务文件与已审查版本一致，当时根目录802个Go/模块/配置文件也与该轮扫描前哈希一致。
