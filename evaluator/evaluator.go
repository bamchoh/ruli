package evaluator

import (
	"ruli/ast"
)

type Environment struct {
	store map[string]interface{}
}

func NewEnvironment() *Environment {
	return &Environment{
		store: make(map[string]interface{}),
	}
}

func (e *Environment) Get(name string) (interface{}, bool) {
	v, ok := e.store[name]
	return v, ok
}

func (e *Environment) Set(name string, val interface{}) {
	e.store[name] = val
}

func Eval(node ast.Node, env *Environment) interface{} {

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
		return node.Value

	case *ast.Identifier:
		v, _ := env.Get(node.Value)
		return v

	case *ast.BinaryExpression:
		return evalBinaryExpression(node, env)
	}

	return nil
}

func evalProgram(program *ast.Program, env *Environment) interface{} {
	var result interface{}

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
	}

	return result
}

func evalBinaryExpression(be *ast.BinaryExpression, env *Environment) interface{} {

	left := Eval(be.Left, env).(int)
	right := Eval(be.Right, env).(int)

	switch be.Operator {
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/":
		return left / right
	}

	return nil
}
