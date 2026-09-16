# Athena Makefile 操作手册

本文档说明项目根目录 `Makefile` 中保留的常用命令，偏向日常本地运行、代码生成和生产部署速查。

## 前置依赖

常用命令会依赖以下工具：

- Go：用于代码生成，以及生产镜像内部构建二进制。
- Docker：用于构建生产镜像和运行生产 compose。
- yarn：用于手动在 `ui` 目录运行前端开发命令。

## 命令约定

- Makefile 只保留当前日常使用的入口：本地运行、代码生成、生产部署和数据清理。
- 生产镜像构建通过 Dockerfile 完成，Dockerfile 内部仍会调用 `make athena-all` 构建二进制。
- 所有命令默认在项目根目录执行。

## 常用环境变量

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `PROD_IMAGE` | `athena:local` | 生产部署使用的镜像名。 |
| `PROD_COMPOSE_FILE` | `docker-compose.prod.yml` | 生产 compose 文件路径。 |
| `PROD_ENV_FILE` | `.env.prod` | 生产部署读取的环境变量文件。 |
| `REMOTE_APP_DIR` | `/root/athena` | 远端服务器上的部署目录。 |
| `REMOTE_USER` | `root` | SSH 登录远端服务器使用的用户。 |
| `PROD_LOG_SERVICE` | 空 | 查看生产日志时指定服务名。为空时查看全部服务。 |
| `PROD_MIGRATE_MODULE` | `all` | 迁移目标模块。可设为 `account-state`、`worm-markets`、`notification`、`wallet`、`sports-live`、`sports-history`、`managed-oo`、`profit-sharing`、`token` 或 `all`。 |
| `PROD_POSTGRES_VOLUME` | `athena-prod-postgres-data` | PostgreSQL external volume 名称。本地停止、远程部署和远程删除都会删除该 volume。 |
| `PROD_REDIS_VOLUME` | `athena-prod-redis-data` | Redis AOF external volume 名称。本地停止、全新远程部署和远程删除都会删除该 volume；热部署保留。 |
| `PROD_MINIO_VOLUME` | `athena-prod-minio-data` | MinIO external volume 名称。本地停止、全新远程部署和远程删除都会删除该 volume；热部署保留。 |
| `MINIO_IMAGE` | `athena-minio:9e49d5e7a648-go1.27.1` | 从固定 MinIO Server commit、Go 1.27.1 构建的镜像名。 |
| `MINIO_MC_IMAGE` | `athena-minio-mc:7394ce0dd2a8-go1.27.1` | 从固定 mc commit、Go 1.27.1 构建的一次性初始化镜像名。 |
| `ATHENA_POSTGRES_AUTO_MIGRATE` | 本地默认 `true`，生产 compose 为 `false` | 控制服务启动时是否自动执行 PostgreSQL migration。生产部署脚本会在启动业务服务前显式迁移。 |
| `ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` | 无 | API Server、系统通知生产者与 Notification gRPC 之间共享的独立 Bearer；至少 32 字节且不得与其他内部凭证相同。 |
| `TARGET_ARCH` | `linux/amd64` | Docker 镜像构建平台。 |
| `ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME` | 无 | 任务完成通知使用的腾讯企业邮箱账号，同时作为邮件发件人。 |
| `ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_PASSWORD` | 无 | 腾讯企业邮箱的客户端专用密码，仅用于任务完成通知的 SMTP 认证。 |
| `TASK_NOTIFICATION_ENV_FILE` | `.env` | 任务完成通知补充读取的环境文件；需要生产配置时必须显式设为 `.env.prod`。进程环境中的同名变量优先。 |

## 环境与工具

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make install-codegen-tools-local` | 安装代码生成需要的工具。 | `make install-codegen-tools-local` |
| `make jwt-secret` | 生成可用于 `ATHENA_JWT_SECRET` 的 HS256 随机签名密钥。 | `make jwt-secret` |
| `make service-password` | 生成可用于 PostgreSQL、Redis 或 MinIO 的随机密码。 | `make service-password` |
| `make notify-task-complete` | 向固定邮箱发送一封任务完成纯文本通知。 | `make notify-task-complete TASK_NOTIFICATION_SUBJECT='任务完成' TASK_NOTIFICATION_BODY='处理已结束。'` |

生成 HS256 JWT secret：

```bash
# 默认生成 base64 编码的 32 字节随机密钥
make jwt-secret

# 也可直接运行
go run tools/jwt-secret/main.go

# 如需 hex 格式
go run tools/jwt-secret/main.go -format hex
```

输出为一行 secret。写入 `.env` 或 `.env.prod`：

```bash
ATHENA_JWT_SECRET='<generated-secret>'
```

Google OIDC、Phantom 钱包登录和 API Key 都使用 Athena 自有 JWT v3；JWT subject
中保存规范化的 UUID `account_id`，不保存 username。切换认证方案或需要强制
所有浏览器会话与 API Key 失效时，生成新值并替换 `ATHENA_JWT_SECRET`；随后所有
用户需要重新登录，自动化调用方需要重新创建 API Key。

生成数据库和 Redis 密码：

```bash
# 默认生成 32 位字母数字密码
make service-password

# 也可直接运行
go run tools/service-password/main.go

# 如需更长密码
go run tools/service-password/main.go -length 48
```

输出为一行仅包含大小写字母和数字的密码，可直接写入 `.env` 或 `.env.prod`：

```bash
POSTGRES_PASSWORD='<generated-password>'
REDIS_PASSWORD='<generated-password>'
```

### 任务完成邮件通知

仓库代理按 [任务结果邮件规则](../../AGENTS.md#task-result-email)执行：任何类型的任务
累计执行超过十分钟后，在任务结束时发送一次结果通知，不限 Plan 任务。失败、取消或
阻塞结束也须如实通知；等待用户回复和暂停时间不计入。计时与触发由代理负责，通知命令
本身不计时，也不会在执行满十分钟时自动发送。

任务完成后可显式调用独立通知目标：

```bash
make notify-task-complete \
  TASK_NOTIFICATION_SUBJECT='任务完成：同步市场数据' \
  TASK_NOTIFICATION_BODY='市场数据同步任务已执行完成。'
```

该目标不会自动挂接到其他 Make 命令。调用方需要为每次通知提供非空的
`TASK_NOTIFICATION_SUBJECT` 和 `TASK_NOTIFICATION_BODY`。发送边界固定为：

- SMTP：腾讯企业邮箱 `smtp.exmail.qq.com:465`，隐式 TLS 1.2 或更高版本；
- 发件认证：`ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_USERNAME` 和
  `ATHENA_TASK_NOTIFICATION_EMAIL_SMTP_PASSWORD`；
- 收件人：`2687665142@qq.com`；
- 内容：UTF-8 纯文本，不支持 HTML、附件、抄送或密送。

默认从项目根目录 `.env` 补充读取 SMTP 配置，当前进程环境中的值优先；如果进程
环境已包含全部必需配置，默认 `.env` 不存在也可发送。需要显式使用生产环境文件时：

```bash
make notify-task-complete \
  TASK_NOTIFICATION_ENV_FILE=.env.prod \
  TASK_NOTIFICATION_SUBJECT='生产任务完成' \
  TASK_NOTIFICATION_BODY='生产任务已执行完成。'
```

配置缺失、显式选择的环境文件无法加载、标题或正文无效时，命令在连接 SMTP 前
直接失败，不重试。
SMTP 连接、TLS、认证或发送失败时，每次使用新连接，总计最多尝试 3 次；第二次前
等待 1 秒，第三次前等待 2 秒。三次均失败后 Make 以非零状态退出，错误输出不会
包含客户端专用密码。

同一次调用的重试复用相同 `Message-ID`。如果网络在服务端接收邮件后、客户端确认
接收前中断，重试可能产生重复邮件；`Message-ID` 可以帮助识别重复，但不保证
只投递一次。服务端已经确认接收邮件后，即使关闭 SMTP 会话失败也不会重发。

## 浏览器认证配置

Google 只负责确认外部身份。任意通过完整 OIDC 校验且邮箱已验证的 Google 用户都可
开始注册；未知 `sub` 的 callback 只创建 15 分钟注册票据并跳转 `/register`，不会
立即创建账号或签发 Athena Cookie。用户提交唯一且永久的 username 后，系统才创建
以 UUID `account_id` 为内部身份的完整账号；普通账号初始没有业务、API Key 或
Profit Sharing 权限。认证启用时必须配置以下变量：

```env
ATHENA_GOOGLE_OIDC_CLIENT_ID='<google-web-client-id>'
ATHENA_GOOGLE_OIDC_REDIRECT_URI='<exact-callback-uri>'
ATHENA_ADMIN_GOOGLE_EMAIL='<administrator-google-email>'
```

在 Google Cloud 中分别创建本地和生产 **Web application** OAuth client，配置
consent screen/audience。要允许任意 Google 用户，Audience 必须为 **External** 且
应用必须发布；Testing 状态仍只允许 Test users。登记完全一致的 authorized
redirect URI：

- 本地：`http://localhost:4000/auth/google/callback`
- 生产：`https://<athena-domain>/auth/google/callback`

生产 URI 必须显式使用 HTTPS。Athena 不会从请求的 `Host`、
`X-Forwarded-Host` 等 header 推断回调地址。该 URI 的 scheme 与 authority 同时是
Phantom SIWS 消息使用的固定 domain/URI 信任边界。已注册 Google 账号始终按
`(google, sub)` 查找；
邮箱只用于安全审计，以及标记尚未注册的管理员候选。管理员邮箱比较仅去除首尾空白
并忽略大小写，不归并 Gmail 点号或 `+alias`。候选人仍需在 `/register` 选择永久
username；注册事务创建 `administrator=true` 的唯一管理员记录，权限来自该字段而
不是 username。持久化 `sub` 此后永久优先，修改环境邮箱不能替换管理员身份。

Phantom 首版只使用桌面浏览器注入的 `window.phantom.solana`，不引入 SDK，也不需要
App ID、client secret、移动端 callback、Solana RPC 或任何新环境变量。浏览器从
`/auth/phantom/challenge` 获取服务端生成的五分钟 SIWS 文本，使用 Ed25519
`signMessage` 签名，再提交 `/auth/phantom/verify`。签名只证明地址控制权，不发起
交易、不产生网络费用，也不读取余额或私钥。未知地址与未知 Google subject 都进入
共享 `/auth/registration` 选择永久 username；两种身份始终创建独立 UUID 账号，
不能合并、换绑或转让。Phantom 注册永远是 Pending 普通账号，不能认领管理员。

本地开发可直接设置 `ATHENA_GOOGLE_OIDC_CLIENT_SECRET`。生产必须保持该变量为空，
只使用独立文件：

```env
ATHENA_GOOGLE_OIDC_CLIENT_SECRET=''
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE='./secrets/google-oidc-client-secret'
```

准备文件时不要把 secret 写入环境文件或提交到 Git：

```bash
mkdir -p secrets
install -m 0600 /secure/source/google-oidc-client-secret secrets/google-oidc-client-secret
```

生产 Compose 将文件以只读 secret 仅挂载到 `athena-server` 的
`/run/secrets/google-oidc-client-secret`；migration 和其他业务容器不会获得文件
内容。本地生产预演让 `athena-server` 使用当前宿主机 UID 读取该用户自己的 `0600`
文件；远程部署脚本把上传副本改为容器 UID/GID `999` 持有并保持 `0600`。设置
`ATHENA_SERVER_DISABLE_AUTH=true` 时会同时创建或复用两个隔离的开发身份：用户端
使用 `local-user`，管理端使用 `local-admin`，不要求外部认证配置，也不再提供角色
选择环境变量或命令行参数。用户端和管理端请求分别携带精确的
`X-Athena-Application-Realm: member` 或 `X-Athena-Application-Realm: admin`；私有图片
GET 和 EventSource 等无法设置请求头的浏览器传输可使用 `athenaRealm` 查询参数。
请求头与查询参数同时存在时必须一致，缺失、非法、重复或冲突的 realm 不会默认映射
到任何身份。该模式只允许非 Compose 的开发进程监听 loopback 地址。生产 Compose
固定启用认证，部署脚本会拒绝该模式。

虽然其他生产服务仍复用选定的部署 env 文件读取各自业务配置，Compose 会把
`ATHENA_JWT_SECRET`、全部 Google OIDC/管理员邮箱输入以及 `REDIS_PASSWORD` 在所有
非 API Server 容器中显式覆盖为空；只有 `athena-server` 能签发 Athena JWT、访问
持久账号目录或访问认证 Redis。`ATHENA_WALLET_INTERNAL_AUTH_TOKEN` 也会在无关容器
中覆盖为空，仅 `athena-wallet` 和 `athena-server` 获得同一个 required 值。
`ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` 仅注入 `athena-notification`、
`athena-server` 以及 Market Radar、Sports Live、Managed OO、Worm Markets
四个系统通知生产者；其他容器中的同名值会被覆盖为空。Telegram Bot Token 与
测试/生产群组 ID 则只注入 `athena-notification`，API Server、生产者和其他容器中的
同名值都会被覆盖为空。
远端上传后的 `.env` 同样改为当前部署用户持有且权限为 `0600`。

## 代码生成

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make codegen-local` | 在本机执行完整代码生成流程。 | `make codegen-local` |
| `make protogen` | 先准备 vendor，再生成 protobuf 相关代码。 | `make protogen` |

## 生产构建

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make minio-images-local` | 从固定源码 commit 构建 MinIO Server 和 mc 初始化镜像。 | `make minio-images-local` |
| `make prod-build-local` | 构建 MinIO/mc 镜像及生产部署使用的 Athena 镜像。 | `make prod-build-local` |

常见用法：

```bash
PROD_IMAGE=athena:local make prod-build-local
```

## 本地运行

所有命令从目标 checkout/worktree 根目录运行；实例资源不能跨 checkout 混用。全栈命令默认实例名为 `full-stack`，可用 `INSTANCE` 指定别名，启停重置时须使用同一名称。

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make build-service` | 独立构建所选 Go 服务，不触发 UI 或聚合 main。 | `make build-service SERVICE=trader-sync` |
| `make run-service` | 启动一个业务服务和它声明的最小依赖。默认实例名为服务名。 | `make run-service SERVICE=trader-sync INSTANCE=ts-dev` |
| `make run-services` | 启动明确选择的业务集合，要求实例名。 | `make run-services SERVICES='trader-sync api-server ui' INSTANCE=ts-integration` |
| `make seed-service` | 为 managed 实例准备开发账户，输出 member/admin UUID；不恢复已撤销权限。 | `make seed-service SERVICE=trader-sync INSTANCE=ts-dev` |
| `make runtime-status` | 查看生命周期、进程、实际资源地址、退出码和日志。 | `make runtime-status INSTANCE=ts-dev` |
| `make stop-instance` | 核验身份后停止该实例拥有的进程和容器，保留数据。 | `make stop-instance INSTANCE=ts-dev` |
| `make reset-instance` | 仅重置已停止的 managed 实例，删除它拥有的数据。 | `make reset-instance INSTANCE=ts-dev` |
| `make run` | 显式启动 full-stack：TS/API/Notification/UI/Wallet/Profit Sharing。 | `make run` |
| `make stop` | 停止 full-stack 实例，沿用相同资源所有权协议。 | `make stop` |
| `make run-reset` | 重置已停止的 full-stack 实例，不自动停止或重新启动。 | `make stop` 后 `make run-reset` |
| `make build-service-image` | 构建独立 Trader Sync/schema tool 镜像。 | `make build-service-image SERVICE=trader-sync TRADER_SYNC_IMAGE=athena-trader-sync:local` |

独立入口支持 `trader-sync`、`api-server`、`notification`、`ui`；UI 无数据库依赖。
API/Notification 不隐式启动 TS。全栈通过显式服务图选择，不再用 `ATHENA_RUN_EXCLUDE`
从全栈中减服务。Bash5.1+/Linux/WSL 是运行器依赖；仅选择 UI 时需要项目 Node24。

默认 `DB_MODE=managed`：每个 checkout/实例自己的持久 PostgreSQL；API 另需自己的
Redis/MinIO 和私有头像 bucket。基础设施端口动态绑定 loopback，以 status 输出为准。
业务默认 UI4000/API8080/Notification8086/TS8122/Wallet8088/ProfitSharing8108，并存时
显式配置各业务端口。前台 Ctrl+C 和 stop 使用同一有界停止协议，保留 volume、fixture、
token 和日志；不会按全局端口、固定容器名或 `/tmp/coverage` 清理其他实例。

使用外部 account-state 库时显式指定 `DB_MODE=external ENV_FILE=.env.ts-external`，文件
提供 `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` 及相应内部凭据。只读 verify，不迁移、seed 或
管理数据库生命周期；external reset 拒绝。允许借用其他实例数据库，需协调原 owner。
全栈仅支持 managed。各业务进程只验证 schema，managed 运行器显式执行独立 up/verify。

没有首次运行必须 reset 的步骤。reset 会删除选定实例的账户、授权、活动、通知、会话、
头像及其全栈模块数据；不能解决远端未知交易或替代远端凭据撤销。disabled-auth 仍只限
loopback，分别以 realm 选择会员/管理员；切回正常认证可使用没有开发身份的新实例。
更多配置与异常恢复见[本地运行编排](../design/development-runtime/local-runtime-orchestration.md)。

## UI

UI 相关命令直接在 `ui` 目录执行，例如 `yarn install`、`yarn start`。

## 文档

文档直接以仓库内 Markdown 维护。开发前读取 `docs/requirements/README.md`、`docs/design/README.md`、相关能力文档和实际源码，按原版 [Superpowers 开发工作流](../developer-guide/superpowers-development.md)推进。任务规格与计划分别保存在 `docs/superpowers/specs/`、`docs/superpowers/plans/`；实现时同步需求与系统设计中的长期知识和源码关系。

## 生产部署

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `make prod-start-local` | 创建本地 PostgreSQL/Redis/MinIO volume、执行 migration、初始化私有 bucket 并启动生产 compose 服务。 | `make prod-start-local` |
| `make prod-stop-local` | 停止本机生产 compose 服务并删除 PostgreSQL/Redis/MinIO volume。 | `make prod-stop-local` |
| `make prod-logs-local` | 查看本机生产 compose 日志。 | `make prod-logs-local` |
| `make prod-reset-secrets` | 更新生产 env 中的 PostgreSQL、Redis、MinIO root、头像应用凭据、JWT secret、Notification 与 Wallet 内部服务 token。 | `make prod-reset-secrets` |
| `make prod-deploy-remote` | 自动轮换凭据、构建镜像、清空远程数据库并完成全新部署。 | `make prod-deploy-remote` |
| `make prod-hot-deploy-remote` | 构建镜像并热部署后端服务，保留远程 PostgreSQL、Redis 和 MinIO 数据。 | `make prod-hot-deploy-remote` |
| `make prod-destroy-remote` | 删除远程 Athena 运行资源及 PostgreSQL/Redis/MinIO volume。 | `make prod-destroy-remote` |

当前 Wallet schema 必须部署到全新的远程数据卷。发布本版本时使用
`make prod-deploy-remote` 完成全新部署，不得使用会保留旧 PostgreSQL/Redis/MinIO
数据的 `make prod-hot-deploy-remote`。

### 部署前本地预演

部署远端服务器前，可以先用生产镜像和生产 compose 在本机跑一次。`prod-start-local` 会创建 PostgreSQL/Redis/MinIO volume、启动 PostgreSQL、执行 migration、初始化私有头像 bucket，再启动其余服务。Redis 使用 AOF 持久化未过期的会话撤销记录。

如果使用 `.env.prod` 作为预演环境文件，至少需要包含：

```env
POSTGRES_PASSWORD=your_postgres_password
REDIS_PASSWORD=your_redis_password
MINIO_ROOT_PASSWORD=your_minio_root_password
ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID=your_avatar_access_key
ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY=your_avatar_secret_key
ATHENA_JWT_SECRET=your_at_least_32_byte_jwt_secret
ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN=your_at_least_32_byte_notification_internal_token
ATHENA_WALLET_ENCRYPTION_KEY=your_wallet_encryption_key
ATHENA_WALLET_INTERNAL_AUTH_TOKEN=your_at_least_32_byte_wallet_internal_token
ATHENA_GOOGLE_OIDC_CLIENT_ID=your_production_web_client_id
ATHENA_GOOGLE_OIDC_CLIENT_SECRET=
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=./secrets/google-oidc-client-secret
ATHENA_GOOGLE_OIDC_REDIRECT_URI=https://athena.example.com/auth/google/callback
ATHENA_ADMIN_GOOGLE_EMAIL=owner@example.com
```

`ATHENA_WALLET_ENCRYPTION_KEY` 可用以下命令生成：

```bash
openssl rand -hex 32
```

该 key 用于加密 wallet 相关敏感数据。已有 wallet 数据后不要随意更换，否则旧数据可能无法解密。

`ATHENA_WALLET_INTERNAL_AUTH_TOKEN` 是 API Server 调用 Wallet gRPC 的独立
Bearer，不是钱包加密主密钥。Wallet 与 API Server 必须配置同一个至少 32 字节的值；
其他容器会被 Compose 显式覆盖为空。可执行 `make prod-reset-secrets` 生成并轮换
40 位字母数字 token。缺失或不匹配时 Wallet health 仍可探测，但所有非 health RPC
都会被拒绝。

`ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN` 采用相同的最小长度和独立性要求，
但只用于 Notification gRPC。Notification、API Server 和四个系统通知生产者必须使用
同一个值；健康检查不携带该凭证，其他内部 RPC 缺失或不匹配时会被拒绝。
`make prod-reset-secrets` 会同时生成一个与 Wallet、Wallet signer 和 Worm Trading
内部 token 均不相同的 40 位值。

推荐预演流程：

```bash
make prod-build-local
make prod-start-local
```

`prod-start-local` 会强制设置 `ATHENA_SERVER_DISABLE_AUTH=false`，即使环境文件中配置为
`true`，本地生产预演仍会启用服务端认证。运行前必须创建
`./secrets/google-oidc-client-secret` 并配置真实的 client ID、生产预演回调 URI 和
管理员 Google 邮箱；缺失配置会使 API Server 拒绝启动。未知 Google 或 Phantom
身份验证成功后会先进入 `/register`；用户提交永久 username 后才创建管理员或
Pending 普通账号，不需要预先提取 `sub` 或配置 Solana 地址。Phantom 不需要额外
生产 secret。反向代理除现有 `/auth/google/*`、`/auth/phantom/*` 和 `/api/*`
外，还必须原样转发 `/auth/wallet-secrets/google` 与
`/auth/wallet-secrets/solana/*`；钱包私钥查看依赖 Redis 中固定 5 分钟的重新认证
lease，Redis 不可用时该能力会关闭。
Compose 会同时使用 `$(PROD_ENV_FILE)` 做变量插值和容器 `env_file` 注入，不会回退
读取仓库根目录的 `.env`。`prod-reset-secrets` 会轮换 Notification 与 Wallet 内部服务
token，并把该文件权限收紧为 `0600`。

生产 compose 中各后端服务设置了 `ATHENA_POSTGRES_AUTO_MIGRATE=false`。`prod-start-local` 会在启动业务服务前自动执行 `athena up --module $(PROD_MIGRATE_MODULE)`，默认迁移全部模块；迁移失败时命令会终止并保留 PostgreSQL 容器，便于排查。

查看日志和访问本地服务：

```bash
make prod-logs-local
```

```text
http://127.0.0.1:8080
```

停止本地预演：

```bash
make prod-stop-local
```

`prod-stop-local` 会删除 compose 容器、孤立容器、网络，以及
`PROD_POSTGRES_VOLUME`、`PROD_REDIS_VOLUME` 和 `PROD_MINIO_VOLUME` 指定的三个
volume，但保留本地
构建的 Athena、MinIO 和 mc 镜像。下一次启动会重新创建空数据库和私有 bucket。

### 远程部署

以下示例使用：

```env
REMOTE_HOST=47.245.181.189
REMOTE_USER=root
REMOTE_APP_DIR=/root/athena
PROD_POSTGRES_VOLUME=athena-prod-postgres-data
PROD_REDIS_VOLUME=athena-prod-redis-data
PROD_MINIO_VOLUME=athena-prod-minio-data
```

部署前先确认远端 Docker 和 Docker Compose v2 可用：

```bash
ssh root@47.245.181.189 'docker --version && docker compose version'
```

如果远端提示 `docker: 'compose' is not a docker command`，说明缺少 Docker Compose v2 插件。优先尝试安装系统包：

```bash
ssh root@47.245.181.189 '
set -e
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  apt-get install -y docker-compose-plugin
elif command -v yum >/dev/null 2>&1; then
  yum install -y docker-compose-plugin
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y docker-compose-plugin
else
  echo "Unsupported package manager; install Docker Compose plugin manually."
  exit 1
fi
docker compose version
'
```

如果系统包不存在，可手动安装 Compose CLI 插件：

```bash
ssh root@47.245.181.189 '
set -e
mkdir -p /usr/local/lib/docker/cli-plugins
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="aarch64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac
curl -SL "https://github.com/docker/compose/releases/download/v5.1.2/docker-compose-linux-${ARCH}" -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version
'
```

一键部署：

```bash
make prod-deploy-remote
```

该命令会先更新 `$(PROD_ENV_FILE)` 中的 PostgreSQL、Redis、MinIO root、
头像 bucket 应用凭据、JWT secret 和 Wallet 内部服务 token，再构建 Athena、固定
源码 MinIO 和 mc
镜像。构建成功后，依次停止远端旧服务，删除并重建
`$(PROD_POSTGRES_VOLUME)`、`$(PROD_REDIS_VOLUME)` 与
`$(PROD_MINIO_VOLUME)`，上传 Compose、环境文件和
Google OIDC client secret 文件以及 PostgreSQL init 脚本，传输三个镜像，执行
migration，初始化私有 bucket，最后启动全部服务并输出容器状态。部署脚本会在
上传前拒绝直接环境变量形式的 client secret、空 client/管理员邮箱、
非 HTTPS 生产回调 URI、少于 32 字节的 JWT signing secret、少于 32 字节的 Wallet
内部服务 token 或空 Google secret 文件，也拒绝生产环境使用
`ATHENA_SERVER_DISABLE_AUTH=true`。Docker 构建上下文会
排除所有 `.env` 文件和 `secrets/` 目录，避免部署凭据进入镜像构建缓存。

**每次远程部署都会永久删除已有 PostgreSQL、Redis 和 MinIO 数据，并轮换
PostgreSQL、Redis、MinIO、JWT 和 Wallet 内部服务凭据，不会自动备份。** 动态账号、
管理员绑定、
Profile、权限、Profit Sharing 引用、头像、会话和 API Key 都不会迁移。所有用户
必须重新使用 Google 或 Phantom 登录并设置永久 username，管理员重新授权；需要
自动化访问的账号必须在获得 API Key 权限后创建新的 v3 Key。migration 或 bucket
初始化失败时不会启动 API Server。

如需只手动更新生产凭据文件而不部署：

```bash
make prod-reset-secrets
```

后端代码小幅修改时，可以保留现有数据库并热部署：

```bash
make prod-hot-deploy-remote
```

该命令会构建并传输三个新镜像，覆盖远端 Compose、环境文件和 Google OIDC
client secret 文件，等待 PostgreSQL 和 Redis 就绪，随后启动并等待 MinIO，幂等确保
精确的 `profit_sharing` 数据库和私有头像 bucket 存在，再在现有数据上执行
migration，然后强制重建全部 Athena
后端服务并最后重建 `athena-server`。PostgreSQL、Redis 和 MinIO 的三个持久化
volume 数据都会保留；Compose 仅在配置变化要求时重建对应 stateful container，
不会删除 volume。任一指定 volume 不存在时命令直接终止，避免意外创建空状态。
依赖初始化或 migration 失败时不会进入应用重建阶段。

热部署不会自动轮换 PostgreSQL、Redis、MinIO、JWT secret 或 Wallet 内部服务 token。
它会校验并重新
上传当前独立的 Google OIDC client secret，然后短暂重启
Athena 服务，不保证零停机；适用于代码更新和兼容性数据库 migration，不用于
修改现有持久化服务凭据。

一键删除：

```bash
make prod-destroy-remote
```

该命令会删除远端 Compose 容器、孤立容器、网络、
`$(PROD_POSTGRES_VOLUME)`、`$(PROD_REDIS_VOLUME)` 和
`$(PROD_MINIO_VOLUME)`。命令可重复执行，不需要
额外确认参数；不存在的 volume 会被忽略，但任何仍存在的 volume 删除失败都会使
命令失败，不会继续宣称销毁完成或在全新部署中复用旧数据。远端部署文件和已加载的
三个镜像会保留。

验证远端服务：

```bash
ssh root@47.245.181.189 'cd /root/athena && PROD_POSTGRES_VOLUME=athena-prod-postgres-data docker compose -f docker-compose.prod.yml --env-file .env ps'
ssh root@47.245.181.189 'curl -sS http://127.0.0.1:8080/api/version'
```

默认生产 compose 将 `athena-server` 绑定到远端 `127.0.0.1:8080`。本机访问时可打开 SSH 隧道：

```bash
ssh -L 8080:127.0.0.1:8080 root@47.245.181.189
```

然后访问：

```text
http://127.0.0.1:8080
```

如果日志报告 Google OIDC client、redirect URI 或管理员邮箱为空，表示认证配置未
完成。正常生产部署不会生成临时管理员凭据，也没有密码兜底入口；修正配置并重启
`athena-server`，不要通过关闭认证绕过问题。

远端日志和状态不再提供独立 Makefile 目标，可直接使用 SSH：

```bash
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env ps'
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env logs -f athena-server'
```
