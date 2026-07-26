package emit

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type targetBinding struct {
	name         string
	sourceFile   *ast.File
	sourcePath   string
	moduleExport bool
}

type declarationRegistry struct {
	byObject map[types.Object]targetBinding
}

type nameOwner struct {
	byObject            map[types.Object]targetBinding
	targetNameByObject  map[types.Object]string
	sourceNameBases     map[string]struct{}
	generatedSuffixes   map[string][]uint64
	nextGeneratedSuffix map[string]uint64
	registry            *declarationRegistry
}

func newNameOwner(packageScope *types.Scope, info *types.Info) *nameOwner {
	return newNameOwnerWithRegistry(packageScope, info, newDeclarationRegistry())
}

func newDeclarationRegistry() *declarationRegistry {
	return &declarationRegistry{byObject: make(map[types.Object]targetBinding)}
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

type fileNames struct {
	owner         *nameOwner
	sourceFile    *ast.File
	packageScope  *types.Scope
	factory       tsgo.Factory
	targetPath    string
	require       func(types.Object) error
	temporaries   map[api.TemporaryKind]uint64
	importNames   map[string]struct{}
	importAliases map[types.Object]string
	primitives    map[api.PrimitiveAlias]string
}

func (n *nameOwner) ForFile(
	sourceFile *ast.File,
	packageScope *types.Scope,
	factory tsgo.Factory,
	targetPath string,
	require func(types.Object) error,
) api.Names {
	return &fileNames{
		owner:         n,
		sourceFile:    sourceFile,
		packageScope:  packageScope,
		factory:       factory,
		targetPath:    targetPath,
		require:       require,
		temporaries:   make(map[api.TemporaryKind]uint64),
		importNames:   make(map[string]struct{}),
		importAliases: make(map[types.Object]string),
		primitives:    make(map[api.PrimitiveAlias]string),
	}
}

func (n *fileNames) Declare(object types.Object) (string, error) {
	if object != nil && object.Parent() == n.packageScope {
		binding, ok := n.owner.byObject[object]
		if !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "package declaration was not reserved",
			}
		}
		return binding.name, nil
	}
	return n.owner.declare(object, targetBinding{})
}

func (n *fileNames) Reference(object types.Object) (api.NameReference, error) {
	if object == nil {
		return api.NameReference{}, &api.NameError{Reason: "reference object is nil"}
	}
	if isPackageObject(object) && n.require != nil {
		if err := n.require(object); err != nil {
			return api.NameReference{}, err
		}
	}
	binding, ok := n.owner.byObject[object]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	if binding.sourceFile != nil && binding.sourceFile != n.sourceFile {
		modulePath, err := output.ModuleSpecifier(n.targetPath, binding.sourcePath)
		if err != nil {
			return api.NameReference{}, err
		}
		localName := binding.name
		if object.Parent() != n.packageScope {
			localName = n.importName(object, binding.name)
		}
		request, err := api.NewImportRequest(
			n.factory,
			api.ImportPhaseValue,
			modulePath,
			binding.name,
			localName,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(localName, request)
	}
	return api.NewNameReference(binding.name)
}

func (n *fileNames) importName(object types.Object, preferred string) string {
	if existing := n.importAliases[object]; existing != "" {
		return existing
	}
	base := preferred + "__from_" + portableIdentifier(object.Pkg().Path())
	candidate := base
	for suffix := uint64(1); n.packageScope.Lookup(candidate) != nil ||
		n.hasImportName(candidate) ||
		n.owner.hasSourceName(candidate); suffix++ {
		candidate = base + "_" + strconv.FormatUint(suffix, 10)
	}
	n.importNames[candidate] = struct{}{}
	n.importAliases[object] = candidate
	return candidate
}

func (n *fileNames) hasImportName(name string) bool {
	_, exists := n.importNames[name]
	return exists
}

func (n *nameOwner) hasSourceName(name string) bool {
	_, exists := n.sourceNameBases[name]
	return exists
}

func (n *fileNames) Primitive(alias api.PrimitiveAlias) (api.NameReference, error) {
	if existing := n.primitives[alias]; existing != "" {
		modulePath, err := output.ModuleSpecifier(n.targetPath, output.ScalarSupportPath)
		if err != nil {
			return api.NameReference{}, err
		}
		request, err := api.NewPrimitiveAliasRequest(n.factory, modulePath, alias, existing)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName, _, err := api.PrimitiveAliasRepresentation(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	if n.packageScope.Lookup(localName) != nil ||
		n.owner.hasSourceName(localName) ||
		n.hasImportName(localName) {
		base := exportedName + "__from_gotots_support"
		localName = base
		for suffix := uint64(1); n.packageScope.Lookup(localName) != nil ||
			n.owner.hasSourceName(localName) ||
			n.hasImportName(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.primitives[alias] = localName
	modulePath, err := output.ModuleSpecifier(n.targetPath, output.ScalarSupportPath)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewPrimitiveAliasRequest(n.factory, modulePath, alias, localName)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}

func (n *fileNames) Temporary(kind api.TemporaryKind) (string, error) {
	prefix, err := api.TemporaryPrefix(kind)
	if err != nil {
		return "", err
	}
	for {
		index := n.temporaries[kind]
		n.temporaries[kind] = index + 1
		candidate := prefix + strconv.FormatUint(index, 10)
		if _, reserved := n.owner.sourceNameBases[candidate]; reserved {
			continue
		}
		return candidate, nil
	}
}

func (n *fileNames) ModuleExport(object types.Object) (bool, error) {
	if object == nil {
		return false, &api.NameError{Reason: "declaration object is nil"}
	}
	binding, ok := n.owner.byObject[object]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok {
		return false, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	return binding.moduleExport, nil
}

func isPackageObject(object types.Object) bool {
	return object != nil && object.Pkg() != nil &&
		object.Parent() == object.Pkg().Scope()
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
	if tsgo.RequiresBindingIdentifierEscape(identifier) {
		return "__go_" + identifier
	}
	return identifier
}
