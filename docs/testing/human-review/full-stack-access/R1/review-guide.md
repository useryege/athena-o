# 人工审查指南：十一应用全栈启动与六板块访问控制

- 任务：十一应用全栈启动与六板块访问控制（full-stack-access）
- 交付轮次：R1
- 仓库：`/home/yege/work/athena`；分支：`rf4`
- 被审查代码及长期文档完整版本：`c0016253d927fc764be0f4b70e86a921b209f31c`
- 设计依据：[已整体批准的技术方案](../../../../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)
- AI 验证入口：[完整验收记录](../../../full-stack-access-acceptance.md)
- 配套结果表：[human-report.md](human-report.md)

## 启动前版本核对

先执行以下只读命令。预期祖先检查成功、两个 diff 无输出、最后 untracked 无输出。任何失败都先在报告记受阻，不切分支或 reset 覆盖工作。后续仅交付文档提交不会改变受审查产品代码。

```bash
cd /home/yege/work/athena
TASK_REVIEW_SHA=c0016253d927fc764be0f4b70e86a921b209f31c
git show -s --format='%H %s' "$TASK_REVIEW_SHA"
git merge-base --is-ancestor "$TASK_REVIEW_SHA" HEAD
git diff --exit-code "$TASK_REVIEW_SHA" HEAD -- cmd internal pkg ui hack Makefile go.mod go.sum sqlc.yaml
git diff --exit-code HEAD -- cmd internal pkg ui hack Makefile go.mod go.sum sqlc.yaml
git ls-files --others --exclude-standard -- cmd internal pkg ui hack Makefile go.mod go.sum sqlc.yaml
```

第一条 show 的完整 SHA 应为 `c0016253d927fc764be0f4b70e86a921b209f31c`。本轮所有三份材料使用这一不可变版本；AI真实前缀构建输入与代码提交的哈希绑定见[源码核对](../../../../../.tmp/full-stack-access-20260917/post-commit-source-manifest.json)。

## 环境恢复

需要真实开发环境。AI 已停止全部本任务临时实例；`full-stack-access` 保留数据并带混合开关，不作为首次默认关闭样本。使用以下专用新实例；若同名 state 已存在，请停止首次默认检查并要求明确新的审查实例名，不删除旧数据。

```bash
cd /home/yege/work/athena
export PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH
node --version
test ! -e .run/instances/full-stack-access-review-r1/state.json
make run INSTANCE=full-stack-access-review-r1
```

只有前面的检查都成功再执行 make。保留该终端；另开终端运行 `make runtime-status INSTANCE=full-stack-access-review-r1`。日志、环境和状态位于 `.run/instances/full-stack-access-review-r1/`。十一项ready、双realm bootstrap、实际Chrome smoke分别核对；端口监听不算通过。

前提为本机Docker、项目Go/Node/Yarn、原 `.env` 所列有效开发上游；不切 `.env.prod`。五个远端 Gateway 仅连接和只读健康，不创建/停止/更改。外部端口隔离是已确认部署前提，人工本地检查不扩展为外网扫描。

| 用途 | 身份与入口 | 初始数据 |
| --- | --- | --- |
| 会员 | 本地 disable-auth 的 local-user；http://localhost:4000/ | 新实例默认六项关闭；无真实下单要求 |
| 管理员 | local-admin；http://localhost:4000/admin/service-status | Module Access 管理六设置 |
| 既有AI样本 | full-stack-access 已停止 | TS/Radar/Profit OPEN，Solana/OO/Worm CLOSED；保留原审计，不重置 |

若实际环境不再使用本地 disable-auth，则用具有相应权限的真实登录会话，不能伪造管理员头绕认证；记录身份差异，缺凭据时受阻。

## 检查顺序

除明确写“在ui目录”的命令外，所有命令均从 `/home/yege/work/athena` 执行；离开ui检查后先返回根目录。所有 CHK 均依据本页链接的批准方案和验收记录，按编号逐项回填；同一编号在报告中仅一行。确定性竞态和故障项可复核/运行指定自动化，明确记录方式，不能写成实际交易已完成。

### CHK-001：十一应用及双 realm 身份

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：按前面的新实例启动命令运行。另开终端执行 `make runtime-status INSTANCE=full-stack-access-review-r1`，打开会员 `/` 和管理员 `/admin/`，分别查看 Network 中 `/api/v1/app/bootstrap`。
- 可观察预期：十一项 Startup ready，CoreUsable、SelectedReady、FullStackReady 均 true，无失败；会员 local-user、管理员 local-admin，各自 bootstrap 为 AUTHENTICATED。状态命令退出 0 本身不算通过。
- 记录：保存启动/status 输出、两份 bootstrap 响应及入口截图。
- 影响与恢复：本项只读；本轮末按 CHK-015 停止新实例。

### CHK-002：首次默认关闭与六键范围

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：在全新 review-r1 实例第一次修改前，以 local-admin 打开 `/admin/service-status` 的 Module Access 页签。读取 `/api/v1/module-access-states` 与管理员 `/api/v1/admin/module-access-settings`。
- 可观察预期：恰好 trader_sync、solana、market_radar、managed_oo、profit_sharing、worm 六项且全 CLOSED；首次没有伪造的修改账户/时间。页面明确 Token 延期、关闭不停止后台和通知。已有主验收 full-stack-access 实例是混合设置，不能用于证明首次默认。
- 记录：保存初始页面和两份响应。
- 影响与恢复：本项不修改；不要重置已有实例或删除数据。

### CHK-003：第四页签桌面手机与核心状态独立

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：在1440px和390px窗口依次切换 Services、Notifications、Trader Sync、Module Access；手机收起导航后检查四个标签及选中指示线。保持六业务关闭时刷新前三个页签。
- 可观察预期：标签不重叠、指示线对应选项，六设置行可操作；三个核心状态来源仍可独立查询，业务开关不使核心 Trader Sync runtime 状态不可读。
- 记录：桌面/手机截图和核心状态请求。
- 影响与恢复：无数据修改。

### CHK-004：六业务逐项开放关闭与页面卸载

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：管理员每次只开放一项；会员进入对应导航（Trader Sync、Solana、Market Radar、Managed OO、Profit Sharing、Worm Trading）并确认读取实际业务或正确空态。关闭同一项，在保持前台在线的会员页观察正文卸载；重开后重新读取。逐项完成六项。
- 可观察预期：开放后业务可达；关闭后通常在下一2秒轮询后收起正文，陈旧开放决定最长5秒失效；有权限菜单和URL保留，显示关闭说明，重新开放没有恢复旧草稿。空列表不能当作请求失败。
- 记录：每个板块记录请求与截图、关闭耗时；不要提交交易来验证页面。
- 影响与恢复：将测试用六设置改回全部关闭；修改审计会如实更新，不伪造恢复原时间。

### CHK-005：真实业务 API 关闭错误与核心继续

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：在新审查实例根路径运行已交付脚本：`python3 .tmp/full-stack-access-20260917/live-access.py --base http://localhost:4000 --evidence .tmp/human-full-stack-r1/api cycle`。打开其输出中的各板块 closed-business、Worm wallet-selection 和 core 文件。
- 可观察预期：六业务及Worm原始HTTP返回503、MODULE_ACCESS_CLOSED、athena.module_access、对应module_key、reason header与private/no-store；登录/bootstrap、账户、Wallet、Notification、Service Status、开关管理仍可用。脚本应退出0；此脚本会显式更新本审查实例六个开关。
- 记录：脚本输出及新证据目录，不覆盖AI原始证据。
- 影响与恢复：脚本结束为TS/Radar/Profit开放、Solana/OO/Worm关闭；可在管理员页全部关闭。

### CHK-006：管理权限和原有授权边界

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：复核上一步 member-admin-settings-rejected 与 member-settings-write-rejected 两份真实403。运行 `go test ./internal/server -run 'TestModuleAccessSettingsRealJSONContractAndAPIKeyWrite' -count=1`，阅读本轮Task2审阅与proof绑定回归证据。
- 可观察预期：会员不能读取管理设置或写开关；API Key不能借管理员身份修改设置；管理员/API Key业务请求仍遵循准入。原认证及proof绑定错误先于模块准入，账户权限revision不因开关改变。
- 记录：区分实际HTTP观察与受控自动化复核，保存命令结果；没有真实外部登录凭据的检查如实记录受阻。
- 影响与恢复：本项命令使用受控store；不改真实账户权限。

### CHK-007：失焦离线恢复及迟到响应隔离

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：开放Solana，会员页进入后用Chrome DevTools切Offline，切回Online或隐藏后回到前台，观察必须重新读取状态才能显示正文。运行 `make ui-acceptance UI_ACCEPTANCE_GREP='module-access'` 复核根路径/前缀的离线、隐藏、旧保存响应和六路由用例。
- 可观察预期：离线/隐藏/焦点恢复先失效，旧OPEN响应不能恢复页面；受控filtered浏览器用例全部通过。该命令创建自己的临时harness，不证明真实上游交易。
- 记录：记录实际页面网络时序和新filtered原生报告，明确证据模式。
- 影响与恢复：关闭DevTools离线；命令原生cleanup应passed，保留报告。

### CHK-008：Worm迟到签名与关闭后不自动驱动

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：在ui目录运行 `yarn test --runInBand --runTestsByPath src/app/member/pages/worm-execution-scope.test.tsx src/app/shared/module-access.test.tsx`。阅读 pending Phantom关闭再开放、driver timer取消、portal回调失效的具体断言与本轮Task3证据。
- 可观察预期：关闭后迟到签名不能verify/start；heartbeat、execute-next、排队timer不因重新开放自动恢复；不自动发送pause/terminate，只读权威现状并等待用户明确操作。
- 记录：作为确定性自动化复核填写，保存输出；不把它写成实际链上订单已验证。
- 影响与恢复：无真实订单或钱包签名，无真实业务数据修改。

### CHK-009：保存未知、切页和并行保存竞态

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：在ui目录运行 `yarn test --runInBand --runTestsByPath src/app/admin/pages/module-access-settings.test.tsx`，并查看CHK-007 filtered管理员用例中丢失成功响应、tab return和上一轮迟到PUT。实际页面保存一行时切到其他页签后返回。
- 可观察预期：未知结果显示不确定并重新读取权威设置，不自动重复PUT；返回页签不永久卡Saving；迟到旧保存不能覆盖新状态，并行行各自独立。
- 记录：命令结果、filtered trace和实际切页观察。
- 影响与恢复：保留可信审计；需要恢复值时显式操作页面。

### CHK-010：同实例重启与数据保持

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：将六项设为混合值，保存管理员settings JSON和已有数据标识。执行 `make stop INSTANCE=full-stack-access-review-r1`，核对退出；再用同名INSTANCE `make run`。重新读取settings和原数据。可参照AI两轮数据库OID/检查点比较证据。
- 可观察预期：六值、最后修改人及时间保持，不重置默认；账户与已有数据保留。数据库OID、成功检查点的深入检查可复核AI原始只读快照，不把未手工重测写成亲测。
- 记录：重启前后JSON、日志与复核来源。
- 影响与恢复：同实例保留，不运行run-reset，不删除数据库/卷。

### CHK-011：关闭后台继续和已受理工作

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：保持Solana CLOSED且不再改其审计，用交付的background-snapshot.py对新实例environment.json取两次真实样本，相隔至少60秒，再运行compare-background-snapshots.py。另执行 `go test -race ./internal/server -run '^TestModuleAccessClosingPreservesWorkAlreadyAdmittedOverHTTP$' -count=1`。精确取样命令见后面的后台取样段。
- 可观察预期：CLOSED及审计保持一致，真实成功slot/last_success_at前进，Notification last_poll_at前进；已受理HTTP请求在关闭后释放成功，新请求被503拦截。上游失败导致未推进须如实记录，不能只看started或端口。
- 记录：两份实际样本、比较结果及受控transport输出；不声称新增通知投递或下单成功。
- 影响与恢复：只读采样；无真实订单和发送通知。

### CHK-012：根路径及前缀双 realm Chrome验收

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：根路径运行 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:4000`。停止review-r1后，使用本轮0600 prefix.env副本以同名INSTANCE启动，再对 `http://localhost:4000/athena` 执行相同smoke，并在两入口重复一次GUI Solana开闭。
- 可观察预期：系统Chrome两次smoke与cleanup均passed，会员/admin bootstrap身份正确；/athena页面资源和API代理正常，直接API RootPath与公开BaseHRef不混淆。手机/桌面检查同CHK-003。
- 记录：两份原生报告及前缀GUI截图，标注system Chrome版本。
- 影响与恢复：停止前缀实例；后续恢复根路径时使用原.env，不覆盖它。

### CHK-013：局部最小依赖与external只读

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：复核implementation-evidence的Task4报告、22-resource-audit.json以及selected-schema integration日志：单选Wallet/Managed OO/Profit不需账户库，Solana验证账户+业务schema；external空结构前后catalog不变，健康借用库停止后仍运行。查看对应测试源TestSelectedExternalSchemasDoNotCreateCatalogOrOwnedResources与TestExternalHealthyBorrowerStopLeavesDatabaseRunning。
- 可观察预期：证据有实际PostgreSQL和精确归属，external不DDL/seed、不记录借用库为Owned、不停止它；没有未选库/服务依赖。此项默认复核已有独立故障证据，若要求重测请用新专用实例并记录。
- 记录：写明复核的报告、日志和断言；无法访问证据则受阻。
- 影响与恢复：默认复核无环境修改；不拿现有共享数据库制造缺表故障。

### CHK-014：故障分级与有界停止

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：复核Task5初审/R1复审、37-39 helper deadline证据与Task6 runtime-six-integration-final.log。逐一定位核心失败全收尾、单业务初始失败保留核心、运行期TS真实退出保留另十项、退出0撤销ready、停止后Failures保留、总180秒共享预算及保存runner恢复。
- 可观察预期：每项都有对应真实进程或受控故障证据和退出结果，停止命令非零时能区分历史业务失败与清理失败；不以HTTP200认定UI/双realm ready。
- 记录：逐项注明证据位置、复核结论；无必要不在真实开发栈随意kill进程。
- 影响与恢复：只读证据复核；需要重测时另建专用实例并按其owner清理。

### CHK-015：人工环境按归属收尾

- 前置：完成版本核对；需要页面/API的项目使用本轮新实例及上表身份。
- 操作：从本仓库执行 `make stop INSTANCE=full-stack-access-review-r1`（前缀时可带相同ENV_FILE），随后status并核对state记录的PID身份、所选端口、Owned容器。本轮合计35个实例（含主验收full-stack-access）已经停止，不应再次启动或reset它们。
- 可观察预期：人工实例stopped，所属进程退出、端口释放、自有容器停止，数据卷/日志/证据保留；原七共享容器和五remote Gateway保持原样。停止命令退出不等于已核对，仍须看状态与身份。
- 记录：保存停止输出/状态/容器检查，在报告注明保留资源及原因。如需继续查看，明确留下实例、地址、日志、停止命令。
- 影响与恢复：保留数据；不全局reset，不docker prune，不终止归属不明进程。


## CHK-011 后台只读取样

从根路径环境取样；Solana须保持CLOSED，第二次取样前至少观察60秒。只读取实际扫描状态和通知运行状态；如果尚未有成功样本，先检查日志和上游，不能把空值当成功。

```bash
python3 .tmp/full-stack-access-20260917/background-snapshot.py --base http://localhost:4000 --environment .run/instances/full-stack-access-review-r1/environment.json --out .tmp/human-full-stack-r1/background-before.json
# 等待至少60秒，在此期间不改Solana设置
python3 .tmp/full-stack-access-20260917/background-snapshot.py --base http://localhost:4000 --environment .run/instances/full-stack-access-review-r1/environment.json --out .tmp/human-full-stack-r1/background-after.json
python3 .tmp/full-stack-access-20260917/compare-background-snapshots.py .tmp/human-full-stack-r1/background-before.json .tmp/human-full-stack-r1/background-after.json --out .tmp/human-full-stack-r1/background-result.json
```

## CHK-012 前缀恢复及本轮收尾

确认review-r1已停止后，从根目录用保存的前缀环境副本运行同一实例；make run保持前台运行，smoke在另一个根目录终端执行：

```bash
make run INSTANCE=full-stack-access-review-r1 ENV_FILE=.tmp/full-stack-access-20260917/prefix.env
make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:4000/athena
```

地址为 http://localhost:4000/athena/ 和 http://localhost:4000/athena/admin/。不要同时启动根路径和前缀两套相同端口实例。完成或受阻结束时：

```bash
make stop INSTANCE=full-stack-access-review-r1 ENV_FILE=.tmp/full-stack-access-20260917/prefix.env
make runtime-status INSTANCE=full-stack-access-review-r1
```

从state逐个核对原PID/PGID/start ticks/boot/executable、所选端口、自有容器停止；保留数据卷和报告，不停止七个原项目共享容器。若需AI协助恢复/核对，明确给出该审查实例。需要保留现场时，在报告写明地址、仓库、实例、会话/进程、日志和上述停止命令。

## 历史结果与人工状态

R1首次交付，无上轮人工结果可沿用。AI已修复的实现/浏览器问题记录在验收报告，不能据此把本轮人工检查预填通过。完整独立检查完成或无法继续时提交报告；例外、未执行、失败和受阻状态如实保留。
