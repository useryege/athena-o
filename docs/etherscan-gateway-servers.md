# Etherscan Gateway Servers

本文记录 Etherscan Gateway 当前使用的五台服务器。服务器顺序与 `.env`
中的 `ETHERSCAN_GATEWAY_IPS` 配置一一对应：

```bash
ETHERSCAN_GATEWAY_IPS='47.245.183.140 47.245.166.57 47.245.161.139 47.245.181.189 47.254.154.128'
```

## 服务器清单

| 顺序 | 服务器 | IP |
| --- | --- | --- |
| 1 | LXM | `47.245.183.140` |
| 2 | HYD | `47.245.166.57` |
| 3 | SYY | `47.245.161.139` |
| 4 | LDM | `47.245.181.189` |
| 5 | SHL | `47.254.154.128` |

## 部署信息

- 部署入口：`make deploy-etherscan-gateway-vps`
- 部署脚本：`hack/deploy-etherscan-gateway.sh`
- systemd 服务名：`athena-etherscan-gateway`
- 监听地址：`0.0.0.0:6776`
- 远程二进制：`/usr/local/bin/athena-etherscan-gateway`
- 远程环境文件：`/etc/athena/etherscan-gateway.env`

部署脚本会按 `ETHERSCAN_GATEWAY_IPS` 中的 IP 顺序依次发布
`athena-etherscan-gateway`，并在每台服务器上启用 systemd 服务。

`athena-ethereum-api` 运行时不直接读取 `ETHERSCAN_GATEWAY_IPS`。它使用
显式 `host:port` 列表：

```bash
ATHENA_ETHEREUM_API_ETHERSCAN_GATEWAY_ADDRS='47.245.183.140:6776 47.245.166.57:6776 47.245.161.139:6776 47.245.181.189:6776 47.254.154.128:6776'
```

Athena 前端的 `/etherscan-gateways` 状态页由 `athena-server` 聚合展示。该页面读取
`ETHERSCAN_GATEWAY_IPS` 并按默认端口 `6776` 调用各 gateway 的
`GetEtherscanGatewayStatus` gRPC 接口；认证 token 来自
`ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN`，不会暴露给浏览器。
