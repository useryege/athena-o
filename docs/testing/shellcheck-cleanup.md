# ShellCheck 与远端参数引用修复验收

日期：2026-09-13。批准范围见[实施计划](../superpowers/plans/2026-09-13-shellcheck-cleanup.md)。本任务只修复脚本引用和静态分析提示，不升级工具，不修改部署动作的业务含义。

## 原因与处理

任务开始在隔离工作区重新执行 `make lint-shell`：49条诊断、14个文件，底层ShellCheck退出1，Make退出2；与前次记录一致。原始结果保留在根工作区 `.tmp/shellcheck-cleanup/baseline.log`。

| 规则 | 基线数量 | 原因与处理 |
| --- | ---: | --- |
| SC2029 | 31 | 本地展开多数符合意图，但配置值直接拼入远端单引号字符串会破坏参数边界。统一 `ssh_exec` 参数转义，固定远端脚本通过位置参数接收数据。 |
| SC2295 | 6 | ROOT中的通配符会参与前缀模式匹配，导致特殊路径日志保留绝对前缀。将前缀按字面引用。 |
| SC1091 | 4 | 两个安装器补准确source路径；NVM及运行时生成的Kubernetes helper采用带原因的外部source说明。 |
| SC2034 | 3 | 共享版本变量实际被安装器使用，沿用export方式声明；Temporal改用实际参与循环条件的计数变量。 |
| SC2181 | 2 | 直接判断清理命令结果；保持原始错误优先、后续清理继续执行。 |
| SC2016 | 1 | 代理测试中的变量必须在子shell启动后展开，保留字面脚本并按行说明。 |
| SC2329 | 1 | MinIO清理函数由INT/TERM/EXIT调用，保留实现并按函数说明。 |
| SC2154 | 1 | protoc版本来自共享source；准确source路径让分析器看到定义。 |

外部source使用 `source=/dev/null` 仅告知静态分析不展开外部/临时脚本，不改变运行时读取路径。`ssh_exec`调用SSH的单行SC2029注释说明该变量已经完成POSIX逐参数转义；固定远端脚本文字以及测试特殊值的SC2016注释明确要求在远端/测试子shell展开；每条仅作用于对应命令或赋值。没有按文件或全局屏蔽SSH规则。未修改工具版本、扫描文件集合或严重级别。

## 验证结果

- ABI先红后绿：包含空格、方括号、星号、问号和单引号的临时仓库路径，旧实现输出错误绝对前缀；修复后成功、空字节码、多字节码、多Solidity文件四场景通过。
- 清理流程：正常、原始命令失败叠加清理失败、各清理失败、INT/TERM、关闭端口清理，共9种退出/信号场景；另有真实归属判断函数的Docker替身测试，拒绝其他项目或组件不匹配的容器，继续清理本项目匹配容器。
- Temporal：立即就绪、延迟就绪、120次超时及Docker失败4场景通过，逐次sleep参数仍为1秒。
- 既有 `hack/ai-dev-tools_test.sh` 和 `hack/trader-sync-local_test.sh` 通过。
- 验收运行器首次38/39：一次SIGHUP后重启harness超出测试设置的450毫秒启动期限。该单项复验1/1、完整重跑39/39通过，未修改运行器/测试、未调超时或筛除用例；首次失败保留，不将其描述为首次全绿。
- SSH辅助入口回归通过：字面参数、缺参退出2、选项样式host、二进制stdin、stdout/stderr及退出码37。部署引用先红后绿，原主应用在含单引号目录报远端shell语法错误，修复后正常。
- 四部署脚本回归通过：主应用deploy/hot-deploy/destroy正常和迁移/删卷失败，验证三卷创建/保留/删除范围，全部动作保留无关卷哨兵且NUL记录精确比对删除目标；两个BSC脚本正常及up失败后的ps/logs；Gateway正常及服务未活动时的journalctl。
- 容器变量分层回归：本地、远端与容器的三个变量值各不相同且含特殊字符。Docker替身以隔离环境实际执行容器 `sh -c`，逐项比较pg_isready、createdb、psql、redis-cli的NUL分隔参数，确保容器层展开且参数不拆分。
- 完整 `make lint-shell` 退出0；同范围JSON扫描结果为 `[]`。原有36个脚本加辅助库和3个测试，共40个 `.sh` 文件，全部 `bash -n` 通过。扫描集合未缩小。
- 本地脚本独立审查通过；远端脚本审查提出的两项测试缺口已补齐并通过定向复审。最终独立审查为 Approved，无待修复问题。22个任务文件按开始基线核对后同步回根目录，与通过测试的隔离工作区逐字节一致；根目录完整 `make lint-shell` 再次退出0，差异和新增文档链接检查通过。

## 证据与边界

本机证据目录为根工作区 `.tmp/shellcheck-cleanup/`（Git忽略，清理后需重跑）：

- [ShellCheck基线](../../.tmp/shellcheck-cleanup/baseline.log)、[最终完整扫描](../../.tmp/shellcheck-cleanup/lint-shell-final.log)、[最终JSON](../../.tmp/shellcheck-cleanup/shellcheck-final.json)、[40文件语法检查](../../.tmp/shellcheck-cleanup/syntax-final.log)
- [根目录完整扫描](../../.tmp/shellcheck-cleanup/lint-shell-root.log)、[最终独立审查](../../.tmp/shellcheck-cleanup/final-review.md)
- [部署引用失败复现](../../.tmp/shellcheck-cleanup/deploy-quoting-red.log)、[SSH入口保护失败复现](../../.tmp/shellcheck-cleanup/ssh-helper-guard-red.log)、[SSH辅助回归](../../.tmp/shellcheck-cleanup/ssh-helper-test.log)、[四脚本部署回归](../../.tmp/shellcheck-cleanup/deploy-scripts-test.log)、[补充边界红测试](../../.tmp/shellcheck-cleanup/deploy-boundary-red.log)
- [ABI失败复现](../../.tmp/shellcheck-cleanup/local-abi-red.log)、[本地脚本绿测试](../../.tmp/shellcheck-cleanup/local-green.log)
- [工具回归](../../.tmp/shellcheck-cleanup/ai-dev-tools-test.log)、[代理环境回归](../../.tmp/shellcheck-cleanup/proxy-test.log)
- [运行器首次结果](../../.tmp/shellcheck-cleanup/acceptance-runner-test.log)、[超时单项复查](../../.tmp/shellcheck-cleanup/runner-sighup-investigation.log)、[运行器完整重跑](../../.tmp/shellcheck-cleanup/acceptance-runner-final.log)
- [完成邮件结果](../../.tmp/shellcheck-cleanup/completion-email.log)：从根目录调用一次，命令退出0；首次连接EOF后由内建重试完成，SMTP第2次尝试接收。

远端测试使用临时仓库与命令替身，真实执行shell解析，但SSH/SCP/Docker操作均由本地替身接收。它证明参数、数据流、顺序与错误传播，不证明真实主机网络、权限、Docker或业务迁移成功。本次不连接真实SSH，不部署、不迁移、不销毁真实资源，保留已有开发环境、数据和其他任务改动。
