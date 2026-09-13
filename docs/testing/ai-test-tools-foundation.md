# 第一批测试工具基础接入验收

日期：2026-09-13。范围仅为用户确认的工具基础接入，不是产品业务验收。

## 交付

- 精确接入 React Testing Library 16.3.3、DOM Testing Library 10.4.1、user-event 14.6.7，沿用原有 Jest/ts-jest/jsdom。
- 独立表单样例覆盖标签定位、输入校验、有效点击提交、Enter 提交与测试间自动清理。
- 独立 Chromium 视觉样例只渲染固定内容，提交经图像检查的 WSL/Linux PNG 基线；普通执行禁止自动新增或更新基线。
- Go 原生 Fuzz 样例调用真实分页游标编解码；种子覆盖零、普通值、超出 JavaScript 安全整数范围的值、int64 上限和负数。
- `make test-ai-tools` 顺序运行 DOM、视觉、Fuzz。Fuzz 入口禁用模块/工具链自动下载，并以只读模块模式运行。

## 验证证据

原工作区基线：29 个 Jest suite／348 项测试通过；前端 lint/类型检查通过；分页相关 Go 测试通过。
Jest 基线和接入后的日志均包含既有 jsdom 导航未实现提示，相关测试仍通过；本批未修改这些产品测试。

| 检查 | 结果 |
| --- | --- |
| 锁文件安装 | `yarn install --frozen-lockfile --offline` 成功 |
| DOM 最小样例 | 1 个 suite／4 项测试通过 |
| 全量 Jest | 30 个 suite／352 项测试通过 |
| 前端 lint/类型 | `yarn lint` 成功，其中 ESLint 配置的 12 项回归通过 |
| 新增样例及配置类型 | 对 DOM 样例、视觉样例、视觉配置显式运行 `tsc --noEmit`，通过 |
| 前端构建 | `yarn build` 成功；保留包体积超过 500 kB 的提示 |
| Go 分页与种子 | 相关分页测试及 Fuzz 种子回归通过 |
| 实际聚合入口 | DOM、视觉、限时 Fuzz 三阶段顺序通过；本次 10 秒 Fuzz 完成 128679 次执行，无失败 |
| 基线缺失 | 普通视觉命令正确失败，保留报告和 trace，未创建基线 |
| 正常视觉比较 | 连续两次匹配通过，PNG 哈希均不变 |
| 视觉差异 | 在临时副本中修改背景颜色；退出码 1，生成实际图／预期图／差异图和 trace，原始及副本基线哈希均不变 |
| 聚合失败传播 | 在临时命令替身下以 `make -j4` 分别令 DOM、视觉、Fuzz 失败；三个场景均非零退出，后续阶段均未执行；全成功场景顺序正确 |

已审阅基线 SHA256：`b122bc8e5d402356d7c6145de6483762240eb2431c603a533cb6cac21cbe5d0a`。

常规视觉报告位于 `.tmp/ai-test-tools/visual/`；本次完整日志与保留的负向报告位于
`.tmp/ai-test-tools-implementation/`。其中 `visual-probes.json` 记录两次匹配和差异测试，
`aggregate-probes/results.json` 记录阶段顺序和退出码。原始报告是本地忽略产物，不随 Git 分发。

## 边界

- 未迁移现有产品组件测试，未建立产品页面视觉基线；本次静态样例不是 ATHENA 页面。
- 未启动业务服务或数据库，未执行真实产品 smoke；现有浏览器验收入口和配置未修改。
- 10 秒 Fuzz 仅证明本次样例探索未发现失败，不代表全部输入或其他解析器已覆盖。
- 未新增 CI、安装器、产品逻辑、兼容分支或第二批工具。使用方法见[开发工具链](../developer-guide/toolchain-guide.md#测试工具基础接入)。
