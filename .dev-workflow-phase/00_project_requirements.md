# Project Requirements

## Project Name

Go mutation testing CLI for AI agents.

## Background

This repository will provide a command-line mutation testing tool for Go code. The primary user is an AI coding agent that needs to find weak tests, inspect mutation results mechanically, and decide the next action without human interpretation.

## Goals

- Run mutation testing against Go packages, the whole repository, or a diff-selected subset.
- Use AST-based mutation candidates for representative operators and guard clauses.
- Execute `go test` for each mutation and classify the result.
- Produce machine-readable output that is easy to stream and filter for `LIVED` mutations.
- Keep the first usable version simple and reliable rather than highly optimized.

## Inputs

- Target mode:
  - package
  - all
  - diff
- Target value:
  - package import path, repository root, or git diff range/commit reference
- Optional timeout per mutation
- Optional output file path for JSON Lines output

## Outputs

- Human-readable summary on stdout
- JSON Lines records for each mutation result
- Exit code `0` when execution completes
- Non-zero exit code when baseline test execution fails or a mutation run cannot proceed

## Functional Requirements

### 1. Baseline Validation

- The tool must run a baseline `go test` before mutation execution.
- If baseline tests fail, mutation execution must stop.

### 2. Target Selection

- The tool must support package mode.
- The tool must support repository-wide mode.
- The tool must support diff-based mode using git changed lines.

### 3. Mutation Discovery

- The tool must parse Go source files using the standard Go parser.
- The tool must generate mutation candidates from AST nodes.
- Initial supported mutation kinds:
  - comparison operators
  - logical operators
  - arithmetic operators
  - guard clause return simplification

### 4. Mutation Execution

- The tool must apply one mutation at a time.
- The tool must run `go test` after applying a mutation.
- The tool must classify each mutation as:
  - `KILLED`
  - `LIVED`
  - `NOT COVERED`
  - `TIMED OUT`
  - `NOT VIABLE`

### 5. Reporting

- The tool must print a summary with total counts.
- The tool must emit one JSON object per mutation result in JSON Lines format.
- Each record must include target metadata, summary metadata, and mutation metadata.
- The output must be stable enough for downstream automation.

## Non-Goals

- HTML reporting
- CI quality gates
- Auto-fixing code
- Exhaustive mutation coverage
- Parallel execution in the first version

## Constraints

- Use only the Go standard library unless a stronger dependency is justified.
- Keep source edits and generated artifacts committed phase by phase.
- Preserve existing documentation files.

## Success Criteria

- A user can run the CLI against a package and get meaningful mutation results.
- A user can identify `LIVED` mutations from JSON Lines output.
- The implementation is organized so additional mutators and execution modes can be added later.
