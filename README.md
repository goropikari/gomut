# gomut

Go 向けの mutation testing CLI です。主な用途は、AI agent がテストの弱さを機械的に見つけることです。

## Features

- Go パッケージ単位で mutation testing を実行
- `--all` でリポジトリ内の Go パッケージを対象化
- `--diff` で git 差分に含まれる対象を走査
- AST ベースで代表的な mutation 候補を生成
- mutation ごとに `go test` を実行し、結果を分類
- JSON Lines で結果を出力

## Install

```bash
go build -o gomut ./cmd/gomut
```

## Usage

### Package mode

```bash
./gomut test --package ./internal/gomut
```

### All packages

```bash
./gomut test --all
```

### Diff mode

```bash
./gomut test --diff HEAD~1..HEAD
```

### JSON Lines output

```bash
./gomut test --package ./internal/gomut --jsonl mutations.jsonl
```

## Output

`stdout` は JSON Lines の出力先です。`--jsonl` にファイルパスを指定した場合のみ、そのファイルに書き込みます。

集計サマリや補助メッセージは `stderr` に出します。

主な結果値:

- `KILLED`
- `LIVED`
- `NOT COVERED`
- `TIMED OUT`
- `NOT VIABLE`

`LIVED` を拾いやすいように、各レコードには対象情報、集計情報、mutation 情報を含めています。

## Preconditions

- mutation 実行前に baseline の `go test` が通ること
- Go 1.22 以上
- `--diff` では git が利用可能であること

## Current Scope

初版で扱う mutation は以下です。

- 比較演算子
- 論理演算子
- 算術演算子
- guard clause の単純な return 変異

未実装のものは今後追加できます。

## Planned Extensions

- `NOT COVERED` 判定の精度向上
- `diff` モードの対象解決をより厳密にする
- 追加 mutation 種別の実装
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
