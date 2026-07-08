# AgentHarbor Evaluation Readiness

Status: developer-preview evaluator guide.

AgentHarbor is ready for local evaluation when a new evaluator can start from a fresh checkout, run the demo stack, complete the Permission Changes journey, export a production acceptance report, and explain the report without author guidance.

AgentHarbor 的本地评估标准是：新的评估者可以从全新检出开始，启动 demo stack，完成权限变更旅程，导出上线验收报告，并能不依赖作者讲解说明报告内容。

## Audience

Use this guide with three evaluator roles:

- Platform engineer: checks setup, compatibility, local services, and operational fit.
- Security reviewer: checks approval records, allow/deny runtime records, audit records, and report digest handling.
- Tenant administrator: checks whether the Permission Changes workflow is understandable without reading code or API contracts.

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

Do not record admin keys, agent keys, bearer tokens, passwords, upstream credentials, full Authorization headers, or personal data.

## Interpretation

Use the result to decide the next product increment:

| Result | Product response |
| --- | --- |
| Three evaluators finish under 30 minutes with no repeated blocker | Move to the next adoption slice, such as Access Handoff. |
| Evaluators finish but cannot explain approval, runtime, or report digest records | Improve copy, report summary, and walkthrough before adding more capability breadth. |
| Evaluators fail before the web console opens | Prioritize setup, toolchain, and demo reliability. |
| Evaluators fail inside Permission Changes | Prioritize workflow guidance, empty states, and action labels. |
| Security reviewers distrust the report | Prioritize report contract, digest verification, and audit references. |

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
