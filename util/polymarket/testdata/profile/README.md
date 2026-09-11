# 固定官方 Profile 样本

这些文件是不可变的一手记录，不是所有实时路由均通过验收的声明。

| 文件 | 来源与时间 | SHA-256 |
| --- | --- | --- |
| profile.html | https://polymarket.com/@GCR，2026-09-10T09:07:34.492103Z | b98b350b9d05f57961af93aad7c93901af4b82fcae9396bac2d4755d217c68e3 |
| public-profile.json | Gamma public-profile，固定 GCR 钱包，同批观察 | 33d5eab0c6bd34b1c9b2ba3cd607aea7ef2c457af99c48c24c45b6de62eac243 |
| value.json | Data v1 /value，2026-09-10T17:03:15.616Z 的独立观察 | f980a17cc0ae08990c65a52392493fc4d1ed15be51903eb217f653d08a3f3287 |

原始来源、完整查询 URL 与适用边界见仓库 requirements/polymarket-copy-trading/evidence 中的 profile 和 position-value 记录。测试通过回环 HTTP 传输回放，生产 allowlist 不因测试改变。固定 HTML 的 1W P/L 含相邻重复时间；不能排序或去重修正原始样本。
