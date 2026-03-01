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
	outputFile := flag.String("o", "", "出力ファイルパス（省略時: stdoutまたはデフォルトファイル名）")
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
		w := os.Stdout
		if *outputFile != "" {
			f, err := os.Create(*outputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "エラー: 出力ファイル作成失敗: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}
		if err := generator.GenerateMarkdown(w, dbml, *enumMode); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: Markdown生成失敗: %v\n", err)
			os.Exit(1)
		}

	case "excel":
		output := *outputFile
		if output == "" {
			output = "output.xlsx"
		}
		if err := generator.GenerateExcel(output, dbml, *enumMode); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: Excel生成失敗: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s を生成しました\n", output)

	default:
		fmt.Fprintf(os.Stderr, "エラー: 不明な出力形式: %s（markdown または excel を指定してください）\n", *format)
		os.Exit(1)
	}
}
