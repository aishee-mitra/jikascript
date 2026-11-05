package checker

import (
	"fmt"
	"strconv"
	"strings"

	"jikascript/internal/ast"
	"jikascript/targets/ch32fun"
)

type functionTypeInfo struct {
	ParamTypes []string
	ReturnType string
}

// Check validates the initial JikaScript subset with minimal semantic checks.
func Check(program *ast.Program) error {
	functionTypes, err := resolveFunctionTypes(program)
	if err != nil {
		return err
	}
	if err := validateFunctionReturnConsistency(program); err != nil {
		return err
	}
	scopeStack := []map[string]string{{}}
	taskNames := map[string]struct{}{}
	functionReturns := map[string]string{}
	functionArity := inferFunctionArity(program)
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDecl); ok {
			info := functionTypes[fn.Name]
			fn.ParamTypes = append([]string(nil), info.ParamTypes...)
			fn.ReturnType = info.ReturnType
			functionReturns[fn.Name] = info.ReturnType
			scopeStack[0][fn.Name] = encodeFunctionBinding(info.ReturnType, info.ParamTypes...)
		}
	}
	for _, stmt := range program.Statements {
		if err := checkStmt(stmt, scopeStack, taskNames, 0, 0, functionReturns, functionArity); err != nil {
			return err
		}
	}
	return nil
}

func resolveFunctionTypes(program *ast.Program) (map[string]functionTypeInfo, error) {
	infos := map[string]functionTypeInfo{}
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDecl); ok {
			infos[fn.Name] = functionTypeInfo{ParamTypes: make([]string, len(fn.Params))}
		}
	}
	for pass := 0; pass < len(infos)+2; pass++ {
		scope := topLevelBindings(program, infos)
		changed := false
		var resolveErr error
		for _, stmt := range program.Statements {
			walkStatementExpressions(stmt, func(expr ast.Expr) {
				call, ok := expr.(*ast.CallExpr)
				if !ok {
					return
				}
				ident, ok := call.Function.(*ast.Identifier)
				if !ok {
					return
				}
				info, exists := infos[ident.Name]
				if !exists {
					return
				}
				for i, arg := range call.Args {
					if i >= len(info.ParamTypes) {
						continue
					}
					if typ, ok := inferLiteralType(arg, []map[string]string{scope}); ok {
						if info.ParamTypes[i] == "" {
							info.ParamTypes[i] = typ
							changed = true
						} else if !typeCompatible(info.ParamTypes[i], typ) {
							resolveErr = fmt.Errorf("line %d:%d: incompatible argument types for function '%s': parameter %d receives both %s and %s", call.Pos.Line, call.Pos.Column, ident.Name, i+1, info.ParamTypes[i], typ)
						}
					}
				}
				infos[ident.Name] = info
			})
		}
		if resolveErr != nil {
			return nil, resolveErr
		}
		for _, stmt := range program.Statements {
			fn, ok := stmt.(*ast.FunctionDecl)
			if !ok {
				continue
			}
			info := infos[fn.Name]
			functionScope := map[string]string{}
			for i, param := range fn.Params {
				if i < len(info.ParamTypes) && info.ParamTypes[i] != "" {
					functionScope[param] = bindingForType(info.ParamTypes[i])
				}
			}
			for i, param := range fn.Params {
				if info.ParamTypes[i] == "" {
					if typ := inferParameterUsage(fn.Body, param); typ != "" {
						info.ParamTypes[i] = typ
						functionScope[param] = bindingForType(typ)
						changed = true
					}
				}
			}
			stack := []map[string]string{scope, functionScope}
			for _, ret := range collectReturns(fn.Body) {
				if ret.Value == nil {
					if info.ReturnType == "" {
						info.ReturnType = "void"
						changed = true
					}
					continue
				}
				typ, known := inferLiteralType(ret.Value, stack)
				if !known {
					if binary, ok := ret.Value.(*ast.BinaryExpr); ok && isNumericOperator(binary.Op) {
						left, leftOK := inferLiteralType(binary.Left, stack)
						right, rightOK := inferLiteralType(binary.Right, stack)
						if leftOK && rightOK && (!isNumericType(left) || !isNumericType(right)) {
							return nil, fmt.Errorf("line %d:%d: operator '%s' requires numeric operands", binary.Pos.Line, binary.Pos.Column, binary.Op)
						}
					}
					continue
				}
				if info.ReturnType == "" {
					info.ReturnType = typ
					changed = true
				} else if !returnTypeCompatible(info.ReturnType, typ) {
					return nil, fmt.Errorf("line %d:%d: inconsistent return type in function '%s': expected %s, got %s", ret.Pos.Line, ret.Pos.Column, fn.Name, info.ReturnType, typ)
				} else if isNumericType(info.ReturnType) && isNumericType(typ) {
					merged := mergeNumericType(info.ReturnType, typ)
					if merged != info.ReturnType {
						info.ReturnType = merged
						changed = true
					}
				}
			}
			infos[fn.Name] = info
		}
		if !changed {
			break
		}
	}
	for _, stmt := range program.Statements {
		fn, ok := stmt.(*ast.FunctionDecl)
		if !ok {
			continue
		}
		info := infos[fn.Name]
		for i, typ := range info.ParamTypes {
			if typ == "" {
				return nil, fmt.Errorf("line %d:%d: cannot resolve type of parameter '%s' in function '%s'", fn.Pos.Line, fn.Pos.Column, fn.Params[i], fn.Name)
			}
		}
		if info.ReturnType == "" {
			return nil, fmt.Errorf("line %d:%d: cannot resolve return type of function '%s'", fn.Pos.Line, fn.Pos.Column, fn.Name)
		}
	}
	return infos, nil
}

func topLevelBindings(program *ast.Program, infos map[string]functionTypeInfo) map[string]string {
	scope := map[string]string{}
	for name, info := range infos {
		scope[name] = encodeFunctionBinding(info.ReturnType, info.ParamTypes...)
	}
	for _, stmt := range program.Statements {
		var name, declared string
		var value ast.Expr
		switch node := stmt.(type) {
		case *ast.VarDecl:
			name, declared, value = node.Name, node.Type, node.Value
		case *ast.ConstDecl:
			name, declared, value = node.Name, node.Type, node.Value
		default:
			continue
		}
		if declared == "" {
			if typ, ok := inferLiteralType(value, []map[string]string{scope}); ok && !strings.Contains(typ, ":") {
				declared = typ
			}
		}
		if declared == "" {
			if _, ok := value.(*ast.ObjectLiteralExpr); ok {
				declared = objectTypeFromLiteral(value.(*ast.ObjectLiteralExpr), []map[string]string{scope})
			}
		}
		if declared != "" {
			if strings.Contains(declared, ":") {
				scope[name] = encodeObjectBinding("var", declared)
			} else {
				scope[name] = bindingForType(declared)
			}
		}
	}
	return scope
}

func bindingForType(typ string) string {
	if strings.Contains(typ, ":") {
		return encodeObjectBinding("param", typ)
	}
	return encodeScalarBinding("param", typ)
}

func encodeFunctionBinding(returnType string, paramTypes ...string) string {
	return "function-" + returnType + "|" + strings.Join(paramTypes, ",")
}

func functionReturnType(binding string) (string, bool) {
	if !strings.HasPrefix(binding, "function-") {
		return "", false
	}
	value := strings.TrimPrefix(binding, "function-")
	if separator := strings.IndexByte(value, '|'); separator >= 0 {
		value = value[:separator]
	}
	return value, true
}

func functionParamTypes(binding string) ([]string, bool) {
	if !strings.HasPrefix(binding, "function-") {
		return nil, false
	}
	value := strings.TrimPrefix(binding, "function-")
	separator := strings.IndexByte(value, '|')
	if separator < 0 || separator == len(value)-1 {
		return []string{}, true
	}
	return strings.Split(value[separator+1:], ","), true
}

func inferParameterUsage(stmts []ast.Stmt, name string) string {
	var result string
	var visitExpr func(ast.Expr, string)
	visitExpr = func(expr ast.Expr, context string) {
		switch node := expr.(type) {
		case *ast.Identifier:
			if node.Name == name && result == "" {
				if context == "condition" || context == "logical" {
					result = "boolean"
				} else if context == "numeric" {
					result = "int32"
				}
			}
		case *ast.BinaryExpr:
			if node.Op == "&&" || node.Op == "||" || isEqualityOp(node.Op) {
				visitExpr(node.Left, "logical")
				visitExpr(node.Right, "logical")
			} else {
				visitExpr(node.Left, "numeric")
				visitExpr(node.Right, "numeric")
			}
		case *ast.UnaryExpr:
			visitExpr(node.Expr, "numeric")
		case *ast.CallExpr:
			for _, arg := range node.Args {
				visitExpr(arg, "")
			}
		case *ast.SelectorExpr:
			visitExpr(node.Object, "")
		}
	}
	var visitStmt func(ast.Stmt)
	visitStmt = func(stmt ast.Stmt) {
		switch node := stmt.(type) {
		case *ast.IfStmt:
			visitExpr(node.Cond, "condition")
			for _, s := range node.Then {
				visitStmt(s)
			}
			for _, s := range node.Else {
				visitStmt(s)
			}
		case *ast.WhileStmt:
			visitExpr(node.Cond, "condition")
			for _, s := range node.Body {
				visitStmt(s)
			}
		case *ast.DoWhileStmt:
			for _, s := range node.Body {
				visitStmt(s)
			}
			visitExpr(node.Cond, "condition")
		case *ast.ForStmt:
			if node.Init != nil {
				visitStmt(node.Init)
			}
			visitExpr(node.Cond, "condition")
			visitExpr(node.Post, "")
			for _, s := range node.Body {
				visitStmt(s)
			}
		case *ast.SwitchStmt:
			visitExpr(node.Expr, "")
			for _, c := range node.Cases {
				for _, s := range c.Body {
					visitStmt(s)
				}
			}
		case *ast.ReturnStmt:
			visitExpr(node.Value, "")
		case *ast.ExprStmt:
			visitExpr(node.Expr, "")
		case *ast.VarDecl:
			visitExpr(node.Value, "")
		case *ast.ConstDecl:
			visitExpr(node.Value, "")
		case *ast.AssignStmt:
			visitExpr(node.Value, "")
		case *ast.BlockStmt:
			for _, s := range node.Statements {
				visitStmt(s)
			}
		}
	}
	for _, stmt := range stmts {
		visitStmt(stmt)
	}
	return result
}

func walkStatementExpressions(stmt ast.Stmt, visit func(ast.Expr)) {
	var walkStmt func(ast.Stmt)
	var walkExpr func(ast.Expr)
	walkExpr = func(expr ast.Expr) {
		if expr == nil {
			return
		}
		visit(expr)
		switch node := expr.(type) {
		case *ast.CallExpr:
			walkExpr(node.Function)
			for _, arg := range node.Args {
				walkExpr(arg)
			}
		case *ast.SelectorExpr:
			walkExpr(node.Object)
		case *ast.IndexExpr:
			walkExpr(node.Object)
			walkExpr(node.Index)
		case *ast.BinaryExpr:
			walkExpr(node.Left)
			walkExpr(node.Right)
		case *ast.UnaryExpr:
			walkExpr(node.Expr)
		case *ast.ArrayLiteralExpr:
			for _, elem := range node.Elements {
				walkExpr(elem)
			}
		case *ast.ObjectLiteralExpr:
			for _, prop := range node.Properties {
				walkExpr(prop.Value)
			}
		case *ast.ArrowFunctionExpr:
			for _, body := range node.Body {
				walkStmt(body)
			}
		}
	}
	walkStmt = func(node ast.Stmt) {
		switch s := node.(type) {
		case *ast.VarDecl:
			walkExpr(s.Value)
		case *ast.ConstDecl:
			walkExpr(s.Value)
		case *ast.FunctionDecl:
			for _, body := range s.Body {
				walkStmt(body)
			}
		case *ast.ReturnStmt:
			walkExpr(s.Value)
		case *ast.ExprStmt:
			walkExpr(s.Expr)
		case *ast.AssignStmt:
			walkExpr(s.Value)
		case *ast.IfStmt:
			walkExpr(s.Cond)
			for _, b := range s.Then {
				walkStmt(b)
			}
			for _, b := range s.Else {
				walkStmt(b)
			}
		case *ast.WhileStmt:
			walkExpr(s.Cond)
			for _, b := range s.Body {
				walkStmt(b)
			}
		case *ast.DoWhileStmt:
			for _, b := range s.Body {
				walkStmt(b)
			}
			walkExpr(s.Cond)
		case *ast.ForStmt:
			walkStmt(s.Init)
			walkExpr(s.Cond)
			walkExpr(s.Post)
			for _, b := range s.Body {
				walkStmt(b)
			}
		case *ast.SwitchStmt:
			walkExpr(s.Expr)
			for _, c := range s.Cases {
				walkExpr(c.Value)
				for _, b := range c.Body {
					walkStmt(b)
				}
			}
		case *ast.TaskDecl:
			walkExpr(s.Interval)
			walkExpr(s.Callback)
		case *ast.BlockStmt:
			for _, b := range s.Statements {
				walkStmt(b)
			}
		}
	}
	walkStmt(stmt)
}

func validateFunctionReturnConsistency(program *ast.Program) error {
	for _, stmt := range program.Statements {
		fn, ok := stmt.(*ast.FunctionDecl)
		if !ok {
			continue
		}
		returns := collectReturns(fn.Body)
		seenBare := false
		var expectedType string
		for _, ret := range returns {
			if ret.Value == nil {
				seenBare = true
				continue
			}
			if seenBare {
				return fmt.Errorf("line %d:%d: function '%s' mixes bare and value returns", ret.Pos.Line, ret.Pos.Column, fn.Name)
			}
			actualType, known := inferLiteralType(ret.Value, nil)
			if !known {
				continue
			}
			if expectedType == "" {
				expectedType = actualType
				continue
			}
			if !returnTypeCompatible(expectedType, actualType) {
				return fmt.Errorf("line %d:%d: inconsistent return type in function '%s': expected %s, got %s", ret.Pos.Line, ret.Pos.Column, fn.Name, expectedType, actualType)
			}
			if isNumericType(expectedType) && isNumericType(actualType) {
				expectedType = mergeNumericType(expectedType, actualType)
			}
		}
		if seenBare {
			for _, ret := range returns {
				if ret.Value != nil {
					return fmt.Errorf("line %d:%d: function '%s' mixes bare and value returns", ret.Pos.Line, ret.Pos.Column, fn.Name)
				}
			}
		}
	}
	return nil
}

func collectReturns(stmts []ast.Stmt) []*ast.ReturnStmt {
	out := make([]*ast.ReturnStmt, 0)
	var walk func(ast.Stmt)
	walk = func(stmt ast.Stmt) {
		switch node := stmt.(type) {
		case *ast.ReturnStmt:
			out = append(out, node)
		case *ast.IfStmt:
			for _, s := range node.Then {
				walk(s)
			}
			for _, s := range node.Else {
				walk(s)
			}
		case *ast.WhileStmt:
			for _, s := range node.Body {
				walk(s)
			}
		case *ast.DoWhileStmt:
			for _, s := range node.Body {
				walk(s)
			}
		case *ast.ForStmt:
			for _, s := range node.Body {
				walk(s)
			}
		case *ast.SwitchStmt:
			for _, c := range node.Cases {
				for _, s := range c.Body {
					walk(s)
				}
			}
		case *ast.BlockStmt:
			for _, s := range node.Statements {
				walk(s)
			}
		case *ast.TaskDecl:
			// Callback return statements are scoped to the callback, not enclosing function.
			return
		}
	}
	for _, stmt := range stmts {
		walk(stmt)
	}
	return out
}

// pushScope appends a fresh lexical scope. It copies the existing stack so
// callers never mutate a backing array shared with an outer frame.
func pushScope(scopeStack []map[string]string) []map[string]string {
	stack := make([]map[string]string, 0, len(scopeStack)+1)
	stack = append(stack, scopeStack...)
	return append(stack, map[string]string{})
}

// checkBlock checks each statement of a block body inside a new lexical scope.
func checkBlock(body []ast.Stmt, scopeStack []map[string]string, taskNames map[string]struct{}, loopDepth int, switchDepth int, functionReturns map[string]string, functionArity map[string]int) error {
	inner := pushScope(scopeStack)
	for _, s := range body {
		if err := checkStmt(s, inner, taskNames, loopDepth, switchDepth, functionReturns, functionArity); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(stmt ast.Stmt, scopeStack []map[string]string, taskNames map[string]struct{}, loopDepth int, switchDepth int, functionReturns map[string]string, functionArity map[string]int) error {
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
		if isArrayBinding(bindingKind) && len(scopeStack) > 1 {
			return fmt.Errorf("line %d:%d: array declarations are only supported at top-level scope", node.Pos.Line, node.Pos.Column)
		}
		if resolvedType, ok := scalarBindingType(bindingKind); ok {
			node.ResolvedType = resolvedType
		}
		current[node.Name] = bindingKind
		if node.Value != nil {
			if err := checkExpr(node.Value, scopeStack, functionArity); err != nil {
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
		if isArrayBinding(bindingKind) && len(scopeStack) > 1 {
			return fmt.Errorf("line %d:%d: array declarations are only supported at top-level scope", node.Pos.Line, node.Pos.Column)
		}
		if resolvedType, ok := scalarBindingType(bindingKind); ok {
			node.ResolvedType = resolvedType
		}
		current[node.Name] = bindingKind
		if node.Value != nil {
			if err := checkExpr(node.Value, scopeStack, functionArity); err != nil {
				return err
			}
		}
	case *ast.FunctionDecl:
		if existing, ok := current[node.Name]; ok && !strings.HasPrefix(existing, "function-") {
			return fmt.Errorf("line %d:%d: function '%s' already declared", node.Pos.Line, node.Pos.Column, node.Name)
		}
		current[node.Name] = encodeFunctionBinding(node.ReturnType, node.ParamTypes...)
		funcScope := map[string]string{}
		for i, param := range node.Params {
			if _, exists := funcScope[param]; exists {
				return fmt.Errorf("line %d:%d: parameter '%s' already declared", node.Pos.Line, node.Pos.Column, param)
			}
			paramType := "int32"
			if i < len(node.ParamTypes) {
				paramType = node.ParamTypes[i]
			}
			funcScope[param] = bindingForType(paramType)
		}
		stack := make([]map[string]string, 0, len(scopeStack)+1)
		stack = append(stack, scopeStack...)
		stack = append(stack, funcScope)
		for _, s := range node.Body {
			if err := checkStmt(s, stack, taskNames, loopDepth, switchDepth, functionReturns, functionArity); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		if err := checkExpr(node.Cond, scopeStack, functionArity); err != nil {
			return err
		}
		if err := checkBlock(node.Then, scopeStack, taskNames, loopDepth, switchDepth, functionReturns, functionArity); err != nil {
			return err
		}
		if err := checkBlock(node.Else, scopeStack, taskNames, loopDepth, switchDepth, functionReturns, functionArity); err != nil {
			return err
		}
	case *ast.WhileStmt:
		if err := checkExpr(node.Cond, scopeStack, functionArity); err != nil {
			return err
		}
		if err := checkBlock(node.Body, scopeStack, taskNames, loopDepth+1, switchDepth, functionReturns, functionArity); err != nil {
			return err
		}
	case *ast.DoWhileStmt:
		if err := checkBlock(node.Body, scopeStack, taskNames, loopDepth+1, switchDepth, functionReturns, functionArity); err != nil {
			return err
		}
		if err := checkExpr(node.Cond, scopeStack, functionArity); err != nil {
			return err
		}
	case *ast.ForStmt:
		inner := pushScope(scopeStack)
		if node.Init != nil {
			if err := checkStmt(node.Init, inner, taskNames, loopDepth+1, switchDepth, functionReturns, functionArity); err != nil {
				return err
			}
		}
		if node.Cond != nil {
			if err := checkExpr(node.Cond, inner, functionArity); err != nil {
				return err
			}
		}
		for _, s := range node.Body {
			if err := checkStmt(s, inner, taskNames, loopDepth+1, switchDepth, functionReturns, functionArity); err != nil {
				return err
			}
		}
		if node.Post != nil {
			if err := checkExpr(node.Post, inner, functionArity); err != nil {
				return err
			}
		}
	case *ast.TaskDecl:
		if _, ok := taskNames[node.Name]; ok {
			return fmt.Errorf("line %d:%d: duplicate task name '%s'", node.Pos.Line, node.Pos.Column, node.Name)
		}
		taskNames[node.Name] = struct{}{}
		if err := checkExpr(node.Interval, scopeStack, functionArity); err != nil {
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
				if err := checkStmt(s, stack, taskNames, loopDepth, switchDepth, functionReturns, functionArity); err != nil {
					return err
				}
			}
		}
	case *ast.SwitchStmt:
		if err := checkExpr(node.Expr, scopeStack, functionArity); err != nil {
			return err
		}
		for _, c := range node.Cases {
			if c.Value != nil {
				if err := checkExpr(c.Value, scopeStack, functionArity); err != nil {
					return err
				}
			}
			if err := checkBlock(c.Body, scopeStack, taskNames, loopDepth, switchDepth+1, functionReturns, functionArity); err != nil {
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
		return checkBlock(node.Statements, scopeStack, taskNames, loopDepth, switchDepth, functionReturns, functionArity)
	case *ast.ReturnStmt:
		if node.Value != nil {
			return checkExpr(node.Value, scopeStack, functionArity)
		}
	case *ast.ExprStmt:
		return checkExpr(node.Expr, scopeStack, functionArity)
	case *ast.AssignStmt:
		binding, ok := lookup(scopeStack, node.Target)
		if !ok {
			return fmt.Errorf("line %d:%d: undefined variable '%s'", node.Pos.Line, node.Pos.Column, node.Target)
		}
		if isConstBinding(scopeStack, node.Target) {
			return fmt.Errorf("line %d:%d: cannot assign to constant variable '%s'", node.Pos.Line, node.Pos.Column, node.Target)
		}
		if err := checkExpr(node.Value, scopeStack, functionArity); err != nil {
			return err
		}
		expectedType, typed := scalarBindingType(binding)
		actualType, known := inferLiteralType(node.Value, scopeStack)
		if typed && known && !typeCompatible(expectedType, actualType) {
			return fmt.Errorf("line %d:%d: type mismatch assigning '%s': expected %s, got %s", node.Pos.Line, node.Pos.Column, node.Target, expectedType, actualType)
		}
		return nil
	}
	return nil
}

func checkExpr(expr ast.Expr, scopeStack []map[string]string, functionArity map[string]int) error {
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
						if err := checkExpr(node.Args[0], scopeStack, functionArity); err != nil {
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
		if err := checkExpr(node.Function, scopeStack, functionArity); err != nil {
			return err
		}
		if fn, ok := node.Function.(*ast.Identifier); ok {
			if expected, known := functionArity[fn.Name]; known && len(node.Args) != expected {
				return fmt.Errorf("line %d:%d: function '%s' expects %d argument(s), got %d", node.Pos.Line, node.Pos.Column, fn.Name, expected, len(node.Args))
			}
		}
		for _, arg := range node.Args {
			if err := checkExpr(arg, scopeStack, functionArity); err != nil {
				return err
			}
		}
		if fn, ok := node.Function.(*ast.Identifier); ok {
			if binding, exists := lookup(scopeStack, fn.Name); exists {
				if params, isFunction := functionParamTypes(binding); isFunction {
					for i, expected := range params {
						actual, known := inferLiteralType(node.Args[i], scopeStack)
						if !known {
							return fmt.Errorf("line %d:%d: cannot resolve type of argument %d to function '%s'", node.Pos.Line, node.Pos.Column, i+1, fn.Name)
						}
						if !typeCompatible(expected, actual) {
							return fmt.Errorf("line %d:%d: argument %d to function '%s' expects %s, got %s", node.Pos.Line, node.Pos.Column, i+1, fn.Name, expected, actual)
						}
					}
				}
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
		return checkExpr(node.Object, scopeStack, functionArity)
	case *ast.IndexExpr:
		ident, ok := node.Object.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("line %d:%d: array indexing requires an identifier target", node.Pos.Line, node.Pos.Column)
		}
		kind, exists := lookup(scopeStack, ident.Name)
		if !exists || !isArrayBinding(kind) {
			return fmt.Errorf("line %d:%d: '%s' is not an array", node.Pos.Line, node.Pos.Column, ident.Name)
		}
		if err := checkExpr(node.Index, scopeStack, functionArity); err != nil {
			return err
		}
		if idx, ok := indexLiteral(node.Index); ok {
			if idx < 0 || idx >= arrayCapacity(kind) {
				return fmt.Errorf("line %d:%d: array index %d out of bounds (capacity %d)", node.Pos.Line, node.Pos.Column, idx, arrayCapacity(kind))
			}
		}
		return nil
	case *ast.BinaryExpr:
		if err := checkExpr(node.Left, scopeStack, functionArity); err != nil {
			return err
		}
		leftType, leftKnown := inferLiteralType(node.Left, scopeStack)
		rightType, rightKnown := inferLiteralType(node.Right, scopeStack)
		if leftKnown && rightKnown && (isNumericOperator(node.Op) || isRelationalOperator(node.Op)) {
			if !isNumericType(leftType) || !isNumericType(rightType) {
				return fmt.Errorf("line %d:%d: operator '%s' requires numeric operands", node.Pos.Line, node.Pos.Column, node.Op)
			}
		}
		if isEqualityOp(node.Op) {
			if leftKnown && rightKnown && !equalityComparable(leftType, rightType) {
				return fmt.Errorf("line %d:%d: cannot compare %s and %s with %s", node.Pos.Line, node.Pos.Column, leftType, rightType, node.Op)
			}
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
				if binding, exists := lookup(scopeStack, ident.Name); exists {
					expectedType, typed := scalarBindingType(binding)
					actualType, known := inferLiteralType(node.Right, scopeStack)
					if typed && known && !typeCompatible(expectedType, actualType) {
						return fmt.Errorf("line %d:%d: type mismatch assigning '%s': expected %s, got %s", node.Pos.Line, node.Pos.Column, ident.Name, expectedType, actualType)
					}
				}
			}
			if _, isArrayLiteral := node.Right.(*ast.ArrayLiteralExpr); isArrayLiteral {
				return fmt.Errorf("line %d:%d: array literals are only supported in top-level declarations", node.Pos.Line, node.Pos.Column)
			}
		}
		if err := checkExpr(node.Right, scopeStack, functionArity); err != nil {
			return err
		}
	case *ast.UnaryExpr:
		if (node.Op == "++" || node.Op == "--") && isConstBinding(scopeStack, renderUnaryTarget(node.Expr)) {
			return fmt.Errorf("line %d:%d: cannot assign to constant variable '%s'", node.Pos.Line, node.Pos.Column, renderUnaryTarget(node.Expr))
		}
		return checkExpr(node.Expr, scopeStack, functionArity)
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
			if err := checkExpr(elem, scopeStack, functionArity); err != nil {
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
			if err := checkExpr(property.Value, scopeStack, functionArity); err != nil {
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
			if err := checkStmt(s, stack, map[string]struct{}{}, 0, 0, map[string]string{}, map[string]int{}); err != nil {
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
			return val == "const" || strings.Contains(val, "-const")
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
	declaredType = strings.TrimSpace(declaredType)
	if declaredType != "" && !isSupportedType(declaredType) {
		return "", fmt.Errorf("unsupported declared type '%s'", declaredType)
	}

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
			et, ok := inferLiteralType(elem, scopeStack)
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
		elemType, ok := inferLiteralType(lit.Elements[0], scopeStack)
		if !ok {
			return "", fmt.Errorf("unsupported array element type")
		}
		for _, elem := range lit.Elements[1:] {
			et, ok := inferLiteralType(elem, scopeStack)
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
				if shape == "void" {
					return "", fmt.Errorf("function '%s' does not return a value", fn.Name)
				}
				if strings.Contains(shape, ":") {
					return encodeObjectBinding(baseKind, shape), nil
				}
				return encodeScalarBinding(baseKind, shape), nil
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
			pinName.CanonicalValue = ch32fun.NormalizePin(pinName.Value)
			return "pin-" + baseKind, nil
		}
	}
	resolvedType := declaredType
	if resolvedType == "" {
		inferred, ok := inferLiteralType(value, scopeStack)
		if ok && !strings.Contains(inferred, ":") {
			resolvedType = inferred
		}
	}
	if resolvedType == "" {
		resolvedType = "int32"
	}
	if inferred, ok := inferLiteralType(value, scopeStack); ok && !strings.Contains(inferred, ":") {
		if !typeCompatible(resolvedType, inferred) {
			return "", fmt.Errorf("type mismatch for declaration: expected %s, got %s", resolvedType, inferred)
		}
	}
	return encodeScalarBinding(baseKind, resolvedType), nil
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

// inferFunctionArity maps each top-level function name to its parameter count.
func inferFunctionArity(program *ast.Program) map[string]int {
	arity := map[string]int{}
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDecl); ok {
			arity[fn.Name] = len(fn.Params)
		}
	}
	return arity
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
		switch node.Name {
		case "HIGH", "LOW":
			return "int32", true
		}
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
		if t, ok := scalarBindingType(binding); ok {
			return t, true
		}
		return "", false
	case *ast.CallExpr:
		if ident, ok := node.Function.(*ast.Identifier); ok && ident.Name == "pin" {
			return "int32", true
		}
		if ident, ok := node.Function.(*ast.Identifier); ok && scopeStack != nil {
			if binding, exists := lookup(scopeStack, ident.Name); exists {
				if returnType, ok := functionReturnType(binding); ok && returnType != "void" {
					return returnType, true
				}
			}
		}
		return "", false
	case *ast.SelectorExpr:
		if ident, ok := node.Object.(*ast.Identifier); ok && scopeStack != nil {
			if binding, exists := lookup(scopeStack, ident.Name); exists && isObjectBinding(binding) {
				return objectPropertyType(binding, node.Field)
			}
		}
		return "", false
	case *ast.ObjectLiteralExpr:
		return objectTypeFromLiteral(node, scopeStack), true
	case *ast.BinaryExpr:
		if node.Op == "&&" || node.Op == "||" || isEqualityOp(node.Op) || node.Op == "<" || node.Op == ">" || node.Op == "<=" || node.Op == ">=" {
			return "boolean", true
		}
		left, leftOK := inferLiteralType(node.Left, scopeStack)
		right, rightOK := inferLiteralType(node.Right, scopeStack)
		if leftOK && rightOK && isNumericType(left) && isNumericType(right) {
			return mergeNumericType(left, right), true
		}
		if node.Op == "=" || isAssignmentOp(node.Op) {
			return left, leftOK
		}
		return "", false
	case *ast.UnaryExpr:
		if node.Op == "!" {
			return "boolean", true
		}
		return inferLiteralType(node.Expr, scopeStack)
	default:
		return "", false
	}
}

func typeCompatible(expected string, actual string) bool {
	if expected == actual {
		return true
	}
	if isNumericType(expected) && isNumericType(actual) {
		return numericAssignable(expected, actual)
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

func encodeScalarBinding(baseKind, t string) string {
	return fmt.Sprintf("scalar-%s-%s", baseKind, t)
}

func scalarBindingType(binding string) (string, bool) {
	parts := strings.SplitN(binding, "-", 3)
	if len(parts) != 3 || parts[0] != "scalar" {
		return "", false
	}
	return parts[2], true
}

func isSupportedType(t string) bool {
	if isNumericType(t) {
		return true
	}
	return t == "boolean" || t == "string"
}

func numericAssignable(expected, actual string) bool {
	switch expected {
	case "int8":
		return actual == "int8"
	case "uint8":
		return actual == "uint8"
	case "int16":
		return actual == "int8" || actual == "uint8" || actual == "int16"
	case "uint16":
		return actual == "uint8" || actual == "uint16"
	case "int32":
		return actual == "int8" || actual == "uint8" || actual == "int16" || actual == "uint16" || actual == "int32"
	case "uint32":
		return actual == "uint8" || actual == "uint16" || actual == "uint32"
	case "float32":
		return actual == "int8" || actual == "uint8" || actual == "int16" || actual == "uint16" || actual == "int32" || actual == "uint32" || actual == "float32"
	case "number":
		return isNumericType(actual)
	default:
		return false
	}
}

func isEqualityOp(op string) bool {
	switch op {
	case "==", "===", "!=", "!==":
		return true
	default:
		return false
	}
}

func isNumericOperator(op string) bool {
	switch op {
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>":
		return true
	default:
		return false
	}
}

func isRelationalOperator(op string) bool {
	switch op {
	case "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}

func equalityComparable(left, right string) bool {
	if left == right {
		return true
	}
	if isNumericType(left) && isNumericType(right) {
		return true
	}
	return false
}

func returnTypeCompatible(expected, actual string) bool {
	if typeCompatible(expected, actual) || typeCompatible(actual, expected) {
		return true
	}
	return isNumericType(expected) && isNumericType(actual)
}

func mergeNumericType(a, b string) string {
	if a == "float32" || b == "float32" {
		return "float32"
	}
	return "int32"
}
