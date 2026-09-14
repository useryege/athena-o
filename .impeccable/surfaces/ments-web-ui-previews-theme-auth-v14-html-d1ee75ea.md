---
version: 1
slug: "ments-web-ui-previews-theme-auth-v14-html-d1ee75ea"
primary_target: "docs/requirements/web-ui/previews/theme-auth-v14.html"
related_targets: []
---

# v14 会员登录与注册视觉提案

Mode: Operate。已获授权继续当前页面视觉确认；范围仅现有会员 /login 与共享 /register 的会员状态。采用既有主题进行有限扩展，正式 ui/、身份协议、权限及管理员页面不在本次修改范围。

## Direction contract

THESIS: 入口只要求用户完成眼前的一步：验证身份，然后为新账户选择永久用户名。保留真实业务内容，不增加邮箱密码、交易承诺或营销区域。

OWN-WORLD: 沿用已确认 v1–v13 的近黑背景、深色面板、白灰文字、青绿色操作、Inter 与地址用 JetBrains Mono；不改变字体及品牌资产。

STORY: 登录者选择 Google 或 Phantom。新身份在下一页核对已验证身份，了解用户名不可更改，检查可用后创建账户。业务访问仍需管理员另行授予。

FIRST VIEWPORT: 居中单列，紧凑 ATHENA 标识在面板上方；桌面面板宽 448px、内边距 32px，手机外侧 20px、面板内侧 24px。登录标题后是两个同级 48px 提供商按钮；注册身份块后是带即时状态的用户名输入与青绿主按钮。说明贴近相关动作，没有会员侧栏。

FORM: 普通扩展、code-led；直接延续当前登录/注册单列任务布局和已批准视觉系统，用户已同意逐页截图确认。该精确范围不进行 concept-seed，seed key 为 not-applicable。主要交互是用户名可用性状态就地更新、忙碌期间防止重复操作。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 证据与边界

参考 docs/requirements/web-ui/visual-theme.md、v2 字体、v4 组件及 v13 账户页。实际源码为 ui/src/app/member/pages/login.tsx、register.tsx、shared/services/registration-service.ts 及 member/app.tsx。官方提供商标识复用仓库资产。Google 与 Phantom 是独立身份，不提供账号合并或自动关联。Phantom 仍只使用浏览器注入的 Solana provider，手机布局不等于新增深链支持。

原型不向服务发起授权、注册或取消请求；演示数据标注 synthetic，提交只回显预览提示。当前 React 对照仅截获 GET fixture。检查 1440、390、320 和 720px/200% 字号，批量查看截图，至多两轮自身检查；随后独立 finish reviewer 和 documenter。无需生成栅格素材，截图全部嵌入来源。

## QUALITY BAR

标题、按钮与身份信息在桌面和手机均清晰；两种登录方法无虚构推荐关系；永久用户名警示在主操作前；错误保留已有上下文；按钮有键盘焦点、错误有文本而非仅颜色；窄屏和 200% 字号无横向溢出。保留注册会话过期、身份切换、可用性与加载边界的说明。沿用已确认字体的单文件 detector 例外应披露。

## 文档与未决项

本轮交付仅设计提案，等待用户按展示图片确认。普通扩展保留原有视觉系统；根 DESIGN.md 与 .impeccable/design.json 预先不存在，报告漂移但不修复。完成后维护 auth-proposal.md、visual-theme.md 和三份索引，不将未确认提案写成已批准，也不修改 AGENTS.md 或 PRODUCT.md。
