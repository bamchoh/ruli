package object

import (
	"fmt"
	"ruli/ast"
	"sort"
	"strings"
)

type Environment struct {
	store map[string]Object
	outer *Environment

	// struct 型定義の登録 (型名 → 定義)
	typeDefs map[string]*StructDef

	// 宣言時に型名が明示された変数の記録 (実行時幅チェックに使用)
	varTypes map[string]string
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

// Set は現在のスコープに宣言する ( := / type / 関数定義 / 引数束縛 )

func (e *Environment) Set(name string, val Object) {
	e.store[name] = val
}

// Assign は既存変数を更新する ( = / ++ / -- )。
// 変数が見つからないスコープなら親をたどり、どこにも存在しなければ false を返す
func (e *Environment) Assign(name string, val Object) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		return e.outer.Assign(name, val)
	}
	return false
}

// DefineType は struct 型定義を現在のスコープに登録する
func (e *Environment) DefineType(name string, def *StructDef) {
	if e.typeDefs == nil {
		e.typeDefs = make(map[string]*StructDef)
	}
	e.typeDefs[name] = def
}

// GetTypeDef は struct 型定義を外側スコープも含めて検索する
func (e *Environment) GetTypeDef(name string) (*StructDef, bool) {
	if def, ok := e.typeDefs[name]; ok {
		return def, true
	}
	if e.outer != nil {
		return e.outer.GetTypeDef(name)
	}
	return nil, false
}

// DeclareTyped は型名を明示して現在スコープに宣言する (x : INT = 10 など)
func (e *Environment) DeclareTyped(name string, val Object, typeName string) {
	e.store[name] = val
	if e.varTypes == nil {
		e.varTypes = make(map[string]string)
	}
	e.varTypes[name] = typeName
}

// VarType は変数の宣言時型名を外側スコープも含めて検索する。
// 型推論による宣言 (:=) はここに登録されない
func (e *Environment) VarType(name string) (string, bool) {
	if t, ok := e.varTypes[name]; ok {
		return t, true
	}
	if e.outer != nil {
		return e.outer.VarType(name)
	}
	return "", false
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
	STRUCT_OBJ   = "STRUCT"
	MULTI_OBJ    = "MULTI"
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

// ReturnValue は return 文の制御信号。複数戻り値 (SPEC 5.3) もここに保持する
type ReturnValue struct {
	Values []Object
}

func (rv *ReturnValue) Type() ObjectType {
	return RETURN_OBJ
}

func (rv *ReturnValue) Inspect() string {
	var out strings.Builder
	out.WriteString("(")
	for i, v := range rv.Values {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(v.Inspect())
	}
	out.WriteString(")")
	return out.String()
}

// MultiValue は複数戻り値の関数呼び出し結果の一時値。
// 束縛先の変数リストと個数が合わなければエラー (SPEC 5.4)
type MultiValue struct {
	Values []Object
}

func (m *MultiValue) Type() ObjectType {
	return MULTI_OBJ
}

func (m *MultiValue) Inspect() string {
	var out strings.Builder
	out.WriteString("(")
	for i, v := range m.Values {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(v.Inspect())
	}
	out.WriteString(")")
	return out.String()
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

// Copy は SPEC 4.4 (値渡し) に従い配列を再帰的に複製する。
// Integer など不変の要素はそのまま共有する
func (a *Array) Copy() *Array {
	elements := make([]Object, len(a.Elements))

	for i, el := range a.Elements {
		if nested, ok := el.(*Array); ok {
			elements[i] = nested.Copy()
		} else {
			elements[i] = el
		}
	}

	return &Array{Elements: elements}
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

// Struct は SPEC 4.3 の struct 型の値。フィールドは公開、値渡し (Copy)
type Struct struct {
	Fields map[string]Object
}

func (s *Struct) Type() ObjectType {
	return STRUCT_OBJ
}

func (s *Struct) Inspect() string {
	// map の順序は不定なのでソートして安定表示にする
	names := make([]string, 0, len(s.Fields))
	for name := range s.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	var out strings.Builder

	out.WriteString("{")

	for i, name := range names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name + ": " + s.Fields[name].Inspect())
	}

	out.WriteString("}")

	return out.String()
}

// Copy は SPEC 4.3 (値渡し) に従い struct を再帰的に複製する
func (s *Struct) Copy() *Struct {
	fields := make(map[string]Object, len(s.Fields))

	for name, el := range s.Fields {
		switch v := el.(type) {
		case *Struct:
			fields[name] = v.Copy()
		case *Array:
			fields[name] = v.Copy()
		default:
			fields[name] = el
		}
	}

	return &Struct{Fields: fields}
}

// StructDef は実行時の型定義 (ゼロ値構築に使用)
type StructDef struct {
	FieldNames []string
	FieldTypes map[string]ast.TypeNode // フィールド名 → 型ノード

	// distinct 型 (SPEC 4.2) のときは基本型名が入る (フィールドは空)
	Base string
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
