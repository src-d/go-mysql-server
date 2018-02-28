package plan

import (
	"io"

	"gopkg.in/src-d/go-mysql-server.v0/sql"
)

// CrossJoin is a cross join between two tables.
type CrossJoin struct {
	BinaryNode
}

// NewCrossJoin creates a new cross join node from two tables.
func NewCrossJoin(left sql.Node, right sql.Node) *CrossJoin {
	return &CrossJoin{
		BinaryNode: BinaryNode{
			Left:  left,
			Right: right,
		},
	}
}

// Schema implements the Node interface.
func (p *CrossJoin) Schema() sql.Schema {
	return append(p.Left.Schema(), p.Right.Schema()...)
}

// Resolved implements the Resolvable interface.
func (p *CrossJoin) Resolved() bool {
	return p.Left.Resolved() && p.Right.Resolved()
}

// RowIter implements the Node interface.
func (p *CrossJoin) RowIter() (sql.RowIter, error) {
	li, err := p.Left.RowIter()
	if err != nil {
		return nil, err
	}

	ri, err := p.Right.RowIter()
	if err != nil {
		return nil, err
	}

	return &crossJoinIterator{
		li: li,
		ri: ri,
	}, nil
}

// TransformUp implements the Transformable interface.
func (p *CrossJoin) TransformUp(f func(sql.Node) sql.Node) sql.Node {
	return f(NewCrossJoin(p.Left.TransformUp(f), p.Right.TransformUp(f)))
}

// TransformExpressionsUp implements the Transformable interface.
func (p *CrossJoin) TransformExpressionsUp(f func(sql.Expression) sql.Expression) sql.Node {
	return NewCrossJoin(
		p.Left.TransformExpressionsUp(f),
		p.Right.TransformExpressionsUp(f),
	)
}

type crossJoinIterator struct {
	li sql.RowIter
	ri sql.RowIter

	// TODO use a method to reset right iterator in order to not duplicate rows into memory
	rightRows []sql.Row
	index     int
	leftRow   sql.Row
}

func (i *crossJoinIterator) Next() (sql.Row, error) {
	if len(i.rightRows) == 0 {
		if err := i.fillRows(); err != io.EOF {
			return nil, err
		}

		if len(i.rightRows) == 0 {
			return nil, io.EOF
		}
	}

	if i.leftRow == nil {
		lr, err := i.li.Next()
		if err != nil {
			return nil, err
		}

		i.index = 0
		i.leftRow = lr
	}

	row := append(i.leftRow, i.rightRows[i.index]...)
	i.index++
	if i.index >= len(i.rightRows) {
		i.index = 0
		i.leftRow = nil
	}

	return row, nil
}

func (i *crossJoinIterator) Close() error {
	if err := i.li.Close(); err != nil {
		_ = i.ri.Close()
		return err
	}

	return i.ri.Close()
}

func (i *crossJoinIterator) fillRows() error {
	for {
		rr, err := i.ri.Next()
		if err != nil {
			return err
		}

		i.rightRows = append(i.rightRows, rr)
	}
}
