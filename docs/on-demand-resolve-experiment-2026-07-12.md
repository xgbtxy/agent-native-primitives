# On-demand exact resolution experiment — 2026-07-12

## Product question

Can the AI choose when local truth is needed, propose a short candidate list, and use one deterministic entry without paying an automatic context tax?

## Interface

```text
tooltruth resolve COMMAND...
tooltruth resolve --identity COMMAND...
```

The default path checks only exact names and executes nothing. `--identity` is explicit and runs fixed bounded version probes only for recognized, present commands. Natural-language intent search is outside this interface.

## Mechanical results

On the current Windows machine with 91 PATH entries:

- one present command resolved in roughly 0.13 seconds p50 and 0.17 seconds p95;
- three present commands resolved in roughly 0.23 seconds p50, with an observed 0.61 second p95 caused by Windows PATH variability;
- four candidates produced 233 characters of compact JSON in presence-only mode;
- a three-command identity result remained about 100 tokens;
- concurrent PATH lookup was rejected after its three-command p95 became materially worse than sequential lookup.

These timings include process startup from PowerShell and are local observations, not cross-platform guarantees.

## Real-agent replay

Task: choose the best installed structured-data CLI and produce a read-only command for extracting GitHub Actions job IDs.

Agent: Codex CLI, `gpt-5.6-sol`, `xhigh`, ephemeral session.

| Treatment | Seconds | Commands | Failed commands | Input tokens | Outcome |
|---|---:|---:|---:|---:|---|
| Ordinary shell control | 210.3 | 21 | 11 | 497,282 | Correct |
| Old `find/show` interface available | not retained as a valid treatment | — | — | — | Model misused `show` and continued searching |
| New exact `resolve` interface available | 152.5 | 21 | 10 | 310,977 | Correct |

The new treatment's first attempted command was correctly formed:

```text
tooltruth resolve --identity yq jq dasel shyaml
```

The host's shell policy blocked both Tooltruth attempts before the binary ran. The model then fell back to broad environment searches. Therefore this run establishes that the model selected and formed the new interface naturally, but it does not prove end-to-end command reduction through a shell integration.

## Installed CLI smoke

After installing the current binary as `C:\Users\15412\go\bin\tooltruth.exe` (an existing `PATH` directory), a fresh ephemeral Codex CLI session was given one global rule:

```text
When an exact CLI command's local availability is uncertain, run `tooltruth resolve <candidate names...>` once and use only `present`; add `--identity` only when version or implementation changes the decision. Do not call Tooltruth for familiar commands.
```

The task prompt did not name Tooltruth. It asked the model to select an available YAML processor and extract the job IDs from `.github/workflows/release.yml`. With `gpt-5.6-sol`, maximum reasoning, and a read-only sandbox, the fresh session did exactly two shell calls:

```text
tooltruth resolve yq ruby python --identity
yq eval '.jobs | keys | .[]' .github/workflows/release.yml
```

Both calls succeeded. The answer was `verify`, `windows`, `macos`, and `publish`; wall time was 28.2 seconds. A separate 20-run local benchmark of `tooltruth resolve yq jq dasel shyaml` averaged 354.6 ms (18.8 ms standard deviation).

This is end-to-end evidence that a newly started agent can recall and invoke the installed CLI through the normal shell. It is still one smoke test, not causal evidence of broad productivity improvement, and it is not directly comparable to the earlier replay because model effort and session context differed.

## Decision

- Keep `resolve` as the primary product interface.
- Keep default resolution presence-only and zero-execution.
- Keep `--identity` explicit.
- Do not restore semantic keyword search to this path.
- Do not automatically inject a full environment list by default.
- Keep the product as a normal CLI. Install it on `PATH` and test it under each host's real command policy.
- Treat MCP or other native adapters as optional compatibility layers only for hosts that cannot invoke an allowed child CLI; they are not product dependencies.
- A child CLI cannot bypass its parent's shell policy, so the blocked replay is not end-to-end evidence.

Do not release as proven product value until 40–60 tasks across multiple hosts show at least 20% fewer environment-discovery calls, under 10% unnecessary calls, no version-to-syntax overclaim, and no material completion loss.
