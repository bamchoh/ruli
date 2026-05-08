package evaluator

import (
	"fmt"
	"ruli/object"
)

func newError(format string, a ...interface{}) object.Object {
	return &object.Error{
		Message: fmt.Sprintf(format, a...),
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}
