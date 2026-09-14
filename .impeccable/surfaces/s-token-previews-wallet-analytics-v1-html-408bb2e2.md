---
version: 1
slug: "s-token-previews-wallet-analytics-v1-html-408bb2e2"
primary_target: "docs/requirements/token/previews/wallet-analytics-v1.html"
related_targets: []
---

# 钱包 Nansen 数据展示页面提案

模式：Operate。本轮用户同意按后端设计、页面预览、正式实现的顺序推进；这一步交付可交互 HTML 与真实保存样本的桌面／手机预览。正式 API、ui/、平台缓存均未实施。业务依据为 docs/requirements/token/wallet-trading-performance.md 和 docs/superpowers/specs/2026-09-14-token-wallet-analytics-backend-design.md。

## Direction contract

THESIS：用户先读钱包区间总览，再核对逐币表现及交易中的资产。提供来源与完整值，准确区分零、缺失和微量余额。胜率与本地计算均不纳入本页。

OWN-WORLD：沿用已确认的 v1–v5 Nansen 深色主题、字体、导航和控件；不是新视觉世界。配色、密度以 docs/requirements/web-ui/visual-theme.md 与 v5 截图为准，不改全站文件。

STORY：输入 Ethereum 钱包和 7／30／90 天范围；阅读两项摘要；按合约地址筛选下方两份列表；查看完整数值与交易哈希。钱包摘要不随代币筛选改变。

FIRST VIEWPORT：224px 侧栏和 64px 顶栏；32px 内容边距。页头左侧标题、右侧次要刷新；集中地址和日期查询；一块双指标总览；逐币表格包含持仓和两种盈亏。手机按相同任务顺序改成分组行，资产与金额保持可读。

FORM：用户已授权的精确页面扩展，直接代码原型；seed 不适用，不新开概念抽签或生成图片。关键交互是点代币筛选两份列表、总览保持原样，再清除返回；轻量 160ms 状态反馈，尊重 reduced motion。真实样本只覆盖记录过的接口范围；缺失样本明确提示，不能伪造请求或数据。

FINISH：unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

本轮保留现有全站文件和已有 DESIGN.md 缺失状态。仅新增本页提案、HTML、提取数据、截图与检查记录，更新相关 Token 索引。用户于 2026-09-14 对完整预览反馈“这一版本很好，通过”，v1 页面设计已确认；范围和文件摘要见 docs/requirements/token/previews/wallet-analytics-v1-approval.json。未将整体确认扩展为逐图查看辅助状态或正式功能验收；样本读取不消耗 Nansen credits。开发契约不进入交付 HTML。
