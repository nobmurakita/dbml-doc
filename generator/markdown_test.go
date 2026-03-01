package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nobmurakita/dbml-doc/model"
)

func TestTableToFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "users.md"},
		{"public.users", "public_users.md"},
		{"Public.Users", "public_users.md"},
		{"schema.table_name", "schema_table_name.md"},
	}
	for _, tt := range tests {
		got := tableToFileName(tt.input)
		if got != tt.want {
			t.Errorf("tableToFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateMarkdownPages(t *testing.T) {
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
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true},
					{Name: "name", Type: "varchar(255)", NotNull: true},
					{Name: "created_at", Type: "timestamp", Default: &defaultVal},
				},
			},
			{
				Name: "orders",
				Note: "注文テーブル",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true},
					{Name: "user_id", Type: "integer", NotNull: true},
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
		Refs: []model.Ref{
			{
				From: model.RefEndpoint{Table: "orders", Columns: []string{"user_id"}},
				To:   model.RefEndpoint{Table: "users", Columns: []string{"id"}},
				Type: ">",
			},
		},
	}

	dir := t.TempDir()
	err := GenerateMarkdownPages(dir, dbml, "independent")
	if err != nil {
		t.Fatalf("GenerateMarkdownPages失敗: %v", err)
	}

	// index.md の検証
	indexData, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatalf("index.md読み込み失敗: %v", err)
	}
	index := string(indexData)
	if !strings.Contains(index, "# データベース定義書") {
		t.Error("index.md: タイトルが含まれていない")
	}
	if !strings.Contains(index, "**プロジェクト:** testdb") {
		t.Error("index.md: プロジェクト名が含まれていない")
	}
	if !strings.Contains(index, "[users](tables/users.md)") {
		t.Error("index.md: usersテーブルへのリンクが含まれていない")
	}
	if !strings.Contains(index, "[orders](tables/orders.md)") {
		t.Error("index.md: ordersテーブルへのリンクが含まれていない")
	}
	if !strings.Contains(index, "[Enum定義一覧](enums.md)") {
		t.Error("index.md: Enum定義へのリンクが含まれていない")
	}

	// enums.md の検証
	enumsData, err := os.ReadFile(filepath.Join(dir, "enums.md"))
	if err != nil {
		t.Fatalf("enums.md読み込み失敗: %v", err)
	}
	enums := string(enumsData)
	if !strings.Contains(enums, "[< 目次に戻る](index.md)") {
		t.Error("enums.md: ナビゲーションリンクが含まれていない")
	}
	if !strings.Contains(enums, "### status") {
		t.Error("enums.md: Enum名が含まれていない")
	}
	if !strings.Contains(enums, "| active | 有効 |") {
		t.Error("enums.md: Enum値が含まれていない")
	}

	// tables/users.md の検証
	usersData, err := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	if err != nil {
		t.Fatalf("tables/users.md読み込み失敗: %v", err)
	}
	users := string(usersData)
	if !strings.Contains(users, "[< 目次に戻る](../index.md)") {
		t.Error("users.md: ナビゲーションリンクが含まれていない")
	}
	if !strings.Contains(users, "## users") {
		t.Error("users.md: テーブル名見出しが含まれていない")
	}
	if !strings.Contains(users, "| 1 | id | integer | NO | - | PK |") {
		t.Error("users.md: カラム定義が含まれていない")
	}
	// users.md: 参照元リレーション（ordersから参照されている）
	if !strings.Contains(users, "**リレーション（参照元）:**") {
		t.Error("users.md: 参照元リレーションセクションが含まれていない")
	}
	if !strings.Contains(users, "[orders.user_id](orders.md)") {
		t.Error("users.md: 参照元リレーションのリンクが含まれていない")
	}

	// tables/orders.md の検証（リレーションリンク付き）
	ordersData, err := os.ReadFile(filepath.Join(dir, "tables", "orders.md"))
	if err != nil {
		t.Fatalf("tables/orders.md読み込み失敗: %v", err)
	}
	orders := string(ordersData)
	if !strings.Contains(orders, "[< 目次に戻る](../index.md)") {
		t.Error("orders.md: ナビゲーションリンクが含まれていない")
	}
	if !strings.Contains(orders, "[users.id](users.md)") {
		t.Error("orders.md: リレーションの参照先リンクが含まれていない")
	}
}

func TestGenerateMarkdownPagesInlineEnum(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{
				Name: "users",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true},
					{Name: "status", Type: "user_status", NotNull: true, Note: "ユーザー状態"},
				},
			},
		},
		Enums: []model.Enum{
			{
				Name: "user_status",
				Values: []model.EnumValue{
					{Name: "active", Note: "有効"},
					{Name: "inactive", Note: "無効"},
				},
			},
		},
	}

	dir := t.TempDir()
	err := GenerateMarkdownPages(dir, dbml, "inline")
	if err != nil {
		t.Fatalf("GenerateMarkdownPages失敗: %v", err)
	}

	// inlineモードではenums.mdが生成されないこと
	if _, err := os.Stat(filepath.Join(dir, "enums.md")); !os.IsNotExist(err) {
		t.Error("inlineモードではenums.mdは生成されないべき")
	}

	// index.mdにEnum定義セクションがないこと
	indexData, _ := os.ReadFile(filepath.Join(dir, "index.md"))
	if strings.Contains(string(indexData), "Enum定義") {
		t.Error("inlineモードのindex.mdにEnum定義セクションは含まれないべき")
	}

	// テーブルページでinline展開されていること
	usersData, _ := os.ReadFile(filepath.Join(dir, "tables", "users.md"))
	users := string(usersData)
	if !strings.Contains(users, "ENUM(<br>'active',<br>'inactive'<br>)") {
		t.Error("inlineモードでEnum型が展開されていない")
	}
	if !strings.Contains(users, "ユーザー状態<br>active=有効, inactive=無効") {
		t.Error("inlineモードでEnumのNoteが展開されていない")
	}
}
