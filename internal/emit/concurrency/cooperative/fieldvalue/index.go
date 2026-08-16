package fieldvalue

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/type/fieldidentity"
	"github.com/tsoniclang/gotots/internal/load"
)

type packageIndex struct {
	owner       *types.Package
	closed      bool
	declared    map[*types.Var]struct{}
	assignments map[*types.Var][]ast.Expr
	invalid     map[*types.Var]struct{}
	addressed   map[*types.Var]struct{}
	canonical   map[*types.Var]*types.Var
}

type Index struct {
	packages map[*types.Package]*packageIndex
}

func New(program *load.Program) *Index {
	index := &Index{packages: make(map[*types.Package]*packageIndex)}
	if program == nil {
		return index
	}
	for _, source := range program.Packages() {
		if source == nil || source.Types() == nil || source.TypesInfo() == nil {
			continue
		}
		selected := &packageIndex{
			owner:       source.Types(),
			closed:      len(source.OtherFiles()) == 0 && !importsOpaque(source.Types()),
			declared:    make(map[*types.Var]struct{}),
			assignments: make(map[*types.Var][]ast.Expr),
			invalid:     make(map[*types.Var]struct{}),
			addressed:   make(map[*types.Var]struct{}),
			canonical:   make(map[*types.Var]*types.Var),
		}
		index.packages[source.Types()] = selected
		if !selected.closed {
			continue
		}
		selected.collect(source)
		selected.sortAssignments()
	}
	return index
}

func (p *packageIndex) sortAssignments() {
	for field := range p.assignments {
		sort.Slice(p.assignments[field], func(left, right int) bool {
			leftValue := p.assignments[field][left]
			rightValue := p.assignments[field][right]
			if leftValue.Pos() != rightValue.Pos() {
				return leftValue.Pos() < rightValue.Pos()
			}
			return leftValue.End() < rightValue.End()
		})
	}
}

func (i *Index) Assignments(field *types.Var) ([]ast.Expr, bool) {
	if i == nil || field == nil || field.Pkg() == nil || field.Exported() ||
		!plainFunctionType(field.Type()) {
		return nil, false
	}
	selected := i.packages[field.Pkg()]
	if selected == nil || !selected.closed {
		return nil, false
	}
	declaration := selected.canonical[field]
	if declaration == nil {
		return nil, false
	}
	if _, invalid := selected.invalid[declaration]; invalid {
		return nil, false
	}
	if _, addressed := selected.addressed[declaration]; addressed {
		return nil, false
	}
	if _, declared := selected.declared[declaration]; !declared {
		return nil, false
	}
	return slices.Clone(selected.assignments[declaration]), true
}

func (p *packageIndex) collect(source *load.Package) {
	if source == nil || source.TypesInfo() == nil {
		p.closed = false
		return
	}
	info := source.TypesInfo()
	for _, object := range info.Defs {
		field, ok := object.(*types.Var)
		if ok && field.IsField() && field.Pkg() == p.owner &&
			!field.Exported() && plainFunctionType(field.Type()) {
			p.declared[field] = struct{}{}
			p.canonical[field] = field
		}
	}
	for selector, selection := range info.Selections {
		field, _ := selection.Obj().(*types.Var)
		if selection.Kind() != types.FieldVal || field == nil {
			continue
		}
		declaration, ok := p.declarationField(selection.Recv(), field)
		if ok {
			p.selectedFieldUse(source, selector, declaration)
		}
	}
	for expression := range info.Types {
		literal, ok := expression.(*ast.CompositeLit)
		if ok {
			p.compositeLiteral(literal, info)
		}
	}
}

func (p *packageIndex) selectedFieldUse(
	source *load.Package,
	selector *ast.SelectorExpr,
	field *types.Var,
) {
	var selected ast.Expr = selector
	parent, ok := source.SyntaxParent(selected)
	for ok {
		wrapped, wrappedOK := parent.(*ast.ParenExpr)
		if !wrappedOK || wrapped.X != selected {
			break
		}
		selected = wrapped
		parent, ok = source.SyntaxParent(selected)
	}
	if !ok {
		p.invalid[field] = struct{}{}
		return
	}
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		for index, target := range parent.Lhs {
			if target != selected {
				continue
			}
			if parent.Tok != token.ASSIGN || len(parent.Lhs) != len(parent.Rhs) {
				p.invalid[field] = struct{}{}
				return
			}
			p.record(field, parent.Rhs[index])
			return
		}
	case *ast.RangeStmt:
		if parent.Key == selected || parent.Value == selected {
			p.invalid[field] = struct{}{}
		}
	case *ast.UnaryExpr:
		if parent.Op == token.AND && parent.X == selected {
			p.addressed[field] = struct{}{}
		}
	}
}

func (p *packageIndex) compositeLiteral(
	source *ast.CompositeLit,
	info *types.Info,
) {
	container := info.TypeOf(source)
	structure := structType(container)
	if structure == nil {
		return
	}
	for index, element := range source.Elts {
		if keyed, ok := element.(*ast.KeyValueExpr); ok {
			identifier, _ := keyed.Key.(*ast.Ident)
			field, _ := info.Uses[identifier].(*types.Var)
			if declaration, exact := p.declarationField(container, field); exact {
				p.record(declaration, keyed.Value)
			}
			continue
		}
		if index < structure.NumFields() {
			field := structure.Field(index)
			if declaration, exact := p.declarationField(container, field); exact {
				p.record(declaration, element)
			}
		}
	}
}

func (p *packageIndex) declarationField(
	container types.Type,
	field *types.Var,
) (*types.Var, bool) {
	if field == nil || field.Pkg() != p.owner || field.Exported() ||
		!plainFunctionType(field.Type()) {
		return nil, false
	}
	declaration := field
	correspondence, resolved, err := fieldidentity.Resolve(container, field)
	if err != nil {
		p.closed = false
		return nil, false
	}
	if resolved {
		declaration = correspondence.DeclarationField()
	}
	if declaration == nil || !plainFunctionType(declaration.Type()) {
		p.closed = false
		return nil, false
	}
	if _, declared := p.declared[declaration]; !declared {
		p.closed = false
		return nil, false
	}
	p.canonical[field] = declaration
	return declaration, true
}

func (p *packageIndex) record(field *types.Var, value ast.Expr) {
	if field == nil || !plainFunctionType(field.Type()) {
		return
	}
	if value == nil {
		p.invalid[field] = struct{}{}
		return
	}
	p.assignments[field] = append(p.assignments[field], value)
}

func plainFunctionType(source types.Type) bool {
	if source == nil {
		return false
	}
	_, ok := types.Unalias(source).(*types.Signature)
	return ok
}

func structType(source types.Type) *types.Struct {
	if source == nil {
		return nil
	}
	if pointer, ok := source.(*types.Pointer); ok {
		source = pointer.Elem()
	}
	structure, _ := source.Underlying().(*types.Struct)
	return structure
}

func importsOpaque(source *types.Package) bool {
	for _, imported := range source.Imports() {
		if imported != nil &&
			(imported.Path() == "unsafe" || imported.Path() == "C") {
			return true
		}
	}
	return false
}
