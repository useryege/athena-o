# Solana 发现列表

状态：已实现。

成员端在 `/solana` 提供只读的 Solana 新 Mint 候选列表。页面位于 Token & Risk 分组，只有拥有 `Solana` `READ` 权限的成员能看到导航项或进入路由；管理员账户权限编辑器包含同一只读模块。

页面沿用成员端 Operate 布局：标题和候选范围说明、扫描状态、Mint 查询与服务端分页表格。候选保留原始 Mint、Token Program、交易和初始化事实，不显示名称、项目类别、当前权限或发行人推断。Mint 和交易地址可复制，并只链接到固定的 `https://solscan.io` 域；地址作为路径段编码。

扫描状态展示服务返回的起点、已处理 slot、最新 finalized slot、最后成功时间、总候选数和错误。slot 与 int64 JSON 值在客户端保留为十进制文本，避免 JavaScript 数值精度丢失；时间仅在安全整数范围内转换。空列表、请求错误和刷新均使用成员端现有状态组件。移动端表格在本身的横向滚动容器中显示。

数据来自 `GET /api/v1/solana/projects` 和 `GET /api/v1/solana/status` 的成员端代理路径。查询使用 `page`、`pageSize` 和 `query`；默认每页 25 条。刷新只读取已保存的发现结果和扫描状态，不触发扫描或研究。
