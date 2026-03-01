package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nobmurakita/dbml-doc/model"
)

// buildEnumMap はEnum名→定義のマップを構築する
func buildEnumMap(dbml *model.DBML) map[string]*model.Enum {
	enumMap := make(map[string]*model.Enum)
	for i := range dbml.Enums {
		e := &dbml.Enums[i]
		name := e.Name
		if e.Schema != "" {
			name = e.Schema + "." + e.Name
		}
		enumMap[name] = e
	}
	return enumMap
}

// formatEnumType はEnum値からENUM('val1','val2',...)形式の型文字列を生成する
// brは値間の改行文字（Markdown: "<br>", Excel: "\n"）
func formatEnumType(e *model.Enum, br string) string {
	var vals []string
	for _, v := range e.Values {
		vals = append(vals, "'"+v.Name+"'")
	}
	return "ENUM(" + br + strings.Join(vals, ","+br) + br + ")"
}

// formatEnumNote はEnum値の説明を生成する（Noteを持つ値のみ）
func formatEnumNote(e *model.Enum) string {
	var parts []string
	for _, v := range e.Values {
		if v.Note != "" {
			parts = append(parts, v.Name+"="+v.Note)
		}
	}
	return strings.Join(parts, ", ")
}

// writeProjectHeader はタイトルとプロジェクト情報を出力する
func writeProjectHeader(w io.Writer, dbml *model.DBML) {
	fmt.Fprintln(w, "# データベース定義書")
	fmt.Fprintln(w)

	if dbml.Project != nil {
		if dbml.Project.Name != "" {
			fmt.Fprintf(w, "**プロジェクト:** %s\n\n", dbml.Project.Name)
		}
		if dbml.Project.DatabaseType != "" {
			fmt.Fprintf(w, "**データベース:** %s\n\n", dbml.Project.DatabaseType)
		}
		if dbml.Project.Note != "" {
			fmt.Fprintf(w, "%s\n\n", dbml.Project.Note)
		}
	}
}

// writeTableList はテーブル一覧テーブルを出力する
// tableLinkはテーブル名からリンク先を生成する関数
func writeTableList(w io.Writer, dbml *model.DBML, tableLink func(string) string) {
	if len(dbml.Tables) == 0 {
		return
	}
	fmt.Fprintln(w, "## テーブル一覧")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| # | テーブル名 | 説明 |")
	fmt.Fprintln(w, "|---|-----------|------|")
	for i, t := range dbml.Tables {
		tableName := fullTableName(&t)
		link := tableLink(tableName)
		fmt.Fprintf(w, "| %d | [%s](%s) | %s |\n", i+1, tableName, link, t.Note)
	}
	fmt.Fprintln(w)
}

// writeEnumSection はEnum定義セクションを出力する（independentモード用）
func writeEnumSection(w io.Writer, dbml *model.DBML) {
	if len(dbml.Enums) == 0 {
		return
	}
	fmt.Fprintln(w, "## Enum定義")
	fmt.Fprintln(w)
	for _, e := range dbml.Enums {
		enumName := e.Name
		if e.Schema != "" {
			enumName = e.Schema + "." + e.Name
		}
		fmt.Fprintf(w, "### %s\n\n", enumName)
		fmt.Fprintln(w, "| 値 | 説明 |")
		fmt.Fprintln(w, "|----|------|")
		for _, v := range e.Values {
			fmt.Fprintf(w, "| %s | %s |\n", v.Name, v.Note)
		}
		fmt.Fprintln(w)
	}
}

// writeTableDetail は1テーブル分の定義（見出し・カラム・インデックス・リレーション）を出力する
// headingLevelはMarkdown見出しレベル（"###" や "##" など）
// refTableLinkはリレーション参照先テーブルへのリンク生成関数（nilの場合リンクなし）
func writeTableDetail(w io.Writer, t *model.Table, refs []refInfo, reverseRefs []refInfo, enumMode string, enumMap map[string]*model.Enum, headingLevel string, refTableLink func(string) string) {
	tableName := fullTableName(t)

	fmt.Fprintf(w, "%s %s\n\n", headingLevel, tableName)
	if t.Note != "" {
		fmt.Fprintf(w, "%s\n\n", t.Note)
	}

	// カラム定義
	fmt.Fprintln(w, "| # | カラム名 | 型 | NULL | デフォルト | 制約 | 説明 |")
	fmt.Fprintln(w, "|---|---------|-----|------|----------|------|------|")
	for i, c := range t.Columns {
		nullable := "YES"
		if c.NotNull || c.PrimaryKey {
			nullable = "NO"
		}
		defaultVal := "-"
		if c.Default != nil {
			defaultVal = *c.Default
		}
		constraints := buildConstraints(c)
		colType := c.Type
		colNote := c.Note
		if enumMode == "inline" {
			if e, ok := enumMap[c.Type]; ok {
				colType = formatEnumType(e, "<br>")
				enumNote := formatEnumNote(e)
				if enumNote != "" {
					if colNote != "" {
						colNote = colNote + "<br>" + enumNote
					} else {
						colNote = enumNote
					}
				}
			}
		}
		fmt.Fprintf(w, "| %d | %s | %s | %s | %s | %s | %s |\n",
			i+1, c.Name, colType, nullable, defaultVal, constraints, colNote)
	}
	fmt.Fprintln(w)

	// インデックス
	if len(t.Indexes) > 0 {
		fmt.Fprintln(w, "**インデックス:**")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| インデックス名 | カラム | 種類 | ユニーク |")
		fmt.Fprintln(w, "|--------------|--------|------|---------|")
		for _, idx := range t.Indexes {
			idxName := idx.Name
			if idxName == "" {
				idxName = "-"
			}
			cols := formatIndexColumns(idx.Columns)
			idxType := idx.Type
			if idxType == "" {
				idxType = "-"
			}
			unique := "NO"
			if idx.Unique || idx.PK {
				unique = "YES"
			}
			fmt.Fprintf(w, "| %s | %s | %s | %s |\n", idxName, cols, idxType, unique)
		}
		fmt.Fprintln(w)
	}

	// リレーション（参照先）
	if len(refs) > 0 {
		fmt.Fprintln(w, "**リレーション（参照先）:**")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| カラム | 参照先 | 種類 |")
		fmt.Fprintln(w, "|--------|-------|------|")
		for _, r := range refs {
			target := r.target
			if refTableLink != nil {
				target = fmt.Sprintf("[%s](%s)", r.target, refTableLink(r.toTable))
			}
			fmt.Fprintf(w, "| %s | %s | %s |\n", r.column, target, r.relType)
		}
		fmt.Fprintln(w)
	}

	// リレーション（参照元）
	if len(reverseRefs) > 0 {
		fmt.Fprintln(w, "**リレーション（参照元）:**")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| カラム | 参照元 | 種類 |")
		fmt.Fprintln(w, "|--------|-------|------|")
		for _, r := range reverseRefs {
			source := r.target
			if refTableLink != nil {
				source = fmt.Sprintf("[%s](%s)", r.target, refTableLink(r.toTable))
			}
			fmt.Fprintf(w, "| %s | %s | %s |\n", r.column, source, r.relType)
		}
		fmt.Fprintln(w)
	}
}


type refInfo struct {
	column  string
	target  string   // 表示用（既存互換）
	toTable string   // リンク生成用
	toCols  string   // リンク生成用
	relType string
}

// buildRefMap はテーブル名→リレーション情報のマップを構築する
func buildRefMap(dbml *model.DBML) map[string][]refInfo {
	result := make(map[string][]refInfo)

	// 明示的Refから
	for _, r := range dbml.Refs {
		fromTable := r.From.Table
		if r.From.Schema != "" {
			fromTable = r.From.Schema + "." + fromTable
		}
		toTable := r.To.Table
		if r.To.Schema != "" {
			toTable = r.To.Schema + "." + toTable
		}

		fromCols := strings.Join(r.From.Columns, ", ")
		toCols := strings.Join(r.To.Columns, ", ")
		relType := formatRefType(r.Type)

		result[fromTable] = append(result[fromTable], refInfo{
			column:  fromCols,
			target:  toTable + "." + toCols,
			toTable: toTable,
			toCols:  toCols,
			relType: relType,
		})
	}

	// インラインRefから
	for _, t := range dbml.Tables {
		tableName := t.Name
		if t.Schema != "" {
			tableName = t.Schema + "." + t.Name
		}
		for _, c := range t.Columns {
			if c.Ref != nil {
				result[tableName] = append(result[tableName], refInfo{
					column:  c.Name,
					target:  c.Ref.Table + "." + c.Ref.Column,
					toTable: c.Ref.Table,
					toCols:  c.Ref.Column,
					relType: formatRefType(c.Ref.Type),
				})
			}
		}
	}

	return result
}

// buildReverseRefMap はテーブル名→参照元リレーション情報のマップを構築する
func buildReverseRefMap(dbml *model.DBML) map[string][]refInfo {
	result := make(map[string][]refInfo)

	// 明示的Refから
	for _, r := range dbml.Refs {
		fromTable := r.From.Table
		if r.From.Schema != "" {
			fromTable = r.From.Schema + "." + fromTable
		}
		toTable := r.To.Table
		if r.To.Schema != "" {
			toTable = r.To.Schema + "." + toTable
		}

		fromCols := strings.Join(r.From.Columns, ", ")
		toCols := strings.Join(r.To.Columns, ", ")
		relType := reverseRelType(formatRefType(r.Type))

		result[toTable] = append(result[toTable], refInfo{
			column:  toCols,
			target:  fromTable + "." + fromCols,
			toTable: fromTable,
			toCols:  fromCols,
			relType: relType,
		})
	}

	// インラインRefから
	for _, t := range dbml.Tables {
		tableName := t.Name
		if t.Schema != "" {
			tableName = t.Schema + "." + t.Name
		}
		for _, c := range t.Columns {
			if c.Ref != nil {
				relType := reverseRelType(formatRefType(c.Ref.Type))
				result[c.Ref.Table] = append(result[c.Ref.Table], refInfo{
					column:  c.Ref.Column,
					target:  tableName + "." + c.Name,
					toTable: tableName,
					toCols:  c.Name,
					relType: relType,
				})
			}
		}
	}

	return result
}

// reverseRelType はリレーション種類を逆方向にする
func reverseRelType(relType string) string {
	switch relType {
	case "N:1":
		return "1:N"
	case "1:N":
		return "N:1"
	default:
		return relType
	}
}

func formatRefType(refType string) string {
	switch refType {
	case ">":
		return "N:1"
	case "<":
		return "1:N"
	case "-":
		return "1:1"
	case "<>":
		return "N:N"
	default:
		return refType
	}
}

// fullTableName はスキーマ付きテーブル名を返す
func fullTableName(t *model.Table) string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

// tableToFileName はテーブル名を安全なファイル名に変換する（.→_、小文字化）
func tableToFileName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, ".", "_")
	return s + ".md"
}


func buildConstraints(c model.Column) string {
	var parts []string
	if c.PrimaryKey {
		parts = append(parts, "PK")
	}
	if c.Unique {
		parts = append(parts, "UNIQUE")
	}
	if c.Increment {
		parts = append(parts, "AUTO INCREMENT")
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func formatIndexColumns(cols []model.IndexColumn) string {
	var names []string
	for _, c := range cols {
		if c.Expression != "" {
			names = append(names, "`"+c.Expression+"`")
		} else {
			names = append(names, c.Name)
		}
	}
	return strings.Join(names, ", ")
}

// writeIndexPage はindex.mdの内容を出力する
func writeIndexPage(w io.Writer, dbml *model.DBML, enumMode string) {
	writeProjectHeader(w, dbml)

	// テーブル一覧（tables/ディレクトリ内のファイルへのリンク）
	writeTableList(w, dbml, func(name string) string {
		return "tables/" + tableToFileName(name)
	})

	// Enum一覧へのリンク（independentモード時のみ）
	if enumMode != "inline" && len(dbml.Enums) > 0 {
		fmt.Fprintln(w, "## Enum定義")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "[Enum定義一覧](enums.md)\n\n")
	}
}

// writeEnumPage はenums.mdの内容を出力する
func writeEnumPage(w io.Writer, dbml *model.DBML) {
	fmt.Fprintln(w, "[< 目次に戻る](index.md)")
	fmt.Fprintln(w)
	writeEnumSection(w, dbml)
}

// writeTablePage はテーブルページの内容を出力する
func writeTablePage(w io.Writer, t *model.Table, refs []refInfo, reverseRefs []refInfo, enumMode string, enumMap map[string]*model.Enum, refTableLink func(string) string) {
	fmt.Fprintln(w, "[< 目次に戻る](../index.md)")
	fmt.Fprintln(w)
	writeTableDetail(w, t, refs, reverseRefs, enumMode, enumMap, "##", refTableLink)
}

// GenerateMarkdownPages はマルチページMarkdown出力のエントリポイント
func GenerateMarkdownPages(outputDir string, dbml *model.DBML, enumMode string) error {
	// ディレクトリ作成
	tablesDir := filepath.Join(outputDir, "tables")
	if err := os.MkdirAll(tablesDir, 0755); err != nil {
		return fmt.Errorf("ディレクトリ作成失敗: %w", err)
	}

	// index.md
	indexFile, err := os.Create(filepath.Join(outputDir, "index.md"))
	if err != nil {
		return fmt.Errorf("index.md作成失敗: %w", err)
	}
	defer indexFile.Close()
	writeIndexPage(indexFile, dbml, enumMode)

	// enums.md（independentモード時のみ）
	if enumMode != "inline" && len(dbml.Enums) > 0 {
		enumFile, err := os.Create(filepath.Join(outputDir, "enums.md"))
		if err != nil {
			return fmt.Errorf("enums.md作成失敗: %w", err)
		}
		defer enumFile.Close()
		writeEnumPage(enumFile, dbml)
	}

	// テーブルページ
	enumMap := buildEnumMap(dbml)
	refMap := buildRefMap(dbml)
	reverseRefMap := buildReverseRefMap(dbml)
	refTableLink := func(name string) string {
		return tableToFileName(name)
	}

	for _, t := range dbml.Tables {
		tableName := fullTableName(&t)
		fileName := tableToFileName(tableName)
		f, err := os.Create(filepath.Join(tablesDir, fileName))
		if err != nil {
			return fmt.Errorf("テーブルファイル作成失敗(%s): %w", fileName, err)
		}
		writeTablePage(f, &t, refMap[tableName], reverseRefMap[tableName], enumMode, enumMap, refTableLink)
		f.Close()
	}

	return nil
}
