package model

// DBML はDBMLファイル全体を表すルートノード
type DBML struct {
	Project     *Project
	Tables      []Table
	Enums       []Enum
	Refs        []Ref
	TableGroups []TableGroup
}

// Project はDBMLプロジェクト定義
type Project struct {
	Name         string
	DatabaseType string
	Note         string
}

// Table はテーブル定義
type Table struct {
	Schema  string
	Name    string
	Alias   string
	Note    string
	Columns []Column
	Indexes []Index
}

// Column はカラム定義
type Column struct {
	Name       string
	Type       string
	PrimaryKey bool
	NotNull    bool
	Unique     bool
	Increment  bool
	Default    *string
	Note       string
	Ref        *InlineRef
}

// InlineRef はカラム内のインラインリレーション定義
type InlineRef struct {
	Type   string // ">", "<", "-", "<>"
	Table  string
	Column string
}

// Index はインデックス定義
type Index struct {
	Columns    []IndexColumn
	Name       string
	Type       string // btree, hash
	Unique     bool
	PK         bool
	Note       string
}

// IndexColumn はインデックスのカラム定義
type IndexColumn struct {
	Name       string
	Expression string // バッククォート式の場合
}

// Ref はリレーション定義
type Ref struct {
	Name     string
	From     RefEndpoint
	To       RefEndpoint
	Type     string // ">", "<", "-", "<>"
	OnDelete string
	OnUpdate string
}

// RefEndpoint はリレーションの端点
type RefEndpoint struct {
	Schema  string
	Table   string
	Columns []string
}

// Enum は列挙型定義
type Enum struct {
	Schema string
	Name   string
	Values []EnumValue
}

// EnumValue は列挙型の値
type EnumValue struct {
	Name string
	Note string
}

// TableGroup はテーブルグループ定義
type TableGroup struct {
	Name   string
	Tables []string
}
