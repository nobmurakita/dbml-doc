package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nobmurakita/dbml-doc/generator"
	"github.com/nobmurakita/dbml-doc/parser"
)

func main() {
	inputFile := flag.String("i", "", "入力DBMLファイル（必須）")
	format := flag.String("f", "markdown", "出力形式: markdown | excel")
	output := flag.String("o", "", "出力先（markdown: ディレクトリ、excel: ファイルパス）")
	enumMode := flag.String("e", "independent", "Enum表示モード: independent | inline")
	flag.Parse()

	if *enumMode != "independent" && *enumMode != "inline" {
		fmt.Fprintf(os.Stderr, "エラー: 不明なEnumモード: %s（independent または inline を指定してください）\n", *enumMode)
		flag.Usage()
		os.Exit(1)
	}

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "エラー: 入力ファイルを指定してください (-i)")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: ファイル読み込み失敗: %v\n", err)
		os.Exit(1)
	}

	dbml, err := parser.Parse(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: DBML解析失敗: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "markdown":
		outDir := *output
		if outDir == "" {
			outDir = "output"
		}
		if err := generator.GenerateMarkdownPages(outDir, dbml, *enumMode); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: Markdown生成失敗: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s/ に出力しました\n", outDir)

	case "excel":
		outFile := *output
		if outFile == "" {
			outFile = "output.xlsx"
		}
		if err := generator.GenerateExcel(outFile, dbml, *enumMode); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: Excel生成失敗: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s を生成しました\n", outFile)

	default:
		fmt.Fprintf(os.Stderr, "エラー: 不明な出力形式: %s（markdown または excel を指定してください）\n", *format)
		os.Exit(1)
	}
}
