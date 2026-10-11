package evaluator

import (
	"fmt"
	"io"
	"math"
	"ruli/ast"
	"ruli/object"
	"strings"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

var builtins = map[string]*object.Builtin{
	"print": {
		Fn: func(args ...object.Object) (object.Object, error) {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return &object.Null{}, nil
		},
	},

	"len": {
		Fn: func(args ...object.Object) (object.Object, error) {

			if len(args) != 1 {
				return &object.Null{}, fmt.Errorf("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {

			case *object.Array:
				return &object.Integer{
					Value: (len(arg.Elements)),
				}, nil

			case *object.String:
				return &object.Integer{
					Value: (len([]rune(arg.Value))),
				}, nil
			}

			return &object.Null{}, fmt.Errorf("argument to `len` not supported, got %s", args[0].Type())
		},
	},
}

// InstallBuiltins は組み込み関数 (print / len) を環境へ登録する。
// out は print の出力先。環境に登録された組込みはホスト関数や
// ユーザ定義によって上書きできる (ユーザ定義が優先)
func InstallBuiltins(env *object.Environment, w io.Writer) {
	env.Set("print", &object.Builtin{
		Fn: func(args ...object.Object) (object.Object, error) {
			for _, arg := range args {
				fmt.Fprintln(w, arg.Inspect())
			}
			return &object.Null{}, nil
		},
	})

	env.Set("len", &object.Builtin{
		Fn: func(args ...object.Object) (object.Object, error) {

			if len(args) != 1 {
				return &object.Null{}, fmt.Errorf("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {

			case *object.Array:
				return &object.Integer{
					Value: (len(arg.Elements)),
				}, nil

			case *object.String:
				return &object.Integer{
					Value: (len([]rune(arg.Value))),
				}, nil
			}

			return &object.Null{}, fmt.Errorf("argument to `len` not supported, got %s", args[0].Type())
		},
	})
}

func Eval(node ast.Node, env *object.Environment) object.Object {

	if node == nil {
		return &object.Null{}
	}

	switch node := node.(type) {

	case *ast.Program:
		return evalProgram(node, env)

	case *ast.VarDeclStatement:
		var val object.Object

		switch {
		case node.Value != nil:
			val = Eval(node.Value, env)
			if isError(val) {
				return val
			}
		case node.Type != nil:
			// 型だけの宣言はゼロ値で初期化する (m : Motor)
			val = zeroValueForTypeNode(node.Type, env)
		default:
			val = &object.Null{}
		}

		// SPEC 5.4: 複数戻り値の単一変数への束縛は禁止
		if mv, ok := val.(*object.MultiValue); ok {
			return newErrorAtNode(
				node,
				"cannot assign %d values to %d variables",
				len(mv.Values),
				1,
			)
		}

		// 型名が明示された宣言は実行時幅チェックの対象になる (x : INT = 10)
		if basic, ok := node.Type.(*ast.BasicType); ok {
			if errObj := checkWidth(node, basic.Name, val); errObj != nil {
				return errObj
			}
			env.DeclareTyped(node.Name.Value, bindValue(val), basic.Name)
			return nullObject
		}

		env.Set(node.Name.Value, bindValue(val))
		return nullObject

	case *ast.TypeDeclStatement:
		// 実行時の型定義を登録する (ゼロ値構築に使用)
		if node.Base != nil {
			// distinct 型 (SPEC 4.2)
			env.DefineType(node.Name, &object.StructDef{
				Base: basicTypeName(node.Base),
			})
			return &object.Null{}
		}

		def := &object.StructDef{
			FieldNames: make([]string, 0, len(node.Fields)),
			FieldTypes: make(map[string]ast.TypeNode, len(node.Fields)),
		}
		for _, field := range node.Fields {
			def.FieldNames = append(def.FieldNames, field.Name)
			def.FieldTypes[field.Name] = field.Type
		}
		env.DefineType(node.Name, def)
		return &object.Null{}

	case *ast.AssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		if res := evalAssignStatement(node, val, env); isError(res) {
			return res
		}
		return nullObject

	case *ast.CompoundAssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		if res := evalCompoundAssignStatement(node, val, env); isError(res) {
			return res
		}
		return nullObject

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.RealLiteral:
		return &object.Real{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.Identifier:
		// 環境を優先 (ホスト関数やユーザ定義が組み込みを上書きできる)
		if v, ok := env.Get(node.Value); ok {
			return v
		}

		if b, ok := builtins[node.Value]; ok {
			return b
		}

		return newErrorAtNode(node, "identifier not found: %s", node.Value)

	case *ast.BinaryExpression:
		return evalBinaryExpression(node, env)

	case *ast.LogicalExpression:
		return evalLogicalExpression(node, env)

	case *ast.IfStatement:
		return evalIfStatement(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.IncDecStatement:
		return evalIncDecStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.SwitchStatement:
		return evalSwitchStatement(node, env)

	case *ast.CallExpression:
		return evalCallExpression(node, env)

	case *ast.StructLiteral:
		return evalStructLiteral(node, env)

	case *ast.CastExpression:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		return evalCast(node, val, env)

	case *ast.SelectorExpression:
		obj := Eval(node.Object, env)
		if isError(obj) {
			return obj
		}
		return evalSelectorExpression(node, obj)

	case *ast.PrefixExpression:
		return evalPrefixExpression(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.BreakStatement:
		return &object.BreakSignal{}

	case *ast.ContinueStatement:
		return &object.ContinueSignal{}

	case *ast.FunctionStatement:
		fn := &object.Function{
			Parameters: node.Parameters,
			Receiver:   node.Receiver,
			Body:       node.Body,
			Env:        env,
		}

		// メソッド (SPEC 5.6) は struct 型定義に登録する
		if node.Receiver != nil {
			if typeName := methodReceiverTypeName(node.Receiver); typeName != "" {
				if def, ok := env.GetTypeDef(typeName); ok {
					if def.Methods == nil {
						def.Methods = make(map[string]*object.Function)
					}
					def.Methods[node.Name] = fn
					return nullObject
				}

				return newErrorAtNode(node, "undefined struct type: %s", typeName)
			}

			return newErrorAtNode(node, "method receiver requires a type")
		}

		env.Set(node.Name, fn)
		return nullObject

	case *ast.ReturnStatement:
		values := make([]object.Object, 0, len(node.Values))
		for _, valueExpr := range node.Values {
			val := Eval(valueExpr, env)
			if isError(val) {
				return val
			}
			values = append(values, val)
		}
		return &object.ReturnValue{Values: values}

	case *ast.MultiVarDeclStatement:
		val := Eval(node.Values, env)
		if isError(val) {
			return val
		}
		return bindMultiValues(node, namesOf(node.Names), val, env, false)

	case *ast.MultiAssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		return bindMultiValues(node, node.Lefts, val, env, true)

	case *ast.ArrayLiteral:
		var elements []object.Object

		for _, el := range node.Elements {
			elements = append(elements, Eval(el, env))
		}

		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}

		return evalIndexExpression(node, left, index)

	}

	return newErrorAtNode(node, "unsupported node type: %T", node)
}

func namesOf(names []*ast.Identifier) []ast.Expression {
	out := make([]ast.Expression, 0, len(names))
	for _, name := range names {
		out = append(out, name)
	}
	return out
}

// bindMultiValues は複数戻り値を変数リストへ束縛する (SPEC 5.4)。
// 個数が合わない、または多値でない値の複数束縛はエラー
func bindMultiValues(
	node ast.Node,
	names []ast.Expression,
	val object.Object,
	env *object.Environment,
	assign bool,
) object.Object {

	values := []object.Object{val}
	if mv, ok := val.(*object.MultiValue); ok {
		values = mv.Values
	}

	if len(values) != len(names) {
		return newErrorAtNode(
			node,
			"cannot assign %d values to %d variables",
			len(values),
			len(names),
		)
	}

	for i, name := range names {
		ident, ok := name.(*ast.Identifier)
		if !ok {
			return newErrorAtNode(name, "invalid assignment target")
		}

		stored := bindValue(values[i])

		if assign {
			if typeName, ok := env.VarType(ident.Value); ok {
				if errObj := checkWidth(name, typeName, stored); errObj != nil {
					return errObj
				}
			}
			if !env.Assign(ident.Value, stored) {
				return newErrorAtNode(name, "undefined variable: %s", ident.Value)
			}
		} else {
			env.Set(ident.Value, stored)
		}
	}

	return &object.Null{}
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object = &object.Null{}

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
		if isError(result) {
			return result
		}
	}

	return result
}

func evalBinaryExpression(be *ast.BinaryExpression, env *object.Environment) object.Object {

	left := Eval(be.Left, env)
	if isError(left) {
		return left
	}

	right := Eval(be.Right, env)
	if isError(right) {
		return right
	}

	return applyBinaryOperator(be.Operator, left, right, be)
}

// applyBinaryOperator は等しい型同士の二項演算を実行する。
// 複合代入 (x += 1) は評価済みの左辺値とここを共有する
func applyBinaryOperator(
	operator string,
	left, right object.Object,
	node ast.Node,
) object.Object {

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerBinaryExpression(operator, left, right, node)
	case left.Type() == object.REAL_OBJ && right.Type() == object.REAL_OBJ:
		return evalRealBinaryExpression(operator, left, right, node)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right, node)
	}

	return newErrorAtNode(
		node,
		"type mismatch: %s %s %s",
		left.Type(),
		operator,
		right.Type(),
	)
}

// evalLogicalExpression は && / || を short-circuit 評価する (SPEC 6.2)。
// 左辺だけで結果が確定する場合、右辺は評価されない
func evalLogicalExpression(node *ast.LogicalExpression, env *object.Environment) object.Object {

	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	lb, ok := left.(*object.Boolean)
	if !ok {
		return newErrorAtNode(
			node,
			"operand of %s must be BOOL, got %s",
			node.Operator,
			left.Type(),
		)
	}

	// short-circuit
	if node.Operator == "&&" && !lb.Value {
		return FALSE
	}
	if node.Operator == "||" && lb.Value {
		return TRUE
	}

	right := Eval(node.Right, env)
	if isError(right) {
		return right
	}

	rb, ok := right.(*object.Boolean)
	if !ok {
		return newErrorAtNode(
			node,
			"operand of %s must be BOOL, got %s",
			node.Operator,
			right.Type(),
		)
	}

	switch node.Operator {
	case "&&":
		return nativeBoolToBooleanObject(lb.Value && rb.Value)
	case "||":
		return nativeBoolToBooleanObject(lb.Value || rb.Value)
	}

	return newErrorAtNode(node, "unsupported operator: %s", node.Operator)
}

func evalIntegerBinaryExpression(
	operator string,
	left, right object.Object,
	node ast.Node,
) object.Object {

	lv := left.(*object.Integer).Value
	rv := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: lv + rv}
	case "-":
		return &object.Integer{Value: lv - rv}
	case "*":
		return &object.Integer{Value: lv * rv}
	case "/":
		if rv == 0 {
			return newErrorAtNode(node, "division by zero")
		}
		return &object.Integer{Value: lv / rv}
	case "%":
		if rv == 0 {
			return newErrorAtNode(node, "division by zero")
		}
		return &object.Integer{Value: lv % rv}

	case ">":
		return nativeBoolToBooleanObject(lv > rv)
	case "<":
		return nativeBoolToBooleanObject(lv < rv)
	case "<=":
		return nativeBoolToBooleanObject(lv <= rv)
	case ">=":
		return nativeBoolToBooleanObject(lv >= rv)
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(node, "unsupported operator: %s", operator)
}

func evalRealBinaryExpression(
	operator string,
	left, right object.Object,
	node ast.Node,
) object.Object {

	lv := left.(*object.Real).Value
	rv := right.(*object.Real).Value

	switch operator {
	case "+":
		return &object.Real{Value: lv + rv}
	case "-":
		return &object.Real{Value: lv - rv}
	case "*":
		return &object.Real{Value: lv * rv}
	case "/":
		if rv == 0 {
			return newErrorAtNode(node, "division by zero")
		}
		return &object.Real{Value: lv / rv}
	case "%":
		if rv == 0 {
			return newErrorAtNode(node, "division by zero")
		}
		return &object.Real{Value: math.Mod(lv, rv)}

	case ">":
		return nativeBoolToBooleanObject(lv > rv)
	case "<":
		return nativeBoolToBooleanObject(lv < rv)
	case "<=":
		return nativeBoolToBooleanObject(lv <= rv)
	case ">=":
		return nativeBoolToBooleanObject(lv >= rv)
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(node, "unsupported operator: %s", operator)
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
	node ast.Node,
) object.Object {
	lv := left.(*object.String).Value
	rv := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: lv + rv}
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(node, "unsupported operator: %s", operator)
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

// evalSwitchStatement は switch 文 (SPEC 7.4) を実行する。fallthrough なし
func evalSwitchStatement(node *ast.SwitchStatement, env *object.Environment) object.Object {

	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	for _, sc := range node.Cases {
		for _, value := range sc.Values {
			cv := Eval(value, env)
			if isError(cv) {
				return cv
			}

			if valuesEqual(val, cv) {
				return evalSwitchCaseBody(sc.Body, env)
			}
		}
	}

	if node.Default != nil {
		return evalSwitchCaseBody(node.Default.Body, env)
	}

	return &object.Null{}
}

// evalSwitchCaseBody は case 本体を実行する。
// break は switch を抜ける (内側の break はここで消化される)
func evalSwitchCaseBody(body *ast.BlockStatement, env *object.Environment) object.Object {
	result := Eval(body, env)

	if result != nil && result.Type() == object.BREAK_OBJ {
		return &object.Null{}
	}

	return result // RETURN / CONTINUE は外側へ伝播する
}

// valuesEqual は switch の一致判定 (型が異なれば不一致)
func valuesEqual(a, b object.Object) bool {
	switch x := a.(type) {
	case *object.Integer:
		y, ok := b.(*object.Integer)
		return ok && x.Value == y.Value
	case *object.Real:
		y, ok := b.(*object.Real)
		return ok && x.Value == y.Value
	case *object.Boolean:
		y, ok := b.(*object.Boolean)
		return ok && x.Value == y.Value
	case *object.String:
		y, ok := b.(*object.String)
		return ok && x.Value == y.Value
	}
	return false
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	// {} は新しいスコープを作る (SPEC 3.1 / 3.4: := 宣言は現在スコープに作成)
	blockEnv := object.NewEnclosedEnvironment(env)

	var result object.Object = &object.Null{}

	for _, stmt := range block.Statements {
		result = Eval(stmt, blockEnv)
		if isError(result) {
			return result
		}

		if result != nil {
			rt := result.Type()
			if rt == object.BREAK_OBJ || rt == object.CONTINUE_OBJ || rt == object.RETURN_OBJ {
				return result
			}
		}
	}

	return result
}

func evalIfStatement(stmt *ast.IfStatement, env *object.Environment) object.Object {

	cond := Eval(stmt.Condition, env)
	if isError(cond) {
		return cond
	}

	if isTruthy(cond) {
		return Eval(stmt.Consequence, env)
	}

	if stmt.Alternative != nil {
		return Eval(stmt.Alternative, env)
	}

	return &object.Null{}
}

func isTruthy(obj object.Object) bool {

	switch v := obj.(type) {
	case *object.Boolean:
		return v.Value
	case *object.Integer:
		return v.Value != 0
	case *object.Null:
		return false
	}

	return false
}

func evalIncDecStatement(stmt *ast.IncDecStatement, env *object.Environment) object.Object {

	v, ok := env.Get(stmt.Name)
	if !ok {
		return newErrorAtNode(stmt, "identifier not found: %s", stmt.Name)
	}

	obj, ok := v.(*object.Integer)
	if !ok {
		return newErrorAtNode(
			stmt,
			"operand of %s must be an integer, got %s",
			stmt.Operator,
			v.Type(),
		)
	}

	proposed := obj.Value
	if stmt.Operator == "++" {
		proposed++
	} else {
		proposed--
	}

	// 宣言時型に幅があれば範囲外はエラー (桁あふれない)
	if typeName, ok := env.VarType(stmt.Name); ok {
		candidate := &object.Integer{Value: proposed}
		if errObj := checkWidth(stmt, typeName, candidate); errObj != nil {
			return errObj
		}
	}

	obj.Value = proposed

	return obj
}

func evalForStatement(stmt *ast.ForStatement, env *object.Environment) object.Object {

	// for 句 (init で宣言した変数を含む) は for 全体のスコープに閉じる
	loopEnv := object.NewEnclosedEnvironment(env)

	initret := Eval(stmt.Init, loopEnv)
	if isError(initret) {
		return initret
	}

	for {
		cond := Eval(stmt.Condition, loopEnv)
		if isError(cond) {
			return cond
		}

		if !isTruthy(cond) {
			break
		}

		result := Eval(stmt.Body, loopEnv)
		if isError(result) {
			return result
		}

		if result != nil {
			switch result.Type() {
			case object.BREAK_OBJ:
				return &object.Null{}
			case object.RETURN_OBJ:
				return result

			case object.CONTINUE_OBJ:
				ret := Eval(stmt.Post, loopEnv)
				if isError(ret) {
					return ret
				}
				continue
			}
		}

		postret := Eval(stmt.Post, loopEnv)
		if isError(postret) {
			return postret
		}
	}

	return &object.Null{}
}

// evalPrefixExpression は -x / !flag のような単項演算を実行する
func evalPrefixExpression(node *ast.PrefixExpression, env *object.Environment) object.Object {

	right := Eval(node.Right, env)
	if isError(right) {
		return right
	}

	switch node.Operator {
	case "-":
		switch v := right.(type) {
		case *object.Integer:
			return &object.Integer{Value: -v.Value}
		case *object.Real:
			return &object.Real{Value: -v.Value}
		}
		return newErrorAtNode(node, "operand of - must be numeric, got %s", right.Type())

	case "!":
		if b, ok := right.(*object.Boolean); ok {
			return nativeBoolToBooleanObject(!b.Value)
		}
		return newErrorAtNode(node, "operand of ! must be BOOL, got %s", right.Type())
	}

	return newErrorAtNode(node, "unsupported prefix operator: %s", node.Operator)
}

func evalCallExpression(node *ast.CallExpression, env *object.Environment) object.Object {

	// メソッド呼び出し (SPEC 5.6): m.reset()
	if sel, ok := node.Function.(*ast.SelectorExpression); ok {
		if result := evalMethodCall(node, sel, env); result != nil {
			return result
		}
	}

	// ユーザ定義型へのキャスト (SPEC 4.2): TEMP(30)
	if ident, ok := node.Function.(*ast.Identifier); ok {
		if _, isBuiltin := builtins[ident.Value]; !isBuiltin {
			if _, defined := env.Get(ident.Value); !defined {
				if _, isType := env.GetTypeDef(ident.Value); isType {
					if len(node.Arguments) != 1 {
						return newErrorAtNode(
							node,
							"wrong number of arguments. got=%d, want=1",
							len(node.Arguments),
						)
					}

					val := Eval(node.Arguments[0], env)
					if isError(val) {
						return val
					}
					return evalCastTo(ident.Value, val, env, node)
				}
			}
		}
	}

	function := Eval(node.Function, env)

	var args []object.Object
	for _, arg := range node.Arguments {
		arg := Eval(arg, env)
		if isError(arg) {
			return arg
		}
		args = append(args, arg)
	}

	switch fn := function.(type) {

	case *object.Builtin:
		result, err := fn.Fn(args...)
		if err != nil {
			return newErrorAtNode(node, "%s", err.Error())
		}
		return result

	case *object.Function:
		if len(args) != len(fn.Parameters) {
			return newErrorAtNode(
				node,
				"wrong number of arguments. got=%d, want=%d",
				len(args),
				len(fn.Parameters),
			)
		}
		return applyFunction(fn, args, node, env, nil, nil)
	}

	return newErrorAtNode(node, "not a function: %s", function.Type())
}

// evalCast は REAL(a) のような明示的キャストを実行する (SPEC 6.3)。
// 静的な型チェックは checker が担当し、ここでは値変換のみを行う
func evalCast(
	node *ast.CastExpression,
	val object.Object,
	env *object.Environment,
) object.Object {
	return evalCastTo(node.Type, val, env, node)
}

// evalCastTo はキャスト実行の本体。CastExpression と
// ユーザ型への呼び出し形式キャスト (SPEC 4.2: TEMP(30)) が共有する
func evalCastTo(
	target string,
	val object.Object,
	env *object.Environment,
	node ast.Node,
) object.Object {

	// distinct 型 (SPEC 4.2) は基本型へ解決してから変換する
	targetName := target
	if def, ok := env.GetTypeDef(target); ok {
		switch {
		case def.Base != "":
			targetName = def.Base
		default:
			return newErrorAtNode(node, "cannot cast to struct: %s", target)
		}
	}

	switch targetName {
	case "REAL", "LREAL":
		switch v := val.(type) {
		case *object.Real:
			return v
		case *object.Integer:
			return &object.Real{Value: float64(v.Value)}
		}

	case "BOOL":
		if b, ok := val.(*object.Boolean); ok {
			return b
		}

	default:
		// 整数族へのキャスト (SINT〜LWORD)。範囲外はエラー (桁あふれはしない)
		var conv int
		switch v := val.(type) {
		case *object.Integer:
			conv = v.Value
		case *object.Real:
			// 実数 → 整数は 0 方向へ切り捨て
			conv = int(v.Value)
		default:
			return newErrorAtNode(node, "invalid cast: %s to %s", val.Type(), target)
		}

		if errObj := checkWidth(node, targetName, &object.Integer{Value: conv}); errObj != nil {
			return errObj
		}

		return &object.Integer{Value: conv}
	}

	return newErrorAtNode(node, "invalid cast: %s to %s", val.Type(), targetName)
}

// evalStructLiteral は Motor{speed: 10} を実行する。
// 未指定フィールドはゼロ値で埋まる (Go 同様)
func evalStructLiteral(
	node *ast.StructLiteral,
	env *object.Environment,
) object.Object {

	def, ok := env.GetTypeDef(node.Name)
	if !ok || def.Base != "" {
		return newErrorAtNode(node, "undefined struct type: %s", node.Name)
	}

	// ゼロ値の struct から始めて、指定フィールドを上書きする
	zero := zeroValue(env, node.Name)
	s, ok := zero.(*object.Struct)
	if !ok {
		return newErrorAtNode(node, "undefined struct type: %s", node.Name)
	}
	s.TypeName = node.Name

	for _, field := range node.Fields {
		if _, ok := s.Fields[field.Name]; !ok {
			return newErrorAtNode(field.Value, "undefined field: %s", field.Name)
		}

		val := Eval(field.Value, env)
		if isError(val) {
			return val
		}

		s.Fields[field.Name] = bindValue(val)
	}

	return s
}

// nullObject は評価結果が不要な文の戻り値
var nullObject = &object.Null{}

// evalSelectorExpression は m.speed のようなフィールド読み出し
func evalSelectorExpression(
	node *ast.SelectorExpression,
	obj object.Object,
) object.Object {

	s, ok := obj.(*object.Struct)
	if !ok {
		return newErrorAtNode(node, "no field '%s' on %s", node.Field, obj.Type())
	}

	v, ok := s.Fields[node.Field]
	if !ok {
		return newErrorAtNode(node, "undefined field: %s", node.Field)
	}

	return v
}

// zeroValueForTypeNode は m : Motor や values : [10]INT のような
// 初期値なし宣言のゼロ値を作る
func zeroValueForTypeNode(node ast.TypeNode, env *object.Environment) object.Object {
	switch t := node.(type) {
	case *ast.ArrayType:
		// SPEC 4.4: 固定長配列のゼロ値 (0 埋め)
		elements := make([]object.Object, t.Size)
		for i := range elements {
			elements[i] = zeroValueForTypeNode(t.Elem, env)
		}
		return &object.Array{Elements: elements}

	case *ast.BasicType:
		return zeroValue(env, t.Name)
	}

	return &object.Null{}
}

func zeroValue(env *object.Environment, typeName string) object.Object {
	if isIntFamilyName(typeName) {
		return &object.Integer{}
	}

	switch typeName {
	case "REAL", "LREAL":
		return &object.Real{}
	case "BOOL":
		return FALSE
	case "STRING":
		return &object.String{}
	}

	if def, ok := env.GetTypeDef(typeName); ok {
		// distinct 型のゼロ値は基本型から構築する (SPEC 4.2)
		if def.Base != "" {
			return zeroValue(env, def.Base)
		}

		fields := make(map[string]object.Object, len(def.FieldTypes))
		for name, ft := range def.FieldTypes {
			fields[name] = zeroValueForTypeNode(ft, env)
		}
		return &object.Struct{Fields: fields, TypeName: typeName}
	}

	return &object.Null{}
}

// 数値型 (整数族) 名の判定。SPEC 4.1 の固定幅型
func isIntFamilyName(name string) bool {
	switch name {
	case "INT", "SINT", "USINT", "BYTE", "UINT", "WORD",
		"DINT", "UDINT", "DWORD", "LINT", "ULINT", "LWORD":
		return true
	}
	return false
}

// evalMethodCall はメソッド呼び出しを解決して適用する。
// 非structレシーバなら nil を返す (通常の呼び出しにフォールバック)
func evalMethodCall(
	node *ast.CallExpression,
	sel *ast.SelectorExpression,
	env *object.Environment,
) object.Object {

	recvObj := Eval(sel.Object, env)
	if isError(recvObj) {
		return recvObj
	}

	// 型名は宣言時型から、なければ値が保持する struct 型名から解決する
	// (m := Motor{...} のような型推論束縛)
	typeName := typeNameOfExpr(sel.Object, env)
	if typeName == "" {
		if sv, ok := recvObj.(*object.Struct); ok {
			typeName = sv.TypeName
		}
	}

	s, ok := recvObj.(*object.Struct)
	if !ok || typeName == "" || s.TypeName == "" {
		return nil
	}

	def, ok := env.GetTypeDef(typeName)
	if !ok || def.Methods == nil {
		return newErrorAtNode(node, "undefined method: %s.%s", typeName, sel.Field)
	}

	method, ok := def.Methods[sel.Field]
	if !ok {
		return newErrorAtNode(node, "undefined method: %s.%s", typeName, sel.Field)
	}

	args := make([]object.Object, 0, len(node.Arguments))
	for _, argExpr := range node.Arguments {
		val := Eval(argExpr, env)
		if isError(val) {
			return val
		}
		args = append(args, val)
	}

	if len(args) != len(method.Parameters) {
		return newErrorAtNode(
			node,
			"wrong number of arguments. got=%d, want=%d",
			len(args),
			len(method.Parameters),
		)
	}

	// レシーバはレシーバスロットとして束縛する (Parameters には含まれない)
	// レシーバはレシーバスロットとして束縛する (Parameters には含まれない)
	return applyFunction(method, args, node, env, method.Receiver, recvObj)
}

// methodReceiverTypeName はメソッドレシーバの struct 型名を返す (解決できなければ "")
func methodReceiverTypeName(recv *ast.Parameter) string {
	if recv.Type == nil {
		return ""
	}
	if bt, ok := recv.Type.(*ast.BasicType); ok {
		return bt.Name
	}
	return ""
}

func applyFunction(
	fn *object.Function,
	args []object.Object,
	callNode *ast.CallExpression,
	callEnv *object.Environment,
	receiverParam *ast.Parameter,
	receiverVal object.Object,
) object.Object {

	extendedEnv := object.NewEnclosedEnvironment(fn.Env)

	// メソッドレシーバ (SPEC 5.6): 値渡し (コピー) / ref レシーバは共有
	if receiverParam != nil {
		var recvStored object.Object
		if receiverParam.Ref {
			recvStored = receiverVal
		} else {
			recvStored = bindValue(receiverVal)
		}

		extendedEnv.Set(receiverParam.Name, recvStored)

		if bt, ok := receiverParam.Type.(*ast.BasicType); ok {
			extendedEnv.DeclareTyped(receiverParam.Name, recvStored, bt.Name)
		}
	}

	for i, param := range fn.Parameters {

		// ref パラメータ (SPEC 8.1 / IEC VAR_IN_OUT) は引数を共有する。
		// 配列 / struct はエイリアスで即時反映、スカラーは終了時に書き戻す
		if param.Ref {
			if bt, ok := param.Type.(*ast.BasicType); ok {
				extendedEnv.DeclareTyped(param.Name, args[i], bt.Name)
				continue
			}
			extendedEnv.Set(param.Name, args[i])
			continue
		}

		stored := bindValue(args[i])

		// 型付きパラメータは実行時幅チェックと型名の記録を行う
		if bt, ok := param.Type.(*ast.BasicType); ok {
			if errObj := checkWidth(fn.Body, bt.Name, stored); errObj != nil {
				return errObj
			}
			extendedEnv.DeclareTyped(param.Name, stored, bt.Name)
			continue
		}

		extendedEnv.Set(param.Name, stored)
	}

	result := Eval(fn.Body, extendedEnv)
	if isError(result) {
		return result
	}

	// スカラー ref パラメータの書き戻し (正常終了時のみ)
	writeBackRefs(fn, callNode, callEnv, extendedEnv)

	if rv, ok := result.(*object.ReturnValue); ok {
		if len(rv.Values) == 1 {
			return rv.Values[0]
		}
		return &object.MultiValue{Values: rv.Values}
	}

	return result
}

func evalIndexExpression(node *ast.IndexExpression, left, index object.Object) object.Object {

	switch {
	case left.Type() == object.ARRAY_OBJ &&
		index.Type() == object.INTEGER_OBJ:

		return evalArrayIndexExpression(node, left, index)
	}

	return newErrorAtNode(node, "not an array")
}

func evalArrayIndexExpression(node *ast.IndexExpression, array, index object.Object) object.Object {

	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value

	max := len(arrayObject.Elements) - 1

	if idx < 0 || idx > max {
		return newErrorAtNode(node, "index out of range: %d", idx)
	}

	return arrayObject.Elements[idx]
}

func evalAssignStatement(
	stmt *ast.AssignStatement,
	val object.Object,
	env *object.Environment,
) object.Object {

	switch left := stmt.Left.(type) {

	case *ast.Identifier:
		// = は宣言ではなく更新。未定義変数への代入は SPEC 4.1.1 でエラー
		stored := bindValue(val)

		// 宣言時に型が明示された変数は実行時幅チェックを行う
		if typeName, ok := env.VarType(left.Value); ok {
			if errObj := checkWidth(stmt, typeName, stored); errObj != nil {
				return errObj
			}
		}

		if !env.Assign(left.Value, stored) {
			return newErrorAtNode(stmt, "undefined variable: %s", left.Value)
		}
		return val

	case *ast.IndexExpression:
		return evalIndexAssign(left, val, env)

	case *ast.SelectorExpression:
		return evalSelectorAssign(left, val, env)
	}

	return newErrorAtNode(stmt, "invalid assignment target")
}

// evalCompoundAssignStatement は x += 1 / a[0] *= 2 / m.speed -= 1 を実行する (SPEC 6.2)。
// 左辺は一度だけ評価され、現在値と右辺に演算子を適用して書き戻す
func evalCompoundAssignStatement(
	stmt *ast.CompoundAssignStatement,
	val object.Object,
	env *object.Environment,
) object.Object {

	operator := strings.TrimSuffix(stmt.Operator, "=")

	mutate := func(old object.Object) object.Object {
		return applyBinaryOperator(operator, old, val, stmt)
	}

	switch target := stmt.Target.(type) {

	case *ast.Identifier:
		old, ok := env.Get(target.Value)
		if !ok {
			return newErrorAtNode(stmt, "undefined variable: %s", target.Value)
		}

		stored := mutate(old)
		if isError(stored) {
			return stored
		}

		// 宣言時に型が明示された変数は実行時幅チェックを行う
		if typeName, ok := env.VarType(target.Value); ok {
			if errObj := checkWidth(stmt, typeName, stored); errObj != nil {
				return errObj
			}
		}

		if !env.Assign(target.Value, bindValue(stored)) {
			return newErrorAtNode(stmt, "undefined variable: %s", target.Value)
		}
		return stored

	case *ast.IndexExpression:
		return evalIndexWrite(target, env, func(old object.Object) object.Object {
			stored := mutate(old)
			if isError(stored) {
				return stored
			}
			return bindValue(stored)
		})

	case *ast.SelectorExpression:
		return evalSelectorWrite(target, env, func(old object.Object) object.Object {
			stored := mutate(old)
			if isError(stored) {
				return stored
			}
			return bindValue(stored)
		})
	}

	return newErrorAtNode(stmt, "invalid assignment target")
}

// evalSelectorAssign は m.speed = 10 のようなフィールドへの代入
func evalSelectorAssign(
	left *ast.SelectorExpression,
	val object.Object,
	env *object.Environment,
) object.Object {
	return evalSelectorWrite(left, env, func(old object.Object) object.Object {
		return bindValue(val)
	})
}

// evalSelectorWrite は struct フィールドへの書き込みを共通化する。
// mutate は現在のフィールド値を受け取り、格納する値を返す
func evalSelectorWrite(
	left *ast.SelectorExpression,
	env *object.Environment,
	mutate func(old object.Object) object.Object,
) object.Object {

	obj := Eval(left.Object, env)
	if isError(obj) {
		return obj
	}

	s, ok := obj.(*object.Struct)
	if !ok {
		return newErrorAtNode(left, "no field '%s' on %s", left.Field, obj.Type())
	}

	old, ok := s.Fields[left.Field]
	if !ok {
		return newErrorAtNode(left, "undefined field: %s", left.Field)
	}

	// フィールドが固定幅型なら実行時幅チェックする
	if parentType := typeNameOfExpr(left.Object, env); parentType != "" {
		if def, ok := env.GetTypeDef(parentType); ok {
			if fieldType, ok := def.FieldTypes[left.Field]; ok {
				stored := mutate(old)
				if isError(stored) {
					return stored
				}
				if errObj := checkWidth(left, basicTypeName(fieldType), stored); errObj != nil {
					return errObj
				}
				s.Fields[left.Field] = stored
				return stored
			}
		}
	}

	stored := mutate(old)
	if isError(stored) {
		return stored
	}

	s.Fields[left.Field] = stored

	return stored
}

// basicTypeName は TypeNode から幅チェック可能な基本型名を取り出す
// (配列型などは空文字を返し、幅チェックは行われない)
func basicTypeName(node ast.TypeNode) string {
	if bt, ok := node.(*ast.BasicType); ok {
		return bt.Name
	}
	return ""
}

// typeNameOfExpr は式が参照する変数の宣言時型名を返す (判定不能なら "")
// セレクタの親をたどって struct フィールドの型名も解決する
func typeNameOfExpr(expr ast.Expression, env *object.Environment) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return envTypeName(e, env)
	case *ast.SelectorExpression:
		parent := typeNameOfExpr(e.Object, env)
		if parent == "" {
			return ""
		}
		if def, ok := env.GetTypeDef(parent); ok {
			return basicTypeName(def.FieldTypes[e.Field])
		}
	}
	return ""
}

func envTypeName(e *ast.Identifier, env *object.Environment) string {
	if t, ok := env.VarType(e.Value); ok {
		return t
	}
	return ""
}

// writeBackRefs はスカラー ref パラメータ (SPEC 8.1 / IEC VAR_IN_OUT) を
// 呼び出し元の変数へ書き戻す。配列 / struct は引数共有 (エイリアス) 的に
// 即時反映のため書き戻しは不要
func writeBackRefs(
	fn *object.Function,
	callNode *ast.CallExpression,
	callEnv *object.Environment,
	fnEnv *object.Environment,
) {

	for i, param := range fn.Parameters {
		if !param.Ref || i >= len(callNode.Arguments) {
			continue
		}

		value, ok := fnEnv.Get(param.Name)
		if !ok {
			continue
		}

		// スカラーのみ書き戻す (配列 / struct はエイリアスで反映済み)
		switch value.(type) {
		case *object.Integer, *object.Real, *object.Boolean, *object.String:
		default:
			continue
		}

		if i >= len(callNode.Arguments) {
			continue
		}

		arg := callNode.Arguments[i]
		switch target := arg.(type) {

		case *ast.Identifier:
			callEnv.Assign(target.Value, value)

		case *ast.SelectorExpression:
			evalSelectorWrite(target, callEnv, func(old object.Object) object.Object {
				return value
			})

		case *ast.IndexExpression:
			evalIndexWrite(target, callEnv, func(old object.Object) object.Object {
				return value
			})
		}
	}
}

func evalIndexAssign(
	left *ast.IndexExpression,
	val object.Object,
	env *object.Environment,
) object.Object {
	return evalIndexWrite(left, env, func(old object.Object) object.Object {
		return bindValue(val)
	})
}

// evalIndexWrite は配列要素への書き込みを共通化する。
// mutate は現在の要素値を受け取り、格納する値 (エラー時は error オブジェクト) を返す
func evalIndexWrite(
	left *ast.IndexExpression,
	env *object.Environment,
	mutate func(old object.Object) object.Object,
) object.Object {

	arrayObj := Eval(left.Left, env)
	if isError(arrayObj) {
		return arrayObj
	}

	indexObj := Eval(left.Index, env)
	if isError(indexObj) {
		return indexObj
	}

	array, ok := arrayObj.(*object.Array)
	if !ok {
		return newErrorAtNode(left, "not an array")
	}

	index, ok := indexObj.(*object.Integer)
	if !ok {
		return newErrorAtNode(left, "index is not an integer")
	}

	idx := index.Value

	if idx < 0 || idx >= (len(array.Elements)) {
		return newErrorAtNode(left, "index out of range: %d", idx)
	}

	stored := mutate(array.Elements[idx])
	if isError(stored) {
		return stored
	}

	array.Elements[idx] = stored

	return stored
}
