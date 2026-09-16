# 人工审查指南：Impeccable 上游更新

## 审查对象

- 任务：2026-09-16-impeccable-update；轮次：R1。
- 仓库：`/home/yege/work/athena`；分支：`rf4`；安装变更未提交。
- 上游版本：`0a4e72a254f3b175c95b36b82e5f2e60fa63f116`。
- 安装清单 SHA256：`03d269ed3c613293fc27a12a6ed8d8d5f63ce44d942b583d53c5bead89711ff0`。
- 依据：用户在本任务会话要求同步最新版本并授权选择实现方式；具体范围见 [AI 交付报告](ai-delivery.md) 和 [来源记录](../../../../../.agents/impeccable.json)。
- 配套报告：[human-report.md](human-report.md)。

## 版本核对

先在 WSL 终端进入本工作区，再执行只读核对：

```bash
cd /home/yege/work/athena
sha256sum .tmp/impeccable-update/0a4e72a254f3b175c95b36b82e5f2e60fa63f116/SHA256SUMS
sha256sum -c .tmp/impeccable-update/0a4e72a254f3b175c95b36b82e5f2e60fa63f116/SHA256SUMS
```

第一条预期 SHA256 为 `03d269ed3c613293fc27a12a6ed8d8d5f63ce44d942b583d53c5bead89711ff0`；第二条预期 61 个文件全部 `OK`。来源记录的 `commit` 必须为 `0a4e72a254f3b175c95b36b82e5f2e60fa63f116`。不一致时记录受阻，不强制重置工作区。

## 环境与身份

无需启动 ATHENA、浏览器或数据库。本轮人工检查只需当前 WSL/Linux x64 终端、Python 3、`sha256sum` 和仓库读权限；不需要账户或业务测试数据。证据及可恢复产物位于上述 `.tmp` 目录。

## 检查顺序

### CHK-001：安装版本与完整性

- 前置条件：处于上述仓库根目录。
- 操作：执行“版本核对”命令，并打开 `.agents/impeccable.json`。
- 预期：61 个文件全部通过；上游 SHA 一致；`files` 包含 57 个 Skill 文件；`installation_scope` 明确为当前 WSL2/Linux x64。
- 依据：完整同步最新上游并保留来源信息。
- 记录：将命令结果和实际 SHA 填入报告；无数据修改，无恢复步骤。

### CHK-002：新命令与原版技能入口

- 前置条件：CHK-001 通过；使用 WSL 终端。
- 操作：执行以下命令，再打开 `.agents/skills/impeccable/SKILL.md` 的命令表及 `reference/generate.md`。

```bash
.agents/skills/impeccable/scripts/impeccable engine-probe
.agents/skills/impeccable/scripts/impeccable live-generate --help
```

- 预期：输出 `impeccable-engine 0.1.5` 和包含 `--selector`、`--action`、`--dry-run` 的帮助，均成功退出；命令表包含 `generate` 且目标文档存在。版本号没有上调是上游现状，以提交 SHA 判断更新。
- 依据：新增 `generate` 需要技能及同提交引擎一起可用。
- 记录：命令输出及文档检查结果；命令不启动服务或改写业务文件，无恢复步骤。应用重新加载技能目录需重新启动 Codex。

### CHK-003：项目配置保留与环境范围

- 前置条件：CHK-001 通过；在仓库根目录。
- 操作：对比安装前归档中的 Hook 和配置，再阅读来源记录的 `engine.note`。

```bash
python3 - <<'PY'
from pathlib import Path
import tarfile
archive = Path('.tmp/impeccable-update/0a4e72a254f3b175c95b36b82e5f2e60fa63f116/before-update.tar.gz')
with tarfile.open(archive) as saved:
    for name in ['.codex/hooks.json', '.impeccable/config.json']:
        assert Path(name).read_bytes() == saved.extractfile(name).read(), name
        print(name, 'unchanged')
PY
```

- 预期：两个文件均输出 `unchanged`；记录说明仅当前 Linux x64 使用同提交构建引擎，其他平台仍受上游发布引擎限制。
- 依据：更新技能并保留项目既有集成与设计偏好。
- 记录：输出及实际观察；仅从归档读取，不解压覆盖文件，无恢复步骤。

## 复验范围与收尾

本轮首次交付，无历史结果沿用、已提交问题或先前受阻项。人工检查不启动资源，因此无停止命令。将每项实际结果填写到配套报告；受阻和未执行如实记录，报告完成后可标为“已提交”。
