# `make run` 排障：Redis 正常、PostgreSQL 启动失败

## 场景

本地执行 `make run` 时，出现以下现象：

- `redis` 进程最终正常启动
- `postgres` 进程退出
- 日志包含权限错误：`mkdir: cannot create directory '/tmp/athena-local': Permission denied`

## 典型日志

```text
redis    | mkdir: cannot create directory '/tmp/athena-local': Permission denied
redis    | Starting Docker container with password.
postgres | mkdir: cannot create directory '/tmp/athena-local': Permission denied
postgres | Terminating postgres
```

## 根因分析

这类问题通常由以下三个条件叠加触发：

1. 开启了本地持久化模式

   `.env` 中存在：

   ```env
   ATHENA_LOCAL_DATA_MODE=persistent
   ```

   持久化模式下，Redis/PostgreSQL 都会尝试在宿主机创建数据目录（默认在 `/tmp/athena-local` 下）。

2. `/tmp/athena-local` 目录权限异常

   例如目录被 root 创建且权限为 `700`：

   ```text
   drwx------ root root /tmp/athena-local
   ```

   这会导致普通用户无法在其下创建 `postgres` 或 `redis` 子目录。

3. Redis 与 PostgreSQL 脚本的失败策略不同

   - PostgreSQL 启动脚本使用了 `set -euo pipefail`，`mkdir` 失败会直接退出。
   - Redis 启动脚本未启用 `set -e`，即使 `mkdir` 失败仍可能继续执行 `docker run`。

因此会出现“同样报权限错，但 Redis 继续启动、PostgreSQL 直接失败”的差异。

## 快速诊断

在项目根目录执行：

```bash
grep -n '^ATHENA_LOCAL_DATA_MODE' .env
ls -ld /tmp /tmp/athena-local
stat -c '%A %a %U:%G %n' /tmp /tmp/athena-local
```

如果 `ATHENA_LOCAL_DATA_MODE=persistent` 且 `/tmp/athena-local` 为 `root:root 700`，即可确认该问题。

## 修复方案

### 方案 1（推荐）：修复目录权限

```bash
sudo chown -R "$USER":"$USER" /tmp/athena-local
```

然后重新执行：

```bash
make run
```

### 方案 2：切换为临时模式（不落盘）

将 `.env` 改为：

```env
ATHENA_LOCAL_DATA_MODE=ephemeral
```

再执行：

```bash
make run
```

### 方案 3：改用项目内可写目录

```bash
export ATHENA_POSTGRES_DATA_DIR="$PWD/.local/postgres"
export ATHENA_REDIS_DATA_DIR="$PWD/.local/redis"
make run
```

## 补充说明

- `make run` 仅调用 `goreman start`，不会自动清理或重建 `/tmp/athena-local`。
- 若你希望每次都自动准备本地目录，可考虑使用 `make start-local` 流程，或在本地脚本中增加目录初始化步骤。
