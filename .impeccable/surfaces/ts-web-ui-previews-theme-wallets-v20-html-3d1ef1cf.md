---
version: 1
slug: "ts-web-ui-previews-theme-wallets-v20-html-3d1ef1cf"
primary_target: "docs/requirements/web-ui/previews/theme-wallets-v20.html"
related_targets: []
---

# v20 Wallets 与 Solana 视觉方向

Mode: Operate。用户要求排除 Trader Sync，其余页面按已确认的通用视觉分批完成。本批与会员 Profit Sharing 两页组成四个实际页面，授权范围是原型、截图和文档。

THESIS: Wallets 以名称、完整地址和用途定位私有钱包；Solana 以发行候选、扫描进度和证据区分已知事实与未补齐信息。两者不展示未提供的余额、价格、评级或身份推断。

OWN-WORLD: 继承 Nansen 单深色、v13 会员壳、v18 分隔记录、Inter 与 JetBrains Mono、44px 控件和既有语义色。第一轮当前 React 对照已读取，两页各桌面和手机采用同批明确合成 GET 数据，不执行写请求。

FIRST VIEWPORT: 224px 侧栏、64px 顶栏、32px 桌面边距。Wallets 页头依次为 Refresh、Import、Create，查询和 EVM/Solana 筛选与钱包列表共处一个面板；名称与地址为主要信息，类型、来源和时间辅助阅读。Solana 标题后是紧凑独立扫描状态，随后查询与候选记录；错误和缺失直接可见，链上证据按需展开。手机20px边距，记录纵排，完整地址换行，详情/表单保持固定标题和操作区。

FORM-SEED: not-applicable。现有业务页面对已批准视觉体系的普通扩展，code-led；不重开视觉世界选择，不生成图片 mock。没有本批 approved comp。v13/v18 截图为延续参考。

QUALITY BAR: 首屏可识别钱包和候选，地址可完整复制；新增/导入与查看私钥有真实语义的视觉区别。Solana 四个元数据/来源状态可辨，Unknown 不写成零值，slot 不经过浮点舍入；手机不靠整页横向滚动。辅助状态有清楚恢复入口，不能把静态模拟写成真实结果。

Wallets 来源：member/pages/wallets.tsx 与 shared/services/wallet-service.ts。保留 EVM/Solana、创建/导入1–10个、每行一个可见私钥、单个可选备注/批量自动命名、50 Unicode字符限制、默认或八个既有 avatar presets、详情中备注/头像编辑、私有图片JPEG/PNG/非动画WebP≤2MiB、五分钟会话内私钥查看 lease；只读不含私钥。原型提供本地演示与合成结果，不执行密钥创建、导入、上传、身份认证或链上请求。Worm凭据与钱包查看lease不合并。

Solana 来源：member/pages/solana.tsx、shared/services/solana-service.ts、requirements/solana/README.md、design/web-ui/solana-discovery.md。保留名称/符号/Mint搜索（Mint大小写敏感）、25每页、Refresh仅读取保存结果；发行来源与Token程序独立、fee payer非创始人、初始化权限非当前权限。未知/待处理/失败各自标识；完整交易与mint用固定Solscan域。扫描状态和列表请求各自有失败反馈。合成样本不说明实际扫描器正在运行，不启动已要求暂停的Solana采集。

FINISH: 最多两轮自身批量截图/检查；合并一次detector，由独立finish reviewer审阅后文档化。所有shipping PNG保存来源。既有 DESIGN.md 与 .impeccable/design.json 缺失保留。全部当前ui/和已有批准资产保持不变。借用root Vite localhost:4000/PID308228，浏览器关闭，无新服务/外部写动作/通知邮件。

## 独立审阅命名修正

根据 .superpowers/v20-finish-review-initial.txt 第2项，地址复制、交易复制/外链、分页和钱包类型按钮改为至少44×44实际命中区域；保留内部SVG尺寸。四种尺寸补实际几何断言，重拍同11张提案图，不改变4张既有React基线、不重跑detector。
