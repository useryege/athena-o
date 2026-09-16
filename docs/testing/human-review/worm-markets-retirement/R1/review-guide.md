# 人工审查指南：Worm Markets 数据退役与 Worm Trading 保留

> 本指南对应已完成 AI 审阅、约定验证与环境收尾的 R1 产品版本。人工检查结果仍由用户填写。

## 审查对象

- 任务标识：<code>worm-markets-retirement</code>
- 轮次：R1
- 仓库／实现 worktree：<code>/home/yege/work/athena/.worktrees/worm-markets-retirement</code>
- 现场数据 checkout：<code>/home/yege/work/athena</code>，原 main default namespace <code>ac252d3201cc3c6d872334f8e6a1bcf5</code>
- 分支：<code>codex/worm-markets-retirement</code>
- 受审产品完整版本：<code>1dfcb795dfdf3da1ecb1b63e4acc9677604d90a5</code>
- Task 10 docs gate 阶段版本：<code>d66c0c3d07a16e3050600fd244e349cbcd79c575</code>；后续产品提交还包含 budget/current docs 修订，本 SHA 只标识该阶段审阅证据
- R1 材料版本：运行本节命令，以正式 R1 目录最后一次提交解析；材料提交不改变上述受审产品版本
- 设计依据：[Worm Trading Market Query 设计](../../../../superpowers/specs/2026-09-16-worm-trading-market-query-design.md)、[Worm Markets 退役需求](../../../../requirements/development-runtime/worm-markets-removal.md)、[验收记录](../../../worm-markets-retirement-acceptance.md)
- 配套报告：[human-report.md](human-report.md)

## 版本核对

DOCS_GATE_SHA 只记录 Task 10 docs gate 阶段提交；PRODUCT_SHA 是本轮固定受审产品并包含其后的 budget/current docs 修订。MATERIAL_SHA 从正式 R1 目录的最后一次提交运行时解析，避免材料自引用；PRODUCT_SHA 必须是该材料提交的祖先。在启动任何服务或执行会修改数据的步骤前完成本节；任何一项不符都停止环境恢复，在报告中记为“受阻”，不得 checkout、reset 或覆盖当前工作。

~~~bash
set -e
REVIEW_WT=/home/yege/work/athena/.worktrees/worm-markets-retirement
FIELD_MAIN=/home/yege/work/athena
TASK_BASE=617bd6a26345905dc7125911ec72c1cee56afe0b
DOCS_GATE_SHA=d66c0c3d07a16e3050600fd244e349cbcd79c575
PRODUCT_SHA=1dfcb795dfdf3da1ecb1b63e4acc9677604d90a5
MATERIAL_SHA=$(git -C "$REVIEW_WT" log -1 --format=%H -- docs/testing/human-review/worm-markets-retirement/R1)

test "$(git -C "$REVIEW_WT" rev-parse "$TASK_BASE^{commit}")" = "$TASK_BASE"
test "$(git -C "$REVIEW_WT" rev-parse "$DOCS_GATE_SHA^{commit}")" = "$DOCS_GATE_SHA"
test "$(git -C "$REVIEW_WT" rev-parse "$PRODUCT_SHA^{commit}")" = "$PRODUCT_SHA"
test -n "$MATERIAL_SHA"
test "$(git -C "$REVIEW_WT" rev-parse "$MATERIAL_SHA^{commit}")" = "$MATERIAL_SHA"
git -C "$REVIEW_WT" merge-base --is-ancestor "$TASK_BASE" "$PRODUCT_SHA"
git -C "$REVIEW_WT" merge-base --is-ancestor "$DOCS_GATE_SHA" "$PRODUCT_SHA"
git -C "$REVIEW_WT" merge-base --is-ancestor "$PRODUCT_SHA" "$MATERIAL_SHA"

# R1 材料提交相对受审产品只能改变 docs。
git -C "$REVIEW_WT" diff --quiet "$PRODUCT_SHA" "$MATERIAL_SHA" -- . ':(exclude)docs'

# main 可包含 PRODUCT_SHA 之后的纯材料提交，不能硬要求 HEAD 等于产品提交。
git -C "$FIELD_MAIN" merge-base --is-ancestor "$PRODUCT_SHA" HEAD

# 核对 main 当前 checkout 的全部非文档／技能内容；明确排除另一任务的通知邮件工具。
git -C "$FIELD_MAIN" diff --quiet "$PRODUCT_SHA" -- . \
  ':(exclude)docs' ':(exclude).agents' ':(exclude).codex' \
  ':(exclude)AGENTS.md' ':(exclude)tools/task-completion-email/main.go'

# 扫描全仓未跟踪／忽略输入。只排除文档、技能、已知运行输出、当前 recipe
# 不读取的本地文件、另一个任务的邮件工具，以及下方单独做精确指纹核对的 vendor。
mapfile -d '' -t ALL_UNTRACKED_INPUTS < <(
  git -C "$FIELD_MAIN" ls-files -z --others --exclude-standard
)
mapfile -d '' -t ALL_IGNORED_INPUTS < <(
  git -C "$FIELD_MAIN" ls-files -z --others --ignored --exclude-standard
)
UNKNOWN_INPUTS=()
for path in "${ALL_UNTRACKED_INPUTS[@]}" "${ALL_IGNORED_INPUTS[@]}"; do
  case "$path" in
    docs/*|.agents/*|.codex/*|tools/task-completion-email/main.go) ;;
    .run/*|.tmp/*|.worktrees/*|.superpowers/*|dist/*) ;;
    ui/node_modules/*|ui/dist/*|ui/coverage/*|ui/junit.xml) ;;
    .impeccable/*|.playwright-mcp/*|.scratch/*) ;;
    vendor/*|.env|PLAN*.md|rerunreport.txt) ;;
    *) UNKNOWN_INPUTS+=("$path") ;;
  esac
done
if ((${#UNKNOWN_INPUTS[@]})); then
  printf 'unknown checkout input: %q\n' "${UNKNOWN_INPUTS[@]}"
  exit 1
fi

# vendor 被 Go 构建自动使用；核对全部相对路径及逐文件 SHA-256 的规范化摘要。
python3 - "$FIELD_MAIN/vendor" <<'PY_VENDOR_DIGEST'
import hashlib, json, pathlib, sys

root = pathlib.Path(sys.argv[1])
if not root.is_dir():
    raise SystemExit(f"vendor directory missing: {root}")
manifest = {}
for path in root.rglob("*"):
    if path.is_symlink():
        raise SystemExit(f"unexpected vendor symlink: {path}")
    if path.is_file():
        relative = path.relative_to(root).as_posix()
        manifest[relative] = hashlib.sha256(path.read_bytes()).hexdigest()
    elif not path.is_dir():
        raise SystemExit(f"unexpected vendor entry: {path}")
payload = json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()
digest = hashlib.sha256(payload).hexdigest()
if len(manifest) != 10872:
    raise SystemExit(f"vendor file count changed: {len(manifest)}")
expected = "5cd2a996eab7d974e7396fd05ceae5214552861d2b7c87e33fe4870c6d1321a1"
if digest != expected:
    raise SystemExit(f"vendor digest changed: {digest}")
print(f"vendor files={len(manifest)} digest={digest}")
PY_VENDOR_DIGEST

# 两个 ignored 本地运行输入必须仍是本轮核对过的 profile 与拒发替身。
PROFILE="$REVIEW_WT/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/field-full-stack.env"
TELEGRAM_HELPER="$REVIEW_WT/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/telegram-substitute.py"
test "$(sha256sum "$PROFILE" | awk '{print $1}')" = c1629c0b439579ec93decea15319f71a4a5a4058bc556bb41fe7fe8be43d42d0
test "$(sha256sum "$TELEGRAM_HELPER" | awk '{print $1}')" = debd3134fc2466c68f2ab33c41c186145d885603c0830ed51cd5a62790dc1c4e
~~~

正式 R1 文件提交后预期全部退出 0。本轮固定的三个完整 SHA 为：

~~~text
617bd6a26345905dc7125911ec72c1cee56afe0b
d66c0c3d07a16e3050600fd244e349cbcd79c575
1dfcb795dfdf3da1ecb1b63e4acc9677604d90a5
~~~

MATERIAL_SHA 不写入材料正文，执行命令时从正式目录解析，并在交付记录中记录完整值。<code>FIELD_MAIN</code> 可保留其他任务的文档、技能、已知构建／运行输出及明确排除的 task-completion-email 修改；<code>vendor/</code> 必须保持 10,872 个文件和上述规范化摘要。其他未知 checkout 输入或本地受控文件摘要不符时必须记为受阻，不得清理、覆盖或继续启动。

## 禁止操作

本轮人工审查以只读 UI、GET 请求和既有证据核对为限。

- 不得再次执行数据库或通知退役工具的 <code>--apply</code>，也不得手工 DROP DATABASE。
- 不得执行真实 Worm 下单、Finalize、Close、Cash Out、撤销凭据、组合保存、Run 启动或其他业务写操作。
- 不得在管理员 Notifications 页面点击发送测试通知。
- 不得执行 make run-reset、reset-instance、删除容器／数据卷或停止归属不明的进程。
- 三条历史详情路由在当前空 Trading 库没有真实 ID；不要为制造 ID 而写数据。
- 写入、恢复和历史详情只检查 CHK-004 指定的受控证据。

## 环境恢复

### 是否需要运行环境

CHK-001 至 CHK-004 只读取既有证据，不需要启动环境。CHK-005 至 CHK-008 需要恢复原 main default full-stack 和一个只借用其 PostgreSQL／Wallet 的 Worm Trading external borrower。所有本任务环境当前均已停止，数据卷保留。

前提：

- Docker 可用；<code>/home/yege/work/athena/.env</code> 与原 namespace 的三个数据卷仍在。
- Node 使用 <code>ui/.nvmrc</code> 的 24.14.1。
- 以下本地受控文件已经过版本核对中的 SHA-256 完整性检查：
  - <code>/home/yege/work/athena/.worktrees/worm-markets-retirement/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/field-full-stack.env</code>
  - <code>/home/yege/work/athena/.worktrees/worm-markets-retirement/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/telegram-substitute.py</code>
- field profile 指向 <code>http://127.0.0.1:61907</code> 的本地 Telegram 替身。替身只允许启动和轮询所需方法，对 sendMessage 等外发方法返回 403。

如受控文件缺失或摘要不符，将运行项记为受阻并请求 AI 从本轮证据恢复；不要覆盖文件或改用真实 Telegram API。

### 1. 执行时归属预检并创建本轮记录

在任何启动前从终端 C 执行。它要求同一 checkout 的 `full-stack` 状态为 stopped，固定应用／helper／borrower 端口均无监听，并以 noclobber 创建本轮唯一 active record。本轮目录直接创建在 W/evidence 下，权限为 700，文件受 umask 077 保护；成功或失败都保留。发现已有环境、端口或旧记录时立即记为受阻；不得借用、停止或覆盖它们。

~~~bash
set -e
umask 077
FIELD_MAIN=/home/yege/work/athena
REVIEW_WT=/home/yege/work/athena/.worktrees/worm-markets-retirement
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
STATE="$FIELD_MAIN/.run/instances/full-stack/state.json"
EVIDENCE_ROOT="$REVIEW_WT/.superpowers/sdd/2026-09-16-worm-trading-market-query/evidence"
PROFILE_SOURCE="$REVIEW_WT/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/field-full-stack.env"
HELPER_SOURCE="$REVIEW_WT/.superpowers/sdd/2026-09-16-worm-trading-market-query/runtime/telegram-substitute.py"

test ! -e "$ACTIVE_RECORD"
test -f "$STATE"
make -C "$FIELD_MAIN" runtime-status INSTANCE=full-stack
test "$(jq -r '.Phase' "$STATE")" = stopped

for port in 4000 8080 8086 8088 8090 8108 8122 61907; do
  if ss -ltnH "sport = :$port" | grep -q .; then
    echo "preflight port already in use: $port" >&2
    exit 1
  fi
done

test "$(sha256sum "$PROFILE_SOURCE" | awk '{print $1}')" = c1629c0b439579ec93decea15319f71a4a5a4058bc556bb41fe7fe8be43d42d0
test "$(sha256sum "$HELPER_SOURCE" | awk '{print $1}')" = debd3134fc2466c68f2ab33c41c186145d885603c0830ed51cd5a62790dc1c4e
test -d "$EVIDENCE_ROOT"
R1_RUN_DIR=$(mktemp -d "$EVIDENCE_ROOT/human-review-R1-XXXXXXXX")
BORROWER_INSTANCE="worm-retirement-human-r1-$(date -u +%Y%m%dT%H%M%SZ)-$$"
install -m 600 "$PROFILE_SOURCE" "$R1_RUN_DIR/field-full-stack.env"
install -m 700 "$HELPER_SOURCE" "$R1_RUN_DIR/telegram-substitute.py"
sha256sum "$R1_RUN_DIR/field-full-stack.env" "$R1_RUN_DIR/telegram-substitute.py" > "$R1_RUN_DIR/input-sha256.txt"
printf '%s\n' "$BORROWER_INSTANCE" > "$R1_RUN_DIR/borrower-instance"
cp "$STATE" "$R1_RUN_DIR/preflight-full-stack-state.json"
jq '{checkedAt: (now | todateiso8601), Key, RunID, Phase, Supervisor}' "$STATE" > "$R1_RUN_DIR/preflight-full-stack.json"
if ! (set -o noclobber; printf '%s\n' "$R1_RUN_DIR" > "$ACTIVE_RECORD") 2>/dev/null; then
  echo "another R1 run record already exists; new blocked evidence retained at $R1_RUN_DIR" >&2
  exit 1
fi
printf 'active record: %s\nborrower instance: %s\nevidence directory: %s\n' \
  "$R1_RUN_DIR" "$BORROWER_INSTANCE" "$R1_RUN_DIR"
~~~

active record 存在后，所有终端都先从该文件加载本轮目录，不自行重建名称：

~~~bash
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
BORROWER_INSTANCE=$(cat "$R1_RUN_DIR/borrower-instance")
BORROWER_ENV="$R1_RUN_DIR/trading.env"
export R1_RUN_DIR BORROWER_INSTANCE BORROWER_ENV
~~~

### 2. 启动拒发 Telegram 替身

终端 A 从 active record 加载目录，直接启动 helper 子进程并记录 PID、start ticks 与 cwd。只有该直接子进程仍存活且 61907 已监听时才写 `helper-owned.json`；终端保持等待。

~~~bash
set -e
REVIEW_WT=/home/yege/work/athena/.worktrees/worm-markets-retirement
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
cd "$REVIEW_WT"
python3 -u "$R1_RUN_DIR/telegram-substitute.py" > "$R1_RUN_DIR/helper.log" 2>&1 &
HELPER_PID=$!
for _ in $(seq 1 50); do
  kill -0 "$HELPER_PID" 2>/dev/null || { cat "$R1_RUN_DIR/helper.log" >&2; exit 1; }
  ss -ltnpH 'sport = :61907' | grep -Fq "pid=$HELPER_PID," && break
  sleep 0.1
done
ss -ltnpH 'sport = :61907' | grep -Fq "pid=$HELPER_PID,"
HELPER_START_TICKS=$(awk '{print $22}' "/proc/$HELPER_PID/stat")
HELPER_CWD=$(readlink -f "/proc/$HELPER_PID/cwd")
jq -n --argjson pid "$HELPER_PID" --argjson startTicks "$HELPER_START_TICKS" --arg cwd "$HELPER_CWD" \
  '{pid: $pid, startTicks: $startTicks, cwd: $cwd}' > "$R1_RUN_DIR/helper-owned.json"
wait "$HELPER_PID"
~~~

另一个终端核对它确实拒发，预期 HTTP 403；这不会向 Telegram 外发：

~~~bash
curl -sS -o /dev/null -w '%{http_code}\n'   -X POST http://127.0.0.1:61907/bot-r1/sendMessage   -d chat_id=1 -d text=blocked
~~~

### 3. 启动原 main default full-stack

终端 B 从现场 checkout 启动。命令复用原 namespace 的保留卷；基础设施端口由 Docker 动态分配，禁止写死先前的 50124 或 59370。

~~~bash
set -e
FIELD_MAIN=/home/yege/work/athena
REVIEW_WT=/home/yege/work/athena/.worktrees/worm-markets-retirement
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
cd "$FIELD_MAIN"
source "$HOME/.nvm/nvm.sh"
nvm use "$(cat ui/.nvmrc)"
LAUNCHER_PID=$$
LAUNCHER_START_TICKS=$(awk '{print $22}' "/proc/$LAUNCHER_PID/stat")
LAUNCHER_CWD=$(readlink -f "/proc/$LAUNCHER_PID/cwd")
jq -n --argjson pid "$LAUNCHER_PID" --argjson startTicks "$LAUNCHER_START_TICKS" --arg cwd "$LAUNCHER_CWD" \
  '{pid: $pid, startTicks: $startTicks, cwd: $cwd}' > "$R1_RUN_DIR/full-stack-launcher.json"
ATHENA_NOTIFICATION_TELEGRAM_API_URL=http://127.0.0.1:61907 make run INSTANCE=full-stack   ENV_FILE="$R1_RUN_DIR/field-full-stack.env"
~~~

报告 <code>instance full-stack is ready</code> 后，在终端 C 核对当前 supervisor 是上述 launcher 的后代、state 身份与实际进程一致，才写入 `full-stack-owned.json` 和 owned-start 标记。如果 `make run` 失败、launcher 身份已变或 supervisor 不是其后代，不得写 owned 标记、不得停止当前 `full-stack`，记录受阻并保留 active record。

~~~bash
set -e
FIELD_MAIN=/home/yege/work/athena
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
STATE="$FIELD_MAIN/.run/instances/full-stack/state.json"
LAUNCHER_PID=$(jq -r '.pid' "$R1_RUN_DIR/full-stack-launcher.json")
LAUNCHER_START_TICKS=$(jq -r '.startTicks' "$R1_RUN_DIR/full-stack-launcher.json")
test "$(awk '{print $22}' "/proc/$LAUNCHER_PID/stat")" = "$LAUNCHER_START_TICKS"

cd "$FIELD_MAIN"
make runtime-status INSTANCE=full-stack
test "$(jq -r '.Phase' "$STATE")" = running
RUN_ID=$(jq -r '.RunID' "$STATE")
SUPERVISOR_PID=$(jq -r '.Supervisor.PID' "$STATE")
test "$RUN_ID" = "$(jq -r '.Supervisor.RunID' "$STATE")"
test "$(awk '{print $22}' "/proc/$SUPERVISOR_PID/stat")" = "$(jq -r '.Supervisor.StartTicks' "$STATE")"
test "$(readlink -f "/proc/$SUPERVISOR_PID/exe")" = "$(jq -r '.Supervisor.Exe' "$STATE")"
test "$(tr '\0' '\n' < "/proc/$SUPERVISOR_PID/environ" | awk -F= '$1 == "ATHENA_LOCAL_RUNTIME_RUN_ID" {print $2}')" = "$RUN_ID"

CURRENT_PID=$SUPERVISOR_PID
DESCENDANT=false
while test "$CURRENT_PID" -gt 1; do
  if test "$CURRENT_PID" = "$LAUNCHER_PID"; then DESCENDANT=true; break; fi
  CURRENT_PID=$(awk '{print $4}' "/proc/$CURRENT_PID/stat")
done
test "$DESCENDANT" = true
cp "$STATE" "$R1_RUN_DIR/full-stack-owned-state.json"
jq '{Key, RunID, Phase, Supervisor, Endpoints}' "$STATE" > "$R1_RUN_DIR/full-stack-owned.json"
touch "$R1_RUN_DIR/full-stack-owned-start"

jq '.Endpoints | {postgres, redis, minio}' .run/instances/full-stack/state.json
jq -r '.ATHENA_ACCOUNT_STATE_POSTGRES_DSN' .run/instances/full-stack/environment.json
curl -fsS -o /dev/null -w 'ui member bootstrap %{http_code}\n'   -H 'X-Athena-Application-Realm: member'   http://127.0.0.1:4000/api/v1/app/bootstrap
curl -fsS -o /dev/null -w 'api admin bootstrap %{http_code}\n'   -H 'X-Athena-Application-Realm: admin'   http://127.0.0.1:8080/api/v1/app/bootstrap
~~~

预期 status 为 running，两个 bootstrap 都是 200。UI 为 <code>http://127.0.0.1:4000</code>，API 为 <code>http://127.0.0.1:8080</code>；PostgreSQL／Redis／MinIO 动态端口以本次 <code>state.json.Endpoints</code> 为准。environment.json 的 DSN 只用于派生 borrower 连接，不当作端口表。

### 4. 生成并启动 external Trading borrower

default full-stack 不内置 Worm Trading。每次恢复都创建带 UTC 时间戳和 PID 的 owned borrower 名称；不能复用固定 instance，因为 full-stack 正常 stop/start 会重新分配动态 PostgreSQL 端口，而旧 external instance 保留的是前一轮 DSN 指纹。

从 active record 重新加载本轮唯一名称：

~~~bash
set -e
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
BORROWER_INSTANCE=$(cat "$R1_RUN_DIR/borrower-instance")
BORROWER_ENV="$R1_RUN_DIR/trading.env"
export R1_RUN_DIR BORROWER_INSTANCE BORROWER_ENV
~~~

下面脚本从本次动态 account DSN 推导同一 server 的 worm_trading DSN，并复用 field profile 中已核对的稳定 credential key，以及 full-stack runtime 中的内部 token 和 Wallet signer。输出只保存在本轮唯一目录。

~~~bash
set -e
FIELD_MAIN=/home/yege/work/athena
python3 - "$R1_RUN_DIR/field-full-stack.env"   "$FIELD_MAIN/.run/instances/full-stack/environment.json" "$BORROWER_ENV" <<'PY_BORROWER_ENV'
import json, pathlib, shlex, sys, urllib.parse
profile, runtime_path, output = map(pathlib.Path, sys.argv[1:])
runtime = json.loads(runtime_path.read_text())
account_dsn = runtime["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"]
parts = urllib.parse.urlsplit(account_dsn)
trading_dsn = urllib.parse.urlunsplit(parts._replace(path="/worm_trading"))
overrides = {
    "ATHENA_ACCOUNT_STATE_POSTGRES_DSN": account_dsn,
    "ATHENA_WORM_TRADING_POSTGRES_DSN": trading_dsn,
    "ATHENA_WALLET_SERVER_ADDRESS": runtime["ATHENA_WALLET_SERVER_ADDRESS"],
    "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN": runtime["ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"],
    "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN": runtime["ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN"],
    "ATHENA_WORM_TRADING_LISTEN_ADDRESS": "127.0.0.1",
    "ATHENA_WORM_TRADING_PORT": "8090",
    "ATHENA_WORM_TRADING_CATALOG_BUDGET": "45s",
    "ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT": "5s",
}
body = profile.read_text() + "\n# R1 dynamic external borrower\n"
body += "".join(f"{key}={shlex.quote(value)}\n" for key, value in overrides.items())
output.write_text(body)
output.chmod(0o600)
print(output)
print(account_dsn.rsplit("@", 1)[-1].split("/", 1)[0])
PY_BORROWER_ENV
~~~

终端 D 使用保存的唯一名称启动 borrower：

~~~bash
set -e
cd /home/yege/work/athena
make run-service SERVICE=worm-trading INSTANCE="$BORROWER_INSTANCE"   DB_MODE=external ENV_FILE="$BORROWER_ENV"
~~~

报告 ready 后，终端 C 从同一 active record 加载名称并核对：

~~~bash
set -e
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
BORROWER_INSTANCE=$(cat "$R1_RUN_DIR/borrower-instance")
cd /home/yege/work/athena
make runtime-status INSTANCE="$BORROWER_INSTANCE"
BORROWER_STATE="$PWD/.run/instances/$BORROWER_INSTANCE/state.json"
test "$(jq -r '.Phase' "$BORROWER_STATE")" = running
cp "$BORROWER_STATE" "$R1_RUN_DIR/borrower-owned-state.json"
jq '{Key, RunID, Phase, Supervisor}' "$BORROWER_STATE" > "$R1_RUN_DIR/borrower-owned.json"
touch "$R1_RUN_DIR/borrower-owned-start"
curl -fsS -o /dev/null -w 'catalog %{http_code}\n'   -H 'X-Athena-Application-Realm: member'   http://127.0.0.1:4000/api/v1/worm-trading/events/2igwY6nZGmC8S3apwxx3N8C9xyLKuQZ6tT2KmedtTZBF
~~~

预期 borrower 为 running、DBMode=external、没有 owned database/container 资源，catalog HTTP 为 200。当前受控现场曾返回 3 个子市场；供应商内容可能变化，应记录本轮实际响应。只要 HTTP/结构仍符合契约，子市场数量变化不自动判为代码失败；供应商不可用则记录受阻和响应。

日志和状态分别在 <code>/home/yege/work/athena/.run/instances/full-stack/</code> 与本轮 <code>/home/yege/work/athena/.run/instances/$BORROWER_INSTANCE/</code>。

## 身份与测试数据

| 用途 | 身份／权限 | 测试数据及准备方法 | 预期初始状态 |
| --- | --- | --- | --- |
| 会员只读 UI | development member local-user，account <code>13eb9bc6-dfb7-49e8-b0f3-312f0f50b9a3</code> | 打开 <code>http://127.0.0.1:4000/</code>；DisableAuth=true | revision 2；七模块；Worm Trading RW；Trading 业务表为空 |
| 管理员只读 UI | development admin local-admin，account <code>499cb821-f7c3-4eac-90d7-9f083c4d3b8b</code> | 打开 <code>http://127.0.0.1:4000/admin/</code> | administrator=true；revision 2；可读账户目录 |
| Catalog | 同一 member | Event ID <code>2igwY6nZGmC8S3apwxx3N8C9xyLKuQZ6tT2KmedtTZBF</code> | 当前受控现场曾返回 3 个子市场；人工轮记录实际响应，供应商不可用时记受阻 |
| 退役数据 | 只读证据 | worm_markets 原 OID 17152、owner athena、server identity 7685665987292844070 | 已 absent；九个保留库存在 |
| 通知 | admin 只读及证据 | 固定五个 worm-markets.* 来源 | pending/sending/cancelled/retired_total 均 0；不发送测试通知 |

development 身份是真实应用 DisableAuth 身份，但没有覆盖 Google 或 Phantom 的交互登录。

## 检查顺序

### CHK-001：核对精确数据库退役和九库保留证据

- 设计依据：退役设计“单一现场目标、完整 retained inventory、无 FORCE”。
- 前置条件：只需本地证据文件，不启动服务。
- 操作步骤：
  1. 打开 W/evidence/field-data-report.json、field-data-apply.json、field-data-after.json。
  2. 打开 field-database-inventory-post-drop.json 与 field-database-inventory-after-restart.json。
  3. W 为 <code>/home/yege/work/athena/.worktrees/worm-markets-retirement/.superpowers/sdd/2026-09-16-worm-trading-market-query</code>。
- 明确预期：
  1. apply 前 target 为 worm_markets、OID 17152、owner athena、server identity 7685665987292844070、active connections 0，九个 retained DSN 均核验。
  2. apply 只删除这一库；after 为 already_absent。
  3. 重启后仍只有 postgres、athena、temporal、temporal_visibility、worm_trading、wallet、managed_oo、profit_sharing、token，没有重建 Markets。
- 记录方式：记录字段和证据文件名；任何额外删除、身份不符或重建即失败。
- 影响与恢复：无；只读文件。禁止重跑 apply。

### CHK-002：核对两账户迁移为七模块且其他权限不变

- 设计依据：[账户权限设计](../../../../design/identity-access/account-access-control.md)。
- 前置条件：只读 W/evidence/field-grants-before.json 与 field-account-migration-verified.json。
- 操作步骤：
  1. 对照迁移前 16 条 grant 与迁移后 14 条 grant。
  2. 检查两个账户的 revision、flags 和模块名。
- 明确预期：
  1. 每个账户只删除 worm_markets 一条 grant；其余 14 条值不变，revision 从 1 精确变为 2。
  2. 七模块恰为 Market Radar、Managed OO、Worm Trading、Token、Solana、Wallet、Trader Sync。
  3. Markets 权限没有转换成 Trading 权限；login/api-key/profit-sharing flags 保持。
- 记录方式：记录两个 account UUID、七模块和对比结论。
- 影响与恢复：无；只读文件。

### CHK-003：核对五来源通知维护且没有外发

- 设计依据：[系统通知运维设计](../../../../design/notifications/system-notification-operations.md)。
- 前置条件：只读 W/evidence/field-notifications-report.json、field-notifications-apply.json、field-notifications-after.json、field-shared-notification-preserved.json。
- 操作步骤：
  1. 核对 report/apply/report 的退出码、五个固定来源和计数。
  2. 核对共享 Topic、delivery、attempt、consumed、sender 和 Bot offset。
- 明确预期：
  1. 三次 exit 0；pending、sending、cancelled、retired_total 均为 0，counts_verified=true。
  2. delivery 0、attempt 0、Topic 1、consumed 1、原两个 sender 指纹和 next_update_id=487823039 保持。
  3. 证据明确没有测试消息或替代消息外发。
- 记录方式：记录计数与证据路径。
- 影响与恢复：无；禁止点击或调用发送测试通知。

### CHK-004：人工核对受控写入／恢复证据及现场限制

- 设计依据：[验收记录](../../../worm-markets-retirement-acceptance.md)中的“隔离后端与 HTTP 证据”和“UI 与真实环境”。
- 前置条件：只读验收记录、W/task-9-report.md 与其引用日志。
- 操作步骤：
  1. 核对隔离 PostgreSQL 测试覆盖组合写入、Preview、Open/Finalize/Close、崩溃恢复、unknown/reconcile、批量 Cash Out 和授权撤销。
  2. 核对 44 次受控 UI 场景覆盖七条路由，包括 edit/preview/detail。
  3. 核对现场限制：真实 Trading 业务表为空、没有旧凭据、三条历史详情没有真实 ID。
- 明确预期：
  1. 写入和恢复只来自受控 fixture／隔离数据库与 fake 外部边界；没有真实 Worm 下单、Close 或 revoke。
  2. 三条历史详情只能沿用 Task 9 受控证据；本轮现场不得造数。
  3. 旧凭据解密现场验证因没有旧凭据而受阻，不能标成通过。
- 记录方式：若认可受控证据，记录测试/日志；旧凭据和真实历史详情分别记“受阻（无数据）”或“证据核对完成”，不得写成现场操作通过。
- 影响与恢复：无；只读证据。

### CHK-005：会员只读 UI、Catalog 与导航

- 设计依据：[Worm Trading 设计](../../../../design/trading/worm-trading.md)与已批准 v22 页面设计。
- 前置条件：full-stack 与 borrower ready；打开 <code>http://127.0.0.1:4000/</code>，身份为 local-user。
- 操作步骤：
  1. 依次打开 /worm-trading、/worm-trading/combinations、/worm-trading/combinations/new、/worm-trading/executions。
  2. 在 New combination 的 Event ID 输入框填指定 Event ID，点击 Add event；不要点击 Save。
  3. 桌面和手机宽度各观察一次主导航。
- 明确预期：
  1. 四页有正确一级标题、深色主题，无 Page not found。
  2. Catalog 只读加载一个事件和结构有效的子市场；没有业务写入。当前受控现场曾有 3 个，供应商数量变化记录实际响应，不自动判代码失败；供应商不可用记受阻。
  3. 主导航只有 Assets、Combinations、Executions 三项 Worm Trading 入口，没有 Worm Markets。
- 记录方式：保存桌面／手机截图、URL、身份和 Event ID。
- 影响与恢复：输入仅在浏览器草稿；刷新清除。不得 Save。

### CHK-006：核对当前 Trading 记录和三条历史详情不可现场覆盖

- 设计依据：[验收记录](../../../worm-markets-retirement-acceptance.md)“UI 与真实环境”。
- 前置条件：member UI 正常；禁止创建记录。
- 操作步骤：
  1. 查看 Combinations、Executions，以及 GET /api/v1/worm-trading/wallet-connections?page=1&pageSize=100。
  2. 记录列表当前实际数量；不构造 combination/run ID。
- 明确预期：
  1. 当前 Trading 业务记录为空，三个列表 items 都为空；这是实际数据状态。
  2. 因无 combination/execution，/combinations/:id/edit、/combinations/:id/execute、/executions/:id 三条详情无法现场检查；其覆盖只在 CHK-004。
- 记录方式：空态截图和 GET 响应计数；三条详情记为受阻（无历史数据）。
- 影响与恢复：无；只读。

### CHK-007：管理员核对七模块权限目录

- 设计依据：[账户权限设计](../../../../design/identity-access/account-access-control.md)。
- 前置条件：打开 <code>http://127.0.0.1:4000/admin/accounts</code>，身份为 local-admin。
- 操作步骤：
  1. 选择 local-user，打开 Access。
  2. 只查看模块列表、access level、revision 和 flags；不要编辑或保存。
  3. 手机宽度再次打开账户详情。
- 明确预期：
  1. 恰有七个模块：Market Radar、Managed OO、Worm Trading、Token、Solana、Wallet、Trader Sync。
  2. local-user levels 与 CHK-002 相同；revision 为 2。
  3. 没有 Worm Markets 行，桌面和手机均可读。
- 记录方式：桌面／手机截图及七行文字记录。
- 影响与恢复：无；只读，不保存。

### CHK-008：旧 Markets HTTP/UI 缺失与通知页面只读状态

- 设计依据：[退役需求](../../../../requirements/development-runtime/worm-markets-removal.md)与[系统通知运维设计](../../../../design/notifications/system-notification-operations.md)。
- 前置条件：full-stack 与 borrower 正常；admin 页面可用。
- 操作步骤：
  1. GET /api/v1/worm-markets/status、/api/v1/worm-markets/events/11111111111111111111111111111111、/api/v1/worm-markets/events。
  2. 在会员和管理员导航搜索 “Worm Markets”。
  3. 打开 <code>http://127.0.0.1:4000/admin/notifications</code>，只查看，不点击发送测试通知。
- 明确预期：
  1. 三个旧 HTTP 路径均为 404，且没有旧 UI 入口。
  2. Notifications 页面可加载；当前 delivery 为空，不出现 worm-markets.* 记录。
  3. Telegram 替身外发探针保持 403。
- 记录方式：保存三个状态码、导航截图和 Notifications 空态截图。
- 影响与恢复：无；全程 GET/页面读取。

## 修复轮复验范围

| 分类 | 本轮检查 | 原因或沿用依据 |
| --- | --- | --- |
| 修复问题 | 不适用 | R1 首次交付 |
| 受影响流程 | CHK-001 至 CHK-008 | 覆盖退役、权限、通知、只读 UI/catalog 和受控写入/恢复证据 |
| 先前受阻／未执行 | 旧凭据解密与三条现场历史详情仍受阻 | 原 main Trading 业务表为空；不得造数 |
| 沿用历史结果 | 不适用 | R1 不把历史 AI 结果写成人工通过 |

## 本轮现场收尾

仅执行 CHK-001 至 CHK-004 时无需停止资源。执行运行项后，先完成全部身份核对；任一 owned-start 标记缺失、RunID／supervisor 或 helper 身份变化时，不执行任何 stop，保留 active record 和证据并记为受阻。全部一致后才按 borrower → full-stack → Telegram 替身收尾。

~~~bash
set -e
ACTIVE_RECORD=/tmp/worm-markets-retirement-R1-active
R1_RUN_DIR=$(cat "$ACTIVE_RECORD")
BORROWER_INSTANCE=$(cat "$R1_RUN_DIR/borrower-instance")
BORROWER_ENV="$R1_RUN_DIR/trading.env"
FIELD_MAIN=/home/yege/work/athena
BORROWER_STATE="$FIELD_MAIN/.run/instances/$BORROWER_INSTANCE/state.json"
FULL_STATE="$FIELD_MAIN/.run/instances/full-stack/state.json"

test -f "$R1_RUN_DIR/borrower-owned-start"
test -f "$R1_RUN_DIR/full-stack-owned-start"
test "$(jq -r '.Phase' "$BORROWER_STATE")" = running
test "$(jq -r '.RunID' "$BORROWER_STATE")" = "$(jq -r '.RunID' "$R1_RUN_DIR/borrower-owned.json")"
test "$(jq -S -c '.Supervisor' "$BORROWER_STATE")" = "$(jq -S -c '.Supervisor' "$R1_RUN_DIR/borrower-owned.json")"
test "$(jq -r '.Phase' "$FULL_STATE")" = running
test "$(jq -r '.RunID' "$FULL_STATE")" = "$(jq -r '.RunID' "$R1_RUN_DIR/full-stack-owned.json")"
test "$(jq -S -c '.Supervisor' "$FULL_STATE")" = "$(jq -S -c '.Supervisor' "$R1_RUN_DIR/full-stack-owned.json")"

HELPER_PID=$(jq -r '.pid' "$R1_RUN_DIR/helper-owned.json")
HELPER_START_TICKS=$(jq -r '.startTicks' "$R1_RUN_DIR/helper-owned.json")
HELPER_CWD=$(jq -r '.cwd' "$R1_RUN_DIR/helper-owned.json")
test "$(awk '{print $22}' "/proc/$HELPER_PID/stat")" = "$HELPER_START_TICKS"
test "$(readlink -f "/proc/$HELPER_PID/cwd")" = "$HELPER_CWD"
mapfile -t DYNAMIC_PORTS < <(
  jq -r '.Endpoints | [.postgres, .redis, .minio][] | split(":")[-1]' "$R1_RUN_DIR/full-stack-owned.json"
)

cd "$FIELD_MAIN"
make stop-instance INSTANCE="$BORROWER_INSTANCE" > "$R1_RUN_DIR/borrower-stop.log" 2>&1
make runtime-status INSTANCE="$BORROWER_INSTANCE" > "$R1_RUN_DIR/borrower-status-after-stop.log" 2>&1
test "$(jq -r '.Phase' "$BORROWER_STATE")" = stopped
cp "$BORROWER_STATE" "$R1_RUN_DIR/borrower-after-stop-state.json"
make stop INSTANCE=full-stack > "$R1_RUN_DIR/full-stack-stop.log" 2>&1
make runtime-status INSTANCE=full-stack > "$R1_RUN_DIR/full-stack-status-after-stop.log" 2>&1
test "$(jq -r '.Phase' "$FULL_STATE")" = stopped
cp "$FULL_STATE" "$R1_RUN_DIR/full-stack-after-stop-state.json"
test "$(awk '{print $22}' "/proc/$HELPER_PID/stat")" = "$HELPER_START_TICKS"
test "$(readlink -f "/proc/$HELPER_PID/cwd")" = "$HELPER_CWD"
kill -TERM "$HELPER_PID"
for _ in $(seq 1 50); do
  test ! -e "/proc/$HELPER_PID" && break
  sleep 0.1
done
test ! -e "/proc/$HELPER_PID"
jq -n --argjson pid "$HELPER_PID" --arg stoppedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '{pid: $pid, exited: true, stoppedAt: $stoppedAt}' > "$R1_RUN_DIR/helper-after-stop.json"

: > "$R1_RUN_DIR/post-stop-ports.txt"
for port in 4000 8080 8086 8088 8090 8108 8122 61907 "${DYNAMIC_PORTS[@]}"; do
  if ss -ltnH "sport = :$port" | grep -q .; then
    echo "still-listening $port" >&2
    exit 1
  fi
  printf 'free %s\n' "$port" >> "$R1_RUN_DIR/post-stop-ports.txt"
done

test "$(cat "$ACTIVE_RECORD")" = "$R1_RUN_DIR"
jq -n \
  --arg completedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg evidenceDirectory "$R1_RUN_DIR" \
  --arg borrowerInstance "$BORROWER_INSTANCE" \
  --arg borrowerRunID "$(jq -r '.RunID' "$R1_RUN_DIR/borrower-owned.json")" \
  --arg fullStackRunID "$(jq -r '.RunID' "$R1_RUN_DIR/full-stack-owned.json")" \
  '{completedAt: $completedAt, evidenceDirectory: $evidenceDirectory,
    borrowerInstance: $borrowerInstance, borrowerRunID: $borrowerRunID,
    fullStackRunID: $fullStackRunID, helperExited: true,
    borrowerPhase: "stopped", fullStackPhase: "stopped", portsReleased: true}' \
  > "$R1_RUN_DIR/cleanup-status.json"
unlink "$ACTIVE_RECORD"
printf 'cleanup complete; evidence retained at %s\n' "$R1_RUN_DIR"
~~~

终端 A 的 `wait` 此时应以 SIGTERM 状态返回。停止块在移除 stable active 指针前已经从本轮 <code>full-stack-owned.json</code> 读取动态 PostgreSQL／Redis／MinIO 端口，并连同固定端口逐一确认无监听，不使用历史端口常量。停止命令保留原三个卷，禁止 reset。成功时只删除 <code>/tmp</code> 下的 active 指针；W/evidence 中权限受限的本轮目录、输入 profile／helper、日志、完整前后 state、身份记录、owned-start 标记、端口与结束状态全部保留，并打印证据路径。任何身份、停止或端口核对失败都保存两个 instance 目录、active record 和上述证据并记录受阻，不终止归属不明进程。
