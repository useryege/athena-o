# Swagger 与 AI 接入展示移除设计

> 状态：详细设计已获用户整体审阅通过；业务源码、运行环境均未修改。
> 日期：2026-09-18。
> 已确认范围：移除 Swagger、Connect AI 和公开 AI 接入文档，保留普通 API Key。
> 核对基线：根仓库 `/home/yege/work/athena`，`033c6e81`；实施前复核同任务涉及文件的新增变化。
> 执行安排：用户要求当前会话只提交设计，实施交由新的 chat，必须先切换到新的 Git 分支。

## 1. 目标与方案选择

当前只有用户独立开发，也没有长期对外接口对接的安排。此次删除整套文档与接入展示能力，消除生成、格式修正、打包、发布和入口维护成本。

| 方案 | 结果与代价 | 选择 |
| --- | --- | --- |
| 完整删除 Swagger 与 AI 接入展示，保留业务 API 和普通 API Key | 删除专用工具、资源和 UI 分支，继续通过源码与测试开发 | 已采用 |
| 只隐藏 Swagger 页面，保留 JSON 与 Connect AI | 仍须维护生成修正、机器文档和接入说明 | 不采用，主要维护成本仍在 |
| 删除 Swagger，另建 AI 接口发现方式 | 需要新规范或工具协议及相应维护 | 不采用，当前没有消费者需求 |

本次不新增替代文档站、MCP、OpenAPI 3 迁移、功能开关、重定向或双轨兼容。未来出现真实对接需求时再按当时需求设计。

## 2. 当前依赖与删除边界

核对结果来自实际源码和[当前 AI 文档设计](../../design/developer-experience/ai-discovery-documentation.md)：

| 当前链路 | 实际位置 | 目标 |
| --- | --- | --- |
| Proto 同时生成 Go、HTTP gateway、Swagger | `hack/generate-proto.sh` | 保留 Go 与 gateway，删除 Swagger 输出、合并和修正 |
| 聚合 Swagger 加载进 Go 程序 | `assets/swagger.json`、`assets/embed.go`、`util/assets/assets.go` | 删除 JSON 和 `SwaggerJSON` 加载；保留其他资源及其嵌入 |
| 后端提供文档页面与 JSON | `util/swagger/swagger.go`、`internal/server/athena-server.go` | 删除渲染器、路由注册及专用导入 |
| Vite 代理 Swagger，静态文件提供 AI 文档 | `ui/vite.config.ts`、`ui/src/assets/` | 删除代理、文档、ReDoc 发布资源；保留必要的非页面路径判定 |
| Connect AI 复用普通 API Key 签发 | `ui/src/app/shared/ai-connection.ts`、`ui/src/app/member/pages/account-security.tsx` | 删除接入说明和验证分支，保留普通密钥完整流程 |
| Help 提供文档与 AI 接入入口 | `ui/src/app/shared/pages/help.tsx` | 删除对应入口，保留已配置的支持和下载链接 |
| 测试比对 Swagger 与真实 JSON | `internal/server/moduleaccess/server_test.go`、`internal/notification/recovery_gateway_integration_test.go` | 删除文档依赖，保留真实传输行为断言 |

业务调用继续沿“客户端 → API Server → 内部业务服务”运行。文档不是 API 注册来源；删除其产物不等于删除 API。既有部署网络边界保持有效，不新增内部业务端口的外部入口。

## 3. 账户与密钥行为

### 3.1 保留普通 API Key

保留 Security 路由和既有可见性：只有具备 API Key 资格的普通会员进入密钥管理，管理员不会因此获得密钥签发能力。保留创建、列表、撤销、过期和实时权限校验。

继续沿用当前 Key ID 校验和有效期选项，不修改后端接口、数据表、JWT、权限字段或会员／管理员边界。

Connect AI 没有独立后端连接记录或凭据类型。已有 `ai-...` 名称密钥仍在普通 API Key 列表中显示，并按原规则使用、到期和撤销。名称只是用户可编辑标识，不能作为识别或删除凭据的依据。本次无数据迁移、批量撤销或密钥轮换。

### 3.2 删除 AI 专用浏览器状态

删除自动连接名生成、接入说明组装、Swagger/LLM 地址拼接、AI 凭据自动验证与重试、AI 结果弹窗及其专用状态。

创建流程只保留普通 API Key 分支。保留防重复提交、一次性密钥展示、复制失败后手工选择、显式 Done 清理密钥，以及账户切换／页面卸载后丢弃迟到结果的保护。这些机制同时服务于普通密钥，不能随 AI 分支一起删除。

删除 AI 发起的额外 user-info 验证请求，保留正常登录和账户会话读取。`/api/v1/session/userinfo`、`/api/v1/app/bootstrap` 与 `/api/version` 不因被旧说明引用而删除。

保留普通 Bearer 客户端的能力，也不尝试识别调用方是否为 AI。退役的是产品提供的 AI 接入展示与说明。

## 4. 页面设计

页面沿用[既有单一深色视觉规则](../../requirements/web-ui/visual-theme.md)和现有组件，不另设视觉方向。

### Security

- 保留账户导航和页头；删除 Connect AI 面板，API keys 成为唯一业务面板。
- `Create API key` 成为该页主要操作，列表继续显示 ID、Issued、Expires 与 Revoke。
- 桌面表格、手机分隔行与现有创建／一次性密钥弹窗延续当前布局和交互。
- 保留加载、失败重试、旧快照提示、无密钥和撤销确认状态；无权限仍走既有访问控制。
- 清理仅由 AI 元素使用的样式和图标，不误删密钥弹窗共用布局。

### 会员与管理员 Help

- 删除 Swagger UI、LLM discovery、Full-Account AI Access、Connect an AI 入口及配套说明。
- 保留 Help 页面、导航、配置的支持链接与下载链接；不借此删除整个 Help 功能。
- 未配置任何资源时，使用现有空态组件显示 `No help resources configured`，不留空白资源框、不补造文档或下载入口。
- 桌面与手机使用相同内容顺序，沿用现有主题、字号、焦点与链接可访问性。

## 5. 文档地址与资源退役

### 5.1 对外可见结果

以下逻辑路径在新版本中返回 HTTP 404，不重定向，不回退成会员或管理员 HTML：

- `/swagger-ui` 及其子路径；
- `/swagger.json`；
- `/llms.txt`；
- `/docs/ai` 及其子路径；
- 原 ReDoc 的 `/assets/scripts/redoc.standalone.js`、`/assets/scripts/redoc-LICENSE.txt` 及该目录中随 ReDoc 发布的 `README.md`。

GET、HEAD、浏览器 HTML Accept 与机器请求都覆盖；配置部署前缀时，在剥离前缀后的同一逻辑路径执行相同规则。根路径与 `/athena` 隔离验收各验证一次。

后端与 Vite 必须在静态文件查找和 SPA 回退之前处理这些已退役资源。旧 Swagger 代理删除；必要的路径排除或 404 判定属于资源不可用行为，不保留旧渲染器、生成器或响应内容。普通页面刷新、`/api`、`/auth` 和仍使用的静态资源继续正常工作。

后端支持附加静态目录，旧文件可能仍留在部署目录。新版本不得因该目录存在旧文档而重新发布它们；退役路径判定先于文件命中。实施验收加入附加目录含旧文档的受控场景，不自动清理归属不明的目录。

### 5.2 删除的资源

- `assets/swagger.json`。
- `util/swagger/` 中仅服务于 Swagger 的代码。
- `ui/src/assets/llms.txt` 与 `ui/src/assets/docs/ai/`。
- `ui/src/assets/assets/scripts/` 当前的 ReDoc JS、许可证和专用说明。
- `ui/scripts/vendor-redoc.py`。
- `ui/osv-scanner.toml` 中仅适用于旧 Swagger/ReDoc 的例外；若文件已无其他配置则删除文件。

UI 新构建使用现有清理输出流程，再检查 `ui/dist/app`，防止已删除的源文件残留在构建产物并被 Go 嵌入。后端也需重新构建。文档提交本身不使旧二进制或已部署实例退役；远端发布不在本次设计授权内。

## 6. 生成与依赖收敛

在 `hack/generate-proto.sh` 中删除 `swagger version`、`--swagger_out`、`ATHENA_SWAGGER_VERSION`、聚合元信息、字段与枚举修正、临时 Swagger 文件收集和清理。`jq` 在该脚本内只用于 Swagger 的部分同步移除，不删除其他脚本仍使用的 jq 工具。

保留 Proto 搜索与生成、HTTP annotations、`protoc-gen-grpc-gateway`、Go/gogo 生成、Trader Sync `ClientConnInterface` 适配和既有源码整理。不得因为它们和 Swagger 在同一个脚本中就整段删除。

清理 `hack/tools.go`、工具安装器和旧 checksum 中的 Swagger 专用项目。当前检索仅发现 `openapi-gen` 的工具导入、安装与 `pkg/apis/application/v1alpha1/doc.go` 标记，未发现实际生成消费者；实施时复核后删除这组闲置入口。保留共享 Kubernetes/gRPC 生成依赖，不按包名批量删除。

通过 Go 模块整理决定 `go-openapi/runtime` 等直接／间接依赖是否移除；某个模块仍被 Kubernetes 或其他现有工具传递依赖时允许保留。完成标准是应用不再使用 Swagger 文档链路，不是 `go.sum` 中不存在 openapi 字符串。

只处理本任务工作区拥有的残留工具和产物，不删除全局工具、共享 Go 缓存或其他工作区的 `dist`。

## 7. 真实接口契约与测试调整

Swagger 的命名修正描述了实际响应，但不是运行时 JSON 转换的实现。移除文档修正不能顺手移除 `pkg/apiclient/account/access_json.go`、`pkg/apiclient/appbootstrap/status_json.go` 或其他真实 JSON 适配。

- Module Access：删除对 `assets/swagger.json` 的枚举检查，继续验证真实响应中的数字枚举、HTTP 行为与访问控制。
- Notification recovery：保留现有集成测试的两次 gRPC 跳转、网关响应、恢复状态与可选字段断言；字段集合改为测试明确声明的预期 HTTP 契约，不能从实际响应反向生成预期集合。
- UI：删除仅验证 AI 连接说明的用例，调整 Security／Help 用例；普通密钥创建、一次性展示、复制失败、账户切换和撤销回归继续保留。
- 生成：确认 `make protogen` 在没有 Swagger 专用可执行文件的验证环境中成功，Go 与 gateway 仍生成，并且无 Swagger 产物重现。

不以文本检索替代真实 HTTP、鉴权或浏览器行为检查，也不为了消除文档依赖而删掉整个混合测试。

## 8. 文档同步与检索边界

实施时同步以下当前说明，使开发入口和实际行为一致：

- [API 开发说明](../../developer-guide/api-docs.md)：保留仍适用的接口调试与授权说明，移除文档站和 AI discovery 操作说明。
- [AI 文档长期设计](../../design/developer-experience/ai-discovery-documentation.md)：改为简洁的已退役说明、剩余边界与本次证据入口；历史完整实现由 Git 和历史记录保留。
- [设计索引](../../design/README.md)及相关当前 UI／工具链说明：区分需求与设计已批准和代码实际实施状态。
- 账户凭据长期设计只按实际受影响边界同步，不改动仍适用的密钥与授权规则。

v12／v22 等视觉批准材料、旧规格、旧验收报告和第三方 API 资料保留。当前覆盖入口注明本次退役替代原 AI 展示要求，不改写历史截图和批准事实。`util/pred/openapi.json` 等外部协议资料不属于 ATHENA 文档产物。

检索结果逐项归类为：当前产品引用（应清理）、明确的退役路由／测试（可保留）、历史证据（保留）、第三方协议或传递依赖（按实际用途保留）。不要求全仓字符串清零。

## 9. 实施后的验收标准

下表是后续实施的验收要求，本轮设计尚未执行这些测试。

| 编号 | 验收项 | 通过依据 |
| --- | --- | --- |
| V1 | 生成链路脱离 Swagger | 无专用工具环境下 `make protogen` 成功，保留产物差异受控，无 Swagger 重建 |
| V2 | 构建与资源 | API 构建、UI lint/build 通过；新 UI 输出和 Go 文档嵌入链路无退役产物 |
| V3 | 文档路由退役 | API 服务与 Vite 的根路径／前缀路径 GET、HEAD、HTML／机器 Accept 均为 404；附加静态目录不能恢复旧内容 |
| V4 | 普通 API Key | 创建、一次性展示、复制失败处理、到期／撤销和当前授权检查仍通过；已有 `ai-...` 密钥按普通密钥处理 |
| V5 | 账户隔离 | 切换账户、卸载页面和迟到签发响应不会显示前一账户密钥或操作错误账户 |
| V6 | 页面行为 | 桌面／手机 Security 与会员／管理员 Help 无退役入口；Help 有配置与无配置均明确可用 |
| V7 | 真实接口 | Module Access 数字枚举、Notification recovery 字段／可选性与现有关键 API 鉴权回归通过 |
| V8 | 真实本地验收 | 准备或复用归属清晰的目标环境，检查 member/admin bootstrap，运行真实 smoke，并核验退役 URL 与受影响页面 |
| V9 | 收尾与交付 | 本任务启动的临时环境按归属停止，保留数据和报告；后续实施完成时提供人工审查材料 |

验证按依赖范围执行：生成后复核差异；对受影响 Go 包与脚本运行检查；完成 UI lint/build、相关隔离 Playwright 用例及真实本地 smoke。真实浏览器验收可使用 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke`，它不能替代 V3/V4 的具体断言；按缺口增加定向请求和受影响流程核验。

真实验收环境未启动时按本地运行说明准备，不把首次连接失败当作验证完成。无需为本任务触发交易、密钥轮换或生产清理；本地验收产生的测试密钥应记录并在同一账户下撤销。

仅文档设计阶段使用文件差异、引用链接和范围一致性检查，不启动业务服务、不运行产品验收，也不建立实施后的人工审查轮次。

## 10. 服务规范与审阅状态

本次主要适用 [SDS-R7 与 SDS-R8](../../developer-guide/service-development-standards.md)：记录被删除的展示消费者、保留契约、验证证据及环境归属。实施期间若启动验收环境，同时遵循 SDS-R5 的资源所有权规则。没有新增或拆分服务、改变业务 RPC／存储／事务，因此不引入独立服务迁移。

用户于 2026-09-18 对本文整体审阅通过，包括已有密钥继续有效、旧地址 404、Help 空态、生成依赖清理和验收矩阵。同日明确要求本会话停止实施准备、只提交已有设计，交由新的 chat 执行，且实施前必须切换到新的 Git 分支。当前尚未编写实施计划，未创建实施分支或 worktree，未修改业务代码、启动验收服务或执行远端发布。

新会话读取本规格后，复用此次设计批准，先从包含本设计的版本创建并切换到新的 `codex/` 分支，再整理实施计划并执行。不得直接在原 `rf4` 分支开展代码移除，也不重复请求用户批准已经确定的设计。
