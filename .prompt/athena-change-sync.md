使用 $athena-change-sync 技能，按既定顺序完成本次联动更新，不要跳步：

【变更背景】
- 需求目标：<一句话描述这次要实现的业务目标>
- ATHENA.sol 改动点：<新增/删除/修改的函数、事件、结构、字段>
- types.go 改动点：<新增/删除/修改的字段与含义>
- 前端期望：<页面/表单/列表/交互需要怎样变化>

【必须执行的流程】
1) 先更新 pkg/abi/ATHENA/ATHENA.sol，然后立刻执行 make abigen-local，并核对 pkg/abi/ATHENA/ATHENA.go 是否正确同步
2) 再更新 pkg/apis/application/v1alpha1/types.go（字段增删改）
3) types.go 完成后立刻执行 make protogen，并核对以下文件是否正确同步：
   - pkg/apis/application/v1alpha1/generated.pb.go
   - pkg/apis/application/v1alpha1/generated.proto
   - pkg/apis/application/v1alpha1/generated.protomessage.pb.go
4) 处理 internal/application 目录下因协议变更引起的后端冲突与逻辑适配
5) 最后更新 ui/src/app 前端源码，完成字段与交互同步

【执行要求】
- 每完成一个阶段，先汇报“改了什么 + 受影响文件”
- 如遇 breaking change，先说明影响面再实施修复
- 只修改与本需求相关代码，不做无关重构
- 最终给出：
  - 变更摘要（按 合约/协议/后端/前端 分组）
  - 执行过的命令清单
  - 验证结果与剩余风险
