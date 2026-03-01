package parser

import (
	"fmt"
	"strings"

	"github.com/nobmurakita/dbml-doc/model"
)

// Parser はDBMLの構文解析器
type Parser struct {
	tokens []Token
	pos    int
}

// Parse はDBMLテキストを解析してDBMLモデルを返す
func Parse(input string) (*model.DBML, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("字句解析エラー: %w", err)
	}

	p := &Parser{tokens: tokens, pos: 0}
	return p.parse()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.peek()
	if tok.Type != tt {
		return Token{}, fmt.Errorf("L%d:C%d: %s を期待しましたが %s (%q) でした",
			tok.Line, tok.Col, tt, tok.Type, tok.Value)
	}
	return p.advance(), nil
}

func (p *Parser) skipNewlines() {
	for p.peek().Type == TokenNewline {
		p.advance()
	}
}

func (p *Parser) parse() (*model.DBML, error) {
	dbml := &model.DBML{}

	for {
		p.skipNewlines()
		tok := p.peek()

		switch tok.Type {
		case TokenEOF:
			return dbml, nil
		case TokenProject:
			proj, err := p.parseProject()
			if err != nil {
				return nil, err
			}
			dbml.Project = proj
		case TokenTable:
			table, err := p.parseTable()
			if err != nil {
				return nil, err
			}
			dbml.Tables = append(dbml.Tables, *table)
		case TokenRef:
			ref, err := p.parseRef()
			if err != nil {
				return nil, err
			}
			dbml.Refs = append(dbml.Refs, *ref)
		case TokenEnum:
			enum, err := p.parseEnum()
			if err != nil {
				return nil, err
			}
			dbml.Enums = append(dbml.Enums, *enum)
		case TokenTableGroup:
			tg, err := p.parseTableGroup()
			if err != nil {
				return nil, err
			}
			dbml.TableGroups = append(dbml.TableGroups, *tg)
		default:
			return nil, fmt.Errorf("L%d:C%d: 予期しないトークン %s (%q)",
				tok.Line, tok.Col, tok.Type, tok.Value)
		}
	}
}

// parseProject は Project ブロックを解析する
func (p *Parser) parseProject() (*model.Project, error) {
	p.advance() // Project

	proj := &model.Project{}

	// プロジェクト名（任意）
	if p.peek().Type == TokenIdent || p.peek().Type == TokenString {
		proj.Name = p.advance().Value
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	for {
		p.skipNewlines()
		tok := p.peek()

		if tok.Type == TokenRBrace {
			p.advance()
			return proj, nil
		}

		switch {
		case tok.Type == TokenIdent && strings.ToLower(tok.Value) == "database_type":
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return nil, err
			}
			val, err := p.expect(TokenString)
			if err != nil {
				return nil, err
			}
			proj.DatabaseType = val.Value

		case tok.Type == TokenNote:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return nil, err
			}
			val, err := p.expect(TokenString)
			if err != nil {
				return nil, err
			}
			proj.Note = val.Value

		default:
			// 未知のプロパティはスキップ
			p.advance()
			if p.peek().Type == TokenColon {
				p.advance()
				p.advance() // 値をスキップ
			}
		}
	}
}

// parseTable は Table ブロックを解析する
func (p *Parser) parseTable() (*model.Table, error) {
	p.advance() // Table

	table := &model.Table{}

	// テーブル名: [schema.]name
	name, err := p.parseQualifiedName()
	if err != nil {
		return nil, err
	}
	if idx := strings.Index(name, "."); idx >= 0 {
		table.Schema = name[:idx]
		table.Name = name[idx+1:]
	} else {
		table.Name = name
	}

	// alias: as alias_name
	if p.peek().Type == TokenAs {
		p.advance()
		alias := p.advance()
		table.Alias = alias.Value
	}

	// テーブル設定 [headercolor: ...]
	if p.peek().Type == TokenLBracket {
		p.parseTableSettings()
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	// カラムとインデックスの解析
	for {
		p.skipNewlines()
		tok := p.peek()

		if tok.Type == TokenRBrace {
			p.advance()
			break
		}

		if tok.Type == TokenIndexes {
			indexes, err := p.parseIndexes()
			if err != nil {
				return nil, err
			}
			table.Indexes = indexes
			continue
		}

		if tok.Type == TokenNote {
			note, err := p.parseNoteProperty()
			if err != nil {
				return nil, err
			}
			table.Note = note
			continue
		}

		col, err := p.parseColumn()
		if err != nil {
			return nil, err
		}
		table.Columns = append(table.Columns, *col)
	}

	return table, nil
}

// parseQualifiedName は schema.name または name を解析する
func (p *Parser) parseQualifiedName() (string, error) {
	tok := p.peek()
	if tok.Type != TokenIdent && tok.Type != TokenString {
		return "", fmt.Errorf("L%d:C%d: 識別子を期待しましたが %s でした", tok.Line, tok.Col, tok.Type)
	}
	name := p.advance().Value

	if p.peek().Type == TokenDot {
		p.advance()
		tok2 := p.advance()
		name = name + "." + tok2.Value
	}

	return name, nil
}

// parseTableSettings は [headercolor: ...] などのテーブル設定を解析する
func (p *Parser) parseTableSettings() {
	p.advance() // [
	depth := 1
	for depth > 0 {
		tok := p.advance()
		switch tok.Type {
		case TokenLBracket:
			depth++
		case TokenRBracket:
			depth--
		case TokenEOF:
			return
		}
	}
}

// isIdentLike はトークンがカラム名などの識別子として使えるかを判定する
// DBMLではキーワードもカラム名として使える
func isIdentLike(t TokenType) bool {
	switch t {
	case TokenIdent, TokenString,
		TokenName, TokenType_, TokenNote, TokenDefault, TokenNull,
		TokenNot, TokenUnique, TokenPK, TokenPrimary, TokenKey,
		TokenIncrement, TokenDelete, TokenUpdate, TokenCascade,
		TokenRestrict, TokenAs, TokenRef:
		return true
	}
	return false
}

// parseColumn はカラム定義を解析する
func (p *Parser) parseColumn() (*model.Column, error) {
	col := &model.Column{}

	// カラム名（キーワードもカラム名として使用可能）
	nameTok := p.peek()
	if isIdentLike(nameTok.Type) {
		col.Name = p.advance().Value
	} else {
		return nil, fmt.Errorf("L%d:C%d: カラム名を期待しましたが %s (%q) でした",
			nameTok.Line, nameTok.Col, nameTok.Type, nameTok.Value)
	}

	// 型
	colType, err := p.parseColumnType()
	if err != nil {
		return nil, err
	}
	col.Type = colType

	// カラム設定 [pk, not null, ...]
	if p.peek().Type == TokenLBracket {
		if err := p.parseColumnSettings(col); err != nil {
			return nil, err
		}
	}

	return col, nil
}

// parseColumnType はカラムの型定義を解析する
func (p *Parser) parseColumnType() (string, error) {
	tok := p.peek()
	if !isIdentLike(tok.Type) {
		return "", fmt.Errorf("L%d:C%d: 型名を期待しましたが %s (%q) でした",
			tok.Line, tok.Col, tok.Type, tok.Value)
	}
	typeName := p.advance().Value

	// varchar(255) のようなパラメータ付き型
	if p.peek().Type == TokenLParen {
		p.advance()
		var params []string
		for {
			ptok := p.peek()
			if ptok.Type == TokenRParen {
				p.advance()
				break
			}
			if ptok.Type == TokenComma {
				p.advance()
				continue
			}
			params = append(params, p.advance().Value)
		}
		typeName = typeName + "(" + strings.Join(params, ", ") + ")"
	}

	return typeName, nil
}

// parseColumnSettings はカラムの設定 [...] を解析する
func (p *Parser) parseColumnSettings(col *model.Column) error {
	p.advance() // [

	for {
		p.skipNewlines()
		tok := p.peek()

		if tok.Type == TokenRBracket {
			p.advance()
			return nil
		}

		if tok.Type == TokenComma {
			p.advance()
			continue
		}

		switch {
		case tok.Type == TokenPK:
			p.advance()
			col.PrimaryKey = true

		case tok.Type == TokenPrimary:
			p.advance()
			if p.peek().Type == TokenKey {
				p.advance()
			}
			col.PrimaryKey = true

		case tok.Type == TokenNot:
			p.advance()
			if p.peek().Type == TokenNull {
				p.advance()
			}
			col.NotNull = true

		case tok.Type == TokenNull:
			p.advance()
			// null許可（デフォルト）

		case tok.Type == TokenUnique:
			p.advance()
			col.Unique = true

		case tok.Type == TokenIncrement:
			p.advance()
			col.Increment = true

		case tok.Type == TokenDefault:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			val := p.advance()
			s := val.Value
			col.Default = &s

		case tok.Type == TokenNote:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			note := p.advance()
			col.Note = note.Value

		case tok.Type == TokenRef:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			ref, err := p.parseInlineRef()
			if err != nil {
				return err
			}
			col.Ref = ref

		default:
			// 未知の設定はスキップ
			p.advance()
			if p.peek().Type == TokenColon {
				p.advance()
				p.advance()
			}
		}
	}
}

// parseInlineRef はインラインのリレーション定義を解析する
func (p *Parser) parseInlineRef() (*model.InlineRef, error) {
	ref := &model.InlineRef{}

	// 関係タイプ: >, <, -, <>
	tok := p.peek()
	switch tok.Type {
	case TokenGT:
		ref.Type = ">"
		p.advance()
	case TokenLT:
		ref.Type = "<"
		p.advance()
	case TokenMinus:
		ref.Type = "-"
		p.advance()
	case TokenLTGT:
		ref.Type = "<>"
		p.advance()
	default:
		return nil, fmt.Errorf("L%d:C%d: リレーション種別（>, <, -, <>）を期待しました", tok.Line, tok.Col)
	}

	// テーブル.カラム
	tableName := p.advance().Value
	if p.peek().Type == TokenDot {
		p.advance()
		colName := p.advance().Value
		// schema.table.column の場合
		if p.peek().Type == TokenDot {
			p.advance()
			realCol := p.advance().Value
			ref.Table = tableName + "." + colName
			ref.Column = realCol
		} else {
			ref.Table = tableName
			ref.Column = colName
		}
	}

	return ref, nil
}

// parseNoteProperty は Note: '...' を解析する
func (p *Parser) parseNoteProperty() (string, error) {
	p.advance() // Note

	tok := p.peek()
	if tok.Type == TokenColon {
		p.advance()
		val, err := p.expect(TokenString)
		if err != nil {
			return "", err
		}
		return val.Value, nil
	}

	// Note { '...' } 形式
	if tok.Type == TokenLBrace {
		p.advance()
		p.skipNewlines()
		val, err := p.expect(TokenString)
		if err != nil {
			return "", err
		}
		p.skipNewlines()
		if _, err := p.expect(TokenRBrace); err != nil {
			return "", err
		}
		return val.Value, nil
	}

	return "", fmt.Errorf("L%d:C%d: Note の後に : または { を期待しました", tok.Line, tok.Col)
}

// parseIndexes は indexes ブロックを解析する
func (p *Parser) parseIndexes() ([]model.Index, error) {
	p.advance() // indexes

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	var indexes []model.Index

	for {
		p.skipNewlines()
		if p.peek().Type == TokenRBrace {
			p.advance()
			return indexes, nil
		}

		idx, err := p.parseIndex()
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, *idx)
	}
}

// parseIndex は個々のインデックス定義を解析する
func (p *Parser) parseIndex() (*model.Index, error) {
	idx := &model.Index{}

	// 単一カラムまたは複合カラム
	if p.peek().Type == TokenLParen {
		// 複合インデックス: (col1, col2)
		p.advance()
		for {
			if p.peek().Type == TokenRParen {
				p.advance()
				break
			}
			if p.peek().Type == TokenComma {
				p.advance()
				continue
			}
			ic, err := p.parseIndexColumn()
			if err != nil {
				return nil, err
			}
			idx.Columns = append(idx.Columns, *ic)
		}
	} else {
		// 単一カラムインデックス
		ic, err := p.parseIndexColumn()
		if err != nil {
			return nil, err
		}
		idx.Columns = append(idx.Columns, *ic)
	}

	// インデックス設定 [name: ..., unique, ...]
	if p.peek().Type == TokenLBracket {
		if err := p.parseIndexSettings(idx); err != nil {
			return nil, err
		}
	}

	return idx, nil
}

// parseIndexColumn はインデックスカラムを解析する
func (p *Parser) parseIndexColumn() (*model.IndexColumn, error) {
	tok := p.peek()
	if tok.Type == TokenBacktick {
		p.advance()
		return &model.IndexColumn{Expression: tok.Value}, nil
	}
	if isIdentLike(tok.Type) {
		p.advance()
		return &model.IndexColumn{Name: tok.Value}, nil
	}
	return nil, fmt.Errorf("L%d:C%d: インデックスカラム名を期待しました", tok.Line, tok.Col)
}

// parseIndexSettings はインデックスの設定を解析する
func (p *Parser) parseIndexSettings(idx *model.Index) error {
	p.advance() // [

	for {
		p.skipNewlines()
		tok := p.peek()

		if tok.Type == TokenRBracket {
			p.advance()
			return nil
		}

		if tok.Type == TokenComma {
			p.advance()
			continue
		}

		switch {
		case tok.Type == TokenName:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			name := p.advance()
			idx.Name = name.Value

		case tok.Type == TokenUnique:
			p.advance()
			idx.Unique = true

		case tok.Type == TokenPK:
			p.advance()
			idx.PK = true

		case tok.Type == TokenType_:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			t := p.advance()
			idx.Type = t.Value

		case tok.Type == TokenNote:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			note := p.advance()
			idx.Note = note.Value

		default:
			p.advance()
			if p.peek().Type == TokenColon {
				p.advance()
				p.advance()
			}
		}
	}
}

// parseRef は Ref ブロックを解析する
func (p *Parser) parseRef() (*model.Ref, error) {
	p.advance() // Ref

	ref := &model.Ref{}

	// Ref名（任意）: Ref name: ...
	if p.peek().Type == TokenIdent || p.peek().Type == TokenString {
		next := p.peek()
		// 次がコロンなら名前付き
		saved := p.pos
		p.advance()
		if p.peek().Type == TokenColon {
			ref.Name = next.Value
			p.advance() // :
		} else {
			p.pos = saved
		}
	}

	// Ref: from > to の短縮形
	if p.peek().Type == TokenColon {
		p.advance()
	}

	// { ... } ブロック形式か一行形式かを判定
	if p.peek().Type == TokenLBrace {
		return p.parseRefBlock(ref)
	}

	return p.parseRefInline(ref)
}

// parseRefBlock は複数行Ref定義を解析する
func (p *Parser) parseRefBlock(ref *model.Ref) (*model.Ref, error) {
	p.advance() // {

	p.skipNewlines()

	// from_endpoint relation to_endpoint
	innerRef, err := p.parseRefInline(ref)
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return innerRef, nil
}

// parseRefInline は一行のRef定義を解析する
func (p *Parser) parseRefInline(ref *model.Ref) (*model.Ref, error) {
	// from endpoint
	from, err := p.parseRefEndpoint()
	if err != nil {
		return nil, err
	}
	ref.From = *from

	// relation type: >, <, -, <>
	tok := p.peek()
	switch tok.Type {
	case TokenGT:
		ref.Type = ">"
		p.advance()
	case TokenLT:
		ref.Type = "<"
		p.advance()
	case TokenMinus:
		ref.Type = "-"
		p.advance()
	case TokenLTGT:
		ref.Type = "<>"
		p.advance()
	default:
		return nil, fmt.Errorf("L%d:C%d: リレーション種別を期待しました", tok.Line, tok.Col)
	}

	// to endpoint
	to, err := p.parseRefEndpoint()
	if err != nil {
		return nil, err
	}
	ref.To = *to

	// オプションの設定 [delete: cascade, update: ...]
	if p.peek().Type == TokenLBracket {
		if err := p.parseRefSettings(ref); err != nil {
			return nil, err
		}
	}

	return ref, nil
}

// parseRefEndpoint はリレーションの端点を解析する
func (p *Parser) parseRefEndpoint() (*model.RefEndpoint, error) {
	ep := &model.RefEndpoint{}

	// table.column または schema.table.column
	// または table.(col1, col2) 複合キー
	name1 := p.advance().Value

	if p.peek().Type != TokenDot {
		return nil, fmt.Errorf("L%d:C%d: . を期待しました", p.peek().Line, p.peek().Col)
	}
	p.advance() // .

	// 複合カラム (col1, col2)
	if p.peek().Type == TokenLParen {
		ep.Table = name1
		p.advance() // (
		for {
			if p.peek().Type == TokenRParen {
				p.advance()
				break
			}
			if p.peek().Type == TokenComma {
				p.advance()
				continue
			}
			ep.Columns = append(ep.Columns, p.advance().Value)
		}
		return ep, nil
	}

	name2 := p.advance().Value

	// schema.table.column の場合
	if p.peek().Type == TokenDot {
		p.advance()
		if p.peek().Type == TokenLParen {
			// schema.table.(col1, col2)
			ep.Schema = name1
			ep.Table = name2
			p.advance() // (
			for {
				if p.peek().Type == TokenRParen {
					p.advance()
					break
				}
				if p.peek().Type == TokenComma {
					p.advance()
					continue
				}
				ep.Columns = append(ep.Columns, p.advance().Value)
			}
		} else {
			name3 := p.advance().Value
			ep.Schema = name1
			ep.Table = name2
			ep.Columns = []string{name3}
		}
	} else {
		ep.Table = name1
		ep.Columns = []string{name2}
	}

	return ep, nil
}

// parseRefSettings はRefの設定を解析する
func (p *Parser) parseRefSettings(ref *model.Ref) error {
	p.advance() // [

	for {
		p.skipNewlines()
		tok := p.peek()

		if tok.Type == TokenRBracket {
			p.advance()
			return nil
		}

		if tok.Type == TokenComma {
			p.advance()
			continue
		}

		switch {
		case tok.Type == TokenDelete:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			action, err := p.parseRefAction()
			if err != nil {
				return err
			}
			ref.OnDelete = action

		case tok.Type == TokenUpdate:
			p.advance()
			if _, err := p.expect(TokenColon); err != nil {
				return err
			}
			action, err := p.parseRefAction()
			if err != nil {
				return err
			}
			ref.OnUpdate = action

		default:
			p.advance()
			if p.peek().Type == TokenColon {
				p.advance()
				p.advance()
			}
		}
	}
}

// parseRefAction はRefのアクション（cascade, restrict等）を解析する
func (p *Parser) parseRefAction() (string, error) {
	tok := p.peek()
	switch tok.Type {
	case TokenCascade:
		p.advance()
		return "cascade", nil
	case TokenRestrict:
		p.advance()
		return "restrict", nil
	case TokenNull:
		p.advance()
		return "set null", nil
	case TokenDefault:
		p.advance()
		return "set default", nil
	case TokenIdent:
		val := strings.ToLower(tok.Value)
		p.advance()
		// "no action" のパース
		if val == "no" {
			if p.peek().Type == TokenIdent && strings.ToLower(p.peek().Value) == "action" {
				p.advance()
				return "no action", nil
			}
		}
		// "set null" / "set default" のパース
		if val == "set" {
			next := p.peek()
			if next.Type == TokenNull {
				p.advance()
				return "set null", nil
			}
			if next.Type == TokenDefault {
				p.advance()
				return "set default", nil
			}
		}
		return val, nil
	default:
		return "", fmt.Errorf("L%d:C%d: リレーションアクションを期待しました", tok.Line, tok.Col)
	}
}

// parseEnum は Enum ブロックを解析する
func (p *Parser) parseEnum() (*model.Enum, error) {
	p.advance() // Enum

	enum := &model.Enum{}

	// Enum名: [schema.]name
	name, err := p.parseQualifiedName()
	if err != nil {
		return nil, err
	}
	if idx := strings.Index(name, "."); idx >= 0 {
		enum.Schema = name[:idx]
		enum.Name = name[idx+1:]
	} else {
		enum.Name = name
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	for {
		p.skipNewlines()
		if p.peek().Type == TokenRBrace {
			p.advance()
			return enum, nil
		}

		ev, err := p.parseEnumValue()
		if err != nil {
			return nil, err
		}
		enum.Values = append(enum.Values, *ev)
	}
}

// parseEnumValue はEnum値を解析する
func (p *Parser) parseEnumValue() (*model.EnumValue, error) {
	ev := &model.EnumValue{}

	tok := p.advance()
	ev.Name = tok.Value

	// オプションの設定 [note: '...']
	if p.peek().Type == TokenLBracket {
		p.advance()
		for {
			p.skipNewlines()
			if p.peek().Type == TokenRBracket {
				p.advance()
				break
			}
			if p.peek().Type == TokenComma {
				p.advance()
				continue
			}
			if p.peek().Type == TokenNote {
				p.advance()
				if _, err := p.expect(TokenColon); err != nil {
					return nil, err
				}
				note := p.advance()
				ev.Note = note.Value
			} else {
				p.advance()
			}
		}
	}

	return ev, nil
}

// parseTableGroup は TableGroup ブロックを解析する
func (p *Parser) parseTableGroup() (*model.TableGroup, error) {
	p.advance() // TableGroup

	tg := &model.TableGroup{}

	nameTok := p.advance()
	tg.Name = nameTok.Value

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	for {
		p.skipNewlines()
		if p.peek().Type == TokenRBrace {
			p.advance()
			return tg, nil
		}

		// テーブル名（schema.name または name）
		tok := p.advance()
		tableName := tok.Value
		if p.peek().Type == TokenDot {
			p.advance()
			tableName = tableName + "." + p.advance().Value
		}
		tg.Tables = append(tg.Tables, tableName)
	}
}
