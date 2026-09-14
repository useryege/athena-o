# Global System Prompt

## Table of Contents

- [Rules](#rules)
  - [服务独立性与开发边界](#服务独立性与开发边界)
  - [Project Status](#project-status)
  - [Requirement-Driven Architecture](#requirement-driven-architecture)
  - [Test Data and API Keys](#test-data-and-api-keys)
  - [No Historical Compatibility](#no-historical-compatibility)
  - [Superpowers Development Workflow](#superpowers-development-workflow)
  - [本地验收环境准备与完成标准](#本地验收环境准备与完成标准)
  - [Chinese Plans](#chinese-plans)
  - [Plan Implementation Completion Email](#plan-implementation-completion-email)
  - [Impeccable Integration](#impeccable-integration)
  - [本地图片路径规则](#本地图片路径规则)

## Rules

### 服务独立性与开发边界

新服务或服务边界改造必须先阅读并遵守[服务开发规范](docs/developer-guide/service-development-standards.md)。该规范以 `SDS-R1` 至 `SDS-R8` 定义服务边界、API、独立构建运行、授权、事务、本地编排和验证证据；设计与 PR 必须按变更适用范围引用相应规则和证据。

既有实现的差距不得新增耦合；无关的小修复不因此强制全系统重构。纯解析器或内部工具库无需为了该规范拆成服务。

### Project Status

The project is currently in development and has not been launched yet. Necessary breaking changes are allowed, and backward compatibility with historical behavior is not required.

### Requirement-Driven Architecture

The current service architecture is designed to satisfy the current requirements and must not be treated as a permanent constraint. Future requirements may expose limitations in the existing architecture or make those requirements inconvenient to implement. When that happens:

- Prioritize satisfying the requirements over preserving the existing architecture.
- Freely perform breaking refactors or replace existing component boundaries, data flows, and technology choices when needed to provide a clear and effective implementation.
- Introduce additional infrastructure or technology stacks, including Kafka, Redis, RabbitMQ, or other appropriate systems, whenever the requirements justify them.
- Do not treat the current architecture or technology stack as immutable.

### Test Data and API Keys

This repository is in development. All data in this repository is test data, and any API keys are keys used for testing only. Do not redact, mask, obfuscate, or otherwise desensitize these values.

### No Historical Compatibility

Do not preserve historical compatibility. When implementing changes:

- Do not add migration paths, feature flags, dual code paths, or shims solely to keep old behavior working.
- Do not retain deprecated APIs, fields, or data formats unless they are still actively used in the current design.
- Prefer direct replacement and cleanup over gradual deprecation.
- Breaking changes are acceptable when they simplify the codebase or align with the current design.

### Superpowers Development Workflow

- Use the upstream Superpowers skills in `.agents/skills/` as the development workflow for this repository. At the start of a task, read and follow [using-superpowers](.agents/skills/using-superpowers/SKILL.md), including its Codex platform reference, then use the applicable skills.
- Follow Superpowers for brainstorming, design approval, planning, implementation, debugging, code review, and completion. Apply the user's current request and existing authorization; do not ask again for an unchanged decision already made in the task.
- Follow [test-driven-development](.agents/skills/test-driven-development/SKILL.md) for behavior changes and [verification-before-completion](.agents/skills/verification-before-completion/SKILL.md) before claiming success. Relevant tests and verification are part of implementation; they do not require a separate request. Honor explicit task-specific exceptions from the user.
- Read the relevant [requirements](docs/requirements/README.md), [designs](docs/design/README.md), and actual source as project context. Preserve business decisions, unresolved questions, and scope; keep affected long-term documents consistent with the resulting implementation. Document statuses describe facts, not additional workflow gates.
- Use Superpowers' default `docs/superpowers/specs/` and `docs/superpowers/plans/` locations when its selected workflow calls for written artifacts. Project-local worktrees belong in `.worktrees/`; temporary Superpowers execution state belongs in `.superpowers/`.
- Project-specific skills supply domain knowledge and repository operations alongside Superpowers. Installation details and the skill map are in [Superpowers Development](docs/developer-guide/superpowers-development.md).
- 选择开发、调试或验证工具时，查阅[按任务选择工具](docs/developer-guide/toolchain-guide.md#按任务选择工具)，根据当前问题和影响范围选用，并在当前工作区确认所选工具可用。工具清单不是每个任务的必跑清单；用户或任务方案已经要求的验证仍须完成。

### 本地验收环境准备与完成标准

- 已授权的本地验收包含必要的环境检查、准备、启动、排查和重验。服务未启动是待处理的前置条件，不能仅凭首次连接拒绝、预检失败或其他测试通过就结束必要的真实环境验收。
- 先确认目标地址、运行进程及其所属仓库/worktree。正确且健康的已有环境直接复用；未启动时按[本地运行说明](docs/developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance)选择项目 Node 版本，从目标仓库执行 `make run`，保存持久运行会话及日志。这些必要操作无须再次请求用户确认。
- smoke 命令本身不启停开发服务；执行验收的代理负责准备环境。不得把工具的职责边界解释成代理不能执行 `make run`。启动后检查进程、前端入口和会员/管理员 bootstrap，再执行真实 smoke 并检查退出结果和报告；端口监听或 HTTP 200 不等于验收通过。
- 失败时保留证据，检查相关日志，在已授权范围内解决环境问题并重验。只有确实无法自行解决的外部依赖、凭据、权限或需要用户决定的问题，才报告阻塞，写明尝试、原因和未完成项；验收任务本身不授权扩大为产品行为修改。
- 必要的真实验收未通过时，区分“实现完成”和“验收未完成”，不得宣称整个任务完成，也不得发送整个任务的完成通知。隔离测试、受控场景和预检不能替代真实环境验收。
- 开发、调试或验收任务结束时，默认停止本任务启动的临时服务、预览、测试替身及所属容器，保留数据库、数据卷、日志、截图和验收报告。完成、取消、暂停，以及保存证据后以失败或阻塞结束任务，均须收尾；任务仍在持续调试或验收时可保持运行，不以单次回复结束作为停服时点。
- 用户在任务前启动的服务、其他任务正在使用的环境和借用的共享基础设施保持原样。长期主开发环境须由用户明确指定；只有用户明确要求继续查看或保留现场调试时，才保留本任务中对应的环境，不能据此保留所有分支和测试实例。
- 收尾由执行任务的代理主动完成：先核对仓库/worktree、实例和资源归属，再从同一仓库使用带相同 `INSTANCE` / profile 的 `make stop`、`make stop-instance` 或对应工具的停止入口；独立测试替身和预览也须按记录停止。不得自动执行 `make run-reset`、删除数据卷、停止借用的基础设施或终止归属不明的进程。
- 停止后核对所属进程退出、端口释放和容器停止；收尾失败须保留日志、排查并报告未停止项，不得仅凭停止命令已执行就宣称完成。交付时列明已停止和仍保留的环境；保留项注明用户要求或已有环境归属，并提供地址、仓库/worktree、实例、会话或进程、日志、验收结果及准确停止命令。
- 用户明确要求只读、仅预检或不启动服务时遵循该限制，仅报告实际检查结果。规则文档修改、纯隔离测试不因此自动扩大为启动真实环境。

### Chinese Plans

When outlining or listing a plan (implementation steps, task breakdown, approach summary, etc.), write the plan in Chinese (简体中文).

### Plan Implementation Completion Email

After a user-confirmed plan has been fully implemented, including all required code, configuration, documentation, and verification, send exactly one completion email from the repository root before returning the final response:

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：<简短任务名称>' \
  TASK_NOTIFICATION_BODY='已完成：<核心成果>；验证：<验证结果>。'
```

- Do not send this notification when only the plan has been written, or when implementation is incomplete, blocked, failed, or cancelled.
- Write a short subject and body in Simplified Chinese. Do not include passwords, tokens, API keys, or any other secrets.
- Use the default `.env` notification configuration; this rule does not switch to `.env.prod`.
- Wait for the Make command to finish before returning the final response.
- If the notification still fails after the command's built-in retries, keep the implementation task complete but report the notification failure and a credential-safe error summary in the final response. Do not claim that the email was sent.

### Impeccable Integration

- Use Superpowers to guide the development process and Impeccable for UI/UX expertise under `ui/`. Read the relevant requirements, `docs/design/web-ui/` documents, and current React/CSS implementation as context.
- 全站 UI 的主要视觉参考已明确为 Nansen，其中主题与配色搭配是用户特别强调的核心审美偏好。后续设计、重构与视觉验收必须读取并遵循[全站视觉主题需求](docs/requirements/web-ui/visual-theme.md)，整体对照其近黑背景、深色面板、白／灰文字、青绿色强调与边框的颜色关系，以及文字、空间与交互风格，采用单一深色主题；具体效果按用户逐项确认的设计落实。现有橙色／双主题仅描述尚未替换的实现，不能当作新设计方向。
- [视觉样板](docs/requirements/web-ui/previews/README.md)的 v1 配色与按钮层级、[v2 字体与数字排版](docs/requirements/web-ui/typography-proposal.md)、[v3 导航／页头／页面密度](docs/requirements/web-ui/layout-proposal.md)视觉效果已确认，后续设计须沿用：Inter 用于英文标题、正文与数字，JetBrains Mono 用于地址与哈希，采用已确认的字号层级、数字对齐及会员样板布局。[v4 第一组通用组件与状态](docs/requirements/web-ui/components-proposal.md)所展示的表单、反馈配色、加载／空态及桌面／手机确认弹窗视觉也已确认。其余组件状态、其他业务／管理员页面和业务展示语义仍按各项确认范围处理，不将样板认可扩大为正式 UI 全部设计已完成。
- 本轮 UI 设计主旨是重构当前 `ui/` 下会员端、管理员端及共享组件的全站前端界面。钱包样板用于确认共用视觉规则；[v5 列表控件](docs/requirements/web-ui/list-controls-proposal.md)的展示视觉也已确认，后续页面级设计应以现有代表业务页面的改版为载体，最终按完整设计实施与验收。
- [v6 Trader Sync 首页](docs/requirements/web-ui/trader-sync-home-proposal.md)的整页布局、桌面／手机展示与辅助信息默认折叠已获用户确认。后续该页改版沿用活动／目标分区和阅读层级，身份快照、Position ID 与来源入口默认收起、按需展开，完整钱包和成交事实仍直接显示；具体范围以 [v6 确认记录](docs/requirements/web-ui/previews/theme-trader-sync-v6-approval.json)为准，不扩展为未展示状态或正式 UI 已实现。
- [v7 添加交易员页](docs/requirements/web-ui/trader-sync-add-proposal.md)所展示的桌面／手机布局、信息密度与操作层级已获用户确认。后续该页改版沿用资料与 P/L 核对、订阅确认分区和手机阅读顺序；具体范围以 [v7 确认记录](docs/requirements/web-ui/previews/theme-trader-add-v7-approval.json)为准，未展示的辅助状态不由本次反馈整体确认，正式 UI 尚未改版。
- [v8 订阅列表](docs/requirements/web-ui/trader-sync-subscriptions-proposal.md)所展示的桌面四列、手机分组、信息密度与监控／通知状态区分已获用户确认。具体范围以 [v8 确认记录](docs/requirements/web-ui/previews/theme-trader-subscriptions-v8-approval.json)为准；Cancelled 列表及其他辅助状态未逐图确认，正式 UI 尚未改版。
- [v9 订阅详情](docs/requirements/web-ui/trader-sync-subscription-detail-proposal.md)所展示的桌面／手机布局、操作层级、手机取消弹窗及历史默认摘要／按需展开方式已获用户确认。具体范围以 [v9 确认记录](docs/requirements/web-ui/previews/theme-trader-detail-v9-approval.json)为准；未直接展示的辅助状态不由本次反馈整体确认，正式 UI 尚未改版。
- [v10 Notifications](docs/requirements/web-ui/notifications-proposal.md)所展示的已连接桌面／手机布局、操作主次及桌面重新连接流程已获用户确认。沿用当前连接与新设置分区，明确旧 Connected 连接在替换前继续有效；具体范围以 [v10 确认记录](docs/requirements/web-ui/previews/theme-notifications-v10-approval.json)为准，未直接展示的辅助状态未逐图确认，正式 UI 尚未改版。
- [v11 Account Center / Profile](docs/requirements/web-ui/account-profile-proposal.md)所展示的桌面资料页、手机编辑状态与手机离开确认视觉已获用户确认。沿用合并头像与身份的单面板布局、只读用户名与可编辑显示名称区分，以及所展示草稿对照和操作层级；具体范围以 [v11 确认记录](docs/requirements/web-ui/previews/theme-account-profile-v11-approval.json)为准，未直接展示的辅助状态和管理员页面未逐图确认，正式 UI 尚未改版。
- [v12 Account Center / Security](docs/requirements/web-ui/account-security-proposal.md)所展示的桌面／手机安全设置页及手机连接说明弹窗视觉已确认。沿用 Connect AI 与 API keys 的阅读顺序、桌面表格／手机分隔行，以及手机连接说明的滚动正文和固定操作区；具体范围以 [v12 确认记录](docs/requirements/web-ui/previews/theme-account-security-v12-approval.json)为准，未直接展示的辅助状态和管理员页面未逐图确认，正式 UI 尚未改版。
- [v13 Account Center / Access & session](docs/requirements/web-ui/account-access-proposal.md)所展示的桌面／手机权限与会话页及手机 Google 待授权页视觉已确认。沿用模块权限在前、当前会话在后及辅助信息默认收起的层级，以及所展示手机待授权身份提示和刷新／退出操作主次；具体范围以 [v13 确认记录](docs/requirements/web-ui/previews/theme-account-access-v13-approval.json)为准，未直接展示的辅助状态和管理员页面未逐图确认，正式 UI 尚未改版。
- [v14 会员登录与注册](docs/requirements/web-ui/auth-proposal.md)所展示的登录页与 Google 注册页桌面／手机视觉已确认。沿用居中单列布局、Google／Phantom 同级登录入口，以及所展示 Google 注册页的身份核对、用户名提示与创建／切换操作层级；具体范围以 [v14 确认记录](docs/requirements/web-ui/previews/theme-auth-v14-approval.json)为准，未直接展示的辅助状态和管理员页面未逐图确认，正式 UI 尚未改版。
- [v15 管理员导航与账户管理](docs/requirements/web-ui/admin-accounts-proposal.md)所展示的桌面账户管理、手机目录、手机权限详情和手机确认弹窗视觉已确认。沿用桌面目录／详情布局、默认 Access 的阅读层级、手机目录到详情的分步展示，以及敏感确认中说明影响、纵向全宽按钮的操作层级；具体范围以 [v15 确认记录](docs/requirements/web-ui/previews/theme-admin-accounts-v15-approval.json)为准，未直接展示的辅助状态和其他管理员页面未逐图确认，正式 UI 尚未改版。
- [v16 管理员 Service Status](docs/requirements/web-ui/service-status-proposal.md)所展示的 Services 桌面／手机、Notifications 桌面和 Trader Sync 手机视觉已确认。沿用三来源页签与独立状态、服务记录、通知恢复／队列分区和手机指标的阅读层级；具体范围以 [v16 确认记录](docs/requirements/web-ui/previews/theme-service-status-v16-approval.json)为准，未直接展示的辅助状态和其他管理员页面未逐图确认，正式 UI 尚未改版。
- [v17 管理员 Etherscan Gateways](docs/requirements/web-ui/etherscan-gateways-proposal.md)所展示 Gateways 与 Live Probe 的桌面／手机视觉已确认。沿用网关健康与请求测试的独立来源、网关记录、表单／结果层级、By Gateway 明细及所展示诊断入口；具体范围以 [v17 确认记录](docs/requirements/web-ui/previews/theme-etherscan-gateways-v17-approval.json)为准，未直接展示的辅助状态和其他管理员页面未逐图确认，正式 UI 尚未改版。
- 后续 UI 审阅按[设计差异分组](docs/requirements/web-ui/visual-theme.md#确认方式按设计差异分组)：已确认的共用规则直接沿用，同类页面和桌面／手机合并审阅；历史“未逐图确认”不自动成为新增审批关卡，只有新的信息结构或关键操作差异需要单独呈现。全部页面和辅助状态仍须覆盖与验证。
- [v18 系统通知列表与详情](docs/requirements/web-ui/system-notifications-proposal.md)所展示的桌面／手机视觉已确认，按 [v18 确认记录](docs/requirements/web-ui/previews/theme-system-notifications-v18-approval.json)落实。后续每次先完成多个实际页面，再集中交付审阅；共用视觉规则直接沿用，不以单页的多个状态替代多页面批次。正式 `ui/` 尚未改版。
- [v19 管理员业务页面](docs/requirements/web-ui/admin-business-batch-proposal.md)所展示的 Trader Sync 订阅列表／概要、Profit Sharing 轮次列表／详情及桌面／手机八张主图已获用户整批确认，无须调整；按 [v19 确认记录](docs/requirements/web-ui/previews/theme-admin-batch-v19-approval.json)落实。后续沿用本批布局、阅读层级和既定 Nansen 主题配色，不重复确认相同决定；辅助状态继续覆盖与验证，正式 `ui/` 尚未改版。
- [v20 会员基础业务页面](docs/requirements/web-ui/member-foundations-batch-proposal.md)所展示的 Wallets 私有钱包管理、Solana 发行候选、会员 Profit Sharing 轮次列表／详情及桌面／手机八张主图已获用户整批确认；按 [v20 确认记录](docs/requirements/web-ui/previews/theme-member-foundations-v20-approval.json)落实。沿用本批布局、阅读层级和既定 Nansen 主题配色，辅助状态继续覆盖与验证；本轮剩余页面按[分批安排](docs/requirements/web-ui/remaining-pages-plan.md)推进，Trader Sync 本轮排除，正式 `ui/` 尚未改版。
- [v21 市场、赛事与 Managed OO 页面](docs/requirements/web-ui/market-intelligence-batch-proposal.md)所展示的 Market Radar 三页、Sports 三页及 Managed OO 两页的桌面／手机十六张主图已获用户整批确认；按 [v21 确认记录](docs/requirements/web-ui/previews/theme-market-intelligence-v21-approval.json)落实。后续重构沿用本批布局、数据阅读层级、操作主次及既定 Nansen 主题配色，辅助状态继续覆盖与验证，不重复确认相同决定。本轮页面进度以[分批安排](docs/requirements/web-ui/remaining-pages-plan.md)为准，Trader Sync 本轮排除，正式 `ui/` 尚未改版。
- [v22 Worm Trading 与共用页面](docs/requirements/web-ui/worm-and-common-batch-proposal.md)所展示的 Worm 六个实际业务页面及管理员登录／共享注册／Profile／Access、会员／管理员 Help 六项共用适配的桌面／手机二十四张主图已获用户整批确认；按 [v22 确认记录](docs/requirements/web-ui/previews/theme-worm-and-common-v22-approval.json)落实。沿用本批布局、阅读顺序、操作层级与既定 Nansen 主题搭配，辅助状态继承覆盖与验证，不追加逐图审批。三批 18 个业务页面及本批共用适配的主视觉均已确认；[总覆盖清单](docs/requirements/web-ui/theme-refactor-coverage.md)、[技术方案](docs/superpowers/specs/2026-09-14-web-ui-theme-refactor-design.md)与[实施计划](docs/superpowers/plans/2026-09-14-web-ui-theme-refactor.md)已整理。Trader Sync 专属页面暂缓重排，共享主题影响须回归；正式 `ui/` 尚未改版。
- 总体审阅后的[一致性修订契约](docs/requirements/web-ui/theme-consistency-contract.md)与 [v23 参考及证据](docs/requirements/web-ui/previews/theme-consistency-v23/README.md)补充最终颜色、字号放大、组件状态、金额列和管理员图标；后续共享层按修订落实。全站 T1／T2 是 Token 前端的明确依赖；本次仅完成设计修订，正式源码仍未改版。
- Reuse the design approval obtained through Superpowers. Layout diagrams and visual previews may support that design discussion without a separate repository-specific layout gate.
- Derive any Impeccable context files from the repository's requirements, designs, and approved task spec; keep those references aligned with the project facts.
- Treat automatic Impeccable hook findings as review input. Do not weaken confirmed behavior, accessibility requirements, established brand decisions, or repository-specific UI conventions merely to clear a generic detector rule.

### 本地图片路径规则

- 读取用户提供的本地图片前，必须先将路径转换为 WSL2 可用的格式。
- 禁止将原始 Windows 路径（如 `C:\Users\name\AppData\Local\Temp\image.png`）直接传给 Linux 图片工具。
- 应转换为对应的 WSL 路径，例如 `/mnt/c/Users/name/AppData/Local/Temp/image.png`；必要时使用 `wslpath -u`。
- 读取前必须确认转换后的路径真实存在。
- Windows 8.3 短路径（如 `FUNDCO~1`）必要时应解析为完整用户目录名称。
- 对 Windows 临时目录中的图片，需要区分：
  - 路径格式不正确；
  - 临时文件已经被删除。
- 转换后的文件存在时，使用其绝对 WSL 路径读取。
- 文件不存在时，应告知用户临时文件可能已经过期，并请用户重新上传或附加图片。
