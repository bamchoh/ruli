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

// checkWidth は SPEC 4.1 の固定幅型の値域を検査する (整数族のみ)。
// 範囲外ならエラーオブジェクト、問題なければ nil を返す
func checkWidth(node ast.Node, typeName string, val object.Object) object.Object {
	w, ok := object.WidthOf(typeName)
	if !ok || typeName == "" {
		return nil
	}

	iv, ok := val.(*object.Integer)
	if !ok {
		return nil // REAL 族や struct 幅は未検査
	}

	if int64(iv.Value) < w.Min || int64(iv.Value) > w.Max {
		return newErrorAtNode(
			node,
			"value %d out of range for %s (%d..%d)",
			iv.Value,
			typeName,
			w.Min,
			w.Max,
		)
	}

	return nil
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
