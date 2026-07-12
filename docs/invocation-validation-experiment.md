# Invocation validation experiment

## Hypothesis

Before executing a proposed CLI command, an AI agent can reduce invalid-argument retries by asking Tooltruth whether each requested flag is observed in the help surface of the exact locally resolved executable.

This is not a claim that help is a complete parser specification. Tooltruth must never turn absence from help into a claim of invalidity.

## Frozen intervention

Treatment agents may call:

```text
tooltruth validate --json -- COMMAND [ARGS...]
```

The implementation may execute only compiled-in fixed help recipes on a small exact-path pilot set. It passes curated subcommand tokens plus a fixed help argument, never user flags or values. A help probe still runs local code with the current user's permissions and is not a sandbox. The output is bound to the executable and help-output SHA-256 digests. There is no cache, daemon, model API, shell evaluation, or arbitrary command fallback.

Control agents receive the same task, environment, model, limits, and ordinary shell access without the Tooltruth instruction.

## Dataset

Use 40 to 60 real or replayed tasks across at least five supported CLIs. Include:

- commands proposed correctly on the first attempt;
- plausible but nonexistent flags;
- flags changed across installed versions;
- global flags before subcommands, combined short flags, and positional-only commands;
- unsupported commands where Tooltruth must abstain.

Task authors must freeze expected outcomes before running either arm. Do not derive the test solely from Tooltruth's recipe table or help parser.

## Primary measures

- Invalid-argument retries per task.
- Task completion rate on the invalid-argument subset.
- Useful-call rate: calls that prevent a retry or provide evidence used in the final command.
- False-valid rate: `requested_flags_observed_in_local_help` when the exact proposed invocation is rejected because of a claimed flag.
- Unnecessary-call rate on commands already correct.
- End-to-end latency overhead.

## Keep threshold

Keep and consider a v0.2 release only if the treatment produces at least one of:

- 20% fewer invalid-argument retries; or
- a 10 percentage-point completion improvement on the invalid-argument subset.

It must also achieve at least 60% useful-call payback, under 2% false-valid claims, under 10% unnecessary calls, and approximately 250 ms or less p95 local overhead.

## Kill threshold

Remove or redesign the feature if retry reduction is under 10%, false-valid claims exceed 5%, unnecessary calls exceed 10%, or useful coverage requires arbitrary executable probing or a broad hand-maintained CLI registry.

## Initial mechanical smoke result

On 2026-07-12, Windows local smoke tests observed real flags for `gh pr create --title`, `git status --short`, `go test -run`, `rg --glob`, `fd --extension`, `jq --raw-output`, `curl --fail`, and `uv pip install --system`; abstained for Python script flags, passthrough commands, and ambiguous `git -C` placement; and reported a fabricated GitHub CLI flag only as not observed. After pre/post executable digest binding, 25 repeated `rg` validations measured about 142 ms p50 and 158 ms p95 on this machine.

These repetitions establish only mechanics and overhead. They are not product-value evidence and do not satisfy the external A/B requirement.
