# 宝塔面板反向代理 Athena UI

本文说明如何在服务器宝塔面板中创建一个 PHP 站点，并通过 Nginx 反向代理访问已部署的 Athena UI。

当前生产部署默认将 `athena-server` 绑定到服务器本机：

```text
127.0.0.1:8080
```

公网访问建议只暴露宝塔/Nginx 的 `80` 和 `443` 端口，由 Nginx 反向代理到 `http://127.0.0.1:8080`。

## 前置条件

1. 域名已经解析到 Athena 所在服务器，例如 `47.245.181.189`。
2. Athena 已经完成远程部署并启动。
3. Google Cloud 生产 Web OAuth client 已登记精确回调 URI
   `https://<你的域名>/auth/google/callback`。
4. 服务器本机可以访问 Athena API：

```bash
ssh root@47.245.181.189
curl -sS http://127.0.0.1:8080/api/version
```

如果返回类似以下内容，说明 Athena 服务正常：

```json
{"Version":"v3.3.6+916cb39.dirty"}
```

## 在宝塔中添加站点

1. 登录宝塔面板。
2. 进入 `网站`。
3. 点击 `添加站点`。
4. 域名填写你的访问域名，例如：

```text
athena.example.com
```

5. 项目类型选择 PHP 站点即可。实际请求会被 Nginx 反向代理到 Athena，因此 PHP 版本不影响 Athena UI。
6. 提交创建站点。

## 配置反向代理

1. 进入刚创建的网站设置。
2. 打开 `反向代理`。
3. 点击 `添加反向代理`。
4. 代理名称填写：

```text
athena
```

5. 目标 URL 填写：

```text
http://127.0.0.1:8080
```

6. 发送域名建议填写：

```text
$host
```

7. 保存并开启反向代理。

如果宝塔允许编辑反向代理配置，建议确认 Nginx 配置中包含以下代理头：

```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

现有整站反向代理必须原样转发 `/auth/google/login` 和
`/auth/google/callback`。不要为 `/auth/google/*` 添加路径重写，也不要剥离
`/auth` 前缀。Google callback 的查询串包含一次性 authorization code 和 state，
因此必须用精确 location 关闭该请求的 Nginx access log；应用响应还会设置
`Referrer-Policy: no-referrer`。如果站点按 location 分开配置，可使用以下规则；
`proxy_pass` 后不要附加会替换请求路径的 URI：

```nginx
location = /auth/google/callback {
    access_log off;
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location ^~ /auth/google/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Athena 使用显式配置的 `ATHENA_GOOGLE_OIDC_REDIRECT_URI`，不会信任这些 header
来动态构造 OAuth 回调地址；保留 header 是为了让应用日志和其他请求保持正确的
外部请求上下文。

如果宝塔提供 WebSocket 相关开关，可以一并开启。基础 UI 和 API 访问通常不依赖该开关，但开启后更稳妥。

## 配置 HTTPS

1. 进入站点设置的 `SSL`。
2. 申请 Let's Encrypt 证书。
3. 开启强制 HTTPS。

完成后使用 HTTPS 访问：

```text
https://athena.example.com
```

## 配置 Google Cloud 回调

在 Google Cloud Console 中使用独立的生产 **Web application** OAuth client，
把 consent audience 配置为 **External** 并发布应用，然后把以下值登记为
Authorized redirect URI。Testing 状态仍只允许 Test users，不能提供开放注册：

```text
https://athena.example.com/auth/google/callback
```

该值必须与生产环境中的 `ATHENA_GOOGLE_OIDC_REDIRECT_URI` 逐字符一致，包括
scheme、域名、端口（如有）和路径。不要登记 HTTP 生产回调，也不要使用通配符。
本地开发应使用另一个 client，其回调为
`http://localhost:4000/auth/google/callback`。

## 验证

在浏览器打开你的域名：

```text
https://athena.example.com
```

也可以在服务器本机验证 Athena 仍然只监听本机代理目标：

```bash
ssh root@47.245.181.189 'curl -sS http://127.0.0.1:8080/api/version'
```

查看 `athena-server` 日志：

```bash
ssh root@47.245.181.189 'cd /root/athena && docker compose -f docker-compose.prod.yml --env-file .env logs -f athena-server'
```

## 安全说明

- 不需要在云服务器安全组或系统防火墙中开放公网 `8080`。
- 公网只需要开放 `80` 和 `443` 给宝塔/Nginx。
- 不建议将 `ATHENA_SERVER_BIND_ADDR` 改成 `0.0.0.0` 后直接暴露 `8080`。
- 生产必须配置 `ATHENA_ADMIN_GOOGLE_EMAIL`，用于在未知 Google 身份注册时标记唯一的
  管理员候选。普通用户无需预登记 Google `sub`；任意 verified Google account 都会先
  进入 `/register` 选择永久 username，提交成功后才创建 UUID `account_id` 和 Pending
  账号。管理员角色来自持久化的 `administrator` 字段，不来自 username。系统没有临时
  管理员密码或密码兜底入口。
- `/auth/google/callback` 的精确 Nginx location 必须保持 `access_log off`，避免
  code/state 进入默认 `$request` 日志。
- Google client secret 只保存在远端由容器 UID/GID `999` 持有的 `0600` 文件中，
  并由 Compose 只读挂载给 `athena-server`。

## 常见问题

### 访问域名时显示宝塔默认页

确认站点域名是否填写正确，并检查该站点的反向代理是否已经启用。

### 访问域名时返回 502

先在服务器本机执行：

```bash
curl -sS http://127.0.0.1:8080/api/version
```

如果该命令失败，说明 Athena 服务未启动或端口映射异常，需要先检查远端 compose 状态：

```bash
ssh root@47.245.181.189 'cd /root/athena && PROD_POSTGRES_VOLUME=athena-prod-postgres-data docker compose -f docker-compose.prod.yml --env-file .env ps'
```

### HTTPS 正常但接口请求失败

检查反向代理配置中是否保留了 `Host`、`X-Forwarded-For` 和 `X-Forwarded-Proto` 等代理头，并确认没有额外路径重写。

### Google 提示 redirect_uri_mismatch

对比 Google Cloud Authorized redirect URI 和
`ATHENA_GOOGLE_OIDC_REDIRECT_URI`，确认两者逐字符一致，并检查 Nginx 是否原样转发
`/auth/google/callback`。生产回调必须使用公开域名的 HTTPS URI。

### 更换 Google client 或管理员邮箱后仍使用旧管理员身份

重启 `athena-server` 使新的 client 或管理员邮箱配置生效。管理员邮箱只用于未知身份
创建注册票据时标记管理员候选；提交 username 后，唯一管理员角色和 Google `sub`
绑定都成为持久化账号状态。修改邮箱配置不会转让或重绑管理员。需要重新初始化身份时
使用全新 PostgreSQL/Redis/MinIO 状态并轮换 `ATHENA_JWT_SECRET`；这会删除账号、
username、权限与注册票据，并使全部旧会话和 API Key 失效。
