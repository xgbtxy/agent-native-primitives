# Design contract

Agent Native Primitives follows six constraints:

1. **One job per CLI.** A tool must have a sentence-sized purpose.
2. **Truth before recall.** Abstain instead of upgrading presence, declarations, or similarity into behavior claims.
3. **Sparse output.** Return one supported signal or `null`, not a discovery dump.
4. **Local and model independent.** No model API, chat loop, IDE dependency, or background agent.
5. **Composable.** Stable exit codes and JSON must work from any shell or agent runtime.
6. **Evidence before expansion.** A new feature or tool needs a real external task where it beats existing primitives.

The repository may contain many commands, but it must never become a general agent framework or package manager.

Invocation validation follows an asymmetric evidence rule: an exact token observed in digest-bound local help may be reported as observed; absence is reported only as absence from that help surface, never as parser rejection. Truncated, ambiguous, unsupported, or combined-token cases abstain. User-provided values are never forwarded to a probe.

The model owns semantic choice and may query exact candidate names only when local uncertainty justifies the call. Default resolution is presence-only and executes nothing. Version identity is explicit opt-in and must never imply syntax or behavior support. Automatic context injection is optional and requires separate host-specific evidence.
