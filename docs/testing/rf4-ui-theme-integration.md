# rf4 UI 主题分支整合记录

日期：2026-09-15（Asia/Shanghai）。本记录对应将已完成并验收的 `codex/ui-theme-refactor` 正式整合回本地 `rf4`；不包含远程 push、PR、Token／Nansen 独立接入、手动交易、单客户端登录功能开发或 Trader Sync 专属页面重排。

## 合并结果

| 对象 | 实际版本与处置 |
| --- | --- |
| 合并前 `rf4` | `8e24ee6409c27fbf564eaefb13b5a3a0e7a54ee0`；手动交易与单客户端登录的最新需求／设计保留 |
| UI 源分支 | `codex/ui-theme-refactor`，`0ec2fb4be2e6e99e39a03dc2a8a2b3b1a392f461`；最终生产代码提交 `78e473b79122441771f1c4ed27c9b15019d875d4` |
| 共同基点 | `bf752f27198c18a225d0c1bfe3c1415786d022fb` |
| 合并提交 | `2e29bb7bc484bb1cc9b3824955966b7fa489e94e`，父提交依次为上述 `rf4` 与 UI 源分支，双方历史均保留 |
| 源工作树 | `/home/yege/work/athena/.worktrees/ui-theme-refactor` 保留且干净；源分支未删除 |

唯一文本冲突是 `docs/requirements/README.md`。解决结果同时保留 UI 已正式实施、T1–T10／整分支审阅结论，以及手动交易和单客户端登录“设计已确认、代码未实现”的最新状态。自动合并的 `PRODUCT.md`、`docs/design/README.md` 与账户资料／本地偏好设计经过语义复核：固定深色、删除 Appearance／主题偏好、profile CAS／access revision／session generation 与新需求的会话边界没有互相覆盖。

合并前主工作区已有的 `.impeccable/config.json` 修改随合并提交保留。Inter 的 `overused-font` 例外仍限定于源工作树 `fonts.css`，并按整合后的实际路径补入 `ui/src/assets/fonts.css`；没有扩大为目录或全局忽略。Nansen 单一深色主题、Inter／JetBrains Mono、v23 修订、旧主题及 Appearance 清理均保持。

## 源码、生成物与证据一致性

- 合并树与 `codex/ui-theme-refactor` 在 `ui/`、`internal/`、`pkg/` 和 `assets/swagger.json` 无差异；最终生产清单的 164 个路径（含 21 个删除项）逐项匹配 `78e473b7`。
- 手动交易、单客户端登录的 `rf4` 独有需求与设计文件相对合并前 `rf4` 字节不变；本轮没有引入对应实现、兼容层或双路径。
- UI 原始证据仍保留在源工作树 `.tmp/ui-theme-refactor/`。为使合并后长期文档入口可访问，主工作区复制了完整证据树，原件未移动或删除：两端均为 16,530 个文件、4,904 个目录、4 个符号链接；文件树聚合 SHA-256 均为 `34eae3ce8cc4603d1d4bcce87802649232a4dbbf46404ceffc29d3e0d324f18a`，符号链接聚合 SHA-256 均为 `a8c49ef4796f10b0e14dc186b95cc9e8e66e171cd013196f289d2ef44df082fa`。验收文档 68 个直接证据链接均可访问。
- 三份上游原件仍保留已记录的空白字符：JetBrains Mono OFL、ReDoc LICENSE 与固定 ReDoc bundle。排除这三份逐字节保留的上游文件后，合并树 `diff --check` 通过；没有为清除提示而改写许可或 vendor 文件。

## 本轮验证

以下结果针对合并树；Go／Jest／lint／build 在创建合并提交前对完全相同的已暂存树运行，真实 smoke 在合并提交 `2e29bb7` 上运行。之后只更新本文和长期文档，不改变生产源码、依赖或运行配置。

| 验证 | 本轮实际结果与证据 |
| --- | --- |
| Go | `go test ./internal/... ./cmd/... ./util/... ./pkg/... ./tools/... -count=1` 退出 0；[日志](../../.tmp/rf4-ui-integration/go-test.log) |
| Jest | 38 suites／420 tests passed；保留既有 jsdom navigation／scrollTo console 噪音；[日志](../../.tmp/rf4-ui-integration/ui-jest.log) |
| lint | 12 项配置测试、TypeScript 与 ESLint 全部通过；[日志](../../.tmp/rf4-ui-integration/ui-lint.log) |
| build | Vite 构建通过；保留已审阅的最大 chunk 约 820.63 kB 提示；[日志](../../.tmp/rf4-ui-integration/ui-build.log) |
| 真实入口 | 独立实例会员／管理员入口均 HTTP 200，两个 bootstrap 均 authenticated；[入口](../../.tmp/rf4-ui-integration/http-entry-check.txt)、[member](../../.tmp/rf4-ui-integration/member-bootstrap.json)、[admin](../../.tmp/rf4-ui-integration/admin-bootstrap.json) |
| fresh smoke | 系统 Google Chrome 149.0.7827.53；会员／管理员 2 passed，0 unexpected／skipped／flaky，退出 0，run／cleanup passed；[报告](../../.tmp/athena-ui-acceptance/2026-09-15T07-26-09-583Z-c2725cfe/report.md)、[原始结果](../../.tmp/athena-ui-acceptance/2026-09-15T07-26-09-583Z-c2725cfe/smoke/results.json) |

本轮没有把历史证据冒充新提交重跑。生产代码逐项等同于 `78e473b7`，因此继续引用原 UI 验收中的版本化证据：`e7ec272b` 无过滤 acceptance 1018 passed、`82f25ad8` 无过滤 a11y 176 passed，以及 `78e473b7` 受影响 acceptance 40 passed 和最后真实 smoke 2 passed；它们的版本、过滤条件和限制见[主题实施与验收记录](web-ui-theme-refactor-acceptance.md#有版本的正式验证)。整分支最终无开放 Critical／Important；保留的三项 Minor 是 jsdom console 噪音、构建 chunk 提示与大型 fixture 重复基线。

## 真实环境与收尾

首次启动在 Notification 同步 Telegram bot profile 时因外部 Telegram HTTP 请求失败而自动停止；schema、构建及此前已启动服务没有代码错误。按既有验收模式启动本任务专用 loopback Telegram 边界后，同一实例第二次启动全部 ready，未向外发送 Telegram 消息。

| 项目 | 实际处置 |
| --- | --- |
| 独立实例 | 主工作区 `INSTANCE=rf4-ui-integration`，RunID `438eaaa9-447a-473b-a6c9-5165298f0f00`；UI 44000、API 48080、Notification 48086、Wallet 48088、Profit Sharing 48108、Trader Sync 48122 |
| 启动状态 | 六项服务与 PostgreSQL／Redis／MinIO 全部 ready；[就绪快照](../../.tmp/rf4-ui-integration/runtime-ready.json) |
| 停止 | 从同一主工作区执行 `make stop INSTANCE=rf4-ui-integration`，退出 0；运行会话退出 0；[停止后状态](../../.tmp/rf4-ui-integration/runtime-after-stop.json) |
| Telegram 替身 | 核对 PID、cwd 与命令后发送 TERM，进程退出 143 为预期；44931 已释放；[停止记录](../../.tmp/rf4-ui-integration/telegram-fixture-stop.json) |
| 端口与进程 | 44000、48080、48086、48088、48108、48122、44931 均释放；[端口核对](../../.tmp/rf4-ui-integration/ports-after-stop.txt) |
| 数据资源 | 本实例 PostgreSQL／Redis／MinIO 三个容器已停止，三个数据卷和容器均保留；未执行 reset、run-reset 或删除数据 |
| 其他任务环境 | 既有 7 个 OpenIM 相关容器运行清单前后无差异；[基线](../../.tmp/rf4-ui-integration/docker-before-names.txt)、[收尾](../../.tmp/rf4-ui-integration/docker-after.txt)、[空差量](../../.tmp/rf4-ui-integration/unrelated-container-delta.txt) |

截图、日志、审阅、浏览器报告、原始批准材料、源分支及源工作树均保留。本次只完成本地整合，没有 push、PR、分支删除或工作树删除。
