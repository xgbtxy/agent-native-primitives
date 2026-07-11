# Competitive research and adopted boundary

Research date: 2026-07-11. X/Twitter discovery used the local Grok CLI and XSearch. The closest projects were then checked against their websites, repositories, and selected source files. The implementation was also independently pressure-tested before publication.

## Closest products

- [CmdHub](https://github.com/Xuepoo/cmdhub-oss) is the strongest neighbor: large offline command registry, hybrid search, ACI contracts, risk levels, MCP integration, and installed/not-installed checks.
- [Need](https://www.npmjs.com/package/@agentneeds/need) semantically searches a large global CLI catalog and can install tools through package managers.
- [getcli](https://github.com/theagentservice/getcli) searches, installs, and verifies a curated registry of agent-friendly CLIs.
- [CLI Finder](https://clifinder.net/) is a curated public directory with installability, structured-output, authentication, and safety notes.
- [skill-search-cli](https://github.com/Michaelliv/skill-search-cli) scans locally installed Agent Skills and exposes fuzzy/JSON search.
- [mcp-cli](https://github.com/philschmid/mcp-cli) demonstrates progressive disclosure through `info → grep → call` for configured MCP tools.
- [Anthropic Tool Search](https://www.anthropic.com/engineering/advanced-tool-use) validates the general problem of tool-schema context cost and selection errors, but not PATH capability discovery specifically.
- [ToolRecall](https://github.com/whiskybeer/toolrecall) is an active deterministic tool-output cache and MCP multiplexer. It solves a different problem, but its prior use of the name caused this CLI to be renamed `tooltruth` before publication.

## Boundary after adversarial review

The initial boundary—persisting a whole PATH inventory and probing known executables—was rejected. It introduced unsafe execution, stale cross-project state, sensitive aggregation, and no semantic discovery for opaque commands.

The adopted experimental boundary is:

> A query-time, project-scoped capability resolver: intent → trustworthy static semantics → active PATH resolution → one compact present candidate or none.

Implementation consequences:

- No global or project index persistence.
- No executable probing or `verified` identity claim.
- No full PATH enumeration; only known semantic candidates are resolved.
- Unknown exact names can be checked but are explicitly unclassified.
- Opaque internal tools obtain semantics only from project-owned `.tooltruth.json` descriptors.
- Default agent output omits paths, secret-adjacent environment metadata, risk labels, internal scores, and confidence percentages.
- Agent output is one supported match or explicit abstention; interchangeable alternatives are not emitted as separate facts.
- Every match separates a semantic claim from a current-scope availability observation and explicitly states that behavior was not verified.
- Package scripts and Make targets are scoped to the current project and are excluded when their runtime is missing.

Explicit `doctor`, `repair`, and `exec` commands form a separate opt-in boundary. They are limited to compiled-in recipes, persist only digest-bound health/install metadata, and never run automatically from `find`.

## Still unproven

No current test establishes improved agent outcomes. The remaining product hypothesis is narrow:

> In projects with opaque or niche local CLIs, one compact resolver call saves more agent rounds than it costs, without increasing context or unsafe actions.

This must beat a shell-native control arm. Competitor feature gaps are not demand evidence.
