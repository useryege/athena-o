Method: dual-agent（A：`/root/theme_design_a` · B：`/root/theme_evidence_b`），主审另做实施计划的独立复现。A 完成前，主审未读取 B 的检测结果；两项评估未读取彼此结论。

# 全站 UI 总体设计审阅 · 2026-09-14

> 审阅结论：Nansen 单一深色方向和页面主体结构可以保留；共用状态、数字排版及实施方案仍有需要修正的地方。尤其当前 Ant Design 配置示例会产生偏色，文字放大方案也未覆盖控件。应先收敛这些共用规则，再按已确认页面实施；无需重开全站审美或逐页审批。
>
> 本次是审阅交付，问题尚未修复。正式 `ui/`、原型、截图和 approval 未修改；只新增本报告、审阅证据及一个已批准 Inter 字体的文件级检测例外。

## 主要发现与优先级

优先处理下面五项；P1 表示应在相应实施任务开始时解决，P2 表示纳入本轮共享层和页面实施。这里评价的是设计与计划，不能将尚未实施的配置问题说成当前正式平台的新回归。

### R1 · P1：同一套颜色变量，尚不能保证最终组件颜色相同

**实施方案有可复现的偏色。** [实施计划 T1](/home/yege/work/athena/docs/superpowers/plans/2026-09-14-web-ui-theme-refactor.md:113)把已批准颜色作为 `theme.darkAlgorithm` 的输入。用当前安装的 Ant Design **6.4.3** 原样复现该配置，经 React 服务端样式提取并由 Chromium 实际计算样式，普通主按钮背景成为 `#03DC91`，并非 [目标色](/home/yege/work/athena/docs/superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md:41) `#00FFA7`。

| 角色 | 已确认值 | 计划配置的最终 token |
| --- | --- | --- |
| 主色 | `#00FFA7` | `#03DC91` |
| 成功 | `#55D9A1` | `#4BBB8C` |
| 警告 | `#E6BF72` | `#C7A564` |
| 错误 | `#F58C9B` | `#D37A87` |
| 信息 | `#9BC6F3` | `#87ABD2` |

原因是这些值属于算法的种子输入，深色算法仍会派生后续颜色；仅写 `colorPrimary` 不等于锁定最后的绘制值。当前安装包的 [颜色派生](/home/yege/work/athena/ui/node_modules/antd/es/theme/themes/shared/genColorMapToken.js:42)与 [seed override 处理](/home/yege/work/athena/ui/node_modules/antd/es/theme/util/alias.js:17)可对应到这一结果。计划虽已要求“核对派生色”，其可执行示例本身仍需修正。

**原型也存在局部颜色角色偏离。** 31 份 HTML 的基础变量相同，但 [Market Radar v21](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-market-radar-v21.html:50)另外写了 `.radar-down:#FF899B`、`.radar-warmup:#EDC474`；`.radar-up` 同时给正变化和 `Sampling connected` 使用品牌青绿。实际访问 `?view=realtime` 已确认这些类可见且生效。共同根变量一致，不能代替对实际使用位置的核对。

**修正建议：** 保留单一 CSS token 来源，在算法派生后的适当层明确映射最终颜色，补齐反馈背景／边框、选中态和各按钮状态；页面业务状态使用明确的角色变量。用真实 Ant 按钮、输入、反馈和浮层验证计算样式及 normal／hover／active，不只检查配置对象。不要让原生 HTML 按钮与 Ant 按钮维护两套最终色。

归属：T1、T2、T6、T9。证据：[计划复现结果](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/ant-plan-probe.json)、[复现脚本](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/ant-token-probe.cjs)、[Market Radar 实测](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/market-semantic-colors.json)。

### R2 · P1：文字放大方案和验收方式仍有盲区

[计划 T1](/home/yege/work/athena/docs/superpowers/plans/2026-09-14-web-ui-theme-refactor.md:120)使用 `fontSize:14`、`controlHeight:44`，而 [文字契约](/home/yege/work/athena/docs/superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md:52)要求 rem 及 200% 根字号适配。独立复现中，根字号从 16px 变为 32px 后，正文由 16px 变为 32px；Ant 按钮、输入和提示正文仍为 **14px**，控件高度仍为 44px。只验证“没有溢出”，可能把没有放大的文字误记为放大验收通过。

原型中另有真实的局部重叠：[Service Status v16 的表格规则](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-service-status-v16.html:12)在 720px、根字号 200% 时，`Unreachable` 与相邻 `Health check timed out after 5 seconds.` 压在一起。状态单元格宽约 146.7px，内容宽 161px；**整页宽度仍为 720px**，仅检查根节点横向溢出无法发现。此实例本身为 P2，说明全站放大验收需要增加局部内容检查。

**修正建议：** 统一原生元素和 Ant 控件的文字尺寸策略，并明确控件随文字增高的规则；验收先确认文字实际变大，再检查单元格、标签、错误说明、头像及固定操作区。Service Status 按内容宽度提前转分组行，或为状态列保留足够空间，保留完整错误文字。根字号模拟与浏览器原生缩放分别记录，不互相冒充。

归属：T1、T2、T4、T9。证据：[Ant 放大结果](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/ant-plan-probe.json)、[Ant 放大截图](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/ant-plan-root-200.png)、[Service Status 重叠截图](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/b/screenshots/theme-service-status-v16-text-200-suspect.png)。

### R3 · P2：同类选中状态和只读状态跨页面不一致

- **页码与筛选：** [Market Radar v21](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-market-radar-v21.html:50)用青绿边框／文字突出当前页；[Worm Combinations v22](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-worm-combinations-v22.html:159)当前页采用普通次要按钮外观。v8 分段选中是深青底／青绿字，v20 钱包类型筛选选中是灰底／白字。
- **只读输入：** [会员 Profile v11](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-account-profile-v11.html:27)通过弱文字与不同底色区分 Username 和 Display name；[管理员 Profile v22](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-common-adaptations-v22.html:156)两者均为白字、近黑底、同一强边框。`readonly` 属性和说明仍在，不存在可非法修改用户名的证据，但视觉上需要额外阅读小字才能发现限制。

这些差异不会改变主色方向，却削弱“同一种外观表示同一种状态”的可预期性。

**修正建议：** 在共享组件中列清导航、页签、分段筛选、页码各自的选中规则；同一组件跨页面保持一致，不能将所有选中项改成实心主按钮。编辑、只读、禁用使用三种明确外观；只读仍可选中和复制，不能用 disabled 代替。分页业务继续保留各模块契约。

归属：T2、T3、T5–T8。证据：[A 的针对性样式](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/a/targeted-styles.json)、[管理员 Profile 手机截图](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/a/common-profile-mobile.png)。

### R4 · P2：可纵向比较的金额列没有统一对齐

[v2 数字规范](/home/yege/work/athena/docs/requirements/web-ui/typography-proposal.md:52)及 Token 钱包分析使用金额右对齐；[Worm Assets 的 SOL／USDC 余额列](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-worm-assets-v22.html:53)和 [Market Radar 重复金额网格](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-market-radar-v21.html:50)采用左侧起始对齐。位数和原始小数精度不同，逐行核对时需要反复横向定位。

**修正建议：** 为可比较的金额列及其表头增加统一数字列角色，使用右对齐和等宽数字；独立指标、文字状态及手机摘要保留适合自身结构的布局。不为对齐而截断原值、改变精度，或强制所有业务统一两位小数。

归属：T2、T6、T7。原图：[Worm Assets](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-worm-assets-v22-desktop.png)、[Market Radar](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-market-radar-v21-hot-desktop.png)；计算样式见 [A 原始记录](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/a/computed-styles.json)。

### R5 · P2：两个实施计划尚未锁定共同主题的依赖顺序

[全站方案](/home/yege/work/athena/docs/superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md:30)将 `tokens.css` 和入口级 Provider 定为唯一来源；[Token 计划 9.2](/home/yege/work/athena/docs/superpowers/plans/2026-09-14-token-wallet-analytics.md:545)仍允许通过 `html[data-ui-surface='wallet-analytics']` 定义页面主题、侧栏、字体和控件，离开时清理标识，并计划另放一套字体。

Token 计划已写“全站公共 tokens 已存在时复用”，因此不是必然重复。但未锁定执行顺序：Token 页面先落地时，仍可能形成一份页面主题与一份全站主题，增加后续清理和跨页维护成本。

**修正建议：** 明确共同主题和共享组件是两个前端任务的前置依赖；Token 后端接入可以独立推进，页面只维护业务布局与已记录的局部适配，不再覆盖全局颜色、字体、壳尺寸或单独控制主题入口。无需扩大为本轮开发 Nansen API。

归属：全站 T1／T2 与 Token 任务 9 的文档依赖。

## 其他需要收敛的内容

| 项目 | 证据与影响 | 处理建议 |
| --- | --- | --- |
| P2：当前状态文档残留旧结论 | [visual-theme.md:249](/home/yege/work/athena/docs/requirements/web-ui/visual-theme.md:249)仍写覆盖与验收“待细化”，[297 行](/home/yege/work/athena/docs/requirements/web-ui/visual-theme.md:297)写矩阵与命令尚未确定；277 行“每轮只推进一个主题”也与之后整批审阅规则不符。页首及现有覆盖／计划已经给出新状态。 | 清理长期文档的当前状态表和执行说明，指向已形成的 coverage/spec/plan；保留原始 HTML、review、approval 中的历史状态。 |
| P3：管理员图标跨页改变 | [v15 导航](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-admin-accounts-v15.html:69)与 [v22 共用适配](/home/yege/work/athena/docs/requirements/web-ui/previews/theme-common-adaptations-v22.html:151)中，Accounts、Service Status、Gateways 的图标分别改变；文字仍明确，因此影响较小。 | 由共同管理员壳使用固定“模块→图标”映射，保留细线笔触，不按页面另画一套。 |
| 已有实施规范可解决的标题差异 | v21／v22 部分区块标题为 20px／700，Saved combinations 为 18px／700；共同规范已经明确 20px／600。 | 按现有 v2 标题角色收敛，不重新选择字体、不重做主图。 |
| P3：早期头像的放大适配 | v3 与 Token 原型在 200% 根字号下头像仍为 28px，字形超出圆圈；v12／v22 已使用能随文字放大的尺寸。 | 纳入 R2 的共享壳验证，直接沿用后期已验证方式。见 [对照测量](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/b/confirm.json)。 |

## 已经统一、应该保留的部分

- **核心颜色关系成立。** 31 份独立 HTML 的 page／panel／raised／line／text／muted／accent 根变量一致，实际页面背景都是 `#06080B`。R1 针对最终绘制值与局部角色，不否认已经确立的主题。
- **主字体和框架稳定。** v2 之后的页面标题与正文使用 Inter；地址使用 JetBrains Mono。v3 之后有壳页面的侧栏 224px、顶栏 64px。v1 只确认配色、v2 只确认字体，不将其早期壳尺寸当成后期回归。
- **业务信息结构有针对性。** 私有 Wallets、外部钱包分析、服务健康、通知状态、Worm 授权与执行分别组织；不应为了视觉整齐而合并独立状态或改变业务流程。
- **覆盖清单没有发现遗漏入口。** 当前源码与机器清单一致：会员 35、管理员 16，共 51 项；37 改版、2 Appearance 删除、8 Trader Sync 暂缓重排、4 路由兜底。Token 仍属独立功能计划。证据：[本次路由复核](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/route-review.json)。

Trader Sync 专属布局暂缓、共用主题仍生效的边界是清楚的。当前正式源码的橙色／Heebo／双主题属于已记录的待替换实现，不能当作本次设计稿的新回归。

## 验证范围与限制

| 检查 | 本次结果及边界 |
| --- | --- |
| 视觉审阅 A | 查看覆盖 v1–v22 的代表图及 Token 两端，访问 30 份 Web UI HTML + 1 份 Token HTML 默认视图，并另查管理员 Profile 变体；不等于所有内嵌页面和辅助状态都已重新操作。 |
| 浏览器测量 B | 31×4＝124 个默认视图：1440×1050、390×844、320×844、720×1000 且根字号 200%。整页溢出 0、脚本错误 0、资源失败 0；存在 R2 的单元格重叠，因此不宣称响应式完全通过。 |
| 主按钮抽样 | 17 个默认面上可见且启用的主按钮，default／hover／active 共 51 个样本；静态原型均为深色按钮文字，测得最低文字对比度约 8.74:1。不能由此替代正式 Ant 组件验证。 |
| 检测器 | 现有配置下 web-ui 0 项，Token 1 项 `overused-font: inter`，已按批准字体处置。扫描使用既有文件级例外，0 项不表示所有基础信号从未出现。 |
| 独立实施计划复现 | 当前本地 React 18.3.1、Ant Design 6.4.3、Playwright 1.60.0／Chromium 148；复现 T1 配置的最终色和字号，未将该配置写入正式源码。 |
| 未执行 | 全栈 make run、正式 React 全页验收、真实登录／供应商请求／交易／通知、完整读屏和浏览器原生缩放。此次设计审阅不需要借用业务服务证明静态设计。 |

五个代表页面的检测覆盖层有四个注入成功；Token 的 CSP 拒绝外部 detect.js，未移除或绕过该限制，使用其 CLI、计算样式及截图作为后备证据。使用的是无头浏览器，没有向用户声称应用里存在可见的检测覆盖层。

Inter 为明确批准的字体。新加的例外只针对 `docs/requirements/token/previews/wallet-analytics-v1.html` 的 `overused-font=inter`，通过 `impeccable hooks ignore-value` 保存；未关闭规则、未忽略整个文件。v15 开关的 aria-label 被当成可见文字、sr-only 的有意裁切、原型页脚行长等信号均经过核实，没有加入产品问题清单。

## 静态提案的启发式参考分

独立视觉审阅 A 为 **28／40**。这是设计表达的参考分，不是正式产品验收分，也不包含对未来 Ant 实现正确性的认证。

| 项目 | 分数／4 | 主要依据 |
| --- | ---: | --- |
| 系统状态可见 | 3 | 时间、局部失败、保留结果和执行阶段明确 |
| 现实语言对应 | 3 | 对象清晰，部分领域术语仍需说明 |
| 用户控制 | 3 | 草稿、返回和取消路径已有表达 |
| 一致性 | 2 | 共用状态与数据对齐尚未收敛 |
| 错误预防 | 3 | 对象、权限和操作后果明确 |
| 识别优于记忆 | 3 | 核心身份可见，图标／选中差异增加识别成本 |
| 使用效率 | 2 | 有筛选与批量操作，真实键盘效率未验 |
| 审美与简洁 | 3 | 主题克制，按业务组织信息 |
| 错误恢复 | 3 | 旧结果、未知与失败有不同表达 |
| 帮助与文档 | 3 | 任务附近有说明，真实资源可达性未验 |

高频数据用户主要受金额对齐和选中态影响；依赖清晰视觉线索的用户主要受只读字段和文字放大问题影响；手机上被打断后返回的用户需要稳定的当前筛选、对象和操作位置。整体已有分组与按需展开，不依据菜单项总数机械地要求重新设计导航。

## 建议的修正顺序

1. 修正 T1 的最终颜色映射和文字缩放方案，补充对应验收断言。
2. 收敛 T2 的只读／禁用／选中状态、数字列和模块图标；保留各页业务结构。
3. 将 Service Status 放大重叠加入 T4／T9 的具体检查案例；已有标题与头像差异沿现成规范处理。
4. 明确 Token 页面依赖共同主题，清理长期文档的过时当前状态，再按原实施顺序推进。

本次不新增待用户选择的审美方向。上述清单是对既定目标的实施修正建议，不代替用户确认未讨论的新业务，也不表示已执行修复。

原始证据：[A 报告](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/a/review.md)、[B 报告](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/b/review.md)、[124 视图清单](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/b/coverage.csv)、[审阅前后核验](/home/yege/work/athena/.superpowers/ui-theme-review-20260914/pre-report-verification.json)。
