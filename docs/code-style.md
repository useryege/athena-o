# 代码规范

本文档定义 ATHENA 仓库中 Go 后端代码的命名与组织约定。目标不是追求形式上的「企业感」，而是：**Go 风格优先、领域语义清楚、跨模块一致、可长期维护**。

## 目录

- [设计原则](#设计原则)
- [命名规范](#命名规范)
  - [包名](#包名)
  - [文件名](#文件名)
  - [类型](#类型)
  - [函数与方法](#函数与方法)
  - [变量](#变量)
  - [接口](#接口)
  - [常量与枚举](#常量与枚举)
  - [错误](#错误)
  - [缩写](#缩写)
- [ATHENA 项目约定](#athena-项目约定)
- [反面示例速查](#反面示例速查)
- [格式与静态检查](#格式与静态检查)

## 设计原则

1. **命名表达业务，不表达技术层级。** 优先 `CreateNotification`、`RouteTelegramMessage`，避免 `DoCreate`、`ProcessData`、`HandleThing`。
2. **包名简短，类型承载领域名，方法表达动作。** 不在类型名中重复包名含义。
3. **边界清晰。** API 入参/出参用 `Request` / `Response` 后缀；失败语义用 `ErrXxx`。
4. **遵循 Go 社区惯例。** 不用 `I` 前缀接口、少用 `Impl` 后缀、不用 Java 风格全大写常量。

一句话总结：**短包名 + 领域类型 + 动词方法 + Request/Response 边界 + ErrXxx 失败语义**。

## 命名规范

### 包名

- 全小写、短、单数：`user`、`order`、`notification`、`ingest`、`cache`
- 避免空泛包名：`utils`、`common`、`helpers`、`models`（仓库根目录 `common/` 为历史例外，新代码不要新增同类包）
- 包名不重复表达路径含义：

```go
// 推荐
internal/notification/service.go
notification.Service

// 不推荐
internal/notification/notification_service.go
notification.NotificationService
```

### 文件名

- 小写；多个单词用下划线：`service.go`、`repository.go`、`http_handler.go`、`evm_reader.go`
- 测试文件：`<name>_test.go`，与被测文件同目录
- 按**职责**命名，不按类型名或「类名」命名

### 类型

- 领域对象用名词：`User`、`Project`、`Notification`、`Ingestor`
- API 请求/响应类型使用明确后缀（protobuf 生成代码亦遵循此约定）：

```go
CreateUserRequest
CreateUserResponse
UpdateProjectRequest
ListNotificationsResponse
```

- 配置与构造参数可用 `Options`、`ServiceOpts` 等后缀，与领域实体区分
- 避免空泛名称：`Data`、`Info`、`Manager`、`Processor`、`HandlerImpl`

### 函数与方法

- 使用**动词 + 业务对象**：

```go
CreateUser()
GetProject()
ListNotifications()
DeleteSession()
SyncBotProfile()
```

- 构造函数：`New<Service>(opts ServiceOpts)` 或 `New<Component>(opts Options)`
- 返回 `bool` 时用语义化谓词：`IsActive()`、`HasPermission()`、`CanAccessProject()`
- 接受 `context.Context` 时，**第一个参数**为 `ctx`

### 变量

- 局部变量在上下文明确时可简短：`ctx`、`req`、`resp`、`err`
- 业务关键字段保持可读长度，不省略语义：

```go
projectID := ...
chainID := ...
telegramThreadID := ...
notificationTopic := ...
```

- 结构体未导出字段用驼峰小写开头，与导出字段区分

### 接口

- 单方法接口优先 `-er` 后缀：

```go
type Reader interface {
    ReadBlock(ctx context.Context, number uint64) (Block, error)
}

type Producer interface {
    Publish(ctx context.Context, topic string, key string, envelope Envelope) error
}
```

- 按能力命名，不按实现命名；不使用 `I` 前缀，避免 `XxxInterface`、`XxxImpl`

### 常量与枚举

- 未导出常量用驼峰；导出常量同样用驼峰，**不用**全大写 `SCREAMING_SNAKE_CASE`
- 类型化枚举值与类型同名前缀：

```go
const defaultPageSize = 50

const (
    NotificationTopicTrade NotificationTopic = "trade"
    NotificationTopicAlert NotificationTopic = "alert"
)
```

### 错误

- 包级 sentinel error 使用 `Err` 前缀：

```go
var ErrUserNotFound = errors.New("user not found")
var ErrCacheMiss = errors.New("cache miss")
```

- 错误名应说明**具体失败原因**，而非笼统状态：`ErrUnauthorized`、`ErrInvalidTopic`、`ErrProjectArchived`
- gRPC 层可使用 `status.Errorf` 返回领域错误（如 `ErrNoSession`），与 HTTP/业务层 sentinel 区分使用场景

### 缩写

- 常见缩写作为首字母缩写词时保持大写：`ID`、`URL`、`HTTP`、`API`、`JSON`、`RPC`

```go
userID
projectID
chainID
HTTPClient
APIKey
```

- 不要写成：`userId`、`httpClient`、`apiKey`

## ATHENA 项目约定

### 模块目录

业务模块位于 `internal/<module>/`，常见子目录：

| 目录 | 职责 |
| --- | --- |
| `api/` | gRPC service 实现 |
| `store/` | 数据访问（sqlc 生成代码 + 薄封装） |
| `apiclient/` | 模块 protobuf 生成代码 |
| `components/` | 可组合的领域组件 |
| `cache/`、`events/`、`workflows/` 等 | 按模块需要划分 |

`internal/server/` 为 API 网关与聚合层，通过 `apiclient` 调用各业务模块。

### 服务构造模式

```go
type Service struct { /* 未导出依赖字段 */ }

type ServiceOpts struct {
    Store   appstore.Store
    ChainID int64
    // ...
}

func NewService(opts ServiceOpts) (*Service, error) { /* ... */ }
```

- 对外暴露 `Service` + `NewService`；依赖通过 `ServiceOpts` / `Options` 注入，避免过长构造函数参数列表
- 未导出字段存放运行时状态与依赖，不在类型名中体现 `Impl`

### import 别名

跨模块或生成代码 import 时使用简短、稳定的别名，避免与本地包名冲突：

```go
appstore "github.com/useryege/athena/internal/application/store"
applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
servernotification "github.com/useryege/athena/internal/server/notification"
```

- 模块 protobuf 客户端包：`<module>pkg` 或 `pkg/apiclient/<module>` 对应别名
- `store` 层：`<module>store` 或 `appstore` 等能一眼识别来源的短名

### 测试

- 测试函数：`Test<FunctionName>_<scenario>`，例如 `TestIngestor_SkipsConfirmedBlocks`
- 表驱动测试优先；测试文件与被测源码同包（`package api`）或 `_test` 外部测试包按场景选择
- 测试数据与 helper 命名清晰，避免 `test1`、`data2`

## 反面示例速查

| 类别 | 不推荐 | 推荐 |
| --- | --- | --- |
| 包/类型重复 | `notification.NotificationService` | `notification.Service` |
| 空泛类型 | `Manager`、`Processor`、`Data` | `Ingestor`、`Dispatcher`、`Project` |
| 接口/实现 | `IUserRepository`、`UserRepositoryImpl` | `Repository`、`store.Store` |
| 方法 | `DoCreate`、`HandleThing`、`ExecuteTask` | `CreateUser`、`RouteMessage` |
| 缩写 | `userId`、`httpClient` | `userID`、`HTTPClient` |
| 常量 | `DEFAULT_PAGE_SIZE` | `defaultPageSize` |

## 格式与静态检查

代码格式与 import 顺序以工具为准，提交前应在本地通过检查：

```bash
gofmt -w <file>.go
goimports -w <file>.go
golangci-lint run
```

CI 会对 PR 执行 lint；常见失败原因包括未 `goimports` 格式化、lint 规则未通过。详见 [Static Code Analysis](developer-guide/static-code-analysis.md) 与 [CI 说明](developer-guide/ci.md)。
