# 人工审查指南：Swagger 与 AI 接入展示退役

## 审查对象

- 轮次：R1
- 仓库／worktree：`/home/yege/work/athena/.worktrees/remove-swagger-ai-discovery`
- 分支：`codex/remove-swagger-ai-discovery`
- 实现提交：`b9254a301bade0d326c823e6ffb48a1ad12e28bb`
- 设计依据：[`2026-09-18-swagger-ai-discovery-removal-design.md`](../../../../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)
- 配套报告：[human-report.md](human-report.md)

## 版本核对

在任何会改变本地数据的步骤前：

1. `cd /home/yege/work/athena/.worktrees/remove-swagger-ai-discovery`
2. `git status --short`；`git rev-parse HEAD`；`git branch --show-current`
3. 预期实现提交为完整 SHA `b9254a301bade0d326c823e6ffb48a1ad12e28bb`，分支为 `codex/remove-swagger-ai-discovery`。若版本不符，停止会改变数据的检查并在报告中记录实际版本，不要切换或重置覆盖工作区。

## 环境恢复

- 需要运行环境：检查 CHK-003、CHK-004、CHK-007、CHK-008 时需要；只查看文档和构建证据时无需启动。
- 前提：Node `24.14.1`（`cd ui && nvm use`）、Docker、Go、Playwright Chromium/系统 Chrome；若要重跑数据库集成，还需要用户提供 `ATHENA_TEST_PG_ADMIN_DSN`。
- 启动：在上述 worktree 执行 `cd ui && nvm use && cd .. && INSTANCE=swagger-removal make run`。
- 日志/状态：`.run/instances/swagger-removal/`、`.tmp/swagger-removal/runtime-status.log`；就绪需看到 `full-stack-ready`、member/admin 入口和数据库 probes ready。
- 地址：UI `http://127.0.0.1:4000/`，管理员 `http://127.0.0.1:4000/admin/`，API `http://127.0.0.1:8080`。实际状态文件中的端口优先。
- bootstrap：分别请求 `/api/v1/app/bootstrap`，带 `X-Athena-Application-Realm: member` 和 `admin`；匿名或本地开发登录态均须记录真实返回，不以端口 200 单独判定。

## 身份与测试数据

| 用途 | 身份／权限 | 测试数据及准备方法 | 预期初始状态 |
| --- | --- | --- | --- |
| 页面与 bootstrap | 本地开发 member/admin（由 `ATHENA_SERVER_DISABLE_AUTH=true` 实例提供） | 不执行外部 Google/Phantom 登录；使用当前实例 bootstrap 返回的本地身份 | member 与 admin bootstrap 可读 |
| 普通 API Key | member 普通账户，管理员不可签发 | 使用唯一临时 ID（例如 `human-review-ordinary`），完成创建、列表、撤销；不得复制或记录完整密钥 | 创建前列表无同名 ID |
| 退役 URL | 无需登录 | 根与 `/athena` 前缀，GET/HEAD，`text/html` 与 `application/json` | 不返回重定向、旧内容或 HTML |

## 检查顺序

### CHK-001：生成链路不再依赖 Swagger

- 设计依据：V1；设计第 9 节。
- 前置条件：版本核对通过；仅使用隔离 GOPATH，不改变用户已有 GOPATH。
- 操作步骤：阅读 `.tmp/swagger-removal/protogen-isolated.log`，确认 `make protogen` 退出 0；确认工具目录没有 Swagger 专用可执行文件；阅读 `protogen-generated.diff` 的说明。
- 明确预期：Go/gogo/gateway 仍生成；不会出现 Swagger/ReDoc/LLM 产物；机械描述符差异不被误报为业务契约变更。
- 记录方式：在 human-report 中记录通过/失败、日志路径和实际生成差异。
- 影响与恢复：无数据影响；不在人工审查中重写或清空生成文件。

### CHK-002：构建产物与资源边界

- 设计依据：V2；设计第 5、6 节。
- 前置条件：版本核对通过。
- 操作步骤：阅读 `.tmp/swagger-removal/api-build-final.log`、UI lint/build 结果和 `ui/dist/app` 文件列表；检查历史批准/第三方资料未被删除。
- 明确预期：API 构建、lint、build 成功；新 dist/嵌入链路没有退役资源或运行时引用，业务 Proto/gateway 保留。
- 记录方式：保存命令结果和发现的例外，不把传递依赖中的 `openapi` 资料当作 ATHENA 展示。
- 影响与恢复：无数据影响。

### CHK-003：后端与 Vite 退役 URL

- 设计依据：V3；退役路径清单。
- 前置条件：目标实例 full-stack-ready，或使用已保存的隔离报告。
- 操作步骤：对 API `:8080` 和 UI `:4000` 的根及 `/athena` 前缀逐一请求 `/swagger-ui`、`/swagger.json`、`/llms.txt`、`/docs/ai` 及子路径和三个 ReDoc 资源；每个路径使用 GET/HEAD 与两类 Accept，`curl --location-trusted` 不得跟随重定向。
- 明确预期：全部 404，无 Location、旧内容和 HTML fallback；普通页面、`/api`、`/auth` 与正常静态资源不因相似名称误封禁。
- 记录方式：记录 `.tmp/swagger-removal/retired-url-smoke-final.log` 或人工复验新日志。
- 影响与恢复：无数据影响。

### CHK-004：member/admin bootstrap 与真实 smoke

- 设计依据：V8。
- 前置条件：CHK-003 同一实例，确认 worktree/instance 归属。
- 操作步骤：请求 member/admin bootstrap；执行 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://127.0.0.1:4000`；分别打开 `/` 和 `/admin/`。
- 明确预期：bootstrap realm 正确，smoke 报告为 passed；记录登录页或本地开发身份，不把 shell smoke 扩大解释为业务全回归。
- 记录方式：保存 smoke report、URL、实例和运行时间。
- 影响与恢复：smoke 不提交业务变更。

### CHK-005：普通 API Key 与已有 `ai-` 名称

- 设计依据：V4；设计第 3、4 节。
- 前置条件：member Security 页面可用，使用当前账户且确认没有同名测试 Key。
- 操作步骤：进入 Security，创建普通 Key；验证一次性展示、关闭后不可重建 secret；若已有 `ai-` 前缀条目，确认它按普通列表行显示；执行复制失败后的手工读取和 Done。
- 明确预期：只有 Create API key；无 Connect AI、验证、重试或说明面板；`ai-` 名称不改变权限或撤销语义。
- 记录方式：不得把完整 secret 写入报告；记录 UI 状态和 Key ID 即可。
- 影响与恢复：创建测试 Key 后在 CHK-006 撤销；不轮换或批量处理其他 Key。

### CHK-006：撤销、到期和实时授权保护

- 设计依据：V4、V5；账户凭据设计。
- 前置条件：CHK-005 的临时 Key。
- 操作步骤：在确认框选择 Keep key，确认请求未发送；再次选择 Revoke key；刷新列表；如环境允许，使用短过期时间观察过期后不可用；切换账户或离开页面后观察迟到请求。
- 明确预期：Keep 不删除，Revoke 只删除当前账户 Key；过期/撤销即时影响授权；迟到响应不会显示前一账户 secret 或操作错误账户。
- 记录方式：记录请求方法、Key ID、列表状态和错误/成功提示，不记录 secret。
- 影响与恢复：仅撤销本次创建的临时 Key。

### CHK-007：Help 配置与空态

- 设计依据：V6。
- 前置条件：分别准备有支持/下载链接和无资源的 member/admin fixture 或本地配置。
- 操作步骤：打开两种 realm 的 Help，检查桌面和窄屏；访问配置链接；再检查无资源状态。
- 明确预期：配置资源按当前顺序显示；无资源精确显示 `No help resources configured`；无 Swagger、LLM discovery、Full-Account AI Access 或 Connect an AI 入口；链接可访问性和焦点顺序正常。
- 记录方式：截图或报告路径、viewport、realm 和实际文本。
- 影响与恢复：不修改历史截图/批准材料。

### CHK-008：真实接口与 JSON/权限边界

- 设计依据：V7。
- 前置条件：Go 单元回归已通过；如执行数据库集成需 `ATHENA_TEST_PG_ADMIN_DSN`。
- 操作步骤：运行 `go test -count=1 ./internal/server ./internal/server/account ./internal/server/moduleaccess ./util/assets`；运行 Module Access 数字 JSON 检查；在有 DSN 时再运行 Security retirement 与 Notification recovery 集成测试。
- 明确预期：普通 JSON 字段/数字枚举、HTTP gateway 和 API Key 授权保持；管理员不能签发普通 Key；无 DSN 时必须记录受阻，不得填写通过。
- 记录方式：记录每个命令退出码和精确缺失变量。
- 影响与恢复：集成测试使用其自身随机/隔离数据库，不执行 reset 或删除用户数据。

### CHK-009：收尾与 R1 材料

- 设计依据：V9；人工审查规则。
- 前置条件：完成需要运行环境的检查并保存报告。
- 操作步骤：核对实例属于目标 worktree；执行 `make stop-instance INSTANCE=swagger-removal`；检查进程、端口、容器；读回本目录三份材料。
- 明确预期：服务停止，数据卷/日志/报告保留；`human-report.md` 仍为草稿且每个 CHK 结果为“未执行”，等待用户填写。
- 记录方式：记录停止命令、状态文件和材料读回结果。
- 影响与恢复：不运行 `make run-reset`，不删除卷，不停止借用环境。

## 修复轮复验范围

| 分类 | 本轮检查 | 原因或沿用依据 |
| --- | --- | --- |
| 修复问题 | V3 的 `/athena` 根部署别名；CHK-003 | 真实 smoke 首轮发现 HTML Accept 下回退 200，已补后端/Vite 判定和测试后重验。 |
| 受影响流程 | 退役 URL、Vite fallback、后端静态 handler | 仅影响退役路径判定与部署别名，普通页面/资源回归已在全量 UI 验收覆盖。 |
| 先前受阻／未执行 | V7 数据库集成 | `ATHENA_TEST_PG_ADMIN_DSN` 未提供；用户提供后按原命令重跑。 |
| 沿用历史结果 | 无 | R1 不把历史验收或截图改写为当前通过。 |

## 本轮现场收尾

- 本轮 AI 启动：`INSTANCE=swagger-removal`，worktree 为本目录；实例服务、进程、端口和自有容器已停止，数据库卷和日志保留。
- 准确停止命令：`make stop-instance INSTANCE=swagger-removal`
- 停止后的核对：`.tmp/swagger-removal/runtime-stop-final-pages.log`；`runtime-status` 为 stopped，目标服务端口无监听。
- 人工如重新启动，完成后再次使用同一命令停止；不要保留该实例仅因为人工审查结束。
