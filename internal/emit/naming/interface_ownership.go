package naming

import (
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
)

func (r *Registry) interfaceAdapterSupportModule(
	sourceType types.Type,
) (string, error) {
	if r == nil || !interfaceAdapterSource(sourceType) {
		return "", &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "interface-adapter support module input is invalid",
		}
	}
	packages := make(map[*types.Package]struct{})
	collectAssociatedPackages(sourceType, packages, make(map[types.Type]struct{}))
	if len(packages) == 0 {
		return "language/" + supportTypeFamily(sourceType), nil
	}
	tokens := make([]string, 0, len(packages))
	for sourcePackage := range packages {
		token, err := r.semanticPackageToken(sourcePackage)
		if err != nil {
			return "", err
		}
		tokens = append(tokens, token)
	}
	slices.Sort(tokens)
	if len(tokens) == 1 {
		return "packages/" + tokens[0], nil
	}
	return "composites/" + supportTypeFamily(sourceType), nil
}

func collectAssociatedPackages(
	sourceType types.Type,
	packages map[*types.Package]struct{},
	seen map[types.Type]struct{},
) {
	if sourceType == nil {
		return
	}
	sourceType = types.Unalias(sourceType)
	if _, visited := seen[sourceType]; visited {
		return
	}
	seen[sourceType] = struct{}{}
	switch source := sourceType.(type) {
	case *types.Named:
		if object := source.Obj(); object != nil && object.Pkg() != nil {
			packages[object.Pkg()] = struct{}{}
		}
		for index := range source.TypeArgs().Len() {
			collectAssociatedPackages(source.TypeArgs().At(index), packages, seen)
		}
	case *types.Pointer:
		collectAssociatedPackages(source.Elem(), packages, seen)
	case *types.Array:
		collectAssociatedPackages(source.Elem(), packages, seen)
	case *types.Slice:
		collectAssociatedPackages(source.Elem(), packages, seen)
	case *types.Map:
		collectAssociatedPackages(source.Key(), packages, seen)
		collectAssociatedPackages(source.Elem(), packages, seen)
	case *types.Chan:
		collectAssociatedPackages(source.Elem(), packages, seen)
	case *types.Struct:
		for index := range source.NumFields() {
			collectAssociatedPackages(source.Field(index).Type(), packages, seen)
		}
	case *types.Signature:
		collectAssociatedPackages(source.Params(), packages, seen)
		collectAssociatedPackages(source.Results(), packages, seen)
	case *types.Tuple:
		for index := range source.Len() {
			collectAssociatedPackages(source.At(index).Type(), packages, seen)
		}
	case *types.Interface:
		contract := source.Complete()
		for index := range contract.NumEmbeddeds() {
			collectAssociatedPackages(contract.EmbeddedType(index), packages, seen)
		}
		for index := range contract.NumMethods() {
			collectAssociatedPackages(contract.Method(index).Type(), packages, seen)
		}
	case *types.TypeParam:
		collectAssociatedPackages(source.Constraint(), packages, seen)
	case *types.Union:
		for index := range source.Len() {
			collectAssociatedPackages(source.Term(index).Type(), packages, seen)
		}
	}
}

func supportTypeFamily(sourceType types.Type) string {
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		return "scalars"
	case *types.Array:
		return "arrays"
	case *types.Slice:
		return "slices"
	case *types.Map:
		return "maps"
	case *types.Pointer:
		return "pointers"
	case *types.Chan:
		return "channels"
	case *types.Signature:
		return "callables"
	case *types.Struct:
		return "structs"
	default:
		return "values"
	}
}

type privateMethodIdentity struct {
	packagePath string
	sourceName  string
}

type privateMethodSelection struct {
	identity privateMethodIdentity
	owner    *types.Package
}

func (r *Registry) indexPrivateMethodNames(
	packages []*types.Package,
	typeInformation []*types.Info,
) error {
	byTargetBase := make(map[string]map[privateMethodIdentity]*types.Package)
	occupiedTargetBases := make(map[string]struct{})
	seen := make(map[types.Type]struct{})
	for _, sourcePackage := range packages {
		scope := sourcePackage.Scope()
		for _, name := range scope.Names() {
			collectPrivateMethods(
				scope.Lookup(name).Type(),
				byTargetBase,
				occupiedTargetBases,
				seen,
			)
		}
	}
	for _, information := range typeInformation {
		collectPrivateMethodsFromInfo(
			information,
			byTargetBase,
			occupiedTargetBases,
			seen,
		)
	}
	bases := make([]string, 0, len(byTargetBase))
	for base := range byTargetBase {
		bases = append(bases, base)
	}
	sort.Strings(bases)
	for _, base := range bases {
		owners := byTargetBase[base]
		selections := make([]privateMethodSelection, 0, len(owners))
		for identity, owner := range owners {
			selections = append(selections, privateMethodSelection{
				identity: identity,
				owner:    owner,
			})
		}
		slices.SortFunc(selections, func(left, right privateMethodSelection) int {
			if compared := strings.Compare(
				left.identity.packagePath,
				right.identity.packagePath,
			); compared != 0 {
				return compared
			}
			return strings.Compare(left.identity.sourceName, right.identity.sourceName)
		})
		_, occupied := occupiedTargetBases[base]
		qualified := len(selections) != 1 ||
			occupied ||
			privateMethodTargetHazard(base)
		used := make(map[string]uint64)
		for _, selection := range selections {
			name := base
			if qualified {
				qualifier := r.importQualifierByPackage[selection.owner]
				if qualifier == "" {
					return &api.NameError{
						Name:   selection.identity.sourceName,
						Reason: "private method package qualifier is absent",
					}
				}
				name = qualifier + "$" + base
				ordinal := used[name]
				used[name] = ordinal + 1
				if ordinal != 0 {
					name += "__method_" + strconv.FormatUint(ordinal+1, 10)
				}
			}
			r.privateMethodNames[selection.identity] = name
		}
	}
	return nil
}

func collectPrivateMethodsFromInfo(
	information *types.Info,
	byTargetBase map[string]map[privateMethodIdentity]*types.Package,
	occupiedTargetBases map[string]struct{},
	seen map[types.Type]struct{},
) {
	if information == nil {
		return
	}
	collectObject := func(object types.Object) {
		if object != nil {
			collectPrivateMethods(
				object.Type(),
				byTargetBase,
				occupiedTargetBases,
				seen,
			)
		}
	}
	for _, object := range information.Defs {
		collectObject(object)
	}
	for _, object := range information.Uses {
		collectObject(object)
	}
	for _, object := range information.Implicits {
		collectObject(object)
	}
	for _, value := range information.Types {
		collectPrivateMethods(
			value.Type,
			byTargetBase,
			occupiedTargetBases,
			seen,
		)
	}
	for _, instance := range information.Instances {
		collectPrivateMethods(
			instance.Type,
			byTargetBase,
			occupiedTargetBases,
			seen,
		)
		if instance.TypeArgs != nil {
			for index := range instance.TypeArgs.Len() {
				collectPrivateMethods(
					instance.TypeArgs.At(index),
					byTargetBase,
					occupiedTargetBases,
					seen,
				)
			}
		}
	}
	for _, selection := range information.Selections {
		if selection == nil {
			continue
		}
		collectPrivateMethods(
			selection.Recv(),
			byTargetBase,
			occupiedTargetBases,
			seen,
		)
		collectObject(selection.Obj())
	}
	for _, scope := range information.Scopes {
		if scope == nil {
			continue
		}
		for _, name := range scope.Names() {
			collectObject(scope.Lookup(name))
		}
	}
}

func collectPrivateMethods(
	sourceType types.Type,
	byTargetBase map[string]map[privateMethodIdentity]*types.Package,
	occupiedTargetBases map[string]struct{},
	seen map[types.Type]struct{},
) {
	if sourceType == nil {
		return
	}
	sourceType = types.Unalias(sourceType)
	if _, visited := seen[sourceType]; visited {
		return
	}
	seen[sourceType] = struct{}{}
	switch source := sourceType.(type) {
	case *types.Named:
		for index := range source.NumMethods() {
			recordMethodTargetBase(
				byTargetBase,
				occupiedTargetBases,
				source.Method(index),
			)
		}
		for index := range source.TypeParams().Len() {
			collectPrivateMethods(source.TypeParams().At(index), byTargetBase, occupiedTargetBases, seen)
		}
		for index := range source.TypeArgs().Len() {
			collectPrivateMethods(source.TypeArgs().At(index), byTargetBase, occupiedTargetBases, seen)
		}
		collectPrivateMethods(source.Underlying(), byTargetBase, occupiedTargetBases, seen)
	case *types.Pointer:
		collectPrivateMethods(source.Elem(), byTargetBase, occupiedTargetBases, seen)
	case *types.Array:
		collectPrivateMethods(source.Elem(), byTargetBase, occupiedTargetBases, seen)
	case *types.Slice:
		collectPrivateMethods(source.Elem(), byTargetBase, occupiedTargetBases, seen)
	case *types.Map:
		collectPrivateMethods(source.Key(), byTargetBase, occupiedTargetBases, seen)
		collectPrivateMethods(source.Elem(), byTargetBase, occupiedTargetBases, seen)
	case *types.Chan:
		collectPrivateMethods(source.Elem(), byTargetBase, occupiedTargetBases, seen)
	case *types.Struct:
		for index := range source.NumFields() {
			collectPrivateMethods(source.Field(index).Type(), byTargetBase, occupiedTargetBases, seen)
		}
	case *types.Signature:
		if source.Recv() != nil {
			collectPrivateMethods(source.Recv().Type(), byTargetBase, occupiedTargetBases, seen)
		}
		for index := range source.RecvTypeParams().Len() {
			collectPrivateMethods(source.RecvTypeParams().At(index), byTargetBase, occupiedTargetBases, seen)
		}
		for index := range source.TypeParams().Len() {
			collectPrivateMethods(source.TypeParams().At(index), byTargetBase, occupiedTargetBases, seen)
		}
		collectPrivateMethods(source.Params(), byTargetBase, occupiedTargetBases, seen)
		collectPrivateMethods(source.Results(), byTargetBase, occupiedTargetBases, seen)
	case *types.Tuple:
		for index := range source.Len() {
			collectPrivateMethods(source.At(index).Type(), byTargetBase, occupiedTargetBases, seen)
		}
	case *types.Interface:
		contract := source.Complete()
		for index := range contract.NumMethods() {
			method := contract.Method(index)
			recordMethodTargetBase(byTargetBase, occupiedTargetBases, method)
			collectPrivateMethods(method.Type(), byTargetBase, occupiedTargetBases, seen)
		}
		for index := range contract.NumEmbeddeds() {
			collectPrivateMethods(contract.EmbeddedType(index), byTargetBase, occupiedTargetBases, seen)
		}
	case *types.TypeParam:
		collectPrivateMethods(source.Constraint(), byTargetBase, occupiedTargetBases, seen)
	case *types.Union:
		for index := range source.Len() {
			collectPrivateMethods(source.Term(index).Type(), byTargetBase, occupiedTargetBases, seen)
		}
	}
}

func recordMethodTargetBase(
	byTargetBase map[string]map[privateMethodIdentity]*types.Package,
	occupiedTargetBases map[string]struct{},
	method *types.Func,
) {
	if method == nil {
		return
	}
	base := portableIdentifier(method.Name())
	if privateMethodTargetHazard(method.Name()) {
		base = semanticname.Identifier(method.Name())
	}
	if method.Exported() || method.Pkg() == nil {
		occupiedTargetBases[base] = struct{}{}
		return
	}
	identity := privateMethodIdentity{
		packagePath: method.Pkg().Path(),
		sourceName:  method.Name(),
	}
	owners := byTargetBase[base]
	if owners == nil {
		owners = make(map[privateMethodIdentity]*types.Package)
		byTargetBase[base] = owners
	}
	owners[identity] = method.Pkg()
}

func (r *Registry) privateMethodName(method *types.Func) (string, error) {
	if method == nil || method.Exported() || method.Pkg() == nil {
		return "", &api.NameError{Reason: "private method identity is invalid"}
	}
	identity := privateMethodIdentity{
		packagePath: method.Pkg().Path(),
		sourceName:  method.Name(),
	}
	if name := r.privateMethodNames[identity]; name != "" {
		return name, nil
	}
	return "", &api.NameError{
		Name:   method.FullName(),
		Reason: "private method identity was not indexed",
	}
}

func privateMethodTargetHazard(name string) bool {
	switch name {
	case "constructor",
		typescriptclass.PromiseAssimilationMember,
		"name",
		"length",
		"caller",
		"arguments",
		"prototype":
		return true
	default:
		return false
	}
}
