package checker

import (
	"fmt"
	"ruli/ast"
)

func Check(node ast.Node, env *Env) (Type, error) {
	switch n := node.(type) {
	case *ast.IntegerLiteral:
		return INT_TYPE, nil
	case *ast.BooleanLiteral:
		return BOOL_TYPE, nil
	case *ast.StringLiteral:
		return STRING_TYPE, nil
	case *ast.ArrayLiteral:
		return ARRAY_TYPE, nil
	case *ast.Identifier:
		t, ok := env.Get(n.Value)

		if !ok {
			return UNKNOWN_TYPE,
				fmt.Errorf(
					"undefined variable: %s",
					n.Value,
				)
		}

		return t, nil
	case *ast.VarDeclStatement:

		valueType, err := Check(n.Value, env)
		if err != nil {
			return UNKNOWN_TYPE, err
		}

		declType, err := resolveTypeNode(n.Type)
		if err != nil {
			return UNKNOWN_TYPE, err
		}

		if declType != valueType {
			return UNKNOWN_TYPE,
				fmt.Errorf(
					"type mismatch: declared %s, but got %s",
					declType,
					valueType,
				)
		}

		env.Set(n.Name.Value, declType)

		return UNKNOWN_TYPE, nil
	}

	return UNKNOWN_TYPE, fmt.Errorf("unknown node type: %T", node)
}

func resolveTypeNode(node ast.TypeNode) (Type, error) {
	switch n := node.(type) {
	case *ast.BasicType:

		switch n.Name {
		case "INT":
			return INT_TYPE, nil
		case "BOOL":
			return BOOL_TYPE, nil
		case "STRING":
			return STRING_TYPE, nil
		case "REAL":
			return REAL_TYPE, nil
		default:
			return UNKNOWN_TYPE, fmt.Errorf("unknown type: %s", n.Name)
		}
	}

	return UNKNOWN_TYPE, fmt.Errorf("unknown type node: %T", node)
}
