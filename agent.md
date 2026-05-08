# Agent Notes

## Scope
This document accompanies `FAILURE_AUDIT.md` and is intended for the execution agent handling follow-up fixes.

## Goal
Reduce failure risk from outdated runtime and dependency assumptions using small, low-blast-radius pull requests.

## Source of Findings
All prioritized findings, evidence, and patch targets are documented in `FAILURE_AUDIT.md`.

## Execution Order
1. Stabilize and pin external integration assumptions (`api/search/helper.go`, `api/search/Search.go`).
2. Reduce brittle parsing/shape assumptions in request/response handling (`helper.go`, `respond.json`).
3. Align model/runtime defaults with current provider behavior (`api/modelpicker.json`, `api/search/model.go`).
4. Update dependency/runtime expectations (`go.mod`, `README.md`) and run validation.

## Validation Expectations
Run at minimum:
- `go test ./...`
- `go build ./...`

If build is blocked by host environment constraints, record the exact blocker and proceed with test evidence.

## Constraints
- Keep each fix small and isolated.
- Avoid unrelated refactors.
- Preserve existing behavior unless the failure mode requires a change.
