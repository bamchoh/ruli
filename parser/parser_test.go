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

	if assignStmt.Name != "x" {
		t.Fatalf("assignStmt.Name not %s. got=%s", "x", assignStmt.Name)
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

	if varDecl.Name != name {
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

	if assignStmt.Name != name {
		t.Errorf("assignStmt.Name not %s. got=%s", name, assignStmt.Name)
		return false
	}

	if !testIdentifier(t, assignStmt.Value, value) {
		return false
	}

	return true
}
