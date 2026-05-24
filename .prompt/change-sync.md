使用 `$change-sync` skill，并将本提示词作为本次代码变更的统一联动规则。按需执行，不涉及的阶段可以跳过；但所有已涉及阶段必须严格遵守顺序，不得跳步、倒序或提前处理 `internal/server/**` 后端和前端。

【变更背景】
- 业务目标：<一句话说明本次需求>
- 涉及范围：
  - SQL：<是否涉及 hack/postgres/init/*.sql，例如 application.sql、worm.sql 或未来模块 SQL>
  - 合约：<是否涉及 pkg/abi/**/*.sol>
  - API types：<是否涉及 pkg/apis/application/v1alpha1/*_types.go，例如 application_types.go、worm_types.go 或未来模块 types>
  - 模块 proto：<是否涉及 internal/<module>/**/*.proto，例如 internal/application、internal/worm 或未来模块；不包含 internal/server>
  - 模块后端：<是否涉及 internal/<module>/ 下非 server 模块后端代码>
  - server proto：<是否涉及 internal/server/**/*.proto>
  - server 后端：<是否涉及 internal/server/** 下后端代码>
  - 前端 UI：<ui/src/app 下页面/表单/列表/详情/筛选等变化>

【总体变更顺序】
1. 先处理底层源文件：`hack/postgres/init/*.sql` 和/或 `pkg/abi/**/*.sol`
2. 再处理 `pkg/apis/application/v1alpha1/*_types.go`
3. 再处理非 server 模块 proto：`internal/<module>/**/*.proto`，然后立刻生成
4. 再处理非 server 模块后端代码：`internal/<module>/`
5. 再处理 server proto：`internal/server/**/*.proto`，然后立刻生成
6. 再处理 server 后端代码：`internal/server/**`
7. 最后处理 `ui/src/app` 前端代码

【顺序约束】
- `hack/postgres/init/*.sql` 与 `pkg/abi/**/*.sol` 可以根据需求先后互换，但所有涉及的 SQL/合约变更都必须早于 API types、proto、后端代码和前端代码。
- 不涉及的阶段可以跳过，但剩余阶段的相对顺序必须保持不变。
- 非 server 模块先于 `internal/server/**`；更新后端代码时，`internal/server/**` 永远放在最后执行。
- 每个涉及 proto 的阶段都必须先修改 proto 文件，执行生成命令并核对生成结果后，才能处理依赖该 proto 的普通后端代码。
- `ui/src/app` 的前端变更必须最后执行。

【强制生成规则】
- 修改 `pkg/abi/**/*.sol` 后，必须立刻执行 `make abigen-local`，并核对 abigen 产物是否正确同步，例如 `pkg/abi/ATHENA/ATHENA.go`。
- 修改 `pkg/apis/application/v1alpha1/*_types.go` 后，必须立刻执行 `make protogen`，并核对相关生成文件是否正确同步，例如 `generated.pb.go`、`generated.proto`、`generated.protomessage.pb.go`。
- 修改 `internal/<module>/**/*.proto` 后，必须立刻执行 `make protogen`，并核对对应模块生成代码是否正确同步，例如 `internal/<module>/apiclient/*.pb.go`。
- 修改 `internal/server/**/*.proto` 后，必须立刻执行 `make protogen`，并核对 server/public client 与 gateway 生成代码是否正确同步，例如 `pkg/apiclient/<module>/*.pb.go`、`pkg/apiclient/<module>/*.pb.gw.go`。
- 严禁手改 `.pb.go`、`.gw.go`、abigen 产物、API generated 产物等生成文件。

【后端变更顺序】
1. 非 server 模块：
   - 先修改相关 `internal/<module>/**/*.proto`
   - 立刻执行 `make protogen`
   - 核对生成代码
   - 再修改对应 `internal/<module>/` 后端代码
2. server 模块：
   - 只在非 server 模块 proto、生成代码和后端适配完成后开始
   - 先修改 `internal/server/**/*.proto`
   - 立刻执行 `make protogen`
   - 核对生成代码
   - 再修改 `internal/server/**` 后端代码

【前端约束】
- `ui/src/app` 的前端变更必须最后执行。
- 前端只适配已经完成的 SQL、合约、API types、proto、后端 API 和生成代码，不提前定义未落地接口。
- 前端变更应限制在本需求直接相关的页面、表单、列表、详情、筛选、请求参数和响应解析逻辑。

【执行要求】
- 只修改与本需求直接相关的代码，不做无关重构。
- 每完成一个阶段，先汇报：修改文件、关键改动、触发命令、触发原因、验证结果。
- 如果出现 breaking change，先说明影响面，再实施修复。
- 如某阶段按需跳过，应在阶段汇报或最终输出中说明跳过原因。

【最终输出】
- 变更摘要：按 SQL / Contract / API Types / Module Proto / Module Backend / Server Proto/Backend / Frontend 分组。
- 执行命令清单：包含每次 `make abigen-local`、`make protogen` 的触发原因。
- 验证结果：编译、测试、关键流程或手工验证。
- 剩余风险与待确认项。
