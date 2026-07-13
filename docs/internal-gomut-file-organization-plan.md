# internal/gomut ファイル構成整理計画

## 目的

`internal/gomut` の責務をファイル単位で読み取りやすくする。

現状は処理の流れ自体は成立しているが、CLI 設定、実行制御、candidate 探索、mutation 実行、出力補助が `internal/gomut` 直下に横並びで置かれており、初見でどこから読めばよいか分かりにくい。

この整理では package を増やすことを主目的にしない。未exportのまま協調できている内部関数を無理に export せず、責務ごとのファイル配置で読み順を明確にする。

## 方針

- `internal/gomut` はこれ以上細かい package に分けない。
- ファイルは細かく分けすぎず、責務単位でまとめる。
- prefix でファイルを量産しない。例: `command_config.go`, `command_args.go` のような分け方は避ける。
- 入口から実行までの読み順を固定できる構成にする。
- 既存の `result`, `report` は独立責務として維持する。
- `execution` は、流れの読みやすさを優先するなら `internal/gomut/execution.go` へ戻す候補にする。

## 目標構成

```text
internal/gomut/
  command.go    // CLI: cobra, args, flags, Command.Run
  config.go     // .gomut.yaml と flags を実行設定に解決する
  runner.go     // mutation testing 全体の流れを制御する

  discovery.go  // 対象 package/file を決め、mutation candidate を集める
  rules.go      // AST から mutation candidate を作るルール
  execution.go  // 1 mutation の適用、go test 実行、結果分類

  output.go     // JSONL/HTML/summary/progress への出力制御
  isolation.go  // 一時コピー、mutation ごとの isolated root
  diff.go       // git diff と diff 行判定
  exclusion.go  // exclude pattern と ignore comment
  util.go       // まだ独立責務にするほどでない補助
```

既存サブパッケージ:

```text
internal/gomut/result/      // 共有する結果型と filter
internal/gomut/report/      // HTML report
internal/gomut/integration/ // 結合テスト
```

## 読む順番

```text
cmd/gomut/main.go
-> internal/gomut/command.go
-> internal/gomut/config.go
-> internal/gomut/runner.go
-> internal/gomut/discovery.go
-> internal/gomut/rules.go
-> internal/gomut/execution.go
-> internal/gomut/output.go
```

## ファイルごとの責務

### command.go

CLI の配線だけを扱う。

- `Command`
- `NewCommand`
- `Command.Run`
- cobra root/test command の定義
- test command の flag 定義
- `runTest`
- `NormalizeTestArgs`

`runTest` は `config.go` で `RunConfig` を作り、`runner.go` に渡すだけにする。

### config.go

設定を扱う。

- `.gomut.yaml` の型
- `DefaultConfigPath`
- `LoadConfig`
- `RunConfig`
- flag と YAML の優先順位解決
- timeout, parallel, progress, output, type filter の解決
- target mode の解決

`Config` と `RunConfig` は粒度が違うだけでどちらも設定なので、同じ `config.go` にまとめる。

### runner.go

mutation testing 全体の流れを制御する。

- 作業 root の準備
- candidate discovery の呼び出し
- exclusion notice の報告
- mutation execution の呼び出し
- 出力の呼び出し

低レベルな出力処理、baseline 実行、package 解決、coverage parsing はここに置きすぎない。

### discovery.go

対象 package/file の決定から candidate 収集までを扱う。

- `resolvePackages`
- package の Go ファイル列挙
- baseline coverage を使った candidate discovery
- exclude filter を使った候補除外
- `DiscoverCandidates`
- `DiscoverCandidatesWithExclusions`

`rules.go` は AST node から candidate を作る責務に限定し、探索の外枠は `discovery.go` に置く。

### rules.go

AST から mutation candidate を作るルールを扱う。

- `mutationCandidateFromNode`
- 比較演算子
- 論理演算子
- 算術演算子
- bitwise/shift/assignment
- literal
- control flow
- guard clause の単純な return mutation

初期段階では 1 ファイルにまとめる。大きくなってから `operators.go`, `literals.go`, `control.go` へ分ける。

### execution.go

mutation candidate を実際に評価する責務を扱う。

- sequential/parallel execution loop
- 1 mutation の適用
- `go test` 実行
- timeout 判定
- killed/lived/not viable/not covered の分類
- record 生成
- result filter 適用

現状の `internal/gomut/execution/execution.go` は責務としては成立している。ただし「流れを追う」観点では package を跨ぐため、`internal/gomut/execution.go` に戻す選択肢がある。

### output.go

出力制御を扱う。

- JSONL writer の決定
- HTML writer の決定
- HTML report 書き出し
- summary 表示
- progress reporter の開始/終了

HTML のレンダリング本体は既存どおり `internal/gomut/report` に残す。

### isolation.go

実行環境の隔離を扱う。

- run root の一時コピー
- mutation ごとの isolated root 作成
- repository tree copy

### diff.go

git diff と diff 行判定を扱う。

- diff range normalization
- `git diff` 実行
- hunk parsing
- diff 対象行判定

将来的には package global な `diffState` を減らし、diff 対象行情報を discovery へ明示的に渡す形を検討する。

### exclusion.go

除外ルールを扱う。

- exclude pattern
- file-level skip
- candidate-level skip
- ignore comment parsing
- exclusion notice

### util.go

まだ独立責務にするほどでない補助だけを置く。

候補:

- `goCommandEnv`
- small path helpers
- small file helpers

大きくなった補助は、責務が見えた時点で対象ファイルへ移す。

## 実装手順

### 1. command.go を薄くする

- `RunConfig` と `testRunInputs` を `config.go` へ移す。
- `buildTestRunConfig` 周辺を `config.go` へ移す。
- `ResolveTarget` を `config.go` へ移すか、target 解決が大きくなった時点で `target.go` を検討する。
- `NormalizeTestArgs` は CLI の特殊処理なので `command.go` に残す。

完了条件:

- `command.go` は cobra 配線と `runTest` が中心になっている。
- `runTest` から実行本体へ進む流れがすぐ読める。

### 2. runner.go から出力処理を分離する

- `openCandidateOutputs`
- `openJSONLOutput`
- `openHTMLOutput`
- `writeCandidateHTML`
- `printSummary`
- `openOutput`
- `chainCleanup`

これらを `output.go` へ移す。

完了条件:

- `runner.go` は orchestration が中心になっている。
- stdout/stderr/JSONL/HTML の詳細は `output.go` にまとまっている。

### 3. runner.go から discovery 周辺を分離する

- `resolvePackages`
- `runBaseline`
- `mergeCoverage`
- `DiscoverCandidates` 周辺の orchestration

これらを `discovery.go` へ寄せる。

coverage profile parsing が `util.go` に残っている場合は、`runBaseline` と一緒に `discovery.go` へ寄せるか、責務が十分大きければ `coverage.go` を検討する。ただし初期整理ではファイル数を増やしすぎない。

完了条件:

- `runner.go` は `discoverCandidates` を呼ぶだけで詳細に踏み込まない。
- candidate discovery の入口は `discovery.go` を読めば分かる。

### 4. execution の置き場所を判断する

選択肢 A: 現状維持

- `internal/gomut/execution/execution.go` を維持する。
- 並列実行の独立性を優先する。

選択肢 B: gomut 直下へ戻す

- `internal/gomut/execution/execution.go` を `internal/gomut/execution.go` へ戻す。
- `runner.go` からの読みやすさを優先する。
- package 境界のために作っている adapter/interface が減らせるか確認する。

今回の目的が「流れを読みやすくする」なので、B を第一候補にする。ただし差分が大きくなりすぎる場合は A のままにする。

完了条件:

- execution loop と 1 mutation 実行の関係が追いやすい。
- 不要な export/interface が増えていない。

### 5. rules.go は現状維持し、入口だけ分かりやすくする

- `mutationCandidateFromNode` を rules の入口として明確に保つ。
- ルールの順序を mutation kind のまとまりで整える。
- 今回は `operators.go` などへ分けない。

完了条件:

- `rules.go` を開けば mutation rule 全体を確認できる。
- 新しい mutation kind を追加する場所が分かる。

### 6. util.go を掃除する

- package 解決、coverage parsing、file helper が混ざっている場合、責務ファイルへ移す。
- まだ分類しにくい小さい helper だけを `util.go` に残す。

完了条件:

- `util.go` が主要処理の隠れ場所になっていない。
- 各 helper の利用先と責務が近い。

## テスト方針

ファイル移動が中心なので、既存テストを落とさないことを優先する。

実行する確認:

```sh
make fmt
make lint
go test ./...
```

必要に応じて CLI の確認:

```sh
go run ./cmd/gomut test ./sample --jsonl
```

## 完了条件

- `internal/gomut` の package 数を増やしていない。
- `command.go -> config.go -> runner.go` の入口が明確になっている。
- `runner.go` が orchestration に寄っている。
- 出力処理が `output.go` にまとまっている。
- candidate discovery は `discovery.go`、mutation rule は `rules.go` を読めば分かる。
- `util.go` に主要責務が隠れていない。
- `make fmt`, `make lint`, `go test ./...` が通る。
