---
version: 1
slug: "ui-previews-theme-account-access-v13-html-d8b2c5bd"
primary_target: "docs/requirements/web-ui/previews/theme-account-access-v13.html"
related_targets: []
---

# Account Center / Access & session v13

范围：现有会员 /account/access 的视觉提案，沿用 v1–v12 已确认体系。用户已授权继续逐页设计并用截图确认；本轮不改正式 ui/、API、授权规则或管理员页面。

Mode: Operate. 会员在现有桌面／手机工作区核对登录身份和模块授权。数据为明确标注的虚构 Google／Phantom 身份、权限与版本值。

## Direction contract

THESIS: 先读当前模块权限，再按需核对会话细节；待授权单独呈现身份已验证与业务未开通，不能混为登录失败。

OWN-WORLD: 沿用已确认的 Nansen 单一深色主题、Inter、地址专用 JetBrains Mono、青绿主操作和克制分隔线。夜间与长时间数据阅读沿用用户固定偏好的深色场景。

STORY: 核对当前账户，看见各模块授权；必要时展开时间与版本。待授权用户可复制完整身份、刷新权限或退出；样板所有业务动作只做本地演示。

FIRST VIEWPORT: 保留 Account Center 页头和紧凑身份，桌面左侧账户导航，右侧主要面板展示 Module access 摘要与十条源码展示模块。手机依序排列标签与权限。Current session 位于权限面板之后，辅助时间／版本默认收起。待授权页用单面板说明与青绿 Refresh permissions。

FORM: 已有账户中心的普通扩展，code-led；无开放概念竞选，seed: not applicable。签名交互为可键盘展开的会话核对区，完整字段可达。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 边界与交付

- 现有 PRODUCT.md、主题需求和 v12 HTML 是上下文；DESIGN.md 与 .impeccable/design.json 先前缺失，普通扩展保持其状态。
- accountAccessDisplayModules 保持现有十项展示（不显示 Token）；完整授权仍使用全部模块。权限只读，不新增开通申请、权限编辑、升级套餐或终止所有会话。
- Pending 判定沿用 loginEnabled / administrator / profitSharingEnabled / moduleAccess；仅 API Key 可用不等于业务已开通。被封锁账户不伪装为 pending。
- Google / Phantom 标签与完整身份按源码；15 秒及窗口聚焦刷新是当前产品行为，静态演示不执行轮询或授权请求。
- 交付桌面、手机、展开会话、待授权 Google／Phantom、320px 与 200% 字体截图；当前 React 使用相同虚构 fixture 对照。真实刷新、退出、轮询、身份切换与权限竞态留待正式实现验收。
- QUALITY BAR: 默认视图能直接区分三种权限，待授权无误导成功态，长地址完整换行，技术信息可展开，键盘与窄屏均可用。
