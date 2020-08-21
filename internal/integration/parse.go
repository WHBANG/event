package integration

import (
	"fmt"
	"go/ast"
	"go/token"
)

func litId(node ast.Expr) (string, error) {
	switch id := node.(type) {
	case *ast.Ident:
		return id.Name, nil
	case *ast.BasicLit:
		value := id.Value
		//字符串常量需要把两边引号去掉
		if id.Kind == token.STRING || id.Kind == token.CHAR {
			value = id.Value[1 : len(id.Value)-1]
		}
		return value, nil
	default:
		return "", fmt.Errorf("unexpected expr type %T", node)
	}
}

func eqNeq(node *ast.BinaryExpr, labels map[string]string) (bool, error) {
	lOperand, err := litId(node.X)
	if err != nil {
		return false, err
	}
	rOperand, err := litId(node.Y)
	if err != nil {
		return false, err
	}

	switch node.Op {
	case token.EQL:
		return rOperand == labels[lOperand], nil
	case token.NEQ:
		return rOperand != labels[lOperand], nil
	}
	return false, fmt.Errorf("unexpected op %s,expecting ==,!=", node.Op.String())
}

func match(parsedAst ast.Expr, labels map[string]string) (bool, error) {
	if parsedAst == nil {
		return true, nil
	}
	switch node := parsedAst.(type) {
	case *ast.BinaryExpr:
		if node.Op == token.EQL || node.Op == token.NEQ {
			return eqNeq(node, labels)
		}
		lOperand, err := match(node.X, labels)
		if err != nil {
			return false, err
		}
		if node.Op == token.LAND {
			if !lOperand {
				return false, nil
			}
			rOperand, err := match(node.Y, labels)
			if err != nil {
				return false, err
			}
			return lOperand && rOperand, nil
		}
		if node.Op == token.LOR {
			if lOperand {
				return true, nil
			}
			rOperand, err := match(node.Y, labels)
			if err != nil {
				return false, err
			}
			return lOperand || rOperand, nil
		}
		return false, fmt.Errorf("unexpected op %s,expecting ==,!=,&&,||", node.Op.String())

	case *ast.ParenExpr:
		//括号
		return match(node.X, labels)
	}

	return false, fmt.Errorf("unexpected expr type %T", parsedAst)
}
