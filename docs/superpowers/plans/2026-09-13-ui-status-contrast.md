# 共用状态标签对比度修复实施计划

> 执行方式：Superpowers subagent-driven-development；用户已确认本方案及共用组件范围。

**目标：** 修复浅色成功标签的 3.37:1 对比度，使普通文字达到 4.5:1 以上。
**架构：** 共用 StatusTag 添加成功状态类，shared.css 只覆盖浅色文字。保留 Ant Design 背景、边框及深色配色。
**技术栈：** React、Ant Design、CSS、现有 Playwright/axe；Node 24.14.1。
**规格：** 本任务用户已批准的对话方案，以下完整记录执行约束。

## 全局约束

- 成功标签文字 #237804，既有背景 #f6ffed；预期约 5.44:1，以浏览器实际渲染为准。
- negative 优先于 positive；空值回退、组件接口、业务状态、API 和依赖不变。
- 深色、负面、中性、独立 Ant Tag 和 Trader Sync 局部覆盖保持。
- 不关闭 axe 规则、不隐藏文字、不通过增大字号降低验收门槛。
- 在隔离 worktree 实现；保留其他任务改动及现有运行服务、数据。最终仅回写本任务差异到根工作区以实际验收和交付；不提交或推送。

## Task 1: 修复共用成功标签

文件：ui/src/app/components/display.tsx、ui/src/app/styles/shared.css。

- [x] 先运行现有 make ui-a11y，确认原有 4 个失败均为浅色服务状态页的 color-contrast；保存原始结果。
- [x] StatusTag 添加 className={props.positive && !props.negative ? 'athena-status-tag--positive' : undefined}，保留既有 color 表达式。
- [x] shared.css 添加 :root:not([data-theme='dark']) .athena-status-tag--positive.ant-tag { color: #237804; }。不添加 !important，保留已有局部覆盖。
- [x] 执行 UI TypeScript 检查与 make ui-a11y，要求原有 32 个场景全部通过；独立审查差异。

## Task 2: 验收、文档与交付

- [x] 查看两种主题截图与 SERVING、Started: Yes、Poller: Active 的实际样式及对比度；抽查账户、网关、Trader Sync 共用标签，验证负面/中性及局部覆盖。
- [x] 运行 make ui-acceptance 隔离模式（含构建）并检查原始报告和清理结果。
- [x] 将已验证修改安全回写根工作区，核对根目录开发服务归属和健康，执行 make ui-acceptance UI_ACCEPTANCE_MODE=smoke。预检不能替代真实 smoke。
- [x] 更新 docs/testing/ai-dev-tools-readiness.md，保留历史失败、补充此次结果与证据位置；在 docs/design/web-ui/application-shell.md 记录共用样式约束。
- [x] 差异和独立审查通过后，从根目录发送一次 make notify-task-complete，等待结束并报告结果。

超出本范围的新问题记录并通知用户，不扩大修复。测试失败保留证据并排查；验收不完整时不得发送完成通知。
