# Task 1 报告：后端、路由与文档生成退役

## 状态

DONE_WITH_CONCERNS

本任务已在 `codex/remove-swagger-ai-discovery` 提交 `a5a558ff`。工作区提交后干净；未修改根 checkout。

## RED / GREEN 证据

- RED：新增 `internal/server/documentation_retirement_test.go` 和 `ui/scripts/documentation-retirement.test.mjs` 后，测试用于锁定旧静态文件、SPA history fallback、GET/HEAD、Accept 头及部署前缀矩阵。控制器提供的初始基线记录了相关实现仍会返回旧内容或 SPA；在本次续作中重新执行后均进入 GREEN。
- GREEN：`go test ./internal/server ./internal/server/moduleaccess ./util/assets` 通过。
- GREEN：`node --test ui/scripts/documentation-retirement.test.mjs` 通过（dev/preview × root/`/athena`，共 4 项）。
- 预期未完成：`go test -tags=integration ./internal/notification -run TestRecoveryRuntimeRealGatewayEvidence -count=1` 因环境未设置 `ATHENA_TEST_PG_ADMIN_DSN`，在测试初始化处失败；不是代码断言失败。
- GREEN：`bash -n hack/generate-proto.sh hack/installers/install-codegen-go-tools.sh` 通过；`go mod tidy` 已执行。

## 变更文件与范围

- 后端静态处理在静态查找/history fallback 前拒绝 Swagger、AI 文档、llms.txt 和 ReDoc 资源；部署前缀、GET/HEAD、Accept 均返回一致 404 且不重定向。
- 删除 Swagger handler 注册、嵌入加载、Swagger 生成/安装/校验和 ReDoc/AI discovery 静态资源；保留 protobuf、gogo、gateway 生成及 Trader Sync `ClientConnInterface` 整理。
- Notification recovery 集成测试改用显式 JSON 字段集合；删除 Module Access 的 Swagger 枚举读取测试。
- 新增后端矩阵测试、Vite dev/preview 实际服务矩阵测试及 API key 生命周期集成覆盖。
- `go.mod`/`go.sum` 已由 `go mod tidy` 收敛；Swagger 专用依赖与 checksum 删除。

## 关注事项

`ui/src/app/shared/pages/help.tsx`、`ui/src/app/shared/ai-connection.ts` 以及对应 E2E 用例仍包含指向已退役 `/llms.txt`、`/docs/ai` 和 `/swagger-ui` 的文字/链接。这些页面级调整未在 Task 1 中扩大处理，Task 2 或后续 UI 批次应删除/改写这些入口并更新断言，否则用户仍会看到指向 404 的旧 discovery 入口。

另外，完整集成 recovery 验证仍需要提供 `ATHENA_TEST_PG_ADMIN_DSN` 后重跑。
