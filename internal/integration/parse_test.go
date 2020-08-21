package integration

import (
	"fmt"
	"go/ast"
	"go/parser"
	"log"
	"testing"
)

var cases = []map[string]string{{"ver": "1.1", "time": "morning"},
	{"ver": "1.2", "time": "afternoon"},
	{"ver": "1.3", "time": "night"},
	{"ver": "1.3", "time": "morning"}}

func matchExpr(expr string, label map[string]string) (bool, error) {
	var parsedAst ast.Expr
	var err error
	if expr != "" {
		parsedAst, err = parser.ParseExpr(expr)
		if err != nil {
			return false, err
		}
	}
	matched, err := match(parsedAst, label)
	if err != nil {
		return false, err
	}
	if matched {
		log.Printf("expr %s matched case %+v\n", expr, label)
		return true, nil
	}
	return false, nil
}

func TestSimpleEqNeq(t *testing.T) {
	expr := `ver==1.3`
	case1Matched, _ := matchExpr(expr, cases[0])
	if case1Matched {
		t.Errorf("expr %s should not match %+v", expr, cases[0])
	}

	case3Matched, _ := matchExpr(expr, cases[2])
	if !case3Matched {
		t.Errorf("expr %s should match %+v", expr, cases[2])
	}

	expr = `time!=night`
	case2Matched, _ := matchExpr(expr, cases[1])
	if !case2Matched {
		t.Errorf("expr %s should match %+v", expr, cases[1])
	}

	case3Matched, _ = matchExpr(expr, cases[2])
	if case3Matched {
		t.Errorf("expr %s should not match %+v", expr, cases[2])
	}
}

func TestComplex(t *testing.T) {
	expr := `ver==1.1||ver==1.3`
	case1Matched, _ := matchExpr(expr, cases[0])
	if !case1Matched {
		t.Errorf("expr %s should match %+v", expr, cases[0])
	}

	case3Matched, _ := matchExpr(expr, cases[2])
	if !case3Matched {
		t.Errorf("expr %s should match %+v", expr, cases[2])
	}

	expr = `ver==1.3&&(time==night||time==afternoon)`
	case3Matched, _ = matchExpr(expr, cases[2])
	if !case3Matched {
		t.Errorf("expr %s should match %+v", expr, cases[2])
	}

	case4Matched, _ := matchExpr(expr, cases[3])
	if case4Matched {
		t.Errorf("expr %s should not match %+v", expr, cases[3])
	}
}

func TestSyntaxErr(t *testing.T) {
	expr := `ver>=1.1`
	_, err := matchExpr(expr, cases[0])
	if err == nil {
		t.Errorf("syntax should be wrong: %s", expr)
	}
	fmt.Printf("expr %s: %s\n", expr, err.Error())
}
