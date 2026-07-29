package selectstatement

import (
	"go/ast"

	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type clause struct {
	source        *ast.CommClause
	selection     int
	alternative   tsgo.Identifier
	receive       *ast.UnaryExpr
	assignment    *ast.AssignStmt
	receiveResult tsgo.Identifier
	channel       channelmodel.Model
}

type prepared struct {
	clauses      []clause
	alternatives []tsgo.Expression
	before       []tsgo.Statement
	hasDefault   bool
}
