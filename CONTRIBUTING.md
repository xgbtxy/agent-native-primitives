# Contributing

Agent Native Primitives accepts focused fixes and small, evidence-backed CLI proposals.

## Before proposing a new tool

Open an issue that answers:

1. What single agent task is repeatedly slow or unreliable?
2. Which mature projects already address it?
3. Why is a new CLI still necessary?
4. What is the smallest deterministic input/output contract?
5. How will an external A/B test detect real improvement?

Tools that duplicate standard Unix utilities, add an agent framework, depend on one model, or return noisy ranked lists are out of scope.

## Development

```bash
go test -race ./...
go vet ./...
go build ./cmd/tooltruth
```

Keep discovery read-only. New executable probes, repair recipes, persistent state, or network behavior require an explicit trust-boundary review and tests for failure, rollback, and output claims.

By contributing, you agree that your contribution is licensed under Apache-2.0.
