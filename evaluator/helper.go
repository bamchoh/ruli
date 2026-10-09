package evaluator

import (
	"fmt"
	"ruli/ast"
	"ruli/object"
)

func newErrorAtNode(
	node ast.Node,
	format string,
	a ...interface{},
) object.Object {

	return &object.Error{
		Message: fmt.Sprintf(format, a...),
		Line:    node.GetToken().Line,
		Column:  node.GetToken().Column,
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

// bindValue は SPEC 4.4 (値渡し) に従い、変数への束縛時に配列を複製する
func bindValue(val object.Object) object.Object {
	if arr, ok := val.(*object.Array); ok {
		return arr.Copy()
	}
	return val
}
