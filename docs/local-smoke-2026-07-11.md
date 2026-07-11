# Local-machine smoke test — 2026-07-11

This is a local engineering smoke test, not the preregistered external effectiveness benchmark.

## Resolver battery

Environment: current Windows machine and active PATH.

- 23 intent queries total.
- 16 positive cases: all returned the expected locally present command.
- 7 negative/absent/ambiguous cases: all returned `no_supported_match`.
- Every returned command was independently confirmed by PowerShell `Get-Command`.
- Median end-to-end CLI latency: 59.7 ms.
- Average serialized response in the initial strict-single-match schema: 340.3 bytes.
- No executable probe and no persisted index were used.

Positive coverage included ripgrep, fd, Git, jq, yq, curl, GitHub CLI, Go, Python, uv, uvx, Node, FFmpeg, FFprobe, exact-name Grok resolution, and project-described Grok XSearch capability.

Negative coverage included absent Docker, 7-Zip, and Make runtimes; unknown intent; Slack; destructive file deletion; and ambiguous structured-data intent.

## Mini AI A/B with Grok 4.5

Task: select a present local CLI that can extract router firmware into a filesystem tree. The PATH contained an opaque `fwx` fixture. The treatment project declared that `fwx` had the capability, but the fixture was intentionally a non-functional stub.

### Control arm

Tooltruth unavailable; ordinary shell discovery only.

| Run | Selected | Shell steps | Wall time | Outcome |
|---|---|---:|---:|---|
| 1 | null | 7 | 57.1 s | Correct abstention |
| 2 | null | 6 | 66.6 s | Correct abstention |

The agent inspected PATH, discovered that `fwx` was a stub, found a broken Binwalk installation, and correctly refused to guess.

### Initial treatment contract

Tooltruth returned project-declared semantics plus PATH presence without clearly distinguishing declaration from observed behavior.

| Run | Selected | Shell steps | Wall time | Outcome |
|---|---|---:|---:|---|
| 1 | fwx | 2 | 25.1 s | Faster but wrong |
| 2 | fwx | 2 | 25.7 s | Faster but wrong |

This invalidated the initial “trusted semantics” framing. A project descriptor is a claim, not behavior proof.

### Corrected epistemic contract

The public schema was changed to expose:

```json
{
  "claim": "...",
  "signal": {
    "semantics": "project_declared",
    "availability": "path_resolved",
    "behavior": "not_verified",
    "match": "intent:..."
  },
  "declared_example": "..."
}
```

The treatment was rerun with instructions not to treat declarations as behavior proof:

| Run | Selected | Shell steps | Wall time | Outcome |
|---|---|---:|---:|---|
| corrected | null | 2 | 21.2 s | Correct abstention |

## What is demonstrated

- The resolver can provide compact, current-environment, non-duplicated signals.
- Epistemically labeled signals can reduce investigation from 6–7 shell steps to 2 while preserving correct abstention in this fixture.
- Explicitly separating declarations, presence, and behavior materially changes agent behavior.

## Binwalk diagnosis and isolated repair

The machine's PATH contained the PyPI `binwalk` 2.1.0 launcher, but its package was incomplete: an explicit probe exited 1 because `binwalk.core` was missing.

The first curated repair recipe then:

- built official Binwalk 3.1.0 from crates.io with an isolated temporary Rust toolchain;
- left global PATH and global Python unchanged;
- removed the temporary build toolchain after success;
- installed one content-addressed executable with SHA-256 `fc97d0122482918f96df9284798c70689f1e91835fc6f02c8f693fb4e4260112`;
- returned one managed signal with `managed_digest_matched` and `help_signature_probe_passed`;
- scanned a local PNG and identified the PNG signature in 4 ms.

A clean-state negative test diagnosed the old PATH launcher as `broken`, returned exit code 1, and then returned `match: null` for firmware analysis instead of recommending the known-broken digest.

## What is not demonstrated

- No completed real-world task was accelerated.
- A narrow Binwalk help probe and one real PNG signature scan passed; this does not verify all Binwalk behavior or extraction dependencies.
- One agent model and one synthetic opaque-command scenario are insufficient for a product claim.
- The 360-run external A/B protocol remains required before claiming AI effectiveness.
