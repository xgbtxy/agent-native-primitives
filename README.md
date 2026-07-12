# Agent Native Primitives

Small, local-first CLI primitives built for AI agents.

Each command in this repository must do one narrow job, emit sparse and evidence-labeled output, compose with ordinary shell tools, and remain useful without a specific model, IDE, agent, daemon, or cloud service.

The English repository is canonical. The capability contract is in docs/AI_CAPABILITY_STANDARD.md. AI agents should read the low-token docs/AI_CAPABILITY_INDEX.md first and open a detailed capability page only when needed. A Chinese documentation mirror is maintained separately; command names, signal names, JSON fields, and safety semantics remain identical.

| Tool | Purpose | Status |
|---|---|---|
| `tooltruth` | Filter AI-proposed command names against the live local PATH and return sparse evidence. | Experimental |

New tools are added only when a focused capability is not already served well by a mature project. This is not an agent framework and will not reimplement standard Unix utilities.

## tooltruth

Tooltruth is an experimental, local-first capability resolver for AI agents. It asks:

> Which exact command names proposed by the AI resolve in the active local scope right now?

Normal discovery does not execute programs or persist a machine inventory. Explicit, curated commands can diagnose, repair, and run a managed tool without changing global PATH. Tooltruth does not run an agent, start a daemon, or expose environment-variable values.

The primary interface is now `tooltruth resolve`: the model supplies a short candidate list, Tooltruth checks only those exact names, and the model remains responsible for semantic choice. There is no intent-keyword search in this path and no automatic environment injection. The first behavior experiment is documented in [docs/on-demand-resolve-experiment-2026-07-12.md](docs/on-demand-resolve-experiment-2026-07-12.md).

The competitive boundary is recorded in [docs/research.md](docs/research.md). Current local-machine results, including a failed initial A/B and the corrected epistemic contract, are in [docs/local-smoke-2026-07-11.md](docs/local-smoke-2026-07-11.md).

## Install a binary release

The [latest GitHub Release](https://github.com/xgbtxy/agent-native-primitives/releases/latest) contains exactly two platform packages:

- `tooltruth_<version>_windows_x86_64.zip`
- `tooltruth_<version>_macos_universal.tar.gz` for both Apple Silicon and Intel Macs

Extract the package and place `tooltruth` or `tooltruth.exe` in a directory already on `PATH`. Then verify:

```text
tooltruth version
```

Every release includes `SHA256SUMS.txt` and GitHub build-provenance attestations. Verify provenance with GitHub CLI:

```text
gh attestation verify <downloaded-package> --repo xgbtxy/agent-native-primitives
```

The macOS binary supports discovery commands. The curated Binwalk repair recipe remains Windows/x86_64-only in this experimental release.

The initial packages are not signed with commercial Windows or Apple developer certificates and are not Apple-notarized. Checksums and GitHub provenance establish release origin, but the operating system may still display an unknown-publisher warning.

## Minimal AI integration

Install the binary on `PATH`, then give an AI host one rule:

```text
When command availability is uncertain, run `tooltruth resolve <exact candidate names...>` once and use only entries in `present`. Add `--identity` only when implementation or version changes the decision. Do not call Tooltruth before familiar commands.
```

That rule is the complete integration. Tooltruth remains an ordinary child CLI: no MCP server, daemon, shell hook, startup scan, or automatically injected environment inventory is required. A host that forbids unknown child commands must allow the installed binary through its own command policy; Tooltruth does not bypass its parent.

The public help intentionally presents only `resolve` and `version`. Earlier experiments remain available under `tooltruth help --all`, but an AI does not need to learn them.

## Current experimental CLI

```powershell
go run ./cmd/tooltruth resolve yq jq python
go run ./cmd/tooltruth resolve --identity yq
go run ./cmd/tooltruth scan --project .
go run ./cmd/tooltruth find "search panic logs with context" --project . --json
go run ./cmd/tooltruth show rg --project . --json
go run ./cmd/tooltruth doctor --project .
go run ./cmd/tooltruth doctor binwalk --json
go run ./cmd/tooltruth context --project .
go run ./cmd/tooltruth validate --json -- gh pr create --title hello
go run ./cmd/tooltruth repair binwalk --json
go run ./cmd/tooltruth exec binwalk -- firmware.bin
```

Every command resolves the current project and PATH live. No index is written to disk.

## On-demand exact resolution

The AI proposes concrete candidates and calls one entry only when local availability is uncertain:

```text
tooltruth resolve yq jq python definitely-not-installed
```

Default output is compact JSON and does not execute any candidate:

```json
{"scope":"...","present":[{"command":"yq","signal":"path_resolved"},{"command":"jq","signal":"path_resolved"},{"command":"python","signal":"path_resolved"}],"absent":["definitely-not-installed"],"limits":"presence_only"}
```

When implementation identity matters, the AI may opt into fixed bounded version probes:

```text
tooltruth resolve --identity yq jq python
```

```json
{"scope":"...","present":[{"command":"yq","version":"4.52.4","implementation":"mikefarah","signal":"path+version_observed"}],"absent":[],"limits":"presence_version_only"}
```

Rules:

- Inputs are exact command names, not task keywords or natural-language intents.
- At most 32 names are accepted; duplicates are removed and output order is stable.
- Default resolution checks only PATH or a digest-bound managed record and executes nothing.
- `--identity` executes only compiled-in version probes and never returns raw probe output.
- Missing and invalid names are structured results with a successful process exit; operational failures exit nonzero.
- The AI should call this only for unfamiliar candidates, version-sensitive choices, or before installing a missing tool—not before every shell command.

## Optional compact environment facts

```text
tooltruth context --project .
```

Example output from the current machine is about 130 tokens:

```text
Verified local command facts (scope ...; trust presence/version without re-checking):
- PATH-resolved: gh@2.91.0, go@1.23.4, jq@1.8.1, rg@15.1.0, yq[mikefarah]@4.52.4, ...
- Digest-bound managed: tooltruth exec binwalk --@3.1.0
- Limits: presence/version only; flags, aliases, shell functions, and runtime behavior remain unknown.
```

`context` remains an opt-in host experiment, not the default integration. It executes only compiled-in version probes, extracts bounded version/implementation tokens, and never forwards raw probe output into model context. On the measured machine it takes roughly 0.5 seconds and persists nothing.

The contract is intentionally narrow:

- A listed command and version may be trusted in the reported scope without another PATH/version check.
- A version never proves that a flag, subcommand, alias, runtime behavior, or network operation is valid.
- Unconditional injection imposes a fixed cost on tasks that do not need environment discovery; prefer `resolve` unless a host-specific A/B proves injection useful.

## Experimental invocation validation

`validate` checks a proposed invocation before an agent runs it:

```text
tooltruth validate --json -- gh pr create --title hello
tooltruth validate --json -- git status --short
tooltruth validate --json -- rg --glob "*.go" needle
```

The `--` separator is mandatory. Tooltruth discards all user-provided values when probing and executes only a compiled-in help recipe. The pilot supports leaf surfaces of `rg`, `fd`/`fdfind`, `jq`, `curl`, and `binwalk`, plus a small exact-path set for `gh`, `git`, `go`, and `uv`. Unknown commands, non-whitelisted command paths, external CLI plugins, ambiguous subcommand paths, incompatible `yq` variants, Docker, and interpreter-owned script arguments cause an abstention without execution.

The result language is deliberately asymmetric:

- `observed_in_local_help` means the exact flag token appeared in help produced by the currently resolved executable.
- `not_observed_in_local_help` is not a claim that the parser rejects the flag; help completeness is explicitly `unknown`.
- Positional values, shell aliases, runtime behavior, and command success are never claimed.
- Evidence includes the executable SHA-256, sanitized probe argv, help-output SHA-256, byte count, exit code, and truncation state. Raw help is not persisted.
- Passthrough subcommands such as `go run`, `docker exec`, and `uv run` abstain because another parser may own later flags.
- Structured evidence statuses exit zero; malformed Tooltruth usage and operational failures exit nonzero. Callers must branch on the JSON `status`, not process success alone.

This feature is on `main` for measurement and is not included in the v0.1.0 binary release. Its preregistered keep/kill test is in [docs/invocation-validation-experiment.md](docs/invocation-validation-experiment.md).

## Legacy semantic discovery experiment

`find` predates the exact-name resolver and is not the primary product path. It remains temporarily available for comparison while real-agent tests determine whether it should be removed.

`find --json` returns exactly one `match` or `null`. It never returns a ranked list:

```json
{
  "scope": {"id": "cd9cefe3c3795ea0", "project": "demo"},
  "match": {
    "id": "jq",
    "command": "jq",
    "claim": "Query, filter, transform, and format JSON from files or stdin.",
    "signal": {
      "semantics": "curated_name_mapping",
      "availability": "path_resolved",
      "behavior": "not_verified",
      "match": "intent:query JSON"
    },
    "declared_example": "jq -r '.server.port' config.json"
  }
}
```

A candidate is emitted only when both traceable signals exist:

1. Semantic claim: a curated name mapping or project-owned declaration matched an explicit intent.
2. Availability observation: the exact command/runtime resolves in the active scope.

For ordinary PATH and project commands, Tooltruth does not execute the command and states `behavior: not_verified`. A project declaration is never presented as observed behavior. A curated explicit probe is bound to the executable digest; its narrow evidence is reported as `help_signature_probe_passed`, not as general behavior verification. Descriptions and examples are labeled `claim` and `declared_example`; they never create a match by themselves. Weak similarity abstains with `status: no_supported_match`. Internal ranking scores are never exposed.

## Curated repair experiment

The first and only repair recipe is Binwalk 3.1.0 on Windows/amd64:

```powershell
tooltruth doctor binwalk --json
tooltruth repair binwalk --json
tooltruth exec binwalk -- firmware.bin
```

- `doctor` is explicit, runs only a fixed help-signature probe, records no raw output, and exits nonzero for broken or missing tools.
- A known-broken executable digest is suppressed from later `find` results.
- `repair` pins the Rust bootstrap digest, exact Rust toolchain, Binwalk version, and crates.io package checksum.
- The build uses temporary Tooltruth-owned directories and a reduced environment; global Python, Cargo, Rust, and PATH are unchanged.
- The retained binary lives at an immutable content-addressed path. The current manifest and health record are digest-bound.
- `exec` accepts only curated managed IDs, rechecks the digest, and passes argv directly without a shell.

This is an explicit network build, not a sandbox. Build scripts run with the current user’s permissions. No additional repair recipes should be added until this lifecycle is validated further.

## Agent instruction under test

This instruction is an experiment, not a proven default:

```text
When local availability of one or more concrete CLI candidates is uncertain,
run once: tooltruth resolve <candidate>...
Add --identity only when version or implementation affects the decision.
```

Do not instruct an agent to call Tooltruth before every shell action. The A/B protocol must measure useful calls and unnecessary-call overhead.

## Static discovery sources

- A small built-in catalog of common CLI names, resolved against the active PATH
- Project-owned `.tooltruth.json` descriptors for opaque/internal commands
- Top-level `package.json` script names with package-manager detection
- Top-level Makefile target names
- Exact command-name fallback when an unknown command is explicitly named

Tooltruth never invokes a discovered command during resolution. `path_resolved` means only that its command name resolves in the current PATH; it is not an identity or behavior verification claim. `managed_digest_matched` means the cached bytes match Tooltruth's local manifest, not publisher authentication.

## Describing an opaque project CLI

Create `.tooltruth.json` in the project root:

```json
{
  "capabilities": [
    {
      "id": "firmware-unpack",
      "family": "firmware_extraction",
      "command": "fwx",
      "description": "Extract a supported router firmware image into a filesystem tree.",
      "capabilities": ["firmware_extraction"],
      "intents": ["extract router firmware", "analyze firmware"],
      "examples": [
        {"intent": "extract firmware", "command": "fwx unpack image.bin"}
      ],
      "risk": "medium"
    }
  ]
}
```

The descriptor is returned only when `fwx` resolves in the active PATH. The file is limited to 256 KiB and no program is executed while reading it.

## Privacy and safety boundaries

- Discovery never invokes an executable. Explicit `context` runs only fixed bounded version probes; `doctor`, `validate`, and `exec` have their separately documented execution boundaries; `repair` additionally performs an explicit network build.
- No PATH inventory or project task body is persisted.
- Health history contains only tool ID, executable digest, probe ID, status, and timestamp; raw output is not persisted.
- Default `find --json` output omits full paths, environment metadata, risk labels, internal scores, and pseudo-confidence.
- One capability family can produce at most one signal, and the public contract returns only the strongest supported match.
- `resolve` omits full paths; `show <tool>` explicitly reveals one resolved path for diagnostics.
- Package-script bodies are not copied into semantic metadata.
- Missing runtimes are visible to diagnostics but never recommended by `find`.

## Evidence status

The catalog ranking test is a mechanical regression test only. Because its queries and expected tools are authored with the same catalog, it is not evidence that Tooltruth improves an AI.

Product value must be established with the preregistered external A/B design in [docs/ab-protocol.md](docs/ab-protocol.md).

## Non-goals

- Natural-language or keyword matching in the primary `resolve` path
- Calling Tooltruth before every shell command
- Automatic repair or probing during `find`
- General-purpose package management; repair is limited to compiled-in recipes
- MCP or Skills marketplace
- Agent planning or arbitrary command execution
- Arbitrary `<command> --help` probing or a static registry of every CLI
- Chat, memory, embeddings, or RAG
- Background indexing service

## Build and test

```powershell
go test -race ./...
go vet ./...
go build -o dist/tooltruth.exe ./cmd/tooltruth
```
