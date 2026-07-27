package emit

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type targetBinding struct {
	name         string
	sourceFile   *ast.File
	sourcePath   string
	moduleExport bool
}

type packageVariableBinding struct {
	fieldName    string
	statePath    string
	assemblyPath string
}

type declarationRegistry struct {
	byObject                 map[types.Object]targetBinding
	memberNameByObject       map[*types.Var]string
	packageVariables         map[*types.Var]packageVariableBinding
	assemblyPathByPackage    map[*types.Package]string
	importQualifierByPackage map[*types.Package]string
}

type nameOwner struct {
	byObject            map[types.Object]targetBinding
	targetNameByObject  map[types.Object]string
	sourceNameBases     map[string]struct{}
	generatedSuffixes   map[string][]uint64
	nextGeneratedSuffix map[string]uint64
	memberNameByObject  map[*types.Var]string
	registry            *declarationRegistry
}

func newNameOwner(packageScope *types.Scope, info *types.Info) *nameOwner {
	return newNameOwnerWithRegistry(packageScope, info, newDeclarationRegistry())
}

func newDeclarationRegistry() *declarationRegistry {
	return &declarationRegistry{
		byObject:                 make(map[types.Object]targetBinding),
		memberNameByObject:       make(map[*types.Var]string),
		packageVariables:         make(map[*types.Var]packageVariableBinding),
		assemblyPathByPackage:    make(map[*types.Package]string),
		importQualifierByPackage: make(map[*types.Package]string),
	}
}

func newNameOwnerWithRegistry(
	packageScope *types.Scope,
	info *types.Info,
	registry *declarationRegistry,
) *nameOwner {
	owner := &nameOwner{
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
	for _, object := range info.Defs {
		if object == nil || object.Name() == "_" {
			continue
		}
		owner.sourceNameBases[portableIdentifier(object.Name())] = struct{}{}
		if object.Parent() == nil {
			continue
		}
		if _, exists := seen[object]; exists {
			continue
		}
		seen[object] = struct{}{}
		objectsByScope[object.Parent()] = append(objectsByScope[object.Parent()], object)
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
	}
	return owner
}

func (n *nameOwner) Reserve(
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

func (n *nameOwner) preallocatePackageInitializers(
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

func (n *nameOwner) declare(object types.Object, binding targetBinding) (string, error) {
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

func (r *declarationRegistry) reserve(
	object types.Object,
	binding targetBinding,
) error {
	if r == nil {
		return &api.NameError{Name: objectName(object), Reason: "declaration registry is nil"}
	}
	if existing, ok := r.byObject[object]; ok {
		if existing.sourceFile != binding.sourceFile ||
			existing.sourcePath != binding.sourcePath ||
			existing.name != binding.name {
			return &api.NameError{
				Name:   objectName(object),
				Reason: "declaration has conflicting target ownership",
			}
		}
		return nil
	}
	r.byObject[object] = binding
	return nil
}

func (r *declarationRegistry) indexPackageTargets(
	sourcePackages []*load.Package,
) error {
	if r == nil {
		return &api.NameError{Reason: "declaration registry is nil"}
	}
	packages := make([]*types.Package, 0, len(sourcePackages))
	for _, sourcePackage := range sourcePackages {
		if sourcePackage == nil || sourcePackage.Types() == nil {
			return &api.NameError{Reason: "source package identity is nil"}
		}
		typesPackage := sourcePackage.Types()
		if _, duplicate := r.assemblyPathByPackage[typesPackage]; duplicate {
			return &api.NameError{
				Name:   typesPackage.Path(),
				Reason: "source package identity is duplicated",
			}
		}
		assemblyPath, err := output.PackageAssemblyPath(sourcePackage)
		if err != nil {
			return err
		}
		r.assemblyPathByPackage[typesPackage] = assemblyPath
		packages = append(packages, typesPackage)
	}
	return r.indexPackageQualifiers(packages)
}

func (r *declarationRegistry) indexPackageQualifiers(
	sourcePackages []*types.Package,
) error {
	if r == nil {
		return &api.NameError{Reason: "declaration registry is nil"}
	}
	packages := slices.Clone(sourcePackages)
	for _, sourcePackage := range packages {
		if sourcePackage == nil ||
			sourcePackage.Path() == "" ||
			sourcePackage.Name() == "" {
			return &api.NameError{Reason: "source package identity is nil"}
		}
	}
	sort.Slice(packages, func(left, right int) bool {
		return packages[left].Path() < packages[right].Path()
	})
	used := make(map[string]struct{}, len(packages))
	paths := make(map[string]struct{}, len(packages))
	for _, sourcePackage := range packages {
		if _, duplicate := paths[sourcePackage.Path()]; duplicate {
			return &api.NameError{
				Name:   sourcePackage.Path(),
				Reason: "source package path is duplicated",
			}
		}
		paths[sourcePackage.Path()] = struct{}{}
		base := portableIdentifier(sourcePackage.Name())
		qualifier := base
		for suffix := uint64(1); ; suffix++ {
			if _, duplicate := used[qualifier]; !duplicate {
				break
			}
			qualifier = base + "__package_" + strconv.FormatUint(suffix, 10)
		}
		used[qualifier] = struct{}{}
		r.importQualifierByPackage[sourcePackage] = qualifier
	}
	return nil
}

func (n *nameOwner) ReservePackageVariable(
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

func (n *nameOwner) preallocateScope(
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

func (n *nameOwner) generatedSuffix(base string, index uint64) uint64 {
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

func (n *nameOwner) preallocateMethods(info *types.Info) {
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

func (n *nameOwner) preallocateMembers(packageScope *types.Scope) {
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
		used := map[string]struct{}{
			"constructor": {},
		}
		for index := range structType.NumFields() {
			field := structType.Field(index)
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
}
