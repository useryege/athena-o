# Polymarket API 文档拉取为 Markdown

## 场景

Polymarket 的 API 文档入口是：

```text
https://docs.polymarket.com/api-reference/introduction
```

该站点由 Mintlify 托管，已经通过 `llms.txt` 暴露了各页面对应的 Markdown 链接。因此不需要使用 HTML 爬虫或 HTML 转 Markdown 工具，直接按 `llms.txt` 中的 `.md` 链接下载即可。

## 推荐方式

在项目根目录执行：

```bash
./util/polymarket/sync-api-docs.sh
```

脚本会：

1. 拉取 `https://docs.polymarket.com/llms.txt`
2. 仅筛选 `https://docs.polymarket.com/api-reference/... .md` 链接
3. 去重排序后下载到 `util/polymarket/polymarket-api-docs/`
4. 每次同步前清空并重建目标目录，避免陈旧文件残留

下载后，本地目录会按远端 API Reference 的分类保存，例如：

```text
util/polymarket/polymarket-api-docs/introduction.md
util/polymarket/polymarket-api-docs/authentication.md
util/polymarket/polymarket-api-docs/markets/list-markets.md
```

## 验证

确认已拉取到 API Reference Markdown：

```bash
find util/polymarket/polymarket-api-docs -type f -name '*.md' | sort
```

也可以抽查目标页：

```bash
sed -n '1,80p' util/polymarket/polymarket-api-docs/introduction.md
```

## 为什么不用通用爬虫

- `https://docs.polymarket.com/api-reference/introduction.md` 等页面已经直接返回 `text/markdown`。
- `https://docs.polymarket.com/llms.txt` 已经是完整索引，可过滤出 `/api-reference/` 下所有 `.md` 链接。
- 直接下载 Markdown 比 Crawl4AI、Firecrawl 等通用爬虫更简单，也更不容易受页面样式、导航结构和前端渲染影响。

## 可选工具

如果需要图形界面或打包导出，可以使用：

- `https://llm.energy/`
- `https://github.com/nirholas/extract-llms-docs`

不过对 Polymarket 当前文档站来说，脚本方式已经足够。
