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
		return
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
			return
		}

		val := stmt.(*ast.VarDeclStatement).Value
		if !testLiteralExpression(t, val, tt.expectedValue) {
			return
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

	if varDecl.Type.String() != expectedType {
		t.Errorf("varDecl.Type not %s. got=%s", expectedType, varDecl.Type.String())
		return false
	}

	return true
}

func testLiteralExpression(t *testing.T, exp ast.Expression, expected interface{}) bool {
	switch v := expected.(type) {
	case int:
		return testIntegerLiteral(t, exp, v)
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
