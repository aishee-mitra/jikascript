package cgen

import (
	"fmt"
	"sort"
	"strings"

	"jikascript/internal/ast"
)

type arrayMeta struct {
	Name     string
	ElemType string
	CType    string
	Capacity int
}

var generatedArrays map[string]arrayMeta
var generatedObjects map[string]objectMeta
var generatedFunctions map[string]functionMeta
var generatedObjectBindings map[string]string
var mangledIdentifiers map[string]string
var usedCIdentifiers map[string]string
var generationErr error

type objectMeta struct {
	Shape  string
	CName  string
	Fields []ast.ObjectProperty
}

type functionMeta struct {
	ParamTypes []string
	ReturnType string
}

type symbolAnnotator struct {
	next   int
	scopes []map[string]string
}

func annotateSymbols(program *ast.Program) {
	a := &symbolAnnotator{scopes: []map[string]string{{}}}
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionDecl); ok {
			fn.Symbol = a.declare(fn.Name)
		}
	}
	for _, stmt := range program.Statements {
		a.stmt(stmt)
	}
}

func (a *symbolAnnotator) declare(name string) string {
	scopeClass := "local"
	if len(a.scopes) == 1 {
		scopeClass = "global"
	}
	symbol := fmt.Sprintf("%s_symbol_%d_%s", scopeClass, a.next, normalizeIdentifier(name))
	a.next++
	a.scopes[len(a.scopes)-1][name] = symbol
	return symbol
}

func (a *symbolAnnotator) lookup(name string) string {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if symbol, ok := a.scopes[i][name]; ok {
			return symbol
		}
	}
	return ""
}

func (a *symbolAnnotator) push() { a.scopes = append(a.scopes, map[string]string{}) }

func (a *symbolAnnotator) pop() { a.scopes = a.scopes[:len(a.scopes)-1] }

func (a *symbolAnnotator) block(body []ast.Stmt) {
	a.push()
	for _, stmt := range body {
		a.stmt(stmt)
	}
	a.pop()
}

func (a *symbolAnnotator) stmt(stmt ast.Stmt) {
	switch node := stmt.(type) {
	case *ast.VarDecl:
		node.Symbol = a.declare(node.Name)
		a.expr(node.Value)
	case *ast.ConstDecl:
		node.Symbol = a.declare(node.Name)
		a.expr(node.Value)
	case *ast.FunctionDecl:
		if node.Symbol == "" {
			node.Symbol = a.lookup(node.Name)
			if node.Symbol == "" {
				node.Symbol = a.declare(node.Name)
			}
		}
		a.push()
		node.ParamSymbols = make([]string, len(node.Params))
		for i, param := range node.Params {
			node.ParamSymbols[i] = a.declare(param)
		}
		for _, body := range node.Body {
			a.stmt(body)
		}
		a.pop()
	case *ast.ReturnStmt:
		a.expr(node.Value)
	case *ast.ExprStmt:
		a.expr(node.Expr)
	case *ast.AssignStmt:
		if symbol := a.lookup(node.Target); symbol != "" {
			// AssignStmt predates symbol annotations and stores its target as text.
			node.Target = symbol
		}
		a.expr(node.Value)
	case *ast.IfStmt:
		a.expr(node.Cond)
		a.block(node.Then)
		a.block(node.Else)
	case *ast.WhileStmt:
		a.expr(node.Cond)
		a.block(node.Body)
	case *ast.DoWhileStmt:
		a.block(node.Body)
		a.expr(node.Cond)
	case *ast.ForStmt:
		a.push()
		if node.Init != nil {
			a.stmt(node.Init)
		}
		a.expr(node.Cond)
		for _, body := range node.Body {
			a.stmt(body)
		}
		a.expr(node.Post)
		a.pop()
	case *ast.SwitchStmt:
		a.expr(node.Expr)
		for _, clause := range node.Cases {
			a.push()
			a.expr(clause.Value)
			for _, body := range clause.Body {
				a.stmt(body)
			}
			a.pop()
		}
	case *ast.TaskDecl:
		a.expr(node.Interval)
		if node.Callback != nil {
			a.push()
			for _, param := range node.Callback.Params {
				a.declare(param)
			}
			for _, body := range node.Callback.Body {
				a.stmt(body)
			}
			a.pop()
		}
	case *ast.BlockStmt:
		a.block(node.Statements)
	}
}

func (a *symbolAnnotator) expr(expr ast.Expr) {
	if expr == nil {
		return
	}
	switch node := expr.(type) {
	case *ast.Identifier:
		node.Symbol = a.lookup(node.Name)
	case *ast.CallExpr:
		a.expr(node.Function)
		for _, arg := range node.Args {
			a.expr(arg)
		}
	case *ast.SelectorExpr:
		a.expr(node.Object)
	case *ast.IndexExpr:
		a.expr(node.Object)
		a.expr(node.Index)
	case *ast.BinaryExpr:
		a.expr(node.Left)
		a.expr(node.Right)
	case *ast.UnaryExpr:
		a.expr(node.Expr)
	case *ast.ArrayLiteralExpr:
		for _, elem := range node.Elements {
			a.expr(elem)
		}
	case *ast.ObjectLiteralExpr:
		for _, property := range node.Properties {
			a.expr(property.Value)
		}
	case *ast.ArrowFunctionExpr:
		a.push()
		for _, param := range node.Params {
			a.declare(param)
		}
		for _, body := range node.Body {
			a.stmt(body)
		}
		a.pop()
	}
}

func symbolKey(symbol, name string) string {
	if symbol != "" {
		return symbol
	}
	return name
}

func sanitizeBinding(symbol, name string) string {
	if symbol == "" {
		return sanitizeName(name)
	}
	if existing, ok := mangledIdentifiers[symbol]; ok {
		return existing
	}
	normalized := normalizeIdentifier(name)
	if normalized == "" {
		normalized = "anon"
	}
	candidate := "js_" + normalized
	for isReservedCIdentifier(candidate) {
		candidate = "js_" + candidate
	}
	base := candidate
	if owner, exists := usedCIdentifiers[candidate]; exists && owner != symbol && symbolScope(owner) == symbolScope(symbol) {
		for i := 2; ; i++ {
			candidate = fmt.Sprintf("%s_%d", base, i)
			if isReservedCIdentifier(candidate) {
				continue
			}
			if _, used := usedCIdentifiers[candidate]; !used {
				break
			}
		}
	}
	mangledIdentifiers[symbol] = candidate
	usedCIdentifiers[candidate] = symbol
	return candidate
}

func symbolScope(symbol string) string {
	if strings.HasPrefix(symbol, "global_") {
		return "global"
	}
	return "local"
}

// Generate emits readable C99-like output for the initial JikaScript subset.
func Generate(program *ast.Program) (string, error) {
	var b strings.Builder
	generatedArrays = map[string]arrayMeta{}
	generatedObjects = map[string]objectMeta{}
	generatedFunctions = map[string]functionMeta{}
	generatedObjectBindings = map[string]string{}
	mangledIdentifiers = map[string]string{}
	usedCIdentifiers = map[string]string{}
	generationErr = nil
	annotateSymbols(program)
	collectObjectTypes(program)
	collectObjectBindings(program)
	collectFunctionTypes(program)
	b.WriteString("#include <stdbool.h>\n")
	b.WriteString("#include <stdint.h>\n\n")
	b.WriteString("#include \"periodic.h\"\n\n")
	b.WriteString("#include \"gpio.h\"\n\n")
	b.WriteString("// Generated by JikaScript transpiler\n\n")
	for _, meta := range sortedObjectMetas() {
		b.WriteString(renderObjectType(meta))
	}
	if len(generatedObjects) > 0 {
		b.WriteString("\n")
	}

	taskHandlers := []taskRegistration{}
	mainStmts := []string{}
	arrayDecls := []arrayMeta{}
	for _, stmt := range program.Statements {
		switch node := stmt.(type) {
		case *ast.FunctionDecl:
			b.WriteString(emitFunction(node))
		case *ast.TaskDecl:
			taskName := taskHandlerName(node.Name)
			b.WriteString(fmt.Sprintf("static void %s(void) {\n", taskName))
			for _, inner := range node.Callback.Body {
				b.WriteString(renderStmt(inner, "    "))
			}
			b.WriteString("}\n\n")
			taskHandlers = append(taskHandlers, taskRegistration{name: taskName, interval: renderExpr(node.Interval)})
		case *ast.VarDecl:
			if meta, ok := objectDeclMeta(node.Value); ok {
				b.WriteString(renderObjectTopLevelDecl(meta, node.Name, node.Symbol, node.Value))
				break
			}
			if meta, initStmts, ok := buildArrayDeclFromVar(node); ok {
				arrayDecls = append(arrayDecls, meta)
				generatedArrays[node.Name] = meta
				b.WriteString(renderArrayDecl(meta))
				for _, init := range initStmts {
					mainStmts = append(mainStmts, "    "+init+";\n")
				}
				break
			}
			decl, init := renderTopLevelDecl(node, false)
			b.WriteString(decl)
			if init != "" {
				mainStmts = append(mainStmts, "    "+init+";\n")
			}
		case *ast.ConstDecl:
			if meta, ok := objectDeclMeta(node.Value); ok {
				// `const` freezes the JikaScript binding, not the record contents.
				b.WriteString(renderObjectTopLevelDecl(meta, node.Name, node.Symbol, node.Value))
				break
			}
			if meta, initStmts, ok := buildArrayDeclFromConst(node); ok {
				arrayDecls = append(arrayDecls, meta)
				generatedArrays[node.Name] = meta
				b.WriteString(renderArrayDecl(meta))
				for _, init := range initStmts {
					mainStmts = append(mainStmts, "    "+init+";\n")
				}
				break
			}
			decl, init := renderTopLevelDecl(node, true)
			b.WriteString(decl)
			if init != "" {
				mainStmts = append(mainStmts, "    "+init+";\n")
			}
		case *ast.ExprStmt:
			mainStmts = append(mainStmts, renderExprStmt(node, "    "))
		default:
			mainStmts = append(mainStmts, renderStmt(node, "    "))
		}
		if generationErr != nil {
			return "", generationErr
		}
	}

	if len(arrayDecls) > 0 {
		b.WriteString("static void jikascript_array_fail(void) {\n")
		b.WriteString("    while (1) { }\n")
		b.WriteString("}\n\n")
		for _, meta := range arrayDecls {
			b.WriteString(renderArrayHelpers(meta))
		}
	}

	for _, task := range taskHandlers {
		b.WriteString(fmt.Sprintf("static periodic_task_t %s_task;\n", task.name))
	}
	if len(taskHandlers) > 0 {
		b.WriteString("\n")
	}

	b.WriteString("int main(void) {\n")
	b.WriteString("    periodic_init();\n")
	for _, task := range taskHandlers {
		b.WriteString(fmt.Sprintf("    periodic_task_add(&%s_task, %s, %s);\n", task.name, task.name, task.interval))
	}
	for _, stmt := range mainStmts {
		b.WriteString(stmt)
	}
	b.WriteString("    while (1) {\n        periodic_run();\n    }\n    return 0;\n}\n")
	if generationErr != nil {
		return "", generationErr
	}
	return b.String(), nil
}

func sortedObjectMetas() []objectMeta {
	metas := make([]objectMeta, 0, len(generatedObjects))
	for _, meta := range generatedObjects {
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].CName < metas[j].CName })
	return metas
}

type taskRegistration struct {
	name     string
	interval string
}

func emitFunction(node *ast.FunctionDecl) string {
	var b strings.Builder
	meta := generatedFunctions[node.Name]
	returnType := meta.ReturnType
	if returnType == "" {
		returnType = "int"
	}
	b.WriteString(fmt.Sprintf("static %s %s(%s) {\n", returnType, sanitizeBinding(node.Symbol, node.Name), renderTypedParams(node.Params, meta.ParamTypes, node.ParamSymbols)))
	for _, stmt := range node.Body {
		b.WriteString(renderStmt(stmt, "    "))
	}
	b.WriteString("}\n\n")
	return b.String()
}

func renderVarDecl(node interface{}, isConst bool) string {
	switch v := node.(type) {
	case *ast.VarDecl:
		return renderLocalDecl(v.Name, v.Symbol, effectiveType(v.Type, v.ResolvedType), v.Value, false)
	case *ast.ConstDecl:
		return renderLocalDecl(v.Name, v.Symbol, effectiveType(v.Type, v.ResolvedType), v.Value, true)
	}
	return ""
}

// renderLocalDecl renders a function-local variable declaration. Locals are
// never static: they live on the stack and are initialized at runtime.
// `const` freezes the JikaScript binding, not record contents, so object
// locals are never C-const (property writes must remain legal).
func renderLocalDecl(name, symbol, declaredType string, value ast.Expr, isConst bool) string {
	cName := sanitizeBinding(symbol, name)
	constPrefix := ""
	if isConst && !isObjectValue(value) {
		constPrefix = "const "
	}
	if meta, ok := objectDeclMeta(value); ok {
		return fmt.Sprintf("%s%s %s = %s;\n", constPrefix, meta.CName, cName, renderExpr(value))
	}
	if cType, ok := functionReturnCType(value); ok {
		return fmt.Sprintf("%s%s %s = %s;\n", constPrefix, cType, cName, renderExpr(value))
	}
	if isPinCall(value) {
		// pin() returns a hardware handle; declare then initialize separately
		// so the call stays inside the function body.
		return fmt.Sprintf("%sint %s = %s;\n", constPrefix, cName, renderExpr(value))
	}
	return fmt.Sprintf("%s%s %s = %s;\n", constPrefix, typeName(declaredType, isConst), cName, renderExpr(value))
}

func isObjectValue(expr ast.Expr) bool {
	_, ok := objectDeclMeta(expr)
	return ok
}

func isPinCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	ident, ok := call.Function.(*ast.Identifier)
	return ok && ident.Name == "pin"
}

func renderTopLevelDecl(node interface{}, isConst bool) (string, string) {
	switch v := node.(type) {
	case *ast.VarDecl:
		name := sanitizeBinding(v.Symbol, v.Name)
		declaredType := effectiveType(v.Type, v.ResolvedType)
		if cType, ok := functionReturnCType(v.Value); ok {
			return fmt.Sprintf("static %s %s;\n", cType, name), fmt.Sprintf("%s = %s", name, renderExpr(v.Value))
		}
		if isStaticInitSafe(v.Value) {
			return fmt.Sprintf("static %s %s = %s;\n", typeName(declaredType, false), name, renderExpr(v.Value)), ""
		}
		return fmt.Sprintf("static %s %s;\n", typeName(declaredType, false), name), fmt.Sprintf("%s = %s", name, renderExpr(v.Value))
	case *ast.ConstDecl:
		name := sanitizeBinding(v.Symbol, v.Name)
		declaredType := effectiveType(v.Type, v.ResolvedType)
		if cType, ok := functionReturnCType(v.Value); ok {
			return fmt.Sprintf("static %s %s;\n", cType, name), fmt.Sprintf("%s = %s", name, renderExpr(v.Value))
		}
		if isStaticInitSafe(v.Value) {
			return fmt.Sprintf("static const %s %s = %s;\n", typeName(declaredType, true), name, renderExpr(v.Value)), ""
		}
		return fmt.Sprintf("static %s %s;\n", typeName(declaredType, false), name), fmt.Sprintf("%s = %s", name, renderExpr(v.Value))
	default:
		return "", ""
	}
}

func effectiveType(declaredType, resolvedType string) string {
	if strings.TrimSpace(declaredType) != "" {
		return declaredType
	}
	return resolvedType
}

func renderExprStmt(node *ast.ExprStmt, indent string) string {
	if shouldSkipExprStmt(node.Expr) {
		return ""
	}
	return indent + renderExpr(node.Expr) + ";\n"
}

func renderStmt(stmt ast.Stmt, indent string) string {
	switch node := stmt.(type) {
	case *ast.ReturnStmt:
		if node.Value == nil {
			return indent + "return;\n"
		}
		return indent + "return " + renderExpr(node.Value) + ";\n"
	case *ast.ExprStmt:
		return renderExprStmt(node, indent)
	case *ast.IfStmt:
		var b strings.Builder
		b.WriteString(indent + "if (" + renderExpr(node.Cond) + ") {\n")
		for _, inner := range node.Then {
			b.WriteString(renderStmt(inner, indent+"    "))
		}
		if len(node.Else) > 0 {
			b.WriteString(indent + "} else {\n")
			for _, inner := range node.Else {
				b.WriteString(renderStmt(inner, indent+"    "))
			}
		}
		b.WriteString(indent + "}\n")
		return b.String()
	case *ast.WhileStmt:
		var b strings.Builder
		b.WriteString(indent + "while (" + renderExpr(node.Cond) + ") {\n")
		for _, inner := range node.Body {
			b.WriteString(renderStmt(inner, indent+"    "))
		}
		b.WriteString(indent + "}\n")
		return b.String()
	case *ast.DoWhileStmt:
		var b strings.Builder
		b.WriteString(indent + "do {\n")
		for _, inner := range node.Body {
			b.WriteString(renderStmt(inner, indent+"    "))
		}
		b.WriteString(indent + "} while (" + renderExpr(node.Cond) + ");\n")
		return b.String()
	case *ast.ForStmt:
		var b strings.Builder
		init := renderForInit(node.Init)
		cond := renderOptionalExpr(node.Cond)
		post := renderOptionalExpr(node.Post)
		b.WriteString(indent + "for (" + init + "; " + cond + "; " + post + ") {\n")
		for _, inner := range node.Body {
			b.WriteString(renderStmt(inner, indent+"    "))
		}
		b.WriteString(indent + "}\n")
		return b.String()
	case *ast.SwitchStmt:
		var b strings.Builder
		b.WriteString(indent + "switch (" + renderExpr(node.Expr) + ") {\n")
		for _, c := range node.Cases {
			if c.Value == nil {
				b.WriteString(indent + "default:\n")
			} else {
				b.WriteString(indent + "case " + renderExpr(c.Value) + ":\n")
			}
			for _, inner := range c.Body {
				b.WriteString(renderStmt(inner, indent+"    "))
			}
		}
		b.WriteString(indent + "}\n")
		return b.String()
	case *ast.BreakStmt:
		return indent + "break;\n"
	case *ast.ContinueStmt:
		return indent + "continue;\n"
	case *ast.VarDecl:
		return renderVarDecl(node, false)
	case *ast.ConstDecl:
		return renderVarDecl(node, true)
	case *ast.AssignStmt:
		return indent + sanitizeName(node.Target) + " = " + renderExpr(node.Value) + ";\n"
	case *ast.TaskDecl:
		return ""
	case *ast.BlockStmt:
		var b strings.Builder
		for _, inner := range node.Statements {
			b.WriteString(renderStmt(inner, indent))
		}
		return b.String()
	default:
		setGenerationErrorf("unsupported statement %T in code generation", node)
		return ""
	}
}

func renderExpr(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Identifier:
		switch node.Name {
		case "HIGH":
			return "1"
		case "LOW":
			return "0"
		}
		if node.Symbol == "" {
			return node.Name
		}
		if mangled, ok := mangledIdentifiers[symbolKey(node.Symbol, node.Name)]; ok {
			return mangled
		}
		return sanitizeBinding(node.Symbol, node.Name)
	case *ast.NumberLiteral:
		return node.Value
	case *ast.StringLiteral:
		if node.CanonicalValue != "" {
			return fmt.Sprintf("\"%s\"", escape(node.CanonicalValue))
		}
		return fmt.Sprintf("\"%s\"", escape(node.Value))
	case *ast.BoolLiteral:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.CallExpr:
		if ident, ok := node.Function.(*ast.Identifier); ok && ident.Name == "pin" {
			if _, shadowed := mangledIdentifiers["pin"]; !shadowed {
				var args []string
				for _, arg := range node.Args {
					args = append(args, renderExpr(arg))
				}
				return "pin(" + strings.Join(args, ", ") + ")"
			}
		}
		if call := renderSelectorCall(node); call != "" {
			return call
		}
		var args []string
		for _, arg := range node.Args {
			args = append(args, renderExpr(arg))
		}
		return renderExpr(node.Function) + "(" + strings.Join(args, ", ") + ")"
	case *ast.SelectorExpr:
		if ident, ok := node.Object.(*ast.Identifier); ok {
			if meta, exists := generatedArrays[ident.Name]; exists && node.Field == "length" {
				return sanitizeName(meta.Name) + ".length"
			}
		}
		return renderExpr(node.Object) + "." + sanitizeFieldName(node.Field)
	case *ast.IndexExpr:
		if ident, ok := node.Object.(*ast.Identifier); ok {
			if meta, exists := generatedArrays[ident.Name]; exists {
				return sanitizeName(meta.Name) + "_get((int32_t)(" + renderExpr(node.Index) + "))"
			}
		}
		setGenerationErrorf("unsupported indexed expression target in code generation")
		return ""
	case *ast.BinaryExpr:
		if isAssignmentOp(node.Op) {
			if idx, ok := node.Left.(*ast.IndexExpr); ok {
				if ident, ok := idx.Object.(*ast.Identifier); ok {
					if meta, exists := generatedArrays[ident.Name]; exists {
						indexExpr := "(int32_t)(" + renderExpr(idx.Index) + ")"
						if node.Op == "=" {
							return sanitizeName(meta.Name) + "_set(" + indexExpr + ", " + renderExpr(node.Right) + ")"
						}
						baseOp := strings.TrimSuffix(node.Op, "=")
						return sanitizeName(meta.Name) + "_set(" + indexExpr + ", " + sanitizeName(meta.Name) + "_get(" + indexExpr + ") " + baseOp + " " + renderExpr(node.Right) + ")"
					}
				}
			}
		}
		op := node.Op
		if op == "===" {
			op = "=="
		} else if op == "!==" {
			op = "!="
		}
		return renderExpr(node.Left) + " " + op + " " + renderExpr(node.Right)
	case *ast.UnaryExpr:
		return node.Op + renderExpr(node.Expr)
	case *ast.ArrayLiteralExpr:
		setGenerationErrorf("array literal expression is not supported in this context")
		return ""
	case *ast.ObjectLiteralExpr:
		parts := make([]string, 0, len(node.Properties))
		for _, property := range node.Properties {
			parts = append(parts, "."+sanitizeFieldName(property.Name)+" = "+renderExpr(property.Value))
		}
		if meta, ok := objectDeclMeta(node); ok {
			return "(" + meta.CName + "){ " + strings.Join(parts, ", ") + " }"
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case *ast.ArrowFunctionExpr:
		setGenerationErrorf("arrow function expression is not supported in this context")
		return ""
	default:
		setGenerationErrorf("unsupported expression %T in code generation", node)
		return ""
	}
}

func setGenerationErrorf(format string, args ...interface{}) {
	if generationErr != nil {
		return
	}
	generationErr = fmt.Errorf(format, args...)
}

func typeName(t string, constDecl bool) string {
	if t == "" {
		return "int"
	}
	switch t {
	case "void":
		return "void"
	case "uint8":
		return "uint8_t"
	case "uint16":
		return "uint16_t"
	case "uint32":
		return "uint32_t"
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "float32":
		return "float"
	case "boolean":
		return "bool"
	case "string":
		if constDecl {
			return "char *"
		}
		return "const char *"
	default:
		if strings.HasPrefix(t, "jikascript_object_") {
			return t
		}
		return "int"
	}
}

func sanitizeName(name string) string {
	if existing, ok := mangledIdentifiers[name]; ok {
		return existing
	}
	normalized := normalizeIdentifier(name)
	if normalized == "" {
		normalized = "anon"
	}
	candidate := "js_" + normalized
	for isReservedCIdentifier(candidate) {
		candidate = "js_" + candidate
	}
	base := candidate
	if owner, exists := usedCIdentifiers[candidate]; exists && owner != name {
		for i := 2; ; i++ {
			candidate = fmt.Sprintf("%s_%d", base, i)
			if isReservedCIdentifier(candidate) {
				continue
			}
			if _, used := usedCIdentifiers[candidate]; !used {
				break
			}
		}
	}
	mangledIdentifiers[name] = candidate
	usedCIdentifiers[candidate] = name
	return candidate
}

func sanitizeFieldName(name string) string {
	normalized := normalizeIdentifier(name)
	if normalized == "" {
		return "field"
	}
	if isImplementationReservedIdentifier(normalized) {
		return "field_" + strings.TrimLeft(normalized, "_")
	}
	if _, reserved := cKeywordIdentifiers[normalized]; reserved {
		return normalized + "_field"
	}
	return normalized
}

func normalizeIdentifier(name string) string {
	var b strings.Builder
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if i == 0 && r >= '0' && r <= '9' {
			b.WriteString("_")
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	return b.String()
}

func isReservedCIdentifier(name string) bool {
	if isImplementationReservedIdentifier(name) {
		return true
	}
	if _, ok := reservedCIdentifiers[name]; ok {
		return true
	}
	_, ok := cKeywordIdentifiers[name]
	return ok
}

func isImplementationReservedIdentifier(name string) bool {
	if strings.HasPrefix(name, "jikascript_") {
		return true
	}
	if strings.HasPrefix(name, "__") {
		return true
	}
	if len(name) >= 2 && name[0] == '_' && (name[1] == '_' || (name[1] >= 'A' && name[1] <= 'Z')) {
		return true
	}
	return false
}

var reservedCIdentifiers = map[string]struct{}{
	"main": {}, "pin": {}, "gpio_output": {}, "gpio_input": {}, "gpio_toggle": {}, "gpio_read": {}, "gpio_write": {},
	"periodic_init": {}, "periodic_task_add": {}, "periodic_run": {}, "periodic_millis": {}, "periodic_tick_ms": {},
	"periodic_set_critical_section": {}, "true": {}, "false": {},
}

var cKeywordIdentifiers = map[string]struct{}{
	"auto": {}, "break": {}, "case": {}, "char": {}, "const": {}, "continue": {}, "default": {}, "do": {},
	"double": {}, "else": {}, "enum": {}, "extern": {}, "float": {}, "for": {}, "goto": {}, "if": {},
	"inline": {}, "int": {}, "long": {}, "register": {}, "restrict": {}, "return": {}, "short": {}, "signed": {},
	"sizeof": {}, "static": {}, "struct": {}, "switch": {}, "typedef": {}, "union": {}, "unsigned": {}, "void": {},
	"volatile": {}, "while": {}, "_Bool": {}, "_Complex": {}, "_Imaginary": {},
}

func escape(s string) string {
	e := strings.ReplaceAll(s, "\\", "\\\\")
	e = strings.ReplaceAll(e, "\"", "\\\"")
	return e
}

func renderParams(params []string) string {
	if len(params) == 0 {
		return "void"
	}
	list := make([]string, 0, len(params))
	for _, p := range params {
		list = append(list, "int "+sanitizeName(p))
	}
	return strings.Join(list, ", ")
}

func renderTypedParams(params []string, types []string, symbols []string) string {
	if len(params) == 0 {
		return "void"
	}
	list := make([]string, 0, len(params))
	for i, param := range params {
		cType := "int"
		if i < len(types) && types[i] != "" {
			cType = types[i]
		}
		symbol := ""
		if i < len(symbols) {
			symbol = symbols[i]
		}
		list = append(list, cType+" "+sanitizeBinding(symbol, param))
	}
	return strings.Join(list, ", ")
}

func isStaticInitSafe(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.NumberLiteral, *ast.StringLiteral, *ast.BoolLiteral:
		return true
	case *ast.UnaryExpr:
		return isStaticInitSafe(node.Expr)
	case *ast.BinaryExpr:
		return isStaticInitSafe(node.Left) && isStaticInitSafe(node.Right)
	case *ast.ObjectLiteralExpr:
		for _, property := range node.Properties {
			if !isStaticInitSafe(property.Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func taskHandlerName(taskName string) string {
	return "jikascript_task_" + normalizeIdentifier(taskName)
}

func shouldSkipExprStmt(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Function.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	object, ok := selector.Object.(*ast.Identifier)
	if !ok {
		return false
	}
	return object.Name == "scheduler" && selector.Field == "start"
}

func renderSelectorCall(call *ast.CallExpr) string {
	selector, ok := call.Function.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	object := renderExpr(selector.Object)
	args := make([]string, 0, len(call.Args))
	for _, arg := range call.Args {
		args = append(args, renderExpr(arg))
	}
	if ident, ok := selector.Object.(*ast.Identifier); ok {
		if meta, exists := generatedArrays[ident.Name]; exists {
			switch selector.Field {
			case "push":
				if len(args) == 0 {
					return sanitizeName(meta.Name) + "_push(0)"
				}
				return sanitizeName(meta.Name) + "_push(" + strings.Join(args, ", ") + ")"
			case "pop":
				return sanitizeName(meta.Name) + "_pop()"
			}
		}
	}
	switch selector.Field {
	case "output":
		return "gpio_output(" + object + ")"
	case "input":
		return "gpio_input(" + object + ")"
	case "toggle":
		return "gpio_toggle(" + object + ")"
	case "read":
		return "gpio_read(" + object + ")"
	case "write":
		if len(args) == 0 {
			return "gpio_write(" + object + ", 0)"
		}
		return "gpio_write(" + object + ", " + strings.Join(args, ", ") + ")"
	default:
		return ""
	}
}

func renderForInit(stmt ast.Stmt) string {
	if stmt == nil {
		return ""
	}
	switch node := stmt.(type) {
	case *ast.VarDecl:
		return typeName(effectiveType(node.Type, node.ResolvedType), false) + " " + sanitizeBinding(node.Symbol, node.Name) + " = " + renderExpr(node.Value)
	case *ast.ConstDecl:
		return "const " + typeName(effectiveType(node.Type, node.ResolvedType), true) + " " + sanitizeBinding(node.Symbol, node.Name) + " = " + renderExpr(node.Value)
	case *ast.ExprStmt:
		return renderExpr(node.Expr)
	default:
		return ""
	}
}

func renderOptionalExpr(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	return renderExpr(expr)
}

func renderArrayDecl(meta arrayMeta) string {
	name := sanitizeName(meta.Name)
	return fmt.Sprintf("static struct { uint32_t length; uint32_t capacity; %s data[%d]; } %s;\n", meta.CType, meta.Capacity, name)
}

func renderArrayHelpers(meta arrayMeta) string {
	name := sanitizeName(meta.Name)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static %s %s_get(int32_t index) {\n", meta.CType, name))
	b.WriteString(fmt.Sprintf("    if (index < 0 || (uint32_t)index >= %s.length) { jikascript_array_fail(); }\n", name))
	b.WriteString(fmt.Sprintf("    return %s.data[index];\n", name))
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("static void %s_set(int32_t index, %s value) {\n", name, meta.CType))
	b.WriteString(fmt.Sprintf("    if (index < 0 || (uint32_t)index >= %s.length) { jikascript_array_fail(); }\n", name))
	b.WriteString(fmt.Sprintf("    %s.data[index] = value;\n", name))
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("static void %s_push(%s value) {\n", name, meta.CType))
	b.WriteString(fmt.Sprintf("    if (%s.length >= %s.capacity) { jikascript_array_fail(); }\n", name, name))
	b.WriteString(fmt.Sprintf("    %s.data[%s.length] = value;\n", name, name))
	b.WriteString(fmt.Sprintf("    %s.length++;\n", name))
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("static %s %s_pop(void) {\n", meta.CType, name))
	b.WriteString(fmt.Sprintf("    if (%s.length == 0) { jikascript_array_fail(); }\n", name))
	b.WriteString(fmt.Sprintf("    %s.length--;\n", name))
	b.WriteString(fmt.Sprintf("    return %s.data[%s.length];\n", name, name))
	b.WriteString("}\n\n")
	return b.String()
}

func buildArrayDeclFromVar(node *ast.VarDecl) (arrayMeta, []string, bool) {
	return buildArrayDecl(node.Name, node.Type, node.ArrayCapacity, node.Value)
}

func buildArrayDeclFromConst(node *ast.ConstDecl) (arrayMeta, []string, bool) {
	return buildArrayDecl(node.Name, node.Type, node.ArrayCapacity, node.Value)
}

func buildArrayDecl(name, declaredType string, declaredCapacity int, value ast.Expr) (arrayMeta, []string, bool) {
	lit, ok := value.(*ast.ArrayLiteralExpr)
	if !ok && declaredCapacity <= 0 {
		return arrayMeta{}, nil, false
	}

	elements := []ast.Expr{}
	if lit != nil {
		elements = lit.Elements
	}
	elemType := declaredType
	if elemType == "" {
		elemType = inferArrayElementType(elements)
	}
	if elemType == "" {
		elemType = "int32"
	}
	capacity := declaredCapacity
	if capacity <= 0 {
		capacity = len(elements)
	}
	if capacity <= 0 {
		return arrayMeta{}, nil, false
	}
	meta := arrayMeta{
		Name:     name,
		ElemType: elemType,
		CType:    typeName(elemType, false),
		Capacity: capacity,
	}
	init := []string{
		fmt.Sprintf("%s.capacity = %d", sanitizeName(name), capacity),
		fmt.Sprintf("%s.length = %d", sanitizeName(name), len(elements)),
	}
	for i, elem := range elements {
		init = append(init, fmt.Sprintf("%s.data[%d] = %s", sanitizeName(name), i, renderExpr(elem)))
	}
	return meta, init, true
}

func inferArrayElementType(elements []ast.Expr) string {
	if len(elements) == 0 {
		return ""
	}
	switch first := elements[0].(type) {
	case *ast.NumberLiteral:
		if strings.Contains(first.Value, ".") {
			return "float32"
		}
		return "int32"
	case *ast.StringLiteral:
		return "string"
	case *ast.BoolLiteral:
		return "boolean"
	case *ast.ObjectLiteralExpr:
		if meta, ok := objectDeclMeta(first); ok {
			return meta.CName
		}
	default:
		return "int32"
	}
	return "int32"
}

// walkExpr visits every sub-expression of expr, including expr itself.
func walkExpr(expr ast.Expr, visit func(ast.Expr)) {
	if expr == nil {
		return
	}
	visit(expr)
	walkExprChildren(expr, visit)
}

// walkExprChildren visits the child expressions of expr without visiting
// expr itself.
func walkExprChildren(expr ast.Expr, visit func(ast.Expr)) {
	if expr == nil {
		return
	}
	switch node := expr.(type) {
	case *ast.ObjectLiteralExpr:
		for _, property := range node.Properties {
			walkExpr(property.Value, visit)
		}
	case *ast.ArrayLiteralExpr:
		for _, element := range node.Elements {
			walkExpr(element, visit)
		}
	case *ast.CallExpr:
		walkExpr(node.Function, visit)
		for _, arg := range node.Args {
			walkExpr(arg, visit)
		}
	case *ast.SelectorExpr:
		walkExpr(node.Object, visit)
	case *ast.IndexExpr:
		walkExpr(node.Object, visit)
		walkExpr(node.Index, visit)
	case *ast.BinaryExpr:
		walkExpr(node.Left, visit)
		walkExpr(node.Right, visit)
	case *ast.UnaryExpr:
		walkExpr(node.Expr, visit)
	case *ast.ArrowFunctionExpr:
		for _, s := range node.Body {
			walkStmt(s, visit)
		}
	}
}

// walkStmt visits every sub-statement and sub-expression of stmt.
func walkStmt(stmt ast.Stmt, visit func(ast.Expr)) {
	switch node := stmt.(type) {
	case nil:
		return
	case *ast.VarDecl:
		walkExpr(node.Value, visit)
	case *ast.ConstDecl:
		walkExpr(node.Value, visit)
	case *ast.ExprStmt:
		walkExpr(node.Expr, visit)
	case *ast.AssignStmt:
		walkExpr(node.Value, visit)
	case *ast.ReturnStmt:
		walkExpr(node.Value, visit)
	case *ast.FunctionDecl:
		for _, s := range node.Body {
			walkStmt(s, visit)
		}
	case *ast.TaskDecl:
		walkExpr(node.Interval, visit)
		if node.Callback != nil {
			for _, s := range node.Callback.Body {
				walkStmt(s, visit)
			}
		}
	case *ast.IfStmt:
		walkExpr(node.Cond, visit)
		for _, s := range node.Then {
			walkStmt(s, visit)
		}
		for _, s := range node.Else {
			walkStmt(s, visit)
		}
	case *ast.WhileStmt:
		walkExpr(node.Cond, visit)
		for _, s := range node.Body {
			walkStmt(s, visit)
		}
	case *ast.DoWhileStmt:
		for _, s := range node.Body {
			walkStmt(s, visit)
		}
		walkExpr(node.Cond, visit)
	case *ast.ForStmt:
		if node.Init != nil {
			walkStmt(node.Init, visit)
		}
		walkExpr(node.Cond, visit)
		for _, s := range node.Body {
			walkStmt(s, visit)
		}
		walkExpr(node.Post, visit)
	case *ast.SwitchStmt:
		walkExpr(node.Expr, visit)
		for _, c := range node.Cases {
			walkExpr(c.Value, visit)
			for _, s := range c.Body {
				walkStmt(s, visit)
			}
		}
	case *ast.BlockStmt:
		for _, s := range node.Statements {
			walkStmt(s, visit)
		}
	case *ast.BreakStmt, *ast.ContinueStmt:
		// leaf statements
	}
}

func collectObjectTypes(program *ast.Program) {
	var register func(ast.Expr)
	register = func(expr ast.Expr) {
		if lit, ok := expr.(*ast.ObjectLiteralExpr); ok {
			shape := objectShapeForLiteral(lit)
			if _, exists := generatedObjects[shape]; !exists {
				generatedObjects[shape] = objectMeta{Shape: shape, CName: fmt.Sprintf("jikascript_object_%d_t", len(generatedObjects)+1), Fields: lit.Properties}
			}
		}
	}
	for _, stmt := range program.Statements {
		walkStmt(stmt, register)
	}
}

func objectShapeForLiteral(lit *ast.ObjectLiteralExpr) string {
	fields := make([]string, 0, len(lit.Properties))
	for _, property := range lit.Properties {
		fields = append(fields, property.Name+":"+objectFieldType(property.Value))
	}
	return strings.Join(fields, ",")
}

func objectFieldType(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.NumberLiteral:
		if strings.Contains(node.Value, ".") {
			return "float32"
		}
		return "int32"
	case *ast.StringLiteral:
		return "string"
	case *ast.BoolLiteral:
		return "boolean"
	case *ast.ObjectLiteralExpr:
		if meta, ok := objectDeclMeta(node); ok {
			return meta.CName
		}
	}
	return "int32"
}

func functionReturnCType(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	ident, ok := call.Function.(*ast.Identifier)
	if !ok {
		return "", false
	}
	meta, exists := generatedFunctions[ident.Name]
	return meta.ReturnType, exists && meta.ReturnType != ""
}

func objectDeclMeta(expr ast.Expr) (objectMeta, bool) {
	lit, ok := expr.(*ast.ObjectLiteralExpr)
	if !ok {
		return objectMeta{}, false
	}
	meta, exists := generatedObjects[objectShapeForLiteral(lit)]
	return meta, exists
}

func renderObjectType(meta objectMeta) string {
	var b strings.Builder
	b.WriteString("typedef struct {\n")
	for _, field := range meta.Fields {
		b.WriteString(fmt.Sprintf("    %s %s;\n", typeName(objectFieldType(field.Value), false), sanitizeFieldName(field.Name)))
	}
	b.WriteString(fmt.Sprintf("} %s;\n", meta.CName))
	return b.String()
}

func renderObjectTopLevelDecl(meta objectMeta, name, symbol string, value ast.Expr) string {
	if isStaticInitSafe(value) {
		return fmt.Sprintf("static %s %s = %s;\n", meta.CName, sanitizeBinding(symbol, name), renderExpr(value))
	}
	return fmt.Sprintf("static %s %s;\n", meta.CName, sanitizeBinding(symbol, name))
}

func collectFunctionTypes(program *ast.Program) {
	for _, stmt := range program.Statements {
		fn, ok := stmt.(*ast.FunctionDecl)
		if !ok {
			continue
		}
		sanitizeBinding(fn.Symbol, fn.Name)
		meta := functionMeta{ParamTypes: make([]string, len(fn.Params))}
		for i, paramType := range fn.ParamTypes {
			meta.ParamTypes[i] = resolvedCType(paramType)
		}
		if fn.ReturnType != "" {
			meta.ReturnType = resolvedCType(fn.ReturnType)
		}
		for _, bodyStmt := range fn.Body {
			if ret, ok := bodyStmt.(*ast.ReturnStmt); ok {
				if meta.ReturnType == "" {
					if object, ok := objectDeclMeta(ret.Value); ok {
						meta.ReturnType = object.CName
					}
				}
				if object, ok := objectDeclMeta(ret.Value); ok && meta.ReturnType == "" {
					meta.ReturnType = object.CName
				}
			}
		}
		generatedFunctions[fn.Name] = meta
	}
	var visit func(ast.Expr)
	visit = func(expr ast.Expr) {
		if call, ok := expr.(*ast.CallExpr); ok {
			if ident, ok := call.Function.(*ast.Identifier); ok {
				if meta, exists := generatedFunctions[ident.Name]; exists {
					for i, arg := range call.Args {
						if i < len(meta.ParamTypes) {
							if object, ok := objectDeclMeta(arg); ok {
								meta.ParamTypes[i] = object.CName
							}
							if argIdent, ok := arg.(*ast.Identifier); ok {
								if cType, exists := generatedObjectBindings[argIdent.Name]; exists {
									meta.ParamTypes[i] = cType
								}
							}
						}
					}
					generatedFunctions[ident.Name] = meta
				}
			}
		}
		walkExprChildren(expr, visit)
	}
	for _, stmt := range program.Statements {
		walkStmt(stmt, visit)
	}
}

func resolvedCType(typ string) string {
	if strings.Contains(typ, ":") {
		if meta, ok := generatedObjects[typ]; ok {
			return meta.CName
		}
		return "int"
	}
	return typeName(typ, false)
}

func collectObjectBindings(program *ast.Program) {
	var visit func(ast.Expr)
	visit = func(expr ast.Expr) {
		if call, ok := expr.(*ast.CallExpr); ok {
			if ident, ok := call.Function.(*ast.Identifier); ok {
				if meta, exists := generatedFunctions[ident.Name]; exists {
					for i, arg := range call.Args {
						if i < len(meta.ParamTypes) {
							if argIdent, ok := arg.(*ast.Identifier); ok {
								if cType, exists := generatedObjectBindings[argIdent.Name]; exists {
									meta.ParamTypes[i] = cType
								}
							}
						}
					}
					generatedFunctions[ident.Name] = meta
				}
			}
		}
		walkExprChildren(expr, visit)
	}
	for _, stmt := range program.Statements {
		switch node := stmt.(type) {
		case *ast.VarDecl:
			if meta, ok := objectDeclMeta(node.Value); ok {
				generatedObjectBindings[node.Name] = meta.CName
			}
		case *ast.ConstDecl:
			if meta, ok := objectDeclMeta(node.Value); ok {
				generatedObjectBindings[node.Name] = meta.CName
			}
		}
		walkStmt(stmt, visit)
	}
}

func isAssignmentOp(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=":
		return true
	default:
		return false
	}
}
