# Solana 首版发现与列表验收

> 后续状态更新（2026-09-13）：用户已要求停止采集，Solana 预览、扫描及补全进程已退出；保留 2321 条候选，起点 446684678、游标 446687144。下文运行地址、PID 和队列统计均为验收当时记录，不代表目前在线。当前状态见 Solana 设计总览；本次文档整理没有重新启动服务。

验收日期：2026-09-13。基线 `3b1cd556`，工作分支 `codex/solana-discovery`。范围为发现新 Mint、持久保存及独立 ATHENA 列表；项目研究与活动刷新没有实现。

## 实现和生成

- `internal/solanadiscovery`、`cmd/athena-solana-discovery` 独立拥有扫描、数据库与查询生命周期；`go list -deps ./cmd/athena-solana-discovery` 未包含 API server、Token 或 Market Radar 实现。`make solana-discovery-build` 成功。
- `make protogen`、`make sqlc-local` 已成功运行，保留 Solana/account 目标 Go/Gateway/Swagger/SQL 消费者。生成使用工作树隔离 GOPATH 和 vendor，未改动主目录文件。
- Solana 权限 enum=13、READ/NONE；成员默认NONE，完整账户矩阵11项。旧账户初始schema升级至000002的真实PostgreSQL验证保留Token READ_WRITE、revision=7，新增Solana NONE。

## 自动验证

|验证|结果与范围|
|---|---|
|Go 单元测试|accountaccess、server、account、Solana facade、业务RPC、内部client、API命令及独立命令通过|
|真实PostgreSQL与race|显式设置SOLANA_TEST_POSTGRES_DSN，核心+RPC+client+独立命令通过；包括失败不越过、事务回滚、并发单赢家、启动失败恢复、取消与Retry-After|
|go vet|Solana所有包及独立命令通过|
|UI Jest|6套件38测试通过；包含矩阵保留、READ授权、页面查询/分页/错误/复制外链|
|yarn lint|TypeScript、ESLint及12项ESLint配置检查通过|
|ShellCheck|局部运行脚本及其行为测试通过|
|真实Goreman故障测试|普通停止/重复启动/伪造归属拒绝、父进程崩溃、孤儿会话、拒绝TERM子进程全部通过|
|git diff --check|通过|

数据库race证据：`.tmp/solana-final-race.log`；Go汇总`.tmp/solana-go-test.log`；UI汇总`.tmp/solana-ui-tests.log`与`.tmp/solana-ui-lint.log`；局部进程`.tmp/solana-final-runtime-test.log`。核心解析另外回放真实17,153,956字节、1935交易区块，成功解析；该区块独立JSON统计也无新Mint，没有把零样本误认为解析失败。

## 主网真实数据与恢复

使用 `https://api.mainnet-beta.solana.com`，核对Mainnet genesis；扫描起点446684678，finalized逐块取证。浏览器验收时已有56个候选；重启验证前61个，重启后111个。它们来自真实成功的Token-2022初始化，未进行NFT/LP/投资项目分类。

可核对样本：Mint `s8z8Xg4b21SnUaucPKEzZ1rCmPFkLxYu2okNHidpump`，slot446684707，[Solscan代币](https://solscan.io/token/s8z8Xg4b21SnUaucPKEzZ1rCmPFkLxYu2okNHidpump)；[发行交易](https://solscan.io/tx/4NtC5WuUWnS281k5gqpFEprUkdwZ6HLVfsqS2UsPSdK4u94cX2GqxJokmKT6HeiXpbjYsjsdXx1WuaMZevsB2zZ3)。链接与节点/API证据一致，本次不宣称第三方站点访问验收。

终止归属已核验的扫描进程后，0.1秒退出；Solana接口503，其他会员bootstrap仍200。保存候选与检查点未丢失。重新启动当前最终代码后，起点保持446684678，处理位置从446684785继续到446684794，数量由61到111；重启前抽样Mint的发行交易和首次发现时间逐字相同。前后原始响应在`.tmp/solana-restart/`。

本次未提交实现迭代期间，临时预览库曾按早期schema创建。验收前仅修正该任务库的nullable列及Token-2022约束，重扫最早已处理范围补齐证据；没有重置共享数据库、删除候选或改动主目录开发服务。最终源码的初始schema即为正确结构。

## 真实浏览器

最终运行环境：工作树 `/home/yege/work/athena/.worktrees/solana-discovery`，UI `http://127.0.0.1:14000/solana`、API18080、扫描gRPC18112，共用已运行基础设施但账户/项目数据库为独立 `athena_solana_preview`，Redis DB13。

正式smoke命令：

```bash
make ui-acceptance UI_ACCEPTANCE_MODE=smoke UI_ACCEPTANCE_BASE_URL=http://127.0.0.1:14000
```

最终重启后退出0，runId `2026-09-13T11-09-14-504Z-cceb012c`。会员和管理员bootstrap及应用壳均通过，原始报告在`.tmp/athena-ui-acceptance/2026-09-13T11-09-14-504Z-cceb012c/`。

另用系统Chrome对实际Solana页面、实际API结果做验收，没有拦截网络：第一页25项、第二页25项；Mint查询命中1项；刷新两个API均200；50个固定Solscan域链接和复制值与API一致；展开初始化authority、fee payer、slot和program一致。1440px桌面和390px手机的亮/暗主题均无整页横溢，表格在自身容器滚动。console error/page error/失败请求均为0。截图、原始JSON及trace在`.tmp/solana-browser-acceptance/`。浏览器业务验收后又做上述停止/重启恢复与最终smoke。

## 运行限制和保留环境

公共节点出现过429，单块响应约17MB，次数预算不等于字节预算。首版持续保存样本，但仍显示catching_up，未证明追平主网或固定发行发现延迟；未证明所有自定义Token程序/历史范围覆盖。真实页面样本均为Token-2022，原始Token Program由解析/存储测试覆盖，本轮不虚称看到该程序的真实新发行。

当前预览由 `make run ATHENA_RUN_PROFILE=solana-preview` 保留运行，执行会话51985，Goreman PID2613015，持久身份文件`.run/solana-preview/process`，日志`.tmp/solana-preview-final.log`。从该工作树执行 `make stop ATHENA_RUN_PROFILE=solana-preview` 停止这组进程；基础设施及数据保留。主目录原有4000/8080环境未停止或替换。

SDS-R1/R2/R3/R4/R5/R6/R7/R8 的设计、源码和实际证据已分别包含于[项目发现设计](../design/solana-intelligence/project-discovery.md)、[运行说明](../developer-guide/running-locally.md#solana-discovery-local)及本文。
