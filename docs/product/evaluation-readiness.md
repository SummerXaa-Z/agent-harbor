# AgentHarbor Evaluation Readiness

Status: developer-preview evaluator guide. Updated for the 0.3.x Phase 0 evaluator loop: after the timed permission-change pass, evaluators now also complete an untimed consumer leg (Access Handoff delivery verified from the credential holder's side) and record upgrade friction.

AgentHarbor is ready for local evaluation when a new evaluator can start from a fresh checkout, run the demo stack, complete the Permission Changes journey, export a production acceptance report, and explain the report without author guidance.

AgentHarbor 的本地评估标准是：新的评估者可以从全新检出开始，启动 demo stack，完成权限变更旅程，导出上线验收报告，并能不依赖作者讲解说明报告内容。0.3.x Phase 0 这一轮在计时主旅程之后追加两段不计时内容：消费侧验证（站在凭据持有者一方验证接入交付）与升级摩擦记录。

## Audience

Use this guide with three evaluator roles:

- Platform engineer: checks setup, compatibility, local services, and operational fit. In the consumer leg, checks that a standard MCP client can connect with the issued token and that authentication failures carry a diagnosable reason. Optionally records the upgrade-friction count.
- Security reviewer: checks approval records, allow/deny runtime records, audit records, and report digest handling. In the consumer leg, checks that token issuance and handoff view/copy events are audited, the plaintext is shown exactly once, and no secret appears in exports or notes.
- Tenant administrator: checks whether the Permission Changes workflow is understandable without reading code or API contracts. In the consumer leg, checks whether the handoff card alone is enough for a team member to start calling: subject header, MCP config snippet, request example, and prompt template.

## Evaluation Goal

The evaluation answers one question: can someone outside the author loop understand and complete the first governed permission change in 30 minutes?

The primary metric is `time-to-first-report`: the time from opening the repository to exporting a production acceptance report from the web console.

Recommended success bar:

- `time-to-first-report` is under 30 minutes.
- The evaluator exports a production acceptance report.
- The evaluator can identify who received access, which permission package was applied, which capability was allowed, which capability was denied, and which report digest should be reviewed.
- Any blocker is recorded as a product or documentation issue, not as private verbal context.

## 30-minute evaluator walkthrough

1. Start from a fresh checkout of the reviewed branch.
2. Capture environment basics:

   ```bash
   git rev-parse --short HEAD
   go version
   node --version
   scripts/pnpm.sh --version
   ```

3. Generate the evaluator pack:

   ```bash
   make evaluation-readiness
   ```

4. Review the generated `environment-snapshot.md` and confirm it captured branch, commit, working-tree state, Go, Node, and pnpm.
5. Start the local product:

   ```bash
   make demo
   ```

6. Open `http://127.0.0.1:5174/`.
7. Follow the visible setup path until the product reaches **Permission Changes** or lets the evaluator start a permission fix from **Access Query**.
8. Use the **Support ticket triage / 客服工单处理** permission package.
9. Complete approval, apply, runtime validation, and go-live status review.
10. Export the production acceptance report.
11. Record the session in `feedback-log.csv` from the generated evaluator pack.

If the evaluator cannot finish within 30 minutes, stop and record the first blocker. Do not explain around the blocker during the timed pass.

## Consumer leg walkthrough (untimed)

After the timed pass, the same session continues without a clock. The goal is to verify that delivered access is usable and observable from the credential holder's side, using only what the handoff card provides:

1. From the ready application, open the **Access Handoff** view and issue a short-lived token. Confirm the plaintext is displayed once and never again.
2. Copy the MCP client config and the runtime request example from the card. Do not ask the author for any additional value.
3. From outside the web console, complete one governed call with the issued token. A standard MCP client (Claude Desktop, Cursor, Cline, MCP Inspector) or the copied request example both count; record which kind was used.
4. Read `GET /api/v1/self/access-profile` with the token and state the boundary it reports: subject, target, allowed capability, expiry.
5. Optionally, exercise one failure path: an expired or revoked token must fail with the specific reason (`bearer token has expired` / `bearer token has been revoked`), not a generic error.
6. Record blockers in `feedback-log.csv` (`first_blocker` for the timed pass, `notes` for consumer-leg blockers) and the details in `acceptance-report-notes.md`.

## Upgrade-friction record (optional, platform engineer)

If the session has a v1-classified capability baseline, switch it to v2 and reach a usable token, counting every manual step (the expected shape today is reclassify, re-preview, re-approve, re-apply, re-issue). Record the step count and wall minutes in `acceptance-report-notes.md`; if no v1 baseline exists in the session, record not-applicable with the reason. This round only records the number — remediation is deliberately out of scope and feeds the v0.4 authorization-lifecycle scoping.

## Evidence to Collect

Keep these records in the generated evaluator pack:

- Branch and commit.
- Toolchain snapshot from `environment-snapshot.md`.
- Evaluator role.
- `time-to-first-report`.
- First blocker, if any.
- Confusing term, if any.
- Exported report digest and digest algorithm.
- Generated-by actor from the report.
- Whether the evaluator could explain the allowed and denied capability.
- Whether the consumer leg completed: governed call from copied artifacts, `GET /api/v1/self/access-profile` read, boundary stated.
- Client kind used for the consumer leg (copied example / standard MCP client).
- Diagnosable failure observation (revoked / expired / unknown), if exercised.
- Handoff and token audit references.
- Upgrade-friction manual step count and wall minutes, or not-applicable with reason.

Do not record admin keys, agent keys, bearer tokens, passwords, upstream credentials, full Authorization headers, or personal data.

## Interpretation

Use the result to decide the next product increment:

| Result | Product response |
| --- | --- |
| Three evaluators finish under 30 minutes with no repeated blocker, and the consumer leg completes | Open the My Access self-service slice per the 0.3.x PRD (Phase A). |
| Evaluators finish but cannot explain approval, runtime, or report digest records | Improve copy, report summary, and walkthrough before adding more capability breadth. |
| Evaluators fail before the web console opens | Prioritize setup, toolchain, and demo reliability. |
| Evaluators fail inside Permission Changes | Prioritize workflow guidance, empty states, and action labels. |
| Security reviewers distrust the report | Prioritize report contract, digest verification, and audit references. |
| Evaluators cannot complete the consumer leg (token to governed call or self access-profile) | Fix the consumer-side blocker before opening My Access; the request_access loop builds on this leg. |
| Evaluators finish but report heavy upgrade friction | Carry the recorded counts into the v0.4 authorization-lifecycle scoping before adding report surface. |

## Maintainer Check

Before inviting external evaluators, maintainers should run:

```bash
make check
make evaluation-readiness
```

For release-candidate handoff, also run:

```bash
make release-check
```

The evaluator pack is intentionally lightweight. It does not replace scenario gates; it captures whether the current product can be understood by a new evaluator.
