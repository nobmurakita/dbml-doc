package generator

import (
	"fmt"
	"io"
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

// GenerateMarkdown はDBMLモデルからMarkdownテーブル定義書を生成する
func GenerateMarkdown(w io.Writer, dbml *model.DBML, enumMode string) error {
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

	// テーブル一覧
	if len(dbml.Tables) > 0 {
		fmt.Fprintln(w, "## テーブル一覧")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| # | テーブル名 | 説明 |")
		fmt.Fprintln(w, "|---|-----------|------|")
		for i, t := range dbml.Tables {
			tableName := t.Name
			if t.Schema != "" {
				tableName = t.Schema + "." + t.Name
			}
			anchor := toAnchor(tableName)
			fmt.Fprintf(w, "| %d | [%s](#%s) | %s |\n", i+1, tableName, anchor, t.Note)
		}
		fmt.Fprintln(w)
	}

	// Enumマップ構築（inlineモード用）
	enumMap := buildEnumMap(dbml)

	// Enum定義（independentモードのみ）
	if enumMode != "inline" && len(dbml.Enums) > 0 {
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

	// テーブル定義
	if len(dbml.Tables) > 0 {
		fmt.Fprintln(w, "## テーブル定義")
		fmt.Fprintln(w)

		// Refをテーブル→カラムでグルーピング
		refMap := buildRefMap(dbml)

		for _, t := range dbml.Tables {
			tableName := t.Name
			if t.Schema != "" {
				tableName = t.Schema + "." + t.Name
			}

			fmt.Fprintf(w, "### %s\n\n", tableName)
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

			// リレーション
			refs := refMap[tableName]
			if len(refs) > 0 {
				fmt.Fprintln(w, "**リレーション:**")
				fmt.Fprintln(w)
				fmt.Fprintln(w, "| カラム | 参照先 | 種類 |")
				fmt.Fprintln(w, "|--------|-------|------|")
				for _, r := range refs {
					fmt.Fprintf(w, "| %s | %s | %s |\n", r.column, r.target, r.relType)
				}
				fmt.Fprintln(w)
			}
		}
	}

	return nil
}

type refInfo struct {
	column  string
	target  string
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
					relType: formatRefType(c.Ref.Type),
				})
			}
		}
	}

	return result
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

// toAnchor はMarkdownの見出しテキストからGitHub互換のアンカーIDを生成する
func toAnchor(heading string) string {
	s := strings.ToLower(heading)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "")
	return s
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
