---
version: 1
slug: "ui-previews-theme-notifications-v10-html-0841f20f"
primary_target: "docs/requirements/web-ui/previews/theme-notifications-v10.html"
related_targets: []
---

# Notifications 视觉提案

模式：Operate。范围是现有会员 `/notifications` 页面在已确认 v1–v9 体系中的扩展；代码原型与截图用于讨论，正式 `ui/` 不修改。沿用用户逐页确认和最新“舒服，继续设计”的授权。

## Direction contract

THESIS：先辨认当前 Telegram 连接，再完成一次新的设置。现有连接和未完成尝试必须各自明确；不把 Connected 当成已读或某条通知已送达。

OWN-WORLD：沿用 Nansen 参考的近黑页面、深色面板、白灰字、青绿操作及 v4 语义反馈，Inter 正文与数字、JetBrains Mono 手动命令，既有会员壳和确认弹窗。

STORY：查看连接身份和绑定时间；需要连接时按三步操作，用同一链接或二维码去 Telegram，再返回核对；旧绑定保留至新设置成功。失败、过期与其他标签页的尝试各给明确恢复入口。

FIRST VIEWPORT：页头 Notifications 与说明；主面板上方渠道名称/状态，中部身份时间与右侧连接/解绑操作，底部旧通知处理说明。设置区在其下方，桌面左侧三步指引/手动命令、右侧二维码；手机先步骤后二维码和有效期。未连接、连接中和已连接分别呈现对应操作。

FORM：已有世界的代码原型，seed 不适用。原生三步序列只表达实际配置流程，不是装饰。主要按钮聚焦当前下一步，解绑入口描边危险色，确认时使用实心危险色。所有恢复操作保留当前源的权限和状态规则。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

二维码、连接资料及计时均为明确的演示数据，链接指向 example.invalid 的保留域名；点击只本地说明，不打开 Telegram、不发送消息、不改变真实绑定。作为普通视觉扩展，本次只写页面提案、截图、检查及索引，不修复预先缺失的 DESIGN.md 和 sidecar。v10 尚待用户截图确认。
