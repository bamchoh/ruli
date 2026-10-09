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

// bindValue は SPEC 4.4 / 4.3 (値渡し) に従い、変数への束縛時に
// 配列・struct を複製する
func bindValue(val object.Object) object.Object {
	switch v := val.(type) {
	case *object.Array:
		return v.Copy()
	case *object.Struct:
		return v.Copy()
	}
	return val
}
