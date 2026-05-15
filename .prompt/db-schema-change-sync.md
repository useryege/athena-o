使用 $db-schema-change-sync 技能，按严格顺序完成本次“数据库表结构变更联动更新”：

【变更背景】
- 业务目标：<一句话说明这次需求>
- SQL 变更：<project.sql 中新增/删除/修改了哪些表字段、约束、索引>
- 后端行为变化：<接口入参/出参、校验、查询条件、写入逻辑等变化>
- 前端期望变化：<页面字段、表单、列表、筛选、展示逻辑变化>

【必须执行顺序】
1) 先更新 `hack/postgres/init/project.sql`
2) 再更新 `internal/application`（必须早于 `internal/server`）
3) 再更新 `internal/server`
4) 最后更新 `ui/src/app`

【RPC 变更强制规则】
- 在 `internal/application` 侧，如果涉及新增/删除 RPC：
  - 先更新 `internal/application/application.proto`
  - 然后立刻执行 `make protogen`
  - 检查并使用最新 `internal/application/apiclient/application.pb.go`
  - 修复 `internal/application` 下因协议变化导致的冲突并实现需求

- 在 `internal/server` 侧，如果涉及新增/删除 RPC：
  - 先按项目约定更新 `internal/server/application/application.go`
  - 然后立刻执行 `make protogen`
  - 检查并使用最新 `pkg/apiclient/application`
  - 修复 `internal/server/application` 下冲突并实现需求

【执行要求】
- 每完成一个阶段都汇报：修改文件 + 关键改动 + 原因
- 严禁手改生成文件（如 `.pb.go`）
- 仅修改与本需求直接相关代码，不做无关重构
- 若出现 breaking change，先给出影响面（后端/前端）再实施修复

【最终输出】
- 变更摘要（SQL / application / server / frontend 分组）
- 执行命令清单（含每次 `make protogen` 的触发原因）
- 验证结果（编译/测试/关键流程）
- 剩余风险与待确认项
