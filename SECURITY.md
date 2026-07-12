# Security policy

## Supported versions

This repository is experimental. Security fixes target the latest commit on `main` until stable releases exist.

## Reporting

Please use GitHub's private vulnerability reporting for this repository. Do not open a public issue containing credentials, private paths, malicious archives, or a working exploit.

## Trust boundary

- `tooltruth find`, `scan`, and `show` do not execute discovered tools.
- `resolve` accepts only exact command-name tokens. Default resolution checks PATH or a digest-bound managed record without executing candidates. `--identity` runs only compiled-in bounded version probes; missing and unclassified commands are never probed.
- `validate` is explicit opt-in and executes only compiled-in help probes for curated commands and built-in subcommands. Intended flags and values are not passed to the process; unknown subcommands and external CLI plugins are not probed.
- `context` executes only compiled-in version probes with a three-second per-command timeout, four-second overall timeout, and 4 KiB combined-output cap per command. Raw probe output is never returned or placed in the generated context; only parsed version and implementation tokens are emitted.
- `doctor`, `repair`, and `exec` are explicit opt-in operations limited to compiled-in recipes.
- Help probes have a five-second timeout and a 256 KiB combined-output cap. Their raw output is neither returned nor persisted by Tooltruth.
- A help probe executes the resolved local binary with the current user's permissions. It is an explicit observation, not a sandbox or a guarantee that third-party code has no side effects.
- A version probe also executes local code with the current user's permissions. Automatic host integration must preserve this explicit trust boundary.
- A local digest match is integrity evidence inside the current user account; it is not publisher authentication or a sandbox.
- Managed tools execute with the current user's permissions.
