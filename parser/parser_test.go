package parser

import (
	"ruli/ast"
	"ruli/lexer"
	"testing"
)

func TestAssignStatement(t *testing.T) {
	input := `x = 10`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]
	assignStmt, ok := stmt.(*ast.AssignStatement)
	if !ok {
		t.Fatalf("stmt not *ast.AssignStatement. got=%T", stmt)
	}

	if assignStmt.Left.(*ast.Identifier).Value != "x" {
		t.Fatalf("assignStmt.Left.(*ast.Identifier).Value not %s. got=%s", "x", assignStmt.Left.(*ast.Identifier).Value)
	}

	val := assignStmt.Value
	if !testLiteralExpression(t, val, 10) {
		t.Fatalf("stmt.Value not as expected. got=%T", val)
	}

}

func TestVarDeclStatement(t *testing.T) {
	tests := []struct {
		input              string
		expectedIdentifier string
		expectedType       string
		expectedValue      interface{}
	}{
		{`x : INT = 10`, "x", "INT", 10},
		{`x : INT`, "x", "INT", nil},
		{`x := 20`, "x", "", 20},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)

		program := p.ParseProgram()

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt := program.Statements[0]
		if !testVarDeclStatement(t, stmt, tt.expectedIdentifier, tt.expectedType) {
			t.Fatalf("stmt not as expected. got=%T", stmt)
		}

		val := stmt.(*ast.VarDeclStatement).Value
		if !testLiteralExpression(t, val, tt.expectedValue) {
			t.Fatalf("stmt.Value not as expected. got=%T", val)
		}
	}
}

func testVarDeclStatement(t *testing.T, s ast.Statement, name string, expectedType string) bool {
	varDecl, ok := s.(*ast.VarDeclStatement)
	if !ok {
		t.Errorf("s not *ast.VarDeclStatement. got=%T", s)
		return false
	}

	if varDecl.Name.Value != name {
		t.Errorf("varDecl.Name not %s. got=%s", name, varDecl.Name)
		return false
	}

	if varDecl.Type == nil && expectedType == "" {
		return true
	}

	if varDecl.Type.String() != expectedType {
		t.Errorf("varDecl.Type not %s. got=%s", expectedType, varDecl.Type.String())
		return false
	}

	return true
}

func testIdentifier(t *testing.T, exp ast.Expression, value string) bool {
	ident, ok := exp.(*ast.Identifier)
	if !ok {
		t.Errorf("exp not *ast.Identifier. got=%T", exp)
		return false
	}

	if ident.Value != value {
		t.Errorf("ident.Value not %s. got=%s", value, ident.Value)
		return false
	}

	return true
}

func testLiteralExpression(t *testing.T, exp ast.Expression, expected interface{}) bool {
	switch v := expected.(type) {
	case int:
		return testIntegerLiteral(t, exp, v)
	case string:
		return testIdentifier(t, exp, v)
	default:
		if exp == nil && expected == nil {
			return true
		}
		t.Errorf("type of exp not handled. got=%T", exp)
	}
	return false
}

func testIntegerLiteral(t *testing.T, exp ast.Expression, value int) bool {
	integ, ok := exp.(*ast.IntegerLiteral)
	if !ok {
		t.Errorf("exp not *ast.IntegerLiteral. got=%T", exp)
		return false
	}

	if integ.Value != value {
		t.Errorf("integ.Value not %d. got=%d", value, integ.Value)
		return false
	}
	return true
}

func TestIfStatement(t *testing.T) {
	input := `if x > 5 { y := x * 2 } else { y := x / 2 }`

	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if ifstmt, ok := stmt.(*ast.IfStatement); !ok {
		t.Fatalf("stmt not *ast.IfStatement. got=%T", stmt)
	} else {
		if !testInfixExpression(t, ifstmt.Condition, "x", ">", 5) {
			t.Fatalf("Condition not as expected")
		}

		if len(ifstmt.Consequence.Statements) != 1 {
			t.Fatalf("ifstmt.Consequence.Statements does not contain 1 statement. got=%d",
				len(ifstmt.Consequence.Statements))
		}

		consequenceStmt := ifstmt.Consequence.Statements[0]

		if !testVarDeclStatement(t, consequenceStmt, "y", "") {
			t.Fatalf("consequenceStmt not as expected")
		}

		if !testInfixExpression(t, consequenceStmt.(*ast.VarDeclStatement).Value, "x", "*", 2) {
			t.Fatalf("consequenceStmt.Value not as expected")
		}

		if len(ifstmt.Alternative.Statements) != 1 {
			t.Fatalf("ifstmt.Alternative.Statements does not contain 1 statement. got=%d",
				len(ifstmt.Alternative.Statements))
		}

		alternativeStmt := ifstmt.Alternative.Statements[0]

		if !testVarDeclStatement(t, alternativeStmt, "y", "") {
			t.Fatalf("alternativeStmt not as expected")
		}

		if !testInfixExpression(t, alternativeStmt.(*ast.VarDeclStatement).Value, "x", "/", 2) {
			t.Fatalf("alternativeStmt.Value not as expected")
		}
	}
}

func TestIfStatement2(t *testing.T) {
	input := `if x > 5 { y := x * 2 } else if x < 3 { y := x / 2 }`

	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if ifstmt, ok := stmt.(*ast.IfStatement); !ok {
		t.Fatalf("stmt not *ast.IfStatement. got=%T", stmt)
	} else {
		if !testInfixExpression(t, ifstmt.Condition, "x", ">", 5) {
			t.Fatalf("Condition not as expected")
		}

		if len(ifstmt.Consequence.Statements) != 1 {
			t.Fatalf("ifstmt.Consequence.Statements does not contain 1 statement. got=%d",
				len(ifstmt.Consequence.Statements))
		}

		consequenceStmt := ifstmt.Consequence.Statements[0]

		if !testVarDeclStatement(t, consequenceStmt, "y", "") {
			t.Fatalf("consequenceStmt not as expected")
		}

		if !testInfixExpression(t, consequenceStmt.(*ast.VarDeclStatement).Value, "x", "*", 2) {
			t.Fatalf("consequenceStmt.Value not as expected")
		}

		if len(ifstmt.Alternative.Statements) != 1 {
			t.Fatalf("ifstmt.Alternative.Statements does not contain 1 statement. got=%d",
				len(ifstmt.Alternative.Statements))
		}

		alternativeStmt := ifstmt.Alternative.Statements[0]

		if ifstmt2, ok := alternativeStmt.(*ast.IfStatement); !ok {
			t.Fatalf("alternativeStmt not *ast.IfStatement. got=%T", alternativeStmt)
		} else {
			if !testInfixExpression(t, ifstmt2.Condition, "x", "<", 3) {
				t.Fatalf("Condition of else if not as expected")
			}

			if len(ifstmt2.Consequence.Statements) != 1 {
				t.Fatalf("ifstmt2.Consequence.Statements does not contain 1 statement. got=%d",
					len(ifstmt2.Consequence.Statements))
			}

			consequenceStmt2 := ifstmt2.Consequence.Statements[0]

			if !testVarDeclStatement(t, consequenceStmt2, "y", "") {
				t.Fatalf("consequenceStmt2 not as expected")
			}

			if !testInfixExpression(t, consequenceStmt2.(*ast.VarDeclStatement).Value, "x", "/", 2) {
				t.Fatalf("consequenceStmt2.Value not as expected")
			}
		}
	}
}

func testInfixExpression(t *testing.T, exp ast.Expression, left interface{}, operator string, right interface{}) bool {
	opExp, ok := exp.(*ast.BinaryExpression)
	if !ok {
		t.Errorf("exp not *ast.BinaryExpression. got=%T", exp)
		return false
	}

	if !testLiteralExpression(t, opExp.Left, left) {
		return false
	}

	if opExp.Operator != operator {
		t.Errorf("exp.Operator is not '%s'. got=%s", operator, opExp.Operator)
		return false
	}

	if !testLiteralExpression(t, opExp.Right, right) {
		return false
	}

	return true
}

func TestIncrementDecrementStatement(t *testing.T) {
	tests := []struct {
		input            string
		expectedName     string
		expectedOperator string
	}{
		{`x++`, "x", "++"},
		{`x--`, "x", "--"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)

		program := p.ParseProgram()

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt := program.Statements[0]

		if incDecStmt, ok := stmt.(*ast.IncDecStatement); !ok {
			t.Fatalf("stmt not *ast.IncDecStatement. got=%T", stmt)
		} else {
			if incDecStmt.Name != tt.expectedName {
				t.Fatalf("Name not as expected")
			}

			if incDecStmt.Operator != tt.expectedOperator {
				t.Fatalf("Operator not as expected. got=%s", incDecStmt.Operator)
			}
		}
	}
}

func TestForStatement(t *testing.T) {
	input := `for i := 0; i < 10; i++ { x = i }`

	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if forStmt, ok := stmt.(*ast.ForStatement); !ok {
		t.Fatalf("stmt not *ast.ForStatement. got=%T", stmt)
	} else {
		if !testVarDeclStatement(t, forStmt.Init, "i", "") {
			t.Fatalf("Init not as expected")
		}

		if !testInfixExpression(t, forStmt.Condition, "i", "<", 10) {
			t.Fatalf("Condition not as expected")
		}

		if incDecStmt, ok := forStmt.Post.(*ast.IncDecStatement); !ok {
			t.Fatalf("Post not *ast.IncDecStatement. got=%T", forStmt.Post)
		} else {
			if incDecStmt.Name != "i" {
				t.Fatalf("Post Name not as expected")
			}

			if incDecStmt.Operator != "++" {
				t.Fatalf("Post Operator not as expected. got=%s", incDecStmt.Operator)
			}
		}

		if len(forStmt.Body.Statements) != 1 {
			t.Fatalf("forStmt.Body.Statements does not contain 1 statement. got=%d",
				len(forStmt.Body.Statements))
		}

		bodyStmt := forStmt.Body.Statements[0]

		if !testAssignStatement(t, bodyStmt, "x", "i") {
			t.Fatalf("Body statement not as expected")
		}
	}
}

func testAssignStatement(t *testing.T, s ast.Statement, name string, value string) bool {
	assignStmt, ok := s.(*ast.AssignStatement)
	if !ok {
		t.Errorf("s not *ast.AssignStatement. got=%T", s)
		return false
	}

	if assignStmt.Left.(*ast.Identifier).Value != name {
		t.Errorf("assignStmt.Left.(*ast.Identifier).Value not %s. got=%s", name, assignStmt.Left.(*ast.Identifier).Value)
		return false
	}

	if !testIdentifier(t, assignStmt.Value, value) {
		return false
	}

	return true
}

func TestBuiltinFunctionStatement(t *testing.T) {
	input := `print(x)`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if exp, ok := stmt.(*ast.ExpressionStatement); !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", stmt)
	} else {
		if callExp, ok := exp.Expression.(*ast.CallExpression); !ok {
			t.Fatalf("exp.Expression not *ast.CallExpression. got=%T", exp.Expression)
		} else {
			if !testIdentifier(t, callExp.Function, "print") {
				t.Fatalf("Function not as expected")
			}

			if len(callExp.Arguments) != 1 {
				t.Fatalf("callExp.Arguments does not contain 1 argument. got=%d",
					len(callExp.Arguments))
			}

			if !testIdentifier(t, callExp.Arguments[0], "x") {
				t.Fatalf("Argument not as expected")
			}
		}
	}
}

func TestFunction1Statement(t *testing.T) {
	input := `func add(a: INT, b: INT) INT {
		return a + b
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if funcStmt, ok := stmt.(*ast.FunctionStatement); !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", stmt)
	} else {
		if funcStmt.Name != "add" {
			t.Fatalf("Function name not as expected. got=%s", funcStmt.Name)
		}

		if len(funcStmt.Parameters) != 2 {
			t.Fatalf("funcStmt.Parameters does not contain 2 parameters. got=%d",
				len(funcStmt.Parameters))
		}

		param1 := funcStmt.Parameters[0]
		if param1.Name != "a" || param1.Type.String() != "INT" {
			t.Fatalf("Parameter 1 not as expected. got=%s %s", param1.Name, param1.Type)
		}

		param2 := funcStmt.Parameters[1]
		if param2.Name != "b" || param2.Type.String() != "INT" {
			t.Fatalf("Parameter 2 not as expected. got=%s %s", param2.Name, param2.Type)
		}

		if len(funcStmt.ReturnTypes) != 1 || funcStmt.ReturnTypes[0].String() != "INT" {
			t.Fatalf("Return type not as expected. got=%v", funcStmt.ReturnTypes)
		}

		if len(funcStmt.Body.Statements) != 1 {
			t.Fatalf("funcStmt.Body.Statements does not contain 1 statement. got=%d",
				len(funcStmt.Body.Statements))
		}

		bodyStmt := funcStmt.Body.Statements[0]

		if returnStmt, ok := bodyStmt.(*ast.ReturnStatement); !ok {
			t.Fatalf("bodyStmt not *ast.ReturnStatement. got=%T", bodyStmt)
		} else {
			if !testInfixExpression(t, returnStmt.Values[0], "a", "+", "b") {
				t.Fatalf("Return value not as expected")
			}
		}
	}
}

func TestFunction2Statement(t *testing.T) {
	input := `func calc_and_dump(a: INT, b: INT) {
		print(a + b)
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if funcStmt, ok := stmt.(*ast.FunctionStatement); !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", stmt)
	} else {
		if funcStmt.Name != "calc_and_dump" {
			t.Fatalf("Function name not as expected. got=%s", funcStmt.Name)
		}

		if len(funcStmt.Parameters) != 2 {
			t.Fatalf("funcStmt.Parameters does not contain 2 parameters. got=%d",
				len(funcStmt.Parameters))
		}

		param1 := funcStmt.Parameters[0]
		if param1.Name != "a" || param1.Type.String() != "INT" {
			t.Fatalf("Parameter 1 not as expected. got=%s %s", param1.Name, param1.Type)
		}

		param2 := funcStmt.Parameters[1]
		if param2.Name != "b" || param2.Type.String() != "INT" {
			t.Fatalf("Parameter 2 not as expected. got=%s %s", param2.Name, param2.Type)
		}

		if len(funcStmt.ReturnTypes) != 0 {
			t.Fatalf("Return type not as expected. got=%v", funcStmt.ReturnTypes)
		}

		if len(funcStmt.Body.Statements) != 1 {
			t.Fatalf("funcStmt.Body.Statements does not contain 1 statement. got=%d",
				len(funcStmt.Body.Statements))
		}

		bodyStmt := funcStmt.Body.Statements[0]

		if exp, ok := bodyStmt.(*ast.ExpressionStatement); !ok {
			t.Fatalf("bodyStmt not *ast.ExpressionStatement. got=%T", bodyStmt)
		} else {
			if callExp, ok := exp.Expression.(*ast.CallExpression); !ok {
				t.Fatalf("exp.Expression not *ast.CallExpression. got=%T", exp.Expression)
			} else {
				if !testIdentifier(t, callExp.Function, "print") {
					t.Fatalf("Function not as expected")
				}

				if len(callExp.Arguments) != 1 {
					t.Fatalf("callExp.Arguments does not contain 1 argument. got=%d",
						len(callExp.Arguments))
				}

				if !testInfixExpression(t, callExp.Arguments[0], "a", "+", "b") {
					t.Fatalf("Argument not as expected")
				}
			}
		}
	}
}

func TestArrayIndexExpression(t *testing.T) {
	input := `
	nums := [1, 2, 3]
	x := nums[1]
	`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d",
			len(program.Statements))
	}

	stmt := program.Statements[0]

	if varDecl, ok := stmt.(*ast.VarDeclStatement); !ok {
		t.Fatalf("stmt not *ast.VarDeclStatement. got=%T", stmt)
	} else {
		if varDecl.Name.Value != "nums" {
			t.Fatalf("Name not as expected. got=%s", varDecl.Name)
		}
		if arrayLit, ok := varDecl.Value.(*ast.ArrayLiteral); !ok {
			t.Fatalf("varDecl.Value not *ast.ArrayLiteral. got=%T", varDecl.Value)
		} else {
			if len(arrayLit.Elements) != 3 {
				t.Fatalf("arrayLit.Elements does not contain 3 elements. got=%d",
					len(arrayLit.Elements))
			}

			if !testLiteralExpression(t, arrayLit.Elements[0], 1) {
				t.Fatalf("Element 0 not as expected")
			}

			if !testLiteralExpression(t, arrayLit.Elements[1], 2) {
				t.Fatalf("Element 1 not as expected")
			}

			if !testLiteralExpression(t, arrayLit.Elements[2], 3) {
				t.Fatalf("Element 2 not as expected")
			}
		}
	}

	stmt = program.Statements[1]

	if assignStmt, ok := stmt.(*ast.VarDeclStatement); !ok {
		t.Fatalf("stmt not *ast.VarDeclStatement. got=%T", stmt)
	} else {
		if assignStmt.Name.Value != "x" {
			t.Fatalf("Name not as expected. got=%s", assignStmt.Name)
		}

		if indexExp, ok := assignStmt.Value.(*ast.IndexExpression); !ok {
			t.Fatalf("assignStmt.Value not *ast.IndexExpression. got=%T", assignStmt.Value)
		} else {
			if !testIdentifier(t, indexExp.Left, "nums") {
				t.Fatalf("Left not as expected")
			}

			if !testLiteralExpression(t, indexExp.Index, 1) {
				t.Fatalf("Index not as expected")
			}
		}
	}
}

func testParseErrors(input string, expectedErrs []string, t *testing.T) {
	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()

	errs := p.Errors()
	if len(errs) != len(expectedErrs) {
		t.Fatalf("wrong number of errors. got=%d (%v), want=%d",
			len(errs), errs, len(expectedErrs))
	}

	for i, want := range expectedErrs {
		if errs[i].Message != want {
			t.Fatalf("error %d message not as expected. got=%s, want=%s",
				i, errs[i].Message, want)
		}
	}
}

func TestParseErrorNoPrefix(t *testing.T) {
	testParseErrors(
		`if flag { } else { x } =`,
		[]string{"unexpected token '='"},
		t,
	)
}

func TestParseErrorInvalidIntegerLiteral(t *testing.T) {
	testParseErrors(
		`999999999999999999999999999`,
		[]string{"invalid integer literal '999999999999999999999999999'"},
		t,
	)
}

func TestBareIndexExpressionStatement(t *testing.T) {
	// a[0] のようなインデックス式単独の文は式文として有効な構文
	l := lexer.New(`a[0]`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
	if _, ok := program.Statements[0].(*ast.ExpressionStatement); !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
}

func TestParseErrorIndexedAssignWithoutValue(t *testing.T) {
	// 変数宣言とインデックスの欠落で式の解析に失敗する(旧実装は文ごと黙って破棄)
	testParseErrors(
		`x := nums[1`,
		[]string{"expected ']', got 'EOF'"},
		t,
	)
}

func TestParseErrorIfWithoutBrace(t *testing.T) {
	testParseErrors(
		`if flag
x := 1`,
		[]string{"expected '{', got 'x'"},
		t,
	)
}

func TestParseErrorUnclosedBlock(t *testing.T) {
	testParseErrors(
		`if flag {
	print("flag is true")
`,
		[]string{"expected '}', got EOF"},
		t,
	)
}

func TestParseErrorMissingClosingParen(t *testing.T) {
	testParseErrors(
		`print("a", "b"`,
		[]string{"expected ')', got 'EOF'"},
		t,
	)
}

func TestParseErrorUnclosedBracket(t *testing.T) {
	testParseErrors(
		`a := nums[1`,
		[]string{
			"expected ']', got 'EOF'",
		},
		t,
	)
}

func TestParseErrorUnknownType(t *testing.T) {
	testParseErrors(
		`x : = 10`,
		[]string{"unknown type '='"},
		t,
	)
}

func TestParseErrorMissingSemicolonInFor(t *testing.T) {
	// エラー復帰機能を持たないため、最初のエラー以降はカスケードする
	testParseErrors(
		`for i := 0 i < 10; i++ { }`,
		[]string{
			"expected ';', got 'i'",
			"unexpected token '{'",
			"unexpected token '}'",
		},
		t,
	)
}

func TestParseErrorHasPosition(t *testing.T) {
	l := lexer.New("x := 1\n}")
	p := New(l)
	p.ParseProgram()

	errs := p.Errors()
	if len(errs) != 1 {
		t.Fatalf("wrong number of errors. got=%d (%v)", len(errs), errs)
	}

	if errs[0].Message != "unexpected token '}'" {
		t.Fatalf("wrong message. got=%s", errs[0].Message)
	}

	// 2 行目の '}' を指す
	if errs[0].Line != 2 {
		t.Fatalf("wrong line. got=%d, want=2", errs[0].Line)
	}
}

func TestCastExpression(t *testing.T) {
	l := lexer.New(`REAL(a)`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	cast, ok := exprStmt.Expression.(*ast.CastExpression)
	if !ok {
		t.Fatalf("expression not *ast.CastExpression. got=%T", exprStmt.Expression)
	}
	if cast.Type != "REAL" {
		t.Fatalf("cast.Type not REAL. got=%s", cast.Type)
	}
	if !testIdentifier(t, cast.Value, "a") {
		t.Fatalf("cast.Value not as expected")
	}
}

func TestTypeDeclStatement(t *testing.T) {
	input := `type Motor struct {
	speed:  INT
	enable: BOOL
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	decl, ok := program.Statements[0].(*ast.TypeDeclStatement)
	if !ok {
		t.Fatalf("stmt not *ast.TypeDeclStatement. got=%T", program.Statements[0])
	}
	if decl.Name != "Motor" {
		t.Fatalf("decl.Name not Motor. got=%s", decl.Name)
	}
	if len(decl.Fields) != 2 {
		t.Fatalf("decl.Fields does not contain 2 fields. got=%d", len(decl.Fields))
	}
	if decl.Fields[0].Name != "speed" || decl.Fields[1].Name != "enable" {
		t.Fatalf("field names not as expected. got=%s, %s", decl.Fields[0].Name, decl.Fields[1].Name)
	}
}

func TestSelectorExpression(t *testing.T) {
	l := lexer.New(`m.speed`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	sel, ok := exprStmt.Expression.(*ast.SelectorExpression)
	if !ok {
		t.Fatalf("expression not *ast.SelectorExpression. got=%T", exprStmt.Expression)
	}
	if sel.Field != "speed" {
		t.Fatalf("sel.Field not speed. got=%s", sel.Field)
	}
	if !testIdentifier(t, sel.Object, "m") {
		t.Fatalf("sel.Object not as expected")
	}
}

func TestNestedSelectorExpression(t *testing.T) {
	l := lexer.New(`d.motor.speed`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	sel, ok := exprStmt.Expression.(*ast.SelectorExpression)
	if !ok {
		t.Fatalf("expression not *ast.SelectorExpression. got=%T", exprStmt.Expression)
	}
	if sel.Field != "speed" {
		t.Fatalf("sel.Field not speed. got=%s", sel.Field)
	}
	if !testIdentifier(t, sel.Object.(*ast.SelectorExpression).Object, "d") {
		t.Fatalf("innermost object not as expected")
	}
	if sel.Object.(*ast.SelectorExpression).Field != "motor" {
		t.Fatalf("inner field not motor")
	}
}

func TestMultiReturnFunction(t *testing.T) {
	input := `func div(a: INT, b: INT) (INT, BOOL) {
	return a / b, b != 0
}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	fn, ok := program.Statements[0].(*ast.FunctionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.FunctionStatement. got=%T", program.Statements[0])
	}
	if len(fn.ReturnTypes) != 2 ||
		fn.ReturnTypes[0].String() != "INT" || fn.ReturnTypes[1].String() != "BOOL" {
		t.Fatalf("ReturnTypes not as expected. got=%v", fn.ReturnTypes)
	}
}

func TestMultiVarDeclStatement(t *testing.T) {
	l := lexer.New(`x, ok := div(10, 2)`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.MultiVarDeclStatement)
	if !ok {
		t.Fatalf("stmt not *ast.MultiVarDeclStatement. got=%T", program.Statements[0])
	}
	if len(stmt.Names) != 2 || stmt.Names[0].Value != "x" || stmt.Names[1].Value != "ok" {
		t.Fatalf("Names not as expected")
	}
}

func TestMultiAssignStatement(t *testing.T) {
	l := lexer.New(`x, ok = div(10, 2)`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	stmt, ok := program.Statements[0].(*ast.MultiAssignStatement)
	if !ok {
		t.Fatalf("stmt not *ast.MultiAssignStatement. got=%T", program.Statements[0])
	}
	if len(stmt.Lefts) != 2 {
		t.Fatalf("Lefts does not contain 2. got=%d", len(stmt.Lefts))
	}
}

func TestArrayTypeParse(t *testing.T) {
	input := `values : [10]INT`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	decl, ok := program.Statements[0].(*ast.VarDeclStatement)
	if !ok {
		t.Fatalf("stmt not *ast.VarDeclStatement. got=%T", program.Statements[0])
	}
	at, ok := decl.Type.(*ast.ArrayType)
	if !ok {
		t.Fatalf("decl.Type not *ast.ArrayType. got=%T", decl.Type)
	}
	if at.Size != 10 {
		t.Fatalf("at.Size not 10. got=%d", at.Size)
	}
	if at.Elem.String() != "INT" {
		t.Fatalf("at.Elem not INT. got=%s", at.Elem.String())
	}
}

func TestNestedArrayTypeParse(t *testing.T) {
	l := lexer.New(`matrix: [3][3]INT`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	decl := program.Statements[0].(*ast.VarDeclStatement)
	at := decl.Type.(*ast.ArrayType)
	if at.Size != 3 {
		t.Fatalf("outer Size not 3. got=%d", at.Size)
	}
	nested, ok := at.Elem.(*ast.ArrayType)
	if !ok {
		t.Fatalf("at.Elem not *ast.ArrayType. got=%T", at.Elem)
	}
	if nested.Size != 3 || nested.Elem.String() != "INT" {
		t.Fatalf("nested not as expected")
	}
}

func TestArrayParamAndReturnTypeParse(t *testing.T) {
	input := `func f(v: [4]INT) [4]INT {
	return [1, 2, 3, 4]
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	fn := program.Statements[0].(*ast.FunctionStatement)
	if fn.Parameters[0].Type.String() != "[4]INT" {
		t.Fatalf("param type not [4]INT. got=%s", fn.Parameters[0].Type.String())
	}
	if fn.ReturnTypes[0].String() != "[4]INT" {
		t.Fatalf("return type not [4]INT. got=%s", fn.ReturnTypes[0].String())
	}
}

func TestArrayTypeParseErrors(t *testing.T) {
	// エラー復帰を持たないためカスケードする
	testParseErrors(`x : [0]INT`, []string{
		"array size must be positive, got 0",
		"unknown type '0'",
		"unexpected token ']'",
	}, t)
}

func TestSwitchStatementParse(t *testing.T) {
	input := `switch x {
	case 1, 2:
		print("low")
	default:
		print("other")
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	sw, ok := program.Statements[0].(*ast.SwitchStatement)
	if !ok {
		t.Fatalf("stmt not *ast.SwitchStatement. got=%T", program.Statements[0])
	}
	if !testIdentifier(t, sw.Value, "x") {
		t.Fatalf("sw.Value not as expected")
	}
	if len(sw.Cases) != 1 {
		t.Fatalf("sw.Cases does not contain 1 case. got=%d", len(sw.Cases))
	}
	if len(sw.Cases[0].Values) != 2 {
		t.Fatalf("case does not contain 2 values. got=%d", len(sw.Cases[0].Values))
	}
	if len(sw.Cases[0].Body.Statements) != 1 {
		t.Fatalf("case body does not contain 1 statement")
	}
	if sw.Default == nil || len(sw.Default.Values) != 0 {
		t.Fatalf("default not as expected")
	}
}

func TestCompoundAssignParse(t *testing.T) {
	input := `x += 1`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.CompoundAssignStatement)
	if !ok {
		t.Fatalf("stmt not *ast.CompoundAssignStatement. got=%T", program.Statements[0])
	}
	if stmt.Operator != "+=" {
		t.Fatalf("Operator not +=. got=%s", stmt.Operator)
	}
	if !testIdentifier(t, stmt.Target, "x") {
		t.Fatalf("Target not as expected")
	}
	if !testLiteralExpression(t, stmt.Value, 1) {
		t.Fatalf("Value not as expected")
	}
}

func TestCompoundAssignIndexParse(t *testing.T) {
	l := lexer.New(`buf[1] *= 10`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	stmt, ok := program.Statements[0].(*ast.CompoundAssignStatement)
	if !ok {
		t.Fatalf("stmt not *ast.CompoundAssignStatement. got=%T", program.Statements[0])
	}
	if _, ok := stmt.Target.(*ast.IndexExpression); !ok {
		t.Fatalf("Target not *ast.IndexExpression. got=%T", stmt.Target)
	}
}

func TestLogicalExpressionPrecedence(t *testing.T) {
	// Go 相当の優先度: || < && < 比較
	input := `x == 1 && flag || other`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	// (x == 1 && flag) || other
	or, ok := stmt.Expression.(*ast.LogicalExpression)
	if !ok || or.Operator != "||" {
		t.Fatalf("top not OR LogicalExpression. got=%T", stmt.Expression)
	}
	and, ok := or.Left.(*ast.LogicalExpression)
	if !ok || and.Operator != "&&" {
		t.Fatalf("left not AND LogicalExpression. got=%T", or.Left)
	}
	if _, ok := and.Left.(*ast.BinaryExpression); !ok {
		t.Fatalf("and.Left not BinaryExpression. got=%T", and.Left)
	}
}

func TestDistinctTypeParse(t *testing.T) {
	l := lexer.New(`type TEMP INT`)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("unexpected parse errors: %v", p.Errors())
	}

	decl, ok := program.Statements[0].(*ast.TypeDeclStatement)
	if !ok {
		t.Fatalf("stmt not *ast.TypeDeclStatement. got=%T", program.Statements[0])
	}
	if decl.Name != "TEMP" {
		t.Fatalf("decl.Name not TEMP. got=%s", decl.Name)
	}
	if decl.Base == nil || decl.Base.String() != "INT" {
		t.Fatalf("decl.Base not INT. got=%v", decl.Base)
	}
	if len(decl.Fields) != 0 {
		t.Fatalf("distinct type must have no fields. got=%d", len(decl.Fields))
	}
}
