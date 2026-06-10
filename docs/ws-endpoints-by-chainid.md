# WebSocket 节点按 Chain ID 分类

> 数据来源：用户提供的节点扫描日志  
> 处理方式：按 `chain id` 分组，并对重复 `url` 去重  
> 链名称来源：[chainid.network](https://chainid.network)

## Chain ID 对照表

| Chain ID | 区块链 | 类型 | 原生代币 |
|---------|--------|------|---------|
| 1 | Ethereum（以太坊主网） | 主网 | ETH |
| 10 | OP Mainnet（Optimism） | 主网 | ETH |
| 56 | BNB Smart Chain（BSC） | 主网 | BNB |
| 96 | Bitkub Chain | 主网 | KUB |
| 100 | Gnosis Chain | 主网 | XDAI |
| 128 | Huobi ECO Chain（HECO，已停服） | 主网 | HT |
| 137 | Polygon PoS | 主网 | POL |
| 171 | CO2e Chain | 主网 | CO2E |
| 252 | Fraxtal | 主网 | FRAX |
| 648 | Endurance Smart Chain | 主网 | ACE |
| 784 | 未在公开链表登记 | 未知 | — |
| 1088 | Metis Andromeda | 主网 | METIS |
| 1112 | WEMIX 3.0 Testnet | 测试网 | tWEMIX |
| 1516 | Story Odyssey Testnet | 测试网 | IP |
| 2222 | Kava EVM | 主网 | KAVA |
| 3501 | JFIN Chain | 主网 | JFIN |
| 5000 | Mantle | 主网 | MNT |
| 8453 | Base | 主网 | ETH |
| 42220 | Celo | 主网 | CELO |
| 80094 | Berachain | 主网 | BERA |
| 88888 | Chiliz Chain | 主网 | CHZ |
| 100011 | QuarkChain L2 | 主网 | QKC |
| 258432 | Althea L1 | 主网 | ALTHEA |
| 845300 | 未在公开链表登记 | 未知 | — |
| 11155111 | Ethereum Sepolia | 测试网 | ETH |

## Chain ID 1 — Ethereum 主网

> 连通性检测：2026-06-10；方法：WebSocket 握手 + `eth_chainId` RPC；10/12 可用

- `ws://65.108.72.217:8546`
- `ws://65.108.75.55:8546`
- `ws://65.108.79.174:8546`
- `ws://65.108.108.60:8546`
- `ws://65.108.129.183:8546`
- `ws://65.108.132.84:8546`
- `ws://65.108.133.93:8546`
- `ws://65.108.142.146:8546` — **不可用**（connection refused）
- `ws://65.108.199.204:8546`
- `ws://65.108.202.46:8546`
- `ws://65.108.230.161:8546`
- `ws://65.108.233.209:8546` — **不可用**（connection refused）

## Chain ID 10 — OP Mainnet（Optimism）

- `ws://65.108.33.57:8546`

## Chain ID 56 — BNB Smart Chain（BSC）

> 连通性检测：2026-06-10；方法：WebSocket 握手 + `eth_chainId` RPC；2/3 可用

- `ws://65.108.34.81:8546`
- `ws://65.108.76.31:8546` — **不可用**（connection refused）
- `ws://65.108.192.118:8546`

## Chain ID 96 — Bitkub Chain

- `ws://65.108.226.232:8546`

## Chain ID 100 — Gnosis Chain

- `ws://65.108.206.150:8546`

## Chain ID 128 — Huobi ECO Chain（HECO，已停服）

- `ws://65.108.11.40:8546`

## Chain ID 137 — Polygon PoS

- `ws://65.108.120.181:8546`

## Chain ID 171 — CO2e Chain

- `ws://65.108.126.254:8546`

## Chain ID 252 — Fraxtal

- `ws://65.108.69.56:8546`

## Chain ID 648 — Endurance Smart Chain

- `ws://65.108.78.48:8546`

## Chain ID 784 — 未在公开链表登记

- `ws://65.108.238.29:8546`

## Chain ID 1088 — Metis Andromeda

- `ws://65.108.237.224:8546`

## Chain ID 1112 — WEMIX 3.0 Testnet

- `ws://65.108.6.166:8546`

## Chain ID 1516 — Story Odyssey Testnet

- `ws://65.108.229.11:8546`

## Chain ID 2222 — Kava EVM

- `ws://65.108.103.254:8546`

## Chain ID 3501 — JFIN Chain

- `ws://65.108.0.235:8546`
- `ws://65.108.14.187:8546`

## Chain ID 5000 — Mantle

- `ws://65.108.232.245:8546`

## Chain ID 8453 — Base

- `ws://65.108.9.159:8546`
- `ws://65.108.122.15:8546`
- `ws://65.108.198.118:8546`
- `ws://65.108.232.180:8546`
- `ws://65.108.233.57:8546`

## Chain ID 42220 — Celo

- `ws://65.108.74.96:8546`

## Chain ID 80094 — Berachain

- `ws://65.108.6.17:8546`

## Chain ID 88888 — Chiliz Chain

- `ws://65.108.110.39:8546`

## Chain ID 100011 — QuarkChain L2

- `ws://65.108.236.27:8546`

## Chain ID 258432 — Althea L1

- `ws://65.108.227.207:8546`

## Chain ID 845300 — 未在公开链表登记

- `ws://65.108.151.162:8546`

## Chain ID 11155111 — Ethereum Sepolia

- `ws://65.108.230.142:8546`
- `ws://65.108.232.214:8546`
