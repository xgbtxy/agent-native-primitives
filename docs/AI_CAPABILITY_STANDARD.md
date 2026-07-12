# AI Capability Standard

This is the canonical English standard for Agent Native Primitives.

## 1. Capability Contract

Every capability publishes these fields:

tool_id
purpose
call_when
common_command
signal
scope
side_effects
failure_behavior
deep_doc
benchmark_ref
status

The first eight fields must fit in the quick index. Detailed usage belongs in a capability page.

## 2. Signal Rules

A signal is an observed fact, not a recommendation or a prediction.

Allowed signal properties:

- source: where the observation came from;
- scope: the exact local/project scope;
- status: present, absent, observed, abstain, or error;
- limits: what the signal does not prove;
- evidence: a short hash, exit code, or bounded reference when available.

Never turn path presence, a catalog description, a version string, or a matching intent into a claim about runtime behavior.

## 3. Invocation Rules

1. The AI calls a capability only when the call_when condition is met.
2. The common command must be safe, bounded, and copyable.
3. A capability must not be called before every shell command.
4. An unknown or ambiguous request must abstain or ask for an exact candidate.
5. The deep page is read only when the common command or signal is insufficient.
6. Output must be sparse JSON or a stable line format.

## 4. Side-Effect Declaration

Each capability declares:

- network: disabled, opt_in, or enabled;
- execution: none, fixed_probe, or explicit_action;
- writes: none or exact workspace path;
- cleanup: command or not_applicable;
- authority: the maximum local authority it can exercise.

The default path must be the least powerful path.

## 5. Lifecycle

The capability lifecycle is:

observed -> candidate -> experiment -> benchmark -> registry -> release

Research or installation alone never promotes a capability.

Admit a capability only when:

- a real AI task exists;
- primary source, version, license, and install facts are verified;
- a clean run is reproducible;
- a new AI session passes without private coaching;
- a multi-task A/B benchmark shows net benefit;
- failure, cleanup, rollback, and expiry are recorded.

## 6. English and Chinese Repositories

The English repository is canonical:

- command IDs, JSON fields, signal names, exit codes, and safety semantics are defined here;
- the Chinese repository mirrors the same contract and translates explanations only;
- translations must not rename commands, signals, fields, or status values;
- English changes land first, then the Chinese mirror is synchronized;
- a Chinese translation never changes the runtime contract.

## 7. Registry Rule

Every registry entry must have a matching quick-index row. A tool not present in the registry may be documented as experimental or candidate, but must not be presented as trusted.

## 8. Canonical Semantic Bundle

contracts/semantic_bundle.json is the normative machine-readable bundle. It owns command IDs, signal IDs, lifecycle values, implementation strategies, exit-code semantics, and safety rule IDs.

The Chinese mirror vendors this file byte-for-byte. Prose may translate; identifiers may not.

## 9. Evidence and Artifact Identity

Evidence is typed:

- lead: research or community signal used only for prioritization;
- primary_verified: upstream identity, version, license, and declared contract;
- locally_measured: clean execution, safety probe, compatibility, benchmark, or rollback result.

Every registered capability binds its evidence to an exact artifact fingerprint, compatibility tuple, owner, verified_at, expires_at, release fingerprint, and rollback reference.

## 10. Two Utility Gates

Intrinsic utility asks whether the capability improves a task when directly available.

Retrieval utility asks whether a fresh AI session can discover it from the normal index and abstain on near misses.

Both must pass before registry admission. Correct output alone is not net AI value.

## 11. Execution Envelope

The contract must declare network, filesystem writes, subprocesses, required environment, working directory, timeout, output limit, nondeterminism, cleanup, and maximum authority. A clean run does not excuse undeclared behavior.

## 12. Lifecycle and Portfolio

Implementation strategy and lifecycle status are separate. Release is an artifact event, not a status. Expiry, artifact drift, safety regression, lost ownership, or incompatibility removes an entry from active retrieval while preserving history. The active index and maintenance budget are finite.
