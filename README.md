# gomut

`gomut` is a mutation testing CLI for Go projects.

It helps you find weak spots in tests by mutating source code, running `go test` for each mutation, and classifying the result.

For the Japanese version, see [README.ja.md](README.ja.md).

## Features

- Run mutation testing for a single Go package
- Run against all Go packages in a repository with `--all`
- Scan only the files touched by a git diff with `--diff`
- Run mutation testing in an isolated temporary copy of the repository
- Discover mutation candidates from the AST
- Execute `go test` per mutation and classify the result
- Emit results as JSON Lines

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

```bash
gomut test --package ./sample
```

### All packages

```bash
gomut test --all
```

### Diff mode

```bash
gomut test --diff HEAD~1..HEAD
```

### Isolated execution

`gomut` runs each mutation in a temporary copy of the repository, so the working tree remains clean even if the process stops early.

### JSON Lines output

```bash
gomut test --package ./internal/gomut --jsonl
gomut test --package ./internal/gomut --jsonl mutations.jsonl
```

## Output

`stdout` is the JSON Lines output stream.

- `--jsonl` by itself writes to `stdout`
- `--jsonl <path>` writes to the given file

Summaries and auxiliary messages go to `stderr`.

Each JSONL record contains:

- `target`
- `started_at`
- `command`
- `summary`
- `mutation`

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

## Supported Mutations

`gomut` currently supports the following mutation kinds:

| Kind                    | Example                                                                        |
| ----------------------- | ------------------------------------------------------------------------------ |
| `comparison_operator`   | `==` -> `!=`, `!=` -> `==`, `<` -> `<=`, `>` -> `>=`, `<=` -> `<`, `>=` -> `>` |
| `logical_operator`      | `&&` -> `&#124;&#124;`, `&#124;&#124;` -> `&&`                                 |
| `guard_clause`          | Simple guard-clause return replacement                                         |
| `arithmetic_operator`   | `+` -> `-`, `-` -> `+`, `*` -> `/`, `/` -> `*`, `%` -> `*`                     |
| `bitwise_operator`      | `&` -> `&#124;`, `&#124;` -> `&`, `^` -> `&`, `&^` -> `&#124;`                 |
| `shift_operator`        | `<<` -> `>>`, `>>` -> `<<`                                                     |
| `assignment_arithmetic` | `+=` -> `-=`, `-=` -> `+=`, `*=` -> `/=`, `/=` -> `*=`, `%=` -> `*=`           |
| `assignment_shift`      | `<<=` -> `>>=`, `>>=` -> `<<=`                                                 |
| `assignment_bitwise`    | `&=` -> `&#124;=`, `&#124;=` -> `&=`, `^=` -> `&=`, `&^=` -> `&#124;=`         |
| `inc_dec`               | `++` -> `--`, `--` -> `++`                                                     |
| `control_flow`          | `switch x` condition inversion                                                 |
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
go run ./cmd/gomut test --package ./sample --jsonl
./gomut test --all
```
