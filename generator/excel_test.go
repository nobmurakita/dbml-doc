package generator

import (
	"os"
	"testing"

	"github.com/nobmurakita/dbml-doc/model"
	"github.com/xuri/excelize/v2"
)

func TestGenerateExcelBasic(t *testing.T) {
	defaultVal := "now()"
	dbml := &model.DBML{
		Tables: []model.Table{
			{
				Name: "users",
				Note: "ユーザーテーブル",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true, NotNull: true, Note: "ユーザーID"},
					{Name: "name", Type: "varchar(255)", NotNull: true, Unique: true, Note: "ユーザー名"},
					{Name: "created_at", Type: "timestamp", Default: &defaultVal},
				},
			},
		},
	}

	tmpFile := t.TempDir() + "/test_output.xlsx"
	err := GenerateExcel(tmpFile, dbml, "independent")
	if err != nil {
		t.Fatalf("Excel生成エラー: %v", err)
	}

	// ファイルが存在することを確認
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("出力ファイルが作成されていない")
	}

	// Excelを開いて内容確認
	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("Excel読み込みエラー: %v", err)
	}
	defer f.Close()

	// テーブル一覧シート
	sheets := f.GetSheetList()
	if len(sheets) < 2 {
		t.Fatalf("シート数: got %d, want >= 2", len(sheets))
	}
	if sheets[0] != "テーブル一覧" {
		t.Errorf("sheet[0]: got %q, want 'テーブル一覧'", sheets[0])
	}
	if sheets[1] != "users" {
		t.Errorf("sheet[1]: got %q, want 'users'", sheets[1])
	}

	// テーブル一覧の内容
	val, _ := f.GetCellValue("テーブル一覧", "B2")
	if val != "users" {
		t.Errorf("テーブル一覧 B2: got %q, want 'users'", val)
	}
	val, _ = f.GetCellValue("テーブル一覧", "C2")
	if val != "ユーザーテーブル" {
		t.Errorf("テーブル一覧 C2: got %q, want 'ユーザーテーブル'", val)
	}

	// usersシートのカラム定義
	val, _ = f.GetCellValue("users", "B5")
	if val != "id" {
		t.Errorf("users B5: got %q, want 'id'", val)
	}
}

func TestGenerateExcelWithMultipleTables(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{Name: "users", Columns: []model.Column{{Name: "id", Type: "integer", PrimaryKey: true}}},
			{Name: "posts", Columns: []model.Column{{Name: "id", Type: "integer", PrimaryKey: true}}},
			{Name: "comments", Columns: []model.Column{{Name: "id", Type: "integer", PrimaryKey: true}}},
		},
	}

	tmpFile := t.TempDir() + "/test_multi.xlsx"
	err := GenerateExcel(tmpFile, dbml, "independent")
	if err != nil {
		t.Fatalf("Excel生成エラー: %v", err)
	}

	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("Excel読み込みエラー: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 4 { // テーブル一覧 + 3テーブル
		t.Errorf("シート数: got %d, want 4", len(sheets))
	}
}

func TestGenerateExcelWithEnumIndependent(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{
				Name: "users",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true},
					{Name: "status", Type: "user_status", NotNull: true},
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

	tmpFile := t.TempDir() + "/test_enum_independent.xlsx"
	err := GenerateExcel(tmpFile, dbml, "independent")
	if err != nil {
		t.Fatalf("Excel生成エラー: %v", err)
	}

	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("Excel読み込みエラー: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	// テーブル一覧 + Enum定義 + users = 3シート
	if len(sheets) != 3 {
		t.Fatalf("シート数: got %d, want 3 (sheets: %v)", len(sheets), sheets)
	}

	// Enum定義シートの内容を確認
	val, _ := f.GetCellValue("Enum定義", "A2")
	if val != "user_status" {
		t.Errorf("Enum定義 A2: got %q, want 'user_status'", val)
	}
	val, _ = f.GetCellValue("Enum定義", "B2")
	if val != "active" {
		t.Errorf("Enum定義 B2: got %q, want 'active'", val)
	}
	val, _ = f.GetCellValue("Enum定義", "C2")
	if val != "有効" {
		t.Errorf("Enum定義 C2: got %q, want '有効'", val)
	}

	// カラム型はEnum名のまま
	val, _ = f.GetCellValue("users", "C5")
	if val != "user_status" {
		t.Errorf("users C5 (型): got %q, want 'user_status'", val)
	}
}

func TestGenerateExcelWithEnumInline(t *testing.T) {
	dbml := &model.DBML{
		Tables: []model.Table{
			{
				Name: "users",
				Columns: []model.Column{
					{Name: "id", Type: "integer", PrimaryKey: true},
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

	tmpFile := t.TempDir() + "/test_enum_inline.xlsx"
	err := GenerateExcel(tmpFile, dbml, "inline")
	if err != nil {
		t.Fatalf("Excel生成エラー: %v", err)
	}

	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("Excel読み込みエラー: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	// テーブル一覧 + users = 2シート（Enum定義シートなし）
	if len(sheets) != 2 {
		t.Fatalf("シート数: got %d, want 2 (sheets: %v)", len(sheets), sheets)
	}

	// カラム型がENUM展開されていること（値ごとに改行）
	val, _ := f.GetCellValue("users", "C5")
	expectedType := "ENUM(\n'active',\n'inactive'\n)"
	if val != expectedType {
		t.Errorf("users C5 (型): got %q, want %q", val, expectedType)
	}

	// NoteにEnum説明が追加されていること
	val, _ = f.GetCellValue("users", "G5")
	expected := "ユーザー状態\nactive=有効, inactive=無効"
	if val != expected {
		t.Errorf("users G5 (説明): got %q, want %q", val, expected)
	}
}
