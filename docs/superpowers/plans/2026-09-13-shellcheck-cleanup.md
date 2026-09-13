# ShellCheck 提示与远端命令引用修复实施计划

> 用户已批准此方案（包括远端参数引用修复）。执行：Superpowers subagent-driven-development，独立实现/审查，主代理集成验证。

**目标：** 完整扫描49条提示归零，修复参数引用问题，保留脚本行为。
**架构：** 小型 ssh_exec 辅助函数逐参数转义；远端脚本内容固定，通过位置参数接收配置。动态 source 和间接调用采用最小范围分析说明。
**规格：** 用户已确认对话方案；本文件记录其约束与执行分工。

## 全局约束

- 仅本地替身测试：禁止连接真实SSH、执行真实部署/迁移/销毁、改动运行服务或业务数据。
- 四部署脚本保留动作顺序、环境变量优先级、迁移、数据卷范围、失败诊断、stdin流和退出码。
- 不升级工具、不降低扫描级别、不全局屏蔽规则；仅对确认误报按行/函数加说明。
- 保留所有既有未提交改动，隔离worktree中实施；经基线比较后仅回写本任务差异到根工作区；不提交或推送。
- 如果必须改部署语义/清理范围或需真实远端才能验证，先向用户说明并确认；普通失败继续在范围内排查。

## Task 1: 远端命令传参及四部署脚本

文件：新增 hack/lib/ssh-command.sh、hack/ssh-command_test.sh、hack/deploy-scripts_test.sh；修改 hack/prod-remote-deploy.sh、hack/deploy-etherscan-gateway.sh、hack/deploy-bsc-transaction-indexer.sh、hack/deploy-bsc-swap-indexer.sh。

- [x] 新增内部 ssh_exec host command args...；逐个参数POSIX单引号转义后调用 ssh，保持stdin/stdout/stderr/退出码，不用eval。空参数、空白、单引号、美元、反引号、命令替换文字、通配符、换行均原样保留。远端经登录shell解析一次，再执行传入命令。
- [x] 多行脚本传给 bash -c 或原语义等价的shell，以固定文本和位置参数分离数据。保持各原脚本set -e或set -euo pipefail语义，不把tar/docker-save流用作脚本文本通道。简单远端命令也使用辅助入口，不留下脆弱动态拼接。
- [x] TDD：先写并运行失败引用回归；再实现。四部署脚本正常/失败分支均用本地PATH替身，主应用覆盖deploy/hot-deploy/destroy，验证迁移顺序、volume范围与错误传播。必须真正把构建出的远端命令经shell解析后检查参数，不能只检查字符串包含。
- [x] 二进制stdin验证与非零远端退出码验证；原脚本本地与远端各层变量（例如容器内POSTGRES_USER/REDIS_PASSWORD）在正确层展开。
- [x] 不在测试中调用真实ssh/scp/docker/sudo或下载；临时环境与产物自清理，跟踪日志保留。不要改其余文件或写用户完成邮件。

## Task 2: 本地脚本、动态分析边界与回归

- [x] generate-abi.sh 六处前缀表达式按字面匹配ROOT；特殊字符路径用本地solcjs/abigen替身测试，先红后绿。
- [x] local-runtime.sh 直接判断清理命令结果，保留原始退出码优先和后续清理执行。测试正常/原始失败/各清理失败/信号状态，验证有归属资源边界。
- [x] start-temporal.sh 改计数循环实际使用变量，保持120次、每次间隔1秒和失败语义；以psql/docker/sleep替身测试。
- [x] install-protoc.sh / install-oras.sh 添加准确source路径；tool-versions.sh 版本变量沿用export方式；版本号不变。
- [x] ui-acceptance.sh NVM和update-codegen.sh 生成脚本采用带原因source=/dev/null；start-minio.sh信号回调SC2329与trader-sync-local_test.sh故意延迟展开SC2016采用带原因最小范围说明。
- [x] 保留已有失败基线；不写映射实现的冗余测试。可新增 hack/shell-local_test.sh，复用现有Bash测试方法。

## Task 3: 集成验收和交付

- [x] 所有项目shell文件 bash -n，完整make lint-shell退出0，范围不缩小。
- [x] 新增全部回归、现有hack/ai-dev-tools_test.sh、hack/trader-sync-local_test.sh、ui/scripts/acceptance-runner.test.mjs通过。按项目Node24.14.1执行。
- [x] 更新工具链用法与docs/testing/ai-dev-tools-readiness.md，保留49条历史及分类，记录误报说明、实际测试和局限；无需全站UI/Go重验。
- [x] 基线比对后安全回写，独立最终审查、差异/链接检查。保留已有运行服务及数据。
- [x] 所有必要验证完成后，从根目录make notify-task-complete一次，等待结果后交付。命令退出0；首次SMTP连接EOF，内建第2次尝试获服务器接收。
