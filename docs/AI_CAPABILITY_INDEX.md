# AI Capability Index

Read this file first. Use one row only when its call_when condition is true. Read the linked capability page only when the common command is insufficient.

Format:

tool | signal | common command | deep page | call when | stop/limit

## Canonical Index

| Tool | Signal | Common command | Deep page | Call when | Stop/limit |
| --- | --- | --- | --- | --- | --- |
| tooltruth resolve | path_resolved / absent / path+version_observed | tooltruth resolve <exact-name>... | capabilities/tooltruth.md | A concrete candidate command is unfamiliar or version identity changes the decision | Does not prove flags, behavior, aliases, or network support |
| tooltruth validate | observed_in_local_help / not_observed_in_local_help / abstain | tooltruth validate --json -- <approved invocation> | capabilities/tooltruth.md | An exact invocation is proposed and its local help surface must be checked before execution | Never forwards user values to the probe; abstains on ambiguity |
| tooltruth context | bounded_context_observed | tooltruth context --project . | capabilities/tooltruth.md | A host-specific experiment proves compact environment facts reduce net work | Opt-in only; version facts do not prove runtime behavior |
| tooltruth version | version_reported | tooltruth version | capabilities/tooltruth.md | The Tooltruth implementation itself must be identified | Reports Tooltruth only |
| tooltruth find | curated_name_mapping / path_resolved / no_supported_match | tooltruth find --json "<intent>" --project . | capabilities/tooltruth.md | Legacy semantic discovery is explicitly under comparison | Experimental; not the default exact-resolution path |
| tooltruth show | project_descriptor_observed | tooltruth show <tool> --project . --json | capabilities/tooltruth.md | One project-defined capability needs its descriptor | Does not execute the described tool |
| tooltruth doctor | fixed_probe_passed / error | tooltruth doctor <managed-id> --json | capabilities/tooltruth.md | An explicit managed-tool health check is requested | Fixed recipes only; no general command probing |
| tooltruth repair | repair_completed / error | tooltruth repair <managed-id> --json | capabilities/tooltruth.md | A declared repair is explicitly approved | May use network/build permissions; never automatic |
| tooltruth exec | managed_exec_completed / error | tooltruth exec <managed-id> -- <args> | capabilities/tooltruth.md | An explicitly managed capability is approved for execution | Managed IDs only; direct argv; review side effects first |

## Minimal Agent Rule

When availability of concrete command candidates is uncertain:

tooltruth resolve <candidate>...

Use only entries in present. Add --identity only when implementation or version changes the decision. Do not call Tooltruth before familiar commands.

## Registry Status

The registry is the authority for trusted capabilities. This index may contain experimental rows, but every row must state its status and deep page.
