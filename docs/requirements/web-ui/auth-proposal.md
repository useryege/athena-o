# 会员登录与注册页面视觉基准

> 状态：v14 所展示的登录页与 Google 注册页桌面／手机视觉已确认；正式 `ui/` 未修改。
>
> 范围：现有会员 `/login` 与共享 `/register` 在 v1–v13 已确认视觉体系中的普通扩展。管理员登录、真实身份协议、账户创建、权限授予及生产验收不在本轮范围。

关联现状：[会员登录页](../../../ui/src/app/member/pages/login.tsx)、[共享注册页](../../../ui/src/app/member/pages/register.tsx)、[注册服务](../../../ui/src/app/shared/services/registration-service.ts)和[会员应用入口](../../../ui/src/app/member/app.tsx)；长期身份契约见 [Account Credentials](../../design/identity-access/account-credentials.md)、[Google OIDC Login](../../design/identity-access/google-oidc-login.md)、[Solana Wallet Authentication](../../design/identity-access/solana-wallet-authentication.md)和[账户访问控制](../../design/identity-access/account-access-control.md)。

## 目标与视觉方向

入口只让会员完成眼前的一步：先验证身份，新账户再选择永久用户名。v14 沿用 Nansen 为主要参考的单一深色世界，以及 v1–v13 已确认的背景、面板、白灰文字、青绿色操作层级和克制边框；英文标题、正文和数字继续使用 Inter，Phantom 地址使用 JetBrains Mono。本提案不建立新的设计系统。

页面采用居中单列结构，没有会员侧栏。任务面板桌面最大宽度 448px、内边距 32px；手机外侧留白 20px、面板内边距 24px，低于 360px 时面板内边距降为 20px。标题桌面 28px、手机 24px。Google 与 Phantom 均使用 48px 高的同级 provider 按钮，不虚构推荐关系；管理员授予业务访问的说明位于面板下方，与身份验证及账户创建保持分离。

## 登录信息层级

紧凑 ATHENA 标识先于任务面板。面板依次展示标题与说明、Google 和 Phantom 两个同级入口、Phantom 只签署登录消息且不会创建交易或收取网络费的事实，以及新身份将在下一步创建账户的说明。

Google 与 Phantom 是两种彼此独立的账户身份，不能链接、合并或作为同一账户的第二凭据。Phantom 继续只使用浏览器注入的 Solana provider；手机排版仅表示页面可用，不新增钱包深链。忙碌态阻止重复操作，provider 缺失或登录错误保留当前上下文并提供对应恢复动作；未截图的各 provider 完整错误码仍由正式实现单独验证。

## 注册信息层级与用户名规则

注册页按“已验证身份 → 用户名输入与本地可用性状态 → 永久用户名说明 → 青绿色 Create account 主操作 → provider 对应的身份切换”排列。Google 显示已验证邮箱；Phantom 显示缩写地址，并提供完整地址复制。真实表单初始用户名为空，默认截图中的 `mira.chen` 是为同时审阅输入、可用状态及主按钮而设置的合成数据。

用户名是永久公开名称，创建后不能修改。正式规则为 3–42 个 ASCII 字母、数字、句点或连字符，至少包含一个字母或数字，并拒绝 `0x` 钱包地址形式；完整安全名单、保留名称与可用性结果以服务端为准。原型只在 400ms 后本地检查少量示例不可用名称，不代表完整规则。

注册只在 provider 已验证且服务端签发注册 ticket 后开始。正式实现仍须保留 ticket、CSRF、`returnTo`、取消注册、provider 切换和过期恢复语义；Pending permissions 与账户创建是独立阶段。管理员注册流程不在本提案范围。

## 原型交互与状态边界

[可操作 HTML](previews/theme-auth-v14.html)支持本地演示 Google／Phantom 登录忙碌、Phantom 缺失、身份切换、用户名为空／无效／不可用／检查失败／可用、可用性加载、创建忙碌、复制成功或失败，以及注册 ticket 过期。忙碌时相关动作禁用，并通过异步结果隔离避免旧检查覆盖新输入。

[固定数据](previews/theme-auth-v14-data.json)全部为 synthetic；原型的 sign-in、create、switch 与 recovery 控件只切换本地展示或显示预览反馈，不发出真实 provider、注册或取消请求，也不导航到外部授权。原型没有验证生产 provider、完整服务端用户名安全规则、账户创建、取消写入、CSRF／ticket 竞争、真实 `returnTo` 或部署路径。只有实际截图状态获得本轮视觉审阅，检查记录中的其他本地状态不因此视为逐图确认。

## 截图与浏览器证据

| 类别 | 证据 |
| --- | --- |
| 四张主提案图 | 登录[桌面](previews/theme-auth-v14-login-desktop.png)、[手机](previews/theme-auth-v14-login-mobile.png)；注册[桌面](previews/theme-auth-v14-register-desktop.png)、[手机](previews/theme-auth-v14-register-mobile.png) |
| 七张辅助图 | 登录 [320px](previews/theme-auth-v14-login-narrow.png)、[200% 文字](previews/theme-auth-v14-login-text-200.png)、[手机 Phantom 缺失](previews/theme-auth-v14-login-mobile-missing.png)；注册 [320px](previews/theme-auth-v14-register-narrow.png)、[200% 文字](previews/theme-auth-v14-register-text-200.png)、[手机 Phantom 身份](previews/theme-auth-v14-register-mobile-wallet.png)、[手机 ticket 过期](previews/theme-auth-v14-register-mobile-expired.png) |
| 四张当前 React 对照 | 登录[桌面](previews/theme-auth-v14-before-login-desktop.png)、[手机](previews/theme-auth-v14-before-login-mobile.png)；注册[桌面](previews/theme-auth-v14-before-register-desktop.png)、[手机](previews/theme-auth-v14-before-register-mobile.png) |
| 机器记录 | [原型检查](previews/theme-auth-v14-checks.json)、[当前 React 采集清单](previews/theme-auth-v14-before-capture.json)、[独立审阅记录](previews/theme-auth-v14-review.json) |

11 张提案图和 4 张当前 React 对照图均为带来源记录的 Playwright Chromium 截图，没有 AI 生成或修图；15 张审阅副本逐字节一致，来源扫描记录 15 张、0 缺失。原型在 1440×900、390×844、320×844 与 720×900／根字号 200% 四个视口通过横向溢出、页面错误、外部请求、字体、交互和抽样文字对比度检查，抽样对比度均不低于 4.5。

两轮自身检查只修正禁用按钮仍可出现 hover／pressed 外观，以及截图中指针影响交互外观的问题。独立审阅最初把注册窄屏图显示误判为顶部裁切；随后直接复核原 PNG，并对七种注册状态执行 320×844 边界检查，确认完整品牌标识顶部为 49.171875px 或更低位置，默认状态为 58.8125px，`scrollY=0`、文档高度 844px，因此撤回该发现。审阅没有要求代码修正，最终处置为 `ship`；它只表示提案与本地演示达到交付标准，不代表用户批准、正式实现或生产验收。Inter detector 的单文件例外沿用用户已确认的 v2 字体决定。

当前 React 对照由 1 个 Playwright 测试生成，结果为 1 passed（4.5s），只截获 GET 并生成 4 张截图，没有提交写请求。注册页当前通过普通 `ConfigProvider` 渲染，截图中的低对比度辅助文字、输入与蓝白按钮属于既有界面事实，不是 v14 新回归。正式改版仍需按真实登录和注册流程进行独立实现与 smoke 验收。

## 环境、确认与文档边界

用户于 2026-09-14 查看登录页桌面／手机和 Google 注册页桌面／手机四张截图后反馈“没问题”。本轮确认所展示的居中布局、疏密、文字与操作层级，具体范围见 [v14 确认记录](previews/theme-auth-v14-approval.json)。其他辅助状态、管理员页面和真实身份流程未由本次反馈确认。

本任务没有启动服务。对照采集借用了任务前已有的根目录 Vite `http://localhost:4000`（PID 308228，工作目录 `/home/yege/work/athena/ui`），其 supervisor PID 为 307846，实例为 `athena-local-runtime`，日志位于 `.run/athena-local-runtime/`。所有任务浏览器上下文均已关闭；已有环境按其归属保留，本任务未执行 owner 停止命令 `cd /home/yege/work/athena && make stop`。准确归属、请求及源码散列见[当前 React 采集清单](previews/theme-auth-v14-before-capture.json)。

v14 以四张展示图的确认范围作为后续改版依据。原始 HTML、固定数据、检查、截图和审阅记录保持不变；原始审阅 JSON 保留提交时的 `awaiting_user_review` 状态，当前确认以独立确认记录为准；`ui/` 没有修改。根目录 `DESIGN.md` 与 `.impeccable/design.json` 在本轮前已不存在，Impeccable 上下文检查将其识别为文档漂移；本轮依照限定范围保持该状态，不进行修复。
