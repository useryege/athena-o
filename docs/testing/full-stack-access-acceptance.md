# 十一应用全栈启动与六板块访问控制验收记录

状态：本轮 AI 交付完成，待人工审查。实现、分项与整分支独立审阅、约定真实验收、文档同步、三份 R1 材料与环境收尾均有证据；人工检查全部未执行，不表示人工最终验收通过。

- 仓库：`/home/yege/work/athena`，分支 `rf4`；本地提交，未推送/合并。
- 被审查代码及长期文档完整版本：`c0016253d927fc764be0f4b70e86a921b209f31c`；随后交付文档提交不改变产品输入。
- 批准方案：[2026-09-17 技术方案](../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)。
- 实施计划：[中文计划](../superpowers/plans/2026-09-17-full-stack-access.md)。
- 证据目录：[本轮实际验收](../../.tmp/full-stack-access-20260917/)、[实施与分项审阅归档](../../.tmp/full-stack-access-20260917/implementation-evidence/)。
- 主验收实例：`full-stack-access`；managed 模式，保留同一实例的 PostgreSQL、Redis、MinIO 数据卷。

## 范围与证据边界

默认应用为 UI、API Server、Wallet、Notification、Etherscan Manager、Trader Sync、Solana Discovery、Market Radar、Managed OO、Profit Sharing、Worm Trading。六个访问键为 `trader_sync`、`solana`、`market_radar`、`managed_oo`、`profit_sharing`、`worm`。

访问开关只限制新的用户业务请求。身份权限、服务健康、后台任务、已受理工作和通知生命周期分别验证；不把进程监听或 HTTP 200 当作业务验收。管理员可管理开关不表示绕过业务请求准入。

外部用户统一经过 API Server，内部业务端口由服务器规则隔离，是用户已确认的部署前提。本轮没有远端部署、外网端口扫描或实测服务器规则；未创建、停止或修改五个远端 Etherscan Gateway。没有新增 Market Radar、Managed OO、Profit Sharing 的内部鉴权、Actor 协议或账户连接池。Token 接入继续延期；没有恢复 BSC、Sports 或 Worm Markets。

## 已完成的分项实现与审阅

| 范围 | 提交 | 验证和审阅 |
| --- | --- | --- |
| 独立入口、schema up/verify、服务有界停止 | `912f7cf6`、`2cba99d8` | 实际 PostgreSQL、独立进程 TERM、受理请求与资源关闭顺序；Task 1 修复后审阅通过 |
| 持久开关、生成契约与 API 准入 | `c6e8ee9a`、`3eddf2cc` | 固定六键、可信审计、不改账户 revision、104 RPC 分类、39 受控 RPC、28 原始 HTTP 路径、proof 消费/绑定优先级；Task 2 修复后审阅通过 |
| 双 realm UI、访问新鲜度及异步竞态 | `d57e8a14`、`3dbe9aa3` | 23 suites / 305 tests；审阅修复后相关 6 suites / 50 tests；lint/build；Task 3 审阅通过 |
| 十一服务注册表、按选择准备依赖、配置注入 | `6e7f46de` | 五应用库、external 只读、凭据持久、五 Gateway 只读健康与真实 schema 验证；Task 4 审阅通过 |
| 分阶段就绪、失败隔离及共享停止预算 | `5e479951`、`b867e145` | 107 顶层 race 测试；helper 共享截止时间修复后 5 项定向 race；Task 5 审阅通过 |
| 当前主网 Solana v1 只读 RPC | `69d4c6ba` | 原请求真实失败、协议测试 RED/GREEN、真实 v1 结构及解析断言、Solana 包 race 通过；Task 6 审阅通过 |
| 全栈/浏览器消费者、接受请求与 UI 回归修复 | `6b2db2ec` | 1076 无过滤浏览器、6 项 runtime integration、真实 harness 契约、定向 Jest/lint/类型均通过；Task 6 审阅通过 |
| 长期文档与索引 | `2fad1225`、`c0016253` | 18 份主文档、当前契约与历史边界同步；4 项文档审阅问题修复，范围复审通过 |

分项报告保留原始失败、修复与最终成功记录，不能把中间失败日志误作最终结果，也不能删除失败证据。既有 jsdom navigation/scrollTo 及构建 chunk 提示与最终定向检查的干净输出分别记载在报告中。

## 真实根路径环境

2026-09-17 从目标仓库、项目 Node 24.14.1 执行：

```bash
make run INSTANCE=full-stack-access
make runtime-status INSTANCE=full-stack-access
```

首次运行代码为 `b867e1459a40a074b8723d4c8098aae90ae36c6c`。十一项 `Startup.Status=ready`，`CoreUsable`、`SelectedReady`、`FullStackReady` 均为 true，未记录启动失败；五个配置 Gateway 分别 Health SERVING。见 [首次启动日志](../../.tmp/full-stack-access-20260917/real-run-1.log)、[实际状态与进程身份](../../.tmp/full-stack-access-20260917/real-initial-runtime.json)。

系统 Chrome 149.0.7827.53 smoke 使用真实开发环境、独立会员/管理员会话，退出 0，原生报告结果和工具清理均 passed：[smoke 报告](../../.tmp/athena-ui-acceptance/2026-09-17T09-11-50-962Z-e3a23d54/report.md)。此 smoke 证明双 realm bootstrap 和应用壳；业务证据另列如下。

### 六开关闭环

[94 次真实 API 请求](../../.tmp/full-stack-access-20260917/live-api-root/) 经 UI 代理到实际 API 和业务服务，没有业务响应替身：

1. 全新实例六键首次 CLOSED，最小状态不包含修改审计；管理设置没有伪造修改人和时间。
2. local-admin 逐项显式 OPEN 后，六个只读业务请求均返回实际业务字段；Profit Sharing 另验证不存在轮次的实际 404。
3. 逐项 CLOSED 后返回 HTTP 503、`MODULE_ACCESS_CLOSED`、`athena.module_access`、对应 `module_key`、reason header 与 private/no-store。
4. 每轮确认双 realm bootstrap、账户身份与权限不变，Wallet、Notification、通知 runtime、Service Status 仍可读；Worm 原始 wallet-selection HTTP 同样被关闭拦截。
5. 会员读取管理设置或写开关均为 403。

最终为混合设置：Trader Sync、Market Radar、Profit Sharing 开放；Solana、Managed OO、Worm 关闭。后续真实 GUI 再次修改 Solana，因此持久性基准使用 [GUI 最后设置及审计](../../.tmp/full-stack-access-20260917/live-browser-root/expected-persisted-settings.json)，不使用更早的 API 快照。

### 真实页面操作

[系统 Chrome 业务操作记录](../../.tmp/full-stack-access-20260917/live-browser-root/result.json) 使用两个独立 context，无 HTTP 拦截：管理员第四页签显示六行及 Token/后台说明；Solana 关闭→GUI 开放→真实业务读取→GUI 关闭→重开读取→再次关闭通过。关闭后 1810 ms 显示关闭页，业务正文卸载，菜单和 URL 保留，无 pageerror。trace、请求日志及桌面/手机截图在同一目录。

手机截图复核发现四页签标签拥挤，已修正标签最小宽度和内边距，并补齐标签间距与选中指示线断言；后续真实前缀桌面/手机截图复核通过。首次桌面缩到手机时捕获的短暂侧栏过渡不是稳定页面状态，独立手机 viewport 的截图已另存。

## Solana 实际上游差异与后台连续性

首次后台样本实际失败：起点 slot `447757858` 的区块包含 v1 交易，而旧 `getBlock` / `getTransaction` 固定 `maxSupportedTransactionVersion=0`，节点返回 RPC -32015。扫描未跳过失败区块，成功时间为空，不能算后台验收通过。

按 [Solana 官方版本契约](https://solana.com/developers/cookbook/transactions/versions) 把两类只读请求上限设为整数 1，保留 finalized/jsonParsed/full 和失败不推进的约束。实际相同区块返回 1086 笔交易，其中 87 笔 v1，既有解析器可解析；没有升级发送/签名协议或伪造扫描数据。取样、原始失败、回归和解析结果：[调查证据](../../.tmp/full-stack-access-20260917/solana-v1-investigation/)。

修复后停止并重启同一实例，从原起点和保留的检查点恢复成功扫描。[后台连续性结果](../../.tmp/full-stack-access-20260917/background-continuity-result.json) 对比 09:19:51–09:20:43 UTC 两个真实样本：Solana 设置及审计始终保持 CLOSED，成功提交的 `last_processed_slot` 前进 8，`last_success_at` 前进；Notification started/poller_active 保持 true，`last_poll_at` 前进。没有创建真实交易或发送业务通知来制造证据；此结果证明扫描提交与通知轮询持续。

## 首次重启与数据保留

[开关持久性验证](../../.tmp/full-stack-access-20260917/persistence-restart-1.log) 与 [数据库比较](../../.tmp/full-stack-access-20260917/first-restart-database-comparison.json) 均通过：

- 应用数据库仅 `athena`、`wallet`、`managed_oo`、`profit_sharing`、`worm_trading`；系统 `postgres` 单列，不新增 Token/Temporal 数据库。
- 五个库及业务表 OID 未变，账户 ID/创建时间、六设置与修改审计未变。
- Solana 起点仍为 `447757858`，修复后开始成功推进。修复前尚无成功扫描，因此此轮不冒充“旧成功扫描历史已验证”；已另外保存成功检查点，并在下述最终前缀重启中验证保持。

首轮和修复轮均从同一仓库执行 `make stop INSTANCE=full-stack-access`，日志分别为 [首次停止](../../.tmp/full-stack-access-20260917/stop-for-solana-fix.log)、[第二次停止](../../.tmp/full-stack-access-20260917/stop-before-final-restart.log)。已记录进程身份全部退出，收尾无失败，卷保留；本轮所有测试后及交付前的总资源审计均已通过，详见下文。

## 最终前缀环境与成功检查点保留

在原 `.env` 的本轮 0600 副本中，只调整公开 URL、Google callback URL 与 `ATHENA_SERVER_BASEHREF=/athena/`，保留 API 自身 RootPath；没有修改原 `.env`。从同一仓库执行：

```bash
make run INSTANCE=full-stack-access ENV_FILE=.tmp/full-stack-access-20260917/prefix.env
```

十一应用全部 ready、三个就绪指标均 true，无失败；见 [前缀启动日志](../../.tmp/full-stack-access-20260917/real-run-prefix.log)、[实际进程与容器状态](../../.tmp/full-stack-access-20260917/real-prefix-runtime.json)。启动源码的 1323 项输入哈希及未提交补丁已保存为 [源码清单](../../.tmp/full-stack-access-20260917/final-live-source-manifest.json) 与 [补丁](../../.tmp/full-stack-access-20260917/final-live-source-working-diff.patch)，已与提交 `6b2db2ec8ca5d488a672e68785eea3272af4f50c` [逐项绑定](../../.tmp/full-stack-access-20260917/post-commit-source-manifest.json)，1323 项无变化；不是仅用启动时 HEAD 代替实际构建输入。

- 系统 Chrome 双 realm [前缀 smoke](../../.tmp/athena-ui-acceptance/2026-09-17T09-40-32-838Z-7c104dc8/report.md) 通过，退出 0，工具清理 passed。
- [92 次真实前缀 API 请求](../../.tmp/full-stack-access-20260917/live-api-prefix/) 通过，覆盖同样六开关、核心未拦截、管理员/会员边界与专用关闭错误。
- [实际 GUI 闭环](../../.tmp/full-stack-access-20260917/live-browser-prefix/result.json) 通过：关闭后 1461 ms 卸载业务正文，重开后重新读取，无 pageerror、无 HTTP 拦截。[桌面截图](../../.tmp/full-stack-access-20260917/live-browser-prefix/module-access-desktop.png) 和 [手机截图](../../.tmp/full-stack-access-20260917/live-browser-prefix/module-access-mobile.png) 已逐图复核，四页签、指示线、六设置行可读。
- 新一轮 GUI 写入前，[六设置及完整审计比较](../../.tmp/full-stack-access-20260917/persistence-restart-final/) 和 [最终数据库比较](../../.tmp/full-stack-access-20260917/final-restart-database-comparison.json) 均通过：五数据库/表 OID、账户、审计保留，原扫描起点不变，已有成功检查点与成功时间不回退。
- 本实例最终混合值仍为 Trader Sync、Market Radar、Profit Sharing 开放，Solana、Managed OO、Worm 关闭；精确审计以 [最终 GUI 设置](../../.tmp/full-stack-access-20260917/live-browser-prefix/expected-persisted-settings.json) 为准。

从同一仓库执行 `make stop INSTANCE=full-stack-access ENV_FILE=.tmp/full-stack-access-20260917/prefix.env`，退出 0；持久 supervisor 会话退出 0。停止前后 [身份核对](../../.tmp/full-stack-access-20260917/after-prefix-final-stop.json) 显示实例 stopped、无失败、记录的进程身份全部退出，三个容器已停止，数据卷保留。[停止日志](../../.tmp/full-stack-access-20260917/stop-final-prefix.log) 为空是该成功命令的实际输出，不据此单独判断收尾成功。

## 完整浏览器与运行器集成回归

[最终无过滤原生报告](../../.tmp/athena-ui-acceptance/2026-09-17T09-37-38-715Z-750b4673/report.md) 的 result 和 cleanup 均为 passed，退出 0：根路径 ui-fixtures 528/528、live 10/10；`/athena` ui-fixtures 528/528、live 10/10。构建输入与产物哈希保留在原生 run.json；没有跳过失败测试。

ui-fixtures 使用实际页面和拦截响应；live 使用产品组件、真实会话与临时 PostgreSQL，链、资料和 Telegram 仍是本地替身。两者与前述真实十一应用、系统 Chrome、实际业务 API 和后台证据分别记录。首轮完整测试的两个失败（Worm 旧时钟测试同时使访问租约过期、TS 取消弹窗 Tab 逃逸）保留，调整测试时钟和修复焦点循环后，17 项定向根/前缀与上述全量均通过。

[六项运行器 integration](../../.tmp/full-stack-access-20260917/runtime-six-integration-final.log) 全部通过（114.135s），包括全部 schema 先准备、schema 失败不启动消费者、十一应用就绪后实际 Trader Sync 退出而其余十项保留、业务初始失败保留核心、UI-only 双 realm readiness 与无数据库依赖。外围服务为受控本地依赖，未改远端 Gateway。

[已受理工作测试](../../.tmp/full-stack-access-20260917/module-accepted-work.log) 通过实际 HTTP→gateway→gRPC 拦截器：请求获准后阻塞，管理员关闭，新请求返回专用 503，原请求释放后 200。它不以同步修改内存开关的单元断言代替传输边界。

## 资源总收尾

[最终只读归属审计](../../.tmp/full-stack-access-20260917/final-resource-audit.json) 通过：本轮新增 35 个 runtime state 实例均 stopped、记录进程身份全部退出、所选应用端口释放；39 个本轮自有容器均停止，39 个数据卷保留，无错误。该计数包含故障测试停止实例和独立 schema/Redis 测试容器；不等于启动了 35 份真实业务全栈。

Task 1 借用至后续测试结束的 PostgreSQL 已执行 `docker stop athena-task1-schema-rf4`，退出 0，容器 stopped、卷 `athena-task1-schema-rf4` 存在；Task 2 Redis 与 Task 4/6 各实例停止记录保留。隔离浏览器自己的临时 harness/数据库按原生 cleanup 收尾，证据目录保留。没有删除既有应用数据库或运行器数据卷。

原有七个其他项目容器（mongo、redis、openim-web-front、openim-admin-front、openim-minio、kafka、etcd）仍为原 ID 且 running，本任务未操作。其归属与地址见审计和初始容器清单；五远端 Gateway 继续作为外部依赖，不属于本任务停止对象。

## 实施裁定与适用规则

实施期间依照实际源码作了四项裁定；没有新增历史兼容路径：

1. 审计时间使用可信数据库时间生成的 RFC3339Nano 字符串。既有 gogofast 对 `google.protobuf.Timestamp` 的生成组合不能编译；没有为此升级无关生成器。若格式不一致，会影响 UI 审计显示，已用真实 JSON 和页面消费核对。
2. 三个 HTTP 接口遵循现有 `encoding/json`：字段为 `module_key` / `updated_at` 等 snake_case，状态为 OPEN=1、CLOSED=2；gRPC 符号保留。若按另一种序列化格式消费，会破坏读写，已同步 UI、Swagger 和真实传输测试。
3. 退役功能之后的账户权限集合精确为 `[1,4,8,9,11,12,13]`，含隐藏 Token 权限；修正四个旧测试消费者，没有恢复 Worm Markets。若集合错误，会回退账户权限契约，已验证精确集合和隐藏项。
4. API 直连就绪使用 `ATHENA_SERVER_ROOTPATH`，公开 UI HTML、资源和 bootstrap 代理使用 `BaseHRef`。若混淆会误判前缀就绪，真实 `/athena` 双 realm smoke、API 与 GUI 已验证。

| 规则 | 本轮实际落实与证据 |
| --- | --- |
| [SDS-R1/R3](../developer-guide/service-development-standards.md#sds-r1) | 五独立入口、十一独立应用、按所选最小依赖构建/schema/启动、有界停止；Task 1/4/5 与真实进程记录 |
| [SDS-R2](../developer-guide/service-development-standards.md#sds-r2) | API 保持 facade/认证/准入职责，保留已有内部鉴权；三服务新增鉴权明确排除，不宣称补齐既存差距；Task 2 认证先于准入与真实协议验证 |
| [SDS-R4/R5](../developer-guide/service-development-standards.md#sds-r4) | 持久配置和凭据、故障分级、就绪期限、单一 owner 与总停止预算；Task 4/5/6 故障及资源审计 |
| [SDS-R6](../developer-guide/service-development-standards.md#sds-r6) | 原权限 revision、撤权事务及通知生命周期不改；接受请求与后台继续证据单列，没有跨 RPC 传 pool/transaction |
| [SDS-R7/R8](../developer-guide/service-development-standards.md#sds-r7) | 长期文档、生成接口及 UI/测试消费者同步；本记录区分真实开发环境、隔离 live、fixture 与受控故障；人工结论另行记录 |

## 最终审阅与人工入口

[整分支独立审阅](../../.tmp/full-stack-access-20260917/implementation-evidence/final-review.md) 覆盖 `d230ce03` 至 `c0016253d927fc764be0f4b70e86a921b209f31c`：规格通过、质量 Approved，0 Critical、0 Important、1 非阻断 Minor，无需产品修复波次。所有分项及修复复审也已通过。最终 [1323 项构建输入比对](../../.tmp/full-stack-access-20260917/delivery-source-manifest.json) 与真实前缀验收时一致。

保留的 FR-M1 是浏览器迟到 PUT 测试可增加刷新前断言的建议。实际生命周期保护及四种迟到成功/失败组合的直接 Jest 已覆盖行为；当前没有确认的产品缺陷或必需验收缺口。已有 jsdom、可选 CLI 和 chunk 输出说明保留，不宣称全部日志无噪声。

R1 三份材料按人工报告→指南→AI交付顺序创建并读回，使用同一完整版本和 CHK-001 至 CHK-015：

- [AI 交付报告](human-review/full-stack-access/R1/ai-delivery.md)
- [人工审查指南](human-review/full-stack-access/R1/review-guide.md)
- [可回填人工报告](human-review/full-stack-access/R1/human-report.md)

未完成项是用户的人工审查与最终确认，15 项初始状态均为“未执行”。指南提供新实例恢复、准确版本核对、实际操作、受控证据复核及停止方式；不把保留数据的旧实例当作首次默认关闭样本。原始失败、修改裁定、各版测试与资源证据均保留在本轮目录；临时 SDD 工作区的同路径证据归档关系见 [归档说明](../../.tmp/full-stack-access-20260917/implementation-evidence/archive-map.md)。

任务结束通知遵循固定通用文案，仅在本地保留 [通知结果](../../.tmp/full-stack-access-20260917/task-notification.json)；通知不表示人工通过，也不含任务细节。
