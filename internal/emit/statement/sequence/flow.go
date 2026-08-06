package sequence

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func FallsThrough(statements []tsgo.Statement) bool {
	for _, statement := range statements {
		if !targetStatementFallsThrough(statement) {
			return false
		}
	}
	return true
}

func targetStatementFallsThrough(statement tsgo.Statement) bool {
	switch statement := statement.(type) {
	case tsgo.ReturnStatement,
		tsgo.ThrowStatement,
		tsgo.BreakStatement,
		tsgo.ContinueStatement:
		return false
	case tsgo.Block:
		return FallsThrough(statement.Statements())
	case tsgo.LabeledStatement:
		return targetStatementFallsThrough(statement.Statement())
	case tsgo.ForStatement:
		return statement.Condition() != nil
	case tsgo.IfStatement:
		return statement.ElseStatement() == nil ||
			targetStatementFallsThrough(statement.ThenStatement()) ||
			targetStatementFallsThrough(statement.ElseStatement())
	default:
		return true
	}
}
