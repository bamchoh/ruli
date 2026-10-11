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

STRINGはv1では未サポート（後日検討）→ 途中実装あり:文字列リテラル・連結・比較・len() が動作する。IEC61131互換のSTRING型（固定幅・代入長制限）は将来導入

実数リテラル（3.5、0.75、2.0）は実数族（REAL / LREAL）として扱える。整数リテラルとの混在演算は明示キャスト必須（6.3）。指数表記は非対応

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
- エラーで異常終了した場合、スカラーの書き戻しは行われない

# 6. 式・演算

## 6.1 演算子優先順位

Go準拠

## 6.2 演算

現在実装：

- \* / %  (優先度 PRODUCT)
- += -= \*= /= %=（複合代入。左辺は一度だけ評価され、二項演算と同じ型規則に従う）
- ++ --
- == != < > <= >=（< > は数値のみ。文字列は == != のみ、SPEC 11）
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

将来構文の予約（使用するとパースエラー）：

- defer
- nil

## 8.2 特殊

- \_

# 9. コメント

```
# comment
```

- 行末まで
- ブロックコメントなし

# 10. nil

- 予約語として存在（使用するとパースエラー）
- v1では意味を持たない
- ref導入後に使用可能予定

# 11. 文字列

リテラルとして実装済み：

```
s := "hello"
t := s + " world"   # 連結
len(s)              # ルーン数
s == t              # == / !=
```

- 将来IEC61131互換STRING型（固定幅・代入長制限付き）として導入予定
- 文字列の `<` `>` 比較は非対応

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

# 13.3 スクリプト単位Env

- スクリプトごとに独立Env
- ファイル間共有なし

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
- 汎用ポインタ（ref はパラメータ限定。変数への保存や nil との比較は将来）
- ヒープ操作API
- 動的配列
- 文字列の < > <= >=（== != のみ対応）

# 16. 実装パイプライン（2026-10 現在）

```
lexer → parser → ast → checker → evaluator → object
```

- `checker.Check` が静的な型検査を行う（実行前）
- エラーは 行:列 形式で報告され、parse → checker の順で最初の段階で停止する
- パーサは panic モード復帰を持つ: 区切り（`,` `)` `:` `;` など）まで読み飛ばして解析を再開し、1 つの間違いがカスケードしないようにする

# 17. CLI

```
ruli <script>    # .ruli ファイルを実行
```

- `go run ./cmd/ruli script.ruli` または `go build ./cmd/ruli` で生成したバイナリ
- 解析エラー・型エラー・実行時エラーともに非ゼロ終了コード
