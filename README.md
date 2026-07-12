# gomut

Go 向けの mutation testing CLI です。主な用途は、AI agent がテストの弱さを機械的に見つけることです。

## Features

- Go パッケージ単位で mutation testing を実行
- `--all` でリポジトリ内の Go パッケージを対象化
- `--diff` で git 差分に含まれる対象を走査
- `--worktree` で一時 git worktree 上で mutation testing を実行
- AST ベースで代表的な mutation 候補を生成
- mutation ごとに `go test` を実行し、結果を分類
- JSON Lines で結果を出力

## Install

```bash
go install ./cmd/gomut
```

## Dev Container

`.devcontainer/devcontainer.json` を使うと、Go 1.26 環境で起動できます。開発用ツールは `make install-dev-tools` で入れます。

開発用ツールが必要なときは `make install-dev-tools` を実行してください。`codex`、`dprint`、`gitleaks` をまとめて入れます。

```bash
make install-dev-tools
```

`~/.codex` 相当のデータは Docker volume に保持します。

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

### Worktree mode

```bash
gomut test --package ./sample --worktree
```

`--worktree` は一時 `git worktree` を作成します。

### JSON Lines output

```bash
gomut test --package ./internal/gomut --jsonl
gomut test --package ./internal/gomut --jsonl mutations.jsonl
```

## Output

`stdout` は JSON Lines の出力先です。`--jsonl` 単体なら stdout に出します。`--jsonl` にファイルパスを指定した場合のみ、そのファイルに書き込みます。

集計サマリや補助メッセージは `stderr` に出します。

主な結果値:

- `KILLED`: 変異後の `go test` が失敗した状態です。既存テストが変異を検出できています。
- `LIVED`: 変異後の `go test` が成功した状態です。テストが変異を検出できていません。
- `NOT COVERED`: baseline のカバレッジでその行が通っていない状態です。変異テスト自体は実行せず、この結果になります。
- `TIMED OUT`: 変異後の `go test` がタイムアウトした状態です。
- `NOT VIABLE`: 変異後のコードが構文エラーや型エラーなどで成立しない状態です。

各レコードには次の情報が入ります。

- `target`: 実行対象
- `summary`: ここまでの集計
- `mutation`: 個別の mutation 結果

`mutation` には `file`、`line`、`kind`、`original`、`replacement`、`result`、`message` が入ります。`message` には `go test` の結果や、実行できなかった理由が入ります。

### Summary の読み方

`summary` は、これまでに処理した mutation の集計です。

- `total`: 処理した mutation の総数
- `killed`: テストが mutation を検出できた数
- `lived`: テストが mutation を検出できなかった数
- `not_covered`: baseline のカバレッジ外だった数
- `timed_out`: mutation 後のテストが時間切れになった数
- `not_viable`: mutation 後のコードが成立しなかった数

基本的には `killed` が多いほど良く、`lived` と `not_covered` が少ないほど良い状態です。`timed_out` と `not_viable` が多い場合は、テストや mutation 候補の見直しが必要です。

## Preconditions

- mutation 実行前に baseline の `go test` が通ること
- Go 1.26 以上
- `--diff` では git が利用可能であること

## Current Scope

現状対応している mutant は以下です。

| 種類                              | 変異内容                                                                          |
| --------------------------------- | --------------------------------------------------------------------------------- |
| 比較演算子                        | `==` -> `!=`                                                                      |
| 比較演算子                        | `!=` -> `==`                                                                      |
| 比較演算子                        | `<` -> `<=`                                                                       |
| 比較演算子                        | `>` -> `>=`                                                                       |
| 比較演算子                        | `<=` -> `<`                                                                       |
| 比較演算子                        | `>=` -> `>`                                                                       |
| 論理演算子                        | `&&` -> `                                                                         |
| 論理演算子                        | `                                                                                 |
| 算術演算子                        | `+` -> `-`                                                                        |
| 算術演算子                        | `-` -> `+`                                                                        |
| 算術演算子                        | `*` -> `/`                                                                        |
| 算術演算子                        | `/` -> `*`                                                                        |
| 算術演算子                        | `%` -> `*`                                                                        |
| bitwise 演算子                    | `&` -> `\|`                                                                       |
| bitwise 演算子                    | `\|` -> `&`                                                                       |
| bitwise 演算子                    | `^` -> `&`                                                                        |
| bitwise 演算子                    | `&^` -> `\|`                                                                      |
| shift 演算子                      | `<<` -> `>>`                                                                      |
| shift 演算子                      | `>>` -> `<<`                                                                      |
| 代入演算子                        | `+=` -> `-=`                                                                      |
| 代入演算子                        | `-=` -> `+=`                                                                      |
| 代入演算子                        | `*=` -> `/=`                                                                      |
| 代入演算子                        | `/=` -> `*=`                                                                      |
| 代入演算子                        | `%=` -> `*=`                                                                      |
| 代入演算子                        | `&=` -> `                                                                         |
| 代入演算子                        | `                                                                                 |
| 代入演算子                        | `^=` -> `&=`                                                                      |
| 代入演算子                        | `&^=` -> `                                                                        |
| 代入演算子                        | `<<=` -> `>>=`                                                                    |
| 代入演算子                        | `>>=` -> `<<=`                                                                    |
| インクリメント/デクリメント       | `++` -> `--`                                                                      |
| インクリメント/デクリメント       | `--` -> `++`                                                                      |
| return                            | `return true` -> `return false`                                                   |
| return                            | `return false` -> `return true`                                                   |
| nil チェック                      | `!= nil` -> `== nil`                                                              |
| nil チェック                      | `== nil` -> `!= nil`                                                              |
| boolean literal                   | `true` -> `false`                                                                 |
| boolean literal                   | `false` -> `true`                                                                 |
| integer literal                   | `0` -> `1`                                                                        |
| integer literal                   | `0` 以外の整数リテラル -> `0`                                                     |
| float literal                     | `0.0` -> `1.0`                                                                    |
| float literal                     | `0.0` 以外の浮動小数点リテラル -> `0.0`                                           |
| rune literal                      | `'a'` -> `'b'`                                                                    |
| rune literal                      | `'a'` 以外の rune リテラル -> `'a'`                                               |
| unary not                         | `!x` -> `x`                                                                       |
| unary minus                       | `-x` -> `x`                                                                       |
| unary bitwise not                 | `^x` -> `x`                                                                       |
| switch condition                  | `switch x` の `x` を `!x` に反転                                                  |
| string literal                    | `""` -> `"mutated"`                                                               |
| string literal                    | 空でない文字列リテラル -> `""`                                                    |
| guard clause の単純な return 変異 | `return x` の `x` を `nil` 以外の単純な識別子として扱い、別の return 値に差し替え |

未実装のものは今後追加できます。

## Planned Extensions

- 追加 mutation 種別
  - 代入演算子の反転
  - 境界値比較の強化
  - `if` / `switch` の条件反転
  - `return` 値の定数化
  - `nil` チェックの反転
  - ブール値の固定化
  - 算術式の一部削除
- `NOT COVERED` 判定の精度向上
- `diff` モードの対象解決をより厳密にする
- 並列実行の導入
- レポート出力形式の拡張
- CI 連携用のしきい値判定
- HTML などの可視化出力
- 自動修正や提案フロー

## Testing

テスト方針は [`docs/testing-guidelines.md`](docs/testing-guidelines.md) にまとめています。

要点:

- `testify` を使う
- AAA パターンで書く
- `t.Run` を常に使う
- テーブル駆動テストは原則使わない
- テスト関数名は `Test{対象関数名}` を基本にする

## Development

```bash
go test ./...
```
