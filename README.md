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
# Markdown出力（ディレクトリに複数ファイル生成）
dbml-doc -i schema.dbml -o docs

# Excel出力
dbml-doc -i schema.dbml -f excel -o schema.xlsx

# Enum表示をMySQL風のインライン展開にする
dbml-doc -i schema.dbml -e inline -o docs
```

### オプション

| オプション | 説明 | デフォルト |
|-----------|------|----------|
| `-i` | 入力DBMLファイル（必須） | - |
| `-f` | 出力形式: `markdown` \| `excel` | `markdown` |
| `-o` | 出力先（Markdown: ディレクトリ、Excel: ファイルパス） | `output`（Markdown）/ `output.xlsx`（Excel） |
| `-e` | Enum表示モード: `independent` \| `inline` | `independent` |

## 出力例

### Markdown

`-o docs` を指定すると以下のディレクトリ構成で出力されます:

```
docs/
  index.md           -- 目次（プロジェクト情報 + テーブル一覧 + Enum一覧リンク）
  enums.md           -- Enum定義（independentモード時のみ）
  tables/
    users.md         -- テーブルごとの定義
    products.md
    ...
```

各テーブルページにはカラム定義・インデックス・リレーション（参照先・参照元、各テーブルへのリンク付き）が含まれます。

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
go run . -i sample.dbml -o output
go run . -i sample.dbml -f excel -o output.xlsx
```
