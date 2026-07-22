# gomut architecture

`gomut` は Go 向けの mutation testing CLI です。

この文書は、`internal/gomut` の処理フローと責務分割を、コードを読む前に把握できるように整理したものです。

## 全体像

```text
cmd/gomut/main.go
-> internal/gomut/command.go
-> internal/gomut/config.go
-> internal/gomut/runner.go
-> internal/gomut/discovery.go
-> internal/gomut/rules.go
-> internal/gomut/executor.go
-> internal/gomut/output.go
```

処理の流れは次の通りです。

1. CLI が root command で対象 package、または `diff` subcommand で差分範囲を受ける
2. `command.go` が flag と引数を解釈する
3. `config.go` が YAML と flag から `RunConfig` を作る
4. `runner.go` が実行全体を組み立てる
5. `discovery.go` と `rules.go` が mutation candidate を集める
6. `executor.go` が candidate 群を sequential または parallel で回す
7. `runner.go` の mutation 実行処理が 1 mutation を適用して `go test` を実行する
8. `output.go` が JSONL / HTML / SARIF / summary を出力する

## レイヤ構造

### 1. CLI layer

- `command.go`
  - cobra の配線
  - root command の flag 定義
  - `Command.Run`
  - `NormalizeTestArgs`

この層は、入力を受け取って次の層に渡すだけにする。

### 2. Configuration layer

- `config.go`
  - `.gomut.yaml` の読み込み
  - flag と設定ファイルの解決
  - `RunConfig` の生成

この層は「何を実行するか」を確定する。

### 3. Orchestration layer

- `runner.go`
  - mutation testing 全体の orchestration
  - run root の準備
  - candidate discovery の呼び出し
  - 出力先の準備
  - 実行結果の最終出力

この層は「どの順で何をやるか」を決める。

### 4. Discovery layer

- `discovery.go`
  - 対象 package / file の決定
  - mutation candidate の収集
- `rules.go`
  - AST から mutation candidate を作るルール

この層は「どのコードを mutation するか」を決める。

### 5. Execution layer

- `executor.go`
  - candidate 群の実行制御
  - sequential / parallel の切り替え
  - covered / not covered の判定
  - summary / record / JSONL の生成
  - result filter の適用
- `runner.go`
  - 1 mutation の適用
  - `go test` の実行
  - timeout 判定
  - `lived / killed / not viable / timed out` の分類

この層は「mutation をどう実行し、どう結果化するか」を担当する。

### 6. Output layer

- `output.go`
  - JSONL / HTML / summary / progress の出力制御
- `report/`
  - HTML / SARIF のレンダリング

この層は「結果をどう見せるか」を担当する。

### 7. Support layer

- `isolation.go`
  - run root の一時コピー
  - mutation ごとの isolated root 作成
- `diff.go`
  - git diff と diff 行判定
- `exclusion.go`
  - exclude ルールと ignore comment
- `util.go`
  - 小さな共通 helper

この層は、上位レイヤから使われる補助機能をまとめる。

## 責務の境界

### `runner.go` と `executor.go`

- `runner.go` は「何をどの順でやるか」を決める
- `executor.go` は「candidate をどう回すか」を決める
- `runner.go` は 1 mutation の低レベル実行も持つ

`runner.go` に全体制御があり、`executor.go` に candidate 実行ループがあります。

### `executor.go` と `runner.go`

- `executor.go` は candidate 群を管理する
- `runner.go` は 1 mutation の実行本体を持つ

`executor.go` はループと集計、`runner.go` は個別 mutation の副作用を担当します。

## 出力方針

- `stdout` は JSON Lines の出力先
- `--jsonl` 単体なら stdout に出す
- `--jsonl` にファイルパスがある場合だけ、そのファイルに書く
- summary や補助メッセージは `stderr` に出す

## 実行前提

- mutation 実行前に baseline の `go test` が通ること
- Go 1.26 以上を前提にする
- `diff` では git が利用可能であること

## 今後の拡張余地

- mutation kind の追加は `rules.go` 側で増やす
- 出力形式の追加は `output.go` と `report/` 側で増やす
- 実行戦略の変更は `executor.go` 側で吸収する
