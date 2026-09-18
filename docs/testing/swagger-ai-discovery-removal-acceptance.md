# Swagger 与 AI 接入展示退役验收

> 本记录对应实施提交 `b9254a30`（完整 SHA：`b9254a301bade0d326c823e6ffb48a1ad12e28bb`）。它区分隔离测试、真实本地实例和待人工复验；隔离测试或端口可达不替代真实证据。

## 范围与版本

- 仓库：`/home/yege/work/athena`
- 实施 worktree：`/home/yege/work/athena/.worktrees/remove-swagger-ai-discovery`
- 分支：`codex/remove-swagger-ai-discovery`
- 已批准设计：[`b8b8e48f`](../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)
- 实施提交：`b9254a301bade0d326c823e6ffb48a1ad12e28bb`
- 真实实例：`INSTANCE=swagger-removal`，UI `http://127.0.0.1:4000/`，管理员入口 `http://127.0.0.1:4000/admin/`；实例已停止，数据库卷保留。

## 验收矩阵

| 编号 | 结果 | 证据与边界 |
| --- | --- | --- |
| V1 生成链路脱离 Swagger | 通过（生成差异受控） | 在隔离 GOPATH 中运行 `make protogen`，退出码 0；日志见 `.tmp/swagger-removal/protogen-isolated.log`。隔离环境未提供 Swagger 专用可执行文件，未重新出现 Swagger/ReDoc/LLM 产物。当前 protoc/gogo 压缩描述符版本使若干既有 `.pb.go` 描述符字节产生机械差异，差异见 `.tmp/swagger-removal/protogen-generated.diff`，未改变 Proto/gateway 源输入；生成差异已恢复干净，未提交这些无关重写。 |
| V2 构建与资源 | 通过 | `make athena-all` 通过，日志见 `.tmp/swagger-removal/api-build-final.log`；`yarn lint` 与 `yarn build` 通过。构建输出 `ui/dist/app` 未包含退役资源或运行时引用；全量隔离验收的 build 阶段也通过。 |
| V3 文档路由退役 | 通过 | 后端定向 Go 测试与 Vite dev/preview 测试通过；真实实例对 API Server 和 Vite 共 144 个组合（根与 `/athena`、9 类路径、GET/HEAD、HTML/机器 Accept）均为 404，无 Location、旧内容或 HTML fallback，证据见 `.tmp/swagger-removal/retired-url-smoke-final.log`。附加静态目录旧文件由测试覆盖。 |
| V4 普通 API Key | 通过（真实本地生命周期） | 真实 `swagger-removal` 实例的 member Security 页面完成普通 Key 创建、一次性展示、Done 后列表可见和撤销；不输出 secret，证据见 `.tmp/swagger-removal/real-pages-and-key-smoke.log`。HTTP 级 `smoke-ordinary` 创建、列表、撤销并确认列表为空见 `.tmp/swagger-removal/api-key-lifecycle-smoke.log`。隔离浏览器还覆盖复制失败后的手工读取和已有 `ai-` 前缀名称；过期和实时授权规则由保留的服务实现及回归测试覆盖。 |
| V5 账户隔离与迟到响应 | 通过（真实本地身份 + 隔离迟到响应） | 真实实例分别以 `local-user`/`local-admin` bootstrap 访问 member/admin，角色标识分别为 `administrator=false/true`，证据见 `.tmp/swagger-removal/real-pages-and-key-smoke.log` 与 `.tmp/swagger-removal/bootstrap-smoke.log`。卸载页面后的迟到创建响应、重复提交和撤销账户保护由定向隔离浏览器报告覆盖；未执行外部 Google/Phantom 登录。 |
| V6 页面行为与可访问性 | 通过（真实页面 + 隔离覆盖） | 真实实例 member Security、member Help、admin Help 均返回 200；两个 Help 页面显示精确空态 `No help resources configured`，Security 保留 `Create API key`，证据见 `.tmp/swagger-removal/real-pages-and-key-smoke.log`。桌面/手机及配置资源状态由定向报告覆盖；完整根与 `/athena` 隔离浏览器验收 530/530 通过，报告见 `.tmp/athena-ui-acceptance/2026-09-18T07-27-49-628Z-3b457b2c/report.md`；a11y 两个前缀 84/84 通过，报告见 `.tmp/athena-ui-acceptance/2026-09-18T07-59-14-171Z-a50ea796/report.md`。 |
| V7 真实接口回归 | 部分完成，集成项受阻 | Module Access、服务器退役路由、真实 JSON 相关 Go 回归通过；`go test -tags=integration ./internal/server/account -run TestDocumentationRetirementPreservesAPIKeyLifecycle -count=1` 和 Notification recovery 集成测试均在业务执行前因缺少 `ATHENA_TEST_PG_ADMIN_DSN` 停止，未伪造为通过。失败输出保留在会话记录中；待提供显式测试管理员 DSN 后重跑。 |
| V8 真实本地验收 | 通过（范围已注明） | 独立 `swagger-removal` 实例由本 worktree 启动，runtime-status 显示 full-stack-ready、11 个服务和数据库就绪；member/admin bootstrap 均 200 且返回对应本地身份，真实 smoke 通过，报告见 `.tmp/athena-ui-acceptance/2026-09-18T07-48-04-116Z-9f64b1ef/report.md`。退役 URL 和普通 Key HTTP 流程另有真实请求证据；smoke 本身不代替业务页面交互。 |
| V9 收尾与交付 | 待人工复验 | `make stop-instance INSTANCE=swagger-removal` 成功；页面复验后的实例服务端口、进程和容器均已释放，停止证据见 `.tmp/swagger-removal/runtime-stop-final-pages.log`，数据库卷与测试报告保留。R1 人工审查三份材料指定同一不可变提交并保持报告草稿；用户复验后才能关闭本项。 |

## 代码和资源范围

已删除 ATHENA 自有 Swagger 页面/JSON、ReDoc 与 `llms.txt`/`docs/ai` 资源、专用生成/发布链路、Connect AI 页面与共享模块；`ai-` 前缀既有密钥未被批量删除或特殊处理。Proto、HTTP gateway、业务 API、数字 JSON 契约、API Server 网络边界、普通 Key 授权和账户边界保留。第三方协议资料、历史批准材料和历史验收材料没有做字符串清零。

## 尚未完成项

1. 两个依赖显式 PostgreSQL 管理员 DSN 的集成测试需要在具备该凭据的环境中重跑。
2. [R1 人工审查报告](human-review/swagger-ai-discovery-removal/R1/human-report.md)目前是“未执行”草稿，不能由本记录代替用户确认最终交付。
