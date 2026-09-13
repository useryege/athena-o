# ESLint files 匹配验收

## 根因与范围

ESLint flat config 的 `files` 模式以配置工作目录中的相对路径匹配。原配置将所有
应用规则写成 `./src/...`，而 ESLint 为目标文件使用 `src/...`，因此 realm 专属的
`no-restricted-imports` 配置没有生效。修复将四个 `files` 条目的前缀改为 `src/`。

本次没有改动受限导入的模式、错误级别、忽略策略、依赖或产品行为。

## 回归覆盖

[`ui/scripts/eslint-config.test.mjs`](../../ui/scripts/eslint-config.test.mjs) 从脚本自身
位置确定 UI 根目录，通过 ESLint API 加载真实 `eslint.config.mjs`，并以绝对虚拟文件名
验证：

- member 到 admin、admin 到 member，以及 shared、session、components 到两个 realm 的
  `.ts`/`.tsx` 嵌套导入；覆盖既有一至三层相对路径模式；
- 每个负例都断言具体 `no-restricted-imports` 规则、消息、位置、严重级别与
  `ImportDeclaration` 节点，并确认没有解析失败；
- 同 realm、共享模块、第三方包导入仍允许，且现有 `*.test.ts` 忽略规则保留。

`yarn lint` 先运行该 Node 原生回归，再执行原有 TypeScript 和 ESLint 检查。

## 验收证据

在 Node `v24.14.1` 上运行。旧配置的红测记录了 10 个跨 realm 导入未产生
`no-restricted-imports` 诊断：[`task-1-red.log`](../../.tmp/eslint-go-lint/task-1-red.log)。
修复后的 12 个配置回归用例结果：[`task-1-green.log`](../../.tmp/eslint-go-lint/task-1-green.log)。
从仓库根目录调用与完整 `yarn lint` 的日志也保存在
[`../../.tmp/eslint-go-lint/`](../../.tmp/eslint-go-lint/)。

2026-09-13交付后，在项目根目录执行 `yarn --cwd ui lint` 再次退出0：配置回归12/12、
TypeScript及完整ESLint均通过，见[根目录验收日志](../../.tmp/eslint-go-lint/root-yarn-lint.log)。
独立任务审查与[最终审查](../../.tmp/eslint-go-lint/final-review.md)均通过。
后续Go检查仅完成[分类与修复方案](go-lint-triage.md)，没有修改Go代码。
