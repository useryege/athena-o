# How to Develop

This project follows a compact local flow: run the app locally, update generated files when needed, and use the production compose path before deployment.

新业务服务或服务边界改造先遵循[服务开发规范](service-development-standards.md)：设计中记录 owner、独立入口、必要依赖、配置/授权、事务与故障边界；PR 按实际影响提供证据。当前可用的仓库级命令仅以本页列出的命令及[本地运行编排](../design/development-runtime/local-runtime-orchestration.md)为准，未实现的局部 `run-service` 等命令不得当作现有能力。目标是让独立服务具备可验证的局部构建、启动、测试和停止入口，并只回收其拥有的资源。

## Recommended Workflow

### 1. Before Development

Install the code generation tools:

```bash
make install-codegen-tools-local
```

For UI work, install dependencies directly in the UI directory:

```bash
cd ui
yarn install
```

### 2. During Development

Run Athena locally:

```bash
make run
```

`make run` 显式选择全栈图，默认实例名为 `full-stack`。开发 Trader Sync 时用 `make run-service SERVICE=trader-sync INSTANCE=ts-dev`，只准备 TS 与该实例 PostgreSQL；组合联调用 `run-services` 正向选择。通过 `runtime-status` 核对归属，以 `stop-instance` 停止并保留数据；不调用全栈 reset 作为局部清理。

Run the UI only:

```bash
cd ui
yarn start
```

Maintain documentation directly as repository Markdown. Follow the upstream [Superpowers workflow](superpowers-development.md) for discovery, design, planning, TDD, execution, and review. Read the relevant [requirements](../requirements/README.md), [system designs](../design/README.md), and source code as project context. Save task specs and plans under `docs/superpowers/specs/` and `docs/superpowers/plans/`, then update the affected long-term project documents as part of the work.

### 3. Generated Files

Run code generation when you change protobufs, API types, command docs, SQL sources, or similar generated inputs:

```bash
make codegen-local
git status
git diff
```

### 4. Production-Like Check

Use the production image and compose flow before deployment:

```bash
make prod-build-local
make prod-start-local
```

View logs or stop the local production compose stack with:

```bash
make prod-logs-local
make prod-stop-local
```
