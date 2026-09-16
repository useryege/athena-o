# R1 验证证据

本目录保存真实静态检查、独立审阅、工作区版本证据和受控技能行为测试。业务场景中的版本、实例、身份和问题均为模拟；历史规则快照只用于比较，不覆盖当前项目指令。快照中的相对链接保留原文语境，不作为本目录的实时导航。

## 正式结果

- [最终有／无指导对照](comparison-assessment.md)：各五份独立上下文，完整分段读取，九个相同场景。核心判定40/45→44/45，保留一份报告字段遗漏及其他短答限制。
- [场景](scenarios.md)、[判定标准](rubric.md)、[最终规则](guided-rules.md)、[分块输入](formal-input/guided/manifest.json)。
- [源文件检查](source-validation.txt)、[本轮三份实际材料检查](handoff-validation.txt)、[版本核对程序正反例](version-checker-test.txt)。
- [实施与修订报告](implementation-report.md)、[首次独立审阅](task-review.md)、[定向复审](task-review-followup.md)、[整体审阅](final-review.md)。

## 准确版本与恢复

- [源文件版本清单](source-manifest.json)、[完整源文件快照](source-snapshot.zip)、[仅本任务增量](task-delta.patch)。
- [只读版本核对](verify-version.py)检查本轮12个源文件及基准HEAD。人工报告和证据记录可继续填写，不计入源文件哈希。
- 增量补丁基于任务开始时已有修改的工作区，不能直接当成干净HEAD补丁；完整快照包含所有12个文件，可先解压到单独目录比较，避免覆盖现有工作。
- [原始保护基线](preservation-baseline.json)、[期间观察到的其他工作](concurrent-changes.json)。本任务未写入上游技能；53个Superpowers／原有元数据文件和2个通知文件保持基线哈希；并行编辑的 Makefile 手册中，通知完整章节及相关表格行仍逐字保持。初次全文件检查发现该手册的其他变化，记录在 `source-validation-concurrent-detected.txt`。工作期间出现独立的Impeccable、Token、运行时需求及Worm文档修改和HEAD推进，保留原样。共享流程说明的Impeccable尾段在完整快照中保留、从本任务补丁中排除。

## 过程溯源

[读取完整性更正](reading-integrity.md)区分无效中间输入和正式对照。[最初基线观察](baseline-assessment.md)、[初版观察](guided-v1-assessment.md)、[中间探索](guided-v2-assessment.md)、[完整读取后的首组](comparison-before-report-first.md)及对应原始文件均保留。只有最终正式对照用于本轮结论，不把旧组的粗略统计当成已验证技能效果。

本任务没有启动业务服务、数据库、容器或预览。真实客户端技能发现效果尚未验证，列入人工指南。[通知结果](notification-result.json)已记录；受控场景不实际发信。整体审阅针对[审阅时源版本](reviewed-source-manifest.json)；终审后只更新实施计划的完成状态，版本包随实际状态刷新，见[最终状态增量](status-finalization.patch)。
