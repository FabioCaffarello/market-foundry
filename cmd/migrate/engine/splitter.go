package engine

import "strings"

// SplitStatements splits a migration file's content into individual
// SQL statements, because the ClickHouse Go driver (clickhouse-go/v2)
// accepts one statement per ExecContext — multi-statement files fail
// with code 62 "Multi-statements are not allowed" (surfaced in
// H-6.d.1; runner enhancement deferred to H-6.f.1, Decisão #5).
//
// The split is ';'-aware: separators inside single-quoted strings,
// double-quoted or backtick-quoted identifiers, `--` line comments,
// and `/* */` block comments do not terminate a statement. Each
// returned statement keeps its surrounding comments verbatim (the
// driver accepts comment-prefixed statements — every 000–013 file
// ships a comment header today); segments with no executable content
// (whitespace/comments only, e.g. after a trailing ';') are dropped.
func SplitStatements(content string) []string {
	var statements []string
	var current strings.Builder

	const (
		code = iota
		lineComment
		blockComment
		singleQuote
		doubleQuote
		backtick
	)
	state := code

	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}

		switch state {
		case code:
			switch {
			case ch == '-' && next == '-':
				state = lineComment
			case ch == '/' && next == '*':
				state = blockComment
			case ch == '\'':
				state = singleQuote
			case ch == '"':
				state = doubleQuote
			case ch == '`':
				state = backtick
			case ch == ';':
				statements = appendExecutable(statements, current.String())
				current.Reset()
				continue
			}
		case lineComment:
			if ch == '\n' {
				state = code
			}
		case blockComment:
			if ch == '*' && next == '/' {
				current.WriteRune(ch)
				i++
				ch = runes[i]
				state = code
			}
		case singleQuote:
			switch {
			case ch == '\\':
				if i+1 < len(runes) {
					current.WriteRune(ch)
					i++
					ch = runes[i]
				}
			case ch == '\'':
				state = code
			}
		case doubleQuote:
			if ch == '"' {
				state = code
			}
		case backtick:
			if ch == '`' {
				state = code
			}
		}

		current.WriteRune(ch)
	}
	return appendExecutable(statements, current.String())
}

// appendExecutable appends stmt (trimmed) when it contains executable
// content — i.e. anything beyond whitespace and comments.
func appendExecutable(statements []string, stmt string) []string {
	trimmed := strings.TrimSpace(stmt)
	if trimmed == "" || !hasExecutableContent(trimmed) {
		return statements
	}
	return append(statements, trimmed)
}

// hasExecutableContent reports whether s contains anything other than
// whitespace, `--` line comments, and `/* */` block comments.
func hasExecutableContent(s string) bool {
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		switch {
		case ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n':
			continue
		case ch == '-' && next == '-':
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case ch == '/' && next == '*':
			i += 2
			for i+1 < len(runes) && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i++ // skip the closing '/'
		default:
			return true
		}
	}
	return false
}
