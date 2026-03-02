package generator

import (
	"fmt"
	"strings"

	"github.com/nobmurakita/dbml-doc/model"
	"github.com/xuri/excelize/v2"
)

// GenerateExcel はDBMLモデルからExcelテーブル定義書を生成する
func GenerateExcel(filename string, dbml *model.DBML, enumMode string) error {
	f := excelize.NewFile()
	defer f.Close()

	// スタイル定義
	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("ヘッダースタイル作成エラー: %w", err)
	}
	cellStyle, err := createCellStyle(f)
	if err != nil {
		return fmt.Errorf("セルスタイル作成エラー: %w", err)
	}

	// テーブル一覧シート
	indexSheet := "テーブル一覧"
	f.SetSheetName("Sheet1", indexSheet)
	if err := writeTableIndex(f, indexSheet, dbml, headerStyle, cellStyle); err != nil {
		return err
	}

	// Enumマップ構築
	enumMap := buildEnumMap(dbml)

	// Enum定義シート（independentモードのみ）
	if enumMode != "inline" && len(dbml.Enums) > 0 {
		if err := writeEnumSheet(f, dbml, headerStyle, cellStyle); err != nil {
			return err
		}
	}

	// Refマップ構築
	refMap := buildRefMap(dbml)
	reverseRefMap := buildReverseRefMap(dbml)

	// テーブルごとのシート
	for _, t := range dbml.Tables {
		sheetName := t.Name
		if t.Schema != "" {
			sheetName = t.Schema + "." + t.Name
		}
		// Excelシート名の制限（31文字）
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}

		if _, err := f.NewSheet(sheetName); err != nil {
			return fmt.Errorf("シート作成エラー (%s): %w", sheetName, err)
		}

		tableName := t.Name
		if t.Schema != "" {
			tableName = t.Schema + "." + t.Name
		}
		refs := refMap[tableName]
		reverseRefs := reverseRefMap[tableName]

		if err := writeTableSheet(f, sheetName, &t, refs, reverseRefs, enumMode, enumMap, headerStyle, cellStyle); err != nil {
			return err
		}
	}

	return f.SaveAs(filename)
}

func createHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#FFFFFF",
			Size:  11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#2B4C7E"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#B0B0B0", Style: 1},
			{Type: "top", Color: "#B0B0B0", Style: 1},
			{Type: "bottom", Color: "#B0B0B0", Style: 1},
			{Type: "right", Color: "#B0B0B0", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
}

func createLinkStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color:     "#0563C1",
			Underline: "single",
			Size:      11,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#B0B0B0", Style: 1},
			{Type: "top", Color: "#B0B0B0", Style: 1},
			{Type: "bottom", Color: "#B0B0B0", Style: 1},
			{Type: "right", Color: "#B0B0B0", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})
}

func createSectionStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})
}

func createCellStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#B0B0B0", Style: 1},
			{Type: "top", Color: "#B0B0B0", Style: 1},
			{Type: "bottom", Color: "#B0B0B0", Style: 1},
			{Type: "right", Color: "#B0B0B0", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
	})
}

func writeTableIndex(f *excelize.File, sheet string, dbml *model.DBML, headerStyle, cellStyle int) error {
	linkStyle, err := createLinkStyle(f)
	if err != nil {
		return fmt.Errorf("リンクスタイル作成エラー: %w", err)
	}

	headers := []string{"#", "テーブル名", "説明"}
	widths := []float64{5, 30, 50}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		f.SetColWidth(sheet, colName(i+1), colName(i+1), widths[i])
	}

	for i, t := range dbml.Tables {
		row := i + 2
		tableName := t.Name
		if t.Schema != "" {
			tableName = t.Schema + "." + t.Name
		}
		sheetName := tableName
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}

		// #
		numCell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheet, numCell, i+1)
		f.SetCellStyle(sheet, numCell, numCell, cellStyle)

		// テーブル名（HYPERLINKで各シートへリンク）
		nameCell, _ := excelize.CoordinatesToCellName(2, row)
		formula := fmt.Sprintf(`HYPERLINK("#'%s'!A1","%s")`, sheetName, tableName)
		f.SetCellFormula(sheet, nameCell, formula)
		f.SetCellStyle(sheet, nameCell, nameCell, linkStyle)

		// 説明
		noteCell, _ := excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheet, noteCell, t.Note)
		f.SetCellStyle(sheet, noteCell, noteCell, cellStyle)
	}

	return nil
}

func writeEnumSheet(f *excelize.File, dbml *model.DBML, headerStyle, cellStyle int) error {
	sheetName := "Enum定義"
	if _, err := f.NewSheet(sheetName); err != nil {
		return fmt.Errorf("シート作成エラー (%s): %w", sheetName, err)
	}

	headers := []string{"Enum名", "値", "説明"}
	widths := []float64{25, 25, 40}
	for i, h := range headers {
		f.SetCellValue(sheetName, cell(i+1, 1), h)
		f.SetCellStyle(sheetName, cell(i+1, 1), cell(i+1, 1), headerStyle)
		f.SetColWidth(sheetName, colName(i+1), colName(i+1), widths[i])
	}

	row := 2
	for _, e := range dbml.Enums {
		enumName := e.Name
		if e.Schema != "" {
			enumName = e.Schema + "." + e.Name
		}
		for _, v := range e.Values {
			values := []interface{}{enumName, v.Name, v.Note}
			for j, val := range values {
				f.SetCellValue(sheetName, cell(j+1, row), val)
				f.SetCellStyle(sheetName, cell(j+1, row), cell(j+1, row), cellStyle)
			}
			row++
		}
	}

	return nil
}

func writeTableSheet(f *excelize.File, sheet string, t *model.Table, refs []refInfo, reverseRefs []refInfo, enumMode string, enumMap map[string]*model.Enum, headerStyle, cellStyle int) error {
	sectionStyle, err := createSectionStyle(f)
	if err != nil {
		return fmt.Errorf("セクションスタイル作成エラー: %w", err)
	}

	row := 1

	// テーブル名
	f.SetCellValue(sheet, cell(1, row), "テーブル名")
	f.MergeCell(sheet, cell(1, row), cell(2, row))
	f.SetCellStyle(sheet, cell(1, row), cell(2, row), headerStyle)
	tableName := t.Name
	if t.Schema != "" {
		tableName = t.Schema + "." + t.Name
	}
	f.SetCellValue(sheet, cell(3, row), tableName)
	f.SetCellStyle(sheet, cell(3, row), cell(3, row), cellStyle)
	row++

	// 説明
	if t.Note != "" {
		f.SetCellValue(sheet, cell(1, row), "説明")
		f.MergeCell(sheet, cell(1, row), cell(2, row))
		f.SetCellStyle(sheet, cell(1, row), cell(2, row), headerStyle)
		f.SetCellValue(sheet, cell(3, row), t.Note)
		f.SetCellStyle(sheet, cell(3, row), cell(3, row), cellStyle)
		row++
	}

	row++ // 空行

	// カラム見出し
	f.SetCellValue(sheet, cell(1, row), "カラム")
	f.SetCellStyle(sheet, cell(1, row), cell(1, row), sectionStyle)
	row++

	// カラム定義ヘッダー
	colHeaders := []string{"#", "カラム名", "型", "NULL", "デフォルト", "制約", "説明"}
	colWidths := []float64{5, 20, 30, 8, 15, 18, 30}
	for i, h := range colHeaders {
		f.SetCellValue(sheet, cell(i+1, row), h)
		f.SetCellStyle(sheet, cell(i+1, row), cell(i+1, row), headerStyle)
		f.SetColWidth(sheet, colName(i+1), colName(i+1), colWidths[i])
	}
	row++

	// カラム定義
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
				colType = formatEnumType(e, "\n")
				enumNote := formatEnumNote(e)
				if enumNote != "" {
					if colNote != "" {
						colNote = colNote + "\n" + enumNote
					} else {
						colNote = enumNote
					}
				}
			}
		}

		values := []interface{}{i + 1, c.Name, colType, nullable, defaultVal, constraints, colNote}
		for j, v := range values {
			f.SetCellValue(sheet, cell(j+1, row), v)
			f.SetCellStyle(sheet, cell(j+1, row), cell(j+1, row), cellStyle)
		}
		row++
	}

	// インデックス
	if len(t.Indexes) > 0 {
		row++ // 空行

		// インデックス見出し
		f.SetCellValue(sheet, cell(1, row), "インデックス")
		f.SetCellStyle(sheet, cell(1, row), cell(1, row), sectionStyle)
		row++

		idxHeaders := []string{"#", "インデックス名", "カラム", "種類", "ユニーク"}
		for i, h := range idxHeaders {
			f.SetCellValue(sheet, cell(i+1, row), h)
			f.SetCellStyle(sheet, cell(i+1, row), cell(i+1, row), headerStyle)
		}
		row++

		for i, idx := range t.Indexes {
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

			values := []interface{}{i + 1, idxName, cols, idxType, unique}
			for j, v := range values {
				f.SetCellValue(sheet, cell(j+1, row), v)
				f.SetCellStyle(sheet, cell(j+1, row), cell(j+1, row), cellStyle)
			}
			row++
		}
	}

	// リンクスタイル
	linkStyle, err := createLinkStyle(f)
	if err != nil {
		return fmt.Errorf("リンクスタイル作成エラー: %w", err)
	}

	// リレーション（参照先）
	if len(refs) > 0 {
		row++ // 空行

		// リレーション（参照先）見出し
		f.SetCellValue(sheet, cell(1, row), "リレーション（参照先）")
		f.SetCellStyle(sheet, cell(1, row), cell(1, row), sectionStyle)
		row++

		refHeaders := []string{"#", "カラム", "参照先", "種類"}
		for i, h := range refHeaders {
			f.SetCellValue(sheet, cell(i+1, row), h)
			f.SetCellStyle(sheet, cell(i+1, row), cell(i+1, row), headerStyle)
		}
		row++

		for i, r := range refs {
			writeRefRow(f, sheet, row, i+1, r, linkStyle, cellStyle)
			row++
		}
	}

	// リレーション（参照元）
	if len(reverseRefs) > 0 {
		row++ // 空行

		// リレーション（参照元）見出し
		f.SetCellValue(sheet, cell(1, row), "リレーション（参照元）")
		f.SetCellStyle(sheet, cell(1, row), cell(1, row), sectionStyle)
		row++

		revHeaders := []string{"#", "カラム", "参照元", "種類"}
		for i, h := range revHeaders {
			f.SetCellValue(sheet, cell(i+1, row), h)
			f.SetCellStyle(sheet, cell(i+1, row), cell(i+1, row), headerStyle)
		}
		row++

		for i, r := range reverseRefs {
			writeRefRow(f, sheet, row, i+1, r, linkStyle, cellStyle)
			row++
		}
	}

	return nil
}

func writeRefRow(f *excelize.File, sheet string, row int, num int, r refInfo, linkStyle, cellStyle int) {
	// #
	f.SetCellValue(sheet, cell(1, row), num)
	f.SetCellStyle(sheet, cell(1, row), cell(1, row), cellStyle)

	// カラム
	f.SetCellValue(sheet, cell(2, row), r.column)
	f.SetCellStyle(sheet, cell(2, row), cell(2, row), cellStyle)

	// 参照先/参照元（HYPERLINKで対象シートへリンク）
	targetCell := cell(3, row)
	sheetName := r.toTable
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}
	formula := fmt.Sprintf(`HYPERLINK("#'%s'!A1","%s")`, sheetName, r.target)
	f.SetCellFormula(sheet, targetCell, formula)
	f.SetCellStyle(sheet, targetCell, targetCell, linkStyle)

	// 種類
	f.SetCellValue(sheet, cell(4, row), r.relType)
	f.SetCellStyle(sheet, cell(4, row), cell(4, row), cellStyle)
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func colName(col int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return name
}

// formatRefColumns は内部で使う（markdownと共有のためexportは不要）
func formatRefColumns(cols []string) string {
	return strings.Join(cols, ", ")
}
