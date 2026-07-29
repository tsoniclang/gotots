package naming

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct {
	byObject            map[types.Object]targetBinding
	targetNameByObject  map[types.Object]string
	sourceNameBases     map[string]struct{}
	generatedSuffixes   map[string][]uint64
	nextGeneratedSuffix map[string]uint64
	memberNameByObject  map[*types.Var]string
	registry            *Registry
}

func newNameOwner(packageScope *types.Scope, info *types.Info) *Owner {
	return NewOwner(packageScope, info, NewRegistry())
}

func NewOwner(
	packageScope *types.Scope,
	info *types.Info,
	registry *Registry,
) *Owner {
	owner := &Owner{
		byObject:            make(map[types.Object]targetBinding),
		targetNameByObject:  make(map[types.Object]string),
		sourceNameBases:     make(map[string]struct{}),
		generatedSuffixes:   make(map[string][]uint64),
		nextGeneratedSuffix: make(map[string]uint64),
		memberNameByObject:  registry.memberNameByObject,
		registry:            registry,
	}
	if info == nil {
		return owner
	}
	objectsByScope := make(map[*types.Scope][]types.Object)
	seen := make(map[types.Object]struct{})
	var labels []*types.Label
	record := func(object types.Object) {
		if object == nil || object.Name() == "_" {
			return
		}
		owner.sourceNameBases[portableIdentifier(object.Name())] = struct{}{}
		if label, ok := object.(*types.Label); ok {
			if _, exists := seen[label]; !exists {
				seen[label] = struct{}{}
				labels = append(labels, label)
			}
			return
		}
		if object.Parent() == nil {
			return
		}
		if _, exists := seen[object]; exists {
			return
		}
		seen[object] = struct{}{}
		objectsByScope[object.Parent()] = append(objectsByScope[object.Parent()], object)
	}
	for _, object := range info.Defs {
		record(object)
	}
	for _, object := range info.Implicits {
		record(object)
	}
	if packageScope != nil {
		owner.preallocateScope(
			packageScope,
			objectsByScope,
			make(map[string]uint64),
			make(map[string]uint32),
		)
		owner.preallocateMethods(info)
		owner.preallocateMembers(packageScope)
		owner.preallocateAnonymousMembers(info)
	}
	owner.preallocateLabels(labels)
	return owner
}

func (n *Owner) preallocateLabels(labels []*types.Label) {
	slices.SortFunc(labels, func(left, right *types.Label) int {
		return compareNameObjects(left, right)
	})
	counts := make(map[string]uint64)
	for _, label := range labels {
		base := portableIdentifier(label.Name())
		index := counts[base]
		counts[base]++
		name := base
		if index != 0 {
			name += "__label_" + strconv.FormatUint(index, 10)
		}
		n.targetNameByObject[label] = name
	}
}

func (n *Owner) Reserve(
	object types.Object,
	sourceFile *ast.File,
	sourcePath string,
) (string, error) {
	if sourceFile == nil {
		return "", &api.NameError{Name: objectName(object), Reason: "source file is nil"}
	}
	if sourcePath == "" {
		return "", &api.NameError{Name: objectName(object), Reason: "target module path is empty"}
	}
	binding := targetBinding{
		name:         objectName(object),
		sourceFile:   sourceFile,
		sourcePath:   sourcePath,
		moduleExport: true,
	}
	name, err := n.declare(object, binding)
	if err != nil {
		return "", err
	}
	binding.name = name
	if err := n.registry.reserve(object, binding); err != nil {
		return "", err
	}
	return name, nil
}

func (n *Owner) PreallocatePackageInitializers(
	objects []types.Object,
) error {
	for index, object := range objects {
		function, ok := object.(*types.Func)
		if !ok || function.Name() != "init" || function.Pkg() == nil {
			return &api.NameError{
				Name:   objectName(object),
				Reason: "package initializer identity is invalid",
			}
		}
		name := portableIdentifier(function.Name())
		if index != 0 {
			name += "__shadow_" + strconv.FormatUint(
				n.generatedSuffix(name, uint64(index-1)),
				10,
			)
		}
		n.targetNameByObject[function] = name
	}
	return nil
}

func (n *Owner) declare(object types.Object, binding targetBinding) (string, error) {
	if object == nil {
		return "", &api.NameError{Reason: "declaration object is nil"}
	}
	if existing, ok := n.byObject[object]; ok {
		if binding.sourceFile != nil &&
			(existing.sourceFile != binding.sourceFile ||
				existing.sourcePath != binding.sourcePath) {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "declaration was reserved by a different target module",
			}
		}
		return existing.name, nil
	}
	if object.Name() == "" {
		return "", &api.NameError{Reason: "declaration name is empty"}
	}
	name, ok := n.targetNameByObject[object]
	if !ok {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "declaration object was not indexed from its Go scope",
		}
	}
	binding.name = name
	n.byObject[object] = binding
	return name, nil
}

func (n *Owner) ReservePackageVariable(
	variable *types.Var,
	statePath string,
	assemblyPath string,
) (string, error) {
	switch {
	case variable == nil:
		return "", &api.NameError{Reason: "package variable is nil"}
	case variable.IsField() || variable.Pkg() == nil ||
		variable.Parent() != variable.Pkg().Scope():
		return "", &api.NameError{
			Name:   variable.Name(),
			Reason: "variable is not package-scoped",
		}
	case statePath == "":
		return "", &api.NameError{
			Name:   variable.Name(),
			Reason: "package state path is empty",
		}
	case assemblyPath == "":
		return "", &api.NameError{
			Name:   variable.Name(),
			Reason: "package assembly path is empty",
		}
	}
	fieldName, ok := n.targetNameByObject[variable]
	if !ok {
		return "", &api.NameError{
			Name:   variable.Name(),
			Reason: "package variable was not indexed from its Go scope",
		}
	}
	binding := packageVariableBinding{
		fieldName:    fieldName,
		statePath:    statePath,
		assemblyPath: assemblyPath,
	}
	if existing, exists := n.registry.packageVariables[variable]; exists {
		if existing != binding {
			return "", &api.NameError{
				Name:   variable.Name(),
				Reason: "package variable has conflicting target ownership",
			}
		}
		return fieldName, nil
	}
	n.registry.packageVariables[variable] = binding
	return fieldName, nil
}

func (n *Owner) preallocateScope(
	scope *types.Scope,
	objectsByScope map[*types.Scope][]types.Object,
	activeCounts map[string]uint64,
	activeNames map[string]uint32,
) {
	objects := slices.Clone(objectsByScope[scope])
	slices.SortFunc(objects, compareNameObjects)
	originalCounts := make(map[string]uint64)
	scopeNames := make([]string, 0, len(objects))
	for _, object := range objects {
		base := portableIdentifier(object.Name())
		if _, recorded := originalCounts[base]; !recorded {
			originalCounts[base] = activeCounts[base]
		}
		rank := activeCounts[base]
		candidate := base
		for {
			if rank != 0 {
				suffix := n.generatedSuffix(base, rank-1)
				candidate = base + "__shadow_" + strconv.FormatUint(suffix, 10)
			}
			if activeNames[candidate] == 0 {
				break
			}
			rank++
		}
		activeCounts[base] = rank + 1
		activeNames[candidate]++
		scopeNames = append(scopeNames, candidate)
		n.targetNameByObject[object] = candidate
	}
	for index := range scope.NumChildren() {
		n.preallocateScope(
			scope.Child(index),
			objectsByScope,
			activeCounts,
			activeNames,
		)
	}
	for _, name := range scopeNames {
		activeNames[name]--
		if activeNames[name] == 0 {
			delete(activeNames, name)
		}
	}
	for base, original := range originalCounts {
		if original == 0 {
			delete(activeCounts, base)
		} else {
			activeCounts[base] = original
		}
	}
}

func (n *Owner) generatedSuffix(base string, index uint64) uint64 {
	suffixes := n.generatedSuffixes[base]
	next := n.nextGeneratedSuffix[base]
	if next == 0 {
		next = 1
	}
	for uint64(len(suffixes)) <= index {
		candidate := base + "__shadow_" + strconv.FormatUint(next, 10)
		suffix := next
		next++
		if _, reserved := n.sourceNameBases[candidate]; reserved {
			continue
		}
		suffixes = append(suffixes, suffix)
	}
	n.generatedSuffixes[base] = suffixes
	n.nextGeneratedSuffix[base] = next
	return suffixes[index]
}

func compareNameObjects(left types.Object, right types.Object) int {
	switch {
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	case left.Name() < right.Name():
		return -1
	case left.Name() > right.Name():
		return 1
	case left.String() < right.String():
		return -1
	case left.String() > right.String():
		return 1
	default:
		return 0
	}
}

func objectName(object types.Object) string {
	if object == nil {
		return ""
	}
	return object.Name()
}

func portableIdentifier(source string) string {
	var result strings.Builder
	for _, value := range source {
		switch {
		case value >= 'A' && value <= 'Z',
			value >= 'a' && value <= 'z',
			value >= '0' && value <= '9',
			value == '_':
			result.WriteRune(value)
		default:
			result.WriteString("__u")
			result.WriteString(strconv.FormatInt(int64(value), 16))
			result.WriteByte('_')
		}
	}
	identifier := result.String()
	if identifier == "__proto__" ||
		tsgo.RequiresBindingIdentifierEscape(identifier) {
		return "__go_" + identifier
	}
	return identifier
}

func (n *Owner) preallocateMethods(info *types.Info) {
	var methods []*types.Func
	for _, object := range info.Defs {
		method, ok := object.(*types.Func)
		if !ok {
			continue
		}
		signature, ok := method.Type().(*types.Signature)
		if !ok || signature.Recv() == nil {
			continue
		}
		methods = append(methods, method)
	}
	slices.SortFunc(methods, func(left, right *types.Func) int {
		return compareNameObjects(left, right)
	})
	used := make(map[string]struct{})
	for _, name := range n.targetNameByObject {
		used[name] = struct{}{}
	}
	for _, method := range methods {
		signature := method.Type().(*types.Signature)
		receiverName := receiverTypeName(signature.Recv().Type())
		base := portableIdentifier(receiverName) + "_" +
			portableIdentifier(method.Name())
		candidate := base
		for suffix := uint64(1); ; suffix++ {
			_, targetCollision := used[candidate]
			_, sourceCollision := n.sourceNameBases[candidate]
			if !targetCollision && !sourceCollision {
				break
			}
			candidate = base + "__method_" + strconv.FormatUint(suffix, 10)
		}
		used[candidate] = struct{}{}
		n.targetNameByObject[method] = candidate
	}
}

func receiverTypeName(sourceType types.Type) string {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		sourceType = pointer.Elem()
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return "receiver"
	}
	return named.Obj().Name()
}

func (n *Owner) preallocateMembers(packageScope *types.Scope) {
	names := packageScope.Names()
	slices.Sort(names)
	for _, name := range names {
		typeName, ok := packageScope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		structType, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		n.preallocateStructMembers(structType)
	}
}

func (n *Owner) preallocateAnonymousMembers(info *types.Info) {
	seen := make(map[*types.Struct]struct{})
	for _, typeAndValue := range info.Types {
		structType, ok := types.Unalias(typeAndValue.Type).(*types.Struct)
		if !ok {
			continue
		}
		if _, exists := seen[structType]; exists {
			continue
		}
		seen[structType] = struct{}{}
		n.preallocateStructMembers(structType)
	}
}

func (n *Owner) preallocateStructMembers(structType *types.Struct) {
	if structType == nil {
		return
	}
	used := map[string]struct{}{
		"constructor": {},
	}
	for index := range structType.NumFields() {
		field := structType.Field(index)
		if existing := n.memberNameByObject[field]; existing != "" {
			used[existing] = struct{}{}
			continue
		}
		base := portableIdentifier(field.Name())
		candidate := base
		for suffix := uint64(1); ; suffix++ {
			if _, duplicate := used[candidate]; !duplicate {
				break
			}
			candidate = base + "__field_" + strconv.FormatUint(suffix, 10)
		}
		used[candidate] = struct{}{}
		n.memberNameByObject[field] = candidate
	}
}
