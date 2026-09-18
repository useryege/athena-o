# Swagger 与 AI 接入展示退役

> 设计状态：退役实施见[验收记录](../../testing/swagger-ai-discovery-removal-acceptance.md)；原设计由 Git 历史保留。

用户于 2026-09-18 批准[移除需求](../../requirements/developer-experience/api-documentation-removal.md)及[详细设计](../../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)。当前实现删除 Swagger 页面/JSON、ReDoc、专用生成与发布链路、Connect AI 浏览器流程，以及公开 `/llms.txt` 和 `/docs/ai/` 文档。

后端与 Vite 在静态文件命中和 SPA 回退前对退役路径返回 404，包括附加静态目录仍存旧文件的情况；部署前缀剥离后规则一致。旧二进制或远端部署是否更新须另行核实，本次只实施与验证本地版本。

Security 只保留普通 API Key 面板；会员与管理员 Help 保留配置的支持/下载入口，无配置时显示 `No help resources configured`。普通密钥创建、一次性展示、复制失败手工选择、Done 清理、撤销、到期、权限和账户切换隔离继续保留。

既有 `ai-...` 名称密钥一直是普通 API Key，没有独立 AI 连接记录或凭据类型；名称不用于授权、识别调用者或撤销。Bearer 客户端继续接受当前账户、模块、资格、归属与操作权限校验；管理员不能签发普通账户 API Key。未新增替代文档站、MCP 或兼容生成分支。

保留业务 Proto、HTTP annotations、Go/gogo/gateway 生成、真实 JSON 适配与测试，以及 `/api/version`、`/api/v1/session/userinfo`、`/api/v1/app/bootstrap`。开发说明见 [API Development](../../developer-guide/api-docs.md)，凭据规则见[账户凭据](../identity-access/account-credentials.md)。第三方协议资料、历史视觉批准与验收证据继续保留；本说明不改写这些材料当时的事实。
