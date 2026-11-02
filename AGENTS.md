# AGENTS.md

## Principles

- Comments are a last resort. If a comment explains what code does, fix the code instead — rename, extract a well-named helper, restructure, or express the case as a test name — until the comment has nothing left to say. Never add a comment to compensate for an unclear name or expression.
- Exception: doc comments on exported identifiers are functional, not decorative — they are rendered into the public README (see Traps). Write and maintain them.
- Prefer table-driven tests for option combinations and edge cases; colocate them with the code (`retry_test.go`).
- Commits follow conventional style: concise, imperative, optional scope — `ci(workflow): skip codecov upload on dependabot PRs`.

## Commands

- `make setup` — install tooling (`gotestcover`, `cover`, `godocdown`); needed once on a fresh clone before `make test` works.
- `make test` — regenerates README, runs tests with coverage, runs linters. Run before every PR. **Requires Docker** — golangci-lint runs in a container.
- `make test_and_cover_report TEST_PATTERN=YourTest` — narrow the suite to a single test.
- `make generate` — regenerate `README.md` from source docs.

## Traps

**`README.md` is generated — never edit it directly.** It is rendered by `godocdown` from the package doc comment in `retry.go` through the `.godocdown.tmpl` template. Content changes (examples, migration notes) go in the `retry.go` doc comment; layout changes go in the template. Then run `make generate` and commit the result.

**The module floor is Go 1.20 (`go.mod`).** All code must compile on 1.20 — stdlib additions newer than that are off-limits (e.g. `maps.Clone`, the `slices` package, `min`/`max` builtins are 1.21+). A modern local toolchain will not catch this; CI's 1.20 jobs will.
