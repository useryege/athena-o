# 人工审查报告：十一应用全栈启动与六板块访问控制

- 任务：十一应用全栈启动与六板块访问控制（full-stack-access）
- 交付轮次：R1
- 仓库：`/home/yege/work/athena`；分支：`rf4`
- 被审查代码及长期文档完整版本：`c0016253d927fc764be0f4b70e86a921b209f31c`
- 设计依据：[已整体批准的技术方案](../../../../superpowers/specs/2026-09-17-full-stack-access-reassessment-design.md)
- AI 验证入口：[完整验收记录](../../../full-stack-access-acceptance.md)
- AI 交付报告：[ai-delivery.md](ai-delivery.md)
- 人工指南：[review-guide.md](review-guide.md)
- 实际环境、身份与数据：指南要求新实例 `full-stack-access-review-r1`，local-user 与 local-admin；实际差异由用户补充。
- 审查时间：待人工填写
- 报告状态：草稿
- 总体结论：待人工填写

先按指南核对上面的完整版本。后续仅交付文档提交允许 HEAD 为其后代，但产品/运行器/测试输入不得有差异；版本不符不要强制重置现有工作。

## 检查结果

| 检查编号 | 具体检查 | 结果 | 实际观察 | 证据 | 问题编号 |
| --- | --- | --- | --- | --- | --- |
| CHK-001 | 十一应用及双 realm 身份：十一项 Startup ready，CoreUsable、SelectedReady、FullStackReady 均 true，无失败；会员 local-user、管理员 local-admin，各自 bootstrap 为 AUTHENTICATED。状态命令退出 0 本身不算通过。 | 未执行 |  |  |  |
| CHK-002 | 首次默认关闭与六键范围：恰好 trader_sync、solana、market_radar、managed_oo、profit_sharing、worm 六项且全 CLOSED；首次没有伪造的修改账户/时间。页面明确 Token 延期、关闭不停止后台和通知。已有主验收 full-stack-access 实例是混合设置，不能用于证明首次默认。 | 未执行 |  |  |  |
| CHK-003 | 第四页签桌面手机与核心状态独立：标签不重叠、指示线对应选项，六设置行可操作；三个核心状态来源仍可独立查询，业务开关不使核心 Trader Sync runtime 状态不可读。 | 未执行 |  |  |  |
| CHK-004 | 六业务逐项开放关闭与页面卸载：开放后业务可达；关闭后通常在下一2秒轮询后收起正文，陈旧开放决定最长5秒失效；有权限菜单和URL保留，显示关闭说明，重新开放没有恢复旧草稿。空列表不能当作请求失败。 | 未执行 |  |  |  |
| CHK-005 | 真实业务 API 关闭错误与核心继续：六业务及Worm原始HTTP返回503、MODULE_ACCESS_CLOSED、athena.module_access、对应module_key、reason header与private/no-store；登录/bootstrap、账户、Wallet、Notification、Service Status、开关管理仍可用。脚本应退出0；此脚本会显式更新本审查实例六个开关。 | 未执行 |  |  |  |
| CHK-006 | 管理权限和原有授权边界：会员不能读取管理设置或写开关；API Key不能借管理员身份修改设置；管理员/API Key业务请求仍遵循准入。原认证及proof绑定错误先于模块准入，账户权限revision不因开关改变。 | 未执行 |  |  |  |
| CHK-007 | 失焦离线恢复及迟到响应隔离：离线/隐藏/焦点恢复先失效，旧OPEN响应不能恢复页面；受控filtered浏览器用例全部通过。该命令创建自己的临时harness，不证明真实上游交易。 | 未执行 |  |  |  |
| CHK-008 | Worm迟到签名与关闭后不自动驱动：关闭后迟到签名不能verify/start；heartbeat、execute-next、排队timer不因重新开放自动恢复；不自动发送pause/terminate，只读权威现状并等待用户明确操作。 | 未执行 |  |  |  |
| CHK-009 | 保存未知、切页和并行保存竞态：未知结果显示不确定并重新读取权威设置，不自动重复PUT；返回页签不永久卡Saving；迟到旧保存不能覆盖新状态，并行行各自独立。 | 未执行 |  |  |  |
| CHK-010 | 同实例重启与数据保持：六值、最后修改人及时间保持，不重置默认；账户与已有数据保留。数据库OID、成功检查点的深入检查可复核AI原始只读快照，不把未手工重测写成亲测。 | 未执行 |  |  |  |
| CHK-011 | 关闭后台继续和已受理工作：CLOSED及审计保持一致，真实成功slot/last_success_at前进，Notification last_poll_at前进；已受理HTTP请求在关闭后释放成功，新请求被503拦截。上游失败导致未推进须如实记录，不能只看started或端口。 | 未执行 |  |  |  |
| CHK-012 | 根路径及前缀双 realm Chrome验收：系统Chrome两次smoke与cleanup均passed，会员/admin bootstrap身份正确；/athena页面资源和API代理正常，直接API RootPath与公开BaseHRef不混淆。手机/桌面检查同CHK-003。 | 未执行 |  |  |  |
| CHK-013 | 局部最小依赖与external只读：证据有实际PostgreSQL和精确归属，external不DDL/seed、不记录借用库为Owned、不停止它；没有未选库/服务依赖。此项默认复核已有独立故障证据，若要求重测请用新专用实例并记录。 | 未执行 |  |  |  |
| CHK-014 | 故障分级与有界停止：每项都有对应真实进程或受控故障证据和退出结果，停止命令非零时能区分历史业务失败与清理失败；不以HTTP200认定UI/双realm ready。 | 未执行 |  |  |  |
| CHK-015 | 人工环境按归属收尾：人工实例stopped，所属进程退出、端口释放、自有容器停止，数据卷/日志/证据保留；原七共享容器和五remote Gateway保持原样。停止命令退出不等于已核对，仍须看状态与身份。 | 未执行 |  |  |  |

## 问题记录

尚未收到人工问题。发现问题时新增稳定编号 ISSUE-001、ISSUE-002，逐项填写对应 CHK、实际步骤、预期、实际结果、复现频率、影响范围和原始证据。原始观察保留，不由后续修复结论覆盖；不适用时写明依据。

## 未执行或受阻项目

| 检查编号 | 状态 | 原因 | 阻塞问题编号 | 可继续的独立检查 |
| --- | --- | --- | --- | --- |
| 待人工填写 | 待人工填写 |  |  |  |

## 本轮结论与交接

- 报告提交声明：待人工填写
- 环境状态及收尾证据：待人工填写
- 保留实例详情及准确停止命令：待人工填写；无则写“无”
- 明确接受的例外及范围：待人工填写；无则写“无”

完成或受阻后可将状态改为“已提交”，并在会话说明“R1 报告已提交”。如实保留失败、受阻和未执行项；AI 不代填通过。完整报告提交授权核实并修复既定设计内问题，新的设计改变另行说明。
