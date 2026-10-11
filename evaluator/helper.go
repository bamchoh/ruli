package evaluator

import (
	"fmt"
	"ruli/ast"
	"ruli/object"
	"strconv"
	"strings"
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

// declaredTypeInfo は宣言時型ノードから (型名, 文字列最大長) を取り出す。
// StringType の型名は "STRING(n)" 形式になる
func declaredTypeInfo(node ast.TypeNode) (string, int) {
	switch t := node.(type) {
	case *ast.BasicType:
		return t.Name, 0
	case *ast.StringType:
		return t.String(), t.Max
	}
	return "", 0
}

// stringMaxOfTypeName は宣言時型名 ("STRING(n)") から最大長を取り出す
// (括弧なしの STRING やその他の型は 0 = 無制限)
func stringMaxOfTypeName(typeName string) int {
	if !strings.HasPrefix(typeName, "STRING(") || !strings.HasSuffix(typeName, ")") {
		return 0
	}

	n, err := strconv.Atoi(typeName[len("STRING(") : len(typeName)-1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// checkStringWidth は SPEC 11 の STRING(n) の最大長を検査する。
// 超過はエラー (IEC の切り詰めではなく、数値幅と同じ決定論的な挙動)
func checkStringWidth(node ast.Node, max int, val object.Object) object.Object {
	sv, ok := val.(*object.String)
	if !ok {
		return nil
	}

	if n := len([]rune(sv.Value)); n > max {
		return newErrorAtNode(
			node,
			"string length %d exceeds STRING(%d)",
			n,
			max,
		)
	}

	return nil
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
