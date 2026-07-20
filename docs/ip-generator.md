# IP Generator 使用说明

`ip-generator` 用于扫描固定 IPv4 `/16` 网段中的以太坊兼容 WebSocket JSON-RPC 节点。程序连接每个地址的 `8546` 端口，读取 Chain ID 和最新区块高度，并仅输出两项查询都成功的节点。

请只扫描自己拥有或已获得授权的网络范围。

## 扫描范围

`-first` 和 `-second` 分别指定 IPv4 的前两个字段。例如输入 `65` 和 `108` 时，程序扫描：

```text
ws://65.108.0.1:8546
...
ws://65.108.255.254:8546
```

每次扫描共包含 65,534 个地址，不包含 `/16` 网络地址 `65.108.0.0` 和广播地址 `65.108.255.255`。

## 本地运行

在仓库根目录执行：

```bash
go run ./tools/ip-generator -first 65 -second 108
```

完整参数示例：

```bash
go run ./tools/ip-generator \
  -first 65 \
  -second 108 \
  -workers 256 \
  -timeout 3s
```

| 参数 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `-first` | 是 | 无 | IPv4 第一个字段，范围为 `0–255` |
| `-second` | 是 | 无 | IPv4 第二个字段，范围为 `0–255` |
| `-workers` | 否 | `256` | 同时探测的地址数量，必须大于 `0` |
| `-timeout` | 否 | `3s` | 单地址连接和全部 RPC 调用的总超时，必须大于 `0` |

每个地址会依次建立 WebSocket 连接、调用 `eth_chainId`，然后调用 `eth_blockNumber`。任一步骤失败，该地址都不会出现在标准输出中。

## 输出格式

成功结果每行包含 WebSocket URL、Chain ID 和最新区块高度：

```text
ws://65.108.72.217:8546 1 22987654
```

结果按照探测完成顺序输出，不保证按照 IP 地址排序。扫描结束后，标准错误会输出汇总：

```text
scanned=65534 successful=12 failed=65522
```

只保存成功结果：

```bash
go run ./tools/ip-generator -first 65 -second 108 > results.txt
```

同时保存成功结果和扫描汇总：

```bash
go run ./tools/ip-generator -first 65 -second 108 \
  > results.txt \
  2> scan.log
```

按 `Ctrl+C` 可以取消扫描；程序会等待已派发任务结束并输出已完成部分的汇总，然后以非零状态退出。

## 编译并上传到远程服务器

部署脚本固定生成无 CGO 依赖的 Linux AMD64 二进制，并直接上传到远程服务器。指定远程主机：

```bash
make deploy-ip-generator REMOTE_HOST=65.108.72.217
```

也可以在仓库 `.env` 中配置：

```bash
REMOTE_HOST=65.108.72.217
```

然后执行：

```bash
make deploy-ip-generator
```

默认远程用户为 `root`，默认上传位置为 `/root/ip-generator`。可以通过 Make 变量覆盖：

```bash
make deploy-ip-generator \
  REMOTE_HOST=65.108.72.217 \
  REMOTE_USER=root \
  IP_GENERATOR_REMOTE_PATH=/root/ip-generator
```

上传完成后，在远程服务器运行：

```bash
/root/ip-generator -first 65 -second 108
```

## 相关文档

- [Hetzner IPv4 常见前缀](hetzner-ip-prefixes.md)
- [WebSocket 节点按 Chain ID 分类](ws-endpoints-by-chainid.md)
