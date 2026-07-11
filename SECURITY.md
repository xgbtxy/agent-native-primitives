# Security policy

## Supported versions

This repository is experimental. Security fixes target the latest commit on `main` until stable releases exist.

## Reporting

Please use GitHub's private vulnerability reporting for this repository. Do not open a public issue containing credentials, private paths, malicious archives, or a working exploit.

## Trust boundary

- `tooltruth find`, `scan`, and `show` do not execute discovered tools.
- `doctor`, `repair`, and `exec` are explicit opt-in operations limited to compiled-in recipes.
- A local digest match is integrity evidence inside the current user account; it is not publisher authentication or a sandbox.
- Managed tools execute with the current user's permissions.
