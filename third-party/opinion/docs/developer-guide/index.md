# Opinion gRPC 开发指南

## 目的

本文档用于记录在 `third-party/opinion` 中新增或调整 gRPC 功能时的标准开发流程。

适用场景包括：

- 给现有 RPC 的请求或响应新增字段
- 新增一个 RPC 方法
- 调整 proto 定义后同步更新服务端实现

## 目录结构

新增 gRPC 功能时，通常会涉及以下文件：

- `opinion/opinion.proto`：gRPC 协议定义
- `src/gen/opinion/opinion.ts`：根据 proto 生成的 TypeScript 类型与 service 定义
- `src/service.ts`：gRPC service 的实际处理逻辑
- `src/mappers.ts`：SDK 返回结果到 proto 响应结构的映射
- `src/client.ts`：Opinion SDK client 封装
- `cmd/server/main.ts`：gRPC server 启动入口
- `scripts/get-markets.ts`：当前已有的 gRPC 调试脚本示例

## 标准流程

### 1. 修改 proto

首先修改 `opinion/opinion.proto`，完成以下一种或多种变更：

- 新增 message 字段
- 新增 enum 值
- 新增 request/response message
- 新增 service rpc 方法

建议注意：

- 已发布字段不要复用原有 tag 编号
- 新字段使用新的 field number
- 尽量保持命名与现有风格一致

### 2. 重新生成 TypeScript 代码

在 `third-party/opinion` 目录下执行：

```bash
npm run proto:gen
```

该命令会根据 `opinion/opinion.proto` 重新生成 `src/gen/opinion/opinion.ts`。

`package.json` 中对应脚本如下：

```json
{
  "scripts": {
    "proto:gen": "mkdir -p src/gen && grpc_tools_node_protoc --plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto --ts_proto_out=src/gen --ts_proto_opt=outputServices=grpc-js,esModuleInterop=true,importSuffix=.js,forceLong=string --proto_path=. opinion/opinion.proto"
  }
}
```

### 3. 更新服务实现

生成代码后，根据变更内容更新 `src/service.ts`。

#### 场景 A：现有 RPC 只新增字段

如果只是为现有 RPC 增加请求字段或响应字段，通常需要：

- 从 `call.request` 中读取新增字段
- 将新增字段转换为底层 SDK 所需参数
- 在响应中补齐新增字段

这类改动通常集中在：

- 请求参数校验
- 枚举值转换
- SDK 调用参数构造
- callback 返回的响应组装

#### 场景 B：新增一个 RPC 方法

如果新增了全新的 rpc，例如：

```proto
rpc GetSomething(GetSomethingRequest) returns (GetSomethingResponse);
```

那么在 `src/service.ts` 中还必须新增对应 handler。仅修改 proto 并生成代码还不够，否则 server 虽然有类型定义，但没有实际实现。

处理时通常需要：

1. 定义请求参数转换逻辑
2. 调用 `getOpinionClient()` 获取 SDK client
3. 调用底层 SDK 方法
4. 将 SDK 返回值映射成 proto response
5. 将异常转换为标准 gRPC error

### 4. 更新响应映射

如果 proto response 结构有变化，或者新增了响应 message，继续更新 `src/mappers.ts`。

这里主要负责：

- 将 SDK 返回的原始对象转换成 proto 定义的结构
- 做必要的类型兜底
- 保持返回值字段名与生成后的 TS 类型一致

如果新增了新的 response message，通常需要新增一个 mapper 函数。

### 5. 视情况更新底层 client 封装

如果现有 `src/client.ts` 已经能满足需求，可以不改。

如果新增功能依赖新的 SDK 调用方式，或者需要补充封装逻辑，则在这里调整：

- 新增或调整 SDK 调用入口
- 复用统一配置
- 避免在 `service.ts` 里直接堆积过多底层接入细节

### 6. 本地验证

完成开发后，至少执行以下检查：

```bash
npm run typecheck
npm run build
```

如果需要联调 gRPC 服务，可启动服务：

```bash
npm run dev
```

当前项目还提供了一个调试脚本示例：

```bash
npm run grpc:test
```

该脚本当前演示的是 `GetMarkets` 调用方式。新增 RPC 后，建议参考现有脚本增加新的调试脚本，或扩展现有脚本以覆盖新的接口。

## 推荐开发顺序

推荐按以下顺序工作，避免漏改：

1. 修改 `opinion/opinion.proto`
2. 执行 `npm run proto:gen`
3. 检查生成结果是否符合预期
4. 更新 `src/service.ts`
5. 更新 `src/mappers.ts`
6. 必要时更新 `src/client.ts`
7. 执行 `npm run typecheck`
8. 执行 `npm run build`
9. 使用调试脚本或本地联调验证行为

## 快速判断：我这次要改哪些文件

### 只给现有 RPC 新增字段

通常需要改：

- `opinion/opinion.proto`
- `src/service.ts`
- `src/mappers.ts`
- `src/gen/opinion/opinion.ts`（通过 `npm run proto:gen` 自动生成）

### 新增一个全新 RPC

通常需要改：

- `opinion/opinion.proto`
- `src/service.ts`
- `src/mappers.ts`（如果有新的返回结构）
- `src/client.ts`（如果需要新的 SDK 调用封装）
- `scripts/*.ts`（建议补一个调试脚本）
- `src/gen/opinion/opinion.ts`（通过 `npm run proto:gen` 自动生成）

## 常见问题

### 修改完 proto 后，下一步是不是执行 `proto:gen`？

是。正确做法是在 `third-party/opinion` 目录下执行：

```bash
npm run proto:gen
```

不是对 `package.json` 文件本身执行操作，而是执行其中定义的脚本。

### 为什么生成完代码后还要改 `src/service.ts`？

因为 `proto:gen` 只会生成类型和 service 定义，不会自动生成你的业务实现。真正的 gRPC handler 仍然需要在 `src/service.ts` 中手动补齐。

### 为什么有时还要改 `src/mappers.ts`？

因为底层 SDK 返回结构和 proto response 结构不一定完全一致，通常需要单独做一层映射，保证对外 gRPC 协议稳定且清晰。

## 维护建议

- 每次修改 proto 后，尽快执行一次 `npm run proto:gen`
- 不要手改 `src/gen/opinion/opinion.ts`
- 生成文件只作为产物维护，真正逻辑放在 `src/service.ts`、`src/mappers.ts`、`src/client.ts`
- 新增 RPC 时，尽量同步补一个最小可运行的调试脚本，方便回归验证