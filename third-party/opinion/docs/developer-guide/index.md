# Opinion gRPC 开发指南

## 目的

本文档用于记录在 `third-party/opinion` 中新增或调整 gRPC 功能时的标准开发流程。

适用场景包括：

- 给现有 RPC 的请求或响应新增字段
- 新增一个 RPC 方法
- 调整 proto 定义后同步更新服务端实现

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

该命令会根据 `opinion/opinion.proto` 重新生成 `src/gen/opinion/opinion.ts`，并在使用 `import "google/protobuf/struct.proto"` 时同步生成 `src/gen/google/protobuf/struct.ts`（第二组 `--proto_path` 指向 `grpc-tools` 自带的 Well-Known Types）。

`package.json` 中对应脚本以仓库内实际内容为准；运行 `npm run proto:gen` 即可。

Market Query 的字段级映射与契约说明见 [`docs/references/sdk-to-proto-market-query.md`](../references/sdk-to-proto-market-query.md)。

### 3. 更新服务实现

生成代码后，根据变更内容更新 `src` 中的服务实现。
