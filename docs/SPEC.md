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

## 4.2 distinct type（将来実装）

```
type TEMP INT
```

- 新しい型として扱う
- 基本型との暗黙変換なし
- 現在は type 文は struct のみ対応（4.3）。基本型の別名化は将来実装

## 4.3 struct

```
type Motor struct {
  speed:  INT
  enable: BOOL
}
```

- フィールドは公開
- 値渡し

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

- 完全値渡し

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

# 6. 式・演算

## 6.1 演算子優先順位

Go準拠

## 6.2 演算

現在実装：

- \* / %  (優先度 PRODUCT)
- ++ --
- == != < >
- 単項 - !
- 単項 - はオペランドを PRODUCT 優先度で読む（`-a * b` は `(-a) * b`）

実装済みだが演算器未対応：

- && || ! の短絡評価（&& と || は将来実装）

将来実装：

- += -= \*= /= %=
- <= >=

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

## 7.4 switch（予約済み）

将来実装

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

将来構文の予約（使用するとパースエラー）：

- switch
- case
- default
- defer
- ref
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
- ポインタ（ref導入は将来）
- ヒープ操作API
- 動的配列
- 複合代入（+= -=\* =/= %=）および && ||（short-circuit）
- switch 文
- distinct type（基本型の別名化）
- 構造体リテラル（Motor{...} 形式）— `m : Motor` のゼロ値 + フィールド代入で代用

# 16. 実装パイプライン（2026-10 現在）

```
lexer → parser → ast → checker → evaluator → object
```

- `checker.Check` が静的な型検査を行う（実行前）
- エラーは 行:列 形式で報告され、parse → checker の順で最初の段階で停止する

# 17. CLI

```
ruli <script>    # .ruli ファイルを実行
```

- `go run ./cmd/ruli script.ruli` または `go build ./cmd/ruli` で生成したバイナリ
- 解析エラー・型エラー・実行時エラーともに非ゼロ終了コード
