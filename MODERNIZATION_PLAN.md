# Modernization Plan (Task #2)

This repository currently builds, but it carries several outdated runtime assumptions and missing guardrails that make upgrades risky. This plan defines focused, small-PR targets.

## Current Baseline

- Language/runtime: `go 1.23.0` with `toolchain go1.23.6` in `go.mod`
- Direct dependency: `github.com/PuerkitoBio/goquery v1.10.3`
- Validation state:
  - `go test ./...` passes (no test files)
  - `go vet ./...` is currently blocked in this environment by host packaging/capability restrictions, not code compile issues

## Outdated Assumptions Identified

1. Hardcoded local model runtime endpoints and ports:
- `localhost:11434` default in provider constructor
- extra hardcoded summarizer port `11436` in search worker flow
- Assumes Ollama-style endpoints only

2. Configuration is file-path and cwd sensitive:
- Startup expects `agents.json` in current working directory
- Tool definitions rely on relative JSON files in project root

3. Network behavior is inconsistent and under-specified:
- Some clients have explicit timeouts, others do not
- Request creation ignores errors in a few places (`http.NewRequest(...); req, _ := ...`)
- No shared context cancellation strategy

4. Validation workflow is minimal:
- No CI/test matrix representation in repo
- `Makefile` had no targets, reducing reproducibility for contributors

5. Security and transport assumptions are mixed:
- Search scraper enforces `https://` only for content fetch
- External query step still relies on remote HTML scraping behavior that may change

## Focused Update Targets (Small PR Sequence)

1. Runtime configuration extraction
- Move model provider base URL, ports, and endpoints to env-driven config with defaults.
- Replace hardcoded `11436` summary port fanout with configurable worker endpoints.

2. Path/config hardening
- Resolve `agents.json` and tool JSON paths from explicit config or executable-relative path.
- Fail with actionable errors when files are missing.

3. HTTP client standardization
- Introduce shared HTTP client factory with timeout + transport defaults.
- Handle `http.NewRequest` errors everywhere.
- Add context-aware request paths for cancellable long-running searches.

4. Validation uplift
- Keep lightweight local validation (`make validate`) and add baseline CI workflow:
  - `go test ./...`
  - `go fmt` / formatting check
  - `go vet ./...` (where environment supports it)

5. Compatibility and dependency hygiene
- Keep Go toolchain pinned and review quarterly.
- Use `go list -m -u all` in CI or release workflow to detect stale modules.

## Suggested Next PR Titles

1. `config: externalize provider endpoints and worker ports`
2. `runtime: harden agent/tool file resolution and startup errors`
3. `net: standardize HTTP clients and request error handling`
4. `ci: add baseline validate workflow for test/fmt/vet`
5. `deps: add module update audit workflow`
