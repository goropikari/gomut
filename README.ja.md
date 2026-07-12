# gomut

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/goropikari/gomut)

`gomut` は Go 向けの mutation testing CLI です。

ソースコードを変異させ、各 mutation ごとに `go test` を実行して、テストの弱い部分を機械的に見つけるために使います。

このリポジトリは、AI を使った vibe coding の成果物です。

英語版の README は [README.md](README.md) を参照してください。

## 機能

- 単一の Go パッケージに対して mutation testing を実行
- `--all` でリポジトリ内の全 Go パッケージを対象化
- `--diff` で git 差分に含まれるファイルだけを走査
- 一時的なコピー上で安全に mutation testing を実行
- AST から mutation 候補を検出
- mutation ごとに `go test` を実行して結果を分類
- `--timeout` で mutation ごとのタイムアウトを設定
- `--progress` で進捗表示を制御
- 結果を JSON Lines 形式で出力
- HTML レポートを任意で生成

## インストール

```bash
go install ./cmd/gomut
```

## Dev Container

このリポジトリには Go 1.26 向けの dev container が含まれています。

開発用ツールが必要な場合は、次を実行してください。

```bash
make install-dev-tools
```

`codex`、`dprint`、`gitleaks` をまとめて導入します。

## 使い方

### パッケージ単位

```bash
gomut test --package ./sample
```

### 全パッケージ

```bash
gomut test --all
```

### diff モード

```bash
gomut test --diff HEAD~1..HEAD
gomut test --diff main
```

### 安全な実行

`gomut` は各 mutation を一時ディレクトリ上のコピーで実行するため、途中で止まっても作業ツリーに変更が残りません。

### JSON Lines 出力

```bash
gomut test --package ./internal/gomut --jsonl
gomut test --package ./internal/gomut --jsonl mutations.jsonl
gomut test --package ./internal/gomut --type lived --jsonl
gomut test --package ./internal/gomut --jsonl mutations.jsonl --html report.html
```

### HTML 出力

```bash
gomut test --package ./internal/gomut --html
gomut test --package ./internal/gomut --html report.html
gomut test --package ./internal/gomut --jsonl mutations.jsonl --html report.html
```

### タイムアウト

各 mutation の実行には個別のタイムアウトがあり、デフォルトは `10s` です。

```bash
gomut test --package ./sample --timeout 30s
```

`.gomut.yaml` にも設定できます。

```yaml
timeout: 30s
```

CLI フラグは config ファイルの値より優先されます。

### progress

`--progress=auto|on|off` で mutation の進捗表示を `stderr` に出すか制御できます。

```bash
gomut test --package ./sample --jsonl mutations.jsonl --progress=on
```

`auto` が既定値です。対話的な端末では進捗を表示し、非 TTY や CI では静かに動作します。
進捗を見やすくしたい場合は、JSONL を `stdout` ではなくファイルに出してください。

`.gomut.yaml` にも設定できます。

```yaml
progress: on
```

CLI フラグは config ファイルの値より優先されます。

### config ファイル

`gomut` はデフォルトでリポジトリルートの `.gomut.yaml` を読みます。`--config` で別ファイルを指定することもできます。

```yaml
target:
  mode: package
  value: ./sample
timeout: 30s
progress: on
jsonl: mutations.jsonl
html: report.html
```

## 出力

JSON Lines はデフォルトで `stdout` に出ます。

- `--jsonl` だけを指定した場合は `stdout` に出力
- `--jsonl <path>` を指定した場合はそのファイルに出力
- `--html` だけを指定した場合は HTML レポートを `stdout` に出力
- `--html <path>` を指定した場合はそのファイルに出力
- `--html <path>` を指定して `--jsonl` を付けない場合、JSONL 出力は抑止される
- `--progress=auto|on|off` で mutation の進捗表示を `stderr` に出すか制御できる
- `--progress` の既定値は `auto` で、対話的な端末では進捗を表示し、非 TTY や CI では静かに動作する
- `--type` は mutation 実行後の結果を絞り込む
- `--type` は単一指定、カンマ区切り、繰り返し指定に対応する
- `--type` は JSONL 出力と `stderr` の summary の両方に反映される

集計サマリや補助メッセージは `stderr` に出ます。

各 JSONL レコードには次の情報が入ります。

- `target`
- `started_at`
- `command`
- `summary`
- `mutation`

JSON Schema は [docs/jsonl-record.schema.json](docs/jsonl-record.schema.json) を参照してください。

`started_at` は実際の実行では RFC3339 形式の時刻になります。

`mutation` には少なくとも次が含まれます。

- `file`
- `line`
- `kind`
- `original`
- `replacement`
- `result`
- `message`

`result` の値は次のとおりです。

- `KILLED`
- `LIVED`
- `NOT COVERED`
- `TIMED OUT`
- `NOT VIABLE`

`--type` には小文字を使います。`not-covered` や `timed-out` のようなハイフン区切り、`not covered` のような空白区切りも受け付けます。

## 対応 mutation

現在対応している mutation 種別は次のとおりです。

| 種類                    | 例                                                                             |
| ----------------------- | ------------------------------------------------------------------------------ |
| `comparison_operator`   | `==` -> `!=`、`!=` -> `==`、`<` -> `<=`、`>` -> `>=`、`<=` -> `<`、`>=` -> `>` |
| `logical_operator`      | `&&` -> `&#124;&#124;`、`&#124;&#124;` -> `&&`                                 |
| `guard_clause`          | 単純な guard clause の return 差し替え                                         |
| `arithmetic_operator`   | `+` -> `-`、`-` -> `+`、`*` -> `/`、`/` -> `*`、`%` -> `*`                     |
| `bitwise_operator`      | `&` -> `&#124;`、`&#124;` -> `&`、`^` -> `&`、`&^` -> `&#124;`                 |
| `shift_operator`        | `<<` -> `>>`、`>>` -> `<<`                                                     |
| `assignment_arithmetic` | `+=` -> `-=`, `-=` -> `+=`, `*=` -> `/=`, `/=` -> `*=`, `%=` -> `*=`           |
| `assignment_shift`      | `<<=` -> `>>=`、`>>=` -> `<<=`                                                 |
| `assignment_bitwise`    | `&=` -> `&#124;=`、`&#124;=` -> `&=`、`^=` -> `&=`、`&^=` -> `&#124;=`         |
| `inc_dec`               | `++` -> `--`、`--` -> `++`                                                     |
| `control_flow`          | `switch x` の条件反転                                                          |
| `return`                | `return true` -> `return false`、`return false` -> `return true`               |
| `nil_check`             | `== nil` -> `!= nil`、`!= nil` -> `== nil`                                     |
| `boolean_literal`       | `true` -> `false`、`false` -> `true`                                           |
| `integer_literal`       | `0` -> `1`、0 以外の整数リテラル -> `0`                                        |
| `float_literal`         | `0.0` -> `1.0`、`0.0` 以外の浮動小数点リテラル -> `0.0`                        |
| `rune_literal`          | `'a'` -> `'b'`、`'a'` 以外の rune リテラル -> `'a'`                            |
| `unary_not`             | `!x` -> `x`                                                                    |
| `unary_minus`           | `-x` -> `x`                                                                    |
| `unary_bitwise_not`     | `^x` -> `x`                                                                    |
| `string_literal`        | `""` -> `"mutated"`、空でない文字列リテラル -> `""`                            |

## 前提条件

- mutation 実行前に baseline の `go test` が通ること
- Go 1.26 以上が必要
- `--diff` では git が必要

## テスト

テスト方針は [docs/testing-guidelines.md](docs/testing-guidelines.md) にまとめています。

要点:

- `testify` を使う
- AAA パターンで書く
- `t.Run` を常に使う
- 可読性が明確に上がる場合を除いてテーブル駆動テストは避ける
- テスト関数名は `Test{対象関数名}` を基本にする

## 開発

```bash
go test ./...
```

```bash
make fmt
```

```bash
make lint
```

必要に応じて、次のコマンドで挙動確認できます。

```bash
go run ./cmd/gomut test --package ./sample --jsonl
./gomut test --all
```
