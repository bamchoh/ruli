package checker

import (
	"fmt"
	"ruli/ast"
	"ruli/lexer"
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

	// 現在チェック中の関数の戻り値型 (関数外なら nil)
	returnType *Type
	inFunction bool

	errors []TypeError
}

// Check はプログラム全体を型チェックし、見つかったエラーすべてを返す。
// SPEC 2.1 に従い、まずトップレベルの func 宣言を登録してから文を順にチェックする
func Check(program *ast.Program) []TypeError {
	c := &Checker{env: NewEnv()}

	// 宣言収集: 順不同の呼び出しと再帰を可能にする
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
	case *ast.IfStatement:
		c.checkIf(stmt, env)
	case *ast.ForStatement:
		c.checkFor(stmt, env)
	case *ast.IncDecStatement:
		c.checkIncDec(stmt, env)
	case *ast.FunctionStatement:
		c.checkFunctionBody(stmt, env)
	case *ast.ReturnStatement:
		c.checkReturn(stmt, env)
	case *ast.ExpressionStatement:
		c.checkExpr(stmt.Expression, env)
	case *ast.BreakStatement, *ast.ContinueStatement:
	}
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

		if stmt.Type != nil {
			dt := c.resolveTypeNode(stmt.Type)
			if dt == nil {
				return
			}

			if !dt.Equal(vt) {
				c.errAt(
					stmt.GetToken(),
					"type mismatch: declared %s, got %s",
					dt.Name,
					vt.Name,
				)
				return
			}

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
				declared.Name,
				"=",
				vt.Name,
			)
		}

	case *ast.IndexExpression:
		array := c.checkExpr(left.Left, env)
		if array == nil {
			return
		}

		if array.Name != ARRAY_TYPE.Name {
			c.errAt(left.GetToken(), "not an array")
			return
		}

		index := c.checkExpr(left.Index, env)
		if index != nil && index.Name != INT_TYPE.Name {
			c.errAt(
				left.Index.GetToken(),
				"index must be INT, got %s",
				index.Name,
			)
			return
		}

		vt := c.checkExpr(stmt.Value, env)
		if vt != nil && array.Elem != nil && !array.Elem.Equal(vt) {
			c.errAt(
				stmt.Value.GetToken(),
				"type mismatch: %s %s %s",
				array.Elem.Name,
				"=",
				vt.Name,
			)
		}
	}
}

func (c *Checker) checkReturn(stmt *ast.ReturnStatement, env *Env) {
	if !c.inFunction {
		c.errAt(stmt.GetToken(), "return outside function")
		return
	}

	vt := c.checkExpr(stmt.Value, env)
	if vt == nil {
		return
	}

	if c.returnType != nil && !c.returnType.Equal(vt) {
		c.errAt(
			stmt.GetToken(),
			"type mismatch: declared %s, got %s",
			c.returnType.Name,
			vt.Name,
		)
	}
}

// registerFunctionSignature は関数のシグネチャ型を現在スコープに登録する
func (c *Checker) registerFunctionSignature(stmt *ast.FunctionStatement, env *Env) *Type {
	params := make([]Type, 0, len(stmt.Parameters))

	for _, param := range stmt.Parameters {
		if param.Type == "" {
			params = append(params, *ANY_TYPE)
			continue
		}

		t, ok := resolveBasic(param.Type)
		if !ok {
			c.errAt(stmt.GetToken(), "unknown type: %s", param.Type)
			params = append(params, *ANY_TYPE)
			continue
		}
		params = append(params, *t)
	}

	var ret *Type
	if stmt.ReturnType != "" {
		if t, ok := resolveBasic(stmt.ReturnType); ok {
			ret = t
		} else {
			c.errAt(stmt.GetToken(), "unknown type: %s", stmt.ReturnType)
			ret = UNKNOWN_TYPE
		}
	}

	fn := &Type{Name: FUNCTION_TYPE.Name, Params: params, Return: ret}
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

	prevReturn, prevInFunction := c.returnType, c.inFunction
	c.returnType = fnType.Return
	c.inFunction = true

	c.checkStatements(stmt.Body.Statements, fnEnv)

	c.returnType, c.inFunction = prevReturn, prevInFunction
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
		c.errAt(n.GetToken(), "unknown type: %s", n.Name)
		return nil
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

		return &Type{Name: ARRAY_TYPE.Name, Elem: elem}

	case *ast.BinaryExpression:
		return c.checkBinary(e, env)

	case *ast.IndexExpression:
		array := c.checkExpr(e.Left, env)
		if array == nil {
			return nil
		}

		if array.Name != ARRAY_TYPE.Name {
			c.errAt(e.GetToken(), "not an array")
			return nil
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

	case *ast.CallExpression:
		return c.checkCall(e, env)
	}

	return nil
}

func (c *Checker) checkBinary(be *ast.BinaryExpression, env *Env) *Type {
	lt := c.checkExpr(be.Left, env)
	rt := c.checkExpr(be.Right, env)
	if lt == nil || rt == nil {
		return nil
	}

	switch be.Operator {
	case "+", "-", "*", "/":
		// SPEC 6.3: INT + REAL はエラー。明示キャスト必須
		if !lt.Equal(rt) {
			break
		}
		if isNumeric(lt) {
			return lt
		}
		if be.Operator == "+" && lt.Name == STRING_TYPE.Name {
			return lt
		}

	case "<", ">":
		if !lt.Equal(rt) {
			break
		}
		if isNumeric(lt) || lt.Name == STRING_TYPE.Name {
			return BOOL_TYPE
		}

	case "==", "!=":
		if lt.Equal(rt) {
			return BOOL_TYPE
		}
	}

	c.errAt(
		be.GetToken(),
		"type mismatch: %s %s %s",
		lt.Name,
		be.Operator,
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
			// ユーザ定義ではない → 組み込み関数
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
		if fnType.Params[i].Name == ANY_TYPE.Name {
			continue
		}
		if !argTypes[i].Equal(&fnType.Params[i]) {
			c.errAt(
				e.Arguments[i].GetToken(),
				"type mismatch: argument %d: expected %s, got %s",
				i+1,
				fnType.Params[i].Name,
				argTypes[i].Name,
			)
		}
	}

	if fnType.Return == nil {
		return VOID_TYPE
	}
	return fnType.Return
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

		if t.Name != ARRAY_TYPE.Name && t.Name != STRING_TYPE.Name {
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
