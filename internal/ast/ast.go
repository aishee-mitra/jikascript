package ast

// Position tracks source location information for diagnostics.
type Position struct {
	Line   int
	Column int
}

// Program is the root node for a source file.
type Program struct {
	Statements []Stmt
}

// Stmt is implemented by all statement nodes.
type Stmt interface{ stmtNode() }

// Expr is implemented by all expression nodes.
type Expr interface{ exprNode() }

// VarDecl describes a mutable binding.
type VarDecl struct {
	Name  string
	Type  string
	Value Expr
	Pos   Position
}

func (*VarDecl) stmtNode() {}

// ConstDecl describes an immutable binding.
type ConstDecl struct {
	Name  string
	Type  string
	Value Expr
	Pos   Position
}

func (*ConstDecl) stmtNode() {}

// FunctionDecl describes a named function.
type FunctionDecl struct {
	Name   string
	Params []string
	Body   []Stmt
	Pos    Position
}

func (*FunctionDecl) stmtNode() {}

// ReturnStmt describes a return statement.
type ReturnStmt struct {
	Value Expr
	Pos   Position
}

func (*ReturnStmt) stmtNode() {}

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Expr Expr
	Pos  Position
}

func (*ExprStmt) stmtNode() {}

// AssignStmt describes assignment to an existing binding.
type AssignStmt struct {
	Target string
	Value  Expr
	Pos    Position
}

func (*AssignStmt) stmtNode() {}

// IfStmt describes a conditional block.
type IfStmt struct {
	Cond Expr
	Then []Stmt
	Else []Stmt
	Pos  Position
}

func (*IfStmt) stmtNode() {}

// WhileStmt describes a while loop.
type WhileStmt struct {
	Cond Expr
	Body []Stmt
	Pos  Position
}

func (*WhileStmt) stmtNode() {}

// DoWhileStmt describes a do-while loop.
type DoWhileStmt struct {
	Body []Stmt
	Cond Expr
	Pos  Position
}

func (*DoWhileStmt) stmtNode() {}

// ForStmt describes a for loop.
type ForStmt struct {
	Init Stmt
	Cond Expr
	Post Expr
	Body []Stmt
	Pos  Position
}

func (*ForStmt) stmtNode() {}

// TaskDecl describes the periodic task syntax.
type TaskDecl struct {
	Name     string
	Interval Expr
	Callback *ArrowFunctionExpr
	Pos      Position
}

func (*TaskDecl) stmtNode() {}

// BlockStmt groups statements in a block.
type BlockStmt struct {
	Statements []Stmt
	Pos        Position
}

func (*BlockStmt) stmtNode() {}

// ArrowFunctionExpr describes a callback or anonymous function.
type ArrowFunctionExpr struct {
	Params []string
	Body   []Stmt
	Pos    Position
}

func (*ArrowFunctionExpr) exprNode() {}

// Identifier references a variable or function.
type Identifier struct {
	Name string
	Pos  Position
}

func (*Identifier) exprNode() {}

// NumberLiteral is a numeric literal.
type NumberLiteral struct {
	Value string
	Pos   Position
}

func (*NumberLiteral) exprNode() {}

// StringLiteral is a static string literal.
type StringLiteral struct {
	Value string
	Pos   Position
}

func (*StringLiteral) exprNode() {}

// BoolLiteral is a boolean literal.
type BoolLiteral struct {
	Value bool
	Pos   Position
}

func (*BoolLiteral) exprNode() {}

// CallExpr represents a function call or method invocation.
type CallExpr struct {
	Function Expr
	Args     []Expr
	Pos      Position
}

func (*CallExpr) exprNode() {}

// SelectorExpr represents property access such as `led.toggle`.
type SelectorExpr struct {
	Object Expr
	Field  string
	Pos    Position
}

func (*SelectorExpr) exprNode() {}

// BinaryExpr represents infix expression.
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
	Pos   Position
}

func (*BinaryExpr) exprNode() {}

// UnaryExpr represents prefix expression.
type UnaryExpr struct {
	Op   string
	Expr Expr
	Pos  Position
}

func (*UnaryExpr) exprNode() {}
