
# Source: .codex/skills/athena-human-review/SKILL.md

---
name: athena-human-review
description: Use when ATHENA development reaches post-delivery human review, when a user submits an ATHENA human review report, when fixes need another review round, or when the user confirms the reviewed version as final.
---

# ATHENA 交付后人工审查

把 AI 阶段完成与用户最终确认分开记录。首次交付必须同时提供可定位的交付报告、可执行指南和可回填报告，不能只给检查建议。

## 触发边界

- ATHENA 开发任务完成实现、AI 审阅、约定验证和第六步环境收尾后，生成 R1 材料。
- 用户明确提交完整人工报告后，处理报告；报告允许包含受阻和未执行项。
- 修复完成并再次收尾后，生成 R2、R3 等材料。
- 用户对准确轮次和版本明确确认通过后，记录最终交付。

纯咨询、独立只读审查、独立 PR 操作、其他项目，以及未明确接续的历史已完成任务不自动进入此闭环。

## 每轮材料契约

从三个实际模板生成文件，保存到 `docs/testing/human-review/<任务标识>/R<轮次>/`：

1. [AI 交付模板](assets/ai-delivery-template.md)生成 `ai-delivery.md`：交付范围、设计依据、准确版本、AI 证据、已知限制、Git 状态、人工状态和环境收尾。
2. [人工审查指南模板](assets/review-guide-template.md)生成 `review-guide.md`：版本核对、适用时的环境恢复、身份和测试数据、逐项操作、可观察预期、证据及恢复方法。
3. [人工报告模板](assets/human-report-template.md)生成 `human-report.md`：AI 预填元信息和真实检查项，结果保持“未执行”，由用户填写观察、问题和结论。

模板中的填写标记只用于作者识别。实际交付必须替换为本轮确定值；不适用项写明原因。AI 不代填用户“通过”。若成果未提交，记录工作区差异和可恢复补丁或等价产物，不能用旧提交的证据代表当前状态。

正式交付前必须实际写入并读回三份文件，核对路径存在和内容完整，再在回复中给出可打开的文件链接。只声称“已生成”、给出尚未落地的路径或复述模板要求不算交付。逐项检查以下契约：

- 三份材料使用同一任务、轮次、设计依据和完整不可缩写的提交 SHA／产物版本；不能用 `111...111`、“最新版本”等代替。
- 指南在任何启动步骤之前写出完整预期版本及核对方法；版本不符即记录受阻。
- 指南中的每个真实 `CHK-*` 都在 `human-report.md` 有且只有一条对应结果行，检查内容已经具体化，初始结果为“未执行”。
- 报告保留可填写的实际观察、证据、问题编号和总体结论；不能用一段泛化回填说明替代逐项结果行。

若当前通道确实只能交付文本或禁止写文件，在回复中直接展示可复制的 `human-report.md`，而不是描述它已经预填。简短回复同样展示报告。下面是合格的已填写结构节选；示例值仅说明结构，实际交付换成本轮事实：

```markdown
# 人工审查报告：Notifications 重新连接期间保留旧连接

- 交付轮次：R1
- 完整版本：5e1c7f8a3d6b4c2e9f0a1b7c8d3e4f5a6b7c8d9e
- 设计依据：docs/requirements/web-ui/notifications-proposal.md
- 报告状态：草稿
- 总体结论：待人工填写

启动前版本核对：在目标 worktree 执行 `git rev-parse HEAD`，预期完整输出
`5e1c7f8a3d6b4c2e9f0a1b7c8d3e4f5a6b7c8d9e`；一致后再启动环境。

| 检查编号 | 检查内容 | 结果 | 实际观察 | 证据 | 问题编号 |
| --- | --- | --- | --- | --- | --- |
| CHK-001 | 从已连接的 Notifications 页面开始重新连接，在替换完成前返回连接区，确认原 Connected 连接仍显示且可用 | 未执行 |  |  |  |
```

这一行是文本交付的节选，不是检查范围上限；正式 `human-report.md` 仍为指南中的每个 `CHK-*` 逐行预填。文本报告直接写入真实任务、轮次、完整不可缩写版本、设计依据和实际检查内容，并保留空白的观察、证据、问题编号及待用户填写的总体结论；可靠的交付报告引用可以补充定位，不能替代报告中的完整版本。

## 四个入口

### 首次交付

R1 覆盖本次交付的完整人工范围。只有 AI 审阅、约定验证、材料和收尾均有证据时，状态才是“本轮 AI 交付完成，待人工审查”；否则写“AI 阶段未完成／存在阻塞”及缺口。文档或技能任务写明无需运行栈，不为人工阶段启动服务。通知只引用根 `AGENTS.md` 的累计时长、固定内容和同任务去重规则。

### 已提交报告

“已提交”可由文件状态或同等自然语言表达。草稿期间保持被审查版本稳定；可协助恢复环境，但不因草稿问题修改产品。若用户要求立即修复，先保存本轮已有观察并明确切换到下一版本。

收到已提交报告后，先用 `receiving-code-review` 核实，再用 `systematic-debugging` 定位；局部修复方案写入交付材料，多任务或跨层修复使用 `writing-plans`，然后对已确认设计范围内的缺陷连续实施。行为修复使用 `test-driven-development`，实施后使用 `requesting-code-review` 和 `verification-before-completion` 完成审阅与验证，并执行仍适用的真实验收。报告提交即包含这部分修复授权，不新增审批。用户明确的仅分析／先审方案限制继续优先；新需求、设计决定及外部操作仍按原授权边界处理。

保留原始观察，问题编号跨轮次不重置。方案拟定或修复进行中不得提前写验证通过；只有实际修复、代码审阅和约定验证的证据均已取得，问题才能标为“AI 已验证，待人工复验”，且不能标为人工关闭。

### 修复轮交付

新轮次列出：原问题复验、受修复影响的关联流程、历史受阻或未执行项，以及可沿用的上轮结果。沿用项注明来源轮次、版本和影响判断，不能伪装成在新版本重新通过。

### 最终人工确认

仅当用户明确确认准确轮次和版本，每个必查项都有真实结论，除用户明确接受并记录范围的例外、范围调整或设计变更外其余必查项均已完成并通过，问题已有处置，AI 证据与该版本一致，且环境状态已核对，才记录“最终交付完成”。失败、受阻和未执行项保留原状态，不能为了结案改写成通过。PR 已合并、AI 测试通过、通知已发送都不能替代此确认。

人工阶段位于第六步 AI 交付与环境收尾之后；该轮可以按任务既有方式保留工作区或分支、提交 PR 或已经合并。Git 操作授权按用户原话中的实际仓库、分支／PR／任务范围、动作和显式限制复用；面向同一任务修复交付的授权不因产生新提交 SHA 自动失效。明确绑定特定不可变 SHA 或一次操作的授权不得自行扩大，后续仓库、目标分支／PR、任务范围、动作或显式限制发生变化时也须按实际授权边界处理。


# Source: .codex/skills/athena-human-review/agents/openai.yaml

interface:
  display_name: "ATHENA Human Review"
  short_description: "Manage ATHENA post-delivery human review rounds"
  default_prompt: "Use $athena-human-review to prepare or process the current ATHENA post-delivery human review round."


# Source: .codex/skills/athena-human-review/assets/ai-delivery-template.md

# AI 交付报告：【AI 填写任务名称】

> 本模板中的 `【AI 填写】` 必须在实际交付时替换为确定值；不适用时写明原因。

## 交付身份

