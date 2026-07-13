# AGENTS.md

## Purpose

`gomut` は Go 向けの mutation testing CLI です。エージェント作業では、主に次を前提に扱います。

- `--package` で単一パッケージを対象化できる
- `--all` でリポジトリ内の Go パッケージを対象化できる
- `--diff` で git 差分に含まれる対象を走査できる
- mutation ごとに `go test` を実行し、結果を分類する
- 結果は JSON Lines で出力できる

## Output Contract

- `stdout` は JSON Lines の出力先
- `--jsonl` 単体なら stdout に出力する
- `--jsonl` にファイルパスを指定した場合のみ、そのファイルに書き込む
- 集計サマリや補助メッセージは `stderr` に出す
- JSONL の `mutation` には少なくとも次を含める
  - `file`
  - `line`
  - `kind`
  - `original`
  - `replacement`
  - `result`
  - `message`
- `LIVED` を拾いやすいように、各レコードには対象情報と集計情報も含める

## Preconditions

- mutation 実行前に baseline の `go test` が通ること
- Go 1.26 以上を前提にする
- `--diff` では git が利用可能であること

## Current Scope

初版で扱う mutation は次の範囲。

- 比較演算子
- 論理演算子
- 算術演算子
- guard clause の単純な return 変異

未実装のものは将来追加できる。

## Testing

テスト方針の要点。

- `testify` を使う
- AAA パターンで書く
- `t.Run` を常に使う
- テーブル駆動テストは原則使わない
- テスト関数名は `Test{対象関数名}` を基本にする

## Development

- 全体テスト: `go test ./...`
- format: `make fmt`
- lint: `make lint`
- コードを編集したら `make fmt` と `make lint` でエラーが出ないことも確認する
- 変更後は、必要なら `go run ./cmd/gomut test --package ./sample --jsonl` や `./gomut test --all` で挙動確認する

## Pull Requests

- PR を作るときは `.github/pull_request_template.md` に必ず従う
- PR 本文は GitHub Markdown として正しい記法で書く
