package ast

import (
	"bytes"
	"fmt"
	"ruli/lexer"
	"strings"
)

type Node interface {
	String() string
	GetToken() lexer.Token
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type TypeNode interface {
	Node
	typeNode()
}

type Program struct {
	Token      lexer.Token
	Statements []Statement
}

func (p *Program) String() string {
	return ""
}

func (p *Program) GetToken() lexer.Token {
	return p.Token
}

type AssignStatement struct {
	Left  Expression
	Value Expression
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) String() string {
	return as.Left.String() + " = " + as.Value.String()
}

func (as *AssignStatement) GetToken() lexer.Token {
	return as.Left.GetToken()
}

type VarDeclStatement struct {
	Name  *Identifier
	Type  TypeNode   // 型推論なら nil
	Value Expression // 初期値なしなら nil
}

func (vs *VarDeclStatement) statementNode() {}

func (vs *VarDeclStatement) String() string {

	var out strings.Builder

	out.WriteString(vs.Name.String())

	if vs.Type != nil {
		out.WriteString(": ")
		out.WriteString(vs.Type.String())
	}

	if vs.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vs.Value.String())
	}

	return out.String()
}

func (vs *VarDeclStatement) GetToken() lexer.Token {
	return vs.Name.GetToken()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string {
	return i.Value
}
func (i *Identifier) GetToken() lexer.Token {
	return i.Token
}

type IntegerLiteral struct {
	Token lexer.Token
	Value int
}

func (i *IntegerLiteral) expressionNode() {}
func (i *IntegerLiteral) String() string {
	return fmt.Sprintf("%d", i.Value)
}

func (i *IntegerLiteral) GetToken() lexer.Token {
	return i.Token
}

type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (be *BinaryExpression) expressionNode() {}

func (be *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)",
		be.Left.String(),
		be.Operator,
		be.Right.String())
}

func (be *BinaryExpression) GetToken() lexer.Token {
	return be.Left.GetToken()
}

// PrefixExpression は -x / !flag のような単項演算
type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode() {}

func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("%s%s", pe.Operator, pe.Right.String())
}

func (pe *PrefixExpression) GetToken() lexer.Token {
	return pe.Token
}

// LogicalExpression は && / || の short-circuit 評価 (SPEC 6.2)。
// BinaryExpression は両辺を評価するため区別する
type LogicalExpression struct {
	Token    lexer.Token
	Operator string // "&&" "||"
	Left     Expression
	Right    Expression
}

func (le *LogicalExpression) expressionNode() {}

func (le *LogicalExpression) String() string {
	return fmt.Sprintf("(%s %s %s)",
		le.Left.String(),
		le.Operator,
		le.Right.String())
}

func (le *LogicalExpression) GetToken() lexer.Token {
	return le.Token
}

// RealLiteral は 3.5 のような実数リテラル (SPEC 4.1 の REAL / LREAL)
type RealLiteral struct {
	Token lexer.Token
	Value float64
}

func (rl *RealLiteral) expressionNode() {}

func (rl *RealLiteral) String() string {
	return fmt.Sprintf("%g", rl.Value)
}

func (rl *RealLiteral) GetToken() lexer.Token {
	return rl.Token
}

// CastExpression は REAL(a) のような明示的キャスト (SPEC 6.3)
type CastExpression struct {
	Token lexer.Token
	Type  string // 目標型名 (REAL, DINT など)
	Value Expression
}

func (ce *CastExpression) expressionNode() {}

func (ce *CastExpression) String() string {
	return fmt.Sprintf("%s(%s)", ce.Type, ce.Value.String())
}

func (ce *CastExpression) GetToken() lexer.Token {
	return ce.Token
}

type BasicType struct {
	Token lexer.Token
	Name  string
}

func (b *BasicType) typeNode() {}
func (b *BasicType) String() string {
	return b.Name
}

func (b *BasicType) GetToken() lexer.Token {
	return b.Token
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode() {}

func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

func (bs *BlockStatement) GetToken() lexer.Token {
	return bs.Token
}

type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfStatement) statementNode() {}

func (ie *IfStatement) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

func (ie *IfStatement) GetToken() lexer.Token {
	return ie.Token
}

type IncDecStatement struct {
	Token    lexer.Token
	Name     string
	Operator string
}

func (is *IncDecStatement) statementNode() {}

func (is *IncDecStatement) String() string {
	return is.Name + is.Operator
}

func (is *IncDecStatement) GetToken() lexer.Token {
	return is.Token
}

// SwitchCase は case 1, 2: または default: の本体込みの 1 ブロック
type SwitchCase struct {
	Token  lexer.Token
	Values []Expression // default は空
	Body   *BlockStatement
}

// SwitchStatement は switch 文 (SPEC 7.4)。fallthrough なし
type SwitchStatement struct {
	Token   lexer.Token
	Value   Expression
	Cases   []*SwitchCase
	Default *SwitchCase
}

func (ss *SwitchStatement) statementNode() {}

func (ss *SwitchStatement) String() string {
	var out strings.Builder
	out.WriteString("switch")
	out.WriteString(ss.Value.String())
	out.WriteString(" {")
	for range ss.Cases {
		out.WriteString(" case")
	}
	if ss.Default != nil {
		out.WriteString(" default")
	}
	out.WriteString(" }")
	return out.String()
}

func (ss *SwitchStatement) GetToken() lexer.Token {
	return ss.Token
}

// ForStatement は for i := 0; i < n; i++ { ... }
type ForStatement struct {
	Token     lexer.Token
	Init      Statement
	Condition Expression
	Post      Statement
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode() {}

func (fs *ForStatement) String() string {
	return "for"
}

func (fs *ForStatement) GetToken() lexer.Token {
	return fs.Token
}

type ExpressionStatement struct {
	Expression Expression
}

func (es *ExpressionStatement) statementNode() {}

func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

func (es *ExpressionStatement) GetToken() lexer.Token {
	return es.Expression.GetToken()
}

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}

func (ce *CallExpression) String() string {
	return "call"
}

func (ce *CallExpression) GetToken() lexer.Token {
	return ce.Token
}

type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode() {}

func (bs *BreakStatement) String() string {
	return "break"
}

func (bs *BreakStatement) GetToken() lexer.Token {
	return bs.Token
}

type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode() {}

func (cs *ContinueStatement) String() string {
	return "continue"
}

func (cs *ContinueStatement) GetToken() lexer.Token {
	return cs.Token
}

type Parameter struct {
	Name string
	Type TypeNode // 未指定なら nil (ANY)
	Ref  bool     // VAR_IN_OUT 相当の参照渡し (SPEC 8.1)
}

type FunctionStatement struct {
	Token       lexer.Token
	Name        string
	Receiver    *Parameter // メソッド (SPEC 5.6)。nil はトップレベル関数
	Parameters  []Parameter
	ReturnTypes []TypeNode // 単一 / 複数 (SPEC 5.3) の戻り値型。空なら戻り値なし
	Body        *BlockStatement
}

func (fs *FunctionStatement) statementNode() {}

func (fs *FunctionStatement) String() string {
	if fs.Receiver != nil {
		return "func" + " (recv) " + fs.Name
	}
	return "func " + fs.Name
}

func (fs *FunctionStatement) GetToken() lexer.Token {
	return fs.Token
}

// ReturnStatement は単一 / 複数 (SPEC 5.3) の戻り値
type ReturnStatement struct {
	Token  lexer.Token
	Values []Expression
}

func (rs *ReturnStatement) statementNode() {}

func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString("return")
	for i, v := range rs.Values {
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString(" " + v.String())
	}
	return out.String()
}

func (rs *ReturnStatement) GetToken() lexer.Token {
	return rs.Token
}

// CompoundAssignStatement は x += 1 / a[0] *= 2 / m.speed -= 1 のような
// 複合代入 (SPEC 6.2)。左辺は一度だけ評価される
type CompoundAssignStatement struct {
	Token    lexer.Token // 代入先の始端
	Target   Expression
	Operator string // "+=" "-=" "*=" "/=" "%="
	Value    Expression
}

func (ca *CompoundAssignStatement) statementNode() {}

func (ca *CompoundAssignStatement) String() string {
	return ca.Target.String() + " " + ca.Operator + " " + ca.Value.String()
}

func (ca *CompoundAssignStatement) GetToken() lexer.Token {
	return ca.Token
}

// MultiVarDeclStatement は x, ok := f() のような複数変数の束縛 (SPEC 5.4)
type MultiVarDeclStatement struct {
	Token  lexer.Token
	Names  []*Identifier
	Values Expression // 多値 (MultiValue) を評価する式
}

func (mv *MultiVarDeclStatement) statementNode() {}

func (mv *MultiVarDeclStatement) String() string {
	var out strings.Builder
	for i, name := range mv.Names {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(name.String())
	}
	out.WriteString(" := ")
	out.WriteString(mv.Values.String())
	return out.String()
}

func (mv *MultiVarDeclStatement) GetToken() lexer.Token {
	return mv.Token
}

// MultiAssignStatement は x, ok = f() のような複数変数への再代入
type MultiAssignStatement struct {
	Token lexer.Token
	Lefts []Expression
	Value Expression
}

func (ma *MultiAssignStatement) statementNode() {}

func (ma *MultiAssignStatement) String() string {
	var out strings.Builder
	for i, left := range ma.Lefts {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(left.String())
	}
	out.WriteString(" = ")
	out.WriteString(ma.Value.String())
	return out.String()
}

func (ma *MultiAssignStatement) GetToken() lexer.Token {
	return ma.Token
}

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode() {}

func (sl *StringLiteral) String() string {
	return `"` + sl.Value + `"`
}

func (sl *StringLiteral) GetToken() lexer.Token {
	return sl.Token
}

type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode() {}

func (al *ArrayLiteral) String() string {
	var out strings.Builder

	out.WriteString("[")

	for i, e := range al.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(e.String())
	}

	out.WriteString("]")

	return out.String()
}

func (al *ArrayLiteral) GetToken() lexer.Token {
	return al.Token
}

type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode() {}

func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

func (ie *IndexExpression) GetToken() lexer.Token {
	return ie.Token
}

// StructLiteralField は構造体リテラルの 1 フィールド
type StructLiteralField struct {
	Name  string
	Value Expression
}

// StructLiteral は Motor{speed: 10, enable: true} のような構造体リテラル。
// 未指定フィールドはゼロ値で埋まる
type StructLiteral struct {
	Token  lexer.Token
	Name   string // 構造体型名
	Fields []StructLiteralField
}

func (sl *StructLiteral) expressionNode() {}

func (sl *StructLiteral) String() string {
	var out strings.Builder
	out.WriteString(sl.Name + "{")
	for i, f := range sl.Fields {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(f.Name + ": " + f.Value.String())
	}
	out.WriteString("}")
	return out.String()
}

func (sl *StructLiteral) GetToken() lexer.Token {
	return sl.Token
}

// SelectorExpression は m.speed のようなフィールドアクセス
type SelectorExpression struct {
	Token  lexer.Token
	Object Expression
	Field  string
}

func (se *SelectorExpression) expressionNode() {}

func (se *SelectorExpression) String() string {
	return se.Object.String() + "." + se.Field
}

func (se *SelectorExpression) GetToken() lexer.Token {
	return se.Token
}

// ArrayType は [10]INT のような固定長配列型 (SPEC 4.4)
type ArrayType struct {
	Token lexer.Token // '['
	Size  int         // 要素数 (常に 1 以上)
	Elem  TypeNode    // 要素型 ([3][3]INT ではさらに ArrayType)
}

func (at *ArrayType) typeNode() {}

func (at *ArrayType) String() string {
	return fmt.Sprintf("[%d]%s", at.Size, at.Elem.String())
}

func (at *ArrayType) GetToken() lexer.Token {
	return at.Token
}

// StructField は struct 宣言の 1 フィールド
type StructField struct {
	Name string
	Type TypeNode
}

// TypeDeclStatement は型宣言
//   - struct:  type Motor struct { ... }  (Fields が設定される)
//   - distinct: type TEMP INT              (Base が設定される, SPEC 4.2)
type TypeDeclStatement struct {
	Token lexer.Token
	Name  string

	Fields []StructField
	Base   TypeNode // distinct 型の基底型
}

func (tds *TypeDeclStatement) statementNode() {}

func (tds *TypeDeclStatement) String() string {
	return "type " + tds.Name
}

func (tds *TypeDeclStatement) GetToken() lexer.Token {
	return tds.Token
}

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode() {}

func (bl *BooleanLiteral) String() string {
	return fmt.Sprintf("%t", bl.Value)
}

func (bl *BooleanLiteral) GetToken() lexer.Token {
	return bl.Token
}
