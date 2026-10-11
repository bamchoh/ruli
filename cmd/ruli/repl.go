package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"ruli"
	"ruli/object"
)

// RunRepl は対話環境を起動する
func runRepl(out io.Writer, in io.Reader) {
	session := newReplSession(ruli.New())

	fmt.Fprintln(out, "ruli REPL — 終了は .exit または Ctrl-D")

	scanner := bufio.NewScanner(in)
	quit := false
	for !quit && scanner.Scan() {
		fmt.Fprint(out, session.prompt())

		var output string
		quit, output = session.processLine(scanner.Text())
		if output != "" {
			fmt.Fprintln(out, output)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(out, err)
	}
}

// replSession は REPL の入力状態を保持する (テスト可能な中核)
type replSession struct {
	script  *ruli.Script
	pending string // 複数行入力の蓄積
}

func newReplSession(s *ruli.Script) *replSession {
	return &replSession{script: s}
}

// prompt は現在の状態に応じたプロンプトを返す
func (r *replSession) prompt() string {
	if r.pending != "" {
		return "    > "
	}
	return "ruli> "
}

// processLine は 1 行を処理して表示すべき文字列を返す。
// quit は終了要求、err は入力エラー (表示して無視)
func (r *replSession) processLine(line string) (quit bool, output string) {
	trimmed := strings.TrimSpace(line)

	// ドットコマンド (蓄積中は通常の入力として扱う)
	if r.pending == "" {
		switch trimmed {
		case "":
			return false, ""
		case ".exit", ".quit":
			return true, ""
		case ".help":
			return false, `式の結果は "= 値" で表示される
if / for / func など複数行構文は "    > " プロンプトで継続 (閉じると即評価)
.exit または Ctrl-D で終了`
		}
	}

	r.pending += line + "\n"

	value, err := r.script.EvalString(r.pending)
	if err != nil {
		if isIncomplete(err) {
			// 構文が未完成 → 次の行で継続
			return false, ""
		}
		r.pending = ""
		return false, err.Error()
	}

	r.pending = ""

	if v := echoValue(value); v != nil {
		return false, "= " + v.Inspect()
	}
	return false, ""
}

// isIncomplete は入力がまだ続いている (EOF までの欠落) 場合 true。
// parse エラーのメッセージに EOF を含むものを未完成とみなす
func isIncomplete(err error) bool {
	return strings.Contains(err.Error(), "EOF")
}

// echoValue は REPL に表示すべき値を返す (表示不要なら nil)
func echoValue(value ruli.Value) ruli.Value {
	switch v := value.(type) {
	case *object.Null:
		return nil
	case *object.ReturnValue:
		if len(v.Values) == 1 {
			return v.Values[0]
		}
	}
	return value
}
