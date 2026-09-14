# Notifications 页面视觉基准

> 状态：v10 所展示的已连接桌面／手机及桌面重新连接视觉已确认；正式 `ui/` 未修改。
>
> 范围：现有会员 `/notifications` 页面在 v1–v9 已确认视觉体系中的普通扩展。仅整理当前 Telegram 私聊连接与设置，不新增收件箱或其他通知渠道。

## 目标与视觉方向

页面先回答“当前连接到哪个 Telegram 账户”，再引导用户完成新的设置。当前连接与进行中的设置是两个独立层次：已有绑定在替换成功前保留；原来处于 Connected 的连接继续有效，取消新设置不改变原绑定；页面不能把 Connected 表达成消息已读或某条通知已送达。

v10 沿用 Nansen 参考和此前已确认的 ATHENA 单一深色体系：近黑 `#06080B` 页面、`#0F1114` 面板、白／灰文字、青绿 `#00FFA7` 主要操作与克制边框。英文标题、正文、数字继续使用 Inter，手动命令使用 JetBrains Mono；状态反馈和危险确认沿用 v4 语义色及弹窗层级。视觉原型见[证据索引](previews/README.md)。

## 页面结构

页头保留 Notifications、当前仅支持 Telegram 的说明和刷新操作。Delivery channels 主面板直接展示 Telegram 状态、连接身份、绑定时间（UTC+8）及可用操作，并保留旧 Trader Sync 通知在解绑或重绑时的处理说明。

存在设置尝试时，Telegram setup 与当前连接分开显示。设置按三步组织：

1. 打开 Athena Bot；
2. 在 Telegram 点击 Start，让一次性链接匹配当前登录的 Athena 账户；
3. 返回 Athena 等待连接状态更新。

同一设置同时提供 Telegram 链接、手动 `/start` 命令和二维码；二维码打开同一一次性链接。桌面把步骤和手动命令置于左侧、二维码置于右侧，手机按步骤、手动命令、二维码、有效期顺序纵向排列。

## 状态与恢复规则

- Connected 直接显示 Telegram 身份与绑定时间。Reconnect 开始独立替换尝试；旧绑定保留到新绑定成功，Cancel new setup 后旧绑定不变。
- Pending 显示一次性链接、二维码、手动命令、剩余时间和到期时间。若尝试来自其他标签页且本标签页没有指令，说明浏览器标签页边界，并提供重新创建或取消入口。
- Expired 与 Failed 的设置恢复区只保留一个主要操作 Create new link，另保留次要 Cancel setup，避免同一恢复动作重复出现。Failed 徽标使用错误色；到期和需要注意使用琥珀警告色。
- Unreachable 保留当前绑定，提示用户检查 Bot 是否被阻止或创建新链接；重连成功前不清除绑定。
- Bot unavailable 时禁用 Configure／Reconnect，但已有绑定仍允许 Disconnect。
- 首次加载失败不展示虚构状态；刷新失败时保留上次成功读取的连接详情并显示来源错误消息。生产实现必须继续使用实际服务错误文案，不能以原型的固定提示覆盖。
- Disconnect 弹窗显示当前身份和完整后果：停止向该账户发送、取消设置、无发送许可的旧 Trader Sync 通知被取消、已获许可的通知仍可能到达，重连后不补发历史。取消或关闭恢复焦点，确认才执行解绑。

身份显示延续现有优先级：展示名与 `@username` 都有时同时显示，仅有 username 时显示 `@username`，仅有展示名时显示展示名。时间统一通过现有 UTC+8 格式显示。页面只表达当前源码支持的 Telegram 私聊设置，不推导新的渠道、消息列表或投递管理能力。

## 正式实现契约

以下来自当前 React 与服务实现，是未来生产改版必须保留并重新验证的行为；本次静态原型没有验证它们：

- 设置读取和写操作使用真实 API；开始设置、取消尝试、解绑的并发与 abort／generation 竞态保护继续有效。
- 页面可见时每 3 秒单飞轮询，窗口聚焦和可见性变化触发刷新；进行写操作时不并发刷新。
- 有效期每秒更新，过期后清除与当前 attempt 对应的本标签页指令。
- deep link 与手动命令仅保存在创建它们的 `sessionStorage` 标签页；服务端 attempt 为权威状态。账户切换不能复用另一账户的指令或 Trader Sync owner 草稿。
- 从 `/trader-sync/add` 带有效 owner 草稿进入时保留条件式返回入口；草稿失效或账户变化后不显示错误返回路径。
- Bot 可用性、binding 与 attempt 状态来自服务响应；重连成功、取消、解绑及失败后的刷新保持当前状态清理规则。
- 保留 `requestErrorMessage` 等实际来源错误处理，避免把网络错误、读取失败、不可达绑定与设置失败合并为同一种状态。

## 原型数据与交互边界

[共享数据](previews/theme-notifications-v10-data.json)是固定时间 `2026-09-14T07:00:00Z` 的虚构 fixture；Telegram 身份、链接、命令、二维码和到期时间均非真实。二维码使用保留域名 `example.invalid`。可操作 HTML 中连接、重连、取消设置、刷新、打开 Telegram 和确认解绑等业务动作只显示本地 notice，不打开 Telegram、不发起 API 写入、不改变绑定。复制仅写入演示命令；导航折叠、样例状态切换及弹窗开关可以在本地操作。

原型检查覆盖 1440×900、390×844、320×844 和 720×900／根字号 200%，核对无页面横向溢出、页面错误、外部请求与重复 ID，并检查弹窗正反向焦点循环、Escape、焦点恢复、复制载荷、字体加载及色板对比度。15 张最终截图覆盖桌面／手机 connected、pending、disconnect，以及 reconnect、其他标签页、expired、stale、unreachable、error、failed、窄屏和文字放大状态。Impeccable detector 仅对这份 v10 HTML 持久忽略 `overused-font=inter`：Inter 是用户在 v2 明确确认的字体，此例外不扩展到其他文件或其他检查项。

四张当前 React 对比图覆盖桌面／手机 connected 与 pending。采集使用同一虚构 fixture，仅发出 fixture GET，请求结果为 1 个测试通过并生成 4 张截图；这不是完整 smoke、后端验收或真实 Telegram 行为验证。环境和进程归属见[当前界面采集记录](previews/theme-notifications-v10-before-capture.json)：本任务 Vite 5620 已停止，用户／共享环境保持原样。

## 确认边界

v10 最终原型已经过两项审阅修正：Expired／Failed 恢复区合并为单一主要 Create new link，Failed 徽标改用错误色而到期／注意保持琥珀色；修正后检查全部通过。[v10 审阅记录](previews/theme-notifications-v10-review.json)的限定 verdict 将这两项均评为 resolved，且在三张变更截图中未观察到批次回归，处置为 `ship`。该 verdict 只覆盖两项修正及批次回归，不是新的整页审阅、用户批准或生产验收。

用户于 2026-09-14 查看已连接桌面／手机及桌面重新连接完整截图后反馈“可以，继续设计”。这些截图的分区、密度和操作主次作为已确认基准，范围见 [v10 确认记录](previews/theme-notifications-v10-approval.json)。首次设置、手机设置、解绑弹窗及其他辅助状态未逐图确认；正式 `ui/` 仍保持当前实现。

根目录 `DESIGN.md` 与 `.impeccable/design.json` 在本轮前均不存在。v10 是既有视觉世界的普通扩展，本轮保持该缺失状态，不在页面提案中补建全局设计系统文件。
