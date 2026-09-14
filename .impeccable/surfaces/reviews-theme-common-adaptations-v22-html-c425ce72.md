---
version: 1
slug: "reviews-theme-common-adaptations-v22-html-c425ce72"
primary_target: "docs/requirements/web-ui/previews/theme-common-adaptations-v22.html"
related_targets: []
---

# v22 共用页面适配

Mode: Operate（身份、账户、反馈）；Read（Help）。普通扩展，沿用 v11/v13/v14/v15 视觉。正式 ui/ 不改，Appearance 单深色清理为未来实现决定。

## Direction contract

THESIS: 管理员身份与自助账户和共用帮助采用已批准结构，清楚表达当前身份、权限和恢复动作，不增加注册流程或帮助内容。

OWN-WORLD: Nansen 近黑 #06080B、面板 #0F1114、抬升层 #181D22；白灰文本、mint #00FFA7；Inter 常规UI，JetBrains Mono 仅标识。沿用224px管理员壳与64px页头。

STORY: 管理员Google登录，新身份复用 /register?athenaRealm=admin；用户核对管理员标记、永久用户名。Profile编辑显示名；Access展示模块和会话。Help展示源码3固定资源、会员API许可下Connect AI。反馈保留两个realm真实恢复路径。

FIRST VIEWPORT: 身份页居中448px面板；账户页在左侧导航后展示标题、用户与页签；Profile一面板，Access模块权限先于会话。Help为可扫描的资源分隔行。手机按单列阅读并使用抽屉导航。

FORM: code-led ordinary extension；FORM-SEED not-applicable，直接沿用已确认体系。关键交互是用户名校验、显示名草稿确认与可恢复反馈。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## QUALITY BAR

六实际页面桌面/手机，320与200%字号，无整页横溢/JS异常/重复ID；44px交互，键盘/焦点/错误/加载/空配置信息保留。每最终PNG打开，首轮批量检查、一次修正、最多一次确认；root独立review/documenter。只本地模拟，不真实授权、注册、写档案或请求供应商。反馈为状态非虚构独立路由；会员无模块权限跳/account/access，无独立403页。Help不构造chatUrl/binaryUrls配置。现有DESIGN.md/sidecar缺失仅报告。
