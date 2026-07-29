package sequence

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type blockLabel struct {
	source *ast.LabeledStmt
	object *types.Label
	name   string
	index  int
}

type directGotoInterval struct {
	label       blockLabel
	target      api.GotoTarget
	direction   api.GotoTargetKind
	first       int
	last        int
	retainLabel bool
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	owner ast.Node,
	statements []ast.Stmt,
) (api.StatementEmission, error) {
	if owner == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, owner)
	}
	if !context.CallableControl().Goto() {
		return emitDirect(context, children, statements)
	}
	labels, err := directLabels(context, statements)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if len(labels) == 0 {
		return emitDirect(context, children, statements)
	}
	intervals := make([]directGotoInterval, 0, len(labels))
	for _, label := range labels {
		interval, ok := directGoto(context, owner, statements, label)
		if !ok {
			return emitStateGoto(
				context,
				children,
				owner,
				statements,
				labels,
			)
		}
		intervals = append(intervals, interval)
	}
	sort.Slice(intervals, func(left, right int) bool {
		if intervals[left].first != intervals[right].first {
			return intervals[left].first < intervals[right].first
		}
		return intervals[left].last < intervals[right].last
	})
	for index := 1; index < len(intervals); index++ {
		if intervals[index].first <= intervals[index-1].last {
			return emitStateGoto(
				context,
				children,
				owner,
				statements,
				labels,
			)
		}
	}
	for index := range intervals {
		targetName := intervals[index].label.name
		if intervals[index].retainLabel {
			targetName, err = context.Names().Temporary(
				api.TemporaryGotoTarget,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		intervals[index].target, err = api.NewDirectGotoTarget(
			intervals[index].direction,
			targetName,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	return emitDirectGoto(context, children, statements, intervals)
}

func directLabels(
	context api.Context,
	statements []ast.Stmt,
) ([]blockLabel, error) {
	var labels []blockLabel
	for index, statement := range statements {
		labeled, ok := statement.(*ast.LabeledStmt)
		if !ok {
			continue
		}
		object, ok := context.TypesInfo().Defs[labeled.Label].(*types.Label)
		if !ok {
			return nil, api.Unsupported(
				context,
				api.CategoryStatement,
				labeled,
			)
		}
		if len(context.GotoUses(object)) == 0 {
			continue
		}
		name, err := context.Names().Declare(object)
		if err != nil {
			return nil, err
		}
		labels = append(labels, blockLabel{
			source: labeled,
			object: object,
			name:   name,
			index:  index,
		})
	}
	return labels, nil
}

func directGoto(
	context api.Context,
	owner ast.Node,
	statements []ast.Stmt,
	label blockLabel,
) (directGotoInterval, bool) {
	positions := context.GotoUses(label.object)
	if len(positions) == 0 {
		return directGotoInterval{}, false
	}
	before := true
	after := true
	first := len(statements)
	last := -1
	for _, position := range positions {
		if position < owner.Pos() || position > owner.End() {
			return directGotoInterval{}, false
		}
		index := containingStatement(statements, position)
		if index < 0 {
			return directGotoInterval{}, false
		}
		if position >= label.source.Pos() {
			before = false
		}
		if position <= label.source.Pos() {
			after = false
		}
		if index < first {
			first = index
		}
		if index > last {
			last = index
		}
	}
	var direction api.GotoTargetKind
	switch {
	case before:
		direction = api.GotoTargetBreak
		last = label.index
	case after:
		direction = api.GotoTargetContinue
		first = label.index
	default:
		return directGotoInterval{}, false
	}
	return directGotoInterval{
		label:       label,
		direction:   direction,
		first:       first,
		last:        last,
		retainLabel: hasNonGotoLabelUse(context, label),
	}, true
}

func hasNonGotoLabelUse(
	context api.Context,
	label blockLabel,
) bool {
	gotoPositions := make(map[token.Pos]struct{})
	for _, position := range context.GotoUses(label.object) {
		gotoPositions[position] = struct{}{}
	}
	for identifier, object := range context.TypesInfo().Uses {
		if object != label.object {
			continue
		}
		if _, gotoUse := gotoPositions[identifier.Pos()]; !gotoUse {
			return true
		}
	}
	return false
}

func containingStatement(
	statements []ast.Stmt,
	position token.Pos,
) int {
	for index, statement := range statements {
		if position >= statement.Pos() && position <= statement.End() {
			return index
		}
	}
	return -1
}

func emitDirectGoto(
	context api.Context,
	children api.ChildEmitter,
	sourceStatements []ast.Stmt,
	intervals []directGotoInterval,
) (api.StatementEmission, error) {
	selected := context
	for _, interval := range intervals {
		selected = selected.WithGotoTarget(
			interval.label.object,
			interval.target,
		)
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	cursor := 0
	for _, interval := range intervals {
		prefix, prefixRequests, err := emitGotoStatementRange(
			selected,
			children,
			sourceStatements,
			cursor,
			interval.first,
			-1,
			nil,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, prefix...)
		requests = append(requests, prefixRequests...)

		replacementIndex := -1
		var replacement ast.Stmt
		if !interval.retainLabel {
			replacementIndex = interval.label.index
			replacement = interval.label.source.Stmt
		}
		if interval.direction == api.GotoTargetBreak {
			body, bodyRequests, err := emitGotoStatementRange(
				selected,
				children,
				sourceStatements,
				interval.first,
				interval.label.index,
				-1,
				nil,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(
				statements,
				context.Factory().LabeledStatement(
					context.Factory().Identifier(
						interval.target.Label(),
					),
					context.Factory().Block(body, true),
				),
			)
			requests = append(requests, bodyRequests...)
			target, targetRequests, err := emitGotoStatementRange(
				selected,
				children,
				sourceStatements,
				interval.label.index,
				interval.label.index+1,
				replacementIndex,
				replacement,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, target...)
			requests = append(requests, targetRequests...)
		} else {
			body, bodyRequests, err := emitGotoStatementRange(
				selected,
				children,
				sourceStatements,
				interval.label.index,
				interval.last+1,
				replacementIndex,
				replacement,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			body = append(
				body,
				context.Factory().BreakStatement(
					context.Factory().Identifier(
						interval.target.Label(),
					),
				),
			)
			statements = append(
				statements,
				context.Factory().LabeledStatement(
					context.Factory().Identifier(
						interval.target.Label(),
					),
					context.Factory().WhileStatement(
						context.Factory().TrueLiteral(),
						context.Factory().Block(body, true),
					),
				),
			)
			requests = append(requests, bodyRequests...)
		}
		cursor = interval.last + 1
	}
	suffix, suffixRequests, err := emitGotoStatementRange(
		selected,
		children,
		sourceStatements,
		cursor,
		len(sourceStatements),
		-1,
		nil,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements = append(statements, suffix...)
	requests = append(requests, suffixRequests...)
	return api.NewStatementEmission(statements, requests)
}

func emitDirect(
	context api.Context,
	children api.ChildEmitter,
	statements []ast.Stmt,
) (api.StatementEmission, error) {
	target, requests, err := emitGotoStatementRange(
		context,
		children,
		statements,
		0,
		len(statements),
		-1,
		nil,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(target, requests)
}

func emitGotoStatementRange(
	context api.Context,
	children api.ChildEmitter,
	statements []ast.Stmt,
	first int,
	last int,
	replacementIndex int,
	replacement ast.Stmt,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var target []tsgo.Statement
	var requests []api.RootRequest
	for index := first; index < last; index++ {
		statement := statements[index]
		if index == replacementIndex {
			statement = replacement
		}
		emission, err := children.Statement(
			context,
			statement,
		)
		if err != nil {
			return nil, nil, err
		}
		target = append(target, emission.Statements()...)
		requests = append(requests, emission.Requests()...)
	}
	return target, requests, nil
}
