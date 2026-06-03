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
3. 服务器本机可以访问 Athena API：

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

如果宝塔提供 WebSocket 相关开关，可以一并开启。基础 UI 和 API 访问通常不依赖该开关，但开启后更稳妥。

## 配置 HTTPS

1. 进入站点设置的 `SSL`。
2. 申请 Let's Encrypt 证书。
3. 开启强制 HTTPS。

完成后使用 HTTPS 访问：

```text
https://athena.example.com
```

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
PROD_ENV_FILE=./.env.prod PROD_LOG_SERVICE=athena-server make prod-logs-remote
```

## 安全说明

- 不需要在云服务器安全组或系统防火墙中开放公网 `8080`。
- 公网只需要开放 `80` 和 `443` 给宝塔/Nginx。
- 不建议将 `ATHENA_SERVER_BIND_ADDR` 改成 `0.0.0.0` 后直接暴露 `8080`。
- 如果启用登录，生产环境应确认账号和密码 hash 配置稳定，避免使用重启后变化的临时 admin 密码。

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
