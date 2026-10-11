// Package ruli は ruli スクリプト言語を Go ホストに埋め込むための API です (SPEC 13)。
//
// 使い方:
//
//	buf := &bytes.Buffer{}
//	s := ruli.New()
//	s.SetOutput(buf)
//	s.SetFunction("hello", "STRING", func(args ...ruli.Value) (ruli.Value, error) {
//		return ruli.NewStringValue("hello " + args[0].Inspect()), nil
//	})
//	err := s.RunString(`x := hello(42)
//	print(x)`)
package ruli

import (
	"errors"
	"io"
	"os"

	"ruli/ast"
	"ruli/checker"
	"ruli/evaluator"
	"ruli/lexer"
	"ruli/object"
	"ruli/parser"
)

// Value はスクリプト内の値 (object.Object のエイリアス)
type Value = object.Object

// Script は 1 本のスクリプトの実行環境。
// 実行環境 (変数・関数) と静的検査のシンボル環境は RunString / RunFile の
// 呼び出し間で共有される (ホストから段階的にスクリプトを駆動できる)
type Script struct {
	env       *object.Environment
	checker   *checker.Checker
	outWriter io.Writer
}

// New は新しい Script を作る。print の出力先は標準出力
func New() *Script {
	s := &Script{
		env:       object.NewEnvironment(),
		checker:   checker.NewChecker(),
		outWriter: os.Stdout,
	}
	s.installBuiltins()
	return s
}

// SetOutput は print の出力先を変更する
func (s *Script) SetOutput(w io.Writer) {
	s.outWriter = w
	s.installBuiltins()
}

// installBuiltins は現在の出力先で print / len を再登録する
func (s *Script) installBuiltins() {
	evaluator.InstallBuiltins(s.env, s.outWriter)
}

// SetFunction はホスト関数 (embed API の Go 関数) を登録する。
//
//   - returnType は静的検査用の型名 (例 "INT")。空なら戻り値なし
//   - 引数の個数と型はホスト側が検証する (ruli からは可変長に見える)
//   - スクリプト内で同名の関数を定義するとそちらが優先される
func (s *Script) SetFunction(name string, returnType string, fn func(args ...Value) (Value, error)) error {
	if err := s.checker.SetExternal(checker.External{Name: name, ReturnType: returnType}); err != nil {
		return err
	}
	s.env.Set(name, &object.Builtin{Fn: fn})
	return nil
}

// 値コンストラクタ (ホスト関数の戻り値を作るために使う)

func NewIntValue(v int) Value              { return &object.Integer{Value: v} }
func NewRealValue(v float64) Value         { return &object.Real{Value: v} }
func NewBoolValue(v bool) Value            { return &object.Boolean{Value: v} }
func NewStringValue(v string) Value        { return &object.String{Value: v} }
func NewArrayValue(elements []Value) Value { return &object.Array{Elements: elements} }
func NewStructValue(fields map[string]Value) Value {
	return &object.Struct{Fields: fields}
}
func NewNullValue() Value { return &object.Null{} }

// RunString はソース文字列を実行する (parse → 型検査 → evaluator)。
// エラーは 行:列 形式。複数の parse / 型エラーは改行で連結される。
// 実行結果の値が不要な用途を想定している。環境は次回の実行に引き継がれる
func (s *Script) RunString(src string) error {
	_, err := s.EvalString(src)
	return err
}

// EvalString はソース文字列を実行し、最後の文の値を返す (REPL 用)。
// break / continue は Null に、トップレベル return はその値に畳まれる
func (s *Script) EvalString(src string) (Value, error) {
	program, err := s.Program(src)
	if err != nil {
		return nil, err
	}

	result := evaluator.Eval(program, s.env)
	if errObj, ok := result.(*object.Error); ok {
		return nil, errors.New(errObj.Inspect())
	}

	switch v := result.(type) {
	case *object.BreakSignal, *object.ContinueSignal:
		return &object.Null{}, nil
	case *object.ReturnValue:
		if len(v.Values) == 1 {
			return v.Values[0], nil
		}
	}

	return result, nil
}

// RunFile は .ruli ファイルを実行する
func (s *Script) RunFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return s.RunString(string(src))
}

// Program はソース文字列を parse と型検査まで通し、AST を返す (解析・テスト用)
func (s *Script) Program(src string) (*ast.Program, error) {
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		return nil, joinParseErrors(errs)
	}

	if errs := s.checker.Check(program); len(errs) > 0 {
		return nil, joinTypeErrors(errs)
	}

	return program, nil
}

// Env は実行環境を返す。変数の読み書きをホストから直接行いたい場合に使う
func (s *Script) Env() *object.Environment {
	return s.env
}

func joinParseErrors(errs []parser.ParseError) error {
	list := make([]error, 0, len(errs))
	for i := range errs {
		list = append(list, &errs[i])
	}
	return errors.Join(list...)
}

func joinTypeErrors(errs []checker.TypeError) error {
	list := make([]error, 0, len(errs))
	for i := range errs {
		list = append(list, &errs[i])
	}
	return errors.Join(list...)
}
