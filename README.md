# Agent Native Primitives

Small, local-first CLI primitives built for AI agents.

Each command in this repository must do one narrow job, emit sparse and evidence-labeled output, compose with ordinary shell tools, and remain useful without a specific model, IDE, agent, daemon, or cloud service.

| Tool | Purpose | Status |
|---|---|---|
| `tooltruth` | Resolve what local CLI capability is actually available for an intent, diagnose curated tools, and isolate repairs. | Experimental |

New tools are added only when a focused capability is not already served well by a mature project. This is not an agent framework and will not reimplement standard Unix utilities.

## tooltruth

Tooltruth is an experimental, local-first capability resolver for AI agents. It asks:

> In the active project and PATH scope, what capability can actually be resolved for this intent right now?

Normal discovery does not execute programs or persist a machine inventory. Explicit, curated commands can diagnose, repair, and run a managed tool without changing global PATH. Tooltruth does not run an agent, start a daemon, or expose environment-variable values.

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

## Current experimental CLI

```powershell
go run ./cmd/tooltruth scan --project .
go run ./cmd/tooltruth find "搜索 panic 日志并显示上下文" --project . --json
go run ./cmd/tooltruth show rg --project . --json
go run ./cmd/tooltruth doctor --project .
go run ./cmd/tooltruth doctor binwalk --json
go run ./cmd/tooltruth repair binwalk --json
go run ./cmd/tooltruth exec binwalk -- firmware.bin
```

Every command resolves the current project and PATH live. No index is written to disk.

## Agent output contract

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
      "match": "intent:查询 JSON"
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
When a task may benefit from an unfamiliar local or project-specific CLI,
run: tooltruth find "<intent>" --project . --json
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
      "intents": ["拆包路由器固件", "extract router firmware"],
      "examples": [
        {"intent": "拆包固件", "command": "fwx unpack image.bin"}
      ],
      "risk": "medium"
    }
  ]
}
```

The descriptor is returned only when `fwx` resolves in the active PATH. The file is limited to 256 KiB and no program is executed while reading it.

## Privacy and safety boundaries

- Discovery never invokes an executable. Only explicit `doctor` and `exec` do.
- No PATH inventory or project task body is persisted.
- Health history contains only tool ID, executable digest, probe ID, status, and timestamp; raw output is not persisted.
- Default `find --json` output omits full paths, environment metadata, risk labels, internal scores, and pseudo-confidence.
- One capability family can produce at most one signal, and the public contract returns only the strongest supported match.
- `show <tool>` explicitly reveals the resolved path for diagnostics.
- Package-script bodies are not copied into semantic metadata.
- Missing runtimes are visible to diagnostics but never recommended by `find`.

## Evidence status

The catalog ranking test is a mechanical regression test only. Because its queries and expected tools are authored with the same catalog, it is not evidence that Tooltruth improves an AI.

Product value must be established with the preregistered external A/B design in [docs/ab-protocol.md](docs/ab-protocol.md).

## Non-goals

- Automatic repair or probing during `find`
- General-purpose package management; repair is limited to compiled-in recipes
- MCP or Skills marketplace
- Agent planning or arbitrary command execution
- Chat, memory, embeddings, or RAG
- Background indexing service

## Build and test

```powershell
go test -race ./...
go vet ./...
go build -o dist/tooltruth.exe ./cmd/tooltruth
```
