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
	functionReturns := inferFunctionObjectReturns(program)
	for _, stmt := range program.Statements {
		if err := checkStmt(stmt, scopeStack, taskNames, 0, 0, functionReturns); err != nil {
			return err
		}
	}
	return nil
}

// pushScope appends a fresh lexical scope. It copies the existing stack so
// callers never mutate a backing array shared with an outer frame.
func pushScope(scopeStack []map[string]string) []map[string]string {
	stack := make([]map[string]string, 0, len(scopeStack)+1)
	stack = append(stack, scopeStack...)
	return append(stack, map[string]string{})
}

// checkBlock checks each statement of a block body inside a new lexical scope.
func checkBlock(body []ast.Stmt, scopeStack []map[string]string, taskNames map[string]struct{}, loopDepth int, switchDepth int, functionReturns map[string]string) error {
	inner := pushScope(scopeStack)
	for _, s := range body {
		if err := checkStmt(s, inner, taskNames, loopDepth, switchDepth, functionReturns); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(stmt ast.Stmt, scopeStack []map[string]string, taskNames map[string]struct{}, loopDepth int, switchDepth int, functionReturns map[string]string) error {
	current := scopeStack[len(scopeStack)-1]
	switch node := stmt.(type) {
	case *ast.VarDecl:
		if _, ok := current[node.Name]; ok {
			return fmt.Errorf("line %d:%d: variable '%s' already declared in this scope", node.Pos.Line, node.Pos.Column, node.Name)
		}
		bindingKind, err := classifyBinding("var", node.Type, node.ArrayCapacity, node.Value, scopeStack, functionReturns)
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
		bindingKind, err := classifyBinding("const", node.Type, node.ArrayCapacity, node.Value, scopeStack, functionReturns)
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
		stack := make([]map[string]string, 0, len(scopeStack)+1)
		stack = append(stack, scopeStack...)
		stack = append(stack, funcScope)
		for _, s := range node.Body {
			if err := checkStmt(s, stack, taskNames, loopDepth, switchDepth, functionReturns); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		if err := checkBlock(node.Then, scopeStack, taskNames, loopDepth, switchDepth, functionReturns); err != nil {
			return err
		}
		if err := checkBlock(node.Else, scopeStack, taskNames, loopDepth, switchDepth, functionReturns); err != nil {
			return err
		}
	case *ast.WhileStmt:
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
		if err := checkBlock(node.Body, scopeStack, taskNames, loopDepth+1, switchDepth, functionReturns); err != nil {
			return err
		}
	case *ast.DoWhileStmt:
		if err := checkBlock(node.Body, scopeStack, taskNames, loopDepth+1, switchDepth, functionReturns); err != nil {
			return err
		}
		if err := checkExpr(node.Cond, scopeStack); err != nil {
			return err
		}
	case *ast.ForStmt:
		inner := pushScope(scopeStack)
		if node.Init != nil {
			if err := checkStmt(node.Init, inner, taskNames, loopDepth+1, switchDepth, functionReturns); err != nil {
				return err
			}
		}
		if node.Cond != nil {
			if err := checkExpr(node.Cond, inner); err != nil {
				return err
			}
		}
		for _, s := range node.Body {
			if err := checkStmt(s, inner, taskNames, loopDepth+1, switchDepth, functionReturns); err != nil {
				return err
			}
		}
		if node.Post != nil {
			if err := checkExpr(node.Post, inner); err != nil {
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
			stack := make([]map[string]string, 0, len(scopeStack)+1)
			stack = append(stack, scopeStack...)
			stack = append(stack, callbackScope)
			for _, s := range node.Callback.Body {
				if err := checkStmt(s, stack, taskNames, loopDepth, switchDepth, functionReturns); err != nil {
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
			if err := checkBlock(c.Body, scopeStack, taskNames, loopDepth, switchDepth+1, functionReturns); err != nil {
				return err
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
		return checkBlock(node.Statements, scopeStack, taskNames, loopDepth, switchDepth, functionReturns)
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
				if isObjectBinding(kind) {
					if _, ok := objectPropertyType(kind, node.Field); !ok {
						return fmt.Errorf("line %d:%d: unknown property '%s' on object '%s'", node.Pos.Line, node.Pos.Column, node.Field, ident.Name)
					}
				}
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
			if sel, ok := node.Left.(*ast.SelectorExpr); ok {
				ident, ok := sel.Object.(*ast.Identifier)
				if !ok {
					return fmt.Errorf("line %d:%d: property assignment requires an object identifier", node.Pos.Line, node.Pos.Column)
				}
				binding, exists := lookup(scopeStack, ident.Name)
				if !exists || !isObjectBinding(binding) {
					return fmt.Errorf("line %d:%d: '%s' is not an object", node.Pos.Line, node.Pos.Column, ident.Name)
				}
				propertyType, exists := objectPropertyType(binding, sel.Field)
				if !exists {
					return fmt.Errorf("line %d:%d: unknown property '%s' on object '%s'", node.Pos.Line, node.Pos.Column, sel.Field, ident.Name)
				}
				valueType, known := inferLiteralType(node.Right, scopeStack)
				if known && !typeCompatible(propertyType, valueType) {
					return fmt.Errorf("line %d:%d: type mismatch assigning property '%s' of object '%s': expected %s, got %s", node.Pos.Line, node.Pos.Column, sel.Field, ident.Name, propertyType, valueType)
				}
			}
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
	case *ast.ObjectLiteralExpr:
		seen := map[string]struct{}{}
		for _, property := range node.Properties {
			if _, exists := seen[property.Name]; exists {
				return fmt.Errorf("line %d:%d: duplicate object property '%s'", property.Pos.Line, property.Pos.Column, property.Name)
			}
			seen[property.Name] = struct{}{}
			if err := checkExpr(property.Value, scopeStack); err != nil {
				return err
			}
			if _, ok := inferLiteralType(property.Value, scopeStack); !ok {
				return fmt.Errorf("line %d:%d: unsupported value for object property '%s'", property.Pos.Line, property.Pos.Column, property.Name)
			}
		}
		return nil
	case *ast.ArrowFunctionExpr:
		callbackScope := map[string]string{}
		for _, param := range node.Params {
			callbackScope[param] = "arrow-param"
		}
		stack := make([]map[string]string, 0, len(scopeStack)+1)
		stack = append(stack, scopeStack...)
		stack = append(stack, callbackScope)
		for _, s := range node.Body {
			if err := checkStmt(s, stack, map[string]struct{}{}, 0, 0, map[string]string{}); err != nil {
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

func classifyBinding(baseKind string, declaredType string, declaredCapacity int, value ast.Expr, scopeStack []map[string]string, functionReturns map[string]string) (string, error) {
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

	if lit, ok := value.(*ast.ObjectLiteralExpr); ok {
		if declaredType != "" {
			return "", fmt.Errorf("object declarations use inferred structural types and do not support type annotations")
		}
		return encodeObjectBinding(baseKind, objectTypeFromLiteral(lit, scopeStack)), nil
	}

	if call, ok := value.(*ast.CallExpr); ok {
		if fn, ok := call.Function.(*ast.Identifier); ok {
			if shape, ok := functionReturns[fn.Name]; ok {
				return encodeObjectBinding(baseKind, shape), nil
			}
		}
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

func inferFunctionObjectReturns(program *ast.Program) map[string]string {
	returns := map[string]string{}
	for _, stmt := range program.Statements {
		fn, ok := stmt.(*ast.FunctionDecl)
		if !ok {
			continue
		}
		for _, bodyStmt := range fn.Body {
			if ret, ok := bodyStmt.(*ast.ReturnStmt); ok {
				if literal, ok := ret.Value.(*ast.ObjectLiteralExpr); ok {
					returns[fn.Name] = objectTypeFromLiteral(literal, nil)
				}
			}
		}
	}
	return returns
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

func encodeObjectBinding(baseKind, shape string) string {
	return "object-" + baseKind + "-" + shape
}

func isObjectBinding(kind string) bool { return strings.HasPrefix(kind, "object-") }

func objectShape(kind string) string {
	parts := strings.SplitN(kind, "-", 3)
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

func objectPropertyType(kind, name string) (string, bool) {
	for _, property := range strings.Split(objectShape(kind), ",") {
		parts := strings.SplitN(property, ":", 2)
		if len(parts) == 2 && parts[0] == name {
			return parts[1], true
		}
	}
	return "", false
}

func objectTypeFromLiteral(lit *ast.ObjectLiteralExpr, scopeStack []map[string]string) string {
	properties := make([]string, 0, len(lit.Properties))
	for _, property := range lit.Properties {
		t, ok := inferLiteralType(property.Value, scopeStack)
		if !ok {
			t = "invalid"
		}
		properties = append(properties, property.Name+":"+t)
	}
	return strings.Join(properties, ",")
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
		if isObjectBinding(binding) {
			return objectShape(binding), true
		}
		switch binding {
		case "var", "const":
			return "int32", true
		default:
			return "", false
		}
	case *ast.SelectorExpr:
		if ident, ok := node.Object.(*ast.Identifier); ok && scopeStack != nil {
			if binding, exists := lookup(scopeStack, ident.Name); exists && isObjectBinding(binding) {
				return objectPropertyType(binding, node.Field)
			}
		}
		return "", false
	case *ast.ObjectLiteralExpr:
		return objectTypeFromLiteral(node, scopeStack), true
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
	if strings.Contains(expected, ":") && strings.Contains(actual, ":") {
		return expected == actual
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
	case "pin", "scheduler", "HIGH", "LOW":
		return true
	default:
		return false
	}
}
