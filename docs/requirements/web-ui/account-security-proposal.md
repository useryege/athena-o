# Account Center / Security 页面视觉基准

> 状态：v12 所展示的桌面／手机安全设置页及手机连接说明弹窗视觉已确认；正式 `ui/` 未修改。
>
> 范围：现有会员 `/account/security` 在 v1–v11 已确认视觉体系中的普通扩展。页面只管理 ATHENA 签发的账户 API Key 与 AI HTTP 接入说明，不配置 Nansen 平台 Key 或外部 AI 提供商密钥。

## 目标与视觉方向

页面应让会员先理解“为 AI 生成一次性完整接入说明”和“管理已签发 API Key”是同一套账户凭据的两种使用入口，再在动作发生处核对完整权限、一次性展示与撤销后果。`Credential ready` 只表示 ATHENA 接受了属于预期账户的凭据，不表示任何外部 AI 已连接。

v12 沿用已确认的 Nansen 单一深色方向与 v1–v11 基准：近黑 `#06080B` 页面、`#0F1114` 面板、白／灰文字、青绿 `#00FFA7` 主要操作、克制边框和既有警告／错误／成功语义。英文标题、正文与数字使用 Inter，凭据载荷与技术标识使用 JetBrains Mono。Account Center 页头、局部导航与紧凑当前账户身份沿用相邻 [Profile 页面基准](account-profile-proposal.md)，没有新增视觉世界。

## 页面结构与访问边界

桌面在 Account Center 页头下保留左侧账户导航，右侧先显示以 `Connect AI` 为主要操作的接入面板，再显示以 `Create API key` 为次要操作的密钥面板。密钥列表只显示源码已有的 `ID`、`Issued`、`Expires` 与 `Revoke`；桌面使用紧凑表格，低于 800px 后改为分隔行，不叠加卡片。手机沿用 Account section 选择器，按当前账户、Connect AI、API keys 的顺序阅读。

Security 入口只对 `!isAdmin && access.apiKeyEnabled` 的普通会员显示。功能关闭时当前路由会重定向至 Account Center 或 Access；管理员没有本轮 Security 页面实现或视觉确认。目标单一深色界面继续不显示 Appearance。

本页签发的是 ATHENA 账户 API Key。Nansen API 仍使用平台环境变量和共享配额，其决定不因本提案改变；页面也不收集、保存或验证外部 AI 提供商密钥。

## 创建 API Key

- `API Key ID` 必填，使用原始 ASCII 规则 `^[A-Za-z0-9][A-Za-z0-9._-]*$`，最多 64 字符；提交时 trim。Connect AI 的默认 ID 继续由 `createAIConnectionID` 生成。
- 有效期选项保持当前值：90 days（默认，`7776000`）、30 days（`2592000`）、1 year（`31536000`）和 No expiration（`0`）。样例中的 UTC+8 只描述固定 fixture；生产时间继续使用当前浏览器 locale／timezone 语义，不建立新的业务时区规则。
- 正式实现必须保留单飞创建、generation 与 account ID 防陈旧写回。账户变化或组件卸载时，中止验证，并清理待撤销、已签发 secret 与表单状态。创建尚未完成时不得关闭或取消弹窗。
- API Key secret 只显示一次。结果弹窗仅以 `Done` 关闭，不提供 Escape、点击遮罩或关闭图标；完整只读值保持可见并可执行 `Copy API key`，不得从列表元数据重建 secret。
- 创建失败继续显示真实 `requestErrorMessage`；忙碌状态、服务端竞态和授权撤销属于正式实现验收，不能由固定样例成功提示代替。

## Connect AI 与一次性结果

Connect AI 使用同一账户 API Key 签发流程，并按当前 `buildAIConnectionDetails` 生成完整说明：base URL、`llms.txt`、`swagger.json`、`GET api/v1/session/userinfo` 验证入口、预期账户 UUID、完整 Bearer 值、执行步骤与当前权限说明均须保留。结果弹窗固定标题和操作区，中间正文独立滚动；手机长说明滚动时，`Copy complete setup`、`Copy key only` 与显式 `Done` 始终可见。

验证状态保持精确：checking 表示正在核对，ready 只表示 ATHENA 以该凭据返回了当前预期账户，failed 表示核对失败。checking 与 failed 仍允许复制；只有 failed 提供 Retry。正式验证请求继续使用 `credentials: omit`、`cache: no-store` 与 `AbortSignal`，并同时匹配已登录状态和 account ID。任何状态都不得写成外部 AI 已经连接。

一次性结果承载账户当前 API 权限。页面不新增 OAuth、scope、只读模式、逐请求审批、provider、最近使用、状态或调用量；现有要求人类交互登录的敏感 API 限制也不能因 Connect AI 被移除。

## 撤销与列表状态

撤销确认显示精确 ID，并明确该凭据会立即失效、使用它的客户端会失败；默认安全焦点在 `Keep key`。确认成功后生产列表移除对应元数据，失败继续显示真实错误。列表仅来自账户拥有的 metadata snapshot，绝不展示或推断 secret。

正式改版必须保留首次加载、空态、错误、刷新失败保留旧值以及账户归属快照；账户变化、权限撤销、组件卸载和服务端竞态需要独立验收。管理员、禁用 API Key 的会员和陈旧账户结果不能借用当前 fixture 状态。

## 原型数据与交互边界

[共享数据](previews/theme-account-security-v12-data.json)使用明确标注的虚构账户、三条密钥元数据、无效合成 secret 与保留域名 `example.invalid`。可操作 HTML 支持表单校验、结果弹窗、复制、样例状态、撤销确认和本地导航提示；它不签发或撤销密钥，不执行验证或外部 AI 请求。演示撤销保留原列表并显示本地 notice；复制只复制完整合成载荷。真实 API 错误在生产中继续来自 `requestErrorMessage`，不能改成固定演示文案。

13 张 PNG 全部是带来源记录的 Playwright Chromium 截图，没有 AI 生成或修图，其中 11 张为新主题原型，2 张为当前 React 对照。当前 React 使用同一身份和三条 token metadata，仅截获 bootstrap 与 token-list 两个 GET；1 个测试在 2.1 秒内通过并生成两张截图，没有写请求。原型在 1440×900、390×844、320×844 和 720×900／根字号 200% 四个视口通过横向溢出、页面错误、外部请求、字体、对比度和弹窗几何检查。

两轮自身检查修正了仅供辅助技术使用的标题隐藏方式、800px 紧凑表格阈值、大字模式账户头像、复制 SVG 符号，以及 AI 结果弹窗固定标题／操作区与滚动正文。Impeccable detector 仅保留本 HTML 的 Inter 提示忽略，依据是 v2 已确认字体；该例外不扩展到其他文件。

当前 React 对照复用了任务前已存在的根目录 Vite `http://localhost:4000`（PID 308228），采集前已核对进程工作目录为 `/home/yege/work/athena/ui`。本任务没有启动服务，所有新建浏览器上下文均已关闭，已有服务保持原样；详情见[当前 React 采集记录](previews/theme-account-security-v12-before-capture.json)。

## 审阅与确认边界

[v12 独立审阅记录](previews/theme-account-security-v12-review.json)覆盖 13 张截图和本地原型，处置为 `ship`，未提出需要修正的视觉问题。审阅确认其忠实延续已确认的主题、字体、组件语义与账户中心结构，并核对一次性结果、焦点、响应式布局和元数据可读性。该结论只表示视觉提案与本地演示达到交付标准，不是用户视觉批准、生产验收或正式实现完成。

用户于 2026-09-14 查看桌面默认页、手机默认页和手机 AI 连接说明弹窗后反馈“舒服，继续”。三张截图的布局、疏密、阅读顺序与操作层级成为已确认基准，具体范围见 [v12 确认记录](previews/theme-account-security-v12-approval.json)。其他辅助状态未逐图确认；HTML、截图、检查及原始审阅记录保持不变。正式实现还需验证真实签发、撤销、验证请求、账户切换、异步防护、权限撤销与服务端失败。

根目录 `DESIGN.md` 与 `.impeccable/design.json` 在本轮前均不存在；v12 作为既有视觉世界的普通扩展继续保持缺失状态。
