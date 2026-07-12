# Local AI context experiment — 2026-07-12

> Follow-up: this experiment proves that verified local facts can reduce environment search, but unconditional injection is no longer the default product route. See [on-demand-resolve-experiment-2026-07-12.md](on-demand-resolve-experiment-2026-07-12.md).

## Question

Does an AI become more efficient when it can call a local capability CLI, or when a small set of verified local facts is injected before its first turn?

## Setup

- Host: Windows, current Agent Native Primitives repository.
- Agent: Codex CLI with `gpt-5.6-sol`, reasoning effort `xhigh`, ephemeral sessions.
- Tasks were read-only. The agent was told not to invoke another AI.
- Measurements came from JSONL command events and turn usage.
- The local command policy rejected several generated PowerShell forms. That behavior materially inflated command failures and means these results must not be generalized to other hosts without repetition.

Three treatments were compared:

1. Control: ordinary local shell access.
2. Callable Tooltruth: the prompt described `find` and `show`, but use was optional.
3. Fact header: verified presence/version facts were inserted before the task. The final run used the exact output of `tooltruth context`, not hand-authored facts.

## Results

| Task and treatment | Seconds | Commands | Failed commands | Input tokens | Outcome |
|---|---:|---:|---:|---:|---|
| Find Go TODO/FIXME — control | 64.7 | 6 | 4 | 113,857 | Correct |
| Find Go TODO/FIXME — callable Tooltruth | 63.4 | 5 | 4 | 156,507 | Correct, more tokens |
| Find Go TODO/FIXME — generated fact header | 10.1 | 0 | 0 | 15,142 | Correct |
| Query workflow job IDs — control | 210.3 | 21 | 11 | 497,282 | Correct |
| Query workflow job IDs — generated fact header | 74.8 | 10 | 6 | 160,654 | Correct |
| Verify GitHub CLI flags — control | 28.0 | 4 | 4 | 78,158 | Correctly inconclusive |
| Verify GitHub CLI flags — hand fact header | 44.9 | 3 | 3 | 62,146 | Overconfident; unacceptable |

The generated fact header was 453 characters, approximately 130 tokens, and took about 0.47 seconds to produce on this machine.

## Findings

### 1. A callable helper is the wrong default integration

Merely telling the model that Tooltruth existed did not make it the first choice. The model tried familiar shell discovery first, later called Tooltruth, misused one command shape, and consumed more tokens than the control. A new CLI creates its own recall and syntax burden.

### 2. Pre-turn fact injection produced real savings

For tool selection and implementation identity, a small live header removed repeated PATH searches, version calls, installation-directory scans, and even binary-string inspection. The two applicable tasks saw large local reductions in elapsed time, commands, failed commands, and input tokens.

### 3. Presence/version evidence must not become syntax evidence

In the GitHub CLI task, supplying `gh@2.91.0` without successful local help inspection led the model to claim flag support despite unavailable help evidence. This is a hard product boundary: versions can suppress redundant presence/version checks, but cannot validate flags or behavior.

### 4. Shell-host policy is a separate problem

Many failed calls came from the benchmark host rejecting generated PowerShell forms. A child CLI cannot reliably fix its parent's execution policy. Host adapters may inject shell-specific execution guidance, but Tooltruth's portable fact contract must not claim to solve shell policy.

## Product decision

Retain the following only as an optional host experiment:

```text
tooltruth context [--project DIR] [--json]
```

An AI host or launcher may inject it only when a host-specific A/B justifies the fixed cost. The primary route is now model-decided exact candidate resolution. The output may establish only:

- command presence in the active PATH scope;
- digest-bound Tooltruth-managed presence;
- version identity from fixed bounded probes;
- explicitly recognized implementation variants such as Mike Farah `yq`.

The output must not establish flags, subcommands, aliases, behavior, or command success.

Do not release this as proven product value yet. Repeat on 40–60 tasks across multiple agent hosts. Kill the direction if automatic injection fails to reduce environment-discovery calls by at least 20%, adds more than 500 tokens by default, or causes any syntax/behavior claim to be inferred from version evidence alone.
