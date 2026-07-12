# Phase Design

- **Phases**: 3

## Phase 1: Core CLI and Shared Types

- **Phase Type**: layer
- **Goal**: establish the module, command-line entrypoint, shared result types, target parsing, and baseline test execution.
- **Deliverables**:
  - `go.mod`
  - `cmd/gomut/main.go`
  - shared domain types for targets, mutation records, and summaries
  - baseline `go test` runner
  - target mode parsing for package/all/diff

## Phase 2: AST Mutation Engine

- **Phase Type**: feature
- **Goal**: detect mutation candidates from Go source and apply representative AST-based mutations.
- **Deliverables**:
  - mutation discovery from Go files
  - mutation kinds for comparison, logical, arithmetic, and guard clause cases
  - source rewrite support with file/line metadata
  - mutation viability checks based on parsing and formatting

## Phase 3: Mutation Execution and Reporting

- **Phase Type**: feature
- **Goal**: execute mutations one by one, classify outcomes, and emit human-readable and JSON Lines reports.
- **Deliverables**:
  - mutation application loop
  - per-mutation `go test` execution with timeout handling
  - result classification
  - stdout summary
  - JSON Lines writer

## Phase Order

1. Phase 1
2. Phase 2
3. Phase 3
