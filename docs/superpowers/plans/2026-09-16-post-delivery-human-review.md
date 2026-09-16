# 将交付后人工审查接入 ATHENA 技能体系

> 状态：用户已批准，技能与项目接入已实施；本轮 AI 交付完成，待人工审查，详细状态见 [R1 交付记录](../../testing/human-review/2026-09-16-athena-human-review/R1/ai-delivery.md)。人工确认尚未完成。
>
> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development for the implementation task. The coordinator owns the isolated skill behavior evaluations and delivery evidence.

**目标：** 在第六步 AI 交付与环境收尾之后增加人工审查，接通问题修复、再次交付和最终人工确认。

**架构：** 新增项目级 `athena-human-review` 技能和三份模板；项目规则调用它，既有技能负责调试、TDD、审查与验证。

**技术栈：** Markdown、YAML；Python 技能格式与文档校验；独立代理行为评估。

**Spec：** [已确认设计](../specs/2026-09-16-post-delivery-human-review-design.md)。用户于 2026-09-16 在当前会话确认完整方案并明确请求实施。

## 全局约束

- 阶段顺序：AI 开发与验证 → AI 交付与环境收尾 → 人工审查 → 问题修复、审查和验证 → 再次交付与收尾 → 人工确认最终交付。
- 保留上游 Superpowers 原文；修改技能、模板与项目规则，不涉及业务代码、API 或数据库。
- 保留当前工作区已有通知相关修改，只增量修改本任务内容；当前 rf4 工作区交付，不移动或重置用户的工作。
- 本次不启动业务服务。测试场景为隔离的模拟材料，无实际邮件、PR、进程或网络副作用。
- 独立代理不发送通知、不提交或推送 Git、不另派子代理。主代理负责通知与总体交付。
- 同一任务沿用稳定的问题编号、原始报告和执行／通知记录；AI 交付与最终交付分别记录。

## 验证与交付顺序

- [x] 保存现有文件快照、工作区差异和上游技能哈希。
- [x] 在修改技能之前完成五份独立无新增指导的行为基线，保存实际响应及缺口。
- [x] 实施任务 1；实现者自检新技能和两份修改的技能，形成报告。
- [x] 使用同一场景和相同模型完成至少五份独立有指导评估，人工检查全部输出；缺陷进入实现者修复与复验。
- [x] 独立审阅需求符合性、任务质量及整体差异，处理有效问题。
- [x] 验证元信息、链接、差异、源文件保护及上游技能保持不变。
- [x] 生成本次 R1 交付报告、具体人工指南、预填报告及版本证据；报告用户待填写。
- [x] 按累计执行时间发送一次通用通知，保留结果；交付状态为“AI 交付完成，待人工审查”。

## Task 1: 实施技能、模板和项目接入

**创建：** `.codex/skills/athena-human-review/SKILL.md`、`agents/openai.yaml`、`assets/ai-delivery-template.md`、`assets/review-guide-template.md`、`assets/human-report-template.md`。

**修改：** `AGENTS.md`、`docs/developer-guide/superpowers-development.md`、`docs/developer-guide/running-locally.md`、`.codex/skills/athena-browser-acceptance/SKILL.md`、`.codex/skills/publish-multi-repo-prs/SKILL.md`、已确认 spec。

**输入：** spec；主代理提供的基线缺口；当前工作区真实规则与授权。

**输出：** 可按正常发现机制使用的项目级技能、三份可具体化模板、统一的项目入口与真实状态规则。

- [x] 描述精确触发条件；实现首次交付、已提交报告处理、修复轮交付和最终人工确认四个入口；纯咨询和独立只读审阅不自动触发完整闭环。
- [x] 模板覆盖任务／轮次／准确版本／设计依据、逐步操作与明确预期、身份和测试数据、适用时的环境恢复、检查结果和证据、问题与原始报告、环境收尾、人工结论。
- [x] 草稿保持被审查版本稳定；完整报告可包括受阻和未执行项目。已提交报告授权已确认设计范围内的修复；明确的仅分析限制、新需求决定和既有外部操作授权继续有效。
- [x] 修复后状态为待人工复验，保留问题编号；针对修复、受影响流程和历史受阻项重查，注明沿用历史结果的依据。
- [x] 在根规则接入第六步后的人工阶段；现有流程文档补充技能入口及按任务／轮次存放的产物位置。
- [x] 浏览器技能和运行说明区分本轮 AI 结束、人工环境恢复与最终交付；移除与当前根规则冲突的“不发完成邮件”表述，通知引用根规则。
- [x] PR 技能保留原有发布授权，在开发闭环中分别报告 Git 与人工审查状态；独立 PR 请求维持其原始范围。
- [x] 将 spec 标为设计已确认，说明实现／验证的当前事实，不提前填为最终交付完成。
- [x] 运行 `python3 /mnt/c/Users/FundConnectHK/.codex/skills/.system/skill-creator/scripts/quick_validate.py` 分别校验新技能、浏览器技能和 PR 技能；校验 YAML UI 元信息并检查新增链接。
- [x] 自检工作区 diff；将变更清单、验证命令与结果、未完成项写入实现报告，返回主代理继续独立评估。

## 行为场景与判定

独立输入覆盖首次交付、停止后的环境恢复、草稿与已提交报告、受阻报告、仅分析、缺陷与新增需求、R2 的问题追踪与证据沿用、已合并但未人工通过、失败验收通知、纯咨询／独立 PR／其他项目，以及用户明确最终确认。

真实场景响应保存于本任务忽略目录，交付时归档到本次 R1 证据目录。每个响应都要人工读取并按行为评分，不能仅凭关键词命中断言通过。文档检查和模拟行为结果不证明真实业务服务验收通过。

## 实际执行说明

预编辑基线和全部中间响应均已保留。发现部分中间代理未读完整规则后，使用任务开始时冻结的基线和最终技能分块重跑正式对照；所有正式样本有完整读取审计。结果与残余输出限制见 [正式评估](../../testing/human-review/2026-09-16-athena-human-review/R1/evidence/comparison-assessment.md)，不把模拟行为当成真实业务验收。工作期间出现本任务之外的 Impeccable／Token／运行时文档修改，保留原样并单独记录；本任务不改写上游 Superpowers。
