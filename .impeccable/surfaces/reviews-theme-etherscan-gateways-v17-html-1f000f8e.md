---
version: 1
slug: "reviews-theme-etherscan-gateways-v17-html-1f000f8e"
primary_target: "docs/requirements/web-ui/previews/theme-etherscan-gateways-v17.html"
related_targets: []
---

# v17 Etherscan Gateways 视觉提案

Mode: Operate。继续用户已授权的逐页设计，覆盖现有管理员 Etherscan Gateways。仅构建原型、截图、文档，正式 ui/ 不修改。

## Direction contract

THESIS: 将网关进程状态和实际请求测试分开阅读。网关运行不等于 Etherscan 请求通过，两个页签都显示所属来源摘要。

OWN-WORLD: 承接已确认 Nansen 单深色主题、v2 Inter／标识专用 JetBrains Mono、v3 导航尺寸、v4 状态反馈、v15 管理员壳及 v16 独立状态的阅读层级。品牌青绿用于操作和选中，成功／警告／失败使用既定语义色。

STORY: 管理员先定位不可达的网关与错误，必要时检查连接信息；切到 Live Probe 后设置现有两项参数，运行请求测试，再按整体结果、网关／API key 汇总及错误样本定位问题。

FIRST VIEWPORT: 224px 桌面侧栏、64px 顶栏、32px 内容边距，页头右侧 Refresh gateway status。Gateways 与 Live Probe 页签带来源摘要。默认展示网关统计细行和服务记录，地址、Reachable、Runtime、Latency 直接显示；每项保留 Checked、错误，Base URL 按需展开。Probe 先表单，再最新运行结果与失败分类，按网关／按 key 两种汇总可切换，时序和样本可展开。手机 20px 留白，记录纵向重排，表单标签和全宽操作不拥挤。

FORM: 普通扩展、code-led；seed key: not-applicable。延续已明确的现有页面和视觉系统，不进行视觉世界竞争。关键交互是来源页签、诊断展开、两个汇总维度切换，以及运行中保持参数只读；短颜色反馈约160ms，无装饰动画。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## 事实和边界

以 ui/src/app/admin/pages/etherscan-gateways.tsx、shared/services/service-status-service.ts、internal/server/servicestatus/etherscan_gateway_probe.go 及 internal/etherscangatewayprobe/gateway_probe.go 为准。源实现输入 interval1–1000ms、requestsPerKey1–20，默认10和6；只允许管理员启动、运行中禁用重复提交。正式探针会发出使用已配置 key/gateway 的真实请求；本地原型只模拟，不调用 API、不消耗额度。

运行总结果以服务端 result 与 requiredSuccess 为准，门槛为90%向上取整。按实体的行结果仍沿用前端：严重分类失败需检查，否则成功率低于90%为 below90%。不能把两种结果混为一谈。总数是keys×requestsPerKey，不额外乘gateway数。七种失败分类、成功、总数、时间、已完成数、请求间隔、Start Spread、网关和key汇总、完整错误样本保留。后台只在运行完成后更新结果，不制造实时逐请求进度或取消能力。0／缺失与来源格式按实际字段处理。

## QUALITY BAR

网关运行与测试结果清晰区分；参数可读、单位明确，成功门槛和失败原因能核对。手机不缩成微型宽表；完整host:port、URL、错误样本可以读取和复制。运行失败、请求读取失败和启动失败分别说明。沿用管理员导航和280px／视口减48的抽屉；不增加编辑key、管理网关、重启、取消或未授权功能。

## 验证与落档

先对照现有 React，同批合成数据、只拦截GET，不实际启动探针。原型验证1440、390、320及720px／200%字号，最多两轮自身截图检查，随后独立finish reviewer和documenter。截图嵌入来源，232份既有已批准摘要保持不变。

完成后只维护etherscan-gateways-proposal.md、visual-theme.md和三份索引，v17待确认；不扩大用户对v16四张图的认可。原先缺失的DESIGN.md与.impeccable/design.json保持原样。借用root既有Vite localhost:4000，任务结束关闭浏览器、不停止共享环境。
