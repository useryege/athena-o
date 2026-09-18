# 移除 Swagger 与 AI 接入展示

> 状态：已实施，待人工复验；本轮实现与验证证据见[验收记录](../../testing/swagger-ai-discovery-removal-acceptance.md)。
> 用户决定：2026-09-18，一并移除 Swagger、Connect AI 与配套 AI 接入文档，保留普通 API Key。

## 背景与目标

ATHENA 当前由用户独立开发，长期内没有其他开发者对接的需求。停止维护面向外部调用者的接口文档与 AI 接入展示，减少生成修正、文档同步、前端资源和依赖维护成本。

本次决定不表示取消业务 API，也不改变已有账户与权限边界。

## 已确认范围

- 移除 Swagger 文档页面、机器可读文档及其专用生成和发布链路。
- 移除 Connect AI 入口、连接说明与公开 AI 接入文档。
- 保留普通 API Key 的创建、一次性密钥展示、列表和撤销，以及既有过期、授权和访问限制。
- 保留现有业务接口、内部服务契约和前后端调用。

## 详细设计中的保留边界

以下处理已随[详细设计](../../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)整体审阅通过：

- Connect AI 签发的凭据本来就是普通 API Key。既有密钥继续遵循正常有效期、授权和撤销规则，不按名称批量作废或删除。
- 不引入替代性的外部接口文档系统、MCP 服务、兼容开关或过渡生成链路。
- 开发所需的 Proto、HTTP 映射、类型、实际 JSON 行为、测试与内部业务设计继续维护。
- 第三方服务的 OpenAPI 资料、历史设计批准材料和历史验收证据不属于本次删除对象。

## 目标行为与成功标准

1. 用户在 Security 与 Help 中不再看到 Swagger、Connect AI 或公开 AI 接入文档入口。
2. 已退役文档地址不再提供旧内容；具体 HTTP 行为按详细设计验收。
3. 生成和构建流程不再依赖 Swagger 专用工具，也不会重新生成被删除的文档。
4. 普通 API Key 与正常业务请求继续工作；删除文档测试时保留真实接口行为的覆盖。
5. 当前开发文档与最终代码一致，不能把历史材料当作仍需实现的要求。

## 关联资料

- [移除详细设计](../../superpowers/specs/2026-09-18-swagger-ai-discovery-removal-design.md)
- [当前 AI 接入与文档实现](../../design/developer-experience/ai-discovery-documentation.md)
- [账户凭据](../../design/identity-access/account-credentials.md)

本需求记录退役决定；验收记录区分隔离验证、真实本地环境和待人工复验，不把历史材料当作当前实现证据。
