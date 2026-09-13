# 第一批测试工具基础接入实施计划

用户已确认：仅完成工具基础接入，不迁移业务测试、不增加产品视觉覆盖、不改造现有验收 runner、不新增 CI。无产品行为和 API 变更。

## Task 1：工具、样例和命令

- 精确新增开发依赖 @testing-library/react 16.3.3、@testing-library/user-event 14.6.7、@testing-library/dom 10.4.1，更新 Yarn 锁文件，沿用 Jest/ts-jest/自定义 jsdom。
- 独立 React 测试表单验证标签定位、输入、点击、键盘提交及测试间自动清理。新增 yarn test:dom，定向样例也属于默认 Jest 集合。
- 独立 ui/playwright.visual.config.ts，仅运行工具视觉样例。用 page.setContent 固定内容/内联样式，不访问服务/数据库/外网。锁定当前 Chromium，固定视口/浅色主题/locale/时区、关闭动画。首批只维护 WSL/Linux 基线。
- toHaveScreenshot 比较固定区域，普通执行不创建/更新基线；缺失基线、视觉差异均失败。yarn test:visual 校验，yarn test:visual:update 显式更新。基线审查后提交，JSON/HTML、差异图和失败轨迹放 .tmp/ai-test-tools/visual/。
- internal/tradersync/pagination_fuzz_test.go：FuzzCursorRoundTrip，以 int64 快照号调用真实 EncodeCursor/DecodeCursor；非负往返一致且跨账户拒绝，负数拒绝。种子 0、1、9007199254740993、MaxInt64、-1。无新 Go 依赖、无 integration 标签。
- make test-fuzz 仅执行该目标，10 秒、2 workers；make test-ai-tools 按 DOM、视觉、Fuzz 顺序执行并保留失败，不更新基线、不下载依赖。
- 安装复用 Yarn 与现有 Chromium 命令，不新增安装器，不改默认产品验收。

## Task 2：验证、文档和交付

- DOM 样例及全量 Jest，前端 lint/类型/构建；相关 Go 分页测试、种子回归和限时 Fuzz。
- 基线两次匹配；临时样例改颜色/尺寸后失败并有差异图；普通命令不写基线。
- 聚合真实通过，并用临时命令替身验证任何阶段失败的退出码传递及后续阶段不运行。
- 工具链文档记录安装/命令/产物/基线审阅/Fuzz复现与覆盖边界，保存验收证据。
- 不启动业务服务、不运行真实产品 smoke。发现产品缺陷只记录，不修产品或削弱断言。
- 独立代码审查，带回原工作区，按仓库规则从仓库根发送一次中文完成通知。
