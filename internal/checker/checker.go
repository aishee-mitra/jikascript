package checker

import (
	"fmt"

	"microjs/internal/ast"
)

// Check validates the initial MicroJS subset with minimal semantic checks.
func Check(program *ast.Program) error {
	scope := map[string]string{}
	for _, stmt := range program.Statements {
		if err := checkStmt(stmt, scope); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(stmt ast.Stmt, scope map[string]string) error {
	switch node := stmt.(type) {
	case *ast.VarDecl:
		if _, ok := scope[node.Name]; ok {
			return fmt.Errorf("line %d:%d: variable '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		scope[node.Name] = node.Type
	case *ast.ConstDecl:
		if _, ok := scope[node.Name]; ok {
			return fmt.Errorf("line %d:%d: constant '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		scope[node.Name] = node.Type
	case *ast.AssignStmt:
		if _, ok := scope[node.Target]; !ok {
			return fmt.Errorf("line %d:%d: undefined variable '%s'", node.Pos.Line, node.Pos.Column, node.Target)
		}
	case *ast.FunctionDecl:
		for _, param := range node.Params {
			if _, ok := scope[param]; ok {
				return fmt.Errorf("line %d:%d: parameter '%s' already declared", node.Pos.Line, node.Pos.Column, param)
			}
			scope[param] = "param"
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scope); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := checkExpr(node.Cond, scope); err != nil {
			return err
		}
		for _, s := range node.Then {
			if err := checkStmt(s, scope); err != nil {
				return err
			}
		}
		for _, s := range node.Else {
			if err := checkStmt(s, scope); err != nil {
				return err
			}
		}
	case *ast.TaskDecl:
		if _, ok := scope[node.Name]; ok {
			return fmt.Errorf("line %d:%d: task '%s' already declared", node.Pos.Line, node.Pos.Column, node.Name)
		}
		scope[node.Name] = "task"
		if err := checkExpr(node.Interval, scope); err != nil {
			return err
		}
		if node.Callback != nil {
			for _, param := range node.Callback.Params {
				scope[param] = "callback-param"
			}
			for _, s := range node.Callback.Body {
				if err := checkStmt(s, scope); err != nil {
					return err
				}
			}
		}
	case *ast.BlockStmt:
		for _, s := range node.Statements {
			if err := checkStmt(s, scope); err != nil {
				return err
			}
		}
	case *ast.ReturnStmt:
		if node.Value != nil {
			return checkExpr(node.Value, scope)
		}
	case *ast.ExprStmt:
		return checkExpr(node.Expr, scope)
	}
	return nil
}

func checkExpr(expr ast.Expr, scope map[string]string) error {
	switch node := expr.(type) {
	case *ast.Identifier:
		if builtin(node.Name) {
			return nil
		}
		if _, ok := scope[node.Name]; !ok {
			return fmt.Errorf("line %d:%d: undefined identifier '%s'", node.Pos.Line, node.Pos.Column, node.Name)
		}
	case *ast.CallExpr:
		if err := checkExpr(node.Function, scope); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := checkExpr(arg, scope); err != nil {
				return err
			}
		}
	case *ast.SelectorExpr:
		return checkExpr(node.Object, scope)
	case *ast.BinaryExpr:
		if err := checkExpr(node.Left, scope); err != nil {
			return err
		}
		if err := checkExpr(node.Right, scope); err != nil {
			return err
		}
	case *ast.UnaryExpr:
		return checkExpr(node.Expr, scope)
	case *ast.NumberLiteral, *ast.StringLiteral, *ast.BoolLiteral:
		return nil
	case *ast.ArrowFunctionExpr:
		for _, param := range node.Params {
			scope[param] = "arrow-param"
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scope); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
	return nil
}

func builtin(name string) bool {
	switch name {
	case "pin", "scheduler", "start", "output", "input", "toggle", "read", "write":
		return true
	default:
		return false
	}
}
