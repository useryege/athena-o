# 全站单一深色 UI 重构技术方案

> 状态：技术方案已整理，正式实现尚未开始。2026-09-14 用户确认 v22 后同意核对整体覆盖并制定实施计划。视觉决定沿用既有确认记录；本文件将这些决定转换为代码边界、依赖和验收要求，不新增逐页视觉确认。

## 目标与范围

将当前会员端、管理员端及共享组件迁移到已确认的 Nansen 单一深色体系，保留业务数据、精度、权限、请求生命周期和操作后果。主要用户仍是需要核对数据并完成操作的会员及管理员，视觉以可读、可核对和清晰的操作层级为目标。

依据：[全站视觉主题](../../requirements/web-ui/visual-theme.md)、[完整覆盖清单](../../requirements/web-ui/theme-refactor-coverage.md)、[机器清单](../../requirements/web-ui/theme-refactor-coverage.json)、各版本 approval。v20–v22 的 18 个业务页面及六项共用适配所展示主视觉均已确认；其他既有界面使用各自已确认版本。

本次实施覆盖 37 条改版页面入口、2 条 Appearance 删除、4 条默认／兜底规则。8 条 Trader Sync 专属路由保留当前业务布局；共享主题、字体、导航和中立组件的变化自然覆盖它们，必须回归其可读性及行为。Service Status 的 Trader Sync 运维页签继续按 v16 改版。Token／Nansen 钱包战绩沿[独立接入计划](../plans/2026-09-14-token-wallet-analytics.md)实施，本计划不新增其 API、路由或供应商请求。

## 当前实现与处理选择

| 当前事实 | 实施处理 |
| --- | --- |
| React 18、Ant Design 6、React Router 6、Vite；会员／管理员独立入口 | 保留技术栈和双应用边界，使用 Ant Design theme token 与受控 CSS 落实已定稿视觉 |
| `shared.css` 有浅色根变量、深色覆盖及橙色强调，存在 `min-width:1280px` | 用单一 token 集直接替换；清除旧深浅分支及全页最小宽度，按业务行重排 |
| `SessionBootstrap` 成功后才提供主题；错误／加载在 Provider 外 | 将中立主题 Provider 提升到两个 React entry，覆盖 bootstrap、错误、登录和业务页 |
| 共享注册使用独立 `RegistrationBootstrap` 与空 ConfigProvider | 复用入口级 Provider；移除注册的空主题层，保持注册票据与 realm 分流 |
| 两份 HTML 首屏脚本读取本地主题并监听系统偏好 | HTML 固定 `data-theme="dark"`、深色首屏背景与 color-scheme；移除主题探测 |
| Appearance 通过服务端 `account_preferences` 同步，包含独立 revision | 删除主题专用持久化与 API 链路；保留 profile、授权和其他本地视图偏好 |
| `ResourceTable` 已有 `compactRender`、分页及选择；`AppPage`／`Section` 已共享 | 扩展既有组件和样式；不另建平行表格、页面容器或跨 realm 服务层 |
| 原型是独立 HTML／合成数据，浏览器 acceptance 主要覆盖 Trader Sync | 将正式 React 场景纳入现有 acceptance 及 a11y 配置，逐域核对 API fixture 与页面字段 |

选用“保留现有组件库和业务状态、重构展示层及主题入口”。仅覆盖颜色无法实现已确认的信息结构；更换整个组件库会扩大表单、弹窗和键盘行为的重写范围。当前没有需要新组件库才能达到的视觉目标。

## 主题、字体与空间契约

`ui/src/app/styles/tokens.css` 是运行时颜色、字体角色和共用尺寸的权威来源，`shared.css` 导入它。`ui/src/app/shared/athena-theme.tsx` 从根元素 CSS 变量读取对应颜色并产生 Ant Design `ThemeConfig`，避免两套颜色表各自演化。Theme Provider 不读取账户数据、存储或系统配色。

总体审阅后的[一致性修订契约](../../requirements/web-ui/theme-consistency-contract.md)与 [v23 参考实现](../../requirements/web-ui/previews/theme-consistency-v23/README.md)补充最终绘制规则。Ant 深色算法完成后由 `athena-color-roles.ts` 锁定最终角色，不能只向算法输入 seed。公共层也是 Token 前端任务的前置依赖；Token 后端可独立推进，页面不得另建主题和字体副本。

| CSS 变量 | 值／职责 |
| --- | --- |
| `--athena-bg` | `#06080B` 页面背景 |
| `--athena-panel`、`--athena-panel-solid` | `#0F1114` 面板 |
| `--athena-panel-elevated`、`--athena-bg-soft` | `#181D22` 浮层／较亮层 |
| `--athena-border` | `#252A30` 分隔线 |
| `--athena-border-strong` | `#61717B` 控件边框，沿 v22 最终原型的 `--edge` |
| `--athena-text` | `#FFFFFF` 主文字 |
| `--athena-muted` | `#9FA0A1` 辅助文字 |
| `--athena-primary`、`--athena-brand`、`--athena-focus` | `#00FFA7` 强调与焦点 |
| `--athena-primary-hover`／`--athena-primary-active` | `#51FFC3`／`#09C385` |
| `--athena-green`／`--athena-amber`／`--athena-red`／`--athena-blue` | `#55D9A1`／`#E6BF72`／`#F58C9B`／`#9BC6F3` 语义反馈 |
| `--athena-selected-bg` | `#102C24` 导航／分段选中底色；数字页码保留面板底与青绿边框 |
| `--athena-{green,amber,red,blue}-{bg,border}` | 沿 v4 四组反馈背景／边框，完整值见修订契约及 token 文件 |
| `--athena-red-hover`／`--athena-red-active` | `#FFB0BD`／`#E5778B` 危险实心按钮交互态 |
| `--athena-font` | Inter，中文按 Noto Sans CJK SC、PingFang SC、Microsoft YaHei 回退 |
| `--athena-data-font` | JetBrains Mono，地址／哈希／逐字符技术标识 |
| `--athena-sidebar-width`／`--athena-header-height` | `224px`／`64px` |

HTML 在样式加载前只内联深色背景和正文色，两份入口值必须与 token 相同，并由浏览器首屏断言核对。其余颜色不在 HTML 重复。Ant Design 主按钮的正常、hover、active 都使用背景深色文字；普通按钮、危险按钮、标签和反馈框分别沿已批准语义，不能将所有按钮或成功状态都变为青绿色。

使用仓库现存的 InterVariable 与 JetBrainsMono-Regular WOFF2，复制到 `ui/src/assets/fonts/athena/` 并附各自 OFL。来源和摘要见[字体记录](../../requirements/web-ui/previews/files/typography-v2-font-sources.json)。`fonts.css` 使用本地 `url()` 和 `font-display:swap`；英文／数字字重限 400、500、600，等宽字体用 400，禁用地址编程连字。移除当前 Heebo 声明与不再使用的字体文件，先核对其他引用。运行时不访问字体 CDN。

文字层级严格沿 v2：页面标题桌面 1.75rem、手机 1.5rem，区块 1.25rem，正文 1rem，控件／表格 0.875rem，辅助／地址 0.8125rem。金额启用 `tabular-nums lining-nums`，不改变业务格式化、原始精度或将未知值写成零。完整地址、原值及复制入口按页面确认范围保留。

上述字号以公共 CSS 角色变量提供。保留用户根字号；`px2remTransformer` 处理普通 Ant 样式，`cssVar.key='athena-theme'` 对应的共用 rem 桥接处理 Ant 6 的全局及组件字号变量。不能假设转换器也会转换 token 变量声明。验证先确认实际字体 14→28px，再检查局部重叠；静态样板的 200% 根字号模拟不等于浏览器原生缩放或正式 React 验收。

桌面应用壳 224px 侧栏、64px 顶栏、32px 内容边距，手机外侧 20px；沿 v3 与 v22 的 900px 应用壳切换点，JS 媒体查询与 CSS 同步。页面内部可以采用其原型的 1100px／1180px／600px 等内容断点。主操作目标至少 44px；200% 字号时允许控件增高，不能用固定高裁字。原型确认过的常规 40px 导航项保留其视觉比例，手机及主要触控操作补足目标面积。

## 组件和页面边界

共享组件继续位于 `ui/src/app/components/`，只接收展示值和回调，不导入会员／管理员业务 service。`AppPage`、`Section`、`ResourceTable`、`ChoiceGroup`、`MetricRow` 等保持现有调用契约；只在确有消费方时增加展示属性。

`ResourceTable` 的桌面列和手机 `compactRender` 使用同一份已筛选、已分页数据。隐藏的表示不参与键盘焦点；不能让手机重新排序、重新请求或获得不同金额。分页仍按模块原契约保留数字页、cursor、Previous／Next 或 Load more，v5 的钱包示例不覆盖全站分页业务。

同类组件跨页面按修订契约维护选中与只读状态；只读仍能聚焦、选中和复制，禁用保持独立外观和语义。可纵向比较的余额／金额列和表头右对齐，独立指标及文字不强制右对齐。管理员导航沿 v15 固定模块图标，区块标题统一 1.25rem／600，顶栏头像随文字放大。Service Status 以容器 44rem 阈值重排，保留完整错误独占一行。

弹窗保留现有 Ant Design 实现、受控开关、单飞和焦点恢复，用共用 class 约束最大宽度、可滚动正文及固定操作区。移动端按钮按原图纵向排列；危险确认显示对象和后果。新增样式不新增业务确认步骤，尤其 Prepare、授权、Start 和 Cash Out 的顺序继续各自独立。

每个业务任务修改对应 React 展示和域样式，不把静态 HTML 的 sample 控件、fixture 状态切换或模拟成功逻辑带入产品。对巨大文件只在展示职责需要时抽取局部组件，不重组无关状态机或服务。

| 文件边界 | 职责 |
| --- | --- |
| `styles/tokens.css`、`shared/athena-theme.tsx`、两份 entry／HTML、`assets/fonts.css` | 单深色、字体、Ant Provider 与首屏 |
| `styles/shared.css`、`components/*` | 中立组件、表格、反馈、排版、弹窗及共用空间 |
| `member/app.tsx`、`admin/app.tsx` | 各自导航、账户菜单、路由和既有 guard；不合并服务图 |
| `styles/member-features.css`／`styles/admin-features.css` | 各域页面样式；必要的新域 CSS 由各自入口导入 |
| 现有各 `pages/*.tsx` 及共享 Account／Help | 按覆盖清单映射到对应视觉版本，继续消费当前 API |
| `e2e/theme-refactor/`、`e2e/theme-refactor.spec.ts`、`e2e/theme-refactor-a11y.spec.ts` | 正式 React 的场景、断言、图片及辅助状态证据 |

## Appearance 完整清理

目标是系统只提供一套深色主题，不将原有 light／system 逻辑作为隐藏兼容路径留下。该清理与前端基础主题在同一交付批次完成。

1. 删除两条 Appearance 路由、账户菜单主题组、切换图标、`changeTheme`、`themeChanging`、`AccountCenterSection` 的 appearance 分支和相关说明。旧 URL 使用既有未找到规则，不新增兼容重定向。
2. 删除 `shared/theme.ts` 的转换函数及所有消费者、`ThemeMode`／`ResolvedTheme`、system media listener、`syncServerTheme` 和主题 DOM 更新。`ViewPreferences` 仍保留 version、pageSizes、sortOptions、hideBannerContent、hideSidebar、position；读取／保存只接受这些当前字段，不清空其他偏好。
3. 当前服务端 Preferences 只有 theme 与专用 revision，没有其他活跃字段。删除 `AccountPreferences`、`AccountThemeMode`、更新 RPC／HTTP endpoint、UserInfo 的 preferences 投影及前端模型。profile revision、access revision 和 session generation 保持独立有效。
4. 从 `internal/accountcenter`、`internal/accountstate/store`、账户创建 CTE、初始 schema 中删除主题专用 preferences；保留账户、身份、grant、profile、头像及密钥的事务关系。更新 schema contract，先 SQL／sqlc，再 proto／生成物，再实际消费者。
5. 生成入口为 `make sqlc-local`、`make account-state-schema-contract`、`make protogen`；只在对应源变更后运行，不手改生成文件。根据变更更新现存测试 fixture，不通过宽松解析器保留空 preferences。

这是已确认主题方向的依赖清理，不新增业务服务。API／session 所有者和可信身份传递不变，适用 [SDS-R2、R6、R7、R8](../../developer-guide/service-development-standards.md) 的相关消费者、事务和证据要求；不会新增 RPC、后台进程或跨服务事务。实施须使用 [sync-athena-changes](../../../.codex/skills/sync-athena-changes/SKILL.md)。

当前账户 schema 有严格 catalog 校验；直接修改初始 schema 后，旧开发数据库不能视为新 schema 已就绪。实施验收使用任务专用的新数据库和明确实例，保留已有数据库、卷和其他任务环境。不能以完成 UI 为由自动 reset 或删除共享表；需要让既有开发数据切换 schema 时，单独提供具体数据库处理方案后再执行。此项环境限制必须计入真实验收准备。

## 数据与交互不变量

- 保留两份 React root、realm cookie／header／query、部署 base 与应用 base 的区分；共享组件不借用另一 realm 的数据、draft、cache 或 service。
- 既有账号切换、撤权、abort、single-flight、可见性刷新、晚返回屏障继续有效。请求失败不显示成成功；部分来源失败不隐藏其他有效来源。
- Market Radar 分别使用 hot／realtime／mover DTO；零值用空值判断，不用 `||` 回退到其他金额。Sports 比分与价格分层，Managed OO 保留事件角色和原始 proposed price。
- Wallets 仍为私有钱包管理；敏感值只按既有授权流程显示。Profit Sharing 保留四阶段、封存、匿名与 exact-five，管理员不得获得会员提案操作。
- Worm 保留 0–20 钱包、1–20 执行子集、冻结 revision、五分钟授权、Prepare／授权／Start 分离、USDC 观察门槛、暂停／终止收敛和未知结果只读核对。视觉重构不发明新的订单重试。
- 当前界面语言和业务格式化规则保持；中文说明和字体回退不等于新建语言切换功能。

## 验证与完成标准

验证矩阵对应覆盖清单 S1–S14。证据分四层，不相互替代：

| 层 | 验证内容 | 证据／限制 |
| --- | --- | --- |
| 单元及契约 | 主题偏好删除、其他偏好保留、profile CAS、realm／scope、精准字段和受控交互 | 相关 Jest、Go 单元与集成结果；不为纯颜色声明编写镜像测试 |
| 正式 React 的受控浏览器 | 每条改版入口、共用状态、关键交互及暂缓页共享影响；API 响应明确 synthetic | Playwright request ledger、截图、trace；不能代替供应商或真实身份验证 |
| 自动无障碍与视觉 | 1440／390／320、200% 字号、键盘／focus、各按钮状态和动态颜色 | axe 原始结果、计算样式、截图；截图与 approved 原型作人工对应，不直接用不同 DOM 的原型图充当像素黄金图 |
| 真实本地应用 | 正确 checkout 的两入口、bootstrap、字体／chunk／资源和页面可达性 | 按本地说明准备 `make run`，执行 smoke；现有 shell smoke 只证明壳，页面实际接入另记结果 |

扩大现有 `ui/playwright.config.ts` 的 testMatch，复用 root 与 `/athena` 隔离 harness；正式主题场景的 fixture、配置、请求断言进入 `ui/e2e/`，不能依赖 `.superpowers/` 临时脚本。现有 `test:visual` 只覆盖工具示例，不可据其成功宣称全站视觉通过。

每页桌面／手机主状态都需覆盖，320px 与 200% 字号检查针对清单内所有实际界面。同类辅助状态按矩阵验证，不索取重复审批。计划增加可选 grep 过滤便于任务内验证；最终无过滤执行完整 acceptance 和 a11y，记录实际测试数量及覆盖。操作性接口采用受控 fixture；真实环境中仅执行已授权的本地动作，不为截图调用供应商交易或发送通知。

视觉检查遵循 Impeccable 的有界批次：每个实现批次完整构建、合并检查一次、集中修正，再验证修正。已批准 Inter 与主题搭配是品牌依据；自动扫描误报必须附具体证据并按文件落库，不能全局关闭检查。已有 `DESIGN.md` 和 sidecar 缺失只作为上下文差距记录，本次不据此重开视觉方向或修复工具状态。

全部任务完成、无过滤验收通过、文档同步且临时环境正确收尾后才可声明正式重构完成。当前本文件只完成规划；不运行业务变更，不发送实施完成通知。

## 交付顺序

[实施计划](../plans/2026-09-14-web-ui-theme-refactor.md)分 T1–T10：主题及偏好清理 → 应用壳／组件／验证接线 → 身份与自助 → 管理员运维 → 钱包／Solana／治理 → 市场／赛事／预言机 → Worm 资产与组合 → Worm 预览与执行 → 整体验收及共享影响 → 文档与收尾。各任务沿已确认视觉直接实施，遇到实质业务契约冲突再报告具体差异。
