# 两个 BSC 索引器与 Sports 删除清理实施计划

> **供执行代理：**使用 [executing-plans](../../../.agents/skills/executing-plans/SKILL.md) 逐项执行并复核；需要子代理时按用户选择使用 [subagent-driven-development](../../../.agents/skills/subagent-driven-development/SKILL.md)。用下方复选框记录实际结果。

**目标：**删除两个 BSC 索引器、Sports Live／History、World Cup Corners 的产品能力及专属运行资源，直接删除其历史数据，同时保留 Worm、核心服务和其他业务。

**架构：**直接移除废弃命令、接口、页面、权限和配置；账户权限使用一次追加迁移清理，Sports 通知由 Notification 存储的一次性维护工具处理。部署退役使用逐环境的精确资源清单，不增加常驻清理服务、兼容入口或后台控制协议。

**技术栈：**Go、PostgreSQL／Goose／sqlc、gRPC／grpc-gateway、React／TypeScript、Docker Compose、现有本地运行器和 Playwright 验收工具。

**设计依据：**[删除与清理配套设计](../specs/2026-09-16-module-removal-cleanup-design.md)、[BSC 索引器删除需求](../../requirements/blockchain-data/bsc-indexer-removal.md)、[Sports 删除与 Worm 保留](../../requirements/development-runtime/sports-removal.md)。执行者必须先读设计再读本计划。

**状态：**2026-09-16 按用户“优先规划清理和删除工作”编写。仅完成计划与静态核对；下方实施、部署、删库和运行验收均未执行。

**执行准备结论：**任务范围、依赖、源码落点、验证和失败处理已明确，可从 T1 开始实施。T2 的维护工具、T6 的现场手册属于明确的实施产物，当前不能当作已存在命令调用；T7 的真实环境盘点是 T8 删除的前置步骤，不需要再开展一轮业务设计。计划可以开始执行不代表远端资源和数据库连接已经核实。

## 全局约束

- 删除对象固定为四个独立程序和一个 API 内嵌业务；本期不实施访问开关，也不将十二应用全栈作为清理交付前提。
- `worm_markets`、`worm_trading`、账户、钱包、私钥、头像、资产、Notification 共享账本和其他业务数据保留。Token 旧代码和数据不删除；其运行接入调整属于后续计划。
- 四个目标历史库 `bsc_inbound`、`bsc_swap`、`sports_live`、`sports_history` 直接删除，**不导出备份或建立历史归档**。数据库覆盖名、容器及卷必须按实际 owner 核对。
- “核心不提供管理员停服开关”不取消已有不兼容 schema 的发布维护流程；同库消费者退出后执行 `up` → `verify` → 新版本启动。
- 六个 Sports 通知来源精确匹配；只取消 `pending`，在途 `sending` 按真实结果收尾，保留 `sent`／`failed`／`unknown` 和发送尝试。不清空共享队列。
- 正常 `make run`／`make stop` 不删数据；不得用 `run-reset`、全机 Docker prune、删除共享整卷或按端口杀进程完成本计划。经核对确属索引器独占的卷按 T8 删除。
- 两台已记录索引器主机同时承载 Gateway；索引器退役不关闭 Gateway、删除主机或清理共享镜像／网络。
- 沿用 [sync-athena-changes](../../../.codex/skills/sync-athena-changes/SKILL.md)：每批 SQL 源稳定后运行一次 sqlc；proto／共享类型从源重新生成，不手改生成文件。
- 执行前按 [using-git-worktrees](../../../.agents/skills/using-git-worktrees/SKILL.md) 准备隔离工作区，携带本任务必需的已确认文档；保留 `rf4` 上其他未提交工作，不进行全仓暂存、reset 或覆盖。
- 采用项目 Go／Node 版本及既有主题；独立测试、实际本地验收、远端退役分别记录。测试不向真实供应商下单、平仓或发送测试通知；真实只读调用与隔离副作用测试分开记证据。
- 服务规范适用 SDS-R2／R3／R4／R5／R6／R7／R8：保留 API 与业务边界、有限超时、事务归属和精确资源回收，不因清理新增耦合。

## 交付顺序与范围

| 批次 | 任务 | 交付结果 | 进入下一批的依据 |
| --- | --- | --- | --- |
| A 仓库清理 | T1–T5 | 新版本源码、迁移、生成产物、通知维护工具及部署配置 | 对应包、迁移、UI 和配置检查通过 |
| B 本地验收 | T6 | 空库／已有库升级证据、真实核心与 Worm 保留回归、现场操作手册 | 保留能力验证通过，形成可部署版本 |
| C 现场退役 | T7–T8 | 逐环境停止生产者、处理旧通知、删除四库及专属资源 | 对象归属与新版本验证明确，逐项有结果 |
| D 收尾 | T9 | 仓库／进程／数据三个层面的最终记录 | 已完成项有实证，失败与残留单列 |

T2 可独立完成工具测试；T3、T4、T5 必须组合成一致版本后发布，不能只上线删除后的前端或只收紧数据库约束。现有权限检查仍保护保留业务；本期不存在尚未实施的 Worm 访问开关前置依赖。

## 文件与职责地图

| 范围 | 主要路径与处理 |
| --- | --- |
| 四个程序 | 删除 `cmd/athena-bsc-transaction-indexer/`、`cmd/athena-bsc-swap-indexer/`、`cmd/athena-sports-live/`、`cmd/athena-sports-history/` |
| 四个实现 | 删除 `internal/bscinbound/`、`internal/bscswap/`、`internal/sportslive/`、`internal/sportshistory/` 及各自专属 proto／存储／生成产物 |
| 公共接口 | 删除 `internal/server/sportslive/`、`internal/server/sportshistory/`、`internal/server/worldcupcorners/`；同步 `internal/server/athena-server.go`、`authz.go`、`cmd/athena-server/commands/athena-server.go`、`cmd/main.go`、`common/common.go` |
| 权限与 schema | `internal/accountaccess/access.go`、`internal/server/account/account.proto`、`account.go`、`internal/accountstate/store/{queries,sqlc,migrations,sql_store.go}`、`internal/accountstate/schema/contract.json` |
| 共享生成 | `pkg/apis/application/v1alpha1/market_intelligence_types.go`、`sqlc.yaml`、`hack/generate-proto.sh`、`assets/swagger.json` 及实际生成消费者；保留 Worm 类型和后处理 |
| 页面 | 删除三页、`sports-market-card.tsx` 及专属用例、三个请求服务、`sports-models.ts`；同步 `ui/src/app/member/{app.tsx,routes.tsx,services.ts}`、`ui/src/app/shared/{access-modules.ts,services/registry.ts}` 及真正受影响的样式／测试 |
| 通知维护（新增） | `internal/notification/store/retired_sports.go`、`queries/retired_sports.sql`、`retired_sports_integration_test.go`；`tools/retire-sports-notifications/{main.go,main_test.go}` |
| 运行与部署 | `internal/migration/modules.go`、`internal/devruntime/{registry.go,fullstack.go,environment.go}`、`Makefile`、`docker-compose.prod.yml`、`hack/postgres/init/00-databases.sql`、`hack/prod-remote-deploy.sh` 及相关测试 |
| 专属部署 | 删除两个索引器的 `deploy/` 子目录和 `hack/deploy-bsc-*-indexer.sh` 精确两文件；混合 `.env`／`.env.prod`／模板只清理专属配置 |
| 记录（新增） | `docs/operator-manual/module-removal-retirement.md` 操作手册；`docs/testing/module-removal-cleanup-acceptance.md` 验收与逐环境结果 |

新增文件在本计划中是目标路径；实施后才能当作已存在工具调用。2026-09-16 源码的账户迁移最高版本为 `000003`，T3 使用 `000004_remove_sports_access.sql`；若执行前已有其他迁移占用该编号，则按实际下一个空号创建并同步本计划与测试，不覆盖其他迁移。

### T1：建立精确清单与保留基线

**文件：**新增 `docs/testing/module-removal-cleanup-acceptance.md`；运行证据放 `.superpowers/module-removal/`。本任务不改业务行为，不新增测试代码。

**输入：**已确认删除设计、实施工作区源码及已有部署记录。

**输出：**`source-inventory.txt`、`preserved-inventory.txt`、基线版本和工作区状态；验收记录分列代码、运行、数据状态。

- [x] 记录工作区路径、分支、HEAD、未提交文件和实际工具版本；检查 `rg`、Go、Python、Docker、Node／Yarn、psql 及生成工具。读取 `ui/.nvmrc`，实施和测试按该版本选 Node。
- [x] 用精确入口和引用建立清单，追踪共享文件的真正消费者，不按 `sports`、`BSC` 或 `Polymarket` 词根整包删除：

```bash
rg -n 'sportslive|sportshistory|worldcupcorners|SportsLive|SportsHistory|WorldCupCorners|bscinbound|bscswap' cmd internal common pkg ui/src hack deploy Makefile sqlc.yaml docker-compose.prod.yml
rg -n 'sports-models|sports-market-card' ui/src ui/e2e
rg -n 'worm-markets|worm-trading|worm_markets|worm_trading|WormExecutionSigner' cmd internal ui/src docker-compose.prod.yml
```

- [x] 明确允许保留旧名称的位置：已应用历史迁移、退役维护工具精确来源清单、删除回归断言、带退役标记的历史设计／证据；它们不是活动功能入口。`util/worm` 的体育市场和相关 `sports` 数据语义保留。
- [x] 记录已有六项本地应用的实际清单及 Worm 当前运行方式；新十二应用编排尚未实施，后续验收不伪报该目标已达成。计划中的远端 IP 和默认卷只作线索，此时不认定资源存在。
- [x] 先保存旧部署文件的定位信息和停止方法，再在 T3／T5 删除源码；只保留操作证据，不保留可自动拉起旧业务的部署副本。

### T2：实现一次性 Sports 通知退役工具

**文件：**新增地图中的四个维护文件和 SQL；生成 `internal/notification/store/sqlc/`；测试复用 `attemptFixture`、`testPermitCandidate`、`deliveryState`（位于现有 `attempts_integration_test.go`）。

**输入：**本环境账户状态 DSN、已验证 schema、已停止全部 Sports 生产者的现场事实。

**输出接口：**在 `store` 包定义如下方法；工具不调用 Telegram、不取得 sender 身份、不修改普通 Notification 启动路径。

```go
type RetiredSportsCounts struct {
    Pending int64 `json:"pending"`
    Sending int64 `json:"sending"`
}

func (s *SQLStore) CountRetiredSports(ctx context.Context) (RetiredSportsCounts, error)
func (s *SQLStore) CancelRetiredSportsPending(ctx context.Context, limit int32) (int64, error)
```

- [x] 先增加数据库集成用例。以下核心断言放在 `store` 包，沿用既有测试 fixture；新增接口未实现时应编译失败，随后转为行为断言通过：

```go
func TestRetiredSportsPendingIsCancelledWithoutDeletingLedger(t *testing.T) {
    s, ref := attemptFixture(t, "system")
    ctx := context.Background()
    _, err := s.pool.Exec(ctx, `UPDATE system_notification_deliveries
        SET source='polymarket.sports-live-score' WHERE id=$1`, ref.ID)
    require.NoError(t, err)
    n, err := s.CancelRetiredSportsPending(ctx, 100)
    require.NoError(t, err)
    require.EqualValues(t, 1, n)
    require.Equal(t, "cancelled", deliveryState(t, s, ref))
    n, err = s.CancelRetiredSportsPending(ctx, 100)
    require.NoError(t, err)
    require.Zero(t, n)
}
```

- [x] 增加同一行许可竞争用例：通过现有 `s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)` 得到 `sending`，清理返回零且仍为 `sending`；再用现有结果记录 API 分别写入 retryable／sent／unknown，核对只有回到 pending 的可被下一次清理取消。另建 Worm、Trader Sync 和近似 Sports 来源，确认均未改变；验证六个精确来源、行锁超时及分批处理。
- [x] 新 SQL 写入六个固定来源，不接受任意前缀或调用方传入来源。取消使用与发送许可相同的 delivery 行锁／状态条件；每批 100 条，事务内设置有限锁与 SQL 超时。核心查询如下，计数查询复用相同六来源集合：

```sql
-- name: CancelRetiredSportsPending :execrows
WITH chosen AS (
  SELECT id FROM system_notification_deliveries
  WHERE status = 'pending' AND source IN (
    'polymarket.sports-live-score',
    'polymarket.sports-live-price-alert-85-15',
    'polymarket.sports-live-price-alert-90-10',
    'polymarket.sports-live-price-alert-95-5',
    'polymarket.sports-live-price-alert-97-3',
    'polymarket.sports-live-price-alert-99-1'
  )
  ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED
)
UPDATE system_notification_deliveries d
SET status='cancelled', error_message='source retired: sports removal',
    locked_at=NULL, locked_by=NULL
FROM chosen c WHERE d.id=c.id AND d.status='pending';
```

- [x] SQL 源稳定后执行 `make sqlc-local`，检查生成差异，再实现存储方法：借用现有 pool；写操作自建并提交事务，`SET LOCAL lock_timeout='2s'`、`statement_timeout='5s'`；计数失败不能当作零，保留发送尝试及终态数据。
- [x] 工具使用 `schema.LoadDSN` 和 `schema.ConnectVerified` 连接、明确打印数据库身份和只读计数；默认只读，指定 `--apply` 才循环取消并复查。`--timeout` 默认 `5m`，每秒复查，单次数据库操作最多 5 秒；超时、锁失败或非零 pending／sending 输出剩余量并退出非零。持有该数据库一条专用连接的 session advisory lock `athena:retire-sports-notifications`，获取失败立即退出，进程结束释放。
- [x] 为工具增加默认只读、参数校验、超时／取消、重复运行和并发运行测试。JSON 结果包含数据库名、取消数、pending、sending、结果状态；日志不输出 DSN 密码。只读计数不证明生产者已退出，T8 必须单独核验。

**运行与预期：**使用明确的隔离测试 PostgreSQL 设置 `ATHENA_TEST_PG_ADMIN_DSN`，fixture 只操作自建库；不能省略环境后把测试未运行当通过。

```bash
go test -tags=integration ./internal/notification/store -run 'TestRetiredSports' -count=1
go test ./tools/retire-sports-notifications -count=1
go build -o .superpowers/module-removal/retire-sports-notifications ./tools/retire-sports-notifications
```

产物随退役批次使用，不注册到 `make run`、生产常驻服务或管理员 API。通过后提交本任务相关文件，保留测试输出。

### T3：删除后端能力并清理账户权限契约

**文件：**删除地图中的四个命令、四个实现和三个公共服务目录；修改账户、API 注册、模块迁移注册、命令分发、共享类型和生成配置。新增 `internal/accountstate/store/migrations/000004_remove_sports_access.sql`、`internal/accountstate/store/sports_removal_integration_test.go`；修改现有权限／API／schema 测试。

**输入：**T1 精确清单；T2 的通知来源与维护能力已独立于待删 Sports 包。

**输出：**无废弃业务注册、账户权限只保留八类模块、新旧库统一 schema 和可独立验证的迁移命令。

- [ ] 先在 `internal/accountaccess/access_test.go` 增加真实权限行为断言，预期当前代码仍识别旧模块而失败：

```go
func TestRetiredModulesAreRejected(t *testing.T) {
    for _, old := range []Module{"sports_live", "sports_history", "world_cup_corners"} {
        if _, ok := MaxAccessLevel(old); ok { t.Fatalf("retired module accepted: %s", old) }
    }
    if got := len(AllModules()); got != 8 { t.Fatalf("module count=%d", got) }
    if level, ok := MaxAccessLevel(ModuleWormMarkets); !ok || level != AccessLevelRead { t.Fatal("Worm Markets permission changed") }
    if level, ok := MaxAccessLevel(ModuleWormTrading); !ok || level != AccessLevelReadWrite { t.Fatal("Worm Trading permission changed") }
}
```

- [ ] 集成测试同时覆盖空库和旧库：用 `pgtest.NewUnmigrated` 与只包含历史迁移的 `fstest.MapFS` 建立旧结构，写入只拥有 Sports 的账户及带 Worm／Wallet 权限的账户，运行全部迁移；断言三类行删除、保留权限与 login／API Key 标志不变、派生 Pending 正确。插入三种废弃模块应被约束拒绝；新旧最终 catalog 均通过 `schema.Verify`。保留已有版本集合检查，不删历史迁移记录。
- [ ] 追加迁移核心内容如下；使用 Goose 默认事务，失败完整回滚本次迁移。保留原访问级别及 Trader Sync 的额外约束：

```sql
-- +goose Up
DELETE FROM account_module_access
WHERE module IN ('sports_live','sports_history','world_cup_corners');
ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_module_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_module_check
CHECK (module IN ('market_radar','managed_oo','worm_markets','worm_trading',
                  'token','solana','wallet','trader_sync'));
ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_max_level_check;
ALTER TABLE account_module_access ADD CONSTRAINT account_module_access_max_level_check
CHECK (access_level <> 'read_write' OR module IN
       ('managed_oo','worm_trading','token','wallet','trader_sync'));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
  RAISE EXCEPTION 'sports removal is irreversible; use the validated current version';
END $$;
-- +goose StatementEnd
```

- [ ] 移除 `account_access.sql` 的三组更新参数、`account_directory.sql` 的三类初始权限及相关 Go 映射；清理 `sqlc.yaml` 的四个专属输入。完成整批 SQL 后运行 `make sqlc-local`，再调整 `sql_store.go` 的生成参数消费者，运行 `make account-state-schema-contract` 更新契约。
- [ ] 删除公共账号 proto 中编号 2、3、7 对应的三个值，声明 `reserved 2, 3, 7` 及其原名字；其余编号不变。清理 API 构造、关闭、权限映射、gateway、健康登记、命令参数及 `common` 专属端口；删除 World Cup Corners 内嵌静态业务数据。
- [ ] 删除模块代码与 `internal/migration/modules.go` 的 Sports 导入／注册，更新共享 `market_intelligence_types.go` 时只删无保留消费者的类型。生成脚本当前自动发现 `internal/**/*.proto`，删除源及专属旧产物即可取消发现；仅清理真正存在的特殊分支，保留 Worm Swagger 后处理。
- [ ] 执行 `make protogen`，检查 protobuf／gateway／Swagger／共享生成类型；若类型变更影响 deepcopy，按仓库既有生成入口补齐实际产物。不因本次删除运行合约 ABI 生成器；依赖仅在最后一个保留消费者确实消失时移除。
- [ ] API 测试核对已注册服务与真实 HTTP 路由：旧 gRPC 方法不再注册，旧 `/api/...` 地址走未知 API 结果而非返回旧业务；Wallet、Worm、账户和健康仍存在。纯权限拒绝不足以证明接口已删除。删除废弃包专属测试，保留混合测试中的其他业务断言。

**运行与预期：**

```bash
go test ./internal/accountaccess ./internal/server/... ./internal/migration ./cmd/athena-server/commands -count=1
go test -tags=integration ./internal/accountstate/... ./cmd/athena-account-state-migrate/commands -count=1
go build ./cmd/athena-server ./cmd/athena-account-state-migrate ./cmd/athena-notification ./cmd/athena-trader-sync ./cmd/athena-solana-discovery
```

预期八类权限、旧接口不存在、新旧库校验通过；没有废弃包 import 或手改生成产物。保持 SQL／proto 源与消费者同一提交范围，提交只包含本任务文件。

### T4：删除前端入口与专属展示

**文件：**按地图清理三页、服务、模型、卡片；同步 `ui/src/app/shared/access-modules.ts`、现有账户权限测试、管理员权限页测试及 `ui/e2e/theme-refactor/` 的混合覆盖清单。新增 `ui/e2e/module-removal.spec.ts`，修改 `ui/playwright.config.ts` 的 `ui-fixtures.testMatch`，把 `module-removal` 加入原正则；smoke 和 live 的证据分类不变。

**输入：**T3 稳定的八类账户权限和移除后的公共协议。

**输出：**菜单、权限编辑和路由不再提供三项业务，Worm 七条页面路由与全部现有操作保留。

- [ ] 先扩展账户权限测试：管理员提交的 `moduleAccess` 长度从 11 改为 8，包含原编号 5／11 的 Worm 权限及 9 的 Wallet；不包含 2／3／7。保留其余权限比较、编辑与读写上限断言。

```ts
// 在现有 AdminAccountsService().updateAccess(...) 调用之后检查真实提交结构。
expect(payload.moduleAccess).toHaveLength(8);
expect(payload.moduleAccess.map((item: {module: number}) => item.module).sort((a: number, b: number) => a - b))
    .toEqual([1, 4, 5, 8, 9, 11, 12, 13]);
expect(payload.moduleAccess.find((item: {module: number}) => item.module === 12))
    .toEqual({module: 12, dataAccess: 0});
```

- [ ] 新增浏览器删除回归：使用既有已登录 member／admin fixture，访问三个废弃详情入口，断言现有未找到页面且没有旧业务请求；管理员权限列表无三项，会员菜单无三项。测试覆盖部署根路径与 `/athena`，Worm 菜单和七路由保持原权限约束。

```ts
import {expect, test} from '@playwright/test';
import {openThemeCase} from './theme-refactor/routes';

for (const oldPath of ['/sports-live', '/sports-history', '/world-cup-corners']) {
    test(`removed module: ${oldPath}`, async ({page}) => {
        const oldRequests: string[] = [];
        page.on('request', request => {
            if (/\/api\/[^?]*(sports-live|sports-history|world-cup-corners)/.test(request.url())) {
                oldRequests.push(request.url());
            }
        });
        await openThemeCase(page, 'member-shell');
        const prefix = (process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '');
        await page.goto(`${prefix}${oldPath}`);
        await expect(page.getByText('Page not found', {exact: true})).toBeVisible();
        expect(oldRequests).toEqual([]);
    });
}
```

管理员权限和 Worm 正向覆盖扩展现有 `admin-accounts`、`worm-assets-combinations`、`worm-executions` 场景；修订 fixture 的八类权限与真实新协议一致。上述 UI fixture 不能证明真实后端旧路由已注销，T3 与 T6 的 API 检查仍需执行。

- [ ] 删除专属实现与请求服务；清理 `app.tsx`、`routes.tsx`、`services.ts`、共享注册和帮助中的调用。`sports-models.ts`／`sports-market-card.tsx` 当前只被 Sports 页和专属用例使用，核对后一并删除；共享组件与其他市场类型保留。
- [ ] 更新主题混合场景中 Sports 的路由和数据，保留 Market Radar、Managed OO、Worm 场景；不重写或抹去 v21 历史批准事实。样式规则只删除已无消费者的选择器。

**运行与预期（在 `ui/`）：**

```bash
yarn test --runInBand --runTestsByPath src/app/shared/account-access.test.ts src/app/shared/account-access-cache.test.tsx src/app/admin/pages/admin-accounts.test.tsx src/app/member/pages/worm-trading-scope.test.tsx src/app/member/pages/worm-execution-scope.test.tsx
yarn lint
yarn build
```

Playwright 真正执行与清理证据由 T6 统一完成；本任务单元检查通过不等于真实页面验收通过。通过后提交前端与相应测试。

### T5：清理启动、部署、配置及构建残留入口

**文件：**地图中的运行／部署文件及 `.env`／`.env.prod`／模板实际存在项；修改 `internal/devruntime/{fullstack_test.go,registry_test.go,environment_test.go}`、`internal/migration/modules_test.go`、`hack/production-compose_test.sh`。

**输入：**T3 删除后的程序与模块集合，T1 的保留配置清单。

**输出：**启动不要求废弃配置，不准备 Sports 库；新部署清单不会自动重建旧程序。永久删除仍仅由 T8 执行。

- [ ] 先给已有 `Modules()` 和 `fullStackModules()` 测试增加“不含 Sports、仍含两个 Worm”的断言；移除 Sports 配置后解析生产 Compose，断言服务／depends_on／migration 目标不存在，Worm、Wallet、Notification 保留。该测试要读取实际解析结果，不只搜索文本。
- [ ] 清理注册表的 API 地址白名单、`fullStackModules()` 的 Sports 两库、初始化 SQL 和迁移模块；普通运行器遇到历史实例记录时保留 owner 信息供停服，不删除状态文件掩盖遗留进程。旧 Token／Temporal 准备保持本期清理边界，交由后续运行计划调整。
- [ ] 移除 `docker-compose.prod.yml` 中 Sports 两服务及其依赖、专属环境项；移除两个索引器的 Make 变量／构建／镜像／部署目标、两份部署脚本和两目录。`.env` 和模板按键删除，不整文件覆盖，不更改保留凭据。
- [ ] 检查部署预检与 schema 服务列表，所有实际账户库使用者都进入不兼容 schema 的维护顺序，包括实际部署的 Solana；不照搬只含三个旧服务的列表。清理后的 API 不得因 Sports 地址／DSN／token 缺失而失败。
- [ ] 检查 `hack/prod-remote-deploy.sh` 及 Make 包装的副作用：现有 `prod-deploy-remote` 带 secrets 重置前置步骤，既有 cleanup 可能执行整 project `down --remove-orphans`。退役手册使用经核对的发布步骤与精确对象，不把这些包装命令当作安全的单模块删除入口，不为清理重置其他服务凭据。
- [ ] 本地 `dist`、镜像、临时部署包按实际归属整理：只删除已经不被任何保留进程使用的专属产物；共享 `PROD_IMAGE`、PostgreSQL 基础镜像及混合日志／缓存保留。现场旧文件等 T8 完成定位与停服后处理。

**运行与预期：**

```bash
go test ./internal/devruntime ./internal/migration ./cmd/athena-local-runtime -count=1
bash hack/production-compose_test.sh
./dist/shellcheck -x -P . hack/prod-remote-deploy.sh
```

仅在上述脚本有改动时对其运行 ShellCheck；独立测试不访问远端。预期新 Compose 不含 Sports，新启动不创建 Sports 库，Worm 与其他模块配置保持完整。通过后提交配置与测试。

### T6：完成仓库与本地真实验收，形成现场手册

**文件：**新增 `docs/operator-manual/module-removal-retirement.md`；更新 `docs/testing/module-removal-cleanup-acceptance.md`、相关活跃使用文档和源码链接。

**输入：**T2–T5 的一致版本及实际生成／测试日志。

**输出：**可部署版本、实际本地验收报告和 T7／T8 可执行的逐环境手册；不把此批次写成现场已经退役。

- [ ] 统一完成保留模块回归，尤其覆盖 Worm 钱包选择、连接、组合、预览、交易授权、执行／Cash Out 状态及 Notification 发送许可。不重复运行未改变输入的生成器：

```bash
go test ./internal/server/... ./internal/accountaccess/... ./internal/accountstate/... ./internal/notification/... ./internal/wallet/... ./internal/wormmarkets/... ./internal/wormtrading/... ./util/worm/... ./internal/devruntime/... ./internal/migration/... ./tools/retire-sports-notifications/... -count=1
go test -tags=integration ./internal/accountstate/... ./internal/notification/store -count=1
go build ./cmd/... ./tools/retire-sports-notifications
```

- [ ] 为集成测试按各包实际测试配置准备独立 PostgreSQL；记录设置与运行报告。故障注入覆盖迁移失败、行锁竞争、sending 回到 pending、超时与中断重入；不连接生产库运行 fixture。当前 Worm 业务包没有独立 Go 测试文件，`[no test files]` 只证明构建经过，不写成业务测试通过；保留能力须有 UI 场景、Wallet 既有 wire regression 和下述真实读取的相应证据。
- [ ] 按 [本地环境准备](../../developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance) 核对工作区／实例后复用正确环境，或从实施工作区用项目 Node 执行 `make run`，保存持久会话及日志。空实例和带旧权限的实例分别验证；启动不包含废弃服务配置和数据库准备。
- [ ] 当前 `make run` 尚不能代表 Worm 两个业务进程已运行。Worm 真实回归使用此次新版本的现有生产 Compose 服务定义，在另一个独立本地 project 中显式选择 API、Worm 与必要依赖；API、Wallet、Worm 的地址和内部凭据统一指向该验收 project，不混接上一套 `make run` 环境。使用 `--no-deps` 按已列明依赖顺序启动，避免 API 的完整 `depends_on` 隐式启动其他业务；不把这一步扩写成十二应用编排实施。
- [ ] 本地 Compose 隔离不能只依靠 project 名：当前三个存储卷使用固定 `name` 且 `external: true`。使用本任务专用环境文件，将 `PROD_POSTGRES_VOLUME`、`PROD_REDIS_VOLUME`、`PROD_MINIO_VOLUME` 全部改为包含本轮 run ID 的专用名称，先检查不存在，再创建并标记 owner；不能复用 `athena-prod-*-data` 默认卷。显式设置 `ATHENA_SERVER_BIND_ADDR=127.0.0.1` 和核对空闲的 `ATHENA_SERVER_PORT`，保存 `docker compose config --format json` 的解析结果，逐项确认卷、服务地址、挂载、凭据来源和服务集合。使用独立本地配置及测试身份；不能直接使用 `.env.prod`，也不能接管其他环境的 Telegram poller。启动前核对新镜像版本及所选 schema 准备步骤；结束时精确停止本 project 的使用者，保留专用卷和证据。若端口／卷归属不符，先修正配置后再启动。
- [ ] 核对真实 member／admin bootstrap 后，按 [athena-browser-acceptance](../../../.codex/skills/athena-browser-acceptance/SKILL.md) 执行以下两类检查。旧路由消失、Worm 七路由和授权交互通过隔离场景验证；真实 smoke 证明当前服务与浏览器链路。Worm 行情／状态的实际读取另留 API 和页面证据，涉及资产副作用只在隔离后端验证。

```bash
make ui-acceptance
make ui-acceptance UI_ACCEPTANCE_MODE=smoke
```

第二套 Compose 的 smoke 使用 `UI_ACCEPTANCE_BASE_URL` 指向该 project 实际的 loopback API／静态页面入口，不能沿用第一套环境的默认 `http://localhost:4000`。检查请求实际到达同一 project 的 API／Worm；分别保存两套环境的地址、版本、报告和停止记录。

真实服务未就绪时检查日志、修复已授权环境问题并重验；不能只凭端口监听、页面 200 或隔离用例通过完成验收。Worm 供应商或必要凭据确实不可用时标记对应验收未完成，数据删除的前置验证不能写成已通过。

- [ ] 手册明确 T7／T8 的只读清单格式、资源识别、超时和失败停止点，链接现有部署资料。失去源码目标的长期文档改为已退役说明／历史路径，不留下坏链接。当前 active 文档不再指导启动废弃模块；历史设计、批准与验收事实保留。
- [ ] 运行适用差异审阅及 `git diff --check`，核对活动入口无残留；移除仅依赖旧功能的构建／测试项。记录所有未解决问题，不能绕过 schema／权限检查使构建通过。
- [ ] 结束本地验收时，按 owner 停止本任务启动的临时进程与 Compose 服务，核验端口及容器退出；保留数据卷、日志和报告，借用／用户原有环境保持原样。普通测试收尾不得执行四库退役操作。

### T7：逐环境只读盘点，锁定真实退役对象

**文件：**维护手册及 `.superpowers/module-removal/retirement/` 下每个环境的清单／结果；在验收文档记录清单位置与摘要，不存储数据库历史副本。

**输入：**T6 验证版本、实际环境访问配置、历史部署线索。

**输出：**每个环境一份包含如下字段的 JSON 清单；对象 ID 等运行事实来自本步骤实查，不在规划阶段猜测。

```text
environment, host, checkout_or_app_dir, version, compose_project
retiring_processes: pid, start_time, executable, run_id, owner
retiring_containers: id, service, image_id, labels, mounts, restart_policy
retiring_databases: server_identity, database_name, oid, owner, active_consumers
exclusive_volumes: name, driver, labels, mountpoint, attached_container_ids
exclusive_files_and_images: exact_path_or_id, owner, retained_consumers
autostart_sources: exact_unit_or_job, owner, disable_method
preserved_resources: Gateway, Worm, core processes, shared storage and networks
steps: action, started_at, finished_at, result, evidence, remaining_objects
```

- [ ] 从实际本地 runtime 状态、生产 Compose project 和部署配置确定本地、主生产、两个索引器环境。执行 `docker ps -a --no-trunc`、`docker inspect`、`docker volume inspect` 和有界 PostgreSQL catalog 查询；记录同机 Gateway 基线。不得把 `.env.prod` 中的覆盖库名遗漏。
- [ ] 历史索引器线索固定为 `47.245.183.140:/opt/athena-bsc-transaction-indexer`、`47.254.154.128:/opt/athena-bsc-swap-indexer`；默认卷分别为 `athena-bsc-transaction-indexer-postgres-data`、`athena-bsc-swap-indexer-postgres-data`。只有现场 project 标签和实际挂载吻合才进入删除集合。
- [ ] 对共享 PostgreSQL 读取四库的实际归属、OID 和连接；查询同实例全部库及使用者，形成保留名单。对专属索引器卷读取全部挂载消费者，包括已停止容器；不因只有目标程序在线就推断卷独占。
- [ ] 识别 systemd／Compose restart policy／定时任务／旧部署自动重建入口，记录精确禁用方法。记录正常 schema 维护所需暂停的同库使用者，区分临时维护与永久退役。
- [ ] 清单先以只读结果交付可审阅记录；既有删除范围和无备份决定不重复审批。对象不明、身份变化或连接占用无法定位时停止该对象后续删除并记录原因，继续不依赖它的只读核对。

### T8：发布新版本、清理通知并直接删除专属数据

**文件：**按 T7 清单更新操作结果，维护手册中的命令必须使用同一环境的精确标识。下列为实施顺序，不在计划编写时执行。

**输入：**T6 的已验证新版本与 T7 当前身份仍匹配的对象清单。

**输出：**每环境明确的成功／部分完成／失败结果；不同主机、数据库与 Docker 资源不承诺原子删除或数据回滚。

- [ ] 每环境只允许一个退役执行者。每个破坏性步骤前重新读取身份，核对容器 ID／标签、数据库服务身份与 OID、卷挂载；与清单不符则停止依赖该对象的后续删除，不根据近似名称替换目标。
- [ ] 先从正常发布入口撤下旧用户业务和权限写入版本，再按 owner 停止四类专属生产者及自动拉起来源。Docker 对象按精确 ID 停止；本地进程通过其原仓库／INSTANCE 的停止入口处理，保留识别记录直到核验退出。Worm 和核心不作为永久退役对象。
- [ ] 完成实际同库使用者的维护停机，使用新版本 `athena-account-state-migrate up`、`verify` 后再启动一致的新版本消费者。使用现有正确密钥和配置，不运行 secrets 重置，不在旧代码在线写入 Sports 权限时修改约束。迁移／验证失败不发布，保持失败记录并修复新版本。
- [ ] 确认所有 Sports 生产者已退出后，运行 T2 工具先只读检查，再执行 `--apply --timeout=5m`；等待 Notification 既有恢复与发送结果。超时或 sender 归属不明时先按原恢复流程处置，不改写 sending 为 cancelled 或伪造成功。
- [ ] 验证新版本登录、账户、Wallet、Notification、Worm 及同机 Gateway 正常，旧接口不提供业务；确认六来源 `pending=0`、`sending=0`。如失败，停止后续数据删除，记录已经完成的不可逆步骤。
- [ ] 共享 PostgreSQL：连接该实例非目标维护库，设置有限 lock／statement 超时，查询 `pg_stat_activity` 确认目标无消费者。依次按精确名称删除实际 `sports_live`、`sports_history`（以及确实位于共享实例的目标 BSC 库），标识符通过 `pgx.Identifier{databaseName}.Sanitize()` 或 psql 标识符变量引用；不得拼接未校验文本，不使用 `WITH (FORCE)` 或强踢未知连接。每删一库立即重新查询 catalog。
- [ ] 独占索引器存储：确认目标程序和专属 PostgreSQL 都退出，停止并移除精确容器，再次确认专属卷无任何消费者，再删除清单中的专属卷。卷内整个实例确实仅属于该索引器时，删卷完成对应数据库数据删除，不另启动旧数据库补做 DROP；共享数据库卷不能走此路径。
- [ ] 完成停服与数据删除后，清理明确专属的二进制、环境文件、Compose／启动文件和部署临时包；精确目录清空后才能删除目录。专属镜像确认无保留引用后删除；Sports 共用的 `PROD_IMAGE` 随正常版本替换，不作为专属镜像删除。保留日志和定位证据。
- [ ] 单次进程／容器正常停止预算 30 秒，单项 SQL／Docker 操作预算 60 秒；遇到未退出／锁超时保留失败对象，不扩大信号或删除范围。Notification 总等待使用工具的 5 分钟预算；不能通过无界轮询掩盖卡住。
- [ ] 失败或中断后从实际状态重新核对；同一已识别对象确认不存在可记为完成，已成功删除不重建。新 ID、重新出现的库或卷必须重新核对。任一未完成项返回非零并记入结果，不宣称整个环境已清理。

### T9：最终核验、文档收尾与交付

**文件：**更新验收文档、操作手册、两个索引器服务器记录、Sports 现状文档及需求／设计索引；本计划复选框以实际完成情况填写。

**输入：**所有代码与测试结果、T8 每环境操作和失败记录。

**输出：**代码清理、运行退役、数据删除三项分别可核验的结论。

- [ ] 仓库：四个命令和对应实现消失，旧路由／权限／生成入口不再存在；重新构建不会恢复废弃能力。账户历史迁移、退役工具来源及历史文档的旧名称有明确保留理由。
- [ ] 每环境：实际查询证明旧进程／容器退出、所属端口释放、自动拉起消除、四库或其独占卷不存在、专属部署文件不可再启动；共享 API 仍监听不等于旧业务仍存在，按具体业务路由核验。
- [ ] 保留能力：Worm 两服务、两库、权限、签名／凭据与页面回归有证据；账户、Wallet、Notification 共享账本、其他业务和同机 Gateway 正常。最终检查不发送新的测试通知或资产变更。
- [ ] 文档只把实际完成项改为“已实施／已退役”；远端未执行或外部验证受阻时单列环境与原因。本地代码通过不替代生产退役，停止命令成功不替代数据删除核验。
- [ ] 按 AGENTS.md 收尾本任务临时环境，列明已停止和仍保留环境、归属、地址、日志及准确停止命令。退役目标的四库直接删除与普通测试环境的数据保留分别记录。
- [ ] 在最终答复前按 [verification-before-completion](../../../.agents/skills/verification-before-completion/SKILL.md) 核对实际证据；任务累计执行超过 600 秒时按 AGENTS.md 发送一次如实的任务结果邮件。仍有未完成项时不使用“全部完成”。

## 计划自查与需求覆盖

| 设计要求 | 承接任务 |
| --- | --- |
| 四程序及内嵌业务删除、Worm 与核心保留 | T1、T3、T4、T6 |
| 权限追加迁移、原编号保留、schema 契约与新旧库一致 | T3、T8 |
| 生成源与实际消费者同步 | T2、T3、T4 |
| 本地／生产配置、部署和构建入口清理 | T5、T7、T8 |
| 六来源通知行锁取消、在途真实结果、有限等待 | T2、T6、T8 |
| 四库直接删除、专属卷和共享数据边界 | T7、T8、T9 |
| 精确部署残留、共享镜像及同机 Gateway 保留 | T5、T7、T8、T9 |
| 身份变化、锁失败、部分成功、中断与重入 | T2、T6、T7、T8 |
| 实际浏览器与保留业务验收、临时环境收尾 | T6、T9 |

本计划不扩大删除范围，不重开已经确认的业务决定；现场对象 ID、真实连接与健康状况属于执行证据。设计和计划齐备不代表上述现场条件已经验证。
