# Token Intelligence 与交易领域边界

当前阶段只实现项目发现、研究和投资标的选择。运行时部署边界如下：

```text
athena-token-scanner
athena-token-validator
athena-token-scheduler
athena-token-collector × 5 data types
athena-token-report-builder
athena-token-selector

athena-token-api
└── 查询与管理 API
```

各 Worker 是独立进程，只通过同一 PostgreSQL 中的持久化任务协作。有效项目进入研究状态，由调度器持续安排 Ave、链状态、相关钱包资产、模拟结果和合约源码采集。采集事实以不可变 observation 保存，报告和项目选择也采用不可变版本。

后续交易领域按以下边界演进：

```text
ProjectSelection
→ TradeOpenWatcher
→ EntryPlan
→ Order
→ Execution
→ Fill
→ Position
→ ExitDecision
```

这些交易实体当前仅作为领域边界，不创建空服务、数据表或 proto。

未来开盘监控以已确认区块为依据：当区块 N 确认目标项目已开放交易后立即提交买入交易，目标进入区块 N+1。系统不监听 mempool，也不在项目选择阶段决策买入时机。
