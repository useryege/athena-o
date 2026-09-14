---
version: 1
slug: "i-previews-theme-account-profile-v11-html-5322f7b3"
primary_target: "docs/requirements/web-ui/previews/theme-account-profile-v11.html"
related_targets: []
---

# Account Center / Profile 视觉提案

模式：Operate。现有会员 `/account/profile` 共享页面在 v1–v10 已确认视觉世界中的普通扩展。沿用逐页设计、截图确认和最新“可以，继续设计”的授权；只维护文档原型，不实施正式 `ui/`。

THESIS：一眼分清不可变的账户用户名和可修改的显示名称，以单个资料表单完成名字修改，并把头像操作和未保存状态交代清楚。

OWN-WORLD：沿用用户确认的 Nansen 单一深色、近黑页面、深色面板、白灰文字、青绿色主操作、语义反馈。Inter 正文数字，既有全局壳、边框与 v4 弹窗。无新增身份风格或概念抽签。

STORY：从账户菜单进入 Profile；在资料面板识别当前保存的身份、Standard 展示层级与 Member 角色；按需修改头像或显示名称；核对未保存变化后保存，或 Reset。离开时保护尚未保存的草稿。用户名只读，tier 不作为授权证明。

FIRST VIEWPORT：页头 Account Center，下方桌面账户分区导航与一张资料面板。把原有重复的身份头／头像编辑合并为面板内的头像和身份一行，随后按 Username、Display name、状态和保存操作阅读。手机以 Account section 选择器替换局部侧栏。

FORM：代码原型，沿用既有世界；无需新 identity seed/comp。为全站已确认的单一深色移除 Appearance 导航和说明；保留其余浏览器偏好及未来账户访问和 Security 页面。会员 Security 仅在 apiKeyEnabled 为真时显示，管理员不扩展为已确认。无新增套餐、账户 UUID、邮箱编辑或身份绑定功能。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

本轮普通扩展仅维护页面提案、原型、截图及索引，保留预先缺失的 DESIGN.md 与 .impeccable/design.json。所有资料为虚构；原型只做本地编辑、校验、重置和离开确认，业务保存／上传／删除均不请求真实服务。没有头像时使用与当前代码一致的首字母回退，不伪造头像照片。头像删除入口的有图状态、实际上传、冲突刷新与身份切换是后续正式实现契约。
