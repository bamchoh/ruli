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
