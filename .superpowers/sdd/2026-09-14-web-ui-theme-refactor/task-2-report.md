# T2：应用壳、共用组件与浏览器场景接线报告

状态：完成。T2 在 T1 的固定深色 Provider 与 token 基础上落地会员／管理员共用应用壳、首批受控浏览器场景及 filtered acceptance runner；未启用 Token，也未重排 Trader Sync 业务页。

## RED 证据

| 行为 | 失败证据 |
| --- | --- |
| runner 过滤契约 | `yarn node --test scripts/acceptance-runner.test.mjs` 初次运行为 39 passed / 3 failed：runner 未传独立 `--grep` 参数、未区分 filtered 报告和项目范围。 |
| 壳结构与 Appearance 兜底 | `yarn test --runTestsByPath src/app/app.test.tsx src/app/admin/app.test.tsx --silent` 新增结构断言后有 2 个失败：账户身份仍在侧栏，侧栏不是 224px；两条 Appearance 404 回归在实现前已通过，证明沿用既有兜底。 |
| 语义标题 | `yarn playwright test --config=playwright.config.ts --project=ui-fixtures --grep='theme:core'` 首次在登录场景因缺少可见 `h1` 失败。 |
| 根字号放大 | `.tmp/t2-red-root/` 显示 `html` 32px 时顶栏高 186px，身份 copy 继承旧 Header 行高。 |
| 严格请求阻断 | `.tmp/t2-red-harness-cross/` 显示跨域 `/markets` 未进入 unexpected ledger；修复后所有 HTTP(S) 跨域请求均中止并记账。 |
| 部署前缀与静态资源 | `UI_ACCEPTANCE_GREP='theme:harness' make ui-acceptance` 首次失败，报告 `.tmp/athena-ui-acceptance/2026-09-14T15-59-51-801Z-a7e69b94/report.md`：`/athena` 下根路径 favicon 被误判为 API 前缀逃逸；随后区分同源静态资源与根 `/api`。 |
| 抽屉焦点 | `.tmp/t2-red-admin-focus/` 显示管理员抽屉打开后关闭按钮未获得焦点。 |
| 隐藏导航 | `.tmp/t2-red-hidden-visibility/` 捕获到移动端折叠动画期间侧栏子项仍可见；改为折叠完成前 `visibility: hidden` 且不渲染菜单。 |
| fixture-only harness | `.tmp/athena-ui-acceptance/2026-09-14T15-49-47-830Z-6cb97277/report.md` 中浏览器 5/5 通过，但 harness 因 filtered 运行没有 live gRPC 调用而清理失败；runner 随后显式设置 fixture-only 环境，完整运行仍保留 gRPC 调用断言。 |
| 200% 门户边界 | `.tmp/t2-red-root-portal/` 显示账户菜单身份行被截断；扩大受视口约束的菜单宽度后加入可读性断言。 |
| v3 壳页脚与头像 | `.tmp/t2-red-sidebar-footer/` 捕获缺少 Help/workspace 页脚，`.tmp/t2-red-header-avatar/` 捕获 Ant 默认灰色头像；实现后均由最终截图复核。 |

## 实现结果

- 会员端与管理员端统一为 224px 桌面侧栏、64px 顶栏、32px 桌面内容边距、20px 手机边距和 900px 抽屉断点；身份菜单移至顶栏，头像使用深色面板／边框，并随根字号放大。
- 抽屉打开后等待可见完成态再聚焦关闭按钮，保留 Tab 焦点环、Escape、背景 inert、关闭后焦点恢复和 skip link；移动端不保留隐藏的可聚焦菜单。账户 Dropdown 也明确响应 Escape。
- 导航组标题改为批准的 Title case、`.8125rem/400`，补回 Help 与 realm workspace 页脚；管理员模块采用固定图标映射，当前项有 `aria-current=page`。
- 共用字号、控件高度、表格、Select、Dropdown、Modal、Message、Notification、只读／禁用／选中态和数字列样式使用 rem/token 桥接；共享壳仅使用批准的 400/500/600 字重。金额／数量和标识组件分别标记为 `.athena-number` 与 `.athena-identifier`，保留原始显示精度。
- 登录、注册、bootstrap error／forbidden 补齐单一可见 `h1`；管理员壳设计文档移除 `/admin/account/appearance` 并说明落入既有 404。
- 新增 `contracts.ts`、`cases.ts`、`routes.ts`、核心和 a11y spec。首批场景覆盖会员／管理员登录、注册、bootstrap error、pending、两 realm 壳和 admin forbidden。
- case 路由严格匹配 method、部署相对 path 与 realm，记录 query/body/次数；未声明 API、部署前缀逃逸和所有 HTTP(S) 跨域请求均报 unexpected 并中止。case 缺失立即失败，写请求必须有独立 fixture，pending 延迟限制在 10 秒内。
- runner 将 `UI_ACCEPTANCE_GREP` 作为 Playwright 独立 argv 传递。filtered acceptance 仅运行 `ui-fixtures`，filtered a11y 仅运行 `a11y`，报告明确标为局部；未过滤运行继续保留原 ui-fixtures/live 路径，零匹配返回失败。

## GREEN 与验证

| 命令 | 结果 |
| --- | --- |
| `cd ui && yarn node --test scripts/acceptance-runner.test.mjs` | 42/42 passed。 |
| `cd ui && yarn test --runTestsByPath src/app/app.test.tsx src/app/admin/app.test.tsx --silent` | 46/46 passed。覆盖登录分流、权限／撤权、菜单、壳位置和两条 Appearance 404。 |
| `cd ui && yarn playwright test --config=e2e/theme-refactor/controls.playwright.config.ts` | 2/2 passed；T1 Button/Input/Alert 夹具在共享 CSS 修改后复验。 |
| `go test -tags=integration ./internal/tradersync/acceptance -run '^TestDoesNotExist$'` | passed；filtered harness 分支可编译。 |
| `cd ui && yarn lint` | passed。 |
| `cd ui && yarn build` | passed；仅保留既有 Vite chunk size warning。 |
| `UI_ACCEPTANCE_GREP='theme:core' make ui-acceptance` | passed，root 与 `/athena` 各 5/5；构建、两套 harness、startup、fixtures、PostgreSQL cleanup 全部通过。报告：`.tmp/athena-ui-acceptance/2026-09-14T15-54-10-745Z-2b9a3ede/report.md`。 |
| `UI_ACCEPTANCE_MODE=a11y UI_ACCEPTANCE_GREP='theme:a11y' make ui-acceptance` | passed，root 与 `/athena` 各 6/6；12 份 axe WCAG 2 A/AA/2.1 A/AA 场景结果均通过，清理通过。报告：`.tmp/athena-ui-acceptance/2026-09-14T15-54-37-891Z-b0198bb3/report.md`。 |
| `UI_ACCEPTANCE_GREP='theme:harness' make ui-acceptance` | 2/2 deployments passed；同源静态资源继续加载，跨域业务请求和 `/athena` 部署误发根 `/api` 均记入 ledger。报告：`.tmp/athena-ui-acceptance/2026-09-14T16-00-28-858Z-6472a713/report.md`。 |
| `git diff --check` | passed。 |
| `rg -n 'font-weight:\\s*(bold\|[789]00\|6[5-9]0)' ui/src/app/styles/shared.css ui/src/app/member/app.tsx ui/src/app/admin/app.tsx` | 无匹配。 |

## 浏览器证据

- 最终桌面壳：`.tmp/athena-ui-acceptance/2026-09-14T15-54-10-745Z-2b9a3ede/root/ui-fixtures/artifacts/theme-refactor-theme-core--ffb3a-d-desktop-application-frame-ui-fixtures/member-shell-desktop.png`
- 最终手机壳（账户菜单与抽屉关闭完成态）：`.tmp/athena-ui-acceptance/2026-09-14T15-54-10-745Z-2b9a3ede/root/ui-fixtures/artifacts/theme-refactor-theme-core--70257-ed-mobile-application-frame-ui-fixtures/member-shell-mobile.png`
- 最终 200% 根字号（32px root、可读身份、抽屉 x=0/width=280、顶边跟随实际 header）：`.tmp/athena-ui-acceptance/2026-09-14T15-54-10-745Z-2b9a3ede/root/ui-fixtures/artifacts/theme-refactor-theme-core--27260-follow-200-root-text-sizing-ui-fixtures/member-shell-root-200.png`
- 深色固定登录：`.tmp/athena-ui-acceptance/2026-09-14T15-54-10-745Z-2b9a3ede/root/ui-fixtures/artifacts/theme-refactor-theme-core-系统浅色和旧偏好不能改变深色登录-ui-fixtures/member-login-dark.png`
- Axe 原始结果目录：`.tmp/athena-ui-acceptance/2026-09-14T15-54-37-891Z-b0198bb3/{root,athena}/a11y/artifacts/`。

人工核对最终截图：桌面导航组为正常大小写且无旧字间距，Help/workspace 页脚可见，顶栏头像为深色边框；手机折叠态无侧栏残影；200% 场景的身份与抽屉均在完成态，无 Tooltip 或 Dropdown 遗留。

## 范围与限制

- 两次 `make ui-acceptance` 都是按 brief 运行的 filtered 局部证据，不代表 T9 的全量无过滤 acceptance 或真实 smoke。
- T2 为 Table、Select、Modal、Dropdown、Message、Notification 提供共享字体／控件桥接，并实际覆盖账户 Dropdown、表格和 T1 控件；全组件状态矩阵、portal 与原生 Chrome zoom 的扩大验证归 T9。
- T3–T8 继续追加真实业务页 fixture，并清理各自业务域遗留的 700–900 字重与金额列消费者。T2 未改业务排序、业务 guard、双 realm 注册、Token 可见性或 Trader Sync 页面信息结构。
- 本任务未启动或保留独立产品服务。`http://localhost:34000` 是主代理既有 `ui-theme-refactor` 环境，保持运行；filtered acceptance 自管的 harness/PostgreSQL 均已清理。
