package evaluator

import (
	"fmt"
	"math"
	"ruli/ast"
	"ruli/object"
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
			return val
		}

		env.Set(node.Name.Value, bindValue(val))
		return val

	case *ast.TypeDeclStatement:
		// 実行時の struct 型定義を登録する (ゼロ値構築に使用)
		def := &object.StructDef{
			FieldNames: make([]string, 0, len(node.Fields)),
			FieldTypes: make(map[string]string, len(node.Fields)),
		}
		for _, field := range node.Fields {
			def.FieldNames = append(def.FieldNames, field.Name)
			def.FieldTypes[field.Name] = field.Type.String()
		}
		env.DefineType(node.Name, def)
		return &object.Null{}

	case *ast.AssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		return evalAssignStatement(node, val, env)

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.Identifier:
		if b, ok := builtins[node.Value]; ok {
			return b
		}

		v, ok := env.Get(node.Value)
		if !ok {
			return newErrorAtNode(node, "identifier not found: %s", node.Value)
		}
		return v

	case *ast.BinaryExpression:
		return evalBinaryExpression(node, env)

	case *ast.IfStatement:
		return evalIfStatement(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.IncDecStatement:
		return evalIncDecStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.CallExpression:
		return evalCallExpression(node, env)

	case *ast.CastExpression:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		return evalCast(node, val)

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
			Body:       node.Body,
			Env:        env,
		}
		env.Set(node.Name, fn)
		return fn

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

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerBinaryExpression(be, left, right)
	case left.Type() == object.REAL_OBJ && right.Type() == object.REAL_OBJ:
		return evalRealBinaryExpression(be, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(be, left, right)
	}

	return newErrorAtNode(
		be,
		"type mismatch: %s %s %s",
		left.Type(),
		be.Operator,
		right.Type(),
	)
}

func evalIntegerBinaryExpression(
	be *ast.BinaryExpression,
	left, right object.Object,
) object.Object {

	lv := left.(*object.Integer).Value
	rv := right.(*object.Integer).Value

	switch be.Operator {
	case "+":
		return &object.Integer{Value: lv + rv}
	case "-":
		return &object.Integer{Value: lv - rv}
	case "*":
		return &object.Integer{Value: lv * rv}
	case "/":
		if rv == 0 {
			return newErrorAtNode(be, "division by zero")
		}
		return &object.Integer{Value: lv / rv}
	case "%":
		if rv == 0 {
			return newErrorAtNode(be, "division by zero")
		}
		return &object.Integer{Value: lv % rv}

	case ">":
		return nativeBoolToBooleanObject(lv > rv)
	case "<":
		return nativeBoolToBooleanObject(lv < rv)
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(be, "unsupported operator: %s", be.Operator)
}

func evalRealBinaryExpression(
	be *ast.BinaryExpression,
	left, right object.Object,
) object.Object {

	lv := left.(*object.Real).Value
	rv := right.(*object.Real).Value

	switch be.Operator {
	case "+":
		return &object.Real{Value: lv + rv}
	case "-":
		return &object.Real{Value: lv - rv}
	case "*":
		return &object.Real{Value: lv * rv}
	case "/":
		if rv == 0 {
			return newErrorAtNode(be, "division by zero")
		}
		return &object.Real{Value: lv / rv}
	case "%":
		if rv == 0 {
			return newErrorAtNode(be, "division by zero")
		}
		return &object.Real{Value: math.Mod(lv, rv)}

	case ">":
		return nativeBoolToBooleanObject(lv > rv)
	case "<":
		return nativeBoolToBooleanObject(lv < rv)
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(be, "unsupported operator: %s", be.Operator)
}

func evalStringInfixExpression(
	be *ast.BinaryExpression,
	left, right object.Object,
) object.Object {
	lv := left.(*object.String).Value
	rv := right.(*object.String).Value

	switch be.Operator {
	case "+":
		return &object.String{Value: lv + rv}
	case "==":
		return nativeBoolToBooleanObject(lv == rv)
	case "!=":
		return nativeBoolToBooleanObject(lv != rv)
	}

	return newErrorAtNode(be, "unsupported operator: %s", be.Operator)
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
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
		return applyFunction(fn, args)
	}

	return newErrorAtNode(node, "not a function: %s", function.Type())
}

// evalCast は REAL(a) のような明示的キャストを実行する (SPEC 6.3)。
// 静的な型チェックは checker が担当し、ここでは値変換のみを行う
func evalCast(node *ast.CastExpression, val object.Object) object.Object {

	target := node.Type

	switch target {
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

		if errObj := checkWidth(node, target, &object.Integer{Value: conv}); errObj != nil {
			return errObj
		}

		return &object.Integer{Value: conv}
	}

	return newErrorAtNode(node, "invalid cast: %s to %s", val.Type(), target)
}

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

// zeroValueForTypeNode は m : Motor のような初期値なし宣言のゼロ値を作る
func zeroValueForTypeNode(node ast.TypeNode, env *object.Environment) object.Object {
	basic, ok := node.(*ast.BasicType)
	if !ok {
		return &object.Null{}
	}
	return zeroValue(env, basic.Name)
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
		fields := make(map[string]object.Object, len(def.FieldTypes))
		for name, ft := range def.FieldTypes {
			fields[name] = zeroValue(env, ft)
		}
		return &object.Struct{Fields: fields}
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

func applyFunction(fn *object.Function, args []object.Object) object.Object {

	extendedEnv := object.NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		// SPEC 5.2: 引数は完全値渡し
		extendedEnv.Set(param.Name, bindValue(args[i]))
	}

	result := Eval(fn.Body, extendedEnv)
	if isError(result) {
		return result
	}

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

// evalSelectorAssign は m.speed = 10 のようなフィールドへの代入
func evalSelectorAssign(
	left *ast.SelectorExpression,
	val object.Object,
	env *object.Environment,
) object.Object {

	obj := Eval(left.Object, env)
	if isError(obj) {
		return obj
	}

	s, ok := obj.(*object.Struct)
	if !ok {
		return newErrorAtNode(left, "no field '%s' on %s", left.Field, obj.Type())
	}

	if _, ok := s.Fields[left.Field]; !ok {
		return newErrorAtNode(left, "undefined field: %s", left.Field)
	}

	// フィールドが固定幅型なら実行時幅チェックする
	if parentType := typeNameOfExpr(left.Object, env); parentType != "" {
		if def, ok := env.GetTypeDef(parentType); ok {
			if fieldType, ok := def.FieldTypes[left.Field]; ok {
				if errObj := checkWidth(left, fieldType, val); errObj != nil {
					return errObj
				}
			}
		}
	}

	s.Fields[left.Field] = bindValue(val)

	return val
}

// typeNameOfExpr は式が参照する変数の宣言時型名を返す (判定不能なら "")
// セレクタの親をたどって struct フィールドの型名も解決する
func typeNameOfExpr(expr ast.Expression, env *object.Environment) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		if t, ok := env.VarType(e.Value); ok {
			return t
		}
	case *ast.SelectorExpression:
		parent := typeNameOfExpr(e.Object, env)
		if parent == "" {
			return ""
		}
		if def, ok := env.GetTypeDef(parent); ok {
			return def.FieldTypes[e.Field]
		}
	}
	return ""
}

func evalIndexAssign(
	left *ast.IndexExpression,
	val object.Object,
	env *object.Environment,
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

	array.Elements[idx] = bindValue(val)

	return val
}
