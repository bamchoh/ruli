package checker

import (
	"fmt"
	"ruli/ast"
	"ruli/lexer"
	"ruli/object"
	"strings"
)

// TypeError は型チェックエラー (行列位置つき)
type TypeError struct {
	Message string

	Line   int
	Column int
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
}

type Checker struct {
	env *Env

	// struct 型宣言のレジストリ (型名 → 型)
	types map[string]*Type

	// 現在チェック中の関数の戻り値型 (関数外なら nil / 空)
	returnTypes []*Type
	inFunction  bool

	errors []TypeError
}

// Check はプログラム全体を型チェックし、見つかったエラーすべてを返す。
// SPEC 2.1 に従い、まずトップレベルの func 宣言を登録してから文を順にチェックする
func Check(program *ast.Program) []TypeError {
	c := &Checker{env: NewEnv(), types: make(map[string]*Type)}

	// 宣言収集 (SPEC 2.1): 順不同の参照と再帰を可能にする。
	// struct (プレースホルダ) → func (シグネチャ) の 2 パスで登録する
	for _, stmt := range program.Statements {
		if td, ok := stmt.(*ast.TypeDeclStatement); ok {
			c.registerStructType(td)
		}
	}
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			c.registerFunctionSignature(fn, c.env)
		}
	}

	for _, stmt := range program.Statements {
		c.checkStatement(stmt, c.env)
	}

	return c.errors
}

func (c *Checker) errAt(token lexer.Token, format string, args ...interface{}) {
	c.errors = append(c.errors, TypeError{
		Message: fmt.Sprintf(format, args...),
		Line:    token.Line,
		Column:  token.Column,
	})
}

func (c *Checker) checkStatements(stmts []ast.Statement, env *Env) {
	for _, stmt := range stmts {
		c.checkStatement(stmt, env)
	}
}

func (c *Checker) checkStatement(n ast.Statement, env *Env) {
	if n == nil {
		return
	}

	switch stmt := n.(type) {
	case *ast.VarDeclStatement:
		c.checkVarDecl(stmt, env)
	case *ast.AssignStatement:
		c.checkAssign(stmt, env)
	case *ast.CompoundAssignStatement:
		c.checkCompoundAssign(stmt, env)
	case *ast.IfStatement:
		c.checkIf(stmt, env)
	case *ast.ForStatement:
		c.checkFor(stmt, env)
	case *ast.SwitchStatement:
		c.checkSwitch(stmt, env)
	case *ast.IncDecStatement:
		c.checkIncDec(stmt, env)
	case *ast.FunctionStatement:
		c.checkFunctionBody(stmt, env)
	case *ast.ReturnStatement:
		c.checkReturn(stmt, env)
	case *ast.MultiVarDeclStatement:
		c.checkMultiVarDecl(stmt, env)
	case *ast.MultiAssignStatement:
		c.checkMultiAssign(stmt, env)
	case *ast.ExpressionStatement:
		c.checkExpr(stmt.Expression, env)
	case *ast.TypeDeclStatement:
		c.checkTypeDecl(stmt, env)
	case *ast.BreakStatement, *ast.ContinueStatement:
	}
}

// registerStructType は型宣言を型レジストリに登録する。
// 宣言収集パスで先に呼ばれ、フィールドは後で checkTypeDecl が解決する
func (c *Checker) registerStructType(n *ast.TypeDeclStatement) *Type {
	if st, ok := c.types[n.Name]; ok {
		return st
	}

	st := &Type{Name: STRUCT_TYPE.Name, Fields: map[string]*Type{}}
	c.types[n.Name] = st
	return st
}

// checkTypeDecl は型宣言の内容を解決して登録する
func (c *Checker) checkTypeDecl(n *ast.TypeDeclStatement, env *Env) {
	if n.Base != nil {
		// distinct 型 (SPEC 4.2): 基本型は基本型のみ (struct/distinct は不可)
		base, ok := resolveBasic(n.Base.String())
		if !ok {
			c.errAt(n.Base.GetToken(),
				"distinct type base must be a basic type, got %s", n.Base.String())
			return
		}

		st := &Type{Name: n.Name, Base: base}
		c.types[n.Name] = st
		return
	}

	st := c.registerStructType(n)

	fields := make(map[string]*Type, len(n.Fields))
	names := make([]string, 0, len(n.Fields))

	for _, field := range n.Fields {
		if _, dup := fields[field.Name]; dup {
			c.errAt(n.GetToken(), "duplicate field: %s", field.Name)
			continue
		}

		ft := c.resolveTypeNode(field.Type)
		if ft == nil {
			continue
		}

		names = append(names, field.Name)
		fields[field.Name] = ft
	}

	st.FieldNames = names
	st.Fields = fields
}

func (c *Checker) checkIf(stmt *ast.IfStatement, env *Env) {
	cond := c.checkExpr(stmt.Condition, env)
	if cond != nil && cond.Name != "BOOL" {
		c.errAt(stmt.GetToken(), "condition must be BOOL, got %s", cond.Name)
	}

	if stmt.Consequence != nil {
		ifEnv := NewEnclosedEnvironment(env)
		c.checkStatements(stmt.Consequence.Statements, ifEnv)
	}

	if stmt.Alternative != nil {
		elseEnv := NewEnclosedEnvironment(env)
		c.checkStatements(stmt.Alternative.Statements, elseEnv)
	}
}

func (c *Checker) checkFor(stmt *ast.ForStatement, env *Env) {
	loopEnv := NewEnclosedEnvironment(env)

	if stmt.Init != nil {
		c.checkStatement(stmt.Init, loopEnv)
	}

	cond := c.checkExpr(stmt.Condition, loopEnv)
	if cond != nil && cond.Name != "BOOL" {
		c.errAt(stmt.GetToken(), "condition must be BOOL, got %s", cond.Name)
	}

	if stmt.Post != nil {
		c.checkStatement(stmt.Post, loopEnv)
	}

	if stmt.Body != nil {
		bodyEnv := NewEnclosedEnvironment(loopEnv)
		c.checkStatements(stmt.Body.Statements, bodyEnv)
	}
}

// checkSwitch は switch 文 (SPEC 7.4) を検査する
func (c *Checker) checkSwitch(stmt *ast.SwitchStatement, env *Env) {
	vt := c.checkExpr(stmt.Value, env)
	if vt == nil {
		return
	}

	// switch の値は比較可能な型のみ (struct / 配列は == と同じく禁止)
	switch {
	case isNumeric(vt) || vt.Name == STRING_TYPE.Name || vt.Name == BOOL_TYPE.Name:
	default:
		c.errAt(
			stmt.GetToken(),
			"switch value must be INT, REAL, STRING or BOOL, got %s",
			vt.String(),
		)
		return
	}

	for _, sc := range stmt.Cases {
		for _, value := range sc.Values {
			cvt := c.checkExpr(value, env)
			if cvt == nil {
				continue
			}

			if !vt.Equal(cvt) {
				c.errAt(
					value.GetToken(),
					"case value type mismatch: expected %s, got %s",
					vt.String(),
					cvt.String(),
				)
			}
		}

		if sc.Body != nil {
			caseEnv := NewEnclosedEnvironment(env)
			c.checkStatements(sc.Body.Statements, caseEnv)
		}
	}

	if stmt.Default != nil && stmt.Default.Body != nil {
		defaultEnv := NewEnclosedEnvironment(env)
		c.checkStatements(stmt.Default.Body.Statements, defaultEnv)
	}
}

func (c *Checker) checkIncDec(stmt *ast.IncDecStatement, env *Env) {
	t, ok := env.Get(stmt.Name)
	if !ok {
		c.errAt(stmt.GetToken(), "undefined variable: %s", stmt.Name)
		return
	}

	if t.Name != INT_TYPE.Name {
		c.errAt(
			stmt.GetToken(),
			"operand of %s must be INT, got %s",
			stmt.Operator,
			t.Name,
		)
	}
}

func (c *Checker) checkVarDecl(stmt *ast.VarDeclStatement, env *Env) {
	name := stmt.Name.Value

	switch {

	case stmt.Value != nil:
		vt := c.checkExpr(stmt.Value, env)
		if vt == nil {
			return
		}

		// SPEC 5.4: 複数戻り値を単一変数へ代入は禁止
		if vt.Name == MULTI_TYPE.Name {
			c.errAt(
				stmt.GetToken(),
				"cannot assign %d return values to %d variables",
				len(vt.Returns),
				1,
			)
			return
		}

		if stmt.Type != nil {
			dt := c.resolveTypeNode(stmt.Type)
			if dt == nil {
				return
			}

			if !dt.Equal(vt) {
				c.errAt(
					stmt.GetToken(),
					"type mismatch: declared %s, got %s",
					dt.String(),
					vt.String(),
				)
				return
			}

			c.checkLiteralRange(dt.Name, stmt.Value)
			env.Set(name, dt)
		} else {
			env.Set(name, vt)
		}

	case stmt.Type != nil:
		dt := c.resolveTypeNode(stmt.Type)
		if dt == nil {
			return
		}
		env.Set(name, dt)

	default:
		c.errAt(
			stmt.GetToken(),
			"declaration requires a type or an initial value",
		)
	}
}

func (c *Checker) checkAssign(stmt *ast.AssignStatement, env *Env) {
	switch left := stmt.Left.(type) {

	case *ast.Identifier:
		declared, ok := env.Get(left.Value)
		if !ok {
			c.errAt(left.GetToken(), "undefined variable: %s", left.Value)
			return
		}

		vt := c.checkExpr(stmt.Value, env)
		if vt == nil {
			return
		}

		if !declared.Equal(vt) {
			c.errAt(
				stmt.GetToken(),
				"type mismatch: %s %s %s",
				declared.String(),
				"=",
				vt.String(),
			)
		}

	case *ast.IndexExpression:
		// 配列ターゲットの共通検査 (境界・インデックス型)
		elem := c.checkIndexTarget(left, env)
		if elem == nil {
			return
		}

		// 固定幅要素へのリテラル代入は範囲を静的に検査する
		c.checkLiteralRange(elem.Name, stmt.Value)

		vt := c.checkExpr(stmt.Value, env)
		if vt != nil && !elem.Equal(vt) {
			c.errAt(
				stmt.Value.GetToken(),
				"type mismatch: %s %s %s",
				elem.String(),
				"=",
				vt.String(),
			)
		}

	case *ast.SelectorExpression:
		ot := c.checkExpr(left.Object, env)
		if ot == nil {
			return
		}

		if ot.Name == ANY_TYPE.Name || ot.Name == UNKNOWN_TYPE.Name {
			c.checkExpr(stmt.Value, env)
			return
		}

		if ot.Name != STRUCT_TYPE.Name {
			c.errAt(left.GetToken(), "no field '%s' on %s", left.Field, ot.Name)
			return
		}

		ft, ok := ot.Fields[left.Field]
		if !ok {
			c.errAt(left.GetToken(), "undefined field: %s", left.Field)
			return
		}

		vt := c.checkExpr(stmt.Value, env)
		if vt != nil && !ft.Equal(vt) {
			c.errAt(
				stmt.Value.GetToken(),
				"type mismatch: %s %s %s",
				ft.Name,
				"=",
				vt.Name,
			)
		}

	default:
		c.errAt(stmt.GetToken(), "invalid assignment target")
	}
}

// checkIndexTarget は配列への書き込みターゲット (a[0] = / a[0] +=) の共通検査。
// 配列性・静的境界・インデックス型を検査し、要素型を返す (エラーなら nil)
func (c *Checker) checkIndexTarget(left *ast.IndexExpression, env *Env) *Type {
	array := c.checkExpr(left.Left, env)
	if array == nil {
		return nil
	}

	if array.Name != ARRAY_TYPE.Name && array.Name != ANY_TYPE.Name {
		c.errAt(left.GetToken(), "not an array")
		return nil
	}

	// 既知サイズへのリテラルインデックスは静的に判定する
	if array.Name == ARRAY_TYPE.Name && array.ArrLen > 0 {
		if lit, ok := left.Index.(*ast.IntegerLiteral); ok {
			if lit.Value < 0 || lit.Value >= array.ArrLen {
				c.errAt(
					lit.GetToken(),
					"index out of range: %d (array size %d)",
					lit.Value,
					array.ArrLen,
				)
				return nil
			}
		}
	}

	index := c.checkExpr(left.Index, env)
	if index != nil && index.Name != INT_TYPE.Name {
		c.errAt(
			left.Index.GetToken(),
			"index must be INT, got %s",
			index.Name,
		)
		return nil
	}

	if array.Elem == nil {
		return UNKNOWN_TYPE
	}
	return array.Elem
}

// checkCompoundAssign は x += 1 などの複合代入を検査する (SPEC 6.2)。
// 二項演算と同じ型規則 (checkOperation) を左辺型に適用する
func (c *Checker) checkCompoundAssign(stmt *ast.CompoundAssignStatement, env *Env) {
	operator := strings.TrimSuffix(stmt.Operator, "=")

	var tLeft *Type

	switch target := stmt.Target.(type) {
	case *ast.Identifier:
		t, ok := env.Get(target.Value)
		if !ok {
			c.errAt(target.GetToken(), "undefined variable: %s", target.Value)
			return
		}
		tLeft = t

	case *ast.IndexExpression:
		tLeft = c.checkIndexTarget(target, env)
		if tLeft == nil {
			return
		}

	case *ast.SelectorExpression:
		ot := c.checkExpr(target.Object, env)
		if ot == nil {
			return
		}

		if ot.Name == ANY_TYPE.Name || ot.Name == UNKNOWN_TYPE.Name {
			c.checkExpr(stmt.Value, env)
			return
		}

		if ot.Name != STRUCT_TYPE.Name {
			c.errAt(target.GetToken(), "no field '%s' on %s", target.Field, ot.Name)
			return
		}

		ft, ok := ot.Fields[target.Field]
		if !ok {
			c.errAt(target.GetToken(), "undefined field: %s", target.Field)
			return
		}
		tLeft = ft

	default:
		c.errAt(stmt.GetToken(), "invalid assignment target")
		return
	}

	vt := c.checkExpr(stmt.Value, env)
	if vt == nil {
		return
	}

	c.checkOperation(tLeft, operator, vt, stmt.GetToken())
}

// checkMultiVarDecl は x, ok := f() のような複数変数の束縛を検査する (SPEC 5.4)
func (c *Checker) checkMultiVarDecl(stmt *ast.MultiVarDeclStatement, env *Env) {
	vt := c.checkExpr(stmt.Values, env)
	if vt == nil {
		return
	}

	names := make([]ast.Expression, 0, len(stmt.Names))
	for _, name := range stmt.Names {
		names = append(names, name)
	}

	c.checkMultiBind(stmt, names, vt, env, true)
}

// checkMultiAssign は x, ok = f() のような複数代入を検査する
func (c *Checker) checkMultiAssign(stmt *ast.MultiAssignStatement, env *Env) {
	vt := c.checkExpr(stmt.Value, env)
	if vt == nil {
		return
	}

	names := make([]ast.Expression, 0, len(stmt.Lefts))
	for _, left := range stmt.Lefts {
		ident, ok := left.(*ast.Identifier)
		if !ok {
			c.errAt(left.GetToken(), "invalid assignment target")
			return
		}
		names = append(names, ident)
	}

	c.checkMultiBind(stmt, names, vt, env, false)
}

// checkMultiBind は束縛先の個数と型を検査する
func (c *Checker) checkMultiBind(node ast.Node, names []ast.Expression, vt *Type, env *Env, declare bool) {
	count := len(names)

	if vt.Name == MULTI_TYPE.Name {
		if len(vt.Returns) != count {
			c.errAt(
				astNodeToken(node),
				"cannot assign %d return values to %d variables",
				len(vt.Returns),
				count,
			)
			return
		}

		for i := range vt.Returns {
			ident := names[i].(*ast.Identifier)

			if declare {
				c.checkDeclName(names[i], vt.Returns[i])
				env.Set(ident.Value, vt.Returns[i])
				continue
			}

			declared, ok := env.Get(ident.Value)
			if !ok {
				c.errAt(names[i].GetToken(), "undefined variable: %s", ident.Value)
				continue
			}
			if !declared.Equal(vt.Returns[i]) {
				c.errAt(
					names[i].GetToken(),
					"type mismatch: declared %s, got %s",
					declared.Name,
					vt.Returns[i].Name,
				)
			}
		}
		return
	}

	// 多値でない
	if count != 1 {
		c.errAt(
			astNodeToken(node),
			"cannot assign %d values to %d variables",
			1,
			count,
		)
		return
	}

	ident := names[0].(*ast.Identifier)

	if declare {
		c.checkDeclName(names[0], vt)
		env.Set(ident.Value, vt)
		return
	}

	declared, ok := env.Get(ident.Value)
	if !ok {
		c.errAt(names[0].GetToken(), "undefined variable: %s", ident.Value)
		return
	}
	if !declared.Equal(vt) {
		c.errAt(
			names[0].GetToken(),
			"type mismatch: declared %s, got %s",
			declared.Name,
			vt.Name,
		)
	}
}

// checkDeclName は宣言先が識別子であることを確認する
func (c *Checker) checkDeclName(expr ast.Expression, t *Type) {
	if _, ok := expr.(*ast.Identifier); !ok {
		c.errAt(expr.GetToken(), "invalid declaration target")
	}
}

func astNodeToken(n ast.Node) lexer.Token {
	return n.GetToken()
}

func (c *Checker) checkReturn(stmt *ast.ReturnStatement, env *Env) {
	if !c.inFunction {
		c.errAt(stmt.GetToken(), "return outside function")
		return
	}

	valueTypes := make([]*Type, 0, len(stmt.Values))
	for _, value := range stmt.Values {
		vt := c.checkExpr(value, env)
		if vt == nil {
			return
		}
		valueTypes = append(valueTypes, vt)
	}

	if len(c.returnTypes) == 0 {
		// 戻り値なし宣言の戻り値は寛容に扱う (v1)
		return
	}

	if len(c.returnTypes) != len(valueTypes) {
		c.errAt(
			stmt.GetToken(),
			"wrong number of return values. got=%d, want=%d",
			len(valueTypes),
			len(c.returnTypes),
		)
		return
	}

	for i := range c.returnTypes {
		if !c.returnTypes[i].Equal(valueTypes[i]) {
			c.errAt(
				stmt.Values[i].GetToken(),
				"type mismatch: declared %s, got %s",
				c.returnTypes[i].Name,
				valueTypes[i].Name,
			)
		}
	}
}

// registerFunctionSignature は関数のシグネチャ型を現在スコープに登録する
func (c *Checker) registerFunctionSignature(stmt *ast.FunctionStatement, env *Env) *Type {
	params := make([]Type, 0, len(stmt.Parameters))
	refs := make([]bool, 0, len(stmt.Parameters))

	for _, param := range stmt.Parameters {
		refs = append(refs, param.Ref)

		if param.Type == nil {
			params = append(params, *ANY_TYPE)
			continue
		}

		t := c.resolveTypeNode(param.Type)
		if t == nil {
			params = append(params, *ANY_TYPE)
			continue
		}
		params = append(params, *t)
	}

	returns := make([]*Type, 0, len(stmt.ReturnTypes))
	for _, rt := range stmt.ReturnTypes {
		t := c.resolveTypeNode(rt)
		if t == nil {
			returns = append(returns, UNKNOWN_TYPE)
			continue
		}
		returns = append(returns, t)
	}

	fn := &Type{Name: FUNCTION_TYPE.Name, Params: params, Refs: refs, Returns: returns}
	env.Set(stmt.Name, fn)
	return fn
}

// checkFunctionBody は関数の本体をチェックする (シグネチャは登録済み前提)
func (c *Checker) checkFunctionBody(stmt *ast.FunctionStatement, env *Env) {
	fnType := c.registerFunctionSignature(stmt, env)

	fnEnv := NewEnclosedEnvironment(env)
	for i, param := range stmt.Parameters {
		fnEnv.Set(param.Name, &fnType.Params[i])
	}

	prevReturn, prevInFunction := c.returnTypes, c.inFunction
	c.returnTypes = fnType.Returns
	c.inFunction = true

	c.checkStatements(stmt.Body.Statements, fnEnv)

	c.returnTypes, c.inFunction = prevReturn, prevInFunction
}

// checkLiteralRange は整数リテラルが固定幅型の値域に収まっているか検査する
func (c *Checker) checkLiteralRange(typeName string, valueExpr ast.Expression) {
	// 単項マイナス付きリテラル (x : SINT = -128) は負値リテラルとして扱う
	if pe, ok := valueExpr.(*ast.PrefixExpression); ok {
		if pe.Operator != "-" {
			return
		}
		lit, ok := pe.Right.(*ast.IntegerLiteral)
		if !ok {
			return // 変数などは実行時に検査する
		}
		c.checkIntRange(typeName, -int64(lit.Value), lit.GetToken())
		return
	}

	lit, ok := valueExpr.(*ast.IntegerLiteral)
	if !ok {
		return // リテラル以外 (変数など) は実行時に検査する
	}
	c.checkIntRange(typeName, int64(lit.Value), lit.GetToken())
}

func (c *Checker) checkIntRange(typeName string, value int64, token lexer.Token) {
	w, ok := object.WidthOf(typeName)
	if !ok {
		return
	}

	if value < w.Min || value > w.Max {
		c.errAt(
			token,
			"value %d out of range for %s (%d..%d)",
			value,
			typeName,
			w.Min,
			w.Max,
		)
	}
}

// resolveTypeNode は ast.TypeNode を型に解決する
func (c *Checker) resolveTypeNode(node ast.TypeNode) *Type {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.BasicType:
		if t, ok := resolveBasic(n.Name); ok {
			return t
		}
		if t, ok := c.types[n.Name]; ok {
			return t
		}
		c.errAt(n.GetToken(), "unknown type: %s", n.Name)
		return nil
	case *ast.ArrayType:
		elem := c.resolveTypeNode(n.Elem)
		if elem == nil {
			return nil
		}
		return &Type{Name: ARRAY_TYPE.Name, Elem: elem, ArrLen: n.Size}
	}

	return nil
}

// checkExpr は式の型を返す。エラーを記録した場合は nil を返す
func (c *Checker) checkExpr(expr ast.Expression, env *Env) *Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {

	case *ast.IntegerLiteral:
		return INT_TYPE

	case *ast.BooleanLiteral:
		return BOOL_TYPE

	case *ast.StringLiteral:
		return STRING_TYPE

	case *ast.Identifier:
		t, ok := env.Get(e.Value)
		if !ok {
			c.errAt(e.GetToken(), "undefined variable: %s", e.Value)
			return nil
		}
		return t

	case *ast.ArrayLiteral:
		var elem *Type

		for _, el := range e.Elements {
			et := c.checkExpr(el, env)
			if et == nil {
				continue
			}

			if elem == nil {
				elem = et
				continue
			}

			if !elem.Equal(et) {
				c.errAt(el.GetToken(),
					"array elements must have the same type: %s and %s",
					elem.Name, et.Name)
			}
		}

		// リテラルの長さはサイズとして型に含める (SPEC 4.4)
		return &Type{
			Name:   ARRAY_TYPE.Name,
			Elem:   elem,
			ArrLen: len(e.Elements),
		}

	case *ast.BinaryExpression:
		return c.checkBinary(e, env)

	case *ast.IndexExpression:
		array := c.checkExpr(e.Left, env)
		if array == nil {
			return nil
		}

		if array.Name != ARRAY_TYPE.Name && array.Name != ANY_TYPE.Name {
			c.errAt(e.GetToken(), "not an array")
			return nil
		}

		// 既知サイズへのリテラルインデックスは静的に判定する (SPEC 4.7 の強化)
		if array.Name == ARRAY_TYPE.Name && array.ArrLen > 0 {
			if lit, ok := e.Index.(*ast.IntegerLiteral); ok {
				if lit.Value < 0 || lit.Value >= array.ArrLen {
					c.errAt(
						lit.GetToken(),
						"index out of range: %d (array size %d)",
						lit.Value,
						array.ArrLen,
					)
					return nil
				}
			}
		}

		index := c.checkExpr(e.Index, env)
		if index != nil && index.Name != INT_TYPE.Name {
			c.errAt(
				e.Index.GetToken(),
				"index must be INT, got %s",
				index.Name,
			)
			return nil
		}

		if array.Elem == nil {
			return UNKNOWN_TYPE
		}
		return array.Elem

	case *ast.CastExpression:
		return c.checkCast(e, env)

	case *ast.CallExpression:
		return c.checkCall(e, env)

	case *ast.LogicalExpression:
		lt := c.checkExpr(e.Left, env)
		rt := c.checkExpr(e.Right, env)
		if lt == nil || rt == nil {
			return nil
		}

		for i, t := range []*Type{lt, rt} {
			// 型不明は通す (未指定引数など)
			if t.Name == ANY_TYPE.Name || t.Name == UNKNOWN_TYPE.Name {
				continue
			}
			if t.Name != BOOL_TYPE.Name {
				operand := "left"
				if i == 1 {
					operand = "right"
				}
				c.errAt(
					e.GetToken(),
					"operand of %s must be BOOL, got %s (%s)",
					e.Operator,
					t.String(),
					operand,
				)
				return nil
			}
		}

		return BOOL_TYPE

	case *ast.PrefixExpression:
		rt := c.checkExpr(e.Right, env)
		if rt == nil {
			return nil
		}

		if e.Operator == "!" {
			if rt.Name != BOOL_TYPE.Name {
				c.errAt(e.GetToken(), "operand of ! must be BOOL, got %s", rt.Name)
				return nil
			}
			return BOOL_TYPE
		}

		// "-" : 数値型のみ
		if rt.Name == ANY_TYPE.Name || rt.Name == UNKNOWN_TYPE.Name {
			return UNKNOWN_TYPE
		}
		if !isNumeric(rt) {
			c.errAt(e.GetToken(), "operand of - must be numeric, got %s", rt.Name)
			return nil
		}
		return rt

	case *ast.SelectorExpression:
		ot := c.checkExpr(e.Object, env)
		if ot == nil {
			return nil
		}

		if ot.Name == ANY_TYPE.Name || ot.Name == UNKNOWN_TYPE.Name {
			return UNKNOWN_TYPE
		}

		if ot.Name != STRUCT_TYPE.Name {
			c.errAt(e.Object.GetToken(), "no field '%s' on %s", e.Field, ot.Name)
			return nil
		}

		ft, ok := ot.Fields[e.Field]
		if !ok {
			c.errAt(e.GetToken(), "undefined field: %s", e.Field)
			return nil
		}
		return ft
	}

	return nil
}

func (c *Checker) checkBinary(be *ast.BinaryExpression, env *Env) *Type {
	lt := c.checkExpr(be.Left, env)
	rt := c.checkExpr(be.Right, env)
	if lt == nil || rt == nil {
		return nil
	}

	return c.checkOperation(lt, be.Operator, rt, be.GetToken())
}

// checkOperation は二項演算の型規則 (SPEC 6.2 / 6.3) を検査し、結果型を返す。
// 複合代入 (x += 1) と二項演算 (x + 1) で同じ規則を共有する
func (c *Checker) checkOperation(lt *Type, operator string, rt *Type, token lexer.Token) *Type {
	// 型不明のオペランド (未指定引数など) は演算の正否を判断できないので通す
	if lt.Name == UNKNOWN_TYPE.Name || lt.Name == ANY_TYPE.Name {
		if isNumeric(rt) {
			return rt
		}
		return UNKNOWN_TYPE
	}
	if rt.Name == UNKNOWN_TYPE.Name || rt.Name == ANY_TYPE.Name {
		if isNumeric(lt) {
			return lt
		}
		return UNKNOWN_TYPE
	}

	switch operator {
	case "+", "-", "*", "/", "%":
		// SPEC 6.3: INT + REAL はエラー。明示キャスト必須
		if !lt.Equal(rt) {
			break
		}
		if isNumeric(lt) {
			return lt
		}
		if operator == "+" && lt.Name == STRING_TYPE.Name {
			return lt
		}

	case "<", ">", "<=", ">=":
		if !lt.Equal(rt) {
			break
		}
		// SPEC 11: 文字列の < > 比較は非対応
		if isNumeric(lt) {
			return BOOL_TYPE
		}

	case "==", "!=":
		// 比較可能なのは基本値型のみ (配列 / struct / 関数は値渡しなので比較は禁止)
		for _, t := range []*Type{lt, rt} {
			switch t.Name {
			case STRUCT_TYPE.Name, ARRAY_TYPE.Name, FUNCTION_TYPE.Name:
				c.errAt(token, "cannot compare %s", t.Name)
				return nil
			}
		}
		if lt.Equal(rt) {
			return BOOL_TYPE
		}
	}

	c.errAt(
		token,
		"type mismatch: %s %s %s",
		lt.Name,
		operator,
		rt.Name,
	)
	return nil
}

func (c *Checker) checkCall(e *ast.CallExpression, env *Env) *Type {

	var fnType *Type
	if ident, ok := e.Function.(*ast.Identifier); ok {
		if t, found := env.Get(ident.Value); found {
			if t.Name != FUNCTION_TYPE.Name {
				c.errAt(e.GetToken(), "not a function: %s", ident.Value)
				return nil
			}
			fnType = t
		} else {
			// ユーザ定義ではない → 型へのキャスト (SPEC 4.2 / 6.3) または組み込み関数
			if _, isBasic := resolveBasic(ident.Value); isBasic || c.types[ident.Value] != nil {
				return c.checkCallCast(ident.Value, e, env)
			}
			return c.checkBuiltinCall(ident.Value, e, env)
		}
	} else {
		c.errAt(
			e.GetToken(),
			"not a function: %s",
			"",
		)
		return nil
	}

	// 引数
	argTypes := make([]*Type, 0, len(e.Arguments))
	for _, arg := range e.Arguments {
		at := c.checkExpr(arg, env)
		if at == nil {
			return nil
		}
		argTypes = append(argTypes, at)
	}

	if len(argTypes) != len(fnType.Params) {
		c.errAt(
			e.GetToken(),
			"wrong number of arguments. got=%d, want=%d",
			len(argTypes),
			len(fnType.Params),
		)
		return nil
	}

	for i := range fnType.Params {
		// ref パラメータ (SPEC 8.1) は変数 (識別子 / フィールド / 要素) のみ渡せる
		if fnType.Refs[i] && !isVariableForm(e.Arguments[i]) {
			c.errAt(
				e.Arguments[i].GetToken(),
				"ref argument must be a variable: %s",
				e.Arguments[i].String(),
			)
			return nil
		}

		if fnType.Params[i].Name == ANY_TYPE.Name {
			continue
		}
		if !argTypes[i].Equal(&fnType.Params[i]) {
			c.errAt(
				e.Arguments[i].GetToken(),
				"type mismatch: argument %d: expected %s, got %s",
				i+1,
				fnType.Params[i].String(),
				argTypes[i].Name,
			)
		}
	}

	switch len(fnType.Returns) {
	case 0:
		return VOID_TYPE
	case 1:
		return fnType.Returns[0]
	default:
		return &Type{Name: MULTI_TYPE.Name, Returns: fnType.Returns}
	}
}

// checkCast は REAL(a) のような明示的キャストを検査する (SPEC 6.3)
func (c *Checker) checkCast(e *ast.CastExpression, env *Env) *Type {
	target, ok := resolveBasic(e.Type)
	if !ok {
		c.errAt(e.GetToken(), "unknown type: %s", e.Type)
		return nil
	}

	vt := c.checkExpr(e.Value, env)
	if vt == nil {
		return nil
	}

	if !castAllowed(target, vt) {
		c.errAt(
			e.Value.GetToken(),
			"invalid cast: %s to %s",
			vt.String(),
			e.Type,
		)
		return nil
	}

	// 整数リテラルの範囲外キャストは静的に検査する (WORD(70000) など)
	c.checkLiteralRange(target.Name, e.Value)

	return target
}

// checkCallCast はユーザ定義型への呼び出し形式キャスト (SPEC 4.2: TEMP(30)) を検査する
func (c *Checker) checkCallCast(targetName string, e *ast.CallExpression, env *Env) *Type {
	if len(e.Arguments) != 1 {
		c.errAt(
			e.GetToken(),
			"wrong number of arguments. got=%d, want=1",
			len(e.Arguments),
		)
		return nil
	}

	target, ok := resolveBasic(targetName)
	if !ok {
		target = c.types[targetName]
	}

	vt := c.checkExpr(e.Arguments[0], env)
	if vt == nil {
		return nil
	}

	if !castAllowed(target, vt) {
		c.errAt(
			e.Arguments[0].GetToken(),
			"invalid cast: %s to %s",
			vt.String(),
			targetName,
		)
		return nil
	}

	return target
}

// castAllowed はキャストの許可規則 (SPEC 6.3 / 4.2):
//   - 同型なら無変換
//   - BOOL へは BOOL からのみ
//   - STRING へは不可
//   - 数値族から数値族へ (distinct 型は Base の族で判定)
//   - struct / 配列 / 関数は不可
func castAllowed(target *Type, vt *Type) bool {
	if target == nil || vt == nil {
		return false
	}
	if target.Equal(vt) {
		return true
	}

	tBase := distinctBase(target)
	vBase := distinctBase(vt)

	switch tBase.Name {
	case BOOL_TYPE.Name:
		return vBase.Name == BOOL_TYPE.Name
	case STRING_TYPE.Name, STRUCT_TYPE.Name:
		return false
	}

	if !isNumeric(tBase) || !isNumeric(vBase) {
		if vt.Name == STRING_TYPE.Name || vBase.Name == STRING_TYPE.Name {
			return false
		}
	}

	return isNumeric(tBase) && isNumeric(vBase)
}

// distinctBase は distinct 型のときは基本型を返す
func distinctBase(t *Type) *Type {
	if t != nil && t.Base != nil {
		return t.Base
	}
	return t
}

// isVariableForm は ref 引数として渡せる (書き込み可能な) 変数形式か
func isVariableForm(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.Identifier, *ast.SelectorExpression, *ast.IndexExpression:
		return true
	}
	return false
}

func (c *Checker) checkBuiltinCall(name string, e *ast.CallExpression, env *Env) *Type {
	switch name {
	case "print":
		for _, arg := range e.Arguments {
			c.checkExpr(arg, env)
		}
		return VOID_TYPE

	case "len":
		if len(e.Arguments) != 1 {
			c.errAt(
				e.GetToken(),
				"wrong number of arguments. got=%d, want=1",
				len(e.Arguments),
			)
			return nil
		}

		t := c.checkExpr(e.Arguments[0], env)
		if t == nil {
			return nil
		}

		if t.Name != ARRAY_TYPE.Name && t.Name != STRING_TYPE.Name &&
			t.Name != ANY_TYPE.Name {
			c.errAt(
				e.GetToken(),
				"argument to `len` not supported, got %s",
				t.Name,
			)
			return nil
		}
		return INT_TYPE

	default:
		c.errAt(
			e.GetToken(),
			"undefined function: %s",
			name,
		)
		return nil
	}
}
