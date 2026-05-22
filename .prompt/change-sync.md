使用 `$change-sync` skill，并将本提示词作为本次代码变更的统一联动规则。按需执行，不涉及的阶段可以跳过；但所有已涉及阶段必须严格遵守顺序，不得跳步、倒序或提前处理前端。

【变更背景】
- 业务目标：<一句话说明本次需求>
- 涉及范围：
  - SQL：<是否涉及 hack/postgres/init/project.sql>
  - 合约：<是否涉及 pkg/abi/ATHENA/ATHENA.sol>
  - API types：<是否涉及 pkg/apis/application/v1alpha1/types.go>
  - application proto：<是否涉及 internal/application 下 proto 文件>
  - server proto：<是否涉及 internal/server 下 proto 文件>
  - 后端代码：<internal/application 与 internal/server 需要怎样适配>
  - 前端 UI：<ui/src/app 下页面/表单/列表/详情/筛选等变化>

【总体变更顺序】
1. 先处理 `hack/postgres/init/project.sql` 或 `pkg/abi/ATHENA/ATHENA.sol`
2. 再处理另一个底层源文件：`project.sql` 或 `ATHENA.sol`
3. 再处理 `pkg/apis/application/v1alpha1/types.go`
4. 再处理 `internal/application` 下 proto 文件，然后立刻生成
5. 再处理 `internal/server` 下 proto 文件，然后立刻生成
6. 再处理 `internal/application` 后端代码
7. 再处理 `internal/server` 后端代码
8. 最后处理 `ui/src/app` 前端代码

【顺序约束】
- `project.sql` 与 `ATHENA.sol` 可以根据需求先后互换，但二者都必须早于 `types.go`、proto、后端代码和前端代码。
- 不涉及的阶段可以跳过，但剩余阶段的相对顺序必须保持不变。
- 后端必须先处理 `internal/application`，再处理 `internal/server`。
- 每个后端目录内必须先处理 proto 文件，执行生成命令并核对生成结果后，才能处理普通后端代码。
- `ui/src/app` 的前端变更必须最后执行。

【强制生成规则】
- 修改 `pkg/abi/ATHENA/ATHENA.sol` 后，必须立刻执行 `make abigen-local`，并核对 `pkg/abi/ATHENA/ATHENA.go` 是否正确同步。
- 修改 `pkg/apis/application/v1alpha1/types.go` 后，必须立刻执行 `make protogen`，并核对相关生成文件是否正确同步。
- 修改 `internal/application` 或 `internal/server` 下任意 proto 文件后，必须立刻执行 `make protogen`，并核对对应生成代码是否正确同步。
- 严禁手改 `.pb.go`、`.gw.go`、abigen 产物等生成文件。

【后端变更顺序】
1. `internal/application`：
   - 先修改 proto 文件
   - 立刻执行 `make protogen`
   - 核对生成代码
   - 再修改 application 层后端代码
2. `internal/server`：
   - 先修改 proto 文件
   - 立刻执行 `make protogen`
   - 核对生成代码
   - 再修改 server 层后端代码

【前端约束】
- `ui/src/app` 的前端变更必须最后执行。
- 前端只适配已经完成的 SQL、合约、types、proto、后端 API 和生成代码，不提前定义未落地接口。
- 前端变更应限制在本需求直接相关的页面、表单、列表、详情、筛选、请求参数和响应解析逻辑。

【执行要求】
- 只修改与本需求直接相关的代码，不做无关重构。
- 每完成一个阶段，先汇报：修改文件、关键改动、触发命令、触发原因、验证结果。
- 如果出现 breaking change，先说明影响面，再实施修复。
- 如某阶段按需跳过，应在阶段汇报或最终输出中说明跳过原因。

【最终输出】
- 变更摘要：按 SQL / Contract / Types / Proto / Backend / Frontend 分组。
- 执行命令清单：包含每次 `make abigen-local`、`make protogen` 的触发原因。
- 验证结果：编译、测试、关键流程或手工验证。
- 剩余风险与待确认项。
