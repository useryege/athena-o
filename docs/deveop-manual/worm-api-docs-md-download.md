# Worm API 文档拉取为 Markdown

## 场景

Worm 的对接文档入口是：

```text
https://docs.worm.wtf/api-reference
```

该站点由 Mintlify 托管，已经通过 `llms.txt` 暴露了各页面对应的 Markdown 链接。因此不需要使用 HTML 爬虫或 HTML 转 Markdown 工具，直接按 `llms.txt` 中的 `.md` 链接下载即可。

## 推荐方式

在项目根目录执行：

```bash
mkdir -p docs/vendor/worm-api

curl -fsSL https://docs.worm.wtf/llms.txt \
  | grep -oE 'https://docs\.worm\.wtf/api-reference/[^)]*\.md' \
  | sort -u \
  | while read -r url; do
      path="${url#https://docs.worm.wtf/api-reference/}"
      mkdir -p "docs/vendor/worm-api/$(dirname "$path")"
      curl -fsSL "$url" -o "docs/vendor/worm-api/$path"
    done
```

下载后，本地目录会按远端 API Reference 的分类保存，例如：

```text
docs/vendor/worm-api/account/account-assets.md
docs/vendor/worm-api/authentication.md
docs/vendor/worm-api/markets/list-markets.md
docs/vendor/worm-api/orders/create-order-draft.md
```

## 验证

确认已拉取到 API Reference Markdown：

```bash
find docs/vendor/worm-api -type f -name '*.md' | sort
```

也可以抽查任意页面：

```bash
sed -n '1,80p' docs/vendor/worm-api/introduction.md
```

## 为什么不用通用爬虫

- `https://docs.worm.wtf/api-reference/introduction.md` 等页面已经直接返回 `text/markdown`。
- `https://docs.worm.wtf/llms.txt` 已经是完整索引，可过滤出 `/api-reference/` 下所有 `.md` 链接。
- 直接下载 Markdown 比 Crawl4AI、Firecrawl 等通用爬虫更简单，也更不容易受页面样式、导航结构和前端渲染影响。

## 可选工具

如果需要图形界面或打包导出，可以使用：

- `https://llm.energy/`
- `https://github.com/nirholas/extract-llms-docs`

不过对 Worm 当前文档站来说，脚本方式已经足够。
