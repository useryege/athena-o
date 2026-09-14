# v20 会员基础业务页面集中视觉提案

> 状态：本批展示视觉已确认。用户于 2026-09-14 明确反馈“没问题，审批通过”，整批确认四个实际页面的桌面／手机八张主图，范围见 [v20 确认记录](previews/theme-member-foundations-v20-approval.json)。[批次审阅记录](previews/theme-member-foundations-v20-review.json)保留交付时的 `awaiting_user_review` 历史状态；本次确认不表示正式 `ui/` 已改版或生产验收通过。

## 范围与依据

本批是[其余页面视觉定稿安排](remaining-pages-plan.md)中的第一批：Wallets 私有钱包管理、Solana 发行候选列表、会员 Profit Sharing 轮次列表和轮次详情。四个页面一起交付，每页提供桌面和手机主图；共用配色、字体、导航及组件沿用[全站视觉主题](visual-theme.md)已确认基准，辅助状态作为覆盖证据，不逐图增加审批。

本轮排除 Trader Sync。独立 Token 目录中的 [Nansen 钱包战绩 v1](../token/wallet-analytics-page-proposal.md)已获用户确认，直接沿用。它展示钱包表现，与本批管理私有钱包元数据的 Wallets 是不同业务。

| 实际页面 | 当前路由 | 现有源码与长期契约 |
| --- | --- | --- |
| Wallets 私有钱包管理 | `/wallet` | [会员页面](../../../ui/src/app/member/pages/wallets.tsx)、[Wallet service](../../../ui/src/app/shared/services/wallet-service.ts)、[钱包归属与托管](../../design/identity-access/wallet-ownership.md)、[独立身份复核与租约](../../design/identity-access/wallet-secret-reauthentication.md) |
| Solana 发行候选列表 | `/solana` | [会员页面](../../../ui/src/app/member/pages/solana.tsx)、[Solana service](../../../ui/src/app/shared/services/solana-service.ts)、[业务需求](../solana/README.md)、[现有列表设计](../../design/web-ui/solana-discovery.md) |
| 会员 Profit Sharing 轮次列表 | `/profit-sharing` | [会员页面](../../../ui/src/app/member/pages/profit-sharing.tsx)、[共享展示](../../../ui/src/app/shared/pages/profit-sharing-shared.tsx)、[治理设计](../../design/governance/profit-sharing.md) |
| 会员 Profit Sharing 轮次详情 | `/profit-sharing/:slug` | [会员命令服务](../../../ui/src/app/member/profit-sharing-service.ts)、[共享读取服务](../../../ui/src/app/shared/services/profit-sharing-service.ts)、[治理设计](../../design/governance/profit-sharing.md) |

## 共用视觉与页面组织

三份最终 HTML 延续近黑页面 `#06080B`、面板 `#0F1114`、层次面 `#181D22`、细分隔线 `#252A30`、白色正文、灰色辅助文字 `#9FA0A1` 和青绿色强调 `#00FFA7`。Inter 用于标题、正文和数字，JetBrains Mono 用于地址、哈希和标识；反馈色继续区分成功、警告、错误和信息，不以颜色替代文字。

桌面沿用 224px 会员侧栏、64px 顶栏及 32px 页面边距；手机使用导航抽屉和 20px 页面边距，列式内容重排为连续记录，完整地址换行。标题沿用桌面 28px／手机 24px。会员导航保持 Markets / Token & Risk / Operations 分组，并按当前账户权限过滤空组；Profit Sharing 属于 Operations。本批 Profit Sharing 合成账户没有模块授权，因此只展示其可见的 Profit Sharing 与 Notifications。

### Wallets：名称与完整地址优先

查询、EVM／Solana 类型筛选与钱包记录位于同一面板，页头按 Refresh、Import、Create 排列操作。记录以名称、完整地址为主，类型、来源和时间为辅助；手机让辅助信息使用整行空间。详情集中展示元数据、备注、头像和私钥入口；创建与详情弹窗采用固定标题／操作区、可滚动正文。

- 保留 EVM 和 Solana、1–10 个批量创建／导入、单个钱包可选备注及批量自动命名。备注限制为 50 个 Unicode code point；导入继续逐行展示私钥并校验数量。
- 保留默认头像和现有八个 preset 的身份，不新增业务 preset；本地 SVG 沿用既定图标笔画。上传仍按现有 JPEG／PNG／非动画 WebP、最大 2 MiB 的边界说明。
- 只读权限可查看和复制安全元数据；备注／头像修改及私钥操作遵循当前写权限与交互会话限制。真实创建、导入和查看私钥仅限允许的交互会话；私钥查看需要 fresh identity check，其五分钟租约仅供当前登录会话复用，与 Worm 凭据租约独立。
- 原型的创建与备份结果明确为合成示例；导入、图片上传和身份验证停在本地预览说明，不实际生成／导入密钥、写入头像或认证身份。备份必须勾选确认才能 Done，原生 `cancel` 事件阻止 Escape 绕过；复制失败使用独立手动复制弹窗，保留父备份内容。

### Solana：候选、扫描状态与来源分别表达

页头先显示独立扫描概况，再进入查询与发行候选记录；名称、符号、Mint、发行来源、时间和交易直接可读，初始化证据按需展开。手机保留完整 Mint 和交易标识，将时间并列、证据纵向排列；复制与 Solscan 外链分开操作。

- 范围仍是 Mainnet Beta 首版新 Mint 发行候选，不把候选写成已认证项目，不新增价格、评级或身份推断。
- 提交查询回到第一页；名称／符号匹配忽略大小写，Mint 保留大小写语义；每页固定 25 条。Refresh 只读取已保存的列表与扫描状态，不触发扫描、补全或研究。
- metadata 的 pending、error、unavailable 与发行来源的待处理、失败、未知分别显示；扫描状态与候选列表请求各自保留错误和恢复入口，不用一处错误覆盖其他已知事实。
- Token 程序与发行平台是不同事实；fee payer 只表示付费账户，不推断为创始人；权限表示初始化快照，不声称是当前权限。缺失时间为 Unknown，slot 保持精确十进制字符串，复制及交易外链使用完整值和固定 Solscan 域。
- 样本均为合成数据；本批不启动已暂停的 Solana 采集，不据演示扫描概况判断真实运行健康。

### 会员 Profit Sharing：按阶段呈现个人任务

列表展示 Round、Phase、Proposals、Votes，Draft／Collecting 的 Votes 使用破折号；桌面采用表格，手机按轮次分隔。详情把当前阶段的会员任务放在主要阅读位置，阶段进度提供上下文；Collecting 默认进入自己的方案，Voting 与 Closed 分别提供匿名选择及最终结果。

- 页面入口继续要求登录和 Profit Sharing entitlement，轮次成员关系按账户 UUID 判断，不由显示名称推断。会员命令不包含管理员 Open／Publish／Close。
- Collecting 保留固定五行成员责任及份额。责任最多 500 字符，份额为 0–100%、最多两位小数，输入步长沿用 `0.25`；草稿可不完整保存，提交必须五项完整且合计 100%（10,000 basis points）。
- Submitted 保持 sealed，发布前可 Reopen 并回到 Draft。Voting 隐藏作者和实时票数，禁止投给本人方案，可更新对其他方案的选择；Closed 才展示最终轮候选的作者、票数与赢家。
- 非参与人的 Collecting／Voting 展示阶段对应的只读拒绝状态，不获得编辑或投票能力；Closed 沿用结果读取边界。草稿、提交和 Reopen 继续受 revision 约束；真实 409 丢弃本地草稿并重载的契约保持有效。
- 原型只模拟本地草稿、确认与阶段变化，不提交真实提案或投票；后端权限、revision、竞态及真实阶段流转不由静态演示认证。

## 原型截图与审阅材料

共 27 张 PNG：8 张主图、11 张辅助提案图、8 张当前 React 对照图。桌面为 1440×900、手机为 390×844；窄屏为 320×844，200% 根字号使用 720×900 视口。Wallets 手机详情／创建图取实际弹窗视口，文档和弹窗正文均在顶部、正文可滚动；其余为全页图。

### 四个页面的八张主图

| 页面 | 桌面 | 手机 |
| --- | --- | --- |
| Wallets | [桌面](previews/theme-wallets-v20-desktop.png) | [手机](previews/theme-wallets-v20-mobile.png) |
| Solana | [桌面](previews/theme-solana-v20-desktop.png) | [手机](previews/theme-solana-v20-mobile.png) |
| Profit Sharing 轮次列表 | [桌面](previews/theme-member-profit-sharing-v20-list-desktop.png) | [手机](previews/theme-member-profit-sharing-v20-list-mobile.png) |
| Profit Sharing 轮次详情 | [桌面](previews/theme-member-profit-sharing-v20-detail-desktop.png) | [手机](previews/theme-member-profit-sharing-v20-detail-mobile.png) |

### 辅助覆盖与当前界面对照

辅助图用于验证既有规则在状态、窄屏及放大情况下的应用，不将每张辅助图作为新的用户审批项。

| 范围 | 辅助提案图 |
| --- | --- |
| Wallets，4 张 | [320px](previews/theme-wallets-v20-narrow.png)、[200% 根字号](previews/theme-wallets-v20-zoom.png)、[手机详情](previews/theme-wallets-v20-detail-mobile.png)、[手机创建](previews/theme-wallets-v20-create-mobile.png) |
| Solana，3 张 | [320px](previews/theme-solana-v20-narrow.png)、[200% 根字号](previews/theme-solana-v20-zoom.png)、[手机链上证据](previews/theme-solana-v20-evidence-mobile.png) |
| Profit Sharing，4 张 | [手机匿名投票](previews/theme-member-profit-sharing-v20-voting-mobile.png)、[桌面结果](previews/theme-member-profit-sharing-v20-closed-desktop.png)、[320px](previews/theme-member-profit-sharing-v20-narrow.png)、[200% 根字号](previews/theme-member-profit-sharing-v20-zoom.png) |

| 当前 React 页面 | 桌面 | 手机 |
| --- | --- | --- |
| Wallets | [桌面](previews/theme-wallets-v20-before-desktop.png) | [手机](previews/theme-wallets-v20-before-mobile.png) |
| Solana | [桌面](previews/theme-solana-v20-before-desktop.png) | [手机](previews/theme-solana-v20-before-mobile.png) |
| Profit Sharing 轮次列表 | [桌面](previews/theme-member-profit-sharing-v20-before-list-desktop.png) | [手机](previews/theme-member-profit-sharing-v20-before-list-mobile.png) |
| Profit Sharing 轮次详情 | [桌面](previews/theme-member-profit-sharing-v20-before-detail-desktop.png) | [手机](previews/theme-member-profit-sharing-v20-before-detail-mobile.png) |

### HTML、数据与检查索引

| 范围 | 独立原型 | 合成数据 | 最终浏览器检查 | 当前 React 对照采集 |
| --- | --- | --- | --- | --- |
| Wallets | [HTML](previews/theme-wallets-v20.html) | [JSON](previews/theme-wallets-v20-data.json) | [checks](previews/theme-wallets-v20-checks.json) | [before-capture](previews/theme-wallets-v20-before-capture.json) |
| Solana | [HTML](previews/theme-solana-v20.html) | [JSON](previews/theme-solana-v20-data.json) | [checks](previews/theme-solana-v20-checks.json) | [before-capture](previews/theme-solana-v20-before-capture.json) |
| 会员 Profit Sharing 两页 | [HTML](previews/theme-member-profit-sharing-v20.html) | [JSON](previews/theme-member-profit-sharing-v20-data.json) | [checks](previews/theme-member-profit-sharing-v20-checks.json) | [before-capture](previews/theme-member-profit-sharing-v20-before-capture.json) |

## 验证与独立审阅

最终 Wallets 检查记录 6 张截图、9 项行为；Solana 记录 5 张截图、8 项行为；两份检查均没有整页横向溢出、重复 ID、JavaScript 错误或外部请求。操作区域修正后，Wallets 的 42 个及 Solana 的 70 个几何样本合计 112 个，全部至少 44×44px，覆盖桌面、手机、窄屏、放大及所声明弹窗／证据状态。

Profit Sharing 最终检查覆盖 8 种截图条件和导航、手机抽屉、键盘进入、草稿可不完整保存、提交／Reopen、sealed、匿名禁投本人及 Closed 结果等本地流程。其当前 React 对照的 1 项 Playwright 通过（3.6 秒、4 张图）；Wallets 与 Solana 各有 2 张当前 React 图。全部当前 React 对照仅截获合成 GET，没有提交写请求。

三份 HTML 合并执行过一次 detector，保留五个文件范围例外：三份 Inter 为明确已确认字体；另两项是 Wallets／Solana 的静态 hover 误判。Wallets 实测 `#06080B` 文字配 `#51FFC3` hover 背景，对比度 15.699:1；Solana 没有实际渲染的 primary 按钮。详情见[检测结果](../../../.superpowers/v20-detector-findings.json)及[处置记录](../../../.superpowers/v20-detector-triage.json)，没有全局放宽。

[首次独立审阅](../../../.superpowers/v20-finish-review-initial.txt)提出两项修正：Profit Sharing 恢复既有会员导航分组，Wallets／Solana 补齐复制、外链、分页和类型选择的 44px 命中范围。完成一个修正批后，[最终审阅](../../../.superpowers/v20-finish-review.txt)将两项均判为 resolved，处置为 `ship`；重新检查了 19 张提案图，8 张 React 基线沿用首次审阅。审阅对源码和行为的覆盖限度以原文为准，没有声称逐行审查全部 React、服务和数据文件。

[批次审阅 JSON](previews/theme-member-foundations-v20-review.json)保存最终处置及资产 SHA-256；27 张 PNG 均有来源，来源扫描为 0 缺失，并在 `.impeccable/review/v20/` 保留同字节镜像。本轮基线中的 354 份既有 web-ui／token 预览资产摘要保持不变，另 1 份预览索引 README 追加本批入口；正式 `ui/` 状态保持不变。既有 `DESIGN.md`／`.impeccable/design.json` 缺失不在本批修复范围，未据原型重新制定共用规则。

## 交付边界与后续状态

用户已整批确认四页八张主图，后续改版沿用本批布局、阅读层级及既定主题。原型、截图、数据、检查和交付时审阅记录共 40 份文件保留原字节，摘要见 [v20 确认记录](previews/theme-member-foundations-v20-approval.json)。辅助状态继续按既定规则覆盖与验证，不补写成用户逐图看过，也不追加逐图审批。后续第二批 8 页、第三批 6 页及共用适配仍按[剩余安排](remaining-pages-plan.md)推进。全部设计与覆盖完成后，再形成正式主题重构技术方案和实施计划。

本轮未启动或停止服务／容器，临时浏览器均已关闭。当前 React 对照借用原有 `http://localhost:4000` Vite（仓库 `/home/yege/work/athena/ui`、PID 308228、实例 `athena-local-runtime` legacy、日志 `.run/athena-local-runtime/`），保持原样；所属环境的停止入口为仓库根 `make stop`，本轮未执行。

真实身份验证、钱包密钥托管／写入、Solana 采集、治理权限／提交／投票／竞态以及 full-stack smoke 均未由本批验证。正式 `ui/` 未改版，主题重构未完成，不发送整个实现任务的完成通知。
