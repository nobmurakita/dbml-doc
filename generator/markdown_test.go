package generator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nobmurakita/dbml-doc/model"
)

func TestGenerateMarkdownBasic(t *testing.T) {
	defaultVal := "now()"
	dbml := &model.DBML{
		Project: &model.Project{
			Name:         "testdb",
			DatabaseType: "PostgreSQL",
			Note:         "テスト用データベース",
		},
		Tables: []model.Table{
			{
				Name: "users",
				Note: "ユーザーテーブル",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true, Note: "ユーザーID"},
					{Name: "name", Type: "varchar(255)", NotNull: true, Unique: true, Note: "ユーザー名"},
					{Name: "created_at", Type: "timestamp", Default: &defaultVal},
				},
				Indexes: []model.Index{
					{
						Columns: []model.IndexColumn{{Name: "name"}},
						Name:    "idx_name",
						Unique:  true,
					},
				},
			},
		},
		Enums: []model.Enum{
			{
				Name: "status",
				Values: []model.EnumValue{
					{Name: "active", Note: "有効"},
					{Name: "inactive", Note: "無効"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := GenerateMarkdown(&buf, dbml, "independent")
	if err != nil {
		t.Fatalf("Markdown生成エラー: %v", err)
	}

	output := buf.String()

	// ヘッダー
	if !strings.Contains(output, "# データベース定義書") {
		t.Error("タイトルが含まれていない")
	}

	// プロジェクト情報
	if !strings.Contains(output, "**プロジェクト:** testdb") {
		t.Error("プロジェクト名が含まれていない")
	}
	if !strings.Contains(output, "**データベース:** PostgreSQL") {
		t.Error("データベース種別が含まれていない")
	}

	// テーブル一覧
	if !strings.Contains(output, "## テーブル一覧") {
		t.Error("テーブル一覧セクションが含まれていない")
	}
	if !strings.Contains(output, "| 1 | [users](#users) | ユーザーテーブル |") {
		t.Error("テーブル一覧の内容が不正")
	}

	// カラム定義
	if !strings.Contains(output, "| 1 | id | integer | NO | - | PK | ユーザーID |") {
		t.Error("カラム定義が不正")
	}
	if !strings.Contains(output, "| 2 | name | varchar(255) | NO | - | PK, UNIQUE |") {
		// nameはNotNull=trueだがPK=false
		if !strings.Contains(output, "| 2 | name | varchar(255) | NO | - | UNIQUE | ユーザー名 |") {
			t.Error("カラム定義（name）が不正")
		}
	}

	// インデックス
	if !strings.Contains(output, "**インデックス:**") {
		t.Error("インデックスセクションが含まれていない")
	}
	if !strings.Contains(output, "| idx_name | name | - | YES |") {
		t.Error("インデックス内容が不正")
	}

	// Enum定義
	if !strings.Contains(output, "## Enum定義") {
		t.Error("Enum定義セクションが含まれていない")
	}
	if !strings.Contains(output, "| active | 有効 |") {
		t.Error("Enum値が不正")
	}
}

func TestGenerateMarkdownWithRefs(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{Name: "orders", Columns: []model.Column{{Name: "id", Type: "integer", PrimaryKey: true}, {Name: "user_id", Type: "integer", NotNull: true}}},
			{Name: "users", Columns: []model.Column{{Name: "id", Type: "integer", PrimaryKey: true}}},
		},
		Refs: []model.Ref{
			{
				From: model.RefEndpoint{Table: "orders", Columns: []string{"user_id"}},
				To:   model.RefEndpoint{Table: "users", Columns: []string{"id"}},
				Type: ">",
			},
		},
	}

	var buf bytes.Buffer
	err := GenerateMarkdown(&buf, dbml, "independent")
	if err != nil {
		t.Fatalf("Markdown生成エラー: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "**リレーション:**") {
		t.Error("リレーションセクションが含まれていない")
	}
	if !strings.Contains(output, "| user_id | users.id | N:1 |") {
		t.Error("リレーション内容が不正")
	}
}

func TestGenerateMarkdownEmpty(t *testing.T) {
	dbml := &model.DBML{}
	var buf bytes.Buffer
	err := GenerateMarkdown(&buf, dbml, "independent")
	if err != nil {
		t.Fatalf("Markdown生成エラー: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# データベース定義書") {
		t.Error("空DBMLでもタイトルは出力されるべき")
	}
}

func TestGenerateMarkdownInlineEnum(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{
				Name: "users",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true},
					{Name: "status", Type: "user_status", NotNull: true, Note: "ユーザー状態"},
					{Name: "role", Type: "user_role", NotNull: true},
				},
			},
		},
		Enums: []model.Enum{
			{
				Name: "user_status",
				Values: []model.EnumValue{
					{Name: "active", Note: "有効"},
					{Name: "inactive", Note: "無効"},
					{Name: "suspended", Note: "停止"},
				},
			},
			{
				Name: "user_role",
				Values: []model.EnumValue{
					{Name: "admin"},
					{Name: "member"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := GenerateMarkdown(&buf, dbml, "inline")
	if err != nil {
		t.Fatalf("Markdown生成エラー: %v", err)
	}

	output := buf.String()

	// Enum定義セクションが出力されないこと
	if strings.Contains(output, "## Enum定義") {
		t.Error("inlineモードではEnum定義セクションは出力されないべき")
	}

	// カラム型がENUM展開されていること（値ごとに<br>で改行）
	if !strings.Contains(output, "ENUM(<br>'active',<br>'inactive',<br>'suspended'<br>)") {
		t.Error("statusカラムの型がENUM展開されていない")
	}

	// Noteを持つEnum値の説明がカラムNoteに追加されていること
	if !strings.Contains(output, "ユーザー状態<br>active=有効, inactive=無効, suspended=停止") {
		t.Error("statusカラムのNoteにEnum説明が追加されていない")
	}

	// Noteを持たないEnum値のカラムは型だけ展開
	if !strings.Contains(output, "ENUM(<br>'admin',<br>'member'<br>)") {
		t.Error("roleカラムの型がENUM展開されていない")
	}
}
