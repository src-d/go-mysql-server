package expression

import (
	"bytes"
	"reflect"
	"time"

	"gopkg.in/src-d/go-mysql-server.v0/sql"
)

// IsBinary is a function that returns whether a blob is binary or not.
type IsBinary struct {
	UnaryExpression
}

// NewIsBinary creates a new IsBinary expression.
func NewIsBinary(e sql.Expression) sql.Expression {
	return &IsBinary{UnaryExpression{Child: e}}
}

// Eval implements the Expression interface.
func (ib *IsBinary) Eval(row sql.Row) (interface{}, error) {
	v, err := ib.Child.Eval(row)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return false, nil
	}

	blob, err := sql.Blob.Convert(v)
	if err != nil {
		return nil, err
	}

	return isBinary(blob.([]byte)), nil
}

// Name implements the Expression interface.
func (ib *IsBinary) Name() string {
	return "is_binary"
}

// TransformUp implements the Expression interface.
func (ib *IsBinary) TransformUp(f func(sql.Expression) sql.Expression) sql.Expression {
	return NewIsBinary(ib.Child.TransformUp(f))
}

// Type implements the Expression interface.
func (ib *IsBinary) Type() sql.Type {
	return sql.Boolean
}

const sniffLen = 8000

// isBinary detects if data is a binary value based on:
// http://git.kernel.org/cgit/git/git.git/tree/xdiff-interface.c?id=HEAD#n198
func isBinary(data []byte) bool {
	if len(data) > sniffLen {
		data = data[:sniffLen]
	}

	if bytes.IndexByte(data, byte(0)) == -1 {
		return false
	}

	return true
}

// Substring is a function to return a part of a string.
// This function behaves as the homonym MySQL function.
// Since Go strings are UTF8, this function does not return a direct sub
// string str[start:start+length], instead returns the substring of rune
// s. That is, "á"[0:1] does not return a partial unicode glyph, but "á"
// itself.
type Substring struct {
	str   sql.Expression
	start sql.Expression
	len   sql.Expression
}

// NewSubstring creates a new substring UDF.
func NewSubstring(args ...sql.Expression) (sql.Expression, error) {
	var str, start, ln sql.Expression
	switch len(args) {
	case 2:
		str = args[0]
		start = args[1]
		ln = nil
	case 3:
		str = args[0]
		start = args[1]
		ln = args[2]
	default:
		return nil, sql.ErrInvalidArgumentNumber.New("2 or 3", len(args))
	}
	return &Substring{str, start, ln}, nil
}

// Eval implements the Expression interface.
func (s *Substring) Eval(row sql.Row) (interface{}, error) {
	str, err := s.str.Eval(row)
	if err != nil {
		return nil, err
	}

	var text []rune
	switch str := str.(type) {
	case string:
		text = []rune(str)
	case []byte:
		text = []rune(string(str))
	case nil:
		return nil, nil
	default:
		return nil, sql.ErrInvalidType.New(reflect.TypeOf(str).String())
	}

	start, err := s.start.Eval(row)
	if err != nil {
		return nil, err
	}

	start, err = sql.Int64.Convert(start)
	if err != nil {
		return nil, err
	}

	var length int64
	runeCount := int64(len(text))
	if s.len != nil {
		len, err := s.len.Eval(row)
		if err != nil {
			return nil, err
		}

		len, err = sql.Int64.Convert(len)
		if err != nil {
			return nil, err
		}

		length = len.(int64)
	} else {
		length = runeCount
	}

	var startIdx int64
	if start := start.(int64); start < 0 {
		startIdx = runeCount + start
	} else {
		startIdx = start - 1
	}

	if startIdx < 0 || startIdx >= runeCount || length <= 0 {
		return "", nil
	}

	if startIdx+length > runeCount {
		length = int64(runeCount) - startIdx
	}

	return string(text[startIdx : startIdx+length]), nil
}

// IsNullable implements the Expression interface.
func (s *Substring) IsNullable() bool { return true }

// Name implements the Expression interface.
func (Substring) Name() string {
	return "substring"
}

// Resolved implements the Expression interface.
func (Substring) Resolved() bool { return true }

// Type implements the Expression interface.
func (Substring) Type() sql.Type { return sql.Text }

// TransformUp implements the Expression interface.
func (s *Substring) TransformUp(f func(sql.Expression) sql.Expression) sql.Expression {
	// It is safe to omit the errors of NewSubstring here because to be able to call
	// this method, you need a valid instance of Substring, so the arity must be correct
	// and that's the only error NewSubstring can return.
	var sub sql.Expression
	if s.len != nil {
		sub, _ = NewSubstring(s.str.TransformUp(f), s.start.TransformUp(f), s.len.TransformUp(f))
	} else {
		sub, _ = NewSubstring(s.str.TransformUp(f), s.start.TransformUp(f))
	}
	return f(sub)
}

// Year is a function that returns the year of a date.
type Year struct {
	UnaryExpression
}

// NewYear creates a new Year UDF.
func NewYear(date sql.Expression) sql.Expression {
	return &Year{UnaryExpression{Child: date}}
}

// Name implements the Expression interface.
func (y *Year) Name() string { return "year" }

// Type implements the Expression interface.
func (y *Year) Type() sql.Type { return sql.Int32 }

// Eval implements the Expression interface.
func (y *Year) Eval(row sql.Row) (interface{}, error) {
	val, err := y.Child.Eval(row)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	date, err := sql.Date.Convert(val)
	if err != nil {
		return nil, err
	}

	return int32(date.(time.Time).Year()), nil
}

// TransformUp implements the Expression interface.
func (y *Year) TransformUp(f func(sql.Expression) sql.Expression) sql.Expression {
	return f(NewYear(y.Child.TransformUp(f)))
}
