# Swagger 与 AI 接入展示移除实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 使用 superpowers:subagent-driven-development 按任务实施；独立任务完成后审阅规格符合性和代码质量，最后审阅整个分支。

**目标：** 删除已批准的 Swagger、ReDoc 和 AI 接入展示，完整保留普通 API Key、真实业务接口与隔离保护。

**架构：** 后端和 Vite 在静态命中及 SPA 回退之前拒绝退役逻辑路径；删除文档生成和打包链路；Security 只保留普通密钥状态，Help 只展示配置资源或明确空态。Proto、gateway、实际 JSON 适配和业务授权不变。

**技术栈：** Go 1.27.1、gogo protobuf/grpc-gateway、React/Ant Design、Vite、Node 24.14.1、Jest、Playwright、项目 managed runtime。

**规格：** [已批准设计](../specs/2026-09-18-swagger-ai-discovery-removal-design.md)，批准提交 `b8b8e48f`。

## 全局约束

- 根仓库 `/home/yege/work/athena` 的 `rf4` 为基线；实施 worktree 为 `.worktrees/remove-swagger-ai-discovery`，分支 `codex/remove-swagger-ai-discovery`，已核实检出。
- 不重新确认已批准设计；不推送、发布或合并；最后本地提交实现。
- 既有 `ai-...` 是普通密钥；不删除、轮换或迁移凭据；保留资格、到期、撤销和实时授权规则。
- 保留 Proto、HTTP 映射、实际 JSON 行为、普通 API Key 的一次性展示、复制失败手工选择、Done 清理、防重复提交、卸载与账户切换保护。
- 删除的逻辑路径：`/swagger-ui` 及子路径、`/swagger.json`、`/llms.txt`、`/docs/ai` 及子路径、`/assets/scripts/redoc.standalone.js`、`/assets/scripts/redoc-LICENSE.txt`、`/assets/scripts/README.md`。根与 `/athena` 前缀 GET/HEAD、HTML/机器 Accept 均 404，不重定向或返回 HTML。
- 附加静态目录中的旧文件不能恢复发布；不清理其他工作区、全局缓存或不明归属目录。
- 使用既有单一深色视觉与组件；第三方协议、传递依赖和历史批准/验收材料保留。
- 生成验证必须在无 Swagger 专用可执行文件环境中运行；只隔离本任务 GOPATH 和 dist，防止生成脚本修改主工作区。
- V1–V9 全覆盖，真实 smoke 不能以隔离测试替代；环境收尾保留数据与证据；人工审查材料不代表最终用户确认。

### Task 1: 后端、路由与文档生成退役

**文件：** `internal/server/athena-server.go`、新增 `internal/server/documentation_retirement_test.go`；`ui/vite.config.ts`、新增 `ui/scripts/documentation-retirement.test.mjs`；删除 `util/swagger/`、`assets/swagger.json`、`ui/src/assets/llms.txt`、`ui/src/assets/docs/ai/`、ReDoc 专用资源和 `ui/scripts/vendor-redoc.py`；修改 `assets/embed.go`、`util/assets/assets.go`、`hack/generate-proto.sh`、`hack/tools.go`、`hack/installers/install-codegen-go-tools.sh`、删除 Swagger checksum；`pkg/apis/application/v1alpha1/doc.go`、`go.mod`/`go.sum`、`ui/osv-scanner.toml`；调整两处混合测试。

**接口：** 输入现有 `newStaticAssetsHandler`、`withRootPath` 与 Vite 中间件；输出相同业务路由及一致的退役 404，不新增公共 API。

- [ ] 先补后端表驱动失败测试，枚举所有退役路径、GET/HEAD、Accept、部署前缀；临时静态目录放入旧文档，确认旧实现错误返回内容或 SPA。保留普通页面、`/api`、`/auth` 与真实静态资源对照。

```go
for _, method := range []string{http.MethodGet, http.MethodHead} {
    req := httptest.NewRequest(method, prefix+retiredPath, nil)
    req.Header.Set("Accept", accept)
    response := httptest.NewRecorder()
    handler.ServeHTTP(response, req)
    require.Equal(t, http.StatusNotFound, response.Code)
    require.Empty(t, response.Header().Get("Location"))
}
```

- [ ] 为真实 Vite 开发/预览服务补对应根与前缀请求矩阵，先运行记录失败；测试使用临时端口、明确关闭服务，并包含普通页面和资源对照。
- [ ] 在静态查找前增加退役路径判定与 404；移除 Swagger 注册、加载与代理。匹配目录边界，勿封禁普通名称相似业务页。

```go
if isRetiredDocumentationPath(r.URL.Path) {
    http.NotFound(w, r)
    return
}
```

- [ ] 删除文档资源及生成链路。保留 Go/gogo/gateway、Trader Sync ClientConnInterface 适配和源码整理；核实 openapi-gen 无实际消费者后移除闲置安装、导入、标记。通过 `go mod tidy` 收敛依赖。
- [ ] 删除 Module Access 的 Swagger 枚举检查，保留/验证 `TestModuleAccessSettingsRealJSONContractAndAPIKeyWrite` 的实际数字枚举。Notification recovery 用显式字段集合替代 Swagger 读取，保留两跳与可选字段断言。

```go
properties := map[string]struct{}{
    "state": {}, "reason": {}, "startedAt": {},
    "remainingMillis": {}, "elapsedMillis": {}, "clockSource": {},
}
```

- [ ] 运行 `go test ./internal/server ./internal/server/moduleaccess ./util/assets`、`go test -tags=integration ./internal/notification -run TestRecoveryRuntimeRealGatewayEvidence -count=1`、Vite 路由测试及相关 shellcheck；报告 RED/GREEN。
- [ ] 本地提交本任务并独立审阅；生成运行放在第三任务稳定依赖后统一执行。

### Task 2: Security、Help 与浏览器回归

**文件：** `ui/src/app/member/pages/account-security.tsx`、`ui/src/app/shared/pages/help.tsx`、删除 `ui/src/app/shared/ai-connection.ts` 及专属测试；按实际引用清理 CSS；`ui/e2e/theme-refactor.spec.ts`、相关 fixtures、必要时新增 `ui/e2e/documentation-removal.spec.ts` 或 Jest 用例。

**接口：** 保留 `services.memberSecurity.createToken/listTokens/deleteToken` 调用和页面路由；Help 消费 `AuthSettings['help']`；不变更后端凭据接口。

- [ ] 先调整并运行失败用例：无 Connect AI/Swagger/LLM 入口；Help 支持与下载链接保留；无资源显示 `No help resources configured`；已存 `ai-...` 列表显示且可撤销。

```ts
await expect(page.getByRole('button', {name: 'Connect AI', exact: true})).toHaveCount(0);
await expect(page.getByRole('link', {name: /Swagger UI|LLM discovery|Full-Account AI Access/})).toHaveCount(0);
await expect(page.getByText('No help resources configured', {exact: true})).toBeVisible();
```

- [ ] 删除 AI 导入、purpose 分支、自动名、验证请求、重试、结果面板和独占样式。以布尔创建状态取代 `apiKey|ai` 联合状态，保留全部普通密钥 ref/generation/accountId 保护。
- [ ] API keys 成为 Security 唯一业务面板，`Create API key` 为主要操作；Help 用现有空态组件呈现无资源状态，配置资源按原顺序呈现。
- [ ] 保留并扩充普通创建、一次性结果、剪贴板失败手动选择、Done 清理、重复提交、账户切换、卸载后迟到成功/失败、撤销确认与访问资格回归；移除仅针对 AI 说明的测试。
- [ ] 桌面/手机及会员/管理员 Help 均验证；沿用深色主题、焦点可访问性；更新当前受影响的验收断言而不修改历史截图和批准记录。
- [ ] 执行定向 Jest/Playwright、`cd ui && yarn lint && yarn build`；本地提交并独立审阅。

### Task 3: 生成、整体验证、当前文档与交付

**文件：** 当前 `docs/developer-guide/api-docs.md`、`docs/design/developer-experience/ai-discovery-documentation.md`、`docs/design/README.md`、`docs/requirements/README.md`、账户凭据及当前 UI/覆盖/工具链说明；新增 `docs/testing/swagger-ai-discovery-removal-acceptance.md`、`docs/testing/human-review/swagger-ai-discovery-removal/R1/{ai-delivery,review-guide,human-report}.md`。

**接口：** 消费前两任务实现和测试；产出 V1–V9 证据、准确版本及可恢复的人工复验步骤。

- [ ] 为 worktree 准备自己的 UI 依赖、dist 工具、隔离 GOPATH；复制所需生成器而不复制 swagger/protoc-gen-swagger/openapi-gen，确认 `command -v` 无专用工具。执行 `make protogen`，检查生成差异，无 Swagger 产物；预期业务 `.pb.go`/gateway 不变，若有差异追溯来源。
- [ ] 执行 API 二进制构建、受影响 Go 包/凭据/JSON 测试、Notification 集成、UI lint/build 与定向浏览器回归。检查清理后的 `ui/dist/app` 和嵌入链路无退役资源。
- [ ] 核对运行中环境归属。目标未运行时从本 worktree 选用项目 Node，使用独立 `INSTANCE=swagger-removal` 执行 `make run`，记录会话及日志；检查 member/admin bootstrap、真实页面和旧地址。

```bash
make run INSTANCE=swagger-removal
make runtime-status INSTANCE=swagger-removal
make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:4000
```

实际端口以该实例状态为准；如已有无关实例占用默认端口，使用运行器分配地址并记入证据，不终止借用环境。

- [ ] 在真实环境定向验证 Security/Help 与退役 URL；测试密钥仅在同一测试账户创建/撤销，记录结果；未覆盖的认证流程准确说明。
- [ ] 同步当前文档，退役长期设计保留边界与证据入口；分类检索当前引用、退役规则/测试、历史材料和第三方/传递依赖。
- [ ] 完成整个分支 AI 审阅并修复有效问题，按变更范围重验。
- [ ] `make stop INSTANCE=swagger-removal`，检查进程、端口、容器停止；保留数据卷和报告。记录隔离 harness 自清理及任何保留环境归属。
- [ ] 按 athena-human-review 交付具体 R1 材料，指定不可变实现提交、启动/停止命令、地址、身份、场景和证据；人工报告保持草稿。
- [ ] `git diff --check`、检查最终工作区并本地提交全部实现/材料；核对分支与提交。任务超过 600 秒时在最终回复前从仓库根目录执行固定通知命令一次并等待结果。

## 计划自检

V1 对应任务 1/3；V2 对应任务 1/2/3；V3 对应任务 1/3；V4–V6 对应任务 2/3；V7 对应任务 1/3；V8–V9 对应任务 3。任务 1 只编辑 UI 的 Vite/静态资源，任务 2 编辑 React/CSS/浏览器用例；任务 3 等前两任务稳定后运行生成和整体验收。未引入新的产品决定或额外设计审批。
