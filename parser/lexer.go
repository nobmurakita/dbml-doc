package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType はトークンの種類
type TokenType int

const (
	// 特殊トークン
	TokenEOF TokenType = iota
	TokenNewline

	// リテラル
	TokenIdent      // 識別子
	TokenString     // '...' または '''...'''
	TokenNumber     // 数値
	TokenBacktick   // `...`

	// キーワード
	TokenTable
	TokenRef
	TokenEnum
	TokenProject
	TokenTableGroup
	TokenIndexes
	TokenNote
	TokenAs
	TokenNull
	TokenNot
	TokenUnique
	TokenPK
	TokenPrimary
	TokenKey
	TokenIncrement
	TokenDefault
	TokenType_
	TokenName
	TokenHeaderColor
	TokenDelete
	TokenUpdate
	TokenCascade
	TokenRestrict
	TokenNoAction
	TokenSetNull
	TokenSetDefault

	// 記号
	TokenLBrace    // {
	TokenRBrace    // }
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenLParen    // (
	TokenRParen    // )
	TokenColon     // :
	TokenComma     // ,
	TokenDot       // .
	TokenGT        // >
	TokenLT        // <
	TokenMinus     // -
	TokenLTGT      // <>
)

var tokenNames = map[TokenType]string{
	TokenEOF:         "EOF",
	TokenNewline:     "Newline",
	TokenIdent:       "Ident",
	TokenString:      "String",
	TokenNumber:      "Number",
	TokenBacktick:    "Backtick",
	TokenTable:       "Table",
	TokenRef:         "Ref",
	TokenEnum:        "Enum",
	TokenProject:     "Project",
	TokenTableGroup:  "TableGroup",
	TokenIndexes:     "indexes",
	TokenNote:        "Note",
	TokenAs:          "as",
	TokenNull:        "null",
	TokenNot:         "not",
	TokenUnique:      "unique",
	TokenPK:          "pk",
	TokenPrimary:     "primary",
	TokenKey:         "key",
	TokenIncrement:   "increment",
	TokenDefault:     "default",
	TokenType_:       "type",
	TokenName:        "name",
	TokenHeaderColor: "headercolor",
	TokenDelete:      "delete",
	TokenUpdate:      "update",
	TokenCascade:     "cascade",
	TokenRestrict:    "restrict",
	TokenNoAction:    "no action",
	TokenSetNull:     "set null",
	TokenSetDefault:  "set default",
	TokenLBrace:      "{",
	TokenRBrace:      "}",
	TokenLBracket:    "[",
	TokenRBracket:    "]",
	TokenLParen:      "(",
	TokenRParen:      ")",
	TokenColon:       ":",
	TokenComma:       ",",
	TokenDot:         ".",
	TokenGT:          ">",
	TokenLT:          "<",
	TokenMinus:       "-",
	TokenLTGT:        "<>",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", int(t))
}

// キーワードマップ（小文字で比較）
var keywords = map[string]TokenType{
	"table":      TokenTable,
	"ref":        TokenRef,
	"enum":       TokenEnum,
	"project":    TokenProject,
	"tablegroup": TokenTableGroup,
	"indexes":    TokenIndexes,
	"note":       TokenNote,
	"as":         TokenAs,
	"null":       TokenNull,
	"not":        TokenNot,
	"unique":     TokenUnique,
	"pk":         TokenPK,
	"primary":    TokenPrimary,
	"key":        TokenKey,
	"increment":  TokenIncrement,
	"default":    TokenDefault,
	"type":       TokenType_,
	"name":       TokenName,
	"headercolor": TokenHeaderColor,
	"delete":     TokenDelete,
	"update":     TokenUpdate,
	"cascade":    TokenCascade,
	"restrict":   TokenRestrict,
}

// Token は字句解析の結果トークン
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Col     int
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, L%d:C%d)", t.Type, t.Value, t.Line, t.Col)
}

// Lexer はDBMLの字句解析器
type Lexer struct {
	input   []rune
	pos     int
	line    int
	col     int
}

// NewLexer は新しいLexerを生成する
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
		col:   1,
	}
}

// Tokenize は入力全体をトークン列に変換する
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.input) {
		return 0
	}
	return l.input[idx]
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) skipLineComment() {
	for l.pos < len(l.input) && l.peek() != '\n' {
		l.advance()
	}
}

func (l *Lexer) skipBlockComment() error {
	startLine := l.line
	// /*はすでに消費済み
	for l.pos < len(l.input) {
		if l.peek() == '*' && l.peekAt(1) == '/' {
			l.advance() // *
			l.advance() // /
			return nil
		}
		l.advance()
	}
	return fmt.Errorf("L%d: ブロックコメントが閉じられていません", startLine)
}

func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Line: l.line, Col: l.col}, nil
	}

	ch := l.peek()

	// コメント処理
	if ch == '/' {
		if l.peekAt(1) == '/' {
			l.advance()
			l.advance()
			l.skipLineComment()
			return l.nextToken()
		}
		if l.peekAt(1) == '*' {
			l.advance()
			l.advance()
			if err := l.skipBlockComment(); err != nil {
				return Token{}, err
			}
			return l.nextToken()
		}
	}

	line, col := l.line, l.col

	// 改行
	if ch == '\n' {
		l.advance()
		return Token{Type: TokenNewline, Value: "\n", Line: line, Col: col}, nil
	}

	// 記号
	switch ch {
	case '{':
		l.advance()
		return Token{Type: TokenLBrace, Value: "{", Line: line, Col: col}, nil
	case '}':
		l.advance()
		return Token{Type: TokenRBrace, Value: "}", Line: line, Col: col}, nil
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Value: "[", Line: line, Col: col}, nil
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Value: "]", Line: line, Col: col}, nil
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Value: "(", Line: line, Col: col}, nil
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Value: ")", Line: line, Col: col}, nil
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Line: line, Col: col}, nil
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Line: line, Col: col}, nil
	case '.':
		l.advance()
		return Token{Type: TokenDot, Value: ".", Line: line, Col: col}, nil
	case '>':
		l.advance()
		return Token{Type: TokenGT, Value: ">", Line: line, Col: col}, nil
	case '<':
		l.advance()
		if l.peek() == '>' {
			l.advance()
			return Token{Type: TokenLTGT, Value: "<>", Line: line, Col: col}, nil
		}
		return Token{Type: TokenLT, Value: "<", Line: line, Col: col}, nil
	case '-':
		l.advance()
		return Token{Type: TokenMinus, Value: "-", Line: line, Col: col}, nil
	}

	// 文字列リテラル（シングルクォート）
	if ch == '\'' {
		return l.readString()
	}

	// バッククォート式
	if ch == '`' {
		return l.readBacktick()
	}

	// 数値
	if unicode.IsDigit(ch) {
		return l.readNumber()
	}

	// 識別子 / キーワード
	if unicode.IsLetter(ch) || ch == '_' {
		return l.readIdentOrKeyword()
	}

	// ダブルクォート文字列
	if ch == '"' {
		return l.readDoubleQuotedString()
	}

	return Token{}, fmt.Errorf("L%d:C%d: 予期しない文字 '%c'", l.line, l.col, ch)
}

func (l *Lexer) readString() (Token, error) {
	line, col := l.line, l.col
	l.advance() // 最初の '

	// 三重クォート '''...''' のチェック
	if l.peek() == '\'' && l.peekAt(1) == '\'' {
		l.advance() // 2つ目の '
		l.advance() // 3つ目の '
		return l.readTripleQuoteString(line, col)
	}

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '\'' {
			l.advance()
			return Token{Type: TokenString, Value: sb.String(), Line: line, Col: col}, nil
		}
		if ch == '\\' {
			l.advance()
			escaped := l.advance()
			switch escaped {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			case '\'':
				sb.WriteRune('\'')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(escaped)
			}
			continue
		}
		if ch == '\n' {
			return Token{}, fmt.Errorf("L%d:C%d: 文字列が閉じられていません", line, col)
		}
		sb.WriteRune(l.advance())
	}
	return Token{}, fmt.Errorf("L%d:C%d: 文字列が閉じられていません", line, col)
}

func (l *Lexer) readTripleQuoteString(line, col int) (Token, error) {
	var sb strings.Builder
	for l.pos < len(l.input) {
		if l.peek() == '\'' && l.peekAt(1) == '\'' && l.peekAt(2) == '\'' {
			l.advance()
			l.advance()
			l.advance()
			// 先頭と末尾の改行を除去
			s := sb.String()
			s = strings.TrimPrefix(s, "\n")
			s = strings.TrimRight(s, " \t\n")
			return Token{Type: TokenString, Value: s, Line: line, Col: col}, nil
		}
		sb.WriteRune(l.advance())
	}
	return Token{}, fmt.Errorf("L%d:C%d: 三重クォート文字列が閉じられていません", line, col)
}

func (l *Lexer) readDoubleQuotedString() (Token, error) {
	line, col := l.line, l.col
	l.advance() // 最初の "

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			return Token{Type: TokenString, Value: sb.String(), Line: line, Col: col}, nil
		}
		if ch == '\\' {
			l.advance()
			escaped := l.advance()
			switch escaped {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(escaped)
			}
			continue
		}
		if ch == '\n' {
			return Token{}, fmt.Errorf("L%d:C%d: 文字列が閉じられていません", line, col)
		}
		sb.WriteRune(l.advance())
	}
	return Token{}, fmt.Errorf("L%d:C%d: 文字列が閉じられていません", line, col)
}

func (l *Lexer) readBacktick() (Token, error) {
	line, col := l.line, l.col
	l.advance() // `
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '`' {
			l.advance()
			return Token{Type: TokenBacktick, Value: sb.String(), Line: line, Col: col}, nil
		}
		sb.WriteRune(l.advance())
	}
	return Token{}, fmt.Errorf("L%d:C%d: バッククォートが閉じられていません", line, col)
}

func (l *Lexer) readNumber() (Token, error) {
	line, col := l.line, l.col
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if unicode.IsDigit(ch) || ch == '.' {
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	return Token{Type: TokenNumber, Value: sb.String(), Line: line, Col: col}, nil
}

func (l *Lexer) readIdentOrKeyword() (Token, error) {
	line, col := l.line, l.col
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	word := sb.String()
	lower := strings.ToLower(word)

	if tt, ok := keywords[lower]; ok {
		return Token{Type: tt, Value: word, Line: line, Col: col}, nil
	}

	return Token{Type: TokenIdent, Value: word, Line: line, Col: col}, nil
}
