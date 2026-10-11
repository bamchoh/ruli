# ruli サンプル

実行 (`cmd/ruli` は embed API `ruli` の上に実装されている):

```
go run ./cmd/ruli examples/hello.ruli
go run ./cmd/ruli examples/methods.ruli
go run ./cmd/ruli examples/pid.ruli
```

またはビルド (`go build -o ruli.exe ./cmd/ruli`) して `./ruli.exe examples/hello.ruli`。
引数なしで起動すると REPL になります (`ruli> ` プロンプト、終了は `.exit` または Ctrl-D)。

- `hello.ruli` — 基本文法: 変数、演算、ループ、固定幅型、キャスト、配列
- `methods.ruli` — struct とメソッド (値渡し / ref レシーバ)
- `pid.ruli` — PLC 向け: struct を状態にした PI 制御ループのシミュレーション

`cmd/ruli` のテストはこのディレクトリの全スクリプトを常に実行して壊れないことを確認しています。
