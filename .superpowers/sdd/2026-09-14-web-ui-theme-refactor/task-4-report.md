# T4 管理员账户与运维五页实施报告

状态：DONE，局部实施与局部验收完成，等待父代理安排独立任务审阅。

- 工作区：`/home/yege/work/athena/.worktrees/ui-theme-refactor`
- 分支：`codex/ui-theme-refactor`
- 准确基线：`4d87e292327acbf3446282ac8d7542517dff9c79`
- 实现提交：`c243b3f4ebaac843c6675311d1459eb6fba244b1`（本报告随后单独提交；该 SHA 精确对应全部实现及测试文件）
- Node：`/home/yege/.nvm/versions/node/v24.14.1/bin/node`，v24.14.1；未改全局 NVM、组件库或服务配置。
- 遵循 task-4-brief、task-4-handoff、AGENTS、using-superpowers/Codex、SDD implementer、TDD、verification、Impeccable、browser-acceptance；已读取 v15–v18 提案、v23 契约并实际查看批准图片。设计授权沿用，不重复审批。未派生代理、未 push/merge/发送通知。

## 逐项实现与源文件

| 要求 | 实现与证据 |
| --- | --- |
| v15 账户桌面目录／详情，手机目录→详情 | `ui/src/app/admin/pages/admin-accounts.tsx`：默认 Access，Profile／Identity 为独立页签；目录身份、时间、状态完整；手机返回目录保留 query/status/page/pageSize，更新 account 参数不覆盖其余查询参数；草稿切换有放弃确认。 |
| 权限完整 aggregate、revision/CAS | 模块权限改用有标签的 Select，描述默认折叠；隐藏 Token 字段仍由完整 aggregate 传递；API Key、Profit Sharing、sign-in 独立。权限冲突重新读取 authoritative access 并丢弃旧 access 草稿；Profile 冲突仍保留名称草稿。Profile、tier、avatar、access 沿实际服务分别保存、分别使用 revision。 |
| 敏感确认与迟到请求 | 说明撤销影响，默认 Cancel，手机按钮纵向全宽且 x／右边界相同；显式首尾 Tab 循环。管理员 scope 失效卸载工作区及静态确认弹窗；新增账户选择代次保护，旧 Profile/tier/avatar/access 请求及冲突重取不能覆盖后来选中账户的草稿。 |
| v16 三来源 | `service-status.tsx`：Services／Notifications／Trader Sync 三页签，独立 loading/error/stale/time；沿用 useVisibleQuery 的可见性、10 秒轮询与单飞。健康 checkedAt、通知收到时间及 trader asOf 不互换，失败来源保留其上次值，其他来源可以独立更新。 |
| v23 服务表格 | `.service-health-table` 使用容器 `service-health / inline-size`，44rem 转分组行；错误占独立行，极窄容器进一步堆叠。正式测试确认桌面 thead 可见；720px、根字号 32px 下先量得实际文字放大，再检查 Unreachable 与完整超时说明矩形不交叠、错误全文仍在容器内。 |
| 通知恢复／队列与 Trader Sync 指标 | 独立恢复状态／剩余和已用毫秒、默认折叠诊断；System／Account 队列并列。Trader Sync 保留 metric 原始字符串值、unit、gauge/window/epoch scope 与来源时间；`9007199254740993` 不变为浮点近似，不可观测 raw 数据不显示成零。 |
| v17 网关／Live Probe | `etherscan-gateways.tsx`：默认 Gateways，完整地址、可达性、runtime、延迟、时间与错误；Base URL 等辅助字段可展开。Live Probe 表单和最新结果分开，保留 interval/request 配置、服务端 overall result／requiredSuccess、失败分类、By Gateway／By API Key、完整可复制诊断与 run 时间。 |
| Probe 实际权限及请求 | 管理员权限／当前 scope／运行中／提交中共同限制；只读页签和展开行为不启动测试。显式提交单飞、失败保留参数和前次结果；既有 running run 通过其 ID 独立轮询，结束后停止，不产生新 POST。 |
| v18 通知列表 | `system-notifications.tsx`：关键词、status、chat 独立控件，桌面五列、手机记录；详情往返保留原筛选和分页。状态/严重程度使用 success/error/warning/processing/default 语义色。 |
| 测试发送 | 单飞；pending 禁用关闭、Escape、Cancel 和重复发送；失败保留弹窗与输入，成功语义为 queued。trim 后 topicLabel 沿现有服务发送；失去管理员 scope 时 abort、卸载并拒绝旧回调发布。 |
| 通知详情 | `system-notification-detail.tsx`：消息、投递记录、时间线分区；手机顺序为消息→投递→时间线。保留完整 provider error、17 项模型事实、五种独立时间及缺失含义；ID/source/providerMessageId 默认收起；保留安全 HTTP(S) 链接及 unknown 不自动重发说明。 |
| 域布局／共用消费 | `ui/src/app/admin/components/operation-facts.tsx` 为 T4 窄面板提供原生 dl，避免共享 KeyValueGrid 只按视口计算列数造成详情裁字；T2 共享组件未改。`ui/src/app/styles/admin-features.css` 只替换 T4 域规则，Profit Sharing 既有块保留；采用共享变量/rem/identifier/numeric 角色。通知记录和 probe error sample 为透明分隔行。Search 针对当前 Ant6 `.ant-input-search-btn` 修复断字。 |

## 请求夹具与行为覆盖

新增 `ui/e2e/theme-refactor/fixtures/admin-operations.json` 五个唯一案例，注册到 `cases.ts`；沿用既有严格 routes/contracts，精确匹配 method/path/realm，记录 query/body/count，未声明 API、跨域或部署前缀逃逸直接失败。合成管理员 UUID 与会员 UUID 不同，没有使用父代理真实账户作为 fixture。

新增 `theme:admin` 每前缀 23 场景（根路径与 `/athena` 共 46）：

1. 五入口桌面／手机，10 场景，真实 DOM 与截图。
2. 720px＋root200 服务容器重排，1 场景。
3. 两尺寸的 Notifications/Trader Sync 来源与 Probe 完整诊断，2 场景；展开布局不会写 API。
4. 通知筛选→详情→返回，单次测试 POST、pending Escape/关闭/重复操作、失败保留，1 场景。
5. 账户手机返回筛选、敏感确认键盘与按钮边界、access 409、完整 Token aggregate 与 authoritative restore，1 场景。
6. 实际 `403 ADMIN_REQUIRED` 触发管理员重验，重验中清屏/关闭旧弹窗；userinfo 失败后 Retry 成功且新表单空白，1 场景。
7. 既有 running Probe 按 ID 轮询至完成且零 POST，1 场景。
8. Probe 启动失败只写一次并保留参数和上次结果，1 场景。
9. `apiKeyEnabled=true`、`administrator=false` 不能访问管理员操作，仅 bootstrap，1 场景。
10. 单来源不可用时保留原始恢复数值/时间，另一来源独立更新，1 场景。
11. 延迟 Profile 保存完成后不能把 Mira 的姓名写进 Noah 草稿，1 场景。
12. Search 文本单行、实际字号与文字矩形位于按钮内：1440/root100、320/root200，2 场景。

相关 unit 共 5 suites／27 tests：`admin-accounts.test.tsx`、`service-status.test.tsx`、`system-notification-detail.test.tsx`、`admin/read-scope.test.tsx`、`admin/notification-service.test.tsx`。保留 scope 对已知会话丢失、同账号不同 issuer、isAdmin 丢失、卸载 abort 的迟到结果保护用例。Service Status 单测只为无 DOM renderer 替换 Ant Tabs 容器，真实 Ant Tabs 的结构和交互由浏览器测试覆盖，服务请求及状态逻辑未 mock 掉。

## RED／GREEN 与失败轮

全部执行证据位于 `.tmp/ui-theme-refactor/t4/`；失败截图和 trace 均保留，未覆盖为成功结果。

| 轮次 | 实际结果与处理 |
| --- | --- |
| 基线 `baseline.log` | 5 suites／26 tests 通过。 |
| `red-unit.log` | 新三来源页签行为测试在旧实现上失败：找不到 Tabs；既有 10 项本轮 service 测试通过。实施后 27 项全部通过。 |
| `red-browser.log` | 初期浏览器命令缺 manifest／project 参数造成设置失败；修正后的该轮与实施时序重合而通过，**不作为 RED**。 |
| `red-reflow.log` | 真正视觉 RED：720/root200 下旧表格 thead 仍显示，未按容器重排。加入44rem容器样式后通过，最终两部署前缀均通过完整矩形断言。 |
| `unit-first/second/third.log` | Ant Tabs 在 react-test-renderer 无真实布局节点导致 addEventListener／ResizeObserver／createNodeMock 循环；定位为测试平台问题，移除该 createNodeMock，仅用稳定 Tabs 容器替身；浏览器仍用真实组件。 |
| `tsc-first/second/third.log` | replaceAll target、AccountStatus 字符串 enum、OperationFacts props 类型错误；按项目真实 TS/DTO 修正，`tsc-fourth.log` 成功。 |
| `lint-focused.log`／`lint.log` | 首先在错误 cwd 运行 eslint 未找到配置；之后 root prettier 与 ui 配置不一致产生13格式错误。改用 ui cwd 的配置，lint-verified 成功。 |
| `browser-first.log` | 第一批11场景通过，但实际查看图片发现窄详情 facts 与状态 metrics 裁字；新增域 OperationFacts 修复，不修改共享 T2。 |
| `browser-behaviors*.log`、`browser-contracts-final.log` | 中间失败为未复位滚动、Ant 虚拟 option 定位、装饰图标 accessible name、错误的 focus 重验触发、input 用 toContainText、Result 并非 heading 等；按真实组件与真实403重验流程修正。所有失败保留。 |
| `red-late-profile.log` | 首次选择器命中 file/text/radio，属于测试定位错误，不作为 RED。 |
| `red-late-profile-second.log` | **真实行为 RED**：先选 Noah，Mira 的延迟 PUT 返回后输入变成 Saved Mira，期望 Noah Park。新增账户选择代次保护后 `green-late-profile.log` 1/1通过；最终两前缀均通过。 |
| `search-boundaries.log` | 两尺寸失败且截图确认 Search 断行。定位 Ant6 实际类名为 ant-input-search-btn，旧样式选择器未命中。修正类名；测试以文本 rect 的独立 top 计真实行数，避免 nested span 重复 rect。`search-boundaries-green.log` 2/2通过；最终两前缀4项通过。 |
| 首轮正式 acceptance | `acceptance-first.log`，root20＋athena20，共40通过，run cleanup passed。不是最终23场景的替代。 |
| 最终正式 acceptance／a11y | 见下节；没有失败、跳过或 flaky。 |

## 最终命令与结果

以下命令使用指定 Node，未改变全局默认值。

```bash
# cwd: worktree/ui
PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH \
JEST_JUNIT_OUTPUT_DIR=$PWD/../.tmp/ui-theme-refactor/t4/unit-verified \
yarn test --runInBand --runTestsByPath \
  src/app/admin/pages/admin-accounts.test.tsx \
  src/app/admin/pages/service-status.test.tsx \
  src/app/admin/pages/system-notification-detail.test.tsx \
  src/app/admin/read-scope.test.tsx \
  src/app/admin/notification-service.test.tsx
PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH yarn lint

# cwd: worktree，严格串行 acceptance → a11y
PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH UI_ACCEPTANCE_GREP='theme:admin' make ui-acceptance
PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH UI_ACCEPTANCE_GREP='theme:a11y' make ui-a11y
```

- `unit-verified.log`：5/5 suites、27/27 tests，exit 0；JUnit 保存在自有 `.tmp`，早期根目录 `junit.xml` 已移到 `t4/junit-unit.xml`。
- `lint-verified.log`：12 项 eslint 配置测试、tsc、全 ui/src eslint 通过，exit 0。
- `acceptance-final.log`：exit 0，build passed；每前缀23，共46通过。
- 最终 acceptance：`.tmp/athena-ui-acceptance/2026-09-14T18-19-54-964Z-4248f163/run.json`。`status=passed`、`cleanup.status=passed`、`cleanup.errors=[]`、`testFailures=[]`。root/ui-fixtures 和 athena/ui-fixtures 的 results.json 各 expected23，unexpected/skipped/flaky 均0。
- `a11y-first.log`：exit 0；`.tmp/athena-ui-acceptance/2026-09-14T18-21-14-821Z-bd8e51cb/run.json` 同样 passed/cleanup passed。每前缀26，共52通过，零 WCAG 2.0/2.1 A/AA violations。每前缀包含原15个已实施身份/共享页面和T4新增11个主入口/辅助页面状态。
- 两次正式运行均保存 build snapshot、SHA256 manifest、runner JSON、截图、原始 axe-results。现有 Vite 大于500KB chunk 提示仍为 warning，无构建失败。
- 直接在借用34000验证时使用：`ATHENA_UI_E2E_MODE=isolated ATHENA_UI_E2E_BASE_URL=http://127.0.0.1:34000 ATHENA_UI_E2E_MANIFEST=/dev/null ATHENA_UI_E2E_OUTPUT_DIR=$PWD/.tmp/ui-theme-refactor/t4/<轮次> PATH=/home/yege/.nvm/versions/node/v24.14.1/bin:$PATH ui/node_modules/.bin/playwright test --config ui/playwright.config.ts --project ui-fixtures --grep '<对应theme:admin用例>'`。请求仍由严格 fixture 拦截。正式命令的 harness manifest 由工具正常生成。
- 提交前 `git diff --check` 通过。

## 图片实际查看

最终 acceptance 根路径的 `root/ui-fixtures/artifacts/` 共22张：五页desktop/mobile（10）、三来源/Probe展开desktop/mobile（6）、service-text-200（1）、手机目录（1）、权限确认（1）、测试通知弹窗（1）、Search两个尺寸（2）。全部类别已用本地 view_image 实际查看；五页最终主图及关键弹窗逐张查看。320/root200 Search 图也在独立 green 轮查看。初轮截图同样查看并用于发现和修复 facts 裁字与 Search 断行。

关键文件名：

- `admin-accounts-desktop.png`、`admin-accounts-mobile.png`、`accounts-directory-mobile.png`
- `admin-services-desktop.png`、`admin-services-mobile.png`、`service-text-200.png`
- `notification-runtime-{desktop,mobile}.png`、`trader-runtime-{desktop,mobile}.png`
- `admin-gateways-{desktop,mobile}.png`、`probe-{desktop,mobile}.png`
- `admin-notifications-{desktop,mobile}.png`、`admin-notification-detail-{desktop,mobile}.png`
- `account-access-confirm-mobile.png`、`test-notification-mobile.png`
- `admin-directory-1440-root100.png`、`admin-directory-320-root200.png`

a11y 根路径6张 `admin-auxiliary.png` 中实际查看 Profile、Live Probe、Notifications、Trader Sync、发送弹窗；Access 与最终 acceptance 主图重复。所有完整路径可在上述 run 的 artifacts/results.json 中解析。

## 资源、边界与交付

- 本任务的临时浏览器均已退出；三个正式运行的六个 harness PID、12个监听地址全部核对不存在/不监听，见 `.tmp/ui-theme-refactor/t4/cleanup-verification.json`。正式运行自带停止文件/退出和 postgres cleanup 阶段均 passed。
- 最终 acceptance 与 a11y 容器按各自 `io.athena.ui-acceptance.run` label 执行 `docker ps -a` 检查为空。没有删除用户数据卷或共享基础设施。
- 保留父代理已有 `http://localhost:34000`，仓库为本worktree、`INSTANCE=ui-theme-refactor`。服务归父代理，未停止、重配或发送真实业务写入；原4000/24000也保持。该环境未来收尾仍由父代理从同worktree执行 `make stop-instance INSTANCE=ui-theme-refactor`，不由本子任务执行。
- 这是**过滤的 T4 局部验收**：真实构建/React/浏览器，API为显式fixture；没有真实 Etherscan 请求、Telegram 投递或会员业务资金操作，不能代表全站T9的未过滤/live/真实smoke验收。外部供应商投递成功不在此宣称。
- 父代理已记录的 Help/ReDoc 404 属于 T9，未扩大本任务范围。共享壳品牌与 modal 外框的全站复核归T9。
- 未编辑/暂存父代理计划、`docs/testing/web-ui-theme-refactor-acceptance.md` 或 task-2-report 勘误。实现提交仅包含14个T4源文件/测试/夹具；本报告另行提交。
- 自查未留已知T4阻塞。独立审阅由父代理随后安排；本报告不替代其结论。
