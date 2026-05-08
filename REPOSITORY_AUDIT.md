# Repository Audit (2026-05-08)

Scope: targeted modernization audit for `github.com/EdersenC/goAgent` with small, mergeable PRs.

## Current Baseline

- Language/runtime: Go module with `go 1.23.0` and `toolchain go1.23.6` in `go.mod`.
- Direct dependency footprint is small:
  - `github.com/PuerkitoBio/goquery v1.10.3`
  - `golang.org/x/net v0.39.0` (indirect)
  - `github.com/andybalholm/cascadia v1.3.3` (indirect)
- Tooling assumptions:
  - CLI starts from repo root and expects local JSON files (`agents.json`, `search.json`, `respond.json`, etc.).
  - Ollama endpoints and model names are hardcoded in `agents.json`.
  - Search integration depends on live network access to DuckDuckGo HTML endpoint.

## Outdated / Fragile Runtime Assumptions

1. Local-only model runtime assumptions
- `agents.json` pins `http://localhost:11434`/`11435` and specific Ollama model tags.
- No environment-variable override path for provider URL/ports/model IDs.

Impact: app startup and behavior are environment-coupled; portability is limited.

2. Build/test environment assumptions
- No Make targets and no CI config in repo.
- Reproducible validation path is undocumented.

Impact: contributor onboarding and upgrade verification are inconsistent.

3. Search provider coupling
- `api/tools/core.go` directly scrapes `https://html.duckduckgo.com/html/` via HTML selectors.
- No provider abstraction for retries/timeouts/backoff tuning per provider instance.

Impact: brittle behavior when HTML format changes; production reliability risk.

4. Crash path left in production code
- `DuckDuckGo.Trace()` has `panic("implement me")` in `api/tools/core.go`.

Impact: accidental call path can terminate process.

5. Error handling style is CLI-hard-exit oriented
- Multiple `os.Exit(1)` paths in initialization.

Impact: difficult embedding as library and harder automated testing.

## Focused Upgrade Targets (Small PR Plan)

### PR 1: Runtime Config Externalization
- Add environment overrides for provider base URL, ports, and model IDs.
- Keep `agents.json` defaults, but allow env to patch at load time.
- Document required env variables in README.

Acceptance target:
- App runs unchanged without env vars.
- App can run against non-default host/ports/models via env only.

### PR 2: Safe Engine Interface + Non-Panic Contract
- Replace `panic("implement me")` in `DuckDuckGo.Trace()` with a safe implementation or remove method from interface if unused.
- Add unit tests for search engine methods that do not require network.

Acceptance target:
- No panic placeholders in non-test runtime code.

### PR 3: Deterministic Dev Validation
- Populate `Makefile` with stable targets (`test`, `vet`, optional `fmt-check`).
- Add contributor note for writable `GOCACHE`/`GOMODCACHE` when sandboxed.

Acceptance target:
- Single command provides baseline validation for contributors and CI.

### PR 4: HTTP Hardening for Search
- Introduce shared HTTP client with timeouts.
- Add retry/backoff policy and explicit status-code handling.
- Add parsing guards and structured errors for selector misses.

Acceptance target:
- Predictable failure modes with actionable errors.

### PR 5: Dependency Maintenance Guardrails
- Add periodic dependency check workflow (or documented command) using `go list -m -u all`.
- Track upgrade cadence in docs.

Acceptance target:
- Clear process to identify dependency updates before breakage accumulates.

## Recommended Validation Commands

Use local writable caches when environment restricts `~/.cache`:

```bash
mkdir -p .codex-tmp/go-build .codex-tmp/go-mod
GOCACHE=$(pwd)/.codex-tmp/go-build \
GOMODCACHE=$(pwd)/.codex-tmp/go-mod \
GOTOOLCHAIN=local /usr/local/go/bin/go test ./...
```

## Notes From This Audit Run

- `go test ./...` could not complete in this sandbox due to network restrictions fetching modules from `proxy.golang.org`.
- A normal networked dev/CI environment is required for first-time module resolution.
