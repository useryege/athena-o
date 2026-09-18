# AI 交付报告：Swagger 与 AI 接入展示退役

## 交付身份

- 任务标识：`swagger-ai-discovery-removal`
- 交付轮次：R1
- AI 阶段状态：AI 阶段未完成／存在阻塞（V7 两个真实数据库集成测试缺少 `ATHENA_TEST_PG_ADMIN_DSN`）
- 人工审查状态：待人工审查；人工报告仍为草稿
- 仓库及 worktree：`/home/yege/work/athena/.worktrees/remove-swagger-ai-discovery`
- 分支：`codex/remove-swagger-ai-discovery`
- 提交／产物版本：实现提交 `b9254a301bade0d326c823e6ffb48a1ad12e28bb`
- 工作区状态：本报告、验收记录和 R1 材料随本轮本地提交归档；实现提交本身 clean
- Git 交付状态：实现、验收和人工审查材料均在本地提交；不推送、不创建 PR、不合并

## 范围与设计依据

- 本轮交付范围：删除 ATHENA 自有 Swagger/ReDoc/LLM discovery/Connect AI 展示与专用生成链路；保留 Proto、gateway、业务 API、真实 JSON、普通 API Key 生命周期、授权和账户隔离；补齐根与 `/athena` 退役 URL 保护；同步当前文档。
- 明确不在本轮范围：第三方 OpenAPI 资料、历史批准/截图/验收材料、外部身份登录、生产发布、MCP/替代文档站、旧 `ai-` Key 批量处理。
- 已确认依据：[移除详细设计](../../../../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)、[移除需求](../../../../requirements/developer-experience/api-documentation-removal.md)、实施计划 `c9dc21da`。
- 相对上一轮的变化：R1 首次交付；真实 smoke 首轮发现 `/athena` 在 HTML Accept 下回退 200，已在 `b9254a30` 增加后端/Vite alias 判定、Go/Vite 回归并重验。

## AI 审阅与验证证据

| 检查或命令 | 针对版本 | 结果 | 证据位置 |
| --- | --- | --- | --- |
| `go test -count=1 ./internal/server ./internal/server/account ./internal/server/moduleaccess ./util/assets` | b9254a30 + 工作区回归 | 通过 | 会话输出；包含新增 `/athena` alias 测试 |
| `node --test ui/scripts/documentation-retirement.test.mjs` | b9254a30 | 4/4 通过 | 会话输出；dev/preview、root/`/athena` |
| `make protogen`（隔离 GOPATH） | 7a1b1a86 生成边界 | 退出 0；无 Swagger 产物；机械 pb 描述符差异未提交 | `.tmp/swagger-removal/protogen-isolated.log`、`protogen-generated.diff` |
| `make athena-all` | b9254a30 | 通过 | `.tmp/swagger-removal/api-build-final.log` |
| `cd ui && yarn lint && yarn build` | b9254a30 | 均通过 | 会话输出；UI build 输出 `ui/dist/app` |
| `make ui-acceptance` | b9254a30 | root/`/athena` 各 530/530，通过且清理通过 | `.tmp/athena-ui-acceptance/2026-09-18T07-27-49-628Z-3b457b2c/report.md` |
| `make ui-a11y` | b9254a30 | root/`/athena` 共 84/84，通过且清理通过 | `.tmp/athena-ui-acceptance/2026-09-18T07-59-14-171Z-a50ea796/report.md` |
| 定向 Security/Help Playwright | b9254a30 | 每个前缀 5/5，通过 | `.tmp/athena-ui-acceptance/2026-09-18T08-03-32-294Z-2bbd0457/report.md` |
| 真实 `swagger-removal` runtime + smoke/bootstrap/页面 | b9254a30 | full-stack-ready；member/admin bootstrap、应用壳 smoke、member/admin Help、member Security 和真实普通 Key 页面生命周期通过 | `.tmp/swagger-removal/runtime-status.log`、`.tmp/athena-ui-acceptance/2026-09-18T07-48-04-116Z-9f64b1ef/report.md`、`.tmp/swagger-removal/bootstrap-smoke.log`、`.tmp/swagger-removal/real-pages-and-key-smoke.log` |
| 真实退役 URL矩阵 | b9254a30 | API/Vite、root/`/athena` 共 144 组合通过 | `.tmp/swagger-removal/retired-url-smoke-final.log` |
| 真实普通 Key HTTP 生命周期 | b9254a30 | 创建/列表/撤销/列表为空通过 | `.tmp/swagger-removal/api-key-lifecycle-smoke.log` |
| 集成测试（显式 PG admin DSN） | b9254a30 | 未执行到业务：`ATHENA_TEST_PG_ADMIN_DSN is required` | 当前会话失败输出；验收记录 V7 |

- 未执行或未通过的约定验证：需要 `ATHENA_TEST_PG_ADMIN_DSN` 的 Security retirement 与 Notification recovery 集成测试未通过前置检查；它们未被标记为通过。
- 已知限制与阻塞：V7 真实数据库集成需要用户/环境提供显式测试管理员 DSN；外部 Google/Phantom 登录和生产部署不在本轮验收。V9 等待用户人工复验。
- 证据与当前版本差异：隔离生成日志来自实现提交前的同一源输入，代码无 Proto/gateway 生成差异；真实 runtime 构建使用 b9254a30 代码。历史预检报告未作为通过证据。

## 问题与修复状态

| 问题编号 | 原始报告 | 核实结论／原因 | 本轮修改 | AI 验证 | 人工状态 |
| --- | --- | --- | --- | --- | --- |
| ISSUE-001 | 真实 smoke 首轮：`/athena/swagger-ui` 等 HTML Accept 返回 200 HTML | 根部署配置 `/` 时 Vite/API 未把 `/athena` 当作部署别名，先进入 fallback | 后端静态 handler 与 Vite `pathWithinDeployment` 增加 `/athena` alias 判定，并补 Go/Vite 测试 | 144 组合真实请求和新增定向测试通过 | 待人工复验 |
| ISSUE-002 | 无 | 不适用；独立代码审阅未发现 Critical/Important 问题；仅记录测试覆盖的 Minor 建议 | 不适用 | 已完成：无 Critical/Important；Minor 为机器 Accept `*/*` 与普通 HTML fallback 单测覆盖建议，不影响本轮实现结论 | 待人工复验 |

## 环境与资源收尾

### 已停止

- `/home/yege/work/athena/.worktrees/remove-swagger-ai-discovery` 的 `INSTANCE=swagger-removal` full stack；启动命令为 `cd ui && nvm use && cd .. && INSTANCE=swagger-removal make run`，停止命令为 `make stop-instance INSTANCE=swagger-removal`。
- 停止后 `runtime-status` 为 stopped，页面复验后的服务进程、监听端口和自有容器已释放；证据 `.tmp/swagger-removal/runtime-stop-final-pages.log`。

### 保留

- 仅保留该实例的数据库卷、`.run/instances/swagger-removal` 日志和 `.tmp` 隔离/真实验收报告，归属为本 worktree；没有保留运行中的服务。

## 人工审查入口

- [本轮人工审查指南](review-guide.md)
- [本轮预填人工报告](human-report.md)
- 提交方式：用户完成或受阻后把 `human-report.md` 标为“已提交”，或在会话中提交同等完整内容；AI 通过不代表人工通过。
- 下一状态：等待用户针对 `b9254a301bade0d326c823e6ffb48a1ad12e28bb` 人工复验；V7 DSN 阻塞需单独重跑。

## 三份材料完整性核对

| 材料 | 实际路径 | 已写入并读回 | 内容核对 |
| --- | --- | --- | --- |
| `ai-delivery.md` | `docs/testing/human-review/swagger-ai-discovery-removal/R1/ai-delivery.md` | 是 | 任务、轮次、完整实现 SHA、验证限制、收尾和人工状态 |
| `review-guide.md` | `docs/testing/human-review/swagger-ai-discovery-removal/R1/review-guide.md` | 是 | 启动前版本核对、CHK-001–CHK-009、真实身份/数据、停止命令 |
| `human-report.md` | `docs/testing/human-review/swagger-ai-discovery-removal/R1/human-report.md` | 是 | CHK-001–CHK-009 逐行预填为未执行，含阻塞表和用户结论占位 |
