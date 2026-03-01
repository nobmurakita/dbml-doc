package parser

import (
	"testing"
)

func TestLexerBasicTokens(t *testing.T) {
	input := `Table users {
  id integer [pk]
}`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	expected := []TokenType{
		TokenTable, TokenIdent, TokenLBrace, TokenNewline,
		TokenIdent, TokenIdent, TokenLBracket, TokenPK, TokenRBracket, TokenNewline,
		TokenRBrace, TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("トークン数が不一致: got %d, want %d\ntokens: %v", len(tokens), len(expected), tokens)
	}

	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("tokens[%d]: got %s, want %s (value=%q)", i, tok.Type, expected[i], tok.Value)
		}
	}
}

func TestLexerStringLiterals(t *testing.T) {
	input := `'hello' "world"`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	if tokens[0].Type != TokenString || tokens[0].Value != "hello" {
		t.Errorf("token[0]: got %s %q, want String 'hello'", tokens[0].Type, tokens[0].Value)
	}
	if tokens[1].Type != TokenString || tokens[1].Value != "world" {
		t.Errorf("token[1]: got %s %q, want String 'world'", tokens[1].Type, tokens[1].Value)
	}
}

func TestLexerTripleQuote(t *testing.T) {
	input := "'''\n  これは\n  複数行のノート\n'''"
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	if tokens[0].Type != TokenString {
		t.Fatalf("三重クォートのトークン種別が不正: %s", tokens[0].Type)
	}
	if tokens[0].Value != "  これは\n  複数行のノート" {
		t.Errorf("三重クォート値: got %q", tokens[0].Value)
	}
}

func TestLexerComments(t *testing.T) {
	input := `// コメント
Table /* ブロックコメント */ users {
}`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	// コメントはスキップされるため Table, users, {, }, EOF のみ
	expectedTypes := []TokenType{TokenNewline, TokenTable, TokenIdent, TokenLBrace, TokenNewline, TokenRBrace, TokenEOF}
	if len(tokens) != len(expectedTypes) {
		t.Fatalf("トークン数: got %d, want %d\ntokens: %v", len(tokens), len(expectedTypes), tokens)
	}
	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("tokens[%d]: got %s, want %s", i, tok.Type, expectedTypes[i])
		}
	}
}

func TestLexerBacktick(t *testing.T) {
	input := "`now()`"
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	if tokens[0].Type != TokenBacktick || tokens[0].Value != "now()" {
		t.Errorf("バッククォート: got %s %q, want Backtick 'now()'", tokens[0].Type, tokens[0].Value)
	}
}

func TestLexerRelationSymbols(t *testing.T) {
	input := `> < - <>`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	expectedTypes := []TokenType{TokenGT, TokenLT, TokenMinus, TokenLTGT, TokenEOF}
	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("tokens[%d]: got %s, want %s", i, tok.Type, expectedTypes[i])
		}
	}
}

func TestLexerKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"Table", TokenTable},
		{"Ref", TokenRef},
		{"Enum", TokenEnum},
		{"Project", TokenProject},
		{"TableGroup", TokenTableGroup},
		{"indexes", TokenIndexes},
		{"Note", TokenNote},
		{"pk", TokenPK},
		{"not", TokenNot},
		{"null", TokenNull},
		{"unique", TokenUnique},
		{"increment", TokenIncrement},
		{"default", TokenDefault},
		{"primary", TokenPrimary},
		{"key", TokenKey},
	}

	for _, tt := range tests {
		lexer := NewLexer(tt.input)
		tokens, err := lexer.Tokenize()
		if err != nil {
			t.Fatalf("字句解析エラー (%s): %v", tt.input, err)
		}
		if tokens[0].Type != tt.expected {
			t.Errorf("keyword %q: got %s, want %s", tt.input, tokens[0].Type, tt.expected)
		}
	}
}

func TestLexerNumber(t *testing.T) {
	input := `255 10.2`
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("字句解析エラー: %v", err)
	}

	if tokens[0].Type != TokenNumber || tokens[0].Value != "255" {
		t.Errorf("tokens[0]: got %s %q", tokens[0].Type, tokens[0].Value)
	}
	if tokens[1].Type != TokenNumber || tokens[1].Value != "10.2" {
		t.Errorf("tokens[1]: got %s %q", tokens[1].Type, tokens[1].Value)
	}
}
