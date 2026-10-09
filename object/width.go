package object

// Width は固定幅型 (SPEC 4.1) の値域
type Width struct {
	Min int64
	Max int64
}

// WidthOf は型名の値域を返す。幅を持たない型 (BOOL / REAL 族 / struct / 配列など) は ok=false
//
// ULINT / LWORD の符号なし上限 (18446744073709551615) は実行値モデル
// (Go int = 符号付き 64bit) を超えるため、ここでは INT64 上限に制限して検査する
func WidthOf(typeName string) (Width, bool) {
	switch typeName {
	case "SINT":
		return Width{Min: -128, Max: 127}, true
	case "USINT", "BYTE":
		return Width{Min: 0, Max: 255}, true
	case "INT":
		return Width{Min: -32768, Max: 32767}, true
	case "UINT", "WORD":
		return Width{Min: 0, Max: 65535}, true
	case "DINT":
		return Width{Min: -2147483648, Max: 2147483647}, true
	case "UDINT", "DWORD":
		return Width{Min: 0, Max: 4294967295}, true
	case "LINT":
		return Width{Min: -9223372036854775808, Max: 9223372036854775807}, true
	case "ULINT", "LWORD":
		return Width{Min: 0, Max: 9223372036854775807}, true
	}
	return Width{}, false
}
