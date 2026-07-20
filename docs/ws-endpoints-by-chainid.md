# WebSocket 节点按 Chain ID 分类

> 数据来源：用户提供的节点扫描日志  
> 处理方式：按 `chain id` 分组，并对重复 `url` 去重  
> 链名称来源：[chainid.network](https://chainid.network) 及相关链官方文档

## Chain ID 对照表

| Chain ID | 区块链 | 类型 | 原生代币 |
|---------|--------|------|---------|
| 1 | Ethereum（以太坊主网） | 主网 | ETH |
| 7 | ThaiChain | 主网 | TCH |
| 10 | OP Mainnet（Optimism） | 主网 | ETH |
| 56 | BNB Smart Chain（BSC） | 主网 | BNB |
| 75 | Decimal Smart Chain Mainnet | 主网 | DEL |
| 96 | Bitkub Chain | 主网 | KUB |
| 97 | BNB Smart Chain Testnet | 测试网 | tBNB |
| 100 | Gnosis Chain | 主网 | XDAI |
| 110 | Proton Testnet | 测试网 | XPR |
| 117 | Uptick Mainnet | 主网 | UPTICK |
| 128 | Huobi ECO Chain（HECO，已停服） | 主网 | HT |
| 137 | Polygon PoS | 主网 | POL |
| 169 | Manta Pacific Mainnet | 主网 | ETH |
| 171 | CO2e Chain | 主网 | CO2E |
| 196 | X Layer | 主网 | OKB |
| 223 | B2 Mainnet | 主网 | BTC |
| 252 | Fraxtal | 主网 | FRAX |
| 298 | Hedera Localnet | 本地网络 | HBAR |
| 369 | PulseChain | 主网 | PLS |
| 648 | Endurance Smart Chain | 主网 | ACE |
| 784 | 未在公开链表登记 | 未知 | — |
| 943 | PulseChain Testnet v4 | 测试网 | tPLS |
| 999 | Wanchain Testnet | 测试网 | WAN |
| 1088 | Metis Andromeda | 主网 | METIS |
| 1112 | WEMIX 3.0 Testnet | 测试网 | tWEMIX |
| 1329 | Sei Network | 主网 | SEI |
| 1337 | Geth Testnet | 测试网 | ETH |
| 1439 | Injective Testnet | 测试网 | INJ |
| 1516 | Story Odyssey Testnet | 测试网 | IP |
| 1708 | TBSI Testnet | 测试网 | JINDA |
| 1776 | Injective | 主网 | INJ |
| 2222 | Kava EVM | 主网 | KAVA |
| 3300 | Realio Testnet | 测试网 | RIO |
| 3301 | Realio | 主网 | RIO |
| 3501 | JFIN Chain | 主网 | JFIN |
| 3502 | JFINPOS | 主网 | JPOS |
| 5000 | Mantle | 主网 | MNT |
| 5858 | Chang Chain Foundation Mainnet | 主网 | CTH |
| 6480 | 未在公开链表登记 | 未知 | — |
| 7117 | 0XL3 | 主网 | XL3 |
| 7269 | 未在公开链表登记 | 未知 | — |
| 8163 | 未在公开链表登记 | 未知 | — |
| 8453 | Base | 主网 | ETH |
| 9008 | Shido Network | 主网 | SHIDO |
| 9069 | Apex Fusion - Nexus Mainnet | 主网 | AP3X |
| 9555 | 未在公开链表登记 | 未知 | — |
| 9601 | KUB Layer 2 Mainnet | 主网 | KUB |
| 9790 | Carbon EVM | 主网 | SWTH |
| 9999 | myOwn Testnet | 测试网 | MYN |
| 10618 | 未在公开链表登记 | 未知 | — |
| 13373 | 未在公开链表登记 | 未知 | — |
| 16601 | 0G Galileo Testnet（已弃用） | 测试网 | A0GI |
| 35147 | CoinAfrica | 主网 | COINA |
| 35935 | 未在公开链表登记 | 未知 | — |
| 42069 | pegglecoin | 主网 | peggle |
| 42220 | Celo | 主网 | CELO |
| 42431 | Tempo Testnet Moderato | 测试网 | USD |
| 59902 | Metis Sepolia Testnet | 测试网 | tMETIS |
| 80069 | Berachain Bepolia | 测试网 | BERA |
| 80094 | Berachain | 主网 | BERA |
| 84532 | Base Sepolia Testnet | 测试网 | ETH |
| 88888 | Chiliz Chain | 主网 | CHZ |
| 100011 | QuarkChain L2 | 主网 | QKC |
| 110011 | QuarkChain L2 Testnet | 测试网 | QKC |
| 123321 | Gemchain | 主网 | GEM |
| 167011 | 未在公开链表登记 | 未知 | — |
| 258432 | Althea L1 | 主网 | ALTHEA |
| 480001 | MVP CHAIN | 主网 | MVP |
| 560048 | Ethereum Hoodi | 测试网 | ETH |
| 713714 | Vordium Chain | 主网 | VORD |
| 845300 | 未在公开链表登记 | 未知 | — |
| 853211 | Testethiq | 测试网 | ETH |
| 2026777 | 未在公开链表登记 | 未知 | — |
| 11155111 | Ethereum Sepolia | 测试网 | ETH |
| 65100004 | Autonity Piccadilly (Tiber) Testnet（已弃用） | 测试网 | ATN |
| 108160679 | Oraichain Mainnet | 主网 | ORAI |
| 383414847825 | Zeniq | 主网 | ZENIQ |

## Chain ID 1 — Ethereum 主网

> 历史连通性检测：2026-06-10；方法：WebSocket 握手 + `eth_chainId` RPC；当时收录的 12 个节点中 10 个可用

- `ws://49.12.19.226:8546`
- `ws://49.12.58.246:8546`
- `ws://49.12.58.248:8546`
- `ws://49.12.58.251:8546`
- `ws://49.12.83.145:8546`
- `ws://49.12.123.150:8546`
- `ws://65.21.12.175:8546`
- `ws://65.21.45.43:8546`
- `ws://65.21.80.13:8546`
- `ws://65.21.93.190:8546`
- `ws://65.21.133.53:8546`
- `ws://65.108.72.217:8546`
- `ws://65.108.75.55:8546`
- `ws://65.108.79.174:8546`
- `ws://65.108.105.28:8546`
- `ws://65.108.108.60:8546`
- `ws://65.108.129.183:8546`
- `ws://65.108.132.84:8546`
- `ws://65.108.133.93:8546`
- `ws://65.108.142.146:8546` — **不可用**（connection refused）
- `ws://65.108.199.204:8546`
- `ws://65.108.202.46:8546`
- `ws://65.108.230.161:8546`
- `ws://65.108.233.209:8546` — **不可用**（connection refused）
- `ws://65.109.30.13:8546`
- `ws://65.109.33.234:8546`
- `ws://65.109.38.15:8546`
- `ws://65.109.53.250:8546`
- `ws://65.109.82.250:8546`
- `ws://65.109.112.157:8546`
- `ws://65.109.121.144:8546`
- `ws://65.109.145.251:8546`
- `ws://78.46.37.162:8546`
- `ws://78.46.80.172:8546`
- `ws://95.216.0.93:8546`
- `ws://116.202.213.113:8546`
- `ws://116.202.218.100:8546`
- `ws://116.202.233.23:8546`
- `ws://135.181.6.56:8546`
- `ws://135.181.22.123:8546`
- `ws://135.181.57.120:8546`
- `ws://135.181.103.235:8546`
- `ws://135.181.177.35:8546`
- `ws://135.181.232.241:8546`
- `ws://135.181.238.121:8546`

## Chain ID 7 — ThaiChain

- `ws://49.12.43.108:8546`
- `ws://116.202.106.98:8546`
- `ws://135.181.193.101:8546`

## Chain ID 10 — OP Mainnet（Optimism）

- `ws://65.21.195.31:8546`
- `ws://65.108.33.57:8546`
- `ws://65.109.117.124:8546`

## Chain ID 56 — BNB Smart Chain（BSC）

- `ws://65.21.25.93:8546`
- `ws://65.108.34.81:8546`
- `ws://65.108.76.31:8546`
- `ws://65.108.192.118:8546`
- `ws://65.109.55.45:8546`
- `ws://65.109.116.104:8546`
- `ws://94.130.66.19:8546`
- `ws://94.130.131.59:8546`
- `ws://95.217.141.77:8546`
- `ws://116.202.50.179:8546`
- `ws://135.181.164.132:8546`
- `ws://135.181.247.212:8546`

## Chain ID 75 — Decimal Smart Chain Mainnet

- `ws://78.46.35.174:8546`

## Chain ID 96 — Bitkub Chain

- `ws://65.108.226.232:8546`

## Chain ID 97 — BNB Smart Chain Testnet

- `ws://65.109.77.111:8546`
- `ws://65.109.148.161:8546`

## Chain ID 100 — Gnosis Chain

- `ws://65.108.206.150:8546`
- `ws://95.216.69.143:8546`

## Chain ID 110 — Proton Testnet

- `ws://95.217.236.160:8546`

## Chain ID 117 — Uptick Mainnet

- `ws://95.217.114.60:8546`

## Chain ID 128 — Huobi ECO Chain（HECO，已停服）

- `ws://65.108.11.40:8546`

## Chain ID 137 — Polygon PoS

- `ws://65.21.138.182:8546`
- `ws://65.108.120.181:8546`
- `ws://65.109.52.201:8546`
- `ws://65.109.92.233:8546`

## Chain ID 169 — Manta Pacific Mainnet

- `ws://94.130.135.56:8546`

## Chain ID 171 — CO2e Chain

- `ws://65.108.126.254:8546`

## Chain ID 196 — X Layer

- `ws://65.108.126.222:8546`

## Chain ID 223 — B2 Mainnet

- `ws://65.109.83.213:8546`

## Chain ID 252 — Fraxtal

- `ws://65.108.69.56:8546`

## Chain ID 298 — Hedera Localnet

- `ws://95.216.235.134:8546`

## Chain ID 369 — PulseChain

- `ws://78.46.93.253:8546`

## Chain ID 648 — Endurance Smart Chain

- `ws://65.108.78.48:8546`
- `ws://65.109.226.1:8546`
- `ws://95.217.228.61:8546`
- `ws://116.202.161.187:8546`

## Chain ID 784 — 未在公开链表登记

- `ws://65.108.238.29:8546`
- `ws://95.217.119.251:8546`

## Chain ID 943 — PulseChain Testnet v4

- `ws://65.21.34.110:8546`

## Chain ID 999 — Wanchain Testnet

- `ws://65.21.85.180:8546`

## Chain ID 1088 — Metis Andromeda

- `ws://65.108.237.224:8546`

## Chain ID 1112 — WEMIX 3.0 Testnet

- `ws://65.108.6.166:8546`

## Chain ID 1329 — Sei Network

- `ws://65.109.49.164:8546`
- `ws://94.130.207.117:8546`
- `ws://95.217.111.212:8546`

## Chain ID 1337 — Geth Testnet

- `ws://65.21.60.195:8546`
- `ws://95.217.216.219:8546`

## Chain ID 1439 — Injective Testnet

- `ws://78.46.46.249:8546`

## Chain ID 1516 — Story Odyssey Testnet

- `ws://65.108.229.11:8546`

## Chain ID 1708 — TBSI Testnet

- `ws://95.217.122.150:8546`

## Chain ID 1776 — Injective

- `ws://65.109.32.209:8546`
- `ws://65.109.105.11:8546`
- `ws://95.216.13.185:8546`

## Chain ID 2222 — Kava EVM

- `ws://65.108.103.254:8546`
- `ws://95.216.33.33:8546`

## Chain ID 3300 — Realio Testnet

- `ws://65.21.197.14:8546`

## Chain ID 3301 — Realio

- `ws://65.109.69.143:8546`

## Chain ID 3501 — JFIN Chain

- `ws://65.108.0.235:8546`
- `ws://65.108.14.187:8546`
- `ws://135.181.220.175:8546`

## Chain ID 3502 — JFINPOS

- `ws://65.109.26.7:8546`

## Chain ID 5000 — Mantle

- `ws://65.108.232.245:8546`

## Chain ID 5858 — Chang Chain Foundation Mainnet

- `ws://65.109.2.81:8546`

## Chain ID 6480 — 未在公开链表登记

- `ws://135.181.19.55:8546`

## Chain ID 7117 — 0XL3

- `ws://65.21.195.240:8546`

## Chain ID 7269 — 未在公开链表登记

- `ws://49.12.69.41:8546`

## Chain ID 8163 — 未在公开链表登记

- `ws://95.217.44.178:8546`

## Chain ID 8453 — Base

- `ws://65.21.96.142:8546`
- `ws://65.108.9.159:8546`
- `ws://65.108.122.15:8546`
- `ws://65.108.140.140:8546`
- `ws://65.108.198.118:8546`
- `ws://65.108.232.180:8546`
- `ws://65.108.233.57:8546`
- `ws://65.109.23.81:8546`
- `ws://65.109.64.132:8546`
- `ws://65.109.113.219:8546`
- `ws://78.46.128.58:8546`
- `ws://94.130.129.107:8546`
- `ws://95.216.38.96:8546`
- `ws://95.216.229.104:8546`
- `ws://95.217.94.137:8546`
- `ws://95.217.94.138:8546`
- `ws://95.217.94.139:8546`
- `ws://95.217.94.140:8546`
- `ws://95.217.94.141:8546`
- `ws://95.217.94.142:8546`
- `ws://116.202.170.48:8546`
- `ws://135.181.17.207:8546`

## Chain ID 9008 — Shido Network

- `ws://65.21.1.153:8546`
- `ws://65.109.30.92:8546`
- `ws://65.109.115.195:8546`

## Chain ID 9069 — Apex Fusion - Nexus Mainnet

- `ws://49.12.104.129:8546`

## Chain ID 9555 — 未在公开链表登记

- `ws://135.181.56.236:8546`

## Chain ID 9601 — KUB Layer 2 Mainnet

- `ws://65.109.90.181:8546`

## Chain ID 9790 — Carbon EVM

- `ws://65.109.69.233:8546`

## Chain ID 9999 — myOwn Testnet

- `ws://95.217.126.214:8546`

## Chain ID 10618 — 未在公开链表登记

- `ws://95.217.105.61:8546`

## Chain ID 13373 — 未在公开链表登记

- `ws://65.109.115.131:8546`

## Chain ID 16601 — 0G Galileo Testnet（已弃用）

- `ws://65.109.83.40:8546`

## Chain ID 35147 — CoinAfrica

- `ws://95.216.229.58:8546`

## Chain ID 35935 — 未在公开链表登记

- `ws://49.12.110.160:8546`

## Chain ID 42069 — pegglecoin

- `ws://65.109.14.178:8546`
- `ws://65.109.16.163:8546`

## Chain ID 42220 — Celo

- `ws://65.108.74.96:8546`

## Chain ID 42431 — Tempo Testnet Moderato

- `ws://135.181.17.53:8546`

## Chain ID 59902 — Metis Sepolia Testnet

- `ws://65.21.126.182:8546`

## Chain ID 80069 — Berachain Bepolia

- `ws://78.46.106.54:8546`

## Chain ID 80094 — Berachain

- `ws://65.108.6.17:8546`
- `ws://65.108.110.37:8546`
- `ws://95.216.0.50:8546`

## Chain ID 84532 — Base Sepolia Testnet

- `ws://95.216.247.246:8546`

## Chain ID 88888 — Chiliz Chain

- `ws://65.21.235.141:8546`
- `ws://65.108.110.39:8546`

## Chain ID 100011 — QuarkChain L2

- `ws://65.108.236.27:8546`
- `ws://65.109.69.90:8546`

## Chain ID 110011 — QuarkChain L2 Testnet

- `ws://65.21.21.253:8546`
- `ws://65.109.63.154:8546`
- `ws://65.109.110.98:8546`

## Chain ID 123321 — Gemchain

- `ws://78.46.66.88:8546`

## Chain ID 167011 — 未在公开链表登记

- `ws://135.181.228.177:8546`

## Chain ID 258432 — Althea L1

- `ws://65.108.227.207:8546`

## Chain ID 480001 — MVP CHAIN

- `ws://65.21.110.118:8546`

## Chain ID 560048 — Ethereum Hoodi

- `ws://135.181.161.106:8546`
- `ws://135.181.162.139:8546`

## Chain ID 713714 — Vordium Chain

- `ws://65.21.249.212:8546`
- `ws://94.130.10.145:8546`

## Chain ID 845300 — 未在公开链表登记

- `ws://65.108.151.162:8546`

## Chain ID 853211 — Testethiq

- `ws://135.181.118.165:8546`

## Chain ID 2026777 — 未在公开链表登记

- `ws://135.181.138.36:8546`

## Chain ID 11155111 — Ethereum Sepolia

- `ws://65.108.230.142:8546`
- `ws://65.108.232.214:8546`
- `ws://65.109.98.79:8546`
- `ws://65.109.102.105:8546`
- `ws://95.216.99.174:8546`
- `ws://95.216.243.91:8546`

## Chain ID 65100004 — Autonity Piccadilly (Tiber) Testnet（已弃用）

- `ws://65.109.92.163:8546`

## Chain ID 108160679 — Oraichain Mainnet

- `ws://135.181.217.58:8546`

## Chain ID 383414847825 — Zeniq

- `ws://95.216.138.137:8546`
