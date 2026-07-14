import { chromium, request as playwrightRequest } from "../e2e/node_modules/playwright/index.mjs";
import { createAcceptanceHarness } from "../.codex/skills/make-run-playwright-acceptance/scripts/acceptance-harness.mjs";

const required = ["QA_ADMIN_USERNAME", "QA_ADMIN_PASSWORD", "QA_TEAM_USERNAME", "QA_TEAM_PASSWORD"];
for (const key of required) {
  if (!process.env[key]) throw new Error(`Missing ${key}`);
}

const harness = await createAcceptanceHarness({
  name: "replace-with-feature-name",
  metadata: {
    browser: "WSL Playwright + system Chrome",
    data_mode: "existing fixtures",
    target_id: process.env.QA_TARGET_ID || "not-required",
  },
});
const browser = await chromium.launch({
  executablePath: process.env.SERVICE_CORE_ACCEPTANCE_CHROME_BIN || "/usr/bin/google-chrome",
  headless: true,
});
const adminContext = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const teamContext = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
await harness.startTrace(adminContext, "admin");
await harness.startTrace(teamContext, "team");
const admin = await adminContext.newPage();
const team = await teamContext.newPage();
harness.observe(admin, "admin", { relevantPath: (value) => value.includes("replace-with-feature-path") });
harness.observe(team, "team", { relevantPath: (value) => value.includes("replace-with-feature-path") });

try {
  await harness.step("Admin real login", async () => {
    await admin.goto("http://127.0.0.1:5173/admin/login", { waitUntil: "networkidle" });
    await admin.getByPlaceholder("請輸入用戶名").fill(process.env.QA_ADMIN_USERNAME);
    await admin.getByPlaceholder("請輸入密碼").fill(process.env.QA_ADMIN_PASSWORD);
    await Promise.all([
      admin.waitForURL((url) => url.pathname !== "/admin/login"),
      admin.getByRole("button", { name: "登錄", exact: true }).click(),
    ]);
  }, admin);

  await harness.step("Team real login", async () => {
    await team.goto("http://127.0.0.1:5174/login", { waitUntil: "networkidle" });
    await team.locator("#login-email").fill(process.env.QA_TEAM_USERNAME);
    await team.locator("#password").fill(process.env.QA_TEAM_PASSWORD);
    await Promise.all([
      team.waitForURL((url) => !url.pathname.endsWith("/login")),
      team.getByRole("button", { name: "登錄", exact: true }).click(),
    ]);
  }, team);

  // Add narrowly scoped real UI/API steps here. Use page.route only for unavailable display states.
  // Use playwrightRequest.newContext for complementary permission/idempotency/header checks.
  void playwrightRequest;
} finally {
  await harness.finish({
    coverage: ["Replace with real-backend and intercepted-state boundaries"],
    commands: ["Replace with commands actually executed"],
    mutations: ["No business data mutation"],
    constraints: ["Replace with actual environment limitations"],
    database: ["Replace with read-only database cross-check results"],
    fixes: ["No code changes during acceptance"],
    notes: ["State which paths used real backend data and which used intercepted responses."],
  });
  await browser.close();
}
