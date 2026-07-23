package frontend

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (index *objectIndex) createBindingCandidates() error {
	objects := map[types.Object]bool{}
	for object := range index.sourceByObject {
		objects[object] = true
	}
	for _, occurrenceID := range index.input.order {
		index.work.ImplicitBindingVisits++
		record := index.input.occurrences[occurrenceID]
		object, implicit := index.input.loaded.CheckerView().
			ImplicitOf(record.node)
		if !implicit || object == nil {
			continue
		}
		if variable, field := object.(*types.Var); field && variable.IsField() {
			continue
		}
		candidate, err := index.implicitBinding(
			occurrenceID, record.node, object,
		)
		if err != nil {
			return err
		}
		if candidate == nil {
			continue
		}
		if existing := index.bindingByObject[object]; existing != nil && existing != candidate {
			return fmt.Errorf(
				"implicit checker binding %s is duplicated",
				object.Name(),
			)
		}
		index.bindingByObject[object] = candidate
		index.bindingBySource[occurrenceID] = candidate
	}
	for object := range objects {
		if !index.isBinding(object) {
			continue
		}
		source := index.sourceByObject[object]
		if source.IsZero() && object.Pos().IsValid() {
			var err error
			source, err = index.input.index.IdentifierOccurrence(
				object.Pos(), object.Name(),
			)
			if err != nil {
				return err
			}
			if err := index.bindObjectSource(
				object, source,
			); err != nil {
				return err
			}
		}
		role := index.bindingRole(object, source)
		if !role.Valid() {
			return fmt.Errorf(
				"checker binding %s has no canonical role",
				object.Name(),
			)
		}
		scope, err := index.bindingScope(object, source)
		if err != nil {
			return err
		}
		definition := identity.DefinitionID{}
		if record := index.input.occurrences[source]; record != nil {
			definition = record.owner
		} else if record := index.input.occurrences[scope]; record != nil {
			definition = record.owner
		}
		typ := object.Type()
		if role == identity.SemanticBindingImport ||
			role == identity.SemanticBindingLabel {
			typ = nil
		}
		candidate := &bindingCandidate{
			object: object, source: source, scope: scope,
			role: role, definition: definition,
			typ: typ, name: object.Name(),
		}
		index.bindingByObject[object] = candidate
		if !source.IsZero() {
			index.bindingBySource[source] = candidate
		}
	}
	return nil
}

func (index *objectIndex) implicitBinding(
	source identity.OccurrenceID,
	node ast.Node,
	object types.Object,
) (*bindingCandidate, error) {
	context := index.contexts.byOccurrence[source]
	role := context.bindingRole
	switch object.(type) {
	case *types.PkgName:
		role = identity.SemanticBindingImport
	case *types.Var:
		if _, field := node.(*ast.Field); !field {
			role = identity.SemanticBindingTypeSwitch
		}
	}
	if !role.Valid() {
		return nil, nil
	}
	scope, err := index.scopeForOccurrence(source)
	if err != nil {
		return nil, err
	}
	typ := object.Type()
	if role == identity.SemanticBindingImport ||
		role == identity.SemanticBindingLabel {
		typ = nil
	}
	if typ == nil &&
		role != identity.SemanticBindingImport &&
		role != identity.SemanticBindingLabel {
		return nil, fmt.Errorf(
			"unnamed binding %s has no checker type", source,
		)
	}
	return &bindingCandidate{
		object:     object,
		source:     identity.OccurrenceID{},
		scope:      scope,
		role:       role,
		typ:        typ,
		name:       object.Name(),
		definition: index.input.occurrences[source].owner,
	}, nil
}

func (index *objectIndex) assignBindingIdentities() error {
	type groupKey struct {
		scope identity.OccurrenceID
		role  identity.SemanticBindingRole
	}
	groups := map[groupKey][]*bindingCandidate{}
	for _, candidate := range index.bindingByObject {
		key := groupKey{scope: candidate.scope, role: candidate.role}
		groups[key] = append(groups[key], candidate)
	}
	for occurrence, candidate := range index.bindingBySource {
		if candidate.object != nil || !candidate.source.IsZero() {
			continue
		}
		key := groupKey{scope: candidate.scope, role: candidate.role}
		groups[key] = append(groups[key], candidate)
		_ = occurrence
	}
	for _, candidates := range groups {
		sort.Slice(candidates, func(left, right int) bool {
			return bindingOrder(candidates[left]) <
				bindingOrder(candidates[right])
		})
		for ordinal, candidate := range candidates {
			candidate.ordinal = ordinal
			id, err := identity.NewSemanticBindingID(
				candidate.scope,
				candidate.source,
				candidate.role,
				ordinal,
			)
			if err != nil {
				return err
			}
			index.bindingIDs[candidate] = id
		}
	}
	return nil
}

func bindingOrder(candidate *bindingCandidate) string {
	if !candidate.source.IsZero() {
		return fmt.Sprintf(
			"%020d|%020d|%s",
			candidate.source.Span().Start(),
			candidate.source.Span().End(),
			candidate.name,
		)
	}
	position := 0
	if candidate.object != nil {
		position = int(candidate.object.Pos())
	}
	return fmt.Sprintf(
		"%020d|%s", position, candidate.name,
	)
}

func (index *objectIndex) bindingScope(
	object types.Object,
	source identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	if scope := object.Parent(); scope != nil {
		if owner := index.ownerOfCheckerScope(scope); !owner.IsZero() {
			return owner, nil
		}
	}
	if !source.IsZero() {
		scope, err := index.scopeForOccurrence(source)
		if err != nil {
			return identity.OccurrenceID{}, fmt.Errorf(
				"binding %q (%T, parent=%v, pos=%d): %w",
				object.Name(), object, object.Parent(),
				object.Pos(), err,
			)
		}
		return scope, nil
	}
	return identity.OccurrenceID{}, fmt.Errorf(
		"binding %q (%T, parent=%v, pos=%d) has no canonical lexical scope",
		object.Name(), object, object.Parent(), object.Pos(),
	)
}

func (index *objectIndex) scopeForOccurrence(
	occurrence identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	if index.scopeOccurrenceResolved[occurrence] {
		scope := index.scopeByOccurrence[occurrence]
		if scope.IsZero() {
			return identity.OccurrenceID{}, fmt.Errorf(
				"occurrence %s has no canonical lexical scope owner",
				occurrence,
			)
		}
		return scope, nil
	}
	var path []identity.OccurrenceID
	current := occurrence
	scopeOwner := identity.OccurrenceID{}
	for !current.IsZero() {
		index.work.OccurrenceScopeProbes++
		if index.scopeOccurrenceResolved[current] {
			scopeOwner = index.scopeByOccurrence[current]
			break
		}
		path = append(path, current)
		record := index.input.occurrences[current]
		if record == nil {
			break
		}
		if scope, present := index.input.loaded.CheckerView().
			ScopeOf(record.node); present &&
			index.scopeOwners[scope] == current {
			scopeOwner = current
			break
		}
		if catalog.LexicalScope(record.occurrence.Kind()) ==
			catalog.LexicalScopeAlways {
			scopeOwner = current
			break
		}
		current = record.occurrence.Parent()
	}
	for _, member := range path {
		index.scopeOccurrenceResolved[member] = true
		index.scopeByOccurrence[member] = scopeOwner
	}
	if !scopeOwner.IsZero() {
		return scopeOwner, nil
	}
	return identity.OccurrenceID{}, fmt.Errorf(
		"occurrence %s has no canonical lexical scope owner",
		occurrence,
	)
}

func (index *objectIndex) ownerOfCheckerScope(
	scope *types.Scope,
) identity.OccurrenceID {
	if scope == nil {
		return identity.OccurrenceID{}
	}
	if index.checkerScopeResolved[scope] {
		return index.checkerScopeOwner[scope]
	}
	var path []*types.Scope
	current := scope
	owner := identity.OccurrenceID{}
	for current != nil {
		index.work.CheckerScopeProbes++
		if index.checkerScopeResolved[current] {
			owner = index.checkerScopeOwner[current]
			break
		}
		path = append(path, current)
		if direct := index.scopeOwners[current]; !direct.IsZero() {
			owner = direct
			break
		}
		current = current.Parent()
	}
	for _, member := range path {
		index.checkerScopeResolved[member] = true
		index.checkerScopeOwner[member] = owner
	}
	return owner
}

func (index *objectIndex) isBinding(object types.Object) bool {
	switch object := object.(type) {
	case *types.PkgName, *types.Label:
		return true
	case *types.TypeName:
		_, typeParameter := object.Type().(*types.TypeParam)
		return typeParameter
	case *types.Var:
		return !object.IsField() &&
			(object.Pkg() == nil ||
				object.Parent() != object.Pkg().Scope())
	default:
		return false
	}
}

func (index *objectIndex) bindingRole(
	object types.Object,
	source identity.OccurrenceID,
) identity.SemanticBindingRole {
	switch object.(type) {
	case *types.PkgName:
		return identity.SemanticBindingImport
	case *types.Label:
		return identity.SemanticBindingLabel
	case *types.TypeName:
		return identity.SemanticBindingTypeParameter
	}
	if !source.IsZero() {
		role := index.contexts.byOccurrence[source].bindingRole
		if role.Valid() {
			return role
		}
	}
	return identity.SemanticBindingLocal
}

func (index *objectIndex) indexMembers(
	typ types.Type,
	nominal types.Type,
) {
	visit := memberVisit{typ: typ, nominal: nominal}
	if typ == nil || index.memberVisits[visit] {
		return
	}
	index.memberVisits[visit] = true
	index.work.MemberTypeVisits++
	switch typed := typ.(type) {
	case *types.Named:
		owner := types.Type(typed)
		for methodIndex := 0; methodIndex < typed.NumMethods(); methodIndex++ {
			index.admitMemberOwner(
				typed.Method(methodIndex), owner,
			)
		}
		index.indexMembers(typed.Underlying(), owner)
		for argument := 0; argument < typed.TypeArgs().Len(); argument++ {
			index.indexMembers(typed.TypeArgs().At(argument), nil)
		}
	case *types.Alias:
		index.indexMembers(types.Unalias(typed), nil)
	case *types.Pointer:
		index.indexMembers(typed.Elem(), nominal)
	case *types.Array:
		index.indexMembers(typed.Elem(), nil)
	case *types.Slice:
		index.indexMembers(typed.Elem(), nil)
	case *types.Map:
		index.indexMembers(typed.Key(), nil)
		index.indexMembers(typed.Elem(), nil)
	case *types.Chan:
		index.indexMembers(typed.Elem(), nil)
	case *types.Struct:
		owner := nominal
		if owner == nil {
			owner = typed
		}
		for fieldIndex := 0; fieldIndex < typed.NumFields(); fieldIndex++ {
			field := typed.Field(fieldIndex)
			index.admitMemberOwner(field, owner)
			index.indexMembers(field.Type(), nil)
		}
	case *types.Interface:
		owner := nominal
		if owner == nil {
			owner = typed
		}
		typed.Complete()
		for methodIndex := 0; methodIndex < typed.NumExplicitMethods(); methodIndex++ {
			method := typed.ExplicitMethod(methodIndex)
			index.admitMemberOwner(method, owner)
			index.indexMembers(method.Type(), nil)
		}
		for embedded := 0; embedded < typed.NumEmbeddeds(); embedded++ {
			index.indexMembers(typed.EmbeddedType(embedded), nil)
		}
	case *types.Signature:
		if typed.Recv() != nil {
			index.indexMembers(typed.Recv().Type(), nil)
		}
		index.indexTuple(typed.Params())
		index.indexTuple(typed.Results())
	case *types.Tuple:
		index.indexTuple(typed)
	case *types.TypeParam:
		index.indexMembers(typed.Constraint(), nil)
	case *types.Union:
		for term := 0; term < typed.Len(); term++ {
			index.indexMembers(typed.Term(term).Type(), nil)
		}
	}
}

func (index *objectIndex) admitMemberOwner(
	member types.Object,
	owner types.Type,
) {
	if function, ok := member.(*types.Func); ok {
		member = function.Origin()
	}
	owner = originMemberOwner(owner)
	owners := index.memberOwnerRelations[member]
	candidateNominal := nominalMemberOwner(owner)
	if candidateNominal {
		retained := owners[:0]
		for _, existing := range owners {
			if !nominalMemberOwner(existing) &&
				types.Identical(
					existing.Underlying(),
					owner.Underlying(),
				) {
				continue
			}
			retained = append(retained, existing)
		}
		owners = retained
	} else {
		for _, existing := range owners {
			if nominalMemberOwner(existing) &&
				types.Identical(
					existing.Underlying(),
					owner.Underlying(),
				) {
				return
			}
		}
	}
	for _, existing := range owners {
		if types.Identical(existing, owner) {
			return
		}
	}
	index.memberOwnerRelations[member] = append(owners, owner)
}

func nominalMemberOwner(typ types.Type) bool {
	typ = stripMemberOwnerPointer(typ)
	_, named := typ.(*types.Named)
	return named
}

func originMemberOwner(typ types.Type) types.Type {
	unwrapped := stripMemberOwnerPointer(typ)
	if named, ok := unwrapped.(*types.Named); ok {
		return named.Origin()
	}
	return unwrapped
}

func stripMemberOwnerPointer(typ types.Type) types.Type {
	for {
		pointer, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = pointer.Elem()
	}
	return typ
}

func (index *objectIndex) indexTuple(
	tuple *types.Tuple,
) {
	if tuple == nil {
		return
	}
	for element := 0; element < tuple.Len(); element++ {
		index.indexMembers(tuple.At(element).Type(), nil)
	}
}

func predeclaredObjects() map[types.Object]catalog.PredeclaredKind {
	out := map[types.Object]catalog.PredeclaredKind{}
	for _, kind := range catalog.AllPredeclared() {
		object := types.Universe.Lookup(kind.Name())
		if object != nil {
			out[object] = kind
		}
	}
	return out
}

func (index *objectIndex) bindingID(
	object types.Object,
) (identity.SemanticBindingID, bool) {
	candidate := index.bindingByObject[object]
	if candidate == nil {
		return identity.SemanticBindingID{}, false
	}
	id, present := index.bindingIDs[candidate]
	return id, present
}

func (index *objectIndex) bindingForOccurrence(
	occurrence identity.OccurrenceID,
) (identity.SemanticBindingID, bool) {
	candidate := index.bindingBySource[occurrence]
	if candidate == nil {
		return identity.SemanticBindingID{}, false
	}
	id, present := index.bindingIDs[candidate]
	return id, present
}

func (index *objectIndex) objectForOccurrence(
	occurrence identity.OccurrenceID,
) (types.Object, bool) {
	object, present := index.objectBySource[occurrence]
	return object, present
}

func (index *objectIndex) packageID(
	pkg *types.Package,
) (identity.PackageID, error) {
	if pkg == nil {
		return identity.PackageID{}, fmt.Errorf(
			"semantic object has no package",
		)
	}
	id := index.packageByPath[pkg.Path()]
	if id.IsZero() {
		return identity.PackageID{}, fmt.Errorf(
			"checker package %s is absent from source universe",
			pkg.Path(),
		)
	}
	return id, nil
}

func (index *objectIndex) objectReference(
	object types.Object,
) (semantic.ObjectReference, error) {
	if object == nil {
		return semantic.NoObjectReference(), nil
	}
	if binding, present := index.bindingID(object); present {
		return semantic.BindingReference(binding)
	}
	declaration, err := index.declarationID(object)
	if err != nil {
		return semantic.ObjectReference{}, err
	}
	return semantic.DeclarationReference(declaration)
}
