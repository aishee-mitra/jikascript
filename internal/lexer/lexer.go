package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

// Lexer converts source text into tokens.
type Lexer struct {
	source []rune
	line   int
	column int
	pos    int
}

// New creates a lexer for source text.
func New(source string) *Lexer {
	return &Lexer{
		source: []rune(source),
		line:   1,
		column: 1,
	}
}

// NextToken reads and returns the next token.
func (l *Lexer) NextToken() Token {
	for {
		if l.pos >= len(l.source) {
			return Token{Type: EOF, Literal: "", Line: l.line, Column: l.column}
		}

		ch := l.source[l.pos]
		if unicode.IsSpace(ch) {
			l.consumeWhitespace(ch)
			continue
		}

		if ch == '/' && l.peek() == '/' {
			l.consumeSingleLineComment()
			continue
		}

		if ch == '/' && l.peek() == '*' {
			l.consumeMultiLineComment()
			continue
		}

		if ch == '"' || ch == '\'' {
			return l.readString(ch)
		}

		if unicode.IsDigit(ch) {
			return l.readNumber()
		}

		if unicode.IsLetter(ch) || ch == '_' {
			return l.readIdentifier()
		}

		if punct, ok := SingleCharPunctuators[byte(ch)]; ok {
			startLine, startColumn := l.line, l.column
			l.advance()
			return Token{Type: PUNCT, Literal: punct, Line: startLine, Column: startColumn}
		}

		if op, ok := l.readDoubleOperator(); ok {
			startLine, startColumn := l.line, l.column
			return Token{Type: OPERATOR, Literal: op, Line: startLine, Column: startColumn}
		}

		if op, ok := SingleCharOperators[byte(ch)]; ok {
			startLine, startColumn := l.line, l.column
			l.advance()
			return Token{Type: OPERATOR, Literal: op, Line: startLine, Column: startColumn}
		}

		startLine, startColumn := l.line, l.column
		l.advance()
		return Token{Type: ILLEGAL, Literal: string(ch), Line: startLine, Column: startColumn}
	}
}

func (l *Lexer) readIdentifier() Token {
	startLine, startColumn := l.line, l.column
	var b strings.Builder
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			b.WriteRune(ch)
			l.advance()
			continue
		}
		break
	}
	literal := b.String()
	if tt, ok := Keywords[literal]; ok {
		return Token{Type: tt, Literal: literal, Line: startLine, Column: startColumn}
	}
	return Token{Type: IDENT, Literal: literal, Line: startLine, Column: startColumn}
}

func (l *Lexer) readNumber() Token {
	startLine, startColumn := l.line, l.column
	var b strings.Builder
	seenDot := false
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		if unicode.IsDigit(ch) {
			b.WriteRune(ch)
			l.advance()
			continue
		}
		if ch == '.' && !seenDot {
			b.WriteRune(ch)
			seenDot = true
			l.advance()
			continue
		}
		break
	}
	return Token{Type: NUMBER, Literal: b.String(), Line: startLine, Column: startColumn}
}

func (l *Lexer) readString(quote rune) Token {
	startLine, startColumn := l.line, l.column
	l.advance()
	var b strings.Builder
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.source) {
				return Token{Type: ILLEGAL, Literal: "unterminated string", Line: startLine, Column: startColumn}
			}
			escaped := l.source[l.pos]
			b.WriteRune(escaped)
			l.advance()
			continue
		}
		if ch == quote {
			l.advance()
			return Token{Type: STRING, Literal: b.String(), Line: startLine, Column: startColumn}
		}
		b.WriteRune(ch)
		l.advance()
	}
	return Token{Type: ILLEGAL, Literal: "unterminated string", Line: startLine, Column: startColumn}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.source) {
		return l.source[l.pos+1]
	}
	return 0
}

func (l *Lexer) consumeWhitespace(ch rune) {
	l.advance()
}

func (l *Lexer) consumeSingleLineComment() {
	l.advance()
	l.advance()
	for l.pos < len(l.source) {
		if l.source[l.pos] == '\n' {
			l.advance()
			return
		}
		l.advance()
	}
}

func (l *Lexer) consumeMultiLineComment() {
	l.advance()
	l.advance()
	for l.pos < len(l.source)-1 {
		if l.source[l.pos] == '*' && l.source[l.pos+1] == '/' {
			l.advance()
			l.advance()
			return
		}
		if l.source[l.pos] == '\n' {
			l.advance()
			continue
		}
		l.advance()
	}
	panic(fmt.Sprintf("unterminated comment at line %d, column %d", l.line, l.column))
}

func (l *Lexer) readDoubleOperator() (string, bool) {
	if l.pos+1 >= len(l.source) {
		return "", false
	}
	pair := string([]rune{l.source[l.pos], l.source[l.pos+1]})
	switch pair {
	case "==", "!=", "<=", ">=", "&&", "||", "++", "--", "=>":
		l.advance()
		l.advance()
		return pair, true
	default:
		return "", false
	}
}

func (l *Lexer) advance() {
	if l.pos >= len(l.source) {
		return
	}
	if l.source[l.pos] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.pos++
}
