package parser

import (
	"testing"
)

func TestParseSimpleTable(t *testing.T) {
	input := `Table users {
  id integer [pk, increment]
  name varchar(255) [not null, unique]
  email varchar(255) [not null]
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if len(dbml.Tables) != 1 {
		t.Fatalf("テーブル数: got %d, want 1", len(dbml.Tables))
	}

	table := dbml.Tables[0]
	if table.Name != "users" {
		t.Errorf("テーブル名: got %q, want 'users'", table.Name)
	}
	if len(table.Columns) != 3 {
		t.Fatalf("カラム数: got %d, want 3", len(table.Columns))
	}

	// id カラム
	col := table.Columns[0]
	if col.Name != "id" || col.Type != "integer" {
		t.Errorf("col[0]: name=%q type=%q", col.Name, col.Type)
	}
	if !col.PrimaryKey {
		t.Error("col[0]: PK=false, want true")
	}
	if !col.Increment {
		t.Error("col[0]: Increment=false, want true")
	}

	// name カラム
	col = table.Columns[1]
	if col.Name != "name" || col.Type != "varchar(255)" {
		t.Errorf("col[1]: name=%q type=%q", col.Name, col.Type)
	}
	if !col.NotNull {
		t.Error("col[1]: NotNull=false, want true")
	}
	if !col.Unique {
		t.Error("col[1]: Unique=false, want true")
	}
}

func TestParseTableWithNote(t *testing.T) {
	input := `Table users {
  id integer [pk]

  Note: 'ユーザーテーブル'
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Tables[0].Note != "ユーザーテーブル" {
		t.Errorf("Note: got %q, want 'ユーザーテーブル'", dbml.Tables[0].Note)
	}
}

func TestParseTableWithSchema(t *testing.T) {
	input := `Table public.users {
  id integer [pk]
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Tables[0].Schema != "public" {
		t.Errorf("Schema: got %q, want 'public'", dbml.Tables[0].Schema)
	}
	if dbml.Tables[0].Name != "users" {
		t.Errorf("Name: got %q, want 'users'", dbml.Tables[0].Name)
	}
}

func TestParseTableWithAlias(t *testing.T) {
	input := `Table users as U {
  id integer [pk]
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Tables[0].Alias != "U" {
		t.Errorf("Alias: got %q, want 'U'", dbml.Tables[0].Alias)
	}
}

func TestParseColumnWithDefault(t *testing.T) {
	input := `Table users {
  status varchar [default: 'active']
  count integer [default: '0']
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	col0 := dbml.Tables[0].Columns[0]
	if col0.Default == nil || *col0.Default != "active" {
		t.Errorf("col[0] Default: got %v", col0.Default)
	}

	col1 := dbml.Tables[0].Columns[1]
	if col1.Default == nil || *col1.Default != "0" {
		t.Errorf("col[1] Default: got %v", col1.Default)
	}
}

func TestParseColumnWithNote(t *testing.T) {
	input := `Table users {
  id integer [pk, note: 'ユーザーID']
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Tables[0].Columns[0].Note != "ユーザーID" {
		t.Errorf("Note: got %q", dbml.Tables[0].Columns[0].Note)
	}
}

func TestParseInlineRef(t *testing.T) {
	input := `Table posts {
  id integer [pk]
  user_id integer [ref: > users.id]
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	col := dbml.Tables[0].Columns[1]
	if col.Ref == nil {
		t.Fatal("InlineRef is nil")
	}
	if col.Ref.Type != ">" {
		t.Errorf("Ref.Type: got %q, want '>'", col.Ref.Type)
	}
	if col.Ref.Table != "users" {
		t.Errorf("Ref.Table: got %q, want 'users'", col.Ref.Table)
	}
	if col.Ref.Column != "id" {
		t.Errorf("Ref.Column: got %q, want 'id'", col.Ref.Column)
	}
}

func TestParseIndexes(t *testing.T) {
	input := `Table users {
  id integer [pk]
  email varchar(255)
  name varchar(255)

  indexes {
    email [unique]
    (email, name) [name: 'idx_email_name', unique]
  }
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	indexes := dbml.Tables[0].Indexes
	if len(indexes) != 2 {
		t.Fatalf("インデックス数: got %d, want 2", len(indexes))
	}

	// 単一カラムインデックス
	idx0 := indexes[0]
	if len(idx0.Columns) != 1 || idx0.Columns[0].Name != "email" {
		t.Errorf("idx[0] columns: %v", idx0.Columns)
	}
	if !idx0.Unique {
		t.Error("idx[0] Unique=false, want true")
	}

	// 複合インデックス
	idx1 := indexes[1]
	if len(idx1.Columns) != 2 {
		t.Fatalf("idx[1] column count: got %d, want 2", len(idx1.Columns))
	}
	if idx1.Name != "idx_email_name" {
		t.Errorf("idx[1] Name: got %q", idx1.Name)
	}
	if !idx1.Unique {
		t.Error("idx[1] Unique=false, want true")
	}
}

func TestParseRef(t *testing.T) {
	input := `Ref: orders.user_id > users.id [delete: cascade]`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if len(dbml.Refs) != 1 {
		t.Fatalf("Ref数: got %d, want 1", len(dbml.Refs))
	}

	ref := dbml.Refs[0]
	if ref.Type != ">" {
		t.Errorf("Type: got %q, want '>'", ref.Type)
	}
	if ref.From.Table != "orders" || ref.From.Columns[0] != "user_id" {
		t.Errorf("From: %v", ref.From)
	}
	if ref.To.Table != "users" || ref.To.Columns[0] != "id" {
		t.Errorf("To: %v", ref.To)
	}
	if ref.OnDelete != "cascade" {
		t.Errorf("OnDelete: got %q, want 'cascade'", ref.OnDelete)
	}
}

func TestParseEnum(t *testing.T) {
	input := `Enum status {
  active [note: '有効']
  inactive [note: '無効']
  deleted
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if len(dbml.Enums) != 1 {
		t.Fatalf("Enum数: got %d, want 1", len(dbml.Enums))
	}

	enum := dbml.Enums[0]
	if enum.Name != "status" {
		t.Errorf("Name: got %q", enum.Name)
	}
	if len(enum.Values) != 3 {
		t.Fatalf("Values count: got %d, want 3", len(enum.Values))
	}
	if enum.Values[0].Name != "active" || enum.Values[0].Note != "有効" {
		t.Errorf("Values[0]: %v", enum.Values[0])
	}
	if enum.Values[2].Name != "deleted" || enum.Values[2].Note != "" {
		t.Errorf("Values[2]: %v", enum.Values[2])
	}
}

func TestParseProject(t *testing.T) {
	input := `Project mydb {
  database_type: 'PostgreSQL'
  Note: 'テストDB'
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Project == nil {
		t.Fatal("Project is nil")
	}
	if dbml.Project.Name != "mydb" {
		t.Errorf("Name: got %q", dbml.Project.Name)
	}
	if dbml.Project.DatabaseType != "PostgreSQL" {
		t.Errorf("DatabaseType: got %q", dbml.Project.DatabaseType)
	}
	if dbml.Project.Note != "テストDB" {
		t.Errorf("Note: got %q", dbml.Project.Note)
	}
}

func TestParseTableGroup(t *testing.T) {
	input := `TableGroup core {
  users
  posts
}`
	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if len(dbml.TableGroups) != 1 {
		t.Fatalf("TableGroup数: got %d, want 1", len(dbml.TableGroups))
	}

	tg := dbml.TableGroups[0]
	if tg.Name != "core" {
		t.Errorf("Name: got %q", tg.Name)
	}
	if len(tg.Tables) != 2 {
		t.Fatalf("Tables count: got %d, want 2", len(tg.Tables))
	}
	if tg.Tables[0] != "users" || tg.Tables[1] != "posts" {
		t.Errorf("Tables: %v", tg.Tables)
	}
}

func TestParseFullDBML(t *testing.T) {
	input := `Project ecommerce {
  database_type: 'PostgreSQL'
  Note: 'ECサイト'
}

Table users {
  id integer [pk, increment, note: 'ユーザーID']
  name varchar(255) [not null, note: 'ユーザー名']

  Note: 'ユーザーテーブル'
}

Table orders {
  id integer [pk, increment]
  user_id integer [not null]
  total decimal(10, 2) [not null]

  Note: '注文テーブル'
}

Ref: orders.user_id > users.id

Enum status {
  active
  inactive
}

TableGroup core {
  users
  orders
}`

	dbml, err := Parse(input)
	if err != nil {
		t.Fatalf("解析エラー: %v", err)
	}

	if dbml.Project == nil {
		t.Error("Project is nil")
	}
	if len(dbml.Tables) != 2 {
		t.Errorf("Tables: got %d, want 2", len(dbml.Tables))
	}
	if len(dbml.Refs) != 1 {
		t.Errorf("Refs: got %d, want 1", len(dbml.Refs))
	}
	if len(dbml.Enums) != 1 {
		t.Errorf("Enums: got %d, want 1", len(dbml.Enums))
	}
	if len(dbml.TableGroups) != 1 {
		t.Errorf("TableGroups: got %d, want 1", len(dbml.TableGroups))
	}
}
