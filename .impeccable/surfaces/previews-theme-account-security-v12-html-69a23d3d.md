---
version: 1
slug: "previews-theme-account-security-v12-html-69a23d3d"
primary_target: "docs/requirements/web-ui/previews/theme-account-security-v12.html"
related_targets: []
---

# Account Center / Security 视觉提案

模式：Operate。现有会员 `/account/security` 在 v1–v11 已确认视觉世界中的普通扩展。用户明确继续安全设置页设计；本轮交付原型、截图和文档，正式 `ui/` 不改。

## Direction contract

THESIS：让用户区分“生成一次性 AI 接入说明”和“管理已经签发的密钥”，把复制与撤销的后果放在发生动作的地方。

OWN-WORLD：沿用 Nansen 参考、单一深色、Inter 和已确认账户中心导航；近黑页面、深色面板、青绿主操作与克制边框。用户在金融工作台内核对凭据，保持与相邻资料页一致的阅读环境。

STORY：先选择 Connect AI 或 Create API key；输入名字与有效期；核对并复制一次性结果；从元数据列表按名称撤销。Credential ready 只表示 Athena 凭据验证通过，不宣称外部 AI 已连接。

FIRST VIEWPORT：Account Center 页头下，桌面左侧账户分区导航，右侧先 Connect AI 再 API keys；上区突出 Connect AI，下区显示 ID、Issued、Expires、Revoke。手机使用既定分区选择器和逐项密钥摘要。一次性结果强调完整复制，危险弹窗显示精确 ID 与即时失效后果。

FORM：已批准世界的代码原型扩展，无新的种子、概念抽签或 comp。功能、过期选项、一次性显示、完整说明与验证含义取自现有源码；不添加 OAuth、权限范围选择、调用量、最近使用、外部 AI 在线状态或管理员 API Key。交互保持可见默认状态，仅使用既有焦点、按钮和弹窗反馈；尊重减少动效设置。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

普通扩展保留全局 DESIGN.md 与 .impeccable/design.json 原先缺失状态，不顺带修复。两轮内完成自身视觉检查，再交给独立审阅和文档化。原型使用明确虚构的密钥、账户和保留域名；允许本地复制演示数据，所有签发、验证、撤销与导航只做本地演示，不发出真实业务请求。当前 React 对照复用已核对归属的根 Vite 4000，所有 API 被 fixture 拦截；不启动或停止共享服务。
