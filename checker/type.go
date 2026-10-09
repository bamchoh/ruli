package checker

// Type は型を表す
type Type struct {
	// 基本型名。INT / BOOL / REAL / STRING / ARRAY / FUNCTION / VOID / ANY
	Name string

	// 関数型のシグネチャ (Name == FUNCTION のときのみ有効)
	Params []Type
	Return *Type

	// 配列の要素型 (Name == ARRAY のときのみ有効)
	Elem *Type
}

func (t *Type) Equal(other *Type) bool {
	if t == nil || other == nil {
		return false
	}
	// ANY / UNKNOWN は検査不能 (未指定引数、要素型不明など) なのですべてと一致扱い
	if t.Name == ANY_TYPE.Name || other.Name == ANY_TYPE.Name ||
		t.Name == UNKNOWN_TYPE.Name || other.Name == UNKNOWN_TYPE.Name {
		return true
	}
	if t.Name != other.Name {
		return false
	}

	switch t.Name {
	case ARRAY_TYPE.Name:
		return t.Elem.Equal(other.Elem)
	case FUNCTION_TYPE.Name:
		if (t.Return == nil) != (other.Return == nil) {
			return false
		}
		if !t.Return.Equal(other.Return) {
			return false
		}
		if len(t.Params) != len(other.Params) {
			return false
		}
		for i := range t.Params {
			if t.Params[i].Name != other.Params[i].Name {
				return false
			}
		}
	}

	return true
}

var (
	INT_TYPE      = &Type{Name: "INT"}
	REAL_TYPE     = &Type{Name: "REAL"}
	BOOL_TYPE     = &Type{Name: "BOOL"}
	STRING_TYPE   = &Type{Name: "STRING"}
	ARRAY_TYPE    = &Type{Name: "ARRAY"}    // 要素型不明の一般配列
	FUNCTION_TYPE = &Type{Name: "FUNCTION"} // シグネチャは必ず上書きして使う
	UNKNOWN_TYPE  = &Type{Name: "UNKNOWN"}
	VOID_TYPE     = &Type{Name: "VOID"}
	// ANY は型チェックを省略するプレースホルダ (未指定の引数など)
	ANY_TYPE = &Type{Name: "ANY"}
)

// resolveBasic は基本型名を Type に解決する
func resolveBasic(name string) (*Type, bool) {
	switch name {
	case "INT":
		return INT_TYPE, true
	case "REAL":
		return REAL_TYPE, true
	case "BOOL":
		return BOOL_TYPE, true
	case "STRING":
		return STRING_TYPE, true
	}
	return nil, false
}

func isNumeric(t *Type) bool {
	return t.Name == "INT" || t.Name == "REAL"
}
