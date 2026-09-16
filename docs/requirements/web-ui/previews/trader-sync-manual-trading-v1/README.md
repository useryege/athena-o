# Trader Sync 手动交易 v1：页面审阅

> 状态：2026-09-15 用户回复「认可」，本批页面方案已确认，见[确认记录](approval.json)。本目录为虚构数据的静态页面设计；三段后端／交互详细设计已经确认，正式手动交易尚未开发。

[打开页面目录](index.html) · [设计说明](../../trader-sync-manual-trading-proposal.md) · [检查记录](checks.json)

## 桌面与手机

| 页面 | 桌面 | 手机 |
| --- | --- | --- |
| 交易钱包设置 | [Trading wallet](screenshots/wallet-desktop.png) | [Trading wallet](screenshots/wallet-mobile.png) |
| 买入 | [Buy](screenshots/buy-desktop.png) | [Buy](screenshots/buy-mobile.png) |
| 卖出 | [Sell](screenshots/sell-desktop.png) | [Sell](screenshots/sell-mobile.png) |
| 领取 | [Redeem](screenshots/redeem-desktop.png) | [Redeem](screenshots/redeem-mobile.png) |
| 持仓 | [My positions](screenshots/positions-desktop.png) | [My positions](screenshots/positions-mobile.png) |
| 本系统记录 | [System records](screenshots/history-desktop.png) | [System records](screenshots/history-mobile.png) |
| 成交历史 | [Trade history](screenshots/trades-desktop.png) | [Trade history](screenshots/trades-mobile.png) |
| 记录详情 | [Record details](screenshots/record-desktop.png) | [Record details](screenshots/record-mobile.png) |

## 关键辅助状态

- [账户启用结果待核对](screenshots/wallet-enable-unknown-mobile.png)
- [买入报价过期](screenshots/buy-expired-mobile.png)
- [首次卖出，比例为空](screenshots/sell-first-mobile.png)
- [领取前需要准备](screenshots/redeem-preparation-mobile.png)
- [持仓单钱包读取失败](screenshots/positions-partial-error-mobile.png)
- [历史分页视图过期](screenshots/history-page-expired-mobile.png)
- [交易结果待核对](screenshots/record-unknown-mobile.png)

页面底部可切换其余代表状态。主页面与状态样本是独立示例；页面中的地址、合约、市场、费用和金额均非实时事实。样稿没有网络交易能力；官网／来源入口说明将来的导航去向，时间／状态筛选与持续恢复仅表达布局。

## 复核方式

从仓库根目录、使用 `ui/.nvmrc` 的 Node 与已安装的项目 Playwright：

```bash
node docs/requirements/web-ui/previews/trader-sync-manual-trading-v1/verify.cjs
```

直接以本地文件读取样稿，无须启动业务服务。脚本保存全页截图、布局测量、原始 axe 结果及交互检查到 `checks.json`，并关闭其启动的浏览器。首轮窄屏放大失败的记录保留为 `checks-first-pass.json`；最终结果以 `checks.json` 为准。

本轮最多两次成组的渲染检查：首次定位问题，统一修正后确认。样稿验证不替代正式 React、交易服务、登录替换、账户一致性、费用或真实成交验收。
