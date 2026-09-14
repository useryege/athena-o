# 全站主题一致性修订契约

> 2026-09-14：用户在总体审阅后要求“先修复这些问题”。本契约收敛已确定的视觉规则，配套 [v23 修订稿](previews/theme-consistency-v23/README.md)。它补充 v1–v22 的已确认依据，不改写原件或虚构新的截图审批。正式 `ui/` 尚未实施。

## 最终颜色与唯一来源

运行时以 `ui/src/app/styles/tokens.css` 为唯一色值来源。会员、管理员、Token 钱包分析共用入口 Provider 和字体资产。页面不得再建主题 Provider、根主题标记或私有颜色表。

完整可运行映射见 [theme-reference.cjs](previews/theme-consistency-v23/theme-reference.cjs)，颜色及字体角色见 [tokens.css](previews/theme-consistency-v23/tokens.css)。正式实施把映射转为类型明确的 `athena-color-roles.ts` 与 `athena-theme.tsx`；不在生产代码中导入文档或运行时解析原型。

| 角色 | 常态 | hover | active | 背景／边框 |
| --- | --- | --- | --- | --- |
| 主按钮 | `#00FFA7` | `#51FFC3` | `#09C385` | 三态文字均为 `#06080B` |
| 危险实心按钮 | `#F58C9B` | `#FFB0BD` | `#E5778B` | 三态文字均为 `#06080B`，沿 v4 |
| 成功／正变化 | `#55D9A1` | 不承担品牌按钮状态 | 同左 | `#0E201B`／`#2D5747` |
| 警告／采样不足 | `#E6BF72` | 同左 | 同左 | `#211C13`／`#665231` |
| 错误／负变化 | `#F58C9B` | 同左 | 同左 | `#24171C`／`#673B47` |
| 信息 | `#9BC6F3` | 同左 | 同左 | `#141D27`／`#35516D` |

先执行 Ant 的完整深色算法，再覆盖最终 Map／Alias 角色；将主色作为 seed 输入不足以固定绘制结果。必须覆盖背景、边框、文字、反馈和交互各态，不以 theme 对象里的值代替浏览器计算样式。品牌青绿用于导航、选中、焦点与主要操作；连接成功、收益、损失和警告各用语义变量，仍附文字、符号或图标。

Market Radar 的 `radar-up`、`radar-down`、`radar-warmup` 分别映射语义绿、红、黄；连接状态使用独立 `radar-connected` 类。禁止按页面另写近似色。组件库升级后重新测最终绘制，不依赖本次版本的派生结果永远不变。

## 文字放大与局部重排

- HTML 根字号保留用户默认值，不固定为 16px；16px 只是设计换算基准。页面／区块／正文／控件／辅助分别使用 1.75、1.25、1、0.875、0.8125rem，沿 v2；手机页标题 1.5rem。
- `StyleProvider` 使用 `px2remTransformer({rootValue:16, mediaQuery:false})` 转换普通 CSS-in-JS 长度。**Ant Design 6 的 CSS 变量声明绕过该转换器**，还需固定 `cssVar.key='athena-theme'`，在共用 CSS 对同名 theme 类映射全局与组件的字号变量、字体行高、控件高度。完整桥接见 [corrections.css](previews/theme-consistency-v23/corrections.css)。不能只装转换器就宣称解决字号问题，也不把根字号乘入 seed 后再转换，避免双重放大。
- Ant 的 `fontSize:14` 只用于派生尺寸。最终 `--ant-font-size`、Button contentFontSize、Input inputFontSize 等角色指向公共 rem 变量；普通与大／小号消费都要映射。其余正式页面使用的 Select、Table、Modal、Dropdown、Message 等在 T2／T9 逐项核对实际文字，组件自有像素变量发现遗漏时补到同一个桥接文件。
- 控件至少 2.75rem，按钮允许自动高度和多行文字；字体放大不裁切输入、状态、说明和操作区。固定应用壳断点仍为 900px，不把整套断点强制变成 rem。
- Service Status 的服务表以表格容器宽度和 `44rem` 阈值转分组行。服务与状态保留关联，错误说明独占下一行；不隐藏 `Unreachable`、不截断错误、不用缩小字号解决重叠。
- 顶栏字母头像使用 1.75rem 的宽高和行高 1；继承 v12／v22 的放大方式。身份页大头像保持其自身尺寸角色。

验收顺序固定为“实际文字放大 → 内容边界检查”。同一控件根字号 16→32px 时，14px 的实际字必须变成 28px；正文 16→32px。之后检查整页溢出、状态与错误的矩形交叠、字形是否超出头像、输入和固定操作区是否裁字。320px 重排、根字号模拟、浏览器原生缩放分别记录；前两项不能冒充第三项。

## 共用状态表

| 控件角色 | 已选／当前外观 | 跨页使用规则 |
| --- | --- | --- |
| 侧栏导航 | 深青底 `#102C24`＋青绿文字，500 字重 | 由共同 realm 壳维护，当前链接用 `aria-current=page` |
| 页面页签 | 青绿文字＋底部细线 | 采用原有页签语义，不改成实心按钮 |
| 分段筛选 | 深青底＋青绿文字 | Wallets 类型、订阅筛选等同类选择一致，保持实际选择状态 |
| 数字页码 | 深色面板底＋青绿边框和文字 | 当前页使用 `aria-current=page`；其他页保留次要外观 |
| 可编辑输入 | 白色文字＋强边界，焦点青绿 | 标签说明字段用途，不用 placeholder 代替 label |
| 只读输入 | 较亮层底 `#181D22`＋灰色文字＋分隔线边界 | 显示只读说明，保留 `readonly`、聚焦、选中、复制；不得换成 disabled |
| 禁用输入 | 较亮层底＋灰色文字，虚线边界作为额外区别 | 保持原生 disabled 与禁用原因，禁用光标；不更改按钮的 v4 禁用视觉 |

页码、cursor、Previous／Next、Load more 的业务合同保持独立，共用的是同类状态外观。分页视觉修复不产生新请求参数或统一翻页方式。只读字段在键盘聚焦时仍要显示青绿轮廓。

## 数字、标题和导航图标

可纵向核对的金额／余额列及其表头右对齐，启用 `tabular-nums lining-nums`。本次覆盖 Worm Assets 的 SOL／USDC 余额、Market Radar Hot 的重复数值列；Movers 的分数单独作为数值角色。使用明确的数值列角色（如 `athena-numeric-column`），不靠共享行容器的第几列推断，以免影响 Wallets 的 Type／Source 文字列。独立指标、Outcome、Direction 等文字以及手机摘要不强制右对齐。不截断原始精度，不统一成两位小数，也不将缺失值改成零。

区块标题统一 1.25rem／600，侧栏分组为 0.8125rem／400；页面标题保持 v2／v3。只纠正浏览器默认 700 与局部 18px 的漂移，保留页内信息层级。

| 管理员模块 | 固定图标，沿 v15 |
| --- | --- |
| Accounts | grid |
| Profit Sharing | pie |
| Trader Sync | swap |
| Service Status | activity |
| Etherscan Gateways | api-link |
| Notifications | bell |

正式共同管理员壳维护一份映射，保持同一家族的 1.6 细线笔触；不由资料页另选人物、盾牌或图层图标替代业务导航。

## 两份计划的依赖

全站 T1（公共主题／字体）→ T2（壳与中立组件）→ Token 任务 9（业务页面）。Token 的后端与服务层任务可以独立推进；前端页面实施必须等待 T1／T2 的实现与验证。Token 仅维护业务布局、760px 列表重排和数据状态，复用 `assets/fonts/athena/`，不再创建 `assets/fonts/wallet-analytics/` 或 `html[data-ui-surface='wallet-analytics']` 主题覆盖。

本轮只修订设计与可执行参考。正式 T1–T10 及 Token 接入任务保持未实施；后续实际验收必须重验 Provider、真实浮层、字体加载、业务状态、realm 和权限边界。
