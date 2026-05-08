package evaluator

import (
	"fmt"
	"ruli/object"
)

func newError(
	line int,
	column int,
	format string,
	a ...interface{},
) object.Object {

	return &object.Error{
		Message: fmt.Sprintf(format, a...),
		Line:    line,
		Column:  column,
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}
