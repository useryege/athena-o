# Etherscan Gateway Servers

本文记录 Etherscan Gateway 当前使用的三台服务器。服务器顺序与 `.env`
中的 `ETHERSCAN_GATEWAY_IPS` 配置一一对应：

```bash
ETHERSCAN_GATEWAY_IPS='47.245.183.140 47.245.166.57 47.245.161.139'
```

## 服务器清单

| 顺序 | 服务器 | IP |
| --- | --- | --- |
| 1 | LXM | `47.245.183.140` |
| 2 | HYD | `47.245.166.57` |
| 3 | SYY | `47.245.161.139` |

## 部署信息

- 部署入口：`make deploy-etherscan-gateway-vps`
- 部署脚本：`hack/deploy-etherscan-gateway.sh`
- systemd 服务名：`athena-etherscan-gateway`
- 监听地址：`0.0.0.0:6776`
- 远程二进制：`/usr/local/bin/athena-etherscan-gateway`
- 远程环境文件：`/etc/athena/etherscan-gateway.env`

部署脚本会按 `ETHERSCAN_GATEWAY_IPS` 中的 IP 顺序依次发布
`athena-etherscan-gateway`，并在每台服务器上启用 systemd 服务。
