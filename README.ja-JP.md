# ruli

<p align="center">
  <img alt="ロゴ" src="logo.png" width="50%" height="50%">
</p>

English | [日本語](README.ja-JP.md)

**ruli(瑠璃)** は Go に埋め込み可能な、静的型付けの AST インタプリタ型スクリプト言語です。

- **静的型付け**:実行前に型不一致・未定義参照・配列境界・固定幅値域を検査
- **IEC 61131-3 風の型**:`SINT`/`DINT`/`WORD` などの固定幅型と `STRING(n)` を提供
- **値渡し基調**:`=` / 引数は常に値のコピー。書き込みたいときだけ明示的に `ref`(VAR_IN_OUT 相当)
- **Go 埋め込み**:ホスト関数の登録・変数の直接読み書き・出力先の差し替えが可能
- エラーはすべて `行:列` 形式で報告され、1 つの間違いは 1 つのエラーに収束します

現時点の言語仕様の詳細は [`docs/SPEC.md`](docs/SPEC.md) を参照してください。

## クイックスタート

```
# サンプルスクリプトを実行
go run ./cmd/ruli examples/hello.ruli

# REPL(引数なしで起動)
go run ./cmd/ruli

# ビルド
go build -o ruli.exe ./cmd/ruli
./ruli.exe examples/methods.ruli
```

examples: [`hello.ruli`](examples/hello.ruli) 基本文法 / [`methods.ruli`](examples/methods.ruli) struct + メソッド / [`pid.ruli`](examples/pid.ruli) PI 制御ループ。

## 言語ツアー

### 変数・スコープ

```
x := 10            # 束縛(現在スコープに宣言)
y : INT            # 型付き宣言(ゼロ値で初期化)
age : WORD = 5     # 宣言 + 初期化

x = 20             # = は既存変数の更新のみ(未定義への代入はエラー)

if true {
	x := 99         # ブロックスコープ + シャドーイング
}
print(x)           # 20(ブロック内の := は外を汚さない)
```

### 固定幅型と値域検査

IEC 61131-3 由来の型。宣言・代入・`++`/`--`・キャスト時に値域が検査され、あふれは静かなラップではなく**エラー**です。

| 型                   | 符号     | 値域                     | 備考               |
| -------------------- | -------- | ------------------------ | ------------------ |
| BOOL                 | -        | true/false               |                    |
| SINT / USINT / BYTE  | ± / なし | −128..127 / 0..255       | 8bit               |
| INT / UINT / WORD    | ± / なし | −32768..32767 / 0..65535 | 16bit              |
| DINT / UDINT / DWORD | ± / なし | 32bit                    |                    |
| LINT / ULINT / LWORD | ± / なし | 64bit                    |                    |
| REAL / LREAL         | -        | float / double           |                    |
| STRING / STRING(n)   | -        | 文字列                   | n は最大長(ルーン) |

```
temp : SINT = -40        # OK
temp = 200               # value 200 out of range for SINT (-128..127)
c := REAL(x) + y         # INT + REAL はエラー。明示キャスト必須
DINT(REAL(7) / REAL(2))  # 実数 → 整数は 0 方向へ切り捨て
```

### 文字列

```
name := "瑠璃"
greeting := "こんにちは" + name
len(greeting)      # ルーン数
greeting[0]        # "こ"(1 文字を STRING で返す)
"abc" < "abd"      # 辞書順比較 (== != < <= > >=)

title : STRING(20) = "..."   # 最大 20 文字。超過代入はエラー
```

### nil

`nil` は「値がない」ことを表すリテラルです。値渡しの言語なので、nil にならないのは struct / 配列 / 関数値で、主に**ホスト関数の戻り値が値なし**かどうかの検査に使います。

```
v := findTag("S1")    # ホスト関数が nil を返した場合
if v != nil {
	print(v)
}
```

- 比較は `== nil` / `!= nil` のみ(他の演算や if/for/switch の条件には使えない)
- `x := nil` で NULL 型変数になる。数値/文字列演算は型エラー
- `n : INT = nil` のような型付き宣言への nil は型不一致エラー

### 型システム

```
# struct(SPEC 4.3)+ 構造体リテラル(未指定フィールドはゼロ値)
type Motor struct {
	speed:  INT
	enable: BOOL
}

m := Motor{speed: 10}
m.speed = 20                # フィールド代入
m.stop()                    # メソッド
n := m                      # 値渡し(深いコピー)

# 固定長配列(サイズは型の一部)
values : [10]INT            # ゼロ埋めで宣言
matrix : [3][3]INT
values[0] = 5               # 実行時 + 静的な境界チェック
buf : [8]WORD = io.inputs   # サイズ違いは静的エラー

# distinct 型(名義型:基本型との暗黙変換なし)
type TEMP INT
t : TEMP = TEMP(30)         # TEMP(30) で構築 / INT(t) で戻す
```

### メソッド

```
func (m: Motor) isStopped() BOOL {       # 値渡しレシーバ(変更は反映されない)
	return m.speed == 0
}

func (m: ref Motor) stop() BOOL { # ref レシーバ(変更が呼び出し側に反映)
	m.speed = 0
	return true
}

m.stop()                                 # postfix セレクタから呼ぶ
```

### 関数

```
func add(a: INT, b: INT) INT {
	return a + b
}

# 複数戻り値(SPEC 5.3)。単一変数への多値代入は静的エラー
func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}

q, ok := div(10, 2)

# ref パラメータ(IEC VAR_IN_OUT 相当)
func reset(m: ref Motor) BOOL {
	m.speed = 0
	return true
}
reset(m)         # 呼び出し側は修飾なし。関数内の変更が反映される
```

### ファーストクラス関数

```
func apply(fn: FUNC(x: INT) INT, v: INT) INT {
	return fn(v)
}

func double(x: INT) INT { return x * 2 }

apply(double, 21)     # 42
g := double           # 変数へ格納
g(10)                 # 20
pick(true)(4)         # 関数を返す関数
```

### 制御構文

```
if temp > 30.0 && !alarm {
	print("warm")
} else if temp > 20.0 {
	print("normal")
}

for i := 0; i < 10; i++ {
	if i == 3 {
		break
	}
}

switch state {
case 1, 2:
	print("ready")
default:
	print("stopped")
}
```

## REPL

```
$ ruli
ruli> 1 + 2
= 3
ruli> func greet(name: STRING) STRING {
    >         return "hi " + name
    > }
ruli> greet("ruli")
= hi ruli
ruli> .exit
```

- 式の結果は `= 値` で表示(宣言・代入・関数定義は表示しない)
- 未完の構文(`if x {` など)は `   >` プロンプトで継続
- `.help` / `.exit`、Ctrl-D で終了

## Go からの埋め込み(embed API)

```go
// control.ruli
//   count := 0
//   for i := 0; i < 3; i++ {
//       count = count + readTag("S1")
//   }
//   if count > 100 {
//       print("alarm!")
//   }
//   alarmCount := count

package main

import (
	"fmt"
	"os"

	"ruli"
)

func main() {
	s := ruli.New()
	s.SetOutput(os.Stdout) // print の出力先(未指定なら標準出力)

	// ホスト関数を登録。ruli からは可変長関数に見え、
	// 戻り値型宣言は静的型検査に使われる
	s.SetFunction("readTag", "INT", func(args ...ruli.Value) (ruli.Value, error) {
		// ここで実際のタグ読み出しを行う
		return ruli.NewIntValue(42), nil
	})

	// 実行環境と型検査のシンボルは呼び出し間で共有される
	if err := s.RunFile("control.ruli"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// ホストから変数を直接読む
	v, _ := s.Env().Get("alarmCount")
	fmt.Println("alarms:", v.Inspect())
}
```

他のホスト API: `RunString` / `EvalString`(値を返す) / `SetOutput` / `Env()`。値コンストラクタは `NewIntValue` / `NewRealValue` / `NewBoolValue` / `NewStringValue` / `NewArrayValue` / `NewStructValue` / `NewNullValue`。

## エラー表示

```
$ cat bad.ruli
temp : SINT = 200

$ ruli bad.ruli
1:7: value 200 out of range for SINT (-128..127)
```

パース → 型検査 → 実行の順で検査し、エラーの段階で停止します。複数のパース/型エラーは行ごとに列挙されます。

## アーキテクチャ

```
lexer → parser → ast → checker → evaluator
                                  ↘ object (値・環境・実行時型定義)

ruli (ルートパッケージ: embed API)
cmd/ruli (CLI / REPL)
```

- **checker**:静的な型検査(型不一致・BOOL 条件・境界・値域・シグネチャ)。実行前にすべてのエラーを収集
- **evaluator**:AST ツリーウォーク。宣言(declarations)と更新(assignment)で環境の振る舞いを分離
- **object**:値の表現(Go int/float64 を値モデルとして共有)+ 値渡しのための深いコピー

## 開発

```
go build ./...    # ビルド
go test ./...     # 全テスト(evaluator/parser/lexer/checker/ruli/cmd)
go vet ./...      # 静的解析
```

- 言語仕様: [`docs/SPEC.md`](docs/SPEC.md)
- テスト: 単体テスト + `examples/` の回帰テスト
- フェーズ 2 (バイトコード VM 化) は将来計画として SPEC 2.3 に記載しています。

## ライセンス

MIT — [LICENSE](LICENSE)