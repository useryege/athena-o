# Solana 发现列表

状态：首版信息展示在源分支已实现并验收，用户已认可；现已按用户授权集成到 `rf4`，并通过[本次集成验证](../../testing/rf4-branch-integration.md)。预览继续暂停，见[设计总览](../solana-intelligence/README.md)。

成员端在 `/solana` 提供只读的 Solana 新 Mint 候选列表。页面位于 Token & Risk 分组，只有拥有 `Solana` `READ` 权限的成员能看到导航项或进入路由；管理员账户权限编辑器包含同一只读模块。

页面沿用成员端 Operate 布局：标题和候选范围说明、扫描状态、名称/符号/Mint 查询与服务端分页表格。首列显示名称、符号和可复制 Mint，发行来源独立显示；发行时间、发现时间、精度和交易保留。名称/符号是链上快照，不代表认证或项目类别；不显示当前权限或发行人推断。Mint 和交易地址可复制，并只链接到固定的 `https://solscan.io` 域；地址作为路径段编码。

支持的发行来源文案为 Pump.fun、Raydium LaunchLab、Direct Token initialization；未知来源显示 Unrecognized platform。来源 pending/error 与未知区分；名称缺失分别显示 Metadata pending、Name unavailable、Metadata read failed，空符号显示 Symbol unavailable。展开详情显示初始化权限、付费账户、Token 程序、发行程序、元数据来源、账户、观察 slot 和时间。Token 程序与发行平台是不同事实；未知 CPI 的发行程序表示可证明的直接父程序。

扫描状态展示服务返回的起点、已处理 slot、最新 finalized slot、最后成功时间、总候选数和错误。slot 与 int64 JSON 值在客户端保留为十进制文本，避免 JavaScript 数值精度丢失；时间仅在安全整数范围内转换。空列表、请求错误和刷新均使用成员端现有状态组件。移动端表格在本身的横向滚动容器中显示。

数据来自 `GET /api/v1/solana/projects` 和 `GET /api/v1/solana/status` 的成员端代理路径。查询使用 `page`、`pageSize` 和 `query`；默认每页 25 条，提交查询回到第一页。名称/符号子串忽略大小写，Mint 保留大小写语义。刷新只读取已保存的发现结果和扫描状态，不触发扫描、补全或研究；后台补全后可手动刷新查看。

## 视觉与功能确认的边界

本轮确认的是独立列表、字段、搜索及链上详情。验收截图记录当时页面实现，不代表全站最终视觉主题。后续页面视觉须遵循当前项目已确认的 Nansen 参考、单一深色、Inter 英文/正文/数字与 JetBrains Mono 地址/哈希方向；主题与字体的权威需求为[全站视觉主题](../../requirements/web-ui/visual-theme.md)及其关联文档。本次集成保留已有确认范围，不将首版认可扩大为全站视觉重构完成。
