# dbml-doc

DBMLファイルからテーブル定義書（Markdown / Excel）を生成するCLIツール。

## インストール

```bash
go install github.com/nobmurakita/dbml-doc@latest
```

または、リポジトリをクローンしてビルド:

```bash
git clone https://github.com/nobmurakita/dbml-doc.git
cd dbml-doc
go build -o dbml-doc .
```

## 使い方

```bash
# Markdown出力（stdout）
dbml-doc -i schema.dbml

# Markdown出力（ファイル）
dbml-doc -i schema.dbml -o schema.md

# Excel出力
dbml-doc -i schema.dbml -f excel -o schema.xlsx

# Enum表示をMySQL風のインライン展開にする
dbml-doc -i schema.dbml -e inline
```

### オプション

| オプション | 説明 | デフォルト |
|-----------|------|----------|
| `-i` | 入力DBMLファイル（必須） | - |
| `-f` | 出力形式: `markdown` \| `excel` | `markdown` |
| `-o` | 出力ファイルパス | stdout（Markdown）/ `output.xlsx`（Excel） |
| `-e` | Enum表示モード: `independent` \| `inline` | `independent` |

## 出力例

### Markdown

```markdown
# データベース定義書

**プロジェクト:** ecommerce

**データベース:** PostgreSQL

## テーブル一覧

| # | テーブル名 | 説明 |
|---|-----------|------|
| 1 | [users](#users) | ユーザー情報を管理するテーブル |
| 2 | [products](#products) | 商品情報を管理するテーブル |

## テーブル定義

### users

ユーザー情報を管理するテーブル

| # | カラム名 | 型 | NULL | デフォルト | 制約 | 説明 |
|---|---------|-----|------|----------|------|------|
| 1 | id | integer | NO | - | PK, AUTO INCREMENT | ユーザーID |
| 2 | username | varchar(255) | NO | - | UNIQUE | ユーザー名 |
| 3 | email | varchar(255) | NO | - | UNIQUE | メールアドレス |
```

### Excel

- **テーブル一覧シート**: 全テーブルの一覧
- **Enum定義シート**: Enum一覧（`independent`モード時のみ）
- **テーブルごとのシート**: カラム定義、インデックス、リレーション

ヘッダー行はスタイリング済み（背景色・罫線・列幅自動調整）。

### Enum表示モード

| モード | 説明 |
|--------|------|
| `independent`（デフォルト） | Enum定義を独立セクション/シートとして出力（PostgreSQL的） |
| `inline` | カラム型を `ENUM('val1','val2',...)` に展開し、説明にEnum値の意味を付加。独立Enum定義は出力しない（MySQL的） |

## 対応するDBML構文

- `Project` / `Table` / `Enum` / `Ref` / `TableGroup`
- スキーマ付きテーブル名（`public.users`）
- テーブルエイリアス（`Table users as U`）
- カラム設定: `pk`, `not null`, `unique`, `increment`, `default`, `note`
- インラインRef（`ref: > users.id`）
- インデックス（単一・複合・式）
- Ref設定（`delete: cascade`, `update: no action` 等）
- コメント: `//` 行コメント、`/* */` ブロックコメント
- 文字列: シングルクォート、ダブルクォート、三重クォート

## 開発

```bash
# テスト実行
go test ./...

# サンプルDBMLで動作確認
go run . -i sample.dbml -f markdown
go run . -i sample.dbml -f excel -o output.xlsx
```
