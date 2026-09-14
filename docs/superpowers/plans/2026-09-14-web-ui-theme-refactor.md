# 全站单一深色 UI 重构实施计划

> **执行代理要求：** 使用 Superpowers 的 `executing-plans` 逐任务实施；采用子代理执行时使用 `subagent-driven-development`。按复选框记录实际进度，不把文档中的预期结果写成已通过证据。

**目标：** 将当前会员端、管理员端和共享组件实现为已确认的 Nansen 单一深色视觉，完成主题偏好清理和实际页面验收。

**架构：** 保留 React／Ant Design 和独立 realm 应用，统一 CSS token、入口级 Theme Provider 与既有中立组件。业务页按批准版本重排展示，继续使用当前 API、权限与状态机；主题专用 preferences 沿数据库至前端消费链一次清理。

**技术栈：** 当前仓库 React 18、Ant Design 6、React Router 6、TypeScript、Vite、Jest／Testing Library、Playwright／axe；偏好清理涉及 Go、PostgreSQL、sqlc、protobuf。

**方案：** [技术方案](../specs/2026-09-14-web-ui-theme-refactor-design.md)、[路由和状态覆盖](../../requirements/web-ui/theme-refactor-coverage.md)、[机器清单](../../requirements/web-ui/theme-refactor-coverage.json)。执行者必须同时读取方案及本计划。

**审阅修订：** 用户已授权修复总体审阅问题；本计划采用[一致性修订契约](../../requirements/web-ui/theme-consistency-contract.md)与 [v23 可运行参考](../../requirements/web-ui/previews/theme-consistency-v23/README.md)。本次修正计划和设计参考，以下正式实施任务仍未执行。

## 全局约束

- 本计划已编写，以下任务均未实施。用户已确认主视觉和规划方向，本轮没有完成生产 UI 或真实业务验收。
- Node 使用 `ui/.nvmrc` 的 `24.14.1`，允许范围 `>=24.14.1 <25`；不改变全局 NVM 默认值，不升级组件库。
- 单一深色；页面 `#06080B`、面板 `#0F1114`、较亮层 `#181D22`、分隔线 `#252A30`、文字 `#FFFFFF`／`#9FA0A1`、强调 `#00FFA7`。
- Inter 用于英文、正文与数字，JetBrains Mono 400 用于地址／哈希；本地字体及 OFL 直接来自批准资产。
- 桌面侧栏 224px、顶栏 64px、内容边距 32px；手机外侧 20px；应用壳切换点 900px；主要操作目标至少 44px。
- 37 条页面入口按批准版本实现，2 条 Appearance 删除，4 条路由／兜底规则覆盖；8 条 Trader Sync 专属路由暂缓重排、保留共享影响回归。
- Token／Nansen 接入仍由[独立计划](2026-09-14-token-wallet-analytics.md)负责，不在此增加接口或开启导航。
- 正式页面不包含 sample 切换、模拟成功或审阅面板；不重新计算业务金额，不丢完整地址、权限或未知状态。
- 不为旧主题保留 flag、双实现或兼容重定向；不清除分页、排序、导航等其他偏好。
- 原始 HTML、PNG、data、checks、review 和 approval 不修改。所有执行截图、trace 和报告保存到独立验收目录。
- 纯视觉样式以浏览器证据验证；行为及契约变化先写有意义的失败测试，不能只测试代码常量等于自己。
- 每个任务完成其有界验证后再推进；最终扩大为无过滤 acceptance、a11y 和真实 smoke。过滤运行必须标记为局部。
- 执行前依 `using-git-worktrees` 检查隔离需求；现有未提交设计及其他任务文件保持。每个可审阅任务单独提交时，只包含自身变更，不自动 push／merge。

## 文件结构与接口

现有 `AppPage`、`Section`、`ResourceTable`、`ChoiceGroup` 和显示组件继续原位维护。各 realm 保留独立路由与 service 注册；共享文件不导入 `member/` 或 `admin/`。

| 新建／改动 | 文件与职责 |
| --- | --- |
| 新建 | `ui/src/app/styles/tokens.css`：统一颜色、字体角色、空间变量 |
| 新建 | `ui/src/app/shared/athena-theme.tsx`：导出 `createAthenaTheme(): ThemeConfig`、`AthenaThemeProvider({children}: {children: React.ReactNode})` |
| 新建 | `ui/src/app/shared/athena-color-roles.ts`：从 CSS 读取完整最终色映射，导出 `createAthenaColorRoles(read: (name: string) => string): Partial<GlobalToken>` |
| 修改 | `ui/package.json`、`ui/yarn.lock`：将当前已安装的 `@ant-design/cssinjs` 2.1.2 声明为直接依赖；不升级 Ant Design |
| 新建 | `ui/src/assets/fonts/athena/`：两份 WOFF2 和两份许可 |
| 修改 | 两份 HTML、`entry/member.tsx`、`entry/admin.tsx`、`session/bootstrap.tsx`：首屏及 Provider 归属 |
| 修改 | `styles/shared.css`、`styles/member.css`、`styles/admin.css` 和各自 features CSS：共用与业务视觉 |
| 修改 | `member/app.tsx`、`admin/app.tsx`、共享账户及各域页面：按覆盖清单改造 |
| 删除／修改 | `shared/theme.ts`、`view-preferences-service.ts`、`models.ts`、`accounts-service.ts` 及主题专用后端链路：移除旧偏好 |
| 新建 | `ui/e2e/theme-refactor/contracts.ts`、`routes.ts`、`cases.ts`、`fixtures/*.json`：可复用受控场景 |
| 新建 | `ui/e2e/theme-refactor.spec.ts`、`theme-refactor-a11y.spec.ts`：React 视觉／交互及无障碍 |
| 修改 | `ui/playwright.config.ts`、`scripts/acceptance-runner.mjs` 及其测试：纳入新场景并支持局部过滤 |
| 新建 | `docs/testing/web-ui-theme-refactor-acceptance.md`：执行时记录真实结果和环境归属 |

所有新函数签名在所属任务定义。各业务任务消费既有组件接口，不新建抽象业务 service。新增 fixture 文件在其业务任务中同时完成，避免用空数组冒充成功数据。

## T1：单深色入口、字体与主题偏好清理

**文件：** 新建上述 tokens、theme provider、color roles、字体；修改 package／lock、两份 HTML、entry、bootstrap、两份 app、共享 account-center、models、accounts-service、view-preferences-service；删除 `shared/theme.ts`。后端修改 `internal/accountcenter/{types,manager}.go`、`internal/accountstate/store/{migrations/000001_init.sql,queries/account_center.sql,queries/account_directory.sql,sql_store.go}`、`internal/server/account/{account.proto,account.go}`、`internal/server/session/{session.proto,session.go}`、`internal/server/authz.go`，生成对应产物和 schema contract。

**消费：** 已批准颜色与本地字体、现有 profile／access／session 合同。

**产出：** 单一 `AthenaThemeProvider`；`ViewPreferences` 不再含 theme；API 不再有主题 preferences，其他身份与资料字段不变。

- [ ] 在 `ui/src/app/shared/services/view-preferences-service.test.ts` 写“主题字段不再持久化，分页和隐藏侧栏仍保存”的失败测试；补 `internal/accountstate/store/theme_removal_integration_test.go` 验证新 schema 没有 preferences 表，并通过现有账户创建集成测试验证 profile／grant 仍完整；保留 profile CAS 和账户创建原子性测试。

```tsx
import {ViewPreferencesService} from './view-preferences-service';

test('保存当前视图偏好时去除主题字段并保留分页', () => {
    const key = 'athena.theme-refactor.test';
    localStorage.setItem(key, JSON.stringify({version: 6, pageSizes: {wallets: 50}, theme: 'light'}));
    const service = new ViewPreferencesService(key);
    service.init();
    service.updatePreferences({hideSidebar: true});
    const saved = JSON.parse(localStorage.getItem(key)!);
    expect(saved.theme).toBeUndefined();
    expect(saved.pageSizes).toEqual({wallets: 50});
    expect(saved.hideSidebar).toBe(true);
    localStorage.removeItem(key);
});
```

- [ ] 运行该 Jest 用例，确认旧实现因保存 theme 失败。Go 新用例用现有 `pgtest` 创建任务专用库；先准备其要求的测试 PostgreSQL DSN，不能把缺少 DSN 导致的 skip 当作通过。失败原因必须来自目标不变量。

```go
//go:build integration

package store_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/useryege/athena/internal/accountstate/store/migrations"
    "github.com/useryege/athena/internal/testutil/pgtest"
)

func TestCanonicalSchemaHasNoThemePreferences(t *testing.T) {
    db := pgtest.New(t, migrations.FS, migrations.Dir)
    var present bool
    err := db.Pool.QueryRow(context.Background(), "SELECT to_regclass('public.account_preferences') IS NOT NULL").Scan(&present)
    require.NoError(t, err)
    require.False(t, present)
}
```
- [ ] 删除主题专用 SQL／创建 CTE／类型与 API 源。SQL 稳定后运行 `make sqlc-local`、`make account-state-schema-contract`；proto 稳定后运行 `make protogen`；然后修正手写消费者和既有 fixture，记录生成范围。不得手改生成物。
- [ ] 删除两条 Appearance 路由、菜单、切换回调、theme props、server 同步和 profile merge 中的 preferences revision。保留 profile revision 与 access revision 单调性、accountId／iss 校验和其他视图字段；现有本地缓存采用支持字段白名单读取，不写历史格式转换分支。
- [ ] 建立 tokens 与本地字体，删除浅色根变量和旧深色覆盖。HTML 固定深色属性、首屏背景；删除 `matchMedia('(prefers-color-scheme: dark)')` 和存储主题探测。Provider 由两个 entry 包裹整个 App；bootstrap 和注册移除重复 ConfigProvider／AntApp。

```tsx
// ui/src/app/shared/athena-theme.tsx
import * as React from 'react';
import {App, ConfigProvider, theme} from 'antd';
import type {ThemeConfig} from 'antd';
import {StyleProvider, px2remTransformer} from '@ant-design/cssinjs';
import {createAthenaColorRoles} from './athena-color-roles';

const transformers = [px2remTransformer({rootValue: 16, mediaQuery: false})];

export const createAthenaTheme = (): ThemeConfig => {
    const style = getComputedStyle(document.documentElement);
    const read = (name: string) => style.getPropertyValue(`--athena-${name}`).trim();
    const roles = createAthenaColorRoles(read);
    return {
        cssVar: {key: 'athena-theme'},
        algorithm: (seed, map) => ({...theme.darkAlgorithm(seed, map), ...roles}),
        token: {
            fontFamily: read('font'), fontSize: 14, controlHeight: 44, borderRadius: 10,
            motionDurationFast: '0.16s', motionDurationMid: '0.2s'
        },
        components: {
            Button: {
                primaryColor: read('bg'), dangerColor: read('bg'), primaryShadow: 'none', dangerShadow: 'none', defaultShadow: 'none',
                defaultColor: read('text'), defaultBg: read('panel'), defaultBorderColor: read('border-strong'),
                defaultHoverColor: read('text'), defaultHoverBg: read('panel-elevated'), defaultHoverBorderColor: read('border-strong'),
                defaultActiveColor: read('text'), defaultActiveBg: read('panel-elevated'), defaultActiveBorderColor: read('border-strong'),
                borderColorDisabled: read('border')
            },
            Input: {activeBorderColor: read('primary'), hoverBorderColor: read('primary'), activeShadow: 'none'},
            Segmented: {trackBg: read('panel'), itemSelectedBg: read('selected-bg'), itemSelectedColor: read('primary')},
            Pagination: {itemActiveBg: read('panel')},
            Alert: {defaultPadding: '16px 20px'}
        }
    };
};

export const AthenaThemeProvider = ({children}: {children: React.ReactNode}) => {
    const [config] = React.useState(createAthenaTheme);
    return <StyleProvider transformers={transformers}>
        <ConfigProvider theme={config}><App>{children}</App></ConfigProvider>
    </StyleProvider>;
};
```

`createAthenaColorRoles` 的完整实现采用 [v23 参考中的同名函数](../../requirements/web-ui/previews/theme-consistency-v23/theme-reference.cjs)，迁为 TypeScript 并导入 `GlobalToken` 类型；它包括主色各态、反馈四组背景／边框／文字、链接、表面和辅助文字，禁止退回只设置 seed 的旧示例。CSS 的完整键和值采用[修订 token 表](../../requirements/web-ui/previews/theme-consistency-v23/tokens.css)，`--athena-brand`／`--athena-accent` 等消费名指向同一 primary。

- [ ] 在共用 CSS 迁入 [v23 rem 桥接及控件高度规则](../../requirements/web-ui/previews/theme-consistency-v23/corrections.css)：Ant 6 的变量声明不会经过 `px2remTransformer`，须在 `.athena-theme.athena-theme` 映射全局字号、Button／Input 字号和控件高度；HTML 不固定根字号。确认 16→32px 根字号下，按钮、输入、提示正文的实际字体 14→28px，正文 16→32px。浮层必须沿同一 Provider／theme key；使用 App 上下文的 message／modal／notification，不能由未包裹的静态调用逃离主题。
- [ ] 以实际 Ant 控件测主按钮和危险按钮 normal／hover／active、四类反馈背景／边框／图标和只读字段。逐项断言最终计算样式与修订契约一致，记录字体实际增幅及容器裁切；颜色配置对象正确不足以通过。v23 的 SSR 样本只提供可复现依据，正式 React 的 portal、交互及其余组件仍在 T2／T9 验证。

- [ ] 运行局部 Jest、`go test ./internal/accountcenter/... ./internal/server/account/... ./internal/server/session/... ./internal/server/appbootstrap/...`，以及 `go test -tags=integration ./internal/accountstate/...`。`ui` 内运行 `yarn lint`，检查全部消费者已更新。
- [ ] 检查生产源无 theme 切换／preferences RPC 消费；本地分页与排序存储仍可使用。后续 T2 浏览器用例补齐浅色系统、首屏、注册和主题旧缓存验证。提交本任务时包含源、生成物、前后端消费者及相关长期账户文档；不先合并一半 API 契约。

## T2：应用壳、共用组件与浏览器场景接线

**文件：** 两份 `app.tsx`、`components/{layout,display,resource-table,choice-group}.tsx`、`styles/{shared,member,admin}.css`；新建 e2e/contracts、routes、cases、主 spec 与 a11y spec；修改 Playwright config、acceptance runner 及测试。

**消费：** T1 Provider／token，原 realm 注册和既有组件 props。

**产出：** 224px／64px／900px 应用壳；现有组件的统一外观；下述受控场景接口供 T3–T9 使用。

```ts
// ui/e2e/theme-refactor/contracts.ts
export interface ThemeReply {
    method: string;
    path: string;
    realm: 'member' | 'admin';
    status: number;
    json: unknown;
    delayMs?: number;
}
export interface ThemeCase {
    id: string;
    route: string;
    realm: 'member' | 'admin';
    heading: string;
    replies: ThemeReply[];
}
export interface ThemeLedger {
    requests: Array<{method: string; path: string; query: string; realm: string | undefined; body: unknown}>;
    unexpected: string[];
}
```

`routes.ts` 导出 `installThemeCase(page: Page, scenario: ThemeCase): Promise<ThemeLedger>`、`openThemeCase(page: Page, id: string): Promise<ThemeLedger>`、`assertThemeLedger(ledger: ThemeLedger): void`、`assertThemeLayout(page: Page): Promise<void>`。`cases.ts` 导出 `themeCases: ThemeCase[]`，每个任务追加实际 API-shaped fixture；案例不存在立即失败，不回退默认成功。

- [ ] 扩展 runner 的可选 `UI_ACCEPTANCE_GREP`：作为独立 `--grep` 参数传 Playwright，原字符串不经 shell 拼接；报告记为 filtered。设置过滤时 acceptance 只运行 `ui-fixtures`，a11y 只运行 `a11y`，不向无匹配用例的 live 项目传该过滤；未设置时保持完整 ui-fixtures／live 路径。先在现有 runner 测试中断言项目选择、有过滤／无过滤 argv 和报告区别，再实现；零匹配必须失败。
- [ ] Playwright 的 `ui-fixtures` 项目匹配 `trader-sync.spec.ts` 与 `theme-refactor.spec.ts`，a11y 项目匹配已有和新 a11y spec。保留原 live 与 smoke 项目，不增加第二套服务启动器；root 和 `/athena` 两次运行使用现有 harness。
- [ ] `installThemeCase` 只拦截声明的当前 ATHENA API／auth，按 method、部署相对 path 和 realm 匹配；记录 query、body 和请求次数。未声明 API 记录 unexpected 并返回明确错误；跨域业务请求 abort，静态同源资源继续加载。PUT／POST 必须有对应 fixture，不因 GET 成功就模拟写成功。`delayMs` 只模拟有界 pending；首次失败／重试成功等状态用测试安装的精确 URL 覆盖响应驱动，并清理覆盖，不能依靠背景轮询次数偶然切换。
- [ ] 为 shell、匿名登录、注册、bootstrap-error、Pending、admin-forbidden 建立首批 fixture；状态响应由当前 service DTO 和 v22 共用 data 核对。对主题行为先增加失败测试，再改入口、侧栏和表格布局。

```ts
// ui/e2e/theme-refactor.spec.ts：主题行为的首个场景
import {test, expect} from '@playwright/test';
import {openThemeCase, assertThemeLedger, assertThemeLayout} from './theme-refactor/routes';

test('theme:core 系统浅色和旧偏好不能改变深色登录', async ({page}) => {
    await page.emulateMedia({colorScheme: 'light'});
    await page.addInitScript(() => {
        localStorage.setItem('athena.member.preferences', JSON.stringify({version: 6, theme: 'light'}));
    });
    const ledger = await openThemeCase(page, 'member-login');
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(6, 8, 11)');
    await page.emulateMedia({colorScheme: 'dark'});
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    await assertThemeLayout(page);
    assertThemeLedger(ledger);
});
```

- [ ] `openThemeCase` 在导航前安装拦截，使用 `ATHENA_UI_E2E_PATH_PREFIX`，等待对应 h1 和字体就绪。`assertThemeLayout` 检查根横向溢出、单个可见 h1、页头／操作区遮挡及可见主控件目标；不要只检查 `body` 的 scrollWidth 或隐藏桌面副本。
- [ ] 沿 v3／v15 移动账户入口至顶栏，保留独立身份菜单；桌面 224px，900px 及以下导航抽屉。保留当前 focus trap、Escape、skip link 和关闭后焦点恢复，禁止用隐藏菜单保留可聚焦子项。
- [ ] 调整 AppPage／Section 和表格数据层级；复用 `compactRender`，将手机卡片改为批准的分隔行。统一按钮、输入、反馈、菜单及现有弹窗 class，保持提交中的关闭规则和每页当前分页合同。
- [ ] 按[共用状态表](../../requirements/web-ui/theme-consistency-contract.md#共用状态表)统一导航／页签／分段／数字页码；只读与禁用分别保留原生语义、外观和复制能力。管理员壳固定模块→图标映射；可比较的金额列及其表头右对齐，保留原始精度。统一区块标题 1.25rem／600、导航组标题 0.8125rem／400 与随字体放大的顶栏头像。

```css
/* shared.css 中替换现有布局声明；不追加第二份旧主题。 */
html, body, #app { min-width: 0; }
.app-page { min-width: 0; }
.app-page__heading h1 { font-size: 1.75rem; line-height: 1.3; font-weight: 600; }
.section-panel__header h2 { font-size: 1.25rem; line-height: 1.4; font-weight: 600; }
.athena-number { font-variant-numeric: tabular-nums lining-nums; }
.athena-identifier { font-family: var(--athena-data-font); font-size: .8125rem; font-variant-ligatures: none; overflow-wrap: anywhere; }
@media (max-width: 900px) {
    .app-page__heading h1 { font-size: 1.5rem; line-height: 1.35; }
    .app-page__header { flex-wrap: wrap; }
    .app-page__actions { max-width: 100%; }
}
@media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { animation-duration: .01ms !important; transition-duration: .01ms !important; scroll-behavior: auto !important; }
}
```

- [ ] 在原 `app.test.tsx`、`admin/app.test.tsx` 运行登录分流、权限、撤权和菜单用例；运行 runner 测试与 `UI_ACCEPTANCE_GREP='theme:core' make ui-acceptance`。后者是本任务新增过滤能力完成后可用的命令，报告必须说明局部范围。

## T3：登录、共享注册、账户自助、连接设置与 Help

**文件：** `member/pages/{login,register,account-security,notifications}.tsx`、`admin/login.tsx`、`shared/pages/{account-center,help}.tsx`、共享／各 realm CSS；`e2e/theme-refactor/fixtures/identity.json`、`cases.ts`、主 spec。

**消费：** T2 的布局、表格、弹窗与场景工具；当前注册、profile、密钥和绑定 service。

**产出：** 覆盖清单 T3 的 11 条入口及共享注册管理员差异，S2／S3／S6／S7。

- [ ] 先建立 member-login、admin-login、member-register、admin-register、member-profile、admin-profile、member-security、member-access、admin-access、member-notifications、member-help、admin-help 场景；分别记录自己的 realm。用 v10–v14／v22 数据核对既有 API 字段，不复用相同 UUID 冒充两种身份。
- [ ] 按 v14／v22 重排登录注册；按 v11／v13／v22 重排 Profile／Access；按 v12 分隔 Connect AI 与 API keys；按 v10 分隔当前绑定与替换表单。保留票据过期、用户名不可变、头像约束、草稿和 revision、秘密一次展示及 single-flight。
- [ ] 依 v22 Help 显示真实资源，有配置才显示下载／聊天；将链接统一由现有 deployment-base 帮助函数产生。管理员没有会员 Security 入口；会员入口由 API key 许可决定。

```tsx
// 共享 Help 的资源行布局使用真实配置／链接结果作为 children。
// 保留当前 HelpPage 的数据判断，只替换承载 DOM 的 class。
<section className='athena-help-resources' aria-label='Help resources'>
    <div className='athena-help-resource'>
        <a href={deploymentPath('llms.txt')}>AI integration guide</a>
    </div>
</section>
```

`deploymentPath` 使用当前 `shared/runtime-base.ts` 导出；资源的原有文字／条件以 v22 和当前 Help 为准，不添加文档外的客服承诺。

- [ ] 在主 spec 增加 `theme:identity` 用例：资料冲突保留草稿、Clipboard 拒绝后全文可选、密钥弹窗 pending 不重复提交、旧 Connected 在替换失败后仍可见。使用 ledger 检查实际发出的 body／次数，避免只断言 toast。
- [ ] 运行当前 `member/pages/notifications.test.tsx`、`shared/models.test.ts`、`shared/account-access-cache.test.tsx` 和相关 app 测试；执行 `UI_ACCEPTANCE_GREP='theme:identity' make ui-acceptance`。桌面／手机、Help 两种配置、Google ticket 失败及 Phantom 缺失均有记录。

## T4：管理员账户与运维五页

**文件：** `admin/pages/{admin-accounts,service-status,etherscan-gateways,system-notifications,system-notification-detail}.tsx`、`styles/admin-features.css`；`e2e/theme-refactor/fixtures/admin-operations.json`、cases 和主 spec。

**消费：** T2 中立组件；既有 admin read scope、账户权限和通知接口。

**产出：** v15–v18 的五条入口，S3／S8 和管理员辅助状态。

- [ ] 为账户目录／详情／编辑冲突、Service Status 三页签、Gateway／Live Probe、通知列表／详情／测试弹窗建立数据与请求用例；明确管理员操作与会员 API key 不能互换。
- [ ] Service Status 表格沿 v23 按容器宽度／44rem 阈值转分组行。增加 720px＋200% 根字号下 `Unreachable` 与完整超时说明不交叠的断言，同时验证正常桌面表格；不能只检查整页 scrollWidth。
- [ ] 按 v15 桌面目录／详情、手机目录到详情；v16 来源分区；v17 网关与 Probe 独立；v18 通知列表／详情重排。移动详情要保留返回列表与原筛选。

```css
.system-notification-card,
.etherscan-probe-sample {
    background: transparent;
    border: 0;
    border-bottom: 1px solid var(--athena-border);
    border-radius: 0;
    padding-block: 1rem;
}
.etherscan-probe-kpi__value { font-variant-numeric: tabular-nums lining-nums; }
```

- [ ] 在 `theme:admin` 中验证权限 aggregate 保留隐藏 Token 字段、role 重验期间清屏、失败可重试；三来源时间不混用；测试发送 pending 不能关闭且不重复发出，失败保留弹窗。Probe 的业务操作完全依当前 permission／service，数据布局不触发测试请求。
- [ ] 运行已有 `admin/pages/admin-accounts.test.tsx`、`service-status.test.tsx`、`system-notification-detail.test.tsx`、`admin/read-scope.test.tsx` 和 `admin/notification-service.test.tsx`；执行 `UI_ACCEPTANCE_GREP='theme:admin' make ui-acceptance`，保存五页桌面／手机及关键弹窗证据。

## T5：Wallets、Solana 与两端 Profit Sharing

**文件：** `member/pages/{wallets,solana,profit-sharing}.tsx`、`admin/pages/profit-sharing-admin.tsx`、`shared/pages/profit-sharing-shared.tsx`、各 realm features CSS；`e2e/theme-refactor/fixtures/{wallets,solana,governance}.json`、cases 和主 spec。

**消费：** T2 ResourceTable／弹窗；当前钱包私有数据、Solana 只读列表和治理阶段接口。

**产出：** v20 四页与 v19 管理员治理两页，S2／S9。

- [ ] 按批准数据建立 Wallets 列表／详情／导入／授权失效，Solana 候选／来源／加载错误，治理两端四阶段 fixture。只对 mock 声明的写入返回结果，真实钱包或提案写入不属于截图采集。
- [ ] Wallets 保留完整身份及私有操作，Solana 将候选事实、采集状态与来源分层；会员治理依阶段展示任务，管理员保留 lifecycle／roster；用现有 `compactRender` 承载手机分隔行。

```css
.profit-sharing-round-card {
    box-shadow: none;
    border-radius: 0;
    border-inline: 0;
    border-top: 0;
    border-bottom: 1px solid var(--athena-border);
}
@media (max-width: 900px) {
    .profit-sharing-editor-row { grid-template-columns: minmax(0, 1fr); }
    .profit-sharing-definition__footer { flex-direction: column; align-items: stretch; }
}
```

- [ ] `theme:foundations` 检查完整地址复制、敏感操作失败恢复、Solana 未知值，exact-five 与 0–5 draft roster、封存／匿名、权限撤销和草稿。分页与资金精度不跟随样板虚构数据改变。
- [ ] 运行现有 `member/pages/solana.test.tsx` 及账户／请求隔离测试；执行 `UI_ACCEPTANCE_GREP='theme:foundations' make ui-acceptance`。若展示拆分改变提交或删除行为，先补对应失效用例再实现。

## T6：Market Radar、Sports 与 Managed OO 八页

**文件：** `member/pages/{market-radar,sports-live,sports-history,world-cup-corners,managed-oo}.tsx`、`styles/member-features.css`；新建同目录 `market-radar-presentation.tsx` 与 `market-radar-presentation.test.tsx`；`e2e/theme-refactor/fixtures/{markets,sports,managed-oo}.json`、cases 和主 spec。

**消费：** T2 组件；现有 `MarketRadarHotMarketItem`、`MarketRadarRealtimeMarketItem`、`MarketRadarMoverMarketItem` 等 service DTO。

**产出：** v21 八页及 S12。新增三个展示函数分别接受其 DTO，不能继续用 `any` 把三页渲染成相同列。

```tsx
// market-radar-presentation.tsx：字段处理不将已知零回退为另一金额。
import * as React from 'react';
export const MarketVolume = ({value}: {value: number | undefined}) =>
    <span className='athena-number'>{value === undefined ? 'Unavailable' : value.toLocaleString('en-US')}</span>;
```

- [ ] 写并运行零与缺失的失败测试：`render(<MarketVolume value={0} />)` 应显示 `0`；`undefined` 应显示 `Unavailable`。再实现上面的显示组件，替换 `volume24hr || volumeNum`；Volume 24h 和 Volume total 各取各自字段。
- [ ] 实现 `HotMarketRecord({item}: {item: MarketRadarHotMarketItem})`、`RealtimeMarketRecord({item}: {item: MarketRadarRealtimeMarketItem})`、`MoverMarketRecord({item}: {item: MarketRadarMoverMarketItem})`，分别输出批准的活动指标、价格／百分点窗口、leader／direction／score。列表分页与刷新仍在现有页面；表格列按同一 DTO 字段提供。
- [ ] Sports 两页沿 v21 保留比分、状态、价格图和来源的区别，World Cup 保留口径与样本；Managed OO 保留 proposal／dispute 角色、完整地址和原始 proposed price。图表读取业务时间，不能用数组序号或假曲线代替。
- [ ] 在 `theme:markets` fixture 中覆盖 known-zero、missing、stale、window-warmup、record-long、empty、failed；校验可见字段和原值，桌面／手机由同一模型呈现。覆盖 Sources 展开与区块解析的 pending／失败，禁止真实扫描。
- [ ] 运行展示单元测试及 `UI_ACCEPTANCE_GREP='theme:markets' make ui-acceptance`。保存八页主图，核对数据显示口径符合 v21；处理第三方/内嵌图表外观时先确认它实际受哪个 CSS 或绘图代码控制。

## T7：Worm Assets、组合列表和组合编辑器

**文件：** `member/pages/{worm-trading,worm-trading-combinations}.tsx`、相关 Worm components／styles、`styles/member-features.css`；`e2e/theme-refactor/fixtures/worm-assets-combinations.json`、cases 和主 spec。

**消费：** T2 组件、当前 Wallet／Worm service 与状态管理。

**产出：** v22 三个实际页面／四条路由，S10 和组合编辑部分 S11。

- [ ] 为只读资产、部分失败、无连接、0／20 钱包、单仓 Cash Out、批次冻结／暂停／未知建立 mock；组合覆盖新增／编辑、草稿、失效选择和移除恢复。采用 v22 完整身份与数据，API fixture 依当前 service 请求形状。
- [ ] Assets 按已选钱包、独立状态、持仓／请求分区；组合桌面目录和有序选择并列，手机顺序阅读。复用现有回调和防护；原型的恢复保存图用于验证移除无效选择后的真实 React 状态。

```css
.worm-trading-activity-card {
    background: transparent;
    border: 0;
    border-bottom: 1px solid var(--athena-border);
    border-radius: 0;
    box-shadow: none;
}
.worm-wallet-selection-modal__body { min-height: 0; overflow-y: auto; }
.worm-wallet-selection-modal__footer { flex-shrink: 0; }
```

- [ ] `theme:worm-assets` 检查 readonly 不发写请求、选择上限／移除保护、五分钟授权、完整批次顺序、Cash Out 授权即排队、严格 USDC 增加才继续。未知响应不再次发出同一操作；操作后果文字与状态不简化。
- [ ] `theme:worm-combinations` 检查每市场单方向、顺序、名称 Unicode／控制字符、离开草稿、失效选择移除后恢复 Save。若当前代码违反已经确认的业务规则，保留失败证据并按行为修复流程处理，不能只改颜色掩盖。
- [ ] 执行 `UI_ACCEPTANCE_GREP='theme:worm-assets|theme:worm-combinations' make ui-acceptance`；检查 390px 弹窗可滚动正文、固定操作和焦点，保留根字号 200% 的头像及长标识检查。

## T8：Worm 执行预览、执行记录与详情

**文件：** `member/pages/{worm-trading-execution-preview,worm-trading-executions}.tsx`、相关执行 components／styles、`styles/member-features.css`；`e2e/theme-refactor/fixtures/worm-executions.json`、cases 和主 spec。

**消费：** T7 组合与钱包展示、T2 组件、现有 plan／Run 状态机。

**产出：** v22 三条执行入口，完整 S11。

- [ ] 建立准备中／失败／过期预览、部分跳过、有效 partial-fill、独立授权、ready-to-start、running／pausing／terminating、unknown 和 completed fixture。钱包选择与 plan／Run revision 均从同一数据场景获取。
- [ ] 按 Combination → Wallets → Checks → Review 重排预览；详情先冻结意图与允许动作，再步骤账本及证据。Prepare 只冻结，授权与 Start 仍各自显式；不会将静态说明弹窗变成新增正式流程。

```css
.worm-execution-detail { min-width: 0; }
.worm-execution-step-card,
.worm-execution-current-card { box-shadow: none; }
.worm-execution-stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 11rem), 1fr)); }
.worm-execution-sticky-actions { padding-bottom: max(1rem, env(safe-area-inset-bottom)); }
```

- [ ] 在 `theme:worm-executions` 检查空／非 CONNECTED 钱包不能隐式执行、执行子集 1–20、变更或过期阻止 Prepare、BUILDING／FAILED 不显示确定总额。单飞、暂停／终止等待当前步骤收敛和 unknown 只读核对要通过 ledger 验证。
- [ ] 核对完成观察、step Last update、Run updated 的来源字段与前后顺序；不存在旧 plan 时不得生成错误的共享预览链接。保留技术标识全文、证据展开和 focus；底部操作栏不遮挡最后一条步骤。
- [ ] 执行 `UI_ACCEPTANCE_GREP='theme:worm-executions' make ui-acceptance`，保存三页主图与未知／授权／终止关键状态。不得通过真实下单、签名或 Cash Out 制作视觉证据。

## T9：整体覆盖、无障碍及真实环境验收

**文件：** 完成主 spec／a11y spec 和 cases；调整现有 Trader Sync fixture／测试的主题与 preferences 预期；新建验收报告；按发现的实际问题修正对应生产文件。

**消费：** T1–T8 全部实现、覆盖清单 S1–S14。

**产出：** 有范围与模式标识的实际通过记录，包含暂缓页共享影响和新 schema 的真实环境。

- [ ] 对机器清单逐条检查：37 条目标入口都有案例，2 条 Appearance 转为既有未找到，4 条默认／兜底按角色生效；共享注册 admin 变体单列。8 条 Trader Sync 只验共享主题影响与行为，不补写成布局已改版。Token 导航保持当前独立接入状态。
- [ ] 现有深浅主题循环改为“系统输入 light／dark 时结果均 dark”，保留两个系统环境输入；只更新批准的主题／preferences 预期，不删除 owner、grant、cursor、撤权与未知状态用例。
- [ ] 每个改版页面采集 1440×900、390×844 主图及 320×844、200% 字号证据。等待字体、关闭动画、移开指针、从页首截图；弹窗使用真实视口并记录内部滚动。与相应批准图逐页对应，不能把原型 DOM 的像素差当成产品回归门槛。
- [ ] 放大检查先断言实际字体增加，再检查单元格、错误说明、头像、输入与固定操作区的局部内容边界。覆盖 Button／Input／Select／Table／Modal／Dropdown／Message／Notification 的实际消费；修订 CSS 若遗漏组件专用字号变量，补入公共桥接并重验。根字号模拟和原生浏览器缩放分开记录。
- [ ] a11y spec 使用现有 `AxeBuilder` 与 WCAG 2／2.1 A／AA tags，保存原始 JSON。覆盖默认页、关键弹窗、失败／只读、焦点和主次按钮 normal／hover／active／disabled；裁字、手机键盘和中文回退另用实际浏览器核对。
- [ ] 在 `ui` 执行 `yarn lint`、`yarn build` 与受影响 Jest 汇总；根目录执行无过滤 `make ui-acceptance` 和 `make ui-a11y`。不得携带 `UI_ACCEPTANCE_GREP`，报告必须含实际匹配／通过／失败／跳过数量。
- [ ] 按[本地说明](../../developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance)核对目标 worktree 与实例。新 schema 使用任务专用数据库，先完成显式 schema 准备；`make run` 启动持久会话并记录日志。现有数据库若校验失败，不 reset、不删除表来强行复用；按方案中的数据库边界处理。
- [ ] 执行 `make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://localhost:4000` 前核对该地址确属任务实例；端口不同时使用已验证实例实际地址。检查会员／管理员 bootstrap、登录／注册、字体／chunk、导航、Help 资源和代表业务页的真实接入，分别记录成功、外部阻塞和未验证项。shell smoke 通过不等于所有业务页真实接入通过。
- [ ] 集中处理发现的问题并仅重验受影响范围；完成后检查临时环境所有权、执行同实例停止命令并验证退出。保留数据库、卷、截图和日志；用户及其他任务的环境不停止。

## T10：长期文档、审阅与交付收尾

**文件：** `docs/testing/web-ui-theme-refactor-acceptance.md`、本方案／计划／覆盖清单、`docs/requirements/web-ui/visual-theme.md`、`docs/design/web-ui/{application-shell,member-application-shell,administrator-application-shell}.md`、`docs/design/identity-access/account-profile-and-preferences.md`、两个索引、`AGENTS.md`、`PRODUCT.md`。

**消费：** T9 实际报告与相关代码差异。

**产出：** 能清楚区分“设计批准”“正式实现”“实际验收”的交付记录。

- [ ] 按最终代码更新长期主题、入口、偏好及生成契约说明；删除当前设计中已失效的双主题／跨设备 theme 功能陈述，历史批准资产保留原字节。执行期间产生的新业务差异只记录有证据且已授权的部分。
- [ ] 对源代码和生成物做独立代码审阅，重点检查 API 清理是否完整、realm 边界、原值精度、陈旧结果和破坏性操作门槛。仅在用户授权整合的范围内提交／发布。
- [ ] 运行 `git diff --check`，检查文档本地链接、每条路由和状态的报告归属、全部原始批准资产 SHA256。核对全量检查和真实环境未完成项，不能将有阻塞的整体任务记为完成。
- [ ] 交付列出实际改造页面、暂缓范围、检查结果及停止／保留环境。只有用户确认的实施计划全部完成且验收通过，才从仓库根执行一次 `make notify-task-complete`；当前仅编写计划，不发送通知。

## 计划自查记录

本计划的 T1–T10 覆盖技术方案各部分和覆盖矩阵 S1–S14。新的过滤命令明确由 T2 增加，当前尚不存在；所有现有命令依据 Makefile、package.json 和本地运行说明。API／数据库破坏性清理与其他偏好保留有明确边界，视觉批准没有扩大到真实身份或交易。

本次规划没有启动浏览器或服务，没有修改正式 `ui/`、后端或数据库。执行时应重新核对当前分支和依赖；本计划中的复选框和预期结果不构成实施证据。
