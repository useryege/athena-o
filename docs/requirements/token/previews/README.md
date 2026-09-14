# 钱包数据展示预览

本目录保存 Token 钱包 Nansen 数据展示的独立页面提案，与正式 `ui/` 分开。用户于 2026-09-14 确认 v1 页面设计，反馈“这一版本很好，通过”，见[确认记录](wallet-analytics-v1-approval.json)；正式业务接入尚未实现。业务以[钱包需求](../wallet-trading-performance.md)为准，布局、样本范围和状态说明见[页面提案](../wallet-analytics-page-proposal.md)。

- [交互 HTML](wallet-analytics-v1.html)：直接用浏览器打开即可；本地字体位于 `../../web-ui/previews/files/`，无需启动服务。
- [桌面首屏](wallet-analytics-v1-desktop.png)、[完整桌面](wallet-analytics-v1-desktop-full.png)、[交易明细](wallet-analytics-v1-desktop-transactions.png)。
- [手机首屏](wallet-analytics-v1-mobile.png)、[逐币表现](wallet-analytics-v1-mobile-tokens.png)、[单条交易](wallet-analytics-v1-mobile-transaction.png)。
- [提取数据](wallet-analytics-v1-data.json)、[浏览器检查记录](wallet-analytics-v1-checks.json)、[独立审阅](wallet-analytics-v1-review.json)与[设计记录核对](wallet-analytics-v1-documentation.json)。

查看入口是本目录中的完整 `wallet-analytics-v1.html`。`.superpowers/wallet-analytics-preview/page.html` 是生成脚本读取的中间模板，直接打开会缺少样式、脚本和样本数据，不用于页面审阅。

独立审阅与设计记录核对保留生成时的状态；后续用户确认单独记录在 `wallet-analytics-v1-approval.json`，不回写此前检查结论。

数据来自 2026-09-13 的保存样本，非实时状态。内嵌数据不包含 API Key；页面 CSP 禁止网络数据请求。样本覆盖之外的日期、代币或交易后续页明确提示，不模拟为真实接口成功。截图是浏览器渲染的页面证据，未使用生成式图像。

浏览器与检查脚本的运行证据保存在 `.tmp/athena-ui-acceptance/wallet-analytics-preview-v1/`。这份证据属于文档预览检查，不是 ATHENA 平台真实环境验收；本轮没有启动开发服务。
