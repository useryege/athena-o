---
name: athena-human-review
description: Use when an ATHENA user needs to manually verify a delivered UI flow, report a UI issue from the browser, or confirm a reviewed UI version as accepted.
---

# ATHENA 人工 UI 验收

## 核心边界

人工验收是用户在真实 ATHENA UI 上完成业务操作并反馈观察的过程。它不是工程测试回填，也不是让用户审阅 Git、API、构建或自动化测试结果。

人工验收阶段禁止运行：

- Go、Node、Playwright 或其他自动测试；
- API 契约、curl、grpcurl 或数据库断言；
- lint、build、codegen、扫描或性能检查；
- 由代理代替用户点击页面并宣称人工通过。

AI 阶段已有的自动验证可以作为交付背景，但不得阻止用户对实际操作过的 UI 流程作出结论。工程验证失败、缺少 DSN 或未执行的自动检查另行记录，不填入用户的 UI 通过结论。

## AI 交付人工验收包

AI 在 `docs/testing/human-review/<任务标识>/R<轮次>/` 生成并绑定同一不可变版本：

- `review-guide.md`：用户可照做的 UI 流程、入口、身份、预期和反馈方式；
- `scripts/start-review.sh`：验证目标 worktree/版本后启动指定人工验收实例，输出 UI 地址；
- `scripts/status-review.sh`：只报告实例归属、运行状态、端口和 UI 入口，不做业务断言；
- `scripts/seed-review.sh`：仅在 UI 无法稳定构造前置状态时提供，必须幂等并列出写入的数据；
- `scripts/stop-review.sh`：停止该脚本启动的实例并保留数据、日志和证据；
- `human-report.md`：由 AI 根据用户反馈维护，用户不需要编辑。

没有必要的前置数据时不生成 seed 脚本。脚本不得执行自动测试、API 测试、构建、reset、无关数据删除或自动判定 UI 是否通过。

用户可以自行运行 start/status/seed/stop 脚本；这些脚本创建的资源归用户，除非用户明确要求 AI 接管或停止。AI 不因脚本存在而自动启动环境。

### 脚本接口契约

- 所有脚本默认无参数，任务、worktree、不可变版本、`INSTANCE`/profile 和日志目录由 AI 在脚本中固定并打印；需要参数时只接受明确的实例名/路径，不接受隐式环境替换。
- `start-review.sh` 先确认当前 checkout 与目标版本一致，再调用项目既有本地运行入口；它可以为启动服务进行必要编译，但不得把编译当作验收结果或单独执行 build 检查。
- `status-review.sh` 只读取运行状态、归属、端口、UI URL 和日志位置；退出失败只表示现场不可用，不表示产品行为失败。
- `seed-review.sh` 只能调用已授权的本地 seed 入口写入最小前置数据，输出写入对象和归属；不得用 curl/grpcurl/API/SQL 断言业务结果，不输出 secret、token 或凭据。
- `stop-review.sh` 只停止自己记录的实例/进程，保留数据库、日志、截图和报告；发现归属不一致时停止并报告，不杀不明进程。

## UI 验收流程

1. AI 先固定版本摘要、worktree、实例名、UI 地址、member/admin 身份和本轮 UI 范围，并把这些信息写入指南；用户不执行 Git 版本核对。
2. 用户运行 `start-review.sh`，按需运行 `status-review.sh` 和 `seed-review.sh`，然后只在浏览器中操作指南列出的页面。
3. 指南按业务流程组织，而不是工程 `CHK-*` 表。每个流程必须写明入口、角色、操作顺序、桌面/手机差异、可观察预期、数据前置和恢复方式。
4. 用户完成一个流程后，用简短消息回复“通过”或描述具体问题；可附截图、录屏、页面地址或时间。除运行验收脚本外，用户不运行 Git、测试、API 或构建命令，也不填写 Markdown。
5. AI 将用户原始反馈、准确版本、页面/身份、证据位置和问题编号写入 `human-report.md`。没有用户结论的流程保持“待用户操作”，不得自行填充为通过。
6. 用户完成范围内流程并明确确认该版本 UI 通过后，本轮人工验收结束。不要再增加独立的“正式记录最终交付状态”或工程报告提交门槛。
7. 用户运行 `stop-review.sh` 完成收尾；若用户要求 AI 收尾，先核对实例归属，再使用同一脚本或对应停止入口并报告结果。

## 数据与问题处理

- UI 优先：普通 API Key、撤销、表单提交、导航和状态切换由用户在页面中完成。
- 最小 seed：只有无法安全或稳定从 UI 构造的账户、配置资源或历史前置状态才注入；seed 必须幂等、可追踪、不可重置无关数据。
- 保留用户原始问题，不用后续诊断覆盖。问题需要修复时，AI 另开修复轮，只复验受影响的 UI 流程；用户不需要掌握 receiving-code-review、TDD 或 Git 状态机。
- 用户反馈“受阻”时记录阻塞原因；它只阻止对应 UI 流程，不自动把其他已操作流程改成未通过。
- 用户明确接受的范围例外要原样记录；不得把未执行的自动检查改写成通过。

## 报告状态

`ai-delivery.md` 记录 AI 已完成的实现和背景验证；`review-guide.md` 是用户操作说明；`human-report.md` 是 AI 维护的 UI 反馈记录。三者不得要求用户手工填写，也不得用自动测试结果替代 UI 观察。

人工验收状态只有以下事实：待用户操作、进行中、存在 UI 问题、受阻、用户确认通过。只有用户对准确版本明确确认后，AI 才能记录“用户确认通过”；是否发布、推送、创建 PR 或集成分支由用户另行授权。
