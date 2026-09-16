# 人工审查报告：Worm Markets 数据退役与 Worm Trading 保留

> AI 预填任务、轮次、准确版本、设计依据和检查项；用户填写结果、观察、问题和结论。AI 不代填用户“通过”。

## 审查身份

- 任务标识：<code>worm-markets-retirement</code>
- 交付轮次：R1
- 仓库／worktree：<code>/home/yege/work/athena/.worktrees/worm-markets-retirement</code>
- 分支：<code>codex/worm-markets-retirement</code>
- 受审产品完整版本：<code>1dfcb795dfdf3da1ecb1b63e4acc9677604d90a5</code>
- R1 材料版本：从本目录的最后一次提交运行时解析，并在交付记录中记录；不改变受审产品版本
- 设计依据：[Worm Trading Market Query 设计](../../../../superpowers/specs/2026-09-16-worm-trading-market-query-design.md)、[Worm Markets 退役需求](../../../../requirements/development-runtime/worm-markets-removal.md)、[验收记录](../../../worm-markets-retirement-acceptance.md)
- AI 交付报告：[ai-delivery.md](ai-delivery.md)
- 人工审查指南：[review-guide.md](review-guide.md)
- 实际环境、身份和数据：原 main default namespace <code>ac252d3201cc3c6d872334f8e6a1bcf5</code>；development member local-user 与 admin local-admin；用户填写实际差异
- 审查时间：【用户填写】
- 报告状态：草稿，待人工审查
- 总体结论：【用户填写：通过／存在问题／受阻】

开始运行项前，按指南核对完整产品 SHA，并确认材料追加提交对运行代码的 diff 为空。版本不符时停止运行项并记录受阻。

## 检查结果

| 检查编号 | 检查内容（AI 预填） | 结果（用户填写） | 实际观察 | 证据 | 问题编号 |
| --- | --- | --- | --- | --- | --- |
| CHK-001 | 核对 worm_markets 的 OID/owner/server/active/retained report、单库 DROP 证据、after absent 及正常重启后九库保留 | 未执行 |  |  |  |
| CHK-002 | 对照两账户迁移前后 grant，确认只删两条 Markets grant、revision 1→2、七模块和其余 flags/levels 保持 | 未执行 |  |  |  |
| CHK-003 | 核对五个 worm-markets.* 来源 report/apply/report、全零计数、共享通知事实保持且没有外发 | 未执行 |  |  |  |
| CHK-004 | 人工阅读隔离写入／恢复／七路由证据，并如实记录旧凭据解密与三条现场历史详情因无数据受阻 | 未执行 |  |  |  |
| CHK-005 | 用 local-user 检查四个会员 Worm Trading 页面、指定真实 catalog 响应、桌面／手机导航且不保存；记录供应商实际数量或受阻 | 未执行 |  |  |  |
| CHK-006 | 检查当前 Combinations、Executions、Wallet Connections 记录为空，并将三条无 ID 的历史详情记为受阻 | 未执行 |  |  |  |
| CHK-007 | 用 local-admin 检查 local-user 恰有七模块、revision 2、levels 正确且无 Worm Markets 行 | 未执行 |  |  |  |
| CHK-008 | 用 GET 核对三个旧 Markets HTTP 路径为 404、UI 无入口、Notifications 为空且拒发替身外发返回 403 | 未执行 |  |  |  |

## 问题记录

### ISSUE-001：【用户填写简短问题描述；无问题时删除本示例小节或写“无”】

- 发现轮次：R1
- 对应检查：【CHK 编号；指南外发现填“自由检查”】
- 操作步骤：
  1. 【用户填写实际动作】
  2. 【用户填写输入或选择】
- 预期结果：【按指南或设计应看到什么】
- 实际结果：【实际看到了什么】
- 发生情况：【必现／偶发／仅遇到一次／不确定】
- 影响：【是否阻塞后续项目及具体范围】
- 原始证据：【截图、日志、URL、业务记录标识；没有时写“无”】

原始观察提交后保留，不用后续诊断或修复结论覆盖。下一轮继续使用同一 ISSUE 编号。

## 未执行或受阻项目

| 检查编号 | 状态 | 原因 | 阻塞问题编号 | 可继续的独立检查 |
| --- | --- | --- | --- | --- |
| 【用户填写】 | 【未执行／受阻】 | 【用户填写】 | 【没有则写“无”】 | 【用户填写】 |

已知数据限制：原 main Trading 业务表为空，旧 Trading 凭据解密和三条需要历史 ID 的详情路由没有现场数据。它们可记录为受阻，同时继续其他证据项和只读运行项。

## 本轮结论与交接

- 报告提交声明：【用户填写“本轮报告已经提交，请根据上述内容处理”，或明确确认准确版本最终通过】
- 环境状态：【用户填写已停止／按要求保留／需要 AI 协助收尾】
- 保留实例详情：【用途、地址、仓库/worktree、实例、会话／进程、日志和停止命令；没有则写“无”】
- 我明确接受的例外及范围：【没有则写“无”】

提交时把“报告状态”从草稿改为已提交。包含受阻和未执行项的完整报告也可以提交；如实记录即可。
