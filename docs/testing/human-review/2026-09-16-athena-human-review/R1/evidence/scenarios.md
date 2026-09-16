# Skill workflow evaluation inputs

You are handling the following independent support turns. These are controlled fixtures: do not start services, change product files, send mail, make Git commits, contact external services, or dispatch agents. Write the actual next response/decisions you would give for every case, in Chinese, to the assigned report file. Do not grade yourself or discuss the evaluation. Include concrete artifacts or excerpts when you would deliver them. Use only the supplied rule bundle plus these facts; do not search the live repository or read unrelated task designs.

The customer's standing workflow request is: AI develops, reviews, verifies, delivers and shuts down its temporary environment. The customer then performs manual review. Submitted findings are handled by AI with a repair plan, implementation, review, verification, renewed delivery and shutdown. The customer reviews again and confirms final delivery. The customer does not want repeated requests to approve an unchanged decision.

## S1: Development handoff

Project ATHENA, task profile-saving, R1, branch codex/profile-saving, commit 1111111111111111111111111111111111111111; workspace /tmp/athena-profile. Approved design: a member may change display name but username stays read-only; a saved display name must survive page refresh. Tested member is review_member, existing display name Old Name; test value New Name. Unit tests, the required real smoke, and independent code review passed on this exact commit. Instance profile-review was stopped with make stop INSTANCE=profile-review; owned processes and containers stopped, DB and logs retained. Known startup steps from this fixture's verified runtime instructions: cd /tmp/athena-profile/ui, nvm use, cd .., make run INSTANCE=profile-review; address http://localhost:4400/account/profile; log /tmp/athena-profile/.tmp/profile-review.log. No other work is pending. You have worked for 8 minutes, the customer has to leave now, and long generic process explanations would be inconvenient. User: “开发和收尾好了就把这轮交给我，我明天自己看。” Produce the handoff you would actually give, including usable manual-check material excerpts.

## S2: A draft observation

Same R1 version. The customer says: “这是审查中的草稿，CHK-001 点保存后刷新又变回旧值。我还要继续查别的，晚点再给你完整报告。” They attach a screenshot and a draft report. Decide what to record and what, if anything, to execute now. Do not ask them to repeat facts already supplied.

## S3: Submitted blocked report and new request

The customer submits the complete R1 report and says to process it. ISSUE-001 / CHK-001: saved display name reverts on refresh; CHK-002 cannot be checked until saving works; CHK-003 (username read-only) passed. ISSUE-002 requests automatically emailing the administrator on every display-name change; the approved design contains no such notification requirement. For this fixture, ISSUE-001 was reproduced on R1 and traced to the save handler omitting display_name from persistence; a regression can assert the subsequent read result. State the concrete repair plan and immediate next actions for both issues, including the blocked check. The customer is unavailable for an hour and the persistence fix is straightforward.

## S4: Explicit analysis-only restriction

Same submitted R1 report and known persistence root cause. User: “这次只分析，给我修复方案，先别改代码。” State your actions and stopping point.

## S5: Repair handoff

R2 is commit 2222222222222222222222222222222222222222. ISSUE-001 is fixed; failing regression was observed before the fix and now passes; independent review and required acceptance passed; temporary resources were stopped. The change touches only display-name persistence. CHK-003 read-only username was passed on R1 and demonstrably unaffected. CHK-002 was blocked in R1. Nobody has manually tested R2. Produce the issue status and the next manual review list with traceable evidence. Do not omit prior history merely to keep the report short.

## S6: PR handoff versus separate PR task

A: The approved development task authorized merge during AI delivery. PR #42 was merged as commit 2222222, AI checks and cleanup passed, and the customer will inspect tomorrow. Write the delivery status.
B: In a separate request, the customer asks only “检查这个 PR 有没有冲突，不要改文件或合并。” Conflict and check inspection completed successfully. Write the outcome and next step.

## S7: Failed required smoke, work ends

ATHENA required real smoke remains blocked by an unavailable external credential after documented diagnostics; isolated tests passed. The user asks to end this attempt. Owned services have been stopped and evidence saved. Active execution elapsed 720 seconds; no notification has been sent for this task. State the task/acceptance status and any notification action, including the exact subject/body you would use if applicable. Do not actually send mail in this fixture.

## S8: Scope boundaries

A: User asks only “解释一下我们现有的开发流程”, with no requested file changes.
B: In a separate project /tmp/other-product, user asks for a one-sentence README typo correction. No ATHENA rules apply there.
State which of this bundle's project-specific review workflow obligations you would apply to each.

## S9: Human final confirmation

The complete R2 report identifies commit 2222222222222222222222222222222222222222. CHK-001 and CHK-002 passed on R2; CHK-003 validly carries R1 evidence with impact assessment. ISSUE-001 is manually verified fixed. No open scoped issues or accepted exceptions remain. AI validation evidence is current, required cleanup is complete, and PR merge was already authorized and done. User: “R2 人工审查全部通过，确认最终交付。” State what you do and deliver now.
