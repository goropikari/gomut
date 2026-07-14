# gomut mutation kind selection の `mode` / `enable` / `disable` 対応 PRD

## 1. タイトル

mutation kind の選択を `mode` ベースにし、`enable` と `disable` を併用できるようにする

## 2. 要約

`gomut` の mutation kind 指定は、現状の「有効にしたい kind を列挙する」方式から、`golangci-lint` に近い `mode` ベースの設定へ拡張する。

これにより、利用者は `standard` を既定にしつつ、必要な kind だけを追加で有効化したり、不要な kind を明示的に無効化できる。
CLI と config file の両方で指定可能にし、CLI が config を上書きする。

## 3. 課題

`gomut` で扱える mutation kind が増えるほど、従来の「有効な kind を全部列挙する」設定は保守しづらくなる。

- 新しい kind が追加されるたびに config を見直す必要がある
- 一部の kind だけを除外したい場合でも、残りをすべて列挙する必要がある
- 利用者ごとに「普段使う標準セット」と「実験的または重い kind」を分けたい

## 4. 目的

- `mode` により kind の既定集合を切り替えられるようにする
- `enable` と `disable` を併用できるようにする
- `disable` を優先して、不要な kind を簡単に外せるようにする
- config file と CLI の両方で同じ考え方の指定ができるようにする

## 5. 非対象

- kind ごとの重み付けや優先順位付け
- kind ごとの実行回数やサンプリング
- ファイル単位・ディレクトリ単位での kind 除外
- mutation result type のフィルタリング変更
- `--diff` や exclusion ルールの仕様変更

## 6. 対象ユーザー

- `gomut` を継続運用するチーム利用者
- config file を管理し、CI で mutation testing を回す利用者
- kind が増えてきたため、明示的な列挙ではなく「標準からの差分」で設定したい利用者

## 7. スコープ

### 初版で扱うこと

- mutation kind の選択に `mode` を導入する
- `mode` は少なくとも `standard` と `all` を持つ
- `enable` と `disable` を config file と CLI の両方で指定できる
- `disable` が `enable` より優先される
- CLI は config file を上書きする
- `mode` の既定値は `standard`
- 既存の `--kind` フラグは廃止し、CLI は `--kind-mode` / `--kind-enable` / `--kind-disable` に分割する

### `standard` の考え方

`standard` は、普段の mutation testing で価値が高く、設定の負担が少ない基本セットとする。

初版の `standard` は次で確定する。

- `comparison_operator`
- `logical_operator`
- `arithmetic_operator`
- `guard_clause`
- `return`
- `nil_check`

### `all` の考え方

`all` は `gomut` が現在サポートする mutation kind をすべて含む。

## 8. ユーザーフロー

1. 利用者が config file に `kind.mode: standard` を書く
2. 必要に応じて `enable` で追加 kind を入れる
3. 不要な kind があれば `disable` で外す
4. CLI で同じ項目を指定した場合は CLI 側が優先される
5. `gomut test` は最終的な kind 集合だけを使って candidate discovery を行う

## 9. 機能要件

### 設定モデル

- `kind` は単一の列挙リストではなく、`mode` / `enable` / `disable` を持つ設定として扱う
- `mode` は `standard` または `all` を受け付ける
- `enable` は追加で有効にしたい kind の配列とする
- `disable` は無効化したい kind の配列とする
- `disable` は `enable` と `mode` より優先される

### 優先順位

- CLI の `mode` / `enable` / `disable` は config file を上書きする
- config file に指定がない場合は `mode: standard` を既定として扱う
- `mode: all` の場合でも `disable` で個別に外せる

### 入力仕様

- kind の名称は既存の mutation kind 名を使う
- 複数指定は配列で行う
- 既存のような「列挙だけで全体を定義する」使い方は、`mode: all` と `disable` の組み合わせで代替できる

### 期待される挙動

- `mode: standard` では基本セットのみを対象にする
- `mode: standard` + `enable` で基本セットに追加できる
- `mode: all` + `disable` で不要な kind を落とせる
- `enable` と `disable` に同じ kind が入っていた場合は `disable` を優先する

## 10. 制約と依存関係

- 既存の config 読み込み処理と CLI 引数解決に依存する
- 既存の mutation kind 定義と整合している必要がある
- `standard` の構成は将来変更されうるため、利用者が期待値を誤解しない説明が必要
- `mode` / `enable` / `disable` の組み合わせが複雑になりすぎないよう、エラー文言は明確である必要がある

## 11. 成功指標

- `gomut` 利用者が「不要な kind を外す」ために全 kind の列挙をしなくてよくなる
- config の更新回数が、kind 追加のたびに必須ではなくなる
- `standard` 運用のまま、必要時だけ `disable` を足す運用が定着する
- CLI と config の両方で同じ考え方が使える

## 12. リスクと例外

- `standard` の定義が曖昧だと、利用者の期待と実装がずれる
- `enable` / `disable` / `mode` の組み合わせが増えると、設定の挙動が理解しづらくなる
- 既存の `kind` の列挙指定との互換性をどう保つかで混乱が起きうる
- 新しい kind 追加時に `standard` への採否判断が必要になる

## 13. リリース計画

1. 設定モデルと優先順位を実装する
2. config file と CLI の両方で読み取れるようにする
3. `standard` と `all` の動作をテストで固定する
4. README と config 例を更新する
5. 既存ユーザーが移行しやすいように、旧来の列挙方式の扱いを明記する

## 14. 未解決事項

## 15. 前提

- `gomut` の利用者は Go プロジェクトの設定ファイルを継続管理するチームである
- mutation kind は今後も増える前提である
- `mode: standard` を既定にすることで、設定の記述量を減らしたい
- CLI は config より優先される
- 旧来の `kind:` 配列設定や `--kind` フラグへの後方互換は不要である

## 16. 付録

### 用語

- `mode`: kind 集合の基準となる既定セット
- `enable`: 追加で有効にする kind
- `disable`: 明示的に無効化する kind

### 参考イメージ

```yaml
kind:
  mode: standard
  enable:
    - bitwise_operator
  disable:
    - guard_clause
```

```yaml
kind:
  mode: all
  disable:
    - string_literal
    - float_literal
```
