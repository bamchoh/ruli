package evaluator

import (
	"fmt"
	"ruli/ast"
	"ruli/object"
)

var builtins = map[string]*object.Builtin{
	"print": {
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return &object.Null{}
		},
	},

	"len": {
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 1 {
				return &object.Null{}
			}

			switch arg := args[0].(type) {

			case *object.Array:
				return &object.Integer{
					Value: (len(arg.Elements)),
				}

			case *object.String:
				return &object.Integer{
					Value: (len([]rune(arg.Value))),
				}
			}

			return &object.Null{}
		},
	},
}

func Eval(node ast.Node, env *object.Environment) object.Object {

	switch node := node.(type) {

	case *ast.Program:
		return evalProgram(node, env)

	case *ast.VarDeclStatement:
		val := Eval(node.Value, env)
		env.Set(node.Name, val)
		return val

	case *ast.AssignStatement:
		val := Eval(node.Value, env)
		env.Set(node.Name, val)
		return val

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Identifier:
		if b, ok := builtins[node.Value]; ok {
			return b
		}

		v, ok := env.Get(node.Value)
		if !ok {
			return &object.Null{}
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
		val := Eval(node.Value, env)
		return &object.ReturnValue{Value: val}

	case *ast.ArrayLiteral:
		var elements []object.Object

		for _, el := range node.Elements {
			elements = append(elements, Eval(el, env))
		}

		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		index := Eval(node.Index, env)

		return evalIndexExpression(left, index)

	}

	return &object.Null{}
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object = &object.Null{}

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
	}

	return result
}

func evalBinaryExpression(be *ast.BinaryExpression, env *object.Environment) object.Object {

	left := Eval(be.Left, env)
	right := Eval(be.Right, env)

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerBinaryExpression(be.Operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(be.Operator, left, right)
	}

	return &object.Null{}
}

func evalIntegerBinaryExpression(operator string, left, right object.Object) object.Object {

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
		return &object.Integer{Value: lv / rv}

	case ">":
		return &object.Boolean{Value: lv > rv}
	case "<":
		return &object.Boolean{Value: lv < rv}
	case "==":
		return &object.Boolean{Value: lv == rv}
	case "!=":
		return &object.Boolean{Value: lv != rv}
	}

	return &object.Null{}
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
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

	return &object.Null{}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return &object.Boolean{Value: true}
	}
	return &object.Boolean{Value: false}
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object = &object.Null{}

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)

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

	v, _ := env.Get(stmt.Name)
	iv := v.(*object.Integer).Value

	if stmt.Operator == "++" {
		iv++
	} else {
		iv--
	}

	obj := &object.Integer{Value: iv}
	env.Set(stmt.Name, obj)
	return obj
}

func evalForStatement(stmt *ast.ForStatement, env *object.Environment) object.Object {

	Eval(stmt.Init, env)

	for {
		cond := Eval(stmt.Condition, env)
		if !isTruthy(cond) {
			break
		}

		result := Eval(stmt.Body, env)

		if result != nil {
			switch result.Type() {
			case object.BREAK_OBJ:
				return &object.Null{}
			case object.RETURN_OBJ:
				return result

			case object.CONTINUE_OBJ:
				Eval(stmt.Post, env)
				continue
			}
		}

		Eval(stmt.Post, env)
	}

	return &object.Null{}
}

func evalCallExpression(node *ast.CallExpression, env *object.Environment) object.Object {

	function := Eval(node.Function, env)

	var args []object.Object
	for _, arg := range node.Arguments {
		args = append(args, Eval(arg, env))
	}

	switch fn := function.(type) {

	case *object.Builtin:
		return fn.Fn(args...)

	case *object.Function:
		return applyFunction(fn, args)
	}

	return &object.Null{}
}

func applyFunction(fn *object.Function, args []object.Object) object.Object {

	extendedEnv := object.NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		if i < len(args) {
			extendedEnv.Set(param.Name, args[i])
		}
	}

	result := Eval(fn.Body, extendedEnv)

	if rv, ok := result.(*object.ReturnValue); ok {
		return rv.Value
	}

	return result
}

func evalIndexExpression(left, index object.Object) object.Object {

	switch {
	case left.Type() == object.ARRAY_OBJ &&
		index.Type() == object.INTEGER_OBJ:

		return evalArrayIndexExpression(left, index)
	}

	return &object.Null{}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {

	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value

	max := len(arrayObject.Elements) - 1

	if idx < 0 || idx > max {
		return &object.Null{}
	}

	return arrayObject.Elements[idx]
}
