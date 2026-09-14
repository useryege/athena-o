---
version: 1
slug: "iews-theme-member-profit-sharing-v20-html-2d4ee370"
primary_target: "docs/requirements/web-ui/previews/theme-member-profit-sharing-v20.html"
related_targets: []
---

# v20 会员 Profit Sharing 视觉方向

模式：Operate。既有 `/profit-sharing` 与 `/profit-sharing/:slug` 在已确认 Nansen 单深色会员世界中的普通代码导向扩展；正式 `ui/` 不改。

FORM-SEED：not-applicable；本页沿用已确认视觉体系与现有源码任务，不进行概念方向抽选。

THESIS：列表帮助会员快速判断轮次阶段和个人下一步；详情将当前阶段的唯一会员任务置于首屏主序列，阶段进度只作为决策上下文。

OWN-WORLD：沿用近黑页面、平面深色面板、细边框、mint 主操作、Inter 正文数字及 JetBrains Mono slug；复用会员导航的 Markets / Token & Risk / Operations 分类、64px 顶栏、桌面 32px 与手机 20px 页面边距，Profit Sharing 归入 Operations。导航继续按当前账户权限过滤空分组；本合成会员没有模块权限，因此只显示 Operations 下已授权的 Profit Sharing 与无模块门槛的 Notifications，不显示禁用的 Token。

STORY：默认详情展示 Collecting 参与人编辑自己的五行方案、精确 100% 汇总、Save draft 与 Submit proposal；Voting 只呈现匿名方案并禁止选择本人方案；Closed 才显示作者、票数和赢家。非参与人只获得阶段对应的只读拒绝提示。

业务契约：页面路由先受当前登录与 Profit Sharing entitlement 约束；round membership 使用 Athena account UUID。Collecting 草稿可不完整保存，每项 responsibility 最多 500 字符，share 为 0–100%、最多两位小数，提交要求五项完整且合计 10,000 basis points；已提交方案保持 sealed，可在发布前 reopen。Voting 可提交或更新一个他人方案选择，自己的方案不可选，作者和实时票数隐藏；Closed 显示最终轮候选、作者、票数和赢家。409 丢弃本地改动并重载，其余错误保留实际请求错误语义。

QUALITY BAR：四张主图覆盖列表／Collecting 详情的桌面与手机；辅助图覆盖 Voting 手机、Closed 桌面、320px 与 200% 文字。全部控件 44px 可触达，移动端表格转为连续记录，长内容不横向溢出，键盘可打开轮次、编辑并完成本地确认流程；侧栏、手机抽屉和面包屑均使用现有会员导航命名、图标与权限过滤结果。

范围：所有导航、保存、提交、reopen 与投票仅本地模拟，不发送 API；合成数据固定用于当前 React GET 对照和原型。制品用于本批用户审阅，不代表用户已批准、正式 UI 已实现或生产写操作已验收。
