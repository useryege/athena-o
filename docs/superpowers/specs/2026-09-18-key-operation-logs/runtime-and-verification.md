# 运行边界、配置与验收设计

> 详细设计已于 2026-09-18 整体审阅通过，按用户要求暂不实施。入口：[总体设计](../2026-09-18-key-operation-logs-design.md)。以下命令是实施目标，尚未存在。

## 1. 职责与入口

| 单元 | 目标位置 | 唯一职责 |
| --- | --- | --- |
| 事件类型／目录 | `internal/operationlog/event` | 类型化事件、白名单、动作分类和合法性校验，不启动进程 |
| producer adapter | `internal/operationlog/ingest` | API 内有界入箱、幂等重试和本实例采集状态；不消费业务记录 |
| 存储／投影 | `internal/operationlog/store` | 日志 schema、sqlc 查询、队列消费和快照发布 |
| 管理员权限适配器 | `internal/operationlog/access` | 只读核验账户管理员身份和 login_enabled，不持有 API runtime |
| 内部 RPC | `internal/operationlog/transport`、`api`、`apiclient` | 服务鉴权、Viewer 校验、查询与运行状态 |
| 独立服务／迁移 | `cmd/athena-operation-log`、`cmd/athena-operation-log-migrate` | 独立运行和 schema owner，不导入 API Server 以组装服务 |
| 公共 facade | `internal/server/operationlog` | 管理员入口、协议映射及两个 API-local 查询 |
| 管理员页面 | `ui/src/app/admin/operation-log-service.ts`、`pages/operation-logs.tsx`、`pages/operation-log-detail.tsx` | 目录、筛选、列表、详情与独立状态源 |

将日志服务的编译、镜像和启动入口纳入现有 Makefile／本地运行 registry／生产 Compose；仅编排文件和构建产物，不执行生产部署。服务容器仅内部暴露端口，用户仍经 API Server。镜像独立构建，只包含自身和声明依赖。

目标命令：`go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate`、`make run-service SERVICE=operation-log`、同一 INSTANCE 的 `make stop-instance`。局部启动只要求 PostgreSQL、account schema 和 operation-log schema，不启动 Wallet、Worm、Trader Sync、Redis 或 UI。

## 2. 配置

| 配置／参数 | 默认／约束 | 消费者 |
| --- | --- | --- |
| ATHENA_ACCOUNT_STATE_POSTGRES_DSN | 复用已配置账户库；不得改为日志专用库而丢失权限适配器前提 | 服务、producer、迁移 |
| ATHENA_OPERATION_LOG_LISTEN_ADDRESS | 127.0.0.1:8124；本地实例可覆盖，启动前核对冲突 | 服务 |
| ATHENA_OPERATION_LOG_SERVER_ADDRESS | 127.0.0.1:8124，编排按实例设置 | API |
| ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN／_FILE | 非空，复用已有 Secret resolver 的互斥规则；实例生成后持久保存 | 服务、API RPC client |
| ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY／_FILE | 32 字节密钥，以 64 个十六进制字符配置；不同于内部 Bearer；重启不随机更换 | 服务 |
| ATHENA_OPERATION_LOG_GRPC_TRANSPORT | plaintext 或 tls；本地默认 plaintext，非本地按部署显式配置 | 服务、API |
| ATHENA_OPERATION_LOG_TLS_CERT_FILE／TLS_KEY_FILE | tls 时必填 | 服务 |
| ATHENA_OPERATION_LOG_TLS_CA_FILE／TLS_SERVER_NAME | tls 客户端验证配置 | API |
| 启动／停止超时 | 60 秒启动、30 秒停止；迁移 120 秒 | 编排与服务 |
| 采集预算／并发 | 每阶段 200 ms、最多两次；同一请求所有动作共享最多 400 ms；4 个 DB 连接、16 个并发记录任务 | API producer |
| 消费预算 | 500 ms 轮询、100 事件／批、2 秒／事务；最多 30 秒退避 | 服务 |
| 查询预算 | List 3 秒、Get 2 秒、Runtime 1 秒；数据库查询设置更短的 statement_timeout | 服务、API |
| 请求容量 | 公共查询消息 64 KiB；event 32 KiB；details 16 KiB；最多 100 资源引用；page_size≤100 | 各边界 |

预算和容量作为有名常量并带边界测试；第一版不增加全部可调的管理配置页面。API 对日志服务地址／凭据缺失使用不可用 client 并显示日志异常，不因这个独立依赖停止既有业务；日志服务自身配置不合法则明确失败退出。

服务启动只验证 schema，不自动迁移。schema／数据库运行中不可用时查询返回 503，消费有界重试；恢复后重新验证日志 schema 和账户只读契约，恢复消费。健康 Check 的 serving 状态反映查询依赖是否可用，quarantine／积压通过 RuntimeStatus 单独显示，不以进程存活代替处理健康。

## 3. 故障矩阵

| 故障 | 用户业务 | 管理员所见／恢复 |
| --- | --- | --- |
| 日志进程停止、数据库正常 | 正常；事件继续持久入箱 | 日志查询不可用，当前 API 采集可正常；重启消费积压 |
| 投影循环暂时失败、查询库可用 | 正常 | 旧快照可查询，处理异常／积压单独提示 |
| 日志 schema 缺失／不可写 | 日志等待有界，业务按自身规则继续 | 当前采集异常；未入箱事件不保证补齐；修复 schema 后新采集恢复 |
| 共享数据库整体不可用 | 日志不增加拒绝规则；既有账户依赖可能按自身规则失败 | 不能保证管理员认证／查询仍可用；显示实际错误，保留旧页面时间 |
| 开始事件成功、结果写入失败 | 不改变已执行业务 | START_ONLY／UNKNOWN，保留业务关联标识 |
| 开始失败、结果成功 | 正常 | FINISH_ONLY，以真实结果展示，说明缺少开始记录 |
| 消费 commit 前崩溃 | 正常 | 事务回滚，重启重投；无半个查询版本 |
| 入箱 commit 回执丢失 | 正常 | 同事件重试确认；不能把超时计数误写成确定丢失条数 |
| 单条冲突／非法事件 | 正常 | QUARANTINED 和稳定原因；不阻断其他日志，不把事件修改为成功 |
| API 进程异常退出 | 已入箱仍可消费 | 旧生产者状态过期，进程内未确认计数可能不完整；不宣称零遗漏 |
| cursor 密钥变化／游标过期 | 不影响业务 | 旧查询要求显式 Refresh；不自动更换筛选或快照 |

采集错误以固定 reason 和实例 ID 记录到既有运行日志，聚合限频每 30 秒一次；不发送业务邮件、Telegram 通知或报警消息。用户要求的“显示日志异常”由管理员日志页承担。

## 4. 生成与源码同步范围

1. 新增日志 schema 的迁移、验证契约及 sqlc 源；在 sqlc.yaml 为日志添加独立条目，所有表用 operation_log 限定名。完成稳定批次后执行 `make sqlc-local`，再修改依赖生成类型的 store。
2. 新增内部和公共 proto 及消息映射；使用 `make protogen`，不手改生成文件。日志 DTO 直接由 proto 定义，不为此修改与日志无关的 Kubernetes API 类型。
3. 为 gRPC 与原生 HTTP 注入 recorder；认证／账户 manager 只在已确认提交点提供观察 hook，不改业务事务、权限和返回协议。hook 必须 no-op-safe，不让其异常改变业务。
4. 管理员服务 registry、请求 feature `admin-operation-logs`、路由、导航、read scope 与 UI 实际消费者同批更新。会员 bundle 不导入管理服务。
5. 本地 registry、schema owner、配置白名单、依赖图、状态／停止及生产 Compose 同步。generic migration 的 all 路径须识别日志自有 schema／版本表，不能调用现有默认 public 版本表入口迁移它。

本阶段只设计，没有运行上述代码生成和构建命令；实施时在对应源稳定后执行。

## 5. 必须取得的验证证据

| 编号 | 证据／通过条件 |
| --- | --- |
| V01 | 描述符与事件目录核对：每个非 Get/List/Version RPC 和每个原生变更路由都有记录或明确排除；ResolveTarget、心跳与自动执行不会误记 |
| V02 | HTTP→gateway→gRPC 与 gRPC-web 每次动作各一条操作，原生 HTTP 也覆盖；客户端重试是新尝试，日志重试不重复 |
| V03 | 普通会话／API Key／开发身份归属正确；伪造 accountId、realm 或 body actor 不影响真实操作者；认证失败不伪归属账户 |
| V04 | 权限、模块开关、Origin、再认证与版本拒绝都保留原结果，并在可识别时记录真实账户 |
| V05 | 注册数据库已提交但 publish／登录失败，保留账户建立事实；登录授权取消、要求注册、退出撤销失败各自准确 |
| V06 | 同步成功、异步受理、部分完成、远端未知、响应写回失败与 panic 不混淆；原子钱包批量不假造部分成功 |
| V07 | 白名单覆盖私钥创建／导入／查看、API Key、备注、签名、Cookie／OAuth；事件与 UI 不出现原始材料 |
| V08 | 真实 PostgreSQL 验证原子入箱、同键同内容重复、同键冲突、START／FINISH 乱序、仅一阶段和单条隔离 |
| V09 | 发布事务中止、进程重启、两个 projector 竞争、低 ingest_id 晚提交、全库临时错误和大批积压恢复 |
| V10 | 查询跨页时新增、结果变更、身份补充和晚到记录不会破坏 W 快照；结果筛选及详情与列表一致 |
| V11 | cursor 篡改、过期、参数不匹配、跨用户／新会话复用均拒绝；无 total 时 UI 不伪造页数 |
| V12 | 日志 schema 迁移前后 public 账户目录与版本完全保持；verify 空库不写入；migration all 路由到正确 owner |
| V13 | 内部服务 token 与管理员持久角色均校验；撤销管理员／关闭登录后服务拒绝，不只依赖导航隐藏 |
| V14 | 日志服务和 schema 故障不改原业务结果；200 ms 预算、并发满、池耗尽、取消／停止均有界；补记不重复业务 |
| V15 | API／日志服务各自状态时间、实例与计数正确；未知采集缺口、过期生产者和重启后的新计数如实展示 |
| V16 | 桌面／手机／200% 缩放、键盘、错误／空态、八种结果、游标失效与旧响应清理完整覆盖 |
| V17 | 独立服务构建、最小依赖启动、内部鉴权、健康、默认／前缀 Web 地址及同实例停止，日志表和其他数据库数据保留 |
| V18 | 真实目标环境中以会员操作产生日志、管理员查询并核对结果；日志进程停止再恢复完成真实重验；外部凭据受阻项明确记录 |

先对行为契约做 TDD，再执行对应 Go、数据库、前端测试和必要的真实浏览器 smoke；隔离用例和 mock 不替代 V18。文档设计阶段只核对契约、源码引用、事件覆盖和一致性，不启动业务服务。

## 6. 服务规范映射与交付

SDS-R1：日志状态、消费和查询归独立进程；API recorder 为窄适配器。SDS-R2：查询 gRPC、内部鉴权、可信 Viewer、服务侧权限复核，采集采用明确的同库持久消息协议。SDS-R3：独立构建／运行／停止。SDS-R4：自有配置、deadline、故障矩阵。SDS-R5：按 INSTANCE 精确管理资源。SDS-R6：各自持有事务与连接，原业务原子性不变。SDS-R7：本文和关联契约记录目标与未实现差距。SDS-R8：V01–V18 与实际证据对应。

实施交付前完成 AI 审阅、约定验证和环境收尾；只停止本任务启动的实例，保留数据库、日志和报告，再交付 ATHENA 人工审查材料。只有本轮 AI 工作全部具备证据时才写“本轮 AI 交付完成，待人工审查”，不以设计审阅代替人工产品验收。
