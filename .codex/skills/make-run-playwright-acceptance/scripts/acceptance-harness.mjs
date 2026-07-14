import fs from "node:fs/promises";
import path from "node:path";

export function redact(value) {
  return String(value ?? "")
    .replace(/\b(?:Bearer\s+)?[A-Za-z0-9._~-]{24,}\b/gi, "<redacted>")
    .replace(/([?&](?:X-Amz-[^=]+|token|signature|credential)=)[^&\s]+/gi, "$1<redacted>")
    .slice(0, 800);
}

function sanitize(value) {
  if (typeof value === "string") return redact(value);
  if (Array.isArray(value)) return value.map(sanitize);
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, sanitize(item)]));
  }
  return value;
}

function urlPath(value) {
  try { return new URL(value).pathname; } catch { return redact(value); }
}

export async function createAcceptanceHarness({ name, artifactRoot = ".tmp", metadata = {} }) {
  const startedAt = new Date();
  const stamp = startedAt.toISOString().replace(/[-:TZ.]/g, "").slice(0, 14);
  const runDir = path.resolve(artifactRoot, `${name}-${stamp}`);
  const screenshotDir = path.join(runDir, "screenshots");
  await fs.mkdir(screenshotDir, { recursive: true });

  const report = {
    started_at: startedAt.toISOString(), completed_at: null, metadata: sanitize(metadata),
    steps: [], issues: [], console: [], network: [], expected_failures: [],
  };
  const tracedContexts = [];

  async function screenshot(page, nameSuffix) {
    const file = path.join(screenshotDir, `${nameSuffix}.png`);
    await page.screenshot({ path: file, fullPage: true });
    return path.relative(runDir, file);
  }

  function issue({ id, severity = "P2", kind = "product_defect", page = "", expected, actual, evidence = {} }) {
    report.issues.push({ id, severity, kind, page: urlPath(page), expected: redact(expected), actual: redact(actual), evidence: sanitize(evidence) });
  }

  function expectedFailure({ role, kind, path: requestPath, status, note = "" }) {
    report.expected_failures.push({ role, kind, path: urlPath(requestPath), status, note: redact(note) });
  }

  async function step(nameValue, fn, page, { continueOnFailure = true } = {}) {
    const entry = { name: nameValue, status: "running", started_at: new Date().toISOString() };
    report.steps.push(entry);
    try {
      const detail = await fn();
      entry.status = "passed";
      if (detail !== undefined) entry.detail = sanitize(detail);
      return detail;
    } catch (error) {
      entry.status = "failed";
      entry.error = redact(error?.message || error);
      if (page) { try { entry.screenshot = await screenshot(page, `failure-${report.steps.length}`); } catch {} }
      if (!continueOnFailure) throw error;
      return null;
    } finally {
      entry.completed_at = new Date().toISOString();
    }
  }

  function observe(page, role, { relevantPath = () => true } = {}) {
    page.on("console", (msg) => {
      if (["error", "warning", "warn"].includes(msg.type())) report.console.push({ role, type: msg.type(), page: urlPath(page.url()), text: redact(msg.text()) });
    });
    page.on("pageerror", (error) => report.console.push({ role, type: "pageerror", page: urlPath(page.url()), text: redact(error.message) }));
    page.on("requestfailed", (request) => report.network.push({ role, method: request.method(), path: urlPath(request.url()), status: "failed", reason: redact(request.failure()?.errorText) }));
    page.on("response", (response) => {
      const requestPath = urlPath(response.url());
      if (relevantPath(requestPath) || response.status() >= 400) report.network.push({ role, method: response.request().method(), path: requestPath, status: response.status() });
    });
  }

  async function startTrace(context, role) {
    await context.tracing.start({ screenshots: true, snapshots: true, sources: false });
    tracedContexts.push({ context, role });
  }

  async function finish({ coverage = [], commands = [], mutations = [], database = [], fixes = [], constraints = [], notes = [] } = {}) {
    for (const { context, role } of tracedContexts) {
      try { await context.tracing.stop({ path: path.join(runDir, `${role}-trace.zip`) }); } catch (error) { report.console.push({ role, type: "trace", page: "", text: redact(error.message) }); }
    }
    report.completed_at = new Date().toISOString();
    report.summary = {
      passed: report.steps.filter((entry) => entry.status === "passed").length,
      failed: report.steps.filter((entry) => entry.status === "failed").length,
      issues: report.issues.length,
    };
    report.constraints = sanitize(constraints);
    report.coverage = sanitize(coverage);
    report.commands = sanitize(commands);
    report.mutations = sanitize(mutations);
    report.database = sanitize(database);
    report.fixes = sanitize(fixes);
    report.notes = sanitize(notes);
    await fs.writeFile(path.join(runDir, "report.json"), JSON.stringify(report, null, 2));
    const lines = [
      `# ${name} 验收报告`, "", `执行时间：${report.started_at}`, "", "## 结论", "",
      `通过 ${report.summary.passed} 项，失败 ${report.summary.failed} 项，记录问题 ${report.summary.issues} 个。`, "", "## 步骤", "",
      ...report.steps.map((entry) => `- ${entry.status === "passed" ? "通过" : "失败"}：${entry.name}${entry.error ? ` — ${entry.error}` : ""}`), "", "## 问题", "",
      ...(report.issues.length ? report.issues.map((entry) => `- ${entry.id} / ${entry.severity} / ${entry.kind}：${entry.actual}`) : ["- 无"]), "", "## 覆盖边界", "",
      ...(coverage.length ? coverage.map((entry) => `- ${redact(entry)}`) : ["- 未记录"]), "", "## 执行命令", "",
      ...(commands.length ? commands.map((entry) => `- \`${redact(entry)}\``) : ["- 未记录"]), "", "## 数据变更", "",
      ...(mutations.length ? mutations.map((entry) => `- ${redact(entry)}`) : ["- 无"]), "", "## 数据库交叉核对", "",
      ...(report.database.length ? report.database.map((entry) => `- ${entry}`) : ["- 未执行"]), "", "## 约束与环境", "",
      ...(constraints.length ? constraints.map((entry) => `- ${redact(entry)}`) : ["- 无"]), "", "## 修复与复验", "",
      ...(fixes.length ? fixes.map((entry) => `- ${redact(entry)}`) : ["- 未修改代码"]), "", "## 证据", "",
      "- `report.json`：步骤、问题、脱敏网络与控制台摘要", "- `*-trace.zip`：角色隔离的 Playwright trace", "- `screenshots/`：关键页面与失败证据", "",
      ...report.notes.map((entry) => `- ${entry}`), "",
    ];
    await fs.writeFile(path.join(runDir, "final-report.md"), lines.join("\n"));
    return { runDir, report };
  }

  return { runDir, report, step, issue, expectedFailure, observe, screenshot, startTrace, finish };
}
