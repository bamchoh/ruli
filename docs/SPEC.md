# 1. 概要

ruli（瑠璃）は Go に埋め込み可能な AST インタプリタ型スクリプト言語である。

設計思想：

- 静的型付け
- 明示的で安全な構文
- PLC / IEC61131 的な制御用途も視野
- Goホスト上で動作（GC依存でメモリ管理は行わない）
- まずASTインタプリタ（Phase1）、後にBytecode VMへ移行（Phase2）

# 2. 実行モデル

## 2.1 実行方式

- Go製ASTインタプリタ
- スクリプト単位で実行
- 各スクリプトは独立した環境（Env）を持つ

## 2.2 実行フェーズ

_Phase1: 宣言収集_

- type / func を先に登録
- シンボル解決可能にする

_Phase2: 実行_

- トップレベル文を上から実行

## 2.3 将来拡張

- Phase2で Bytecode VMへ移行可能（AST互換設計）

# 3. スコープモデル

## 3.1 スコープ階層

- Scriptスコープ（ファイル単位）
- Blockスコープ（{}）
- Functionスコープ

## 3.2 ルール

- トップレベルスコープはファイル内に閉じる
- ブロックスコープを持つ
- 関数スコープは独立
- 外側スコープ参照は可能（read/write）

## 3.3 変数探索ルール

代入時（=）：

- 現在スコープから検索
- 見つからなければ親へ
- Scriptトップまで探索
- 見つかった最初の変数を更新
- 存在しなければエラー

## 3.4 宣言ルール

- := / type 宣言は現在スコープに作成
- 再宣言はシャドーイング可能

## 3.5 シャドーイング

許可する：

```
x := 10

if true {
  x := 20
}
```

# 4. 型システム

## 4.1 基本型（v1）

| 型    | 符号     | サイズ | 備考           |
| ----- | -------- | ------ | -------------- |
| BOOL  | -        | 1 bit  | true/false     |
| SINT  | Signed   | 8 bit  |                |
| USINT | Unsigned | 8 bit  |                |
| BYTE  | Unsigned | 8 bit  | Alias of USINT |
| INT   | Signed   | 16 bit |                |
| UINT  | Unsigned | 16 bit |                |
| WORD  | Unsigned | 16 bit | Alias of UINT  |
| DINT  | Signed   | 32 bit |                |
| UDINT | Unsigned | 32 bit |                |
| DWORD | Unsigned | 32 bit | Alias of UDINT |
| LINT  | Signed   | 64 bit |                |
| ULINT | Unsigned | 64 bit |                |
| LWORD | Unsigned | 64 bit | Alias of ULINT |
| REAL  | -        | 32 bit | float          |
| LREAL | -        | 64 bit | double         |

STRING は文字列族として実装済み（リテラル・連結・比較・len・インデックスアクセス）。IEC61131 風の固定上限 `STRING(n)` も利用可（詳細は 11）。WSTRING と IEC の文字列演算関数（LEFT/MID など）は将来導入

実数リテラル（3.5、0.75、2.0、指数表記 1e5 / 2.5e-3）は実数族（REAL / LREAL）として扱える。整数リテラルとの混在演算は明示キャスト必須（6.3）

## 4.1.1 変数代入

```
a = 10
```

- 未定義の変数への代入はエラー

## 4.1.2 変数束縛

```
a := 10
```

## 4.1.3 変数宣言

```
a : INT
b: DINT
```

## 4.1.4 変数宣言 + 初期化

```
a : INT = 10
```

## 4.2 distinct type

```
type TEMP INT
```

- 新しい型として扱う（nominal: TEMP と INT は別型）
- 基本型との暗黙変換なし（`t : TEMP = 30` はエラー）
- 明示キャストで変換（双方向）: `TEMP(30)` / `INT(t)`
- ゼロ値は基本型から構築される

## 4.3 struct

```
type Motor struct {
  speed:  INT
  enable: BOOL
}
```

- フィールドは公開
- 値渡し

構造体リテラル:

```
m : Motor = Motor{speed: 10, enable: true}
m2 := Motor{speed: 5}      # 未指定フィールドはゼロ値
m3 := Motor{}              # 全フィールドゼロ値
```

- 指定するのは `フィールド名: 値`。区切りは `,` または `;`
- 重複フィールド・未知フィールドは静的エラー
- if / for の条件部・switch の値では `if flag { }` との曖昧性を避けるため struct リテラルには括弧が必要（`(Motor{...})`。Go と同じ規則）
- フィールドアクセスは postfix (`m.speed`、リテラルや戻り値にも有効: `Motor{speed: 1}.speed`)
- メソッドは 5.6

## 4.4 配列

固定長配列のみサポート

```
values: [10]INT
matrix: [3][3]INT
```

_特徴_

- 型の一部としてサイズを持つ
- 多次元配列対応
- 値渡し（コピー）

## 4.5 配列初期化

```
a := [1, 2, 3]      # リテラル（現在実装）
values : [10]INT    # 固定長宣言（サイズは型の一部）
```

- リテラルの長さが配列型のサイズになる（`[1, 2, 3]` は `[3]INT` 相当）
- `[3]INT{1, 2, 3}` 形式の型付きリテラルは将来対応

## 4.6 添字アクセス

```
a[0] = 10
x := a[1]
```

## 4.7 境界チェック

- 既知サイズへのリテラルインデックスは静的チェックあり（ 型チェッカー)
- 実行時チェックあり
- 範囲外アクセス → error

## 4.8 len()

組み込み関数：

```
len(array)
```

- 配列の要素数を返す（文字列はルーン数）
- サイズ既知の配列は型から定数化可能

# 5. 関数

## 5.1 定義

```
func add(a: INT, b: INT) INT {
  return a + b
}
```

## 5.2 引数

- 完全値渡し（ref パラメータのみ参照、SPEC 5.5）
- パラメータは必ず型付き（`func f(a)` はエラー）

## 5.3 戻り値

- 複数戻り値対応

```
func div(a: INT, b: INT) (INT, BOOL)
```

## 5.4 呼び出しルール

- 戻り値は無視可能

```
div(10, 2)
```

- 複数戻り値を単一変数へ代入は禁止

```
x := div(10, 2) // error
```

正しくは

```
x, ok := div(10, 2)
```

## 5.5 ref（IEC61131 の VAR_IN_OUT 相当）

```
type Motor struct { speed: INT }

func reset(m: ref Motor) BOOL {
	m.speed = 0
	return true
}

m : Motor
reset(m)       # 呼び出し側は修飾なしで変数を渡す
print(m.speed) # 0（関数内の変更が反映される）
```

- ref はパラメータ型の一部（Go の `*Motor` に相当）。パラメータ以外（変数宣言、struct フィールド、戻り値型）ではエラー
- パラメータは必ず型付き（`func f(a)` はエラー、ref も型を要求する）
- 配列 / struct はオブジェクト共有：関数内の変更が即時反映される
- スカラー（INT / REAL / BOOL / STRING）は呼び出し元変数へ書き戻す
- ref 引数に渡せるのは書き込み可能な変数のみ（識別子 / フィールド / 添字）。リテラルや式はエラー
- ref パラメータ自体への再代入（`n = ...`）は呼び出し元に反映されない（フィールド / 要素経由の変更のみ反映）

## 5.6 メソッド

struct 型にレシーバ付き関数を定義する。呼び出しは postfix selector から行う:

```
type Motor struct {
	speed: INT
}

func (m: Motor) isStopped() BOOL {
	return m.speed == 0
}

func (m: ref Motor) stop() BOOL {
	m.speed = 0
	return true
}

m : Motor
m.speed = 5
m.stop()           # ref レシーバの変更は呼び出し側に反映される (VAR_IN_OUT と同じ仕組み)
print(m.speed)     # 0
print(m.isStopped())   # true（値渡しレシーバ）
```

- メソッドはその struct 型の名前空間に属する（`Motor.stop`）。トップレベルの名前を汚さない
- レシーバは 1 つで struct 型のみ（パラメータと同じ構文で `: 型` は必須）
- レシーバのデフォルトは値渡し（フィールド変更は外に反映されない）。`ref` レシーバの変更は呼び出し側に反映される
- フィールド名と同名のメソッドは定義できない（静的エラー）
- メソッドはファーストクラスではない: `m.reset` を値として取り出すことはできない（呼び出し形のみ）
- 継承はなく、静的ディスパッチで解決される
- エラーで異常終了した場合、スカラーの書き戻しは行われない

## 5.7 ファーストクラス関数

関数型 `FUNC(...)` により、関数を値として引数・戻り値・変数に渡せる:

```
func apply(fn: FUNC(x: INT) INT, v: INT) INT {
	return fn(v)
}

func double(x: INT) INT {
	return x * 2
}

print(apply(double, 21))   # 42

g := double                # 変数への格納
print(g(10))               # 20
```

- 関数型の構文: `FUNC(名前: 型, ...) 戻り値型`。パラメータは `名前: 型` が必須。ネスト可 (`FUNC(cb: FUNC(x: INT) INT)`)
- 戻り値型は省略可能（`FUNC()` は VOID）。`(INT, BOOL)` の複数戻り値も記述できる
- 関数値のシグネチャは静的に一致必須（引数・戻り値、ref 修飾を含む）
- 組み込み関数（print / len / ホスト関数）も関数型に渡せる
- メソッドはファーストクラスではない（5.6）
- v1 ではクロージャ非対応（12.2）: 関数値は定義時の環境を参照する

# 6. 式・演算

## 6.1 演算子優先順位

Go準拠

## 6.2 演算

現在実装：

- \* / %  (優先度 PRODUCT)
- += -= \*= /= %=（複合代入。左辺は一度だけ評価され、二項演算と同じ型規則に従う）
- ++ --
- == != < > <= >=（数値は数値族同士、文字列は辞書順で比較）
- && ||（short-circuit 評価。オペランドは BOOL のみで、優先度は || < && < 比較）
- 単項 - !
- 単項 - はオペランドを PRODUCT 優先度で読む（`-a * b` は `(-a) * b`）

## 6.3 型制約

- INT + REAL → エラー
- 明示キャスト必須

```
c := REAL(a) + b
```

# 7. 制御構文

## 7.1 if

```
if x > 0 {
}
```

## 7.2 for（while廃止）

```
for i := 0; i < 10; i++ {
}
```

## 7.3 break / continue

予約語として使用可能

## 7.4 switch

```
switch x {
case 1, 2:
	print("one or two")
default:
	print("other")
}
```

- fallthrough なし（一致した case のみ実行）
- `break` は switch を抜ける（最内側。ループ継続には影響しない）
- `return` は関数へ伝播する
- case 値の型は switch の値と一致必須（静的検査）
- case 本体はスコープを持つ

# 8. 予約語

## 8.1 現在予約

実行可能構文：

- if
- else
- for
- func
- return
- type
- struct
- true
- false
- break
- continue
- switch
- case
- default
- ref
- nil

将来構文の予約（使用するとパースエラー）：

- defer

## 8.2 特殊

- \_

# 9. コメント

```
# comment
```

- 行末まで
- ブロックコメントなし

# 10. nil

`nil` は「値がない」ことを表すリテラル (Null 値に評価される):

```
v := findTag("S1")    # ホスト関数が nil を返した場合
if v != nil {
	print(v)
}
```

- 比較可能なのは `== nil` / `nil == x` の等値検査 (任意のスカラー型と VOID)。TRUE/FALSE は実行時の値で決まる
- `nil` は変数へ束縛可 (`x := nil`)。NULL 型となるが、数値/文字列演算や if/for/switch の条件には使えない (型エラー)
- `x : INT = nil` のような型付き宣言への nil 代入はエラー (NULL は他の型と一致しない)
- struct / 配列 / 関数値 (ファーストクラス) との nil 比較は不可 (値渡し言語であり、値は nil にならない)
- 一般ポインタ (SPEC 15) には関連しない

# 11. 文字列

リテラルとして実装済み：

```
s := "hello"
t := s + " world"   # 連結
len(s)              # ルーン数
s == t              # == / !=
```

- `s[0]` のようなインデックスアクセスは 1 文字（ルーン基準）を STRING として返す。代入は不可（不変値）
- IEC61131 風の固定上限 `STRING(n)`（n は正の整数リテラル）:
    name : STRING(20)          # 最大 20 文字（ルーン基準）
    s : STRING(3) = "abc"     # 宣言 + 初期化
  - `STRING`（括弧なし）は無制限。IEC の既定長 80 は採用しない
  - `STRING(n)` 同士・`STRING(n)` と `STRING` は同一族として互換（n は型の同一性ではなく代入上限制約）
  - 上限を超える代入はエラー（IEC は切り詰めを許すが、ruli は数値の幅チェックと同じく決定論的にエラーとする。明示的な切り詰めは将来の LEFT/MID 相当で行う）
  - 静的検査：リテラル初期化の長さ超過は実行前に検出される

# 12. 関数スコープ仕様

## 12.1 外側スコープ参照

許可（read/write）

```
x := 10

func f() {
  x++
}
```

## 12.2 クロージャ

v1では未対応（将来検討）

# 13. Go埋め込みランタイム設計

## 13.1 値モデル

Go GC上で管理：

- Integer（int）
- Real（float64）
- Boolean
- Array（値渡し: 束縛時に深いコピー）
- Struct（値渡し: 同上）
- MultiValue（複数戻り値の一時値）
- Function / Builtin / Error

値モデルは Go int / float64 を共有し、固定幅型の幅は checker の型と実行時値域検査（object.WidthOf）で保証する。

# 13.2 Envモデル

```
type Environment struct {
  store map[string]Object
  outer *Environment
}
```

- 宣言（:= / type / 引数束縛）は Set（現在スコープ）
- 更新（= / ++ / --）は Assign（親をたどって最初の変数を更新）

# 13.3 embed API（実装済み）

ルートパッケージ `ruli` がホスト向けの公開 API を持つ:

```go
s := ruli.New()
s.SetOutput(buf)                       // print の出力先
s.SetFunction("hello", "STRING", func(args ...ruli.Value) (ruli.Value, error) {
	return ruli.NewStringValue("hello " + args[0].Inspect()), nil
})                                     // ホスト関数登録
err := s.RunString(`x := hello(42)`)
err = s.RunFile("script.ruli")
```

- 実行環境と checker のシンボル環境は呼び出し間で共有（段階的に駆動できる）
- `SetFunction`: 引数の個数・型はホスト側が検証（ruli からは可変長）。戻り値型は静的検査に使われ、不正な型名は登録時にエラー
- スクリプト内で同名の関数を定義するとホスト関数より優先される
- 値コンストラクタ: `NewIntValue` / `NewRealValue` / `NewBoolValue` / `NewStringValue` / `NewArrayValue` / `NewStructValue` / `NewNullValue`
- エラーは `行:列` 形式（parse / 型検査 / 実行時）。複数エラーは改行連結
- CLI `cmd/ruli` もこの API 上に実装されている

# 14. 実装フェーズ

_Phase1（現在）_

- ASTインタプリタ
- Go実装
- スクリプト実行可能化

_Phase2（将来）_

- AST → Bytecode変換
- VM実行
- 高速化

# 15. 非対応（v1）

- クロージャ
- GC設計（Go任せ）
- 汎用ポインタ（ref はパラメータ限定。変数への保存は将来）
- ヒープ操作API
- 動的配列
- IEC61131互換の固定幅 STRING 型（宣言構文と代入長制限）

# 16. 実装パイプライン（2026-10 現在）

```
lexer → parser → ast → checker → evaluator → object
```

- `checker.Check` が静的な型検査を行う（実行前）
- エラーは 行:列 形式で報告され、parse → checker の順で最初の段階で停止する
- パーサは panic モード復帰を持つ: 区切り（`,` `)` `:` `;` など）まで読み飛ばして解析を再開し、1 つの間違いがカスケードしないようにする

# 17. CLI

```
ruli            # REPL (式の結果は "= 値" で表示、複数行継続あり)
ruli <script>   # .ruli ファイルを実行
```

- `go run ./cmd/ruli script.ruli` または `go build ./cmd/ruli` で生成したバイナリ
- REPL: 未完の構文 (`if x {` など) は `    > ` プロンプトで継続。`.help` / `.exit`
- サンプルスクリプトは `examples/` (基本文法 / メソッド / PI 制御ループ)
- 解析エラー・型エラー・実行時エラーともに非ゼロ終了コード
