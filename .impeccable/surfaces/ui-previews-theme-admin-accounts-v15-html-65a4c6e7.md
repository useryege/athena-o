---
version: 1
slug: "ui-previews-theme-admin-accounts-v15-html-65a4c6e7"
primary_target: "docs/requirements/web-ui/previews/theme-admin-accounts-v15.html"
related_targets: []
---

# v15 管理员导航与账户管理视觉提案

Mode: Operate。用户已授权继续逐页设计。本轮覆盖既有管理员导航与 /admin/accounts 的目录、权限、资料和身份；仅制作可审阅原型，不修改正式 ui/。

## Direction contract

THESIS: 从账户目录直接进入权限核对。桌面保留左右目录/详情布局，将原先纵向堆叠的三个区域改为页签，默认展示 Access。

OWN-WORLD: 沿用已确认 Nansen 参考的单一深色主题：#06080B 背景、#0F1114 面板、#181D22 控件底、#252A30 分隔、白灰文字和 #00FFA7 强调；Inter 正文和数字、JetBrains Mono 地址与标识符。沿用 v3 导航和 v4 表单/确认层级。

STORY: 管理员按身份查找账户，核对状态，调整授权，阅读敏感变更影响再保存；资料与身份在相邻页签，当前管理员账户保留只读边界。

FIRST VIEWPORT: 桌面 224px 管理员侧栏、64px 页头、32px 内容边距。标题下是一行搜索/状态筛选，左侧约 304px 账户目录，右侧详情先显示身份摘要和三个页签；Access 开始于三项开关，随后是按组排列的模块权限和保存行。手机 20px 内容边距，目录与详情分步展示，保留返回目录和导航抽屉。

FORM: 普通扩展、code-led，seed key 为 not-applicable。延续已确认系统和现有目录结构，不进行视觉世界选择。关键交互是原地保留草稿、变更影响确认和手机列表/详情切换；只用短暂颜色反馈，无装饰性动画。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 业务与证据边界

导航仅含 Accounts、Profit Sharing、Trader Sync、Service Status、Etherscan Gateways、Notifications，管理员身份清晰。单深色主题不再提供 Appearance。其他导航目的页和管理员账户中心留待后续设计，点击原型只说明范围。

以 admin-accounts.tsx、accounts-service.ts、access-modules.ts 和 models.ts 为准。完整授权聚合为 11 项，当前显示 10 项，隐藏 Token 原值始终保留。Trader Sync 仅 No access / Read & write；Solana 仅 No access / Read only。当前管理员权限只读，自身资料在 Account Center 修改。用户名只读，名称与 tier 独立保存，tier 不授予权限，头像沿用现有格式与大小边界。

前后截图使用相同虚构账户。当前 React 只拦截 GET fixture；本地原型不调用服务。校验 1440、390、320 和 720px/200% 字号，批量打开截图，最多两轮自检，然后独立 finish reviewer 和 documenter。截图嵌入来源，不生成或修改栅格像素。

## QUALITY BAR

默认能看到正在管理的身份及权限；搜索、选择、保存层级清楚。手机不同时堆叠目录和详情。全地址可换行、不会被截断；文本对比度、焦点、标签、未保存提示和敏感确认完整。空态/加载/错误保留恢复入口，原型模拟反馈不冒充真实保存。

## 文档与环境

完成后维护 admin-accounts-proposal.md、visual-theme.md 与三份索引，状态为待确认，不更新 AGENTS.md/PRODUCT.md 的批准记录。DESIGN.md 与 .impeccable/design.json 原先缺失，本次普通扩展不修复漂移。借用 root 仓库既有 Vite localhost:4000，保留其进程；结束关闭本次浏览器。
