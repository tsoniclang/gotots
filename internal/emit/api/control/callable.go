package control

import (
	"go/ast"
	"slices"
)

type CallableFacet uint8

const (
	CallableInvalid CallableFacet = iota
	CallableDefer
	CallableRecovery
	CallableGoto
	CallableIteratorReturn
)

func (f CallableFacet) Valid() bool {
	return f == CallableDefer ||
		f == CallableRecovery ||
		f == CallableGoto ||
		f == CallableIteratorReturn
}

type CallableDemand struct {
	deferEnvelope   bool
	unlocatedDefer  bool
	deferSites      map[*ast.DeferStmt]struct{}
	recovery        bool
	gotoControl     bool
	iteratorReturns map[*ast.RangeStmt]struct{}
}

func (d CallableDemand) With(facet CallableFacet) CallableDemand {
	switch facet {
	case CallableDefer:
		d.deferEnvelope = true
		d.unlocatedDefer = true
	case CallableRecovery:
		d.recovery = true
	case CallableGoto:
		d.gotoControl = true
	case CallableIteratorReturn:
		panic("iterator-return control requires an exact range")
	default:
		panic("callable-control facet is invalid")
	}
	return d
}

func (d CallableDemand) WithDefer(source *ast.DeferStmt) CallableDemand {
	if source == nil || source.Call == nil {
		panic("defer control requires an exact statement")
	}
	selected := make(map[*ast.DeferStmt]struct{}, len(d.deferSites)+1)
	for existing := range d.deferSites {
		selected[existing] = struct{}{}
	}
	selected[source] = struct{}{}
	d.deferEnvelope = true
	d.deferSites = selected
	return d
}

func (d CallableDemand) WithIteratorReturn(
	source *ast.RangeStmt,
) CallableDemand {
	if source == nil {
		panic("iterator-return range is nil")
	}
	selected := make(map[*ast.RangeStmt]struct{}, len(d.iteratorReturns)+1)
	for existing := range d.iteratorReturns {
		selected[existing] = struct{}{}
	}
	selected[source] = struct{}{}
	d.iteratorReturns = selected
	return d
}

func (d CallableDemand) Has(facet CallableFacet) bool {
	switch facet {
	case CallableDefer:
		return d.deferEnvelope
	case CallableRecovery:
		return d.recovery
	case CallableGoto:
		return d.gotoControl
	case CallableIteratorReturn:
		return len(d.iteratorReturns) != 0
	default:
		return false
	}
}

func (d CallableDemand) Defer() bool {
	return d.deferEnvelope
}

func (d CallableDemand) Recovery() bool {
	return d.recovery
}

func (d CallableDemand) Goto() bool {
	return d.gotoControl
}

func (d CallableDemand) IteratorReturn(source *ast.RangeStmt) bool {
	_, selected := d.iteratorReturns[source]
	return selected
}

func (d CallableDemand) ExactDefers(
	direct []*ast.DeferStmt,
) ([]*ast.DeferStmt, bool) {
	if !d.deferEnvelope ||
		d.unlocatedDefer ||
		len(d.deferSites) == 0 ||
		len(d.deferSites) != len(direct) {
		return nil, false
	}
	for _, source := range direct {
		if _, selected := d.deferSites[source]; !selected {
			return nil, false
		}
	}
	return slices.Clone(direct), true
}
