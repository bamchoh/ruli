package object

import (
	"fmt"
	"ruli/ast"
	"strings"
)

type Environment struct {
	store map[string]Object
	outer *Environment
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Object)}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	v, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return v, ok
}

func (e *Environment) Set(name string, val Object) {
	e.store[name] = val
}

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
	FUNCTION_OBJ = "FUNCTION"
	RETURN_OBJ   = "RETURN"
	STRING_OBJ   = "STRING"
	ARRAY_OBJ    = "ARRAY"
	ERROR_OBJ    = "ERROR"
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

type Function struct {
	Parameters []ast.Parameter
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType {
	return FUNCTION_OBJ
}

func (f *Function) Inspect() string {
	return "function"
}

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType {
	return RETURN_OBJ
}

func (rv *ReturnValue) Inspect() string {
	return rv.Value.Inspect()
}

type String struct {
	Value string
}

func (s *String) Type() ObjectType {
	return "STRING"
}

func (s *String) Inspect() string {
	return s.Value
}

type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType {
	return ARRAY_OBJ
}

func (a *Array) Inspect() string {
	var out strings.Builder

	out.WriteString("[")

	for i, e := range a.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(e.Inspect())
	}

	out.WriteString("]")

	return out.String()
}

type Error struct {
	Message string

	Line   int
	Column int
}

func (e *Error) Type() ObjectType {
	return ERROR_OBJ
}

func (e *Error) Inspect() string {
	return fmt.Sprintf(
		"ERROR: line %d, column %d: %s",
		e.Line,
		e.Column,
		e.Message,
	)
}
