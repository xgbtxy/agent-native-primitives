# External AI-efficiency A/B protocol

This experiment determines whether Tooltruth improves real terminal agents. Unit tests and catalog ranking smoke tests do not answer that question.

## Frozen task set

Use 30 held-out real tasks, five from each stratum:

1. Common installed tools already familiar to models.
2. Niche installed tools with non-obvious names and project descriptors.
3. Project-local tasks, including at least one monorepo.
4. Environment-sensitive tasks involving shadowing, venv, PowerShell, WSL, or version differences.
5. Negative or ambiguous tasks where no local recommendation is correct.
6. End-to-end coding tasks where capability selection affects checker success.

Task authors must not author the catalog or descriptors. Prompts describe outcomes and never name the expected command.

## Arms and runs

- Agent A: pinned Codex CLI build.
- Agent B: pinned independent terminal agent.
- Control: ordinary shell and project access; no Tooltruth binary, mention, or instruction.
- Treatment: identical environment plus Tooltruth and the exact experimental instruction from README.
- Three fresh repetitions per task, agent, and arm.
- Total runs: `30 × 2 × 2 × 3 = 360`.

## Primary metrics

- Task-checker pass/fail.
- Completion within fixed turn and time budgets.
- Model-to-tool round trips.

Secondary metrics:

- Wall time and input/output tokens.
- Shell commands and Tooltruth calls.
- Useful versus unnecessary Tooltruth calls.
- Runnable candidate rate in the active environment.
- Wrong-install attempts, unsafe actions, and recovery rounds.
- Serialized Tooltruth result bytes.

Per-call payback:

```text
native discovery calls avoided
- Tooltruth-related calls
- recovery calls caused by its recommendation
```

## Success threshold

Require all of:

1. Discovery-needed tasks gain at least 10 percentage points completion, or remain within 2 points while reducing round trips by at least 20%.
2. All tasks reduce round trips by at least 10%.
3. Total context tokens increase by no more than 5%.
4. Unnecessary Tooltruth calls occur on at most 10% of no-benefit tasks.
5. At least 60% of Tooltruth invocations have positive payback.
6. No increase in unsafe execution, wrong-environment selection, or unnecessary installation.
7. Task-clustered confidence intervals exclude zero for the claimed primary gain.

## Kill criteria

Stop the product direction if any remains true after one bounded correction cycle:

- Precision@1 is below 80% or negative-task false positives exceed 5%.
- More than 1% of recommendations are unrunnable or belong to the wrong environment.
- Median total rounds do not improve.
- Context grows more than 5% without a completion gain.
- Fewer than 60% of invocations save more calls than they cost.
- Unknown project capabilities remain undiscoverable even when trustworthy descriptors exist.
- Any default operation executes discovered programs or emits secret-like project data.
- Gains require expansion into a global directory, installer, MCP marketplace, or RAG platform.
