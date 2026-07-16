# gomut

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/goropikari/gomut)

`gomut` is a mutation testing CLI for Go projects.

It helps you find weak spots in tests by mutating source code, running `go test` for each mutation, and classifying the result.

This repository is an AI-assisted vibe-coding project.

For the Japanese version, see [README.ja.md](README.ja.md).

## Features

- Run mutation testing for a single Go package
- Run against all Go packages in a repository with `./...`
- Scan only the files touched by a git diff with `--diff`
- Run mutation testing in an isolated temporary copy of the repository
- Run mutations in parallel with `--parallel`
- Discover mutation candidates from the AST
- Execute `go test` per mutation and classify the result
- Configure a per-mutation timeout with `--timeout`
- Control progress reporting with `--progress`
- Filter mutation candidates by kind with `--mode`, `--enable`, and `--disable`
- Emit results as JSON Lines
- Generate an optional HTML report
- Generate an optional SARIF report for IDEs and editor integrations

## Install

```bash
go install ./cmd/gomut
```

## Dev Container

The repository includes a dev container configured for Go 1.26.

If you need the development tools, run:

```bash
make install-dev-tools
```

That installs `codex`, `dprint`, and `gitleaks`.

## Usage

### Package mode

Use `./sample` for one package, or `./sample/...` to include all packages under `sample/`.
For example, `./sample/...` picks up packages such as `./sample/alpha` and `./sample/beta`.

```bash
gomut test ./sample
gomut test ./sample/...
```

### Repository packages

```bash
gomut test ./...
```

### Diff mode

```bash
gomut test --diff HEAD~1..HEAD
gomut test --diff main
```

### Isolated execution

`gomut` runs each mutation in a temporary copy of the repository, so the working tree remains clean even if the process stops early.

### Parallel execution

`gomut` can run mutation candidates in parallel with `--parallel <n>`.

- The default worker count is the number of CPU cores.
- Set `--parallel 1` to keep sequential behavior.
- Workers use isolated temporary copies, and results are collected before output is written, so JSONL stays valid.

### JSON Lines output

```bash
gomut test ./sample --jsonl
gomut test ./sample --jsonl mutations.jsonl
gomut test ./sample --type lived --jsonl
gomut test ./sample --mode all --disable string_literal --jsonl
gomut test ./sample --enable bitwise_operator --disable return --jsonl
gomut test ./sample --jsonl mutations.jsonl --html report.html
```

### HTML output

```bash
gomut test ./sample --html
gomut test ./sample --html report.html
gomut test ./sample --jsonl mutations.jsonl --html report.html
```

### SARIF output

```bash
gomut test ./sample --sarif
gomut test ./sample --sarif gomut.sarif
gomut test ./sample --jsonl mutations.jsonl --sarif gomut.sarif
```

SARIF is a good fit for editors that can import static-analysis results. For Neovim, point a SARIF-aware plugin or importer at the generated file to surface surviving mutations as diagnostics.

### Timeout

Each mutation run uses a per-mutation timeout. The default is `10s`.

```bash
gomut test ./sample --timeout 30s
```

You can also set the default in `.gomut.yaml`:

```yaml
timeout: 30s
```

CLI flags and positional targets override config file values.

### Progress

Progress reporting is controlled with `--progress=auto|on|off`.

```bash
gomut test ./sample --jsonl mutations.jsonl --progress=on
```

`auto` is the default. It shows progress in interactive terminals and stays quiet in non-TTY and CI runs.
If you want to watch progress comfortably, send JSONL to a file instead of `stdout`.

You can also set the default in `.gomut.yaml`:

```yaml
progress: on
```

CLI flags override config file values.

### Config file

`gomut` loads `.gomut.yaml` from the repository root by default. You can also point to a different file with `--config`.

```yaml
target:
  mode: package
  value: ./sample/...
timeout: 30s
progress: on
parallel: 4
jsonl: mutations.jsonl
html: report.html
sarif: report.sarif
type:
  - lived
kind:
  mode: standard
  enable:
    - bitwise_operator
  disable:
    - return
exclude:
  - "*.pb.go"
  - "*_mock.go"
  - internal/generated
isolation:
  copy_exclude:
    - tmp
    - internal/cache/**
```

`exclude` skips mutation candidates. `isolation.copy_exclude` skips files or directories when gomut creates isolated temporary repository copies. It accepts entry names, repository-relative paths, and glob patterns.

See [docs/gomut-config.schema.json](docs/gomut-config.schema.json) for the JSON Schema.

## Output

By default, JSON Lines are written to `stdout`.

- `--jsonl` by itself writes to `stdout`
- `--jsonl <path>` writes to the given file
- `--html` by itself writes the HTML report to `stdout`
- `--html <path>` writes the HTML report to the given file
- `--html <path>` without `--jsonl` suppresses JSONL output
- `--sarif` by itself writes the SARIF report to `stdout`
- `--sarif <path>` writes the SARIF report to the given file
- If you request multiple pathless machine-readable outputs, only one can use `stdout`; give the others file paths
- `--progress=auto|on|off` controls mutation progress reporting on `stderr`
- `--progress` defaults to `auto`, which shows progress in interactive terminals and stays quiet in non-TTY and CI runs
- `--mode` selects the base mutation kind set (`standard` or `all`)
- `--enable` adds mutation kinds on top of the selected mode
- `--disable` removes mutation kinds, and it wins over `--enable`
- `--exclude` adds file patterns to skip before mutation discovery
- When no kind flags are set, `standard` is used by default
- Kind selection affects the JSONL records, HTML report, and summary counts
- `--type` filters emitted mutation results after execution
- `--type` accepts single values, comma-separated values, and repeated flags
- `--type` affects both JSONL output and the summary on `stderr`
- Excluded files and candidates are skipped before mutation generation
- Exclusion reasons are printed to `stderr`

Summaries and auxiliary messages go to `stderr`.

Each JSONL record contains:

- `target`
- `started_at`
- `command`
- `summary`
- `mutation`

See [docs/jsonl-record.schema.json](docs/jsonl-record.schema.json) for the JSON Schema.

`started_at` is an RFC3339 timestamp in actual runs.

`mutation` includes at least:

- `file`
- `line`
- `kind`
- `original`
- `replacement`
- `result`
- `message`

The `result` field uses these values:

- `KILLED`
- `LIVED`
- `NOT COVERED`
- `TIMED OUT`
- `NOT VIABLE`

`--type` accepts these values in lower case, and also accepts hyphenated or space-separated forms such as `not-covered` and `timed-out`.

## Exclusions

`gomut` supports multiple exclusion rules:

- File patterns configured in `.gomut.yaml` with `exclude`
- File patterns passed on the CLI with `--exclude`
- Function-level or line-level comments using `//gomut:ignore`

File patterns are matched against repository-relative paths. A pattern can match a full path or a base name, so both of the following are valid:

```yaml
exclude:
  - "*.pb.go"
  - "*_mock.go"
  - internal/generated
```

`//gomut:ignore` applies to the annotated function, statement, or block. By default, excluded candidates are kept quiet on `stderr`. Pass `--verbose` to print the exclusion reason on `stderr` for diagnostics.

## Supported Mutations

`gomut` currently supports the following mutation kinds:

| Kind                    | Example                                                                        |
| ----------------------- | ------------------------------------------------------------------------------ |
| `comparison_operator`   | `==` -> `!=`, `!=` -> `==`, `<` -> `<=`, `>` -> `>=`, `<=` -> `<`, `>=` -> `>` |
| `logical_operator`      | `&&` -> \|\|, \|\| -> `&&`                                                     |
| `guard_clause`          | Simple guard-clause return replacement                                         |
| `arithmetic_operator`   | `+` -> `-`, `-` -> `+`, `*` -> `/`, `/` -> `*`, `%` -> `*`                     |
| `bitwise_operator`      | `&` -> \|, \| -> `&`, `^` -> `&`, `&^` -> \|                                   |
| `shift_operator`        | `<<` -> `>>`, `>>` -> `<<`                                                     |
| `assignment_arithmetic` | `+=` -> `-=`, `-=` -> `+=`, `*=` -> `/=`, `/=` -> `*=`, `%=` -> `*=`           |
| `assignment_shift`      | `<<=` -> `>>=`, `>>=` -> `<<=`                                                 |
| `assignment_bitwise`    | `&=` -> \|=, \|= -> `&=`, `^=` -> `&=`, `&^=` -> \|=                           |
| `inc_dec`               | `++` -> `--`, `--` -> `++`                                                     |
| `control_flow`          | `if` / `for` / `switch` condition inversion                                    |
| `loop_control`          | `break` -> `continue`, `continue` -> `break` within `for` / `range` loops      |
| `return`                | `return true` -> `return false`, `return false` -> `return true`               |
| `nil_check`             | `== nil` -> `!= nil`, `!= nil` -> `== nil`                                     |
| `boolean_literal`       | `true` -> `false`, `false` -> `true`                                           |
| `integer_literal`       | `0` -> `1`, non-zero integer literal -> `0`                                    |
| `float_literal`         | `0.0` -> `1.0`, non-zero float literal -> `0.0`                                |
| `rune_literal`          | `'a'` -> `'b'`, non-`'a'` rune literal -> `'a'`                                |
| `unary_not`             | `!x` -> `x`                                                                    |
| `unary_minus`           | `-x` -> `x`                                                                    |
| `unary_bitwise_not`     | `^x` -> `x`                                                                    |
| `string_literal`        | `""` -> `"mutated"`, non-empty string literal -> `""`                          |

The default `standard` kind set is:

- `comparison_operator`
- `logical_operator`
- `arithmetic_operator`
- `guard_clause`
- `return`
- `nil_check`

## Preconditions

- Baseline `go test` must pass before mutation testing starts
- Go 1.26 or later is required
- `--diff` requires git

## Testing

Repository testing guidelines are documented in [docs/testing-guidelines.md](docs/testing-guidelines.md).

Key points:

- Use `testify`
- Follow the AAA pattern
- Always use `t.Run`
- Avoid table-driven tests unless they clearly improve readability
- Prefer test names in the form `Test{TargetFunctionName}`

## Development

```bash
go test ./...
```

```bash
make fmt
```

```bash
make lint
```

When needed, verify behavior with commands such as:

```bash
go run ./cmd/gomut test ./sample --jsonl
./gomut test ./...
```
