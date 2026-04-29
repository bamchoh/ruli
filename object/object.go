package object

import "fmt"

type ObjectType string

type Object interface {
	Type() ObjectType
	Inspect() string
}

const (
	INTEGER_OBJ  = "INTEGER"
	REAL_OBJ     = "REAL"
	BOOL_OBJ     = "BOOL"
	NULL_OBJ     = "NULL"
	BREAK_OBJ    = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
)

type Integer struct {
	Value int
}

func (i *Integer) Type() ObjectType {
	return INTEGER_OBJ
}

func (i *Integer) Inspect() string {
	return fmt.Sprintf("%d", i.Value)
}

type Real struct {
	Value float64
}

func (r *Real) Type() ObjectType {
	return REAL_OBJ
}

func (r *Real) Inspect() string {
	return fmt.Sprintf("%f", r.Value)
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType {
	return BOOL_OBJ
}

func (b *Boolean) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}

type Null struct{}

func (n *Null) Type() ObjectType {
	return NULL_OBJ
}

func (n *Null) Inspect() string {
	return "nil"
}

type BreakSignal struct{}

func (b *BreakSignal) Type() ObjectType { return BREAK_OBJ }
func (b *BreakSignal) Inspect() string  { return "break" }

type ContinueSignal struct{}

func (c *ContinueSignal) Type() ObjectType { return CONTINUE_OBJ }
func (c *ContinueSignal) Inspect() string  { return "continue" }
