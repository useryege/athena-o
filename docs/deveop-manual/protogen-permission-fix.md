# `make protogen` 权限问题快速修复

## 场景

执行 `make protogen` 时出现类似错误：

```text
go: .../tmp/athena-local/postgres: open ... permission denied
go: .../tmp/athena-local/redis/appendonlydir: open ... permission denied
```

常见原因是 `tmp/athena-local` 下目录由容器用户（如 UID 999）创建，且权限为 `700`，当前用户无法遍历目录，导致 `go mod tidy` 失败。

## 一键修复步骤

在项目根目录执行：

```bash
cd /home/yege/work/athena
sudo chown -R "$USER":"$USER" tmp/athena-local
find tmp/athena-local -type d -exec chmod u+rwx {} \;
```

然后重试：

```bash
make protogen
```

## 验证

可选执行以下命令确认权限已恢复：

```bash
ls -ld tmp/athena-local tmp/athena-local/postgres tmp/athena-local/redis tmp/athena-local/redis/appendonlydir
```

确认目录 owner 为当前用户，并且不再是仅 UID 999 可访问的状态。

## 说明

- 该修复会保留已有本地数据，只调整 owner 和目录权限。
- 如果你不需要保留本地数据，也可以删除 `tmp/athena-local/postgres` 和 `tmp/athena-local/redis` 后重建。
