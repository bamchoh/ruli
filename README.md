# ruli

<p align="center">
  <img alt="Logo" src="logo.png" width="50%" height="50%">
</p>

[English](README.md) | 日本語

**ruli** is an embeddable, statically-typed, AST-interpreter scripting language written in Go.

- **Statically typed**: type mismatches, undefined references, array bounds, and fixed-width range violations are all checked before execution
- **IEC 61131-3-style types**: fixed-width types such as `SINT`/`DINT`/`WORD`, plus `STRING(n)`
- **Pass-by-value semantics**: `=` and arguments always copy values. Use `ref` explicitly when you need to write through (equivalent to VAR_IN_OUT)
- **Go embedding**: register host functions, read and write variables directly, and swap the output destination
- Every error is reported in `line:column` form, and a single mistake converges into a single error

For the full language specification as it stands, see [`docs/SPEC.md`](docs/SPEC.md) (in Japanese).

## Quick Start

```
# Run a sample script
go run ./cmd/ruli examples/hello.ruli

# REPL (starts with no arguments)
go run ./cmd/ruli

# Build
go build -o ruli.exe ./cmd/ruli
./ruli.exe examples/methods.ruli
```

Examples: [`hello.ruli`](examples/hello.ruli) basic syntax / [`methods.ruli`](examples/methods.ruli) structs + methods / [`pid.ruli`](examples/pid.ruli) PI control loop.

## Language Tour

### Variables and Scope

```
x := 10            # binding (declared in the current scope)
y : INT            # typed declaration (initialized to the zero value)
age : WORD = 5     # declaration + initialization

x = 20             # = only updates existing variables (assigning to an undefined name is an error)

if true {
	x := 99         # block scope + shadowing
}
print(x)           # 20 (:= inside a block doesn't pollute the outer scope)
```

### Fixed-Width Types and Range Checking

Types descended from IEC 61131-3. Ranges are checked at declaration, assignment, `++`/`--`, and casts; overflow is an **error**, not a silent wrap.

| Type                 | Signedness    | Range                    | Notes                    |
| -------------------- | ------------- | ------------------------ | ------------------------ |
| BOOL                 | -             | true/false               |                          |
| SINT / USINT / BYTE  | ± / none      | −128..127 / 0..255       | 8-bit                    |
| INT / UINT / WORD    | ± / none      | −32768..32767 / 0..65535 | 16-bit                   |
| DINT / UDINT / DWORD | ± / none      | 32-bit                   |                          |
| LINT / ULINT / LWORD | ± / none      | 64-bit                   |                          |
| REAL / LREAL         | -             | float / double           |                          |
| STRING / STRING(n)   | -             | string                   | n is the max length (runes) |

```
temp : SINT = -40        # OK
temp = 200               # value 200 out of range for SINT (-128..127)
c := REAL(x) + y         # INT + REAL is an error; an explicit cast is required
DINT(REAL(7) / REAL(2))  # real → integer truncates toward zero
```

### Strings

```
name := "瑠璃"
greeting := "こんにちは" + name
len(greeting)      # number of runes
greeting[0]        # "こ" (a single character, returned as a STRING)
"abc" < "abd"      # lexicographic comparison (== != < <= > >=)

title : STRING(20) = "..."   # max 20 characters; overlong assignment is an error
```

### nil

`nil` is a literal meaning "no value". Since the language is pass-by-value, structs / arrays / function values can never become nil; the main use is checking whether a **host function returned no value**.

```
v := findTag("S1")    # when a host function returns nil
if v != nil {
	print(v)
}
```

- The only comparisons available are `== nil` / `!= nil` (it cannot be used in other operations, or as an if/for/switch condition)
- `x := nil` creates a NULL-typed variable; arithmetic or string operations on it are type errors
- Assigning nil to a typed declaration such as `n : INT = nil` is a type mismatch error

### Type System

```
# struct (SPEC 4.3) + struct literal (unspecified fields get the zero value)
type Motor struct {
	speed:  INT
	enable: BOOL
}

m := Motor{speed: 10}
m.speed = 20                # field assignment
m.stop()                    # method
n := m                      # pass-by-value (deep copy)

# Fixed-size arrays (the size is part of the type)
values : [10]INT            # declared zero-filled
matrix : [3][3]INT
values[0] = 5               # runtime + static bounds checking
buf : [8]WORD = io.inputs   # mismatched sizes are a static error

# distinct types (nominal: no implicit conversion with the underlying type)
type TEMP INT
t : TEMP = TEMP(30)         # build with TEMP(30) / get back with INT(t)
```

### Methods

```
func (m: Motor) isStopped() BOOL {       # value receiver (changes are not reflected)
	return m.speed == 0
}

func (m: ref Motor) stop() BOOL { # ref receiver (changes are reflected in the caller)
	m.speed = 0
	return true
}

m.stop()                                 # called via postfix selector
```

### Functions

```
func add(a: INT, b: INT) INT {
	return a + b
}

# Multiple return values (SPEC 5.3). Assigning multiple values to a single variable is a static error
func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}

q, ok := div(10, 2)

# ref parameters (equivalent to IEC VAR_IN_OUT)
func reset(m: ref Motor) BOOL {
	m.speed = 0
	return true
}
reset(m)         # unqualified at the call site; changes inside the function are reflected
```

### First-Class Functions

```
func apply(fn: FUNC(x: INT) INT, v: INT) INT {
	return fn(v)
}

func double(x: INT) INT { return x * 2 }

apply(double, 21)     # 42
g := double           # stored in a variable
g(10)                 # 20
pick(true)(4)         # a function returning a function
```

### Control Flow

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

- Expression results are printed as `= value` (declarations, assignments, and function definitions are not echoed)
- Incomplete syntax (e.g. `if x {`) continues on the `   >` prompt
- `.help` / `.exit`, or Ctrl-D, to quit

## Embedding from Go (embed API)

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
	s.SetOutput(os.Stdout) // the output destination for print (standard output by default)

	// Register a host function. It appears as a variadic function in ruli;
	// the declared return type is used for static type checking
	s.SetFunction("readTag", "INT", func(args ...ruli.Value) (ruli.Value, error) {
		// Do the actual tag read here
		return ruli.NewIntValue(42), nil
	})

	// The runtime environment and the type-checking symbols are shared across calls
	if err := s.RunFile("control.ruli"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Read a variable directly from the host
	v, _ := s.Env().Get("alarmCount")
	fmt.Println("alarms:", v.Inspect())
}
```

Other host APIs: `RunString` / `EvalString` (returns a value) / `SetOutput` / `Env()`. Value constructors include `NewIntValue` / `NewRealValue` / `NewBoolValue` / `NewStringValue` / `NewArrayValue` / `NewStructValue` / `NewNullValue`.

## Error Reporting

```
$ cat bad.ruli
temp : SINT = 200

$ ruli bad.ruli
1:7: value 200 out of range for SINT (-128..127)
```

Checking runs in the order parse → type check → execute, and stops at the first failing stage. Multiple parse/type errors are listed line by line.

## Architecture

```
lexer → parser → ast → checker → evaluator
                                  ↘ object (values, environment, runtime type definitions)

ruli (root package: embed API)
cmd/ruli (CLI / REPL)
```

- **checker**: static type checking (type mismatches, BOOL conditions, bounds, ranges, signatures). Collects all errors before execution
- **evaluator**: AST tree walk. Differentiates environment behavior between declarations and assignments
- **object**: value representation (sharing Go int/float64 as the value model) + deep copies for pass-by-value

## Development

```
go build ./...    # Build
go test ./...     # All tests (evaluator/parser/lexer/checker/ruli/cmd)
go vet ./...      # Static analysis
```

- Language spec: [`docs/SPEC.md`](docs/SPEC.md) (in Japanese)
- Tests: unit tests + regression tests over `examples/`
- Phase 2 (bytecode VM) is described as future work in SPEC 2.3

## License

MIT — [LICENSE](LICENSE)