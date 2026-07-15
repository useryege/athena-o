使用 `$sync-athena-changes` skill，根据真实依赖维护 ATHENA 源文件、生成物和消费者的一致性，不套用固定的全仓库阶段顺序。

【变更背景】
- 业务目标：<一句话说明本次需求>
- 生成源：<是否涉及模块 migration/query SQL、Solidity、共享 API Types 或 Proto>
- 依赖层：<是否涉及模块后端、server 后端、前端或其他直接消费者>

【同步原则】
- 只处理本需求涉及的源文件、生成物和直接消费者，不扩大修改范围。
- 仅沿真实依赖关系保持“源文件 → 生成物 → 消费者”的顺序；互不依赖的层可以按合适顺序处理。
- 每组连贯的源文件修改完成后再执行一次对应生成命令，不要求每次保存后立即生成。
- 模块 migration/query SQL 使用 `make sqlc-local`，Solidity 使用 `make abigen-local`，共享 API Types 和非生成 Proto 使用 `make protogen`。
- 不手动编辑 sqlc、protobuf、gateway、abigen 或其他生成产物；生成差异异常时回查源文件。
- 如果只是单层手写后端、前端、样式或文案修改，且不改变生成契约或其他层，则保持单层处理。

【执行边界】
- 不预设 module、server 和 frontend 的全局固定顺序，只在消费者依赖尚未稳定的接口时延后该消费者。
- 不自动运行测试、`make run` 或 Playwright；只有用户明确要求或其他更高优先级项目规则要求时才执行扩大验证。
- 最终只汇报实际修改的源文件与消费者、执行过的生成命令、相关验证结果和剩余风险，不输出空阶段。
