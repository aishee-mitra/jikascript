package checker

import (
	"fmt"
	"strconv"
	"strings"

	"jikascript/internal/ast"
	"jikascript/targets/ch32fun"
)

// Check validates the initial JikaScript subset with minimal semantic checks.
func Check(program *ast.Program) error {
	scopeStack := []map[string]string{{}}
	taskNames := map[string]struct{}{}
	for _, stmt := range program.Statements {
		if err := checkStmt(stmt, scopeStack, taskNames, 0, 0); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(stmt ast.Stmt, scopeStack []map[string]string, taskNames map[string]struct{}, loopDepth int, switchDepth int) error {
	current := scopeStack[len(scopeStack)-1]
	switch node := stmt.(type) {
	case *ast.VarDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: variable '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		bindingKind, err := classifyBinding("var", node.Type, node.ArrayCapacity, node.Value)
		if err != nil {
			return fmt.Errorf("line %d:%d: %w", node.Pos.Line, node.Pos.Column, err)
		}
		current[node.Name] = bindingKind
		if node.Value != nil {
			if err := checkExpr(node.Value, scopeStack); err != nil {
				return err
			}
		}
	case *ast.ConstDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: constant '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		bindingKind, err := classifyBinding("const", node.Type, node.ArrayCapacity, node.Value)
		if err != nil {
			return fmt.Errorf("line %d:%d: %w", node.Pos.Line, node.Pos.Column, err)
		}
		current[node.Name] = bindingKind
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
			if err := checkStmt(s, append(scopeStack, funcScope), taskNames, loopDepth, switchDepth); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		for _, s := range node.Then {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth, switchDepth); err != nil {
				return err
			}
		}
		for _, s := range node.Else {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth, switchDepth); err != nil {
				return err
			}
		}
	case *ast.WhileStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth+1, switchDepth); err != nil {
				return err
			}
		}
	case *ast.DoWhileStmt:
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth+1, switchDepth); err != nil {
				return err
			}
		}
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
	case *ast.ForStmt:
		if node.Init != nil {
			if err := checkStmt(node.Init, scopeStack, taskNames, loopDepth+1, switchDepth); err != nil {
				return err
			}
		}
		if node.Cond != nil {
			if err := checkExpr(node.Cond, scopeStack); err != nil {
				return err
			}
		}
		for _, s := range node.Body {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth+1, switchDepth); err != nil {
				return err
			}
		}
		if node.Post != nil {
			if err := checkExpr(node.Post, scopeStack); err != nil {
				return err
			}
		}
	case *ast.TaskDecl:
		if _, ok := taskNames[node.Name]; ok {
			return fmt.Errorf("line %d:%d: duplicate task name '%s'", node.Pos.Line, node.Pos.Column, node.Name)
		}
		taskNames[node.Name] = struct{}{}
		if err := checkExpr(node.Interval, scopeStack); err != nil {
			return err
		}
		if node.Callback != nil {
			callbackScope := map[string]string{}
			for _, param := range node.Callback.Params {
				callbackScope[param] = "callback-param"
			}
			for _, s := range node.Callback.Body {
				if err := checkStmt(s, append(scopeStack, callbackScope), taskNames, loopDepth, switchDepth); err != nil {
					return err
				}
			}
		}
	case *ast.SwitchStmt:
		if err := checkExpr(node.Expr, scopeStack); err != nil {
			return err
		}
		for _, c := range node.Cases {
			if c.Value != nil {
				if err := checkExpr(c.Value, scopeStack); err != nil {
					return err
				}
			}
			for _, s := range c.Body {
				if err := checkStmt(s, scopeStack, taskNames, loopDepth, switchDepth+1); err != nil {
					return err
				}
			}
		}
	case *ast.BreakStmt:
		if loopDepth == 0 && switchDepth == 0 {
			return fmt.Errorf("line %d:%d: break is only valid inside loops or switch", node.Pos.Line, node.Pos.Column)
		}
	case *ast.ContinueStmt:
		if loopDepth == 0 {
			return fmt.Errorf("line %d:%d: continue is only valid inside loops", node.Pos.Line, node.Pos.Column)
		}
	case *ast.BlockStmt:
		for _, s := range node.Statements {
			if err := checkStmt(s, scopeStack, taskNames, loopDepth, switchDepth); err != nil {
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
		if sel, ok := node.Function.(*ast.SelectorExpr); ok {
			if ident, ok := sel.Object.(*ast.Identifier); ok {
				if binding, exists := lookup(scopeStack, ident.Name); exists && isArrayBinding(binding) {
					switch sel.Field {
					case "push":
						if len(node.Args) != 1 {
							return fmt.Errorf("line %d:%d: push() expects one argument", node.Pos.Line, node.Pos.Column)
						}
						if err := checkExpr(node.Args[0], scopeStack); err != nil {
							return err
						}
						argType, ok := inferLiteralType(node.Args[0], scopeStack)
						if ok && !typeCompatible(arrayElemType(binding), argType) {
							return fmt.Errorf("line %d:%d: push() type mismatch for array '%s'", node.Pos.Line, node.Pos.Column, ident.Name)
						}
						return nil
					case "pop":
						if len(node.Args) != 0 {
							return fmt.Errorf("line %d:%d: pop() expects no arguments", node.Pos.Line, node.Pos.Column)
						}
						return nil
					default:
						return fmt.Errorf("line %d:%d: unsupported array method '%s'", node.Pos.Line, node.Pos.Column, sel.Field)
					}
				}
			}
		}
		if err := checkExpr(node.Function, scopeStack); err != nil {
			return err
		}
		for _, arg := range node.Args {
			if err := checkExpr(arg, scopeStack); err != nil {
				return err
			}
		}
	case *ast.SelectorExpr:
		if ident, ok := node.Object.(*ast.Identifier); ok {
			if kind, exists := lookup(scopeStack, ident.Name); exists {
				if isPinBinding(kind) {
					if !ch32fun.IsValidPinMethod(node.Field) {
						return fmt.Errorf("line %d:%d: unsupported pin method '%s'", node.Pos.Line, node.Pos.Column, node.Field)
					}
				}
				if isArrayBinding(kind) {
					if node.Field != "length" {
						return fmt.Errorf("line %d:%d: unsupported array property '%s'", node.Pos.Line, node.Pos.Column, node.Field)
					}
				}
			}
		}
		return checkExpr(node.Object, scopeStack)
	case *ast.IndexExpr:
		ident, ok := node.Object.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("line %d:%d: array indexing requires an identifier target", node.Pos.Line, node.Pos.Column)
		}
		kind, exists := lookup(scopeStack, ident.Name)
		if !exists || !isArrayBinding(kind) {
			return fmt.Errorf("line %d:%d: '%s' is not an array", node.Pos.Line, node.Pos.Column, ident.Name)
		}
		if err := checkExpr(node.Index, scopeStack); err != nil {
			return err
		}
		if idx, ok := indexLiteral(node.Index); ok {
			if idx < 0 || idx >= arrayCapacity(kind) {
				return fmt.Errorf("line %d:%d: array index %d out of bounds (capacity %d)", node.Pos.Line, node.Pos.Column, idx, arrayCapacity(kind))
			}
		}
		return nil
	case *ast.BinaryExpr:
		if err := checkExpr(node.Left, scopeStack); err != nil {
			return err
		}
		if isAssignmentOp(node.Op) {
			if idxExpr, ok := node.Left.(*ast.IndexExpr); ok {
				ident, ok := idxExpr.Object.(*ast.Identifier)
				if !ok {
					return fmt.Errorf("line %d:%d: indexed assignment requires identifier target", node.Pos.Line, node.Pos.Column)
				}
				binding, exists := lookup(scopeStack, ident.Name)
				if !exists || !isArrayBinding(binding) {
					return fmt.Errorf("line %d:%d: '%s' is not an array", node.Pos.Line, node.Pos.Column, ident.Name)
				}
				if idx, ok := indexLiteral(idxExpr.Index); ok {
					if idx < 0 || idx >= arrayCapacity(binding) {
						return fmt.Errorf("line %d:%d: array index %d out of bounds (capacity %d)", node.Pos.Line, node.Pos.Column, idx, arrayCapacity(binding))
					}
				}
				valueType, ok := inferLiteralType(node.Right, scopeStack)
				if ok && !typeCompatible(arrayElemType(binding), valueType) {
					return fmt.Errorf("line %d:%d: type mismatch for array '%s' assignment", node.Pos.Line, node.Pos.Column, ident.Name)
				}
			}
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
	case *ast.ArrayLiteralExpr:
		if len(node.Elements) == 0 {
			return nil
		}
		firstType, ok := inferLiteralType(node.Elements[0], scopeStack)
		if !ok {
			return fmt.Errorf("line %d:%d: unsupported array element type", node.Pos.Line, node.Pos.Column)
		}
		for _, elem := range node.Elements[1:] {
			if err := checkExpr(elem, scopeStack); err != nil {
				return err
			}
			elemType, ok := inferLiteralType(elem, scopeStack)
			if !ok || !typeCompatible(firstType, elemType) {
				return fmt.Errorf("line %d:%d: array literal elements must be homogeneous", node.Pos.Line, node.Pos.Column)
			}
		}
		return nil
	case *ast.ArrowFunctionExpr:
		callbackScope := map[string]string{}
		for _, param := range node.Params {
			callbackScope[param] = "arrow-param"
		}
		for _, s := range node.Body {
			if err := checkStmt(s, append(scopeStack, callbackScope), map[string]struct{}{}, 0, 0); err != nil {
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
			return val == "const" || strings.HasPrefix(val, "array-const-")
		}
	}
	return false
}

func isAssignmentOp(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=":
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

func classifyBinding(baseKind string, declaredType string, declaredCapacity int, value ast.Expr) (string, error) {
	if declaredCapacity > 0 {
		elemType := declaredType
		if elemType == "" {
			return "", fmt.Errorf("array declarations with explicit capacity require an element type annotation")
		}
		lit, ok := value.(*ast.ArrayLiteralExpr)
		if !ok {
			return "", fmt.Errorf("array declarations currently require an array literal initializer")
		}
		if len(lit.Elements) > declaredCapacity {
			return "", fmt.Errorf("array initializer length %d exceeds declared capacity %d", len(lit.Elements), declaredCapacity)
		}
		if len(lit.Elements) == 0 {
			return encodeArrayBinding(baseKind, elemType, declaredCapacity), nil
		}
		for _, elem := range lit.Elements {
			et, ok := inferLiteralType(elem, nil)
			if !ok || !typeCompatible(elemType, et) {
				return "", fmt.Errorf("array element type mismatch for declared type '%s'", elemType)
			}
		}
		return encodeArrayBinding(baseKind, elemType, declaredCapacity), nil
	}

	if lit, ok := value.(*ast.ArrayLiteralExpr); ok {
		if len(lit.Elements) == 0 {
			return "", fmt.Errorf("empty array literal requires explicit type and capacity, e.g. let a: int32[8] = []")
		}
		elemType, ok := inferLiteralType(lit.Elements[0], nil)
		if !ok {
			return "", fmt.Errorf("unsupported array element type")
		}
		for _, elem := range lit.Elements[1:] {
			et, ok := inferLiteralType(elem, nil)
			if !ok || !typeCompatible(elemType, et) {
				return "", fmt.Errorf("array literal elements must be homogeneous")
			}
		}
		return encodeArrayBinding(baseKind, elemType, len(lit.Elements)), nil
	}

	if call, ok := value.(*ast.CallExpr); ok {
		if fn, ok := call.Function.(*ast.Identifier); ok && fn.Name == "pin" {
			if len(call.Args) != 1 {
				return "", fmt.Errorf("pin() expects exactly one argument")
			}
			pinName, ok := call.Args[0].(*ast.StringLiteral)
			if !ok {
				return "", fmt.Errorf("pin() argument must be a string literal")
			}
			if !ch32fun.IsValidPin(pinName.Value) {
				return "", fmt.Errorf("unknown hardware pin %q", pinName.Value)
			}
			return "pin-" + baseKind, nil
		}
	}
	return baseKind, nil
}

func isPinBinding(kind string) bool {
	return strings.HasPrefix(kind, "pin-")
}

func encodeArrayBinding(baseKind, elemType string, capacity int) string {
	return fmt.Sprintf("array-%s-%s-%d", baseKind, elemType, capacity)
}

func isArrayBinding(kind string) bool {
	return strings.HasPrefix(kind, "array-")
}

func arrayElemType(kind string) string {
	parts := strings.Split(kind, "-")
	if len(parts) < 4 {
		return "number"
	}
	return parts[2]
}

func arrayCapacity(kind string) int {
	parts := strings.Split(kind, "-")
	if len(parts) < 4 {
		return 0
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0
	}
	return n
}

func indexLiteral(expr ast.Expr) (int, bool) {
	n, ok := expr.(*ast.NumberLiteral)
	if !ok {
		return 0, false
	}
	if strings.Contains(n.Value, ".") {
		return 0, false
	}
	v, err := strconv.Atoi(n.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

func inferLiteralType(expr ast.Expr, scopeStack []map[string]string) (string, bool) {
	switch node := expr.(type) {
	case *ast.NumberLiteral:
		if strings.Contains(node.Value, ".") {
			return "float32", true
		}
		return "int32", true
	case *ast.StringLiteral:
		return "string", true
	case *ast.BoolLiteral:
		return "boolean", true
	case *ast.Identifier:
		if scopeStack == nil {
			return "", false
		}
		binding, ok := lookup(scopeStack, node.Name)
		if !ok {
			return "", false
		}
		if isArrayBinding(binding) {
			return arrayElemType(binding), true
		}
		switch binding {
		case "var", "const":
			return "int32", true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func typeCompatible(expected string, actual string) bool {
	if expected == actual {
		return true
	}
	if isNumericType(expected) && isNumericType(actual) {
		return true
	}
	return false
}

func isNumericType(t string) bool {
	switch t {
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "float32", "number":
		return true
	default:
		return false
	}
}

func builtin(name string) bool {
	switch name {
	case "pin", "scheduler", "start", "output", "input", "toggle", "read", "write":
		return true
	default:
		return false
	}
}
