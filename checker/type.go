package checker

import "fmt"

// Type は型を表す
type Type struct {
	// 基本型名。INT / BOOL / REAL / STRING / ARRAY / FUNCTION / VOID / ANY
	Name string

	// 関数型のシグネチャ (Name == FUNCTION のときのみ有効)
	Params   []Type
	Refs     []bool  // Params[i] が ref パラメータか (SPEC 8.1)
	Returns  []*Type // 複数戻り値 (SPEC 5.3)。空なら戻り値なし
	Variadic bool    // ホスト関数 (embed API) の引数は個数・型不定。ホスト側が検証する

	// 配列の要素型 (Name == ARRAY のときのみ有効)
	Elem   *Type
	ArrLen int // 配列サイズ (SPEC 4.4: 型の一部)。0 はサイズ不明 (推論失敗など)

	// struct フィールド (Name == STRUCT のときのみ有効)
	FieldNames []string
	Fields     map[string]*Type

	// distinct 型 (SPEC 4.2) の基本型 (Name == 宣言した型名のときのみ有効)
	Base *Type
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
	// 同一ファミリ (整数族 / 実数族) 内の暗黙変換は許可する
	if isIntFamily(t.Name) && isIntFamily(other.Name) {
		return true
	}
	if isRealFamily(t.Name) && isRealFamily(other.Name) {
		return true
	}
	if t.Name != other.Name {
		return false
	}

	switch t.Name {
	case ARRAY_TYPE.Name:
		// サイズは型の一部 (SPEC 4.4)
		if t.ArrLen != other.ArrLen {
			return false
		}
		return t.Elem.Equal(other.Elem)
	case STRUCT_TYPE.Name:
		// 構造的に同型か (同名の異なる宣言は同一視しないが、同構造は許す)
		if len(t.FieldNames) != len(other.FieldNames) {
			return false
		}
		for _, fname := range t.FieldNames {
			oft, ok := other.Fields[fname]
			if !ok {
				return false
			}
			if !t.Fields[fname].Equal(oft) {
				return false
			}
		}
	case FUNCTION_TYPE.Name:
		if len(t.Returns) != len(other.Returns) {
			return false
		}
		for i := range t.Returns {
			if !t.Returns[i].Equal(other.Returns[i]) {
				return false
			}
		}
		if len(t.Params) != len(other.Params) {
			return false
		}
		for i := range t.Params {
			if t.Params[i].Name != other.Params[i].Name {
				return false
			}
		}
		if len(t.Returns) != len(other.Returns) {
			return false
		}
		for i := range t.Returns {
			if !t.Returns[i].Equal(other.Returns[i]) {
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
	// STRUCT_TYPE は struct 型。シグネチャ (フィールド) は必ず上書きして使う
	STRUCT_TYPE = &Type{Name: "STRUCT", Fields: map[string]*Type{}}
	MULTI_TYPE  = &Type{Name: "MULTI"} // 複数戻り値 (Returns を必ず設定)
)

// IEC61131 固定幅整数型 (SPEC 4.1) — 値モデルは INT と共有 (Go int)
var (
	SINT_TYPE  = &Type{Name: "SINT"}
	USINT_TYPE = &Type{Name: "USINT"}
	BYTE_TYPE  = &Type{Name: "BYTE"}
	UINT_TYPE  = &Type{Name: "UINT"}
	WORD_TYPE  = &Type{Name: "WORD"}
	DINT_TYPE  = &Type{Name: "DINT"}
	UDINT_TYPE = &Type{Name: "UDINT"}
	DWORD_TYPE = &Type{Name: "DWORD"}
	LINT_TYPE  = &Type{Name: "LINT"}
	ULINT_TYPE = &Type{Name: "ULINT"}
	LWORD_TYPE = &Type{Name: "LWORD"}
	LREAL_TYPE = &Type{Name: "LREAL"} // 値モデルは REAL と共有 (float64)
)

// resolveBasic は基本型名を Type に解決する
func resolveBasic(name string) (*Type, bool) {
	switch name {
	case "INT":
		return INT_TYPE, true
	case "SINT":
		return SINT_TYPE, true
	case "USINT":
		return USINT_TYPE, true
	case "BYTE":
		return BYTE_TYPE, true
	case "UINT":
		return UINT_TYPE, true
	case "WORD":
		return WORD_TYPE, true
	case "DINT":
		return DINT_TYPE, true
	case "UDINT":
		return UDINT_TYPE, true
	case "DWORD":
		return DWORD_TYPE, true
	case "LINT":
		return LINT_TYPE, true
	case "ULINT":
		return ULINT_TYPE, true
	case "LWORD":
		return LWORD_TYPE, true
	case "REAL":
		return REAL_TYPE, true
	case "LREAL":
		return LREAL_TYPE, true
	case "BOOL":
		return BOOL_TYPE, true
	case "STRING":
		return STRING_TYPE, true
	}
	return nil, false
}

// isIntFamily は IEC61131 整数族 (SPEC 4.1) をまとめて判定する。
// ファミリ内では暗黙変換を許可する
func isIntFamily(name string) bool {
	switch name {
	case "INT", "SINT", "USINT", "BYTE", "UINT", "WORD",
		"DINT", "UDINT", "DWORD", "LINT", "ULINT", "LWORD":
		return true
	}
	return false
}

func isRealFamily(name string) bool {
	return name == "REAL" || name == "LREAL"
}

// 数値型 (整数族 + 実数族)
func isNumeric(t *Type) bool {
	return isIntFamily(t.Name) || isRealFamily(t.Name)
}

// String は型を人間可読に整形する (配列は [10]INT 形式)
func (t *Type) String() string {
	switch t.Name {
	case ARRAY_TYPE.Name:
		if t.ArrLen > 0 {
			return fmt.Sprintf("[%d]%s", t.ArrLen, t.Elem.String())
		}
		if t.Elem != nil {
			return fmt.Sprintf("%s[]", t.Elem.String())
		}
	case FUNCTION_TYPE.Name:
		params := ""
		for i := range t.Params {
			if i > 0 {
				params += ", "
			}
			params += t.Params[i].String()
		}
		out := "FUNC(" + params + ")"
		for _, r := range t.Returns {
			out += " " + r.String()
		}
		return out
	}
	return t.Name
}
