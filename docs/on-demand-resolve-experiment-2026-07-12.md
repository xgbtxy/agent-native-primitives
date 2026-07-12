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

## Decision

- Keep `resolve` as the primary product interface.
- Keep default resolution presence-only and zero-execution.
- Keep `--identity` explicit.
- Do not restore semantic keyword search to this path.
- Do not automatically inject a full environment list by default.
- Add a native host/MCP adapter before claiming complete AI integration; a child CLI cannot bypass its parent's shell policy.

Do not release as proven product value until 40–60 tasks across multiple hosts show at least 20% fewer environment-discovery calls, under 10% unnecessary calls, no version-to-syntax overclaim, and no material completion loss.
