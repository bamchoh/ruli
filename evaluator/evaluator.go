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
}

type Environment struct {
	store map[string]object.Object
}

func NewEnvironment() *Environment {
	return &Environment{
		store: make(map[string]object.Object),
	}
}

func (e *Environment) Get(name string) (object.Object, bool) {
	v, ok := e.store[name]
	return v, ok
}

func (e *Environment) Set(name string, val object.Object) {
	e.store[name] = val
}

func Eval(node ast.Node, env *Environment) object.Object {

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

	}

	return &object.Null{}
}

func evalProgram(program *ast.Program, env *Environment) object.Object {
	var result object.Object = &object.Null{}

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
	}

	return result
}

func evalBinaryExpression(be *ast.BinaryExpression, env *Environment) object.Object {

	left := Eval(be.Left, env)
	right := Eval(be.Right, env)

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerBinaryExpression(be.Operator, left, right)
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

func evalBlockStatement(block *ast.BlockStatement, env *Environment) object.Object {
	var result object.Object = &object.Null{}

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)

		if result != nil {
			rt := result.Type()
			if rt == object.BREAK_OBJ || rt == object.CONTINUE_OBJ {
				return result
			}
		}
	}

	return result
}

func evalIfStatement(stmt *ast.IfStatement, env *Environment) object.Object {

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

func evalIncDecStatement(stmt *ast.IncDecStatement, env *Environment) object.Object {

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

func evalForStatement(stmt *ast.ForStatement, env *Environment) object.Object {

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

			case object.CONTINUE_OBJ:
				Eval(stmt.Post, env)
				continue
			}
		}

		Eval(stmt.Post, env)
	}

	return &object.Null{}
}

func evalCallExpression(node *ast.CallExpression, env *Environment) object.Object {

	function := Eval(node.Function, env)

	var args []object.Object
	for _, arg := range node.Arguments {
		args = append(args, Eval(arg, env))
	}

	if builtin, ok := function.(*object.Builtin); ok {
		return builtin.Fn(args...)
	}

	return &object.Null{}
}
