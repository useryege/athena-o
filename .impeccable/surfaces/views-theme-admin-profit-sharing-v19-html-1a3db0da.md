---
version: 1
slug: "views-theme-admin-profit-sharing-v19-html-1a3db0da"
primary_target: "docs/requirements/web-ui/previews/theme-admin-profit-sharing-v19.html"
related_targets: []
---

# v19 管理员 Profit Sharing 视觉方向

模式：Operate。现有 `/admin/profit-sharing` 与 `/admin/profit-sharing/:slug` 在 v1–v18 已确认视觉体系中的普通扩展；正式 `ui/` 不改。

THESIS：列表用于判断每一轮正处在哪个治理阶段、是否已具备下一步条件；详情把唯一当前阶段的控制、五人状态与保密边界放在同一阅读序列中。

OWN-WORLD：沿用 Nansen 单一深色、Inter、JetBrains Mono、v15 管理员壳及 v17/v18 的紧凑操作型页面。列表按阶段与进度扫描，详情只突出当前可执行的生命周期动作。

STORY：管理员从四阶段列表进入 collecting 轮次，先核对 5/5 sealed submissions，再看到 Publish proposals；参与人状态保持可查，内容仍保密。脚本状态覆盖 draft 编辑、voting 匿名提案、closed 最终结果与冲突／失败提示。

FACTS：创建时可填写 title 与 slug；已有 Draft 轮次可编辑标题和不足五人的 roster，slug 保持只读。打开前必须正好五名不重复、仍可登录且有 Profit Sharing 权限的非管理员。打开后 roster 锁定并进入 Collecting；全部提交前不得发布，期间管理员只见状态。发布会同时公开全部匿名提案并开始 Voting，作者与票数隐藏。全员投票后才可关闭；平票自动创建只含最高平票提案的匿名 runoff，直到唯一赢家。更新定义与生命周期操作带 expectedRevision，创建不携带预期版本；409 会丢弃本地编辑并刷新当前轮次。

FORM-SEED：not-applicable；这是沿用已确认世界的普通代码导向扩展，不进行概念方向抽选。

FORM：创建与 Draft 均允许 0–5 名 participant；可添加、移除并从合成 eligible accounts 选择账户，display name／username 身份快照只读，baseline responsibility 可编辑且必填，并校验空字段、重复账户与最多五人。已有 Draft slug 只读；只有保存正好五名有效参与人后才可打开。所有创建、保存、打开、发布、关闭和导航只做本地演示，不发出请求。

FINISH：四张主图覆盖列表／详情的桌面和手机；辅助只使用创建手机、320px 与 200% 文字。其他状态由脚本检查。截图均由本地 Chromium 生成，使用虚构数据，不构成用户批准、生产实现或业务验收。
