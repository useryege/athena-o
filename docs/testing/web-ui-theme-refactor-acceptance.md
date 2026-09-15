# 全站前端主题重构实施与验收

正式实现与 T9 约定验收已完成，T1–T9 独立任务审阅均通过。生产版本为 `20167913b4384559e60740b75480592a33c08179`；T10 只同步长期文档。最终独立审阅、环境收尾和通知尚待控制代理执行，准确状态统一在本文[交付状态与环境收尾](#交付状态与环境收尾)。

本报告区分设计批准、正式实现、受控页面、真实读取和外部未验证项。[最终证据入口](../../.tmp/ui-theme-refactor/task-9/final-delivery-20167913.json)、[T9 完整实施报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-9-report.md)及[独立 T9 审阅](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-9-review.md)保留原始结果与修正过程；阶段中的待办描述属于当时状态，不覆盖本报告的最终实施结论。

## 实施范围与当前契约

工作区 `/home/yege/work/athena/.worktrees/ui-theme-refactor`，分支 `codex/ui-theme-refactor`，起点 `bf752f27`。本任务未 push／merge，其他工作区的未提交内容不纳入本次文档更新。

| 范围 | 实际结果与归属 |
| --- | --- |
| 37 条改版入口 | 身份／账户／Help、管理员账户／运维／通知、Wallets／Solana／两端 Profit Sharing、Market Radar／Sports／Managed OO、Worm 资产／组合／执行；[51 项机器清单](../requirements/web-ui/theme-refactor-coverage.json)逐条指向 case、原始报告和 JSON pointer |
| 共享管理员注册 | 与会员共用 `/register`，按 `athenaRealm=admin` 单列场景；37 条入口加此变体为 38 个主场景，不新增一条业务路由 |
| 2 条 Appearance | `/account/appearance` 和 `/admin/account/appearance` 已删除，进入各 realm 既有 404，无兼容重定向 |
| 4 条路由规则 | 两端默认入口与未知兜底均验证，保留角色和 realm 分流 |
| 8 条 Trader Sync | 会员首页／添加／订阅列表／订阅详情／活动／摘要，管理员订阅列表／详情保留专属布局；共享颜色、字体、壳和原有行为已回归。Service Status 的 Trader Sync 页签已按 v16 重排 |
| Token／Nansen | 共享 T1／T2 依赖完成；独立功能接入及导航未开启 |

两份 HTML 和入口 Provider 始终深色；颜色、字体及字号放大使用共用 token／Ant rem 桥接。Inter 用于英文、正文与数字，JetBrains Mono 用于完整地址／哈希；本地字体及许可直接来自批准资产。两 realm 独立路由、服务、身份代际与持久键保持。

主题专用数据库、SQL／sqlc、proto、HTTP／OpenAPI、UserInfo／前端模型及 Appearance 已端到端清理；profile CAS、access revision、session generation 保留。浏览器偏好只保留 version、pageSizes、sortOptions、hideBannerContent、hideSidebar、position，不读写或同步 theme。聚合头像 HTTP 消费者遗漏已修复并通过完整构建；验收使用独立新 schema 数据库，未重置原开发数据。

实施中经证据确认的修复包括：API Key／Profit Sharing 既有手写 access JSON 双向投影遗漏、首次读取失败误显示成功空态、陈旧结果与跨身份回调、Worm 草稿及操作门槛、Help ReDoc 本地资源、控件命名／标签／字体／文字边界及选中图形对比。保留真实 wire 合同的合法 proto3 false／0 省略语义，不一律解释为 Unknown 或把缺失金额变零；详见 [T6 合同勘误](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-6-review-contract-addendum.md)。

## 有版本的正式验证

最后提交只修改三份生产主题／样式文件的 Close 定位和 Checkbox／Radio／Switch 选中图形，另有对应测试。完整基线与后续受影响范围复验共同构成最终证据；没有在 `20167913` 重新执行全部 1018／176／38／56 项。

| 代码版本／模式 | 实际结果 | 原始证据 |
| --- | --- | --- |
| `e7ec272b` 无过滤 acceptance | root／`/athena` fixtures 与 live 合计 1018 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed | [完整原始清单](../../.tmp/ui-theme-refactor/t9-final-official-runs-e7ec272b.json)，run `a0cfd851`；`filtered=false`、grep 为空 |
| `82f25ad8` 语义差量 acceptance | 两部署 19+19=38 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed | [82 清单](../../.tmp/ui-theme-refactor/t9-official-82f25ad8.json)，run `4b8b17b6`；`filtered=true`，`theme:semantics\|theme:read-states`，不含 live |
| `82f25ad8` 无过滤 a11y | 两部署 88+88=176 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed；原始 axe 违规 0，84 条 color-contrast incomplete 保留 | 同上，run `04d4d74b`；`filtered=false`、grep 为空 |
| `20167913` 控件／Close acceptance | 两部署 7+7=14 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed | [最终原始清单](../../.tmp/ui-theme-refactor/t9-final-official-20167913.json)，run `6d2023f5`；`filtered=true` |
| `20167913` 受影响 a11y | 两部署 12+12=24 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed；原始 axe 违规 0，14 条 color-contrast incomplete 保留 | 同上，run `8bacc9c1`；`filtered=true`，12 个场景与完整 grep 见原始清单 |
| `20167913` 真实 smoke | 会员／管理员 2 passed，0 failed／skipped／flaky，退出 0，run／cleanup passed；系统 Google Chrome 149.0.7827.53 | 同上，run `14599154`；`mode=smoke`，正确实例 UI 34000 |
| `20167913` lint／Jest／build | lint 含 12 配置测试、tsc、eslint 通过；37 suites／407 tests passed；Vite build 通过 | [lint](../../.tmp/ui-theme-refactor/task-9/control-lint-final.log)、[Jest](../../.tmp/ui-theme-refactor/task-9/control-jest-final.log)、[build](../../.tmp/ui-theme-refactor/task-9/control-build-final.log) |
| 后端／生成消费合同基线 | 54 tests/subtests passed，0 failed／skipped，另 3 包无测试；40 个相关源 hash 与复验一致 | [后端复验](../../.tmp/ui-theme-refactor/t9-backend-after-recovery/results.json)、[82 父核对](../../.tmp/ui-theme-refactor/t9-parent-verified-82f25ad8-summary.json)；201 无 Go／proto／生成合同变化 |

完整模式使用 `env -u UI_ACCEPTANCE_GREP make ui-acceptance` 和 `env -u UI_ACCEPTANCE_GREP make ui-a11y`。真实 smoke 的入口是 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://127.0.0.1:34000`，不是 `MODE`／`BASE_URL`。所有正式清单保留原始命令、mode、suite、grep、filtered、匹配数量和 cleanup；早期旧 schema 缺少的标记保持 null，不推断为无过滤。

## 视觉、状态与无障碍证据

[最终逐页面索引](../../.tmp/ui-theme-refactor/task-9/route-visual-review-20167913.json)覆盖 38 主场景及 76 张批准 desktop／mobile 图。四条件为 1440×900、390×844、320×844、720×1000 且 root32；每页先核实际字体，再看边界。批准图与正式 React 有人工对应，不使用不同 DOM 的像素差充当验收门槛。e7 主矩阵、82 新 152 项主图及 201 受影响差量分别标注，不覆盖历史图片。

[状态索引](../../.tmp/ui-theme-refactor/task-9/state-coverage-20167913.json)逐条列 S1–S14、2 条 Appearance／4 条路由规则／8 条 Trader Sync、82 语义差量和 201 控件结果。每条记录给出原始 results.json、深度优先 spec 索引、标题和状态；覆盖清单的 JSON pointer 提供稳定入口。

原生 200% 使用 `chrome.tabs.setZoom(2)`：82 版本 35+3=38 主场景通过、76 张截图为真实物理宽 1440，layout 1440→720、DPR 1→2、root 仍 16px。它与 root32 文字放大不同。最终 201 另验选择弹窗 1 项，鼠标及 Enter 关闭均成功，完整 Close 44×44 CSS px 落在 720×406.5 dialog 内；[原生结果](../../.tmp/ui-theme-refactor/native-selection-20167913-normal-motion/results.json)及[父最终核对](../../.tmp/ui-theme-refactor/t9-parent-final-20167913-summary.json)记录实际看图和 context 关闭。root32 的 88×88 目标另外由控件差量验证。

axe incomplete 仍是人工复核项。旧命名／角色、重复 label、空表头、链接和 Steps 低对比问题已修正；82 的 84 条文本对比 incomplete 与最终局部 14 条均保留 raw、目标和逐节点实测。[人工分类](../../.tmp/ui-theme-refactor/task-9/axe-manual-classification-20167913.json)记录 82 的 180 个可测节点、最低祖先合成对比约 6.45969，以及 201 的 38 节点、最低约 7.21880。结合真实背景、透明 wrapper、动画和 modal 遮罩的人工复核解释，未改写为 axe 自动 passes。

Checkbox／Radio／Switch 非文字标记由三个实际业务消费者及真实 Ant harness 独立验证：正常／focus 对比约 15.14548，hover 约 15.69918；OFF／disabled 与 Space／物理点击状态保持。字体通过实际平台命中 Inter Variable、JetBrains Mono、Noto Sans CJK SC。手机范围是浏览器 focus、Tab／Shift+Tab／Enter／Escape／End／ArrowRight 和缩小视口；实体手机 OS 软键盘、完整读屏器及其他操作系统字体未实测，不宣称跨平台全部通过。

## 真实接入范围及未验证项

[82 实际页面结果](../../.tmp/ui-theme-refactor/real-verified-82f25ad8-pages/results.json)在正确实例、无 API fixture 的同源只读浏览器中完成 28 页×桌面／手机=56 次呈现：24 次观察到业务 GET 成功、22 次观察到未启用来源的 503、10 次没有业务 GET。另验 10 个独立来源页签和 2 项 Help 资源；浏览器均关闭。22 次只证明错误／Retry／非 Empty 表现准确，不表示外部业务数据成功。

没有真实成功记录的六项是 Worm 组合编辑、执行预览、执行详情、会员 Profit Sharing 轮次详情、管理员 Profit Sharing 轮次详情、管理员通知详情。它们的正式 React 成功及关键状态由受控场景验证；没有为截图制造写入。[真实认证 4 项](../../.tmp/ui-theme-refactor/real-verified-82f25ad8-auth/results.json)只覆盖本地 DisableAuth／registration 404 恢复边界；外部 Google OAuth、Phantom 签名、Telegram provider 投递、供应商扫描／同步、链上写操作或 Cash Out 未在此完成。

Help 的 llms.txt／AI 说明资源实际可读；固定 ReDoc 2.4.0 本地 bundle 与嵌入产物 hash 一致，实际页面渲染 108 operations、无 pageerror／HTTP 失败，原始记录见 T9 报告。ReDoc 页面仍请求 Google Fonts 与 cdn.redoc.ly logo；本地固定脚本不表示整个文档页离线。主应用字体使用本地资产。

## 历史失败与独立任务审阅

[完整历史清单](../../.tmp/ui-theme-refactor/all-official-runs-after-t9-20167913.json)保留 72 次正式 run：47 passed、25 failed；cleanup 66 passed、6 failed。失败轮、主动取消、工具假设错误和后续修复不被重写为成功，也不将中间 smoke／预检替代最终结果。六项 cleanup failed 都有后续实际退出补证：

| 原始失败 | 资源补证与处置 |
| --- | --- |
| T2 `043c31c8`／`3aa38545` | [两项退出补证](../../.tmp/ui-theme-refactor/early-harness-cleanup-resolution-final.json)：未触发 live gRPC 导致末尾断言，harness／监听／一次性 PG 已退出 |
| T7 `a27a2657` | [退出补证](../../.tmp/ui-theme-refactor/t7-fix1-cleanup-resolution.json)：并行 build 清空运行中的 dist，后续串行 `79507e1c` 通过；原 cleanup failed 保留 |
| T9 `12cedf1c` | [退出补证](../../.tmp/ui-theme-refactor/task-9-full-1-cleanup-resolution.json)：root fixtures 480 passed／18 failed，未进入 live／prefix，后续修正并全量通过 |
| T9 `8188df63` | [主动取消记录](../../.tmp/ui-theme-refactor/task-9/acceptance-frozen-1-interrupted.json)：退出 130，325 条 console passed 但无正式 results，不计整轮通过 |
| 父误模式 `ab5c3e32` | [命令纠正及退出补证](../../.tmp/ui-theme-refactor/smoke-command-correction-cleanup-20167913.json)：实际 isolated，SIGINT／runner130／make2，确切 PID／端口／label PG 无残留；正确 smoke 为 `14599154` |

新增原生弹窗工具首轮强制 animation:none 使 Ant 关闭不能完成，失败目录保留；工具恢复正常 motion 后 1/1 通过，未改产品或放宽条件。其余失败、修正和有限验证详情保留在各稳定任务报告，父移交的[阶段记录](../../.tmp/ui-theme-refactor/acceptance-progress-before-t10.md)只作历史补充。

| 任务 | 成果与独立审阅 |
| --- | --- |
| T1 | [主题／字体／偏好及生成清理报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-1-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-1-rereview-1.md) |
| T2 | [壳／组件／runner 报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-2-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-2-rereview-1.md) |
| T3 | [身份／自助报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-3-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-3-rereview-2.md) |
| T4 | [管理员运维报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-4-report.md)，[审阅](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-4-review.md) |
| T5 | [钱包／Solana／治理报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-5-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-5-rereview-1.md) |
| T6 | [市场／赛事／OO 报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-6-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-6-rereview-1.md) |
| T7 | [Worm 资产／组合报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-7-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-7-rereview-1.md) |
| T8 | [Worm 执行报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-8-report.md)，[最终复审](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-8-rereview-1.md) |
| T9 | [整体验收报告](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-9-report.md)，[独立审阅 Approved](../../.tmp/ui-theme-refactor/sdd-archive/2026-09-14-web-ui-theme-refactor/task-9-review.md) |

T1–T9 最终审阅无 Critical／Important。T9 两项 Minor 保留：6 条 jsdom navigation／scrollTo 的 console.error（测试环境能力限制，未静音）和最大 chunk 820.627 kB 的 Vite >500 kB 提示。Jest／build 通过不等于日志没有提示；真实跳转／滚动由浏览器证据补充。

## 文档、资产与证据完整性

[625 份批准资产](../../.tmp/ui-theme-refactor/approved-assets-20167913.json)未修改：438 PNG／136 JSON／39 HTML／2 Markdown／3 CJS／2 CSS／2 WOFF2／1 SVG／2 TXT；两份预览 README 也冻结。所有新执行证据置于独立目录。[最终五索引引用检查](../../.tmp/ui-theme-refactor/t9-final-evidence-reachability-20167913.json)确认 1322 个唯一文件／目录可访问，只是路径核对，不新增视觉通过记录。

T10 的[文档与映射检查](../../.tmp/ui-theme-refactor/t10-document-checks.json)核对全部修改文档本地链接、51 条入口／38 主场景／S1–S14 精确证据归属；[批准资产复核](../../.tmp/ui-theme-refactor/approved-assets-t10.json)保留实际哈希结果。文档 `git diff --check` 退出 0。完整分支检查退出 2：JetBrainsMono-OFL.txt 第 21 行尾空格；固定 ReDoc 上游 LICENSE 第 3 行尾空格／第 22 行 EOF 空行；redoc.standalone.js 第 1801 行尾空格。仅排除 OFL 的检查仍退出 2，保留后两项；再仅排除这三份经 SHA 验证的上游原件，补充检查退出 0。完整与两次排除检查均保存原始输出，不称全分支无提示，也不修改许可／vendor 原字节。

## 交付状态与环境收尾

以下为控制代理最后统一更新的状态区；上文实现版本和历史证据无需改写成新的运行结果。

| 项目 | 当前事实 |
| --- | --- |
| 正式实现／T9 验收 | 已完成，生产版本 `20167913`；T1–T9 独立任务审阅通过 |
| T10 长期文档 | 已同步；独立任务审阅待执行 |
| 源码及生成物整分支独立审阅 | 待执行，包含 API 清理、realm、精度、陈旧结果及破坏性操作边界 |
| 主实例最终停止／验证 | 待控制代理在审阅后执行；当前仍供必要审阅使用，未获长期保留指令 |
| 一次完成邮件 | 未发送；全部计划、审阅与收尾完成后才运行仓库通知命令 |
| 分支整合 | 未 push／merge；保留本地分支及工作区 |

最新归属见[环境登记](../../.tmp/ui-theme-refactor/environment.json)与[readiness](../../.tmp/ui-theme-refactor/t9-controls-final-runtime/readiness.json)：同 worktree、`INSTANCE=ui-theme-refactor`，RunID `422550b0-362d-43ea-921f-4044837a3f34`，session `99824`，supervisor PID `3231802`，日志 [start.log](../../.tmp/ui-theme-refactor/t9-controls-final-runtime/start.log)。UI `http://127.0.0.1:34000`、API `http://127.0.0.1:38080`，七进程身份和两 realm 原账户均核对，嵌入 JS／CSS 匹配冻结构建。

| 本任务资源 | 当前处置／准确停止入口 |
| --- | --- |
| 主实例及所属服务 | 仍运行供审阅；从上述 worktree 执行 `make stop INSTANCE=ui-theme-refactor`，再按运行身份核对七进程、端口和所属容器停止，保留数据库／卷 |
| 独立测试数据库 `athena-ui-theme-tests` | 当前运行，`127.0.0.1:54648`；最终 `docker stop athena-ui-theme-tests`，保留容器／数据 |
| Telegram loopback 替身 | 当前运行，端口 39931、session 81052；读取 [identity](../../.tmp/ui-theme-refactor/telegram-fixture.json) 校验 PID／启动身份后执行 `kill -TERM <verified-pid>`，核端口释放；日志 [telegram-fixture.log](../../.tmp/ui-theme-refactor/recovery-2026-09-15/telegram-fixture.log) |
| 执行者临时 preview／隔离资源 | 已结束；T9 [临时 preview 清理](../../.tmp/ui-theme-refactor/task-9/temporary-preview-cleanup.json)、[最终控件 preview 清理](../../.tmp/ui-theme-refactor/task-9/control-preview-cleanup-20167913.json)，历史六项 raw cleanup failed 的退出补证见上表 |
| 既有用户／其他任务环境 | 保持原样，主工作区 4000 与 trader-sync-independent 24000 不属于本次停止目标；[原归属快照](../../.tmp/ui-theme-refactor/preserved-environments-snapshot.json)保留 |

最终停止必须先核归属，再执行停止和实际退出校验；当前 readiness／cleanup 报告不代替主实例最后停止证据。T10 不启停服务、不发送邮件，也不清除数据库、数据卷或原始验收证据。
