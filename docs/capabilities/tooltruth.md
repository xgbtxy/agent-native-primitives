# Capability: tooltruth

status: experimental
tool_id: tooltruth
purpose: resolve exact local command availability and emit bounded evidence for AI agents
source: this repository
primary_signal: path_resolved
signal_ids:
- tooltruth.resolve.availability
- tooltruth.validate.help_surface
- tooltruth.context.bounded_facts
- tooltruth.version.identity
- tooltruth.find.legacy_semantic
- tooltruth.show.project_descriptor
- tooltruth.doctor.managed_health
- tooltruth.repair.explicit_action
- tooltruth.exec.managed_action

## Call Contract

Call Tooltruth only when a concrete command candidate is unfamiliar, version-sensitive, or about to be installed. Do not call it before familiar commands and do not use it as a general environment inventory.

## Common Commands

    tooltruth resolve yq jq python
    tooltruth resolve --identity yq jq python
    tooltruth version

## Signals

- path_resolved: the exact command name resolves in the active PATH or an approved managed record.
- path+version_observed: a fixed compiled-in identity probe returned a bounded version and implementation.
- managed_digest_matched: a managed artifact matches its local digest-bound record.
- absent: the exact candidate did not resolve.
- observed_in_local_help: an exact flag token appeared in the current executable's bounded local help output.
- not_observed_in_local_help: the token was not observed in that help surface; this is not parser rejection.
- abstain: the command or invocation is ambiguous, unsupported, truncated, or outside the fixed probe set.

## Invocation Matrix

| Need | Command | Executes candidate? | Output boundary |
| --- | --- | --- | --- |
| exact availability | tooltruth resolve <names...> | no | sparse JSON, presence only |
| bounded identity | tooltruth resolve --identity <names...> | fixed compiled-in probe only | version/implementation only |
| invocation help check | tooltruth validate --json -- <command> | no user-provided values | help hash, token status, exit code |
| project descriptor | tooltruth show <tool> --project . --json | no | one descriptor and resolution fact |
| compact context | tooltruth context --project . | fixed version probes | bounded facts, opt-in |
| managed health | tooltruth doctor <id> --json | fixed recipe only | health result |
| managed repair | tooltruth repair <id> --json | explicit action; may build/network | repair result and bounded evidence |
| managed execution | tooltruth exec <id> -- <args> | yes, managed ID only | direct argv result |

## Limits

Tooltruth does not prove:

- flags, subcommands, aliases, or runtime behavior from path presence;
- that a version supports a particular option;
- that a missing help token means parser rejection;
- safety or correctness of an arbitrary command;
- publisher authenticity from a local digest record;
- usefulness to an AI without a real blind benchmark.

## Safety

Discovery is read-only and does not persist a machine inventory. context is opt-in. doctor, repair, and exec are explicit higher-authority paths. Never pass secrets or untrusted shell text through the command contract.

## Execution Envelope

network_default: disabled
filesystem_writes: none for discovery and validation; managed paths are explicit actions
subprocesses: fixed probes only unless managed execution is explicitly selected
working_directory: current project or explicit project argument
timeout: bounded by the implementation
output_limit: bounded; raw probe output is not part of the public signal
cleanup: no background process; managed actions own their cleanup
authority: local command discovery and explicitly managed actions only

## Registry Admission Status

Tooltruth is an experimental candidate. Passing unit tests and clean local smoke do not prove intrinsic AI utility or retrieval utility. Those require separate blind experiments with held-out tasks and abstention controls.

## Deep Source

The implementation contract is in the repository README and DESIGN.md. Behavior claims must remain narrower than the observed signal.
