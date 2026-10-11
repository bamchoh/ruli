# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ruli（瑠璃）is an embeddable, statically-typed AST-interpreter scripting language written in Go, aimed at control/PLC (IEC 61131-style) use cases. The language spec lives in `docs/SPEC.md` — treat it as the source of truth for intended behavior; the implementation is still catching up to parts of it (e.g. distinct types, structs, fixed-size array types, multi-return functions are specced but not all implemented yet).

`cmd/ruli/main.go` is the CLI: `ruli <script>` runs a `.ruli` file; with no arguments it starts a REPL (`cmd/ruli/repl.go`, built on the root embed API). Wire up new language features by extending that pipeline.

## Commands

- Build: `go build -v ./...`
- Run all tests: `go test ./...`
- Run one package's tests: `go test ./parser` (or `./lexer`)
- Run a single test: `go test ./parser -run TestParserName`
- Run a script: `go run ./cmd/ruli script.ruli` (or build once with `go build -o ruli.exe ./cmd/ruli`, then `./ruli.exe script.ruli`). No arguments starts the REPL (multi-line continuation, `= value` echo). `examples/` holds runnable sample scripts and cmd/ruli's tests execute them all
- Embed in a Go host: root package `ruli` (see `ruli.go`) — `ruli.New()` → `SetOutput` / `SetFunction(name, returnType, fn)` / `RunString` / `RunFile`. The evaluator environment and the checker's symbol env persist across `RunString` calls on one `Script`

CI (`.github/workflows/go.yml`) runs build + `go test ./...` on Go 1.25 for pushes/PRs to main. There are no linters configured.

## Architecture

Classic Monkey-style interpreter pipeline — packages depend on each other in this order only:

`lexer → parser → ast → checker/evaluator → object`

- `lexer/` — tokenizes (`#` for line comments). Defines `token.go`'s token types; `lexer_test.go` has table-driven tests.
- `ast/` — AST nodes. All nodes implement `ast.Node` via `GetToken()` (used by the evaluator to attach line/column info to runtime errors).
- `parser/` — Pratt parser producing `ast` nodes; `parser_test.go` checks `*ast.Program` structures.
- `checker/` — **nascent type checker**, not yet wired into main. `checker.Check` recursively infers `Type` from `ast` nodes against a `checker.Env` (separate from the runtime `object.Environment`; it maps names → `Type` instead of names → `Object`).
- `evaluator/` — tree-walking evaluator. Single entry `Eval(node ast.Node, env *object.Environment) object.Object`. Control flow (`break`/`continue`) and errors propagate as sentinel `*object.*` values up the call chain — check for them in every statement/expression case.
- `object/` — runtime values: `Integer`, `Real`, `Bool`, `Array`, `Function`, `Builtin`, `Error`, plus `Environment` (lexical scoping with `outer` chain). Built-ins (e.g. `print`) live in `evaluator`'s `builtins` map.

Key idioms to follow when extending:

- **Control-flow/error propagation:** expressions return `object.Object`; statements return sentinel objects (`BREAK_OBJ`, `CONTINUE_OBJ`, `RETURN_OBJ` wrapped in `*object.ReturnValue`, `*object.Error`) that callers unwrap at each level. Use `isError()` from `evaluator/helper.go`; create errors with `newErrorAtNode` so line/column are reported.
- **New language feature = touches across the stack:** token (lexer) → node (ast) → Pratt parser (parser) → evaluator case → tests. `docs/SPEC.md` defines the target semantics — match it, not Go semantics, where they differ (e.g. `:` declarations, `:=` binding, `#` comments, integer-typed language with fixed-width types planned).
- **Type model:** static types live in `checker` (fixed-width set INT/SINT~LWORD, REAL/LREAL/BOOL/STRING; same-family implicit conversion allowed, cross-family requires casts like `REAL(a)`). Runtime values are Go `int` (`object.Integer`) / `float64` (`object.Real`) — widths are not enforced at runtime yet. Arrays are deep-copied when bound (SPEC 4.4 値渡し) via `bindValue` in `evaluator/helper.go` and `Array.Copy`.
- Docs and SPEC.md are in Japanese; **commit messages are in English**.