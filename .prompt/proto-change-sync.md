使用 `$proto-change-sync` skill，按既定顺序完成本次 proto 联动更新，不要跳步。

【需求目标】
- 一句话说明要实现的业务能力：
  - <例如：为项目创建接口新增 xxx 字段并支持前端展示与筛选>

【变更背景】
- 为什么必须改 proto：
  - <例如：现有接口字段不足，无法表达 xxx 业务语义>
- 影响范围预期：
  - application 层：<新增/删除/修改哪些 message/field/service>
  - server 层：<网关接口、聚合响应、路由是否变化>
  - 前端：<页面、表单、列表、筛选、详情等变化>

【Proto 变更要求】
1. 先更新 `internal/application/application.proto`
- 具体改动：
  - <新增/删除/重命名字段>
  - <字段类型变化>
  - <message/service 变化>

2. 完成后立刻执行 `make protogen`
- 必须核对：
  - `internal/application/apiclient/application.pb.go`

3. 再更新 `internal/server/application/application.proto`
- 具体改动：
  - <新增/删除/重命名字段>
  - <字段类型变化>
  - <service/http 映射变化>

4. 完成后立刻执行 `make protogen`
- 必须核对：
  - `pkg/apiclient/application/application.pb.go`
  - `pkg/apiclient/application/application.pb.gw.go`

【后端适配要求】
- 更新 `internal/application`：
  - <mapping/validation/business logic 需要怎样调整>
- 更新 `internal/server`：
  - <handler/assembler/integration 需要怎样调整>
- 如有 breaking change，先说明影响面再修复。

【前端适配要求】
- 更新 `ui/src/app`：
  - 请求参数：<新增/删除/重命名字段>
  - 响应解析：<字段映射变化>
  - UI 交互：<表单/表格/详情/筛选变化>

【执行约束】
- 严格遵循 skill 顺序，不可跳步或重排。
- 仅修改与本需求相关代码，不做无关重构。
- 每阶段完成后，先汇报“改了什么 + 受影响文件”。

【最终输出格式】
请给出：
1. 变更摘要（按 proto / generated / backend / frontend 分组）
2. 执行命令清单（含每次 `make protogen` 的触发原因）
3. 验证结果（编译/测试/手工验证）
4. 剩余风险与后续建议
