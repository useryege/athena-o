好的，我帮你梳理一遍完整教程，从 **排查 C 盘爆满 → 清理 Docker → 压缩/迁移 WSL2 → 设置默认发行版 → 避免未来占用**，步骤清晰、按顺序执行即可。

---

# **Windows C 盘爆满 + Docker/WSL2 解决完整教程**

---

## **步骤 0：检查 C 盘空间**

在 PowerShell 中：

```powershell
Get-PSDrive
```

* 查看 C 盘已用和剩余空间
* 如果空间 <20GB，就必须腾出空间

---

## **步骤 1：检查 Docker 占用**

```powershell
docker system df
```

输出示例：

| 类型            | 占用大小   | 可回收   |
| ------------- | ------ | ----- |
| Images        | 25 GB  | 4 GB  |
| Containers    | 70 KB  | 0B    |
| Local Volumes | 758 MB | 0B    |
| Build Cache   | 12 GB  | 12 GB |

---

## **步骤 2：清理 Docker**

### 2.1 清理 Build Cache（释放最多空间）

```powershell
docker builder prune
```

或者彻底清理所有未使用资源：

```powershell
docker system prune -a --volumes
```

> ⚠️ 注意：会删除所有未使用镜像、容器和卷，请先确认重要数据备份

---

### 2.2 避免以后 Build Cache 堆积

构建镜像时加 `--no-cache`：

```powershell
docker build --no-cache -t myimage .
```

---

## **步骤 3：识别 WSL2 占用（.vhdx 文件）**

* WSL2 虚拟磁盘位置：

```text
C:\Users\<用户名>\AppData\Local\Packages\<发行版>\LocalState\ext4.vhdx
```

* Docker Desktop 默认在：

```text
C:\Users\<用户名>\AppData\Local\Docker\wsl\data\ext4.vhdx
```

* `.vhdx` 文件很大（几十 GB），C 盘爆满的主要原因。

---

## **步骤 4：压缩或迁移 WSL2 虚拟磁盘（推荐方法：导出 → 注销 → 导入）**

### 4.1 关闭 WSL2

```powershell
wsl --shutdown
```

确保所有发行版状态为 `Stopped`：

```powershell
wsl --list --verbose
```

---

### 4.2 导出 Ubuntu-24.04 到其他盘（例如 E 盘）

```powershell
wsl --export Ubuntu-24.04 E:\wsl\Ubuntu-24.04.tar
```

* `.tar` 文件只占实际数据大小，比原 `.vhdx` 小很多

---

### 4.3 注销旧发行版（释放 C 盘空间）

```powershell
wsl --unregister Ubuntu-24.04
```

* 删除原来的 `.vhdx` 文件
* C 盘立即腾出几十 GB

---

### 4.4 导入到新位置（E 盘）

```powershell
wsl --import Ubuntu-24.04 E:\wsl\Ubuntu-24.04 E:\wsl\Ubuntu-24.04.tar --version 2
```

* 新 `.vhdx` 文件会根据实际数据生成
* 导入完成后，C 盘不再被占满

---

### 4.5 删除临时导出文件

```powershell
Remove-Item E:\wsl\Ubuntu-24.04.tar
```

---

## **步骤 5：设置默认 WSL2 发行版**

```powershell
wsl --setdefault Ubuntu-24.04
```

确认：

```powershell
wsl --list --verbose
```

* `*` 标记显示默认发行版为 Ubuntu-24.04

---

## **步骤 6：迁移 Docker Desktop Disk Image（防止未来占用 C 盘）**

1. 打开 Docker Desktop → **Settings → Resources → Advanced → Disk image location**
2. 指向 D/E 盘新路径
3. 保存，重启 Docker Desktop
4. 以后 Docker 镜像、容器都会存储在新盘

---

## **步骤 7：验证空间**

1. 检查 C 盘空间：

```powershell
Get-PSDrive
```

2. 检查 Docker 和 WSL2：

```powershell
docker system df
wsl --list --verbose
```

* 确保 C 盘有几十 GB 空间，Docker 镜像和 WSL2 `.vhdx` 已在新盘

---

💡 **小技巧总结**：

* Docker Build Cache 清理 → 立即释放 10~12 GB
* `.vhdx` 导出 → 注销 → 导入 → 真正释放几十 GB
* Docker Disk Image 指向非 C 盘 → 避免未来爆满
* WSL 默认发行版切换到 Ubuntu-24.04 → 启动方便

---

如果你需要，我可以帮你写一个 **一键 PowerShell 脚本**，自动完成：

1. 停止 WSL2
2. 清理 Docker Build Cache
3. 导出 Ubuntu-24.04 → 注销 → 导入到 E 盘
4. 删除临时 `.tar`
5. 设置默认 WSL2

这样你只需执行一次就完成整个迁移和清理。

你希望我帮你写吗？
