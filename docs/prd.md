# Go Mutation Testing Tool PRD

## タイトル

Go 向け mutation testing ツール

## 要約

Go コードに対して mutation testing を実行し、既存テストが検知できない弱い箇所を可視化する CLI ツールを作る。
主な利用者は人間ではなく AI agent であり、AI agent が自律的に対象選択、実行、結果解析、次アクション決定を行えることを重視する。
初版は実行速度よりもまず「AI agent が弱いテストを機械的に見つけられること」を重視し、`go test` と連携して package 単位・全体・差分対象で実行できるようにする。
設計上は `go-gremlins/gremlins` のカバレッジ駆動・差分実行の強みと、`avito-tech/go-mutesting` の AST ベース変異の強みを組み合わせる。

## 課題

- Go のテストは通っていても、重要な条件分岐や境界値の欠落を見逃していることがある。
- AI agent がテストの強さを定量的・具体的に確認し、次に何を試すべきか判断できる仕組みが不足しがちである。
- 既存の mutation testing ツールは、人間が読む前提の表示や操作が多く、AI agent にとっては実行範囲やレポートの扱いが合わないことがある。
- 既存ツールは高速性と網羅性のどちらかに寄りやすく、AI agent が広い範囲を自律探索する用途にそのまま使いにくい。

## 目的

- mutation testing により、テストで検知できない変更候補を洗い出す。
- 弱いテストや未検知の変異点を、AI agent が機械可読に取得できる形で出力する。
- `go test` と自然に接続できる形で、Go プロジェクトに導入しやすくする。
- 既存ツールの長所を取り込み、AI agent が差分・全体・パッケージ単位のいずれでも扱える実用的な mutation testing 基盤にする。

## 非対象

初版では以下は対象外とする。

- HTML レポートや高度な可視化
- CI の品質ゲートや fail 判定
- 自動修正提案やコード書き換え
- 高度な最適化を前提にした大規模スケール対応
- 変異ロジックの完全網羅を目指すこと

## 対象ユーザー

- AI coding agent
- 自動テスト生成 agent
- CI 連携 agent
- 補助的に結果を確認する人間の開発者

## スコープ

### 初版で含めること

- Go パッケージに対する mutation testing の実行
- `go test` と連携した実行
- AST ベースでの変異生成
- 変異点ごとの実行結果の出力
- `LIVED` な変異点を中心に、弱いテストを機械的に拾える基本レポート
- 機械可読な出力
- 対象範囲の選択
  - 指定パッケージ実行
  - 全体実行
  - 差分対象実行

### 初版の優先順位

1. 指定パッケージに対する mutation 実行
2. AST ベースの代表的 mutation と結果分類
3. `go test` 連携と機械可読出力
4. 全体実行
5. 差分対象実行
6. カバレッジ情報を使った変異スキップ
7. 影響範囲の局所化やその他の高速化

### 初版で含めないこと

- HTML ダッシュボード
- しきい値判定による CI 失敗
- 自動修正や提案フロー
- 複雑なカバレッジ最適化や並列最適化の完成形

## ユーザーフロー

1. ユーザーが対象を選ぶ。
   - リポジトリ全体
   - 指定パッケージ
   - 差分対象
2. ツールは mutation 実行前にベースラインの `go test` が成功することを確認する。
   - ベースラインが失敗している場合は mutation 実行を開始しない。
3. ツールが対象コードを解析し、変異候補を列挙する。
4. 各変異点に対してテストを実行する。
5. ツールが結果を分類する。
   - `KILLED`
   - `LIVED`
   - `NOT COVERED`
   - `TIMED OUT`
   - `NOT VIABLE`
6. ユーザーは、少なくとも `LIVED` の一覧から、弱いテスト箇所を把握する。

## 機能要件

### 1. 対象選択

- リポジトリ全体を対象にできること。
- 任意の package を指定して実行できること。
- Git 差分を対象に実行できること。
- 差分対象は行単位で扱えること。

### 2. 変異生成

- Go のソースコードを解析し、変異候補を列挙できること。
- AST を基準に変異候補を扱えること。
- 代表的な mutation は、以下の優先順位で実装すること。

#### 優先度 P0

- 比較演算子の変更
  - `<` ⇔ `<=`
  - `>` ⇔ `>=`
  - `==` ⇔ `!=`
- 論理演算子の反転
  - `&&` ⇔ `||`
- ガード句の簡易な破壊
  - `if err != nil { return err }` の早期 return を崩す

#### 優先度 P1

- 算術演算子の反転
  - `+` ⇔ `-`
  - `*` ⇔ `/`
  - `%` の扱いを含む簡易な変形

#### 優先度 P2

- 制御フローの一部変更
  - `break` ⇔ `continue`
  - 条件式の反転
- 代入やビット演算の一部変更
  - `+=` ⇔ `-=`
  - `&` ⇔ `|`

#### 優先度 P3

- より攻撃的で解釈コストの高い mutation
  - 空 return
  - より広い構文木の改変
  - 実行結果の意味解釈が難しい破壊的 mutation

#### 対象範囲

- 初版の mutation 対象は通常の Go ソースコードに限定すること。
- テストファイル、自動生成コード、mock は初版の対象外とすること。

### 3. 実行連携

- mutation 実行前にベースラインの `go test` が成功していることを確認できること。
- 変異ごとに `go test` を実行できること。
- 変異によってコンパイル不能になった場合は `NOT VIABLE` として扱えること。
- 実行が長時間停止した場合は `TIMED OUT` として扱えること。
- `TIMED OUT` は `KILLED` と同様に検知成功のシグナルとして扱うこと。

### 4. 結果分類

- 各変異点に対して結果を分類できること。
- 結果は CLI と機械可読出力の両方で取得できること。
- `LIVED` を優先的に抽出しやすい構造であること。
- `NOT COVERED` はカバレッジ判定で事前にスキップした変異を表すこと。
- `TIMED OUT` は変異実行後にテストが規定時間内に完了しなかった変異を表すこと。

### 5. レポート

- 標準出力で概要を確認できること。
- 変異点のファイル、行番号、種類、結果を確認できること。
- 後続の自動処理に使えるよう、機械可読な出力を主要形式として用意すること。
- 機械可読出力は JSON Lines を基本形式とすること。
- 将来の拡張として、単一 JSON ドキュメント形式も扱える余地を残すこと。

#### JSON Lines 出力の最低要件

- フィールド名は安定しており、後方互換性を意識して固定すること。
- 1 行 1 mutation のレコードとして出力すること。
- 実行単位ごとのメタ情報を含めること。
  - 対象範囲
  - 実行時刻
  - 実行コマンド相当の情報
- 集計情報は各レコードに複製して含めること。
- 各変異点の情報を含めること。
  - ファイルパス
  - 行番号
  - mutation 種別
  - 実行結果
  - 失敗時の簡易メッセージ
- AI agent が `LIVED` を抽出しやすい構造であること。
  - 結果フィルタに使えるフィールド名
  - 安定した enum 値
  - 1 行 1 レコードでストリーミング処理しやすいこと。

#### 固定フィールド名

- `target`
  - `mode`
  - `value`
- `started_at`
- `command`
- `summary`
  - `total`
  - `killed`
  - `lived`
  - `not_covered`
  - `timed_out`
  - `not_viable`
- `mutation`
  - `file`
  - `line`
  - `kind`
  - `result`
  - `message`

#### `kind` の固定 enum 値

- `comparison_operator`
- `logical_operator`
- `guard_clause`
- `arithmetic_operator`
- `control_flow`
- `assignment_bitwise`
- `return`

#### JSON Lines 出力例

```json
{
  "target": {
    "mode": "package",
    "value": "./internal/foo"
  },
  "started_at": "2026-07-12T10:00:00+09:00",
  "command": "gomut test --package ./internal/foo",
  "summary": {
    "total": 3,
    "killed": 1,
    "lived": 1,
    "not_covered": 0,
    "timed_out": 0,
    "not_viable": 1
  },
  "mutation": {
    "file": "internal/foo/bar.go",
    "line": 57,
    "kind": "logical_operator",
    "result": "LIVED",
    "message": "tests passed"
  }
}
```

```json
{"target":{"mode":"package","value":"./internal/foo"},"started_at":"2026-07-12T10:00:00+09:00","command":"gomut test --package ./internal/foo","summary":{"total":3,"killed":1,"lived":1,"not_covered":0,"timed_out":0,"not_viable":1},"mutation":{"file":"internal/foo/bar.go","line":57,"kind":"logical_operator","result":"LIVED","message":"tests passed"}}
```

## 制約と依存関係

- Go の構文解析とテスト実行に依存する。
- mutation 実行前に `go test` が通る状態を前提とする。
- Git 差分機能は Git が利用可能な環境を前提とする。
- 初版は大規模最適化よりも、まず正しく動くことを優先する。
- 1 回の実行は数分以内で終わることを初版の目安とする。
- 既存の調査結果では、カバレッジ駆動と差分実行は高速化に有効であり、AST ベース変異は表現力に有効である。

## 成功指標

- AI agent が `LIVED` の変異点を抽出し、次に追加すべきテスト候補を判断できる。
- 全体実行、package 指定、差分対象の 3 種の実行方法が利用できる。
- 少なくとも主要な mutation に対して、結果分類とレポートが得られる。

## リスクと例外

- 変異対象が増えると実行時間が長くなる。
- `go test` の実行結果が環境や依存に左右される。
- AST 変換で意図しないコード変化や `NOT VIABLE` が増える可能性がある。
- 差分対象だけに絞ると、見逃しが増える可能性がある。
- 初版では出力が簡素なため、詳細分析には不十分な可能性がある。

## リリース計画

1. 最小実行版を作る。
   - package 指定と全体実行
   - P0 mutation
   - CLI 出力
2. 差分対象実行を追加する。
3. カバレッジ駆動のスキップを追加する。
4. 機械可読なレポート出力を整える。
5. 必要に応じて最適化や可視化を段階的に追加する。

## 未解決事項

- なし

## 前提

- 主な利用者は AI agent である。
- 人間向けの UI よりも、機械可読な出力と再実行可能な CLI 挙動を優先する。
- 初版は「弱いテストを見つける」ことを最優先にする。
- 実装・最適化・可視化は段階的に増やす。
- `survey.md` にある既存ツール調査を前提に、`gremlins` の高速化方針と `go-mutesting` の変異表現力の両方を意識する。

## 付録

- [survey.md](/home/ubuntu/workspace/github/gomut/survey.md)
- Go ミューテーションテスト調査メモ
- 参考ツール: `go-gremlins/gremlins`, `avito-tech/go-mutesting`
