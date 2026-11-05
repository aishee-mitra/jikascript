package checker

import (
	"fmt"

	"microjs/internal/ast"
)

// Check validates the initial MicroJS subset with minimal semantic checks.
func Check(program *ast.Program) error {
	scopeStack := []map[string]string{{}}
	for _, stmt := range program.Statements {
		if err := checkStmt(stmt, scopeStack); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(stmt ast.Stmt, scopeStack []map[string]string) error {
	current := scopeStack[len(scopeStack)-1]
	switch node := stmt.(type) {
	case *ast.VarDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: variable '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		current[node.Name] = "var"
		if node.Value != nil {
			if err := checkExpr(node.Value, scopeStack); err != nil {
				return err
			}
		}
	case *ast.ConstDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: constant '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		current[node.Name] = "const"
		if node.Value != nil {
			if err := checkExpr(node.Value, scopeStack); err != nil {
				return err
			}
		}
	case *ast.FunctionDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: function '%s' already declared", node.Pos.Line, node.Pos.Column, node.Name)
		}
		current[node.Name] = "function"
		funcScope := map[string]string{}
		for _, param := range node.Params {
			if _, exists := funcScope[param]; exists {
				return fmt.Errorf("line %d:%d: parameter '%s' already declared", node.Pos.Line, node.Pos.Column, param)
			}
			funcScope[param] = "param"
		}
		for _, s := range node.Body {
			if err := checkStmt(s, append(scopeStack, funcScope)); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		for _, s := range node.Then {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
		for _, s := range node.Else {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
	case *ast.WhileStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
	case *ast.DoWhileStmt:
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
	case *ast.ForStmt:
		if node.Init != nil {
			if err := checkStmt(node.Init, scopeStack); err != nil {
				return err
			}
		}
		if node.Cond != nil {
			if err := checkExpr(node.Cond, scopeStack); err != nil {
				return err
			}
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
		if node.Post != nil {
			if err := checkExpr(node.Post, scopeStack); err != nil {
				return err
			}
		}
	case *ast.TaskDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: task '%s' already declared", node.Pos.Line, node.Pos.Column, node.Name)
		}
		current[node.Name] = "task"
		if err := checkExpr(node.Interval, scopeStack); err != nil {
			return err
		}
		if node.Callback != nil {
			callbackScope := map[string]string{}
			for _, param := range node.Callback.Params {
				callbackScope[param] = "callback-param"
			}
			for _, s := range node.Callback.Body {
				if err := checkStmt(s, append(scopeStack, callbackScope)); err != nil {
					return err
				}
			}
		}
	case *ast.BlockStmt:
		for _, s := range node.Statements {
			if err := checkStmt(s, scopeStack); err != nil {
				return err
			}
		}
	case *ast.ReturnStmt:
		if node.Value != nil {
			return checkExpr(node.Value, scopeStack)
		}
	case *ast.ExprStmt:
		return checkExpr(node.Expr, scopeStack)
	case *ast.AssignStmt:
		if _, ok := lookup(scopeStack, node.Target); !ok {
			return fmt.Errorf("line %d:%d: undefined variable '%s'", node.Pos.Line, node.Pos.Column, node.Target)
		}
		if isConstBinding(scopeStack, node.Target) {
			return fmt.Errorf("line %d:%d: cannot assign to constant variable '%s'", node.Pos.Line, node.Pos.Column, node.Target)
		}
		return checkExpr(node.Value, scopeStack)
	}
	return nil
}

func checkExpr(expr ast.Expr, scopeStack []map[string]string) error {
	switch node := expr.(type) {
	case *ast.Identifier:
		if builtin(node.Name) {
			return nil
		}
		if _, ok := lookup(scopeStack, node.Name); !ok {
			return fmt.Errorf("line %d:%d: undefined identifier '%s'", node.Pos.Line, node.Pos.Column, node.Name)
		}
	case *ast.CallExpr:
		if err := checkExpr(node.Function, scopeStack); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := checkExpr(arg, scopeStack); err != nil {
				return err
			}
		}
	case *ast.SelectorExpr:
		return checkExpr(node.Object, scopeStack)
	case *ast.BinaryExpr:
		if err := checkExpr(node.Left, scopeStack); err != nil {
			return err
		}
		if isAssignmentOp(node.Op) {
			if ident, ok := node.Left.(*ast.Identifier); ok {
				if isConstBinding(scopeStack, ident.Name) {
					return fmt.Errorf("line %d:%d: cannot assign to constant variable '%s'", node.Pos.Line, node.Pos.Column, ident.Name)
				}
			}
		}
		if err := checkExpr(node.Right, scopeStack); err != nil {
			return err
		}
	case *ast.UnaryExpr:
		if (node.Op == "++" || node.Op == "--") && isConstBinding(scopeStack, renderUnaryTarget(node.Expr)) {
			return fmt.Errorf("line %d:%d: cannot assign to constant variable '%s'", node.Pos.Line, node.Pos.Column, renderUnaryTarget(node.Expr))
		}
		return checkExpr(node.Expr, scopeStack)
	case *ast.NumberLiteral, *ast.StringLiteral, *ast.BoolLiteral:
		return nil
	case *ast.ArrowFunctionExpr:
		callbackScope := map[string]string{}
		for _, param := range node.Params {
			callbackScope[param] = "arrow-param"
		}
		for _, s := range node.Body {
			if err := checkStmt(s, append(scopeStack, callbackScope)); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
	return nil
}

func lookup(scopeStack []map[string]string, name string) (string, bool) {
	for i := len(scopeStack) - 1; i >= 0; i-- {
		if val, ok := scopeStack[i][name]; ok {
			return val, true
		}
	}
	return "", false
}

func isConstBinding(scopeStack []map[string]string, name string) bool {
	for i := len(scopeStack) - 1; i >= 0; i-- {
		if val, ok := scopeStack[i][name]; ok {
			return val == "const"
		}
	}
	return false
}

func isAssignmentOp(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=":
		return true
	default:
		return false
	}
}

func renderUnaryTarget(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Identifier); ok {
		return ident.Name
	}
	return ""
}

func builtin(name string) bool {
	switch name {
	case "pin", "scheduler", "start", "output", "input", "toggle", "read", "write":
		return true
	default:
		return false
	}
}
