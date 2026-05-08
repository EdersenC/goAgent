# Failure Audit (Orchestration #8 - Task #4)

Scope: inspect the current `water` codebase for outdated dependency/runtime assumptions and define focused update targets for resuming work safely.

## Validation Run
- `go test ./...` -> pass (`no test files` across packages)
- `go build ./...` -> failed in this sandbox due environment/tooling (`/snap/bin/go` cannot run: missing snap capabilities), not a compile error from project code.

## Priority Update Targets

1. Runtime pinning to unreleased/fragile Go toolchain
- Evidence: [go.mod](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/go.mod:3) sets `go 1.23.0`; [go.mod](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/go.mod:5) sets `toolchain go1.23.6`.
- Risk: builds depend on a very specific local toolchain path/state; fails in constrained CI/containers and breaks reproducibility.
- Focused target: set a stable, commonly available Go baseline and make `toolchain` optional/documented rather than required.

2. Hard-coded runtime dependency on local Ollama endpoints/ports
- Evidence: [agents.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/agents.go:166) uses `http://localhost:11434`; [api/search/Search.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/search/Search.go:192) hard-codes alternate summarizer port `11436`.
- Risk: startup and summarization fail outside the original workstation layout.
- Focused target: move provider host/port into env-backed config with defaults and validate connectivity during startup.

3. Working-directory sensitive config/tool JSON loading
- Evidence: [cmd/main.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/cmd/main.go:15) opens `agents.json` via relative path; [api/tools/core.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/tools/core.go:22) and [api/search/Search.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/search/Search.go:138) load JSON files by relative names.
- Risk: binary works only when executed from repo root.
- Focused target: resolve config paths via explicit flag/env and provide clear boot-time errors.

4. Incomplete interface implementation triggers panic path
- Evidence: [api/tools/core.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/tools/core.go:44) has `panic("implement me")` in `DuckDuckGo.Trace()`.
- Risk: future use of this method becomes a runtime crash.
- Focused target: either implement the method or remove it from interface expectations so panic path is impossible.

5. HTTP safety/robustness assumptions in search and scraping
- Evidence: request creation ignores errors (`req, _ := http.NewRequest(...)`) in [api/tools/core.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/tools/core.go:53) and [api/search/Search.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/search/Search.go:21); no status-code checks after requests.
- Risk: silent failures, nil request usage risk, and hard-to-debug behavior when providers/search engines throttle or change responses.
- Focused target: handle request construction errors and enforce explicit `2xx` checks before parsing.

6. Search result filtering assumes HTTPS-only links
- Evidence: [api/search/Search.go](/home/eddy/.arc-tech/excalidraw-workspaces/excalidraw/water-water/worktrees/orch-8-agent-1-failure-audit/api/search/Search.go:17) skips non-HTTPS URLs.
- Risk: DuckDuckGo redirect/outbound formats can be skipped unexpectedly, reducing recall to zero for valid queries.
- Focused target: normalize/resolve URLs before filtering; log skip reasons with counters.

7. Observability and regression protection gaps
- Evidence: test run reports no tests; there are no unit/integration checks for agent load, URL construction, search parsing, or summarization flow.
- Risk: modernization PRs can regress behavior with no CI signal.
- Focused target: add minimal tests for path resolution, provider URL building, and parser logic before larger refactors.

## Suggested Small PR Sequence
1. Config bootstrap PR: env/flag-driven provider + config paths; startup validation.
2. Networking hardening PR: request/status error handling + logging.
3. Interface cleanup PR: remove/implement panic path (`Trace`).
4. Baseline quality PR: add focused tests for boot/config/network parser paths.
