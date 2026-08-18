package localtype

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/declaration/definedtype"
	interfaceadapter "github.com/tsoniclang/gotots/internal/emit/declaration/interfaceadapter"
	interfacedynamictype "github.com/tsoniclang/gotots/internal/emit/declaration/interfacedynamictype"
	interfacemethodtoken "github.com/tsoniclang/gotots/internal/emit/declaration/interfacemethodtoken"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/declaration/interfacetype"
	namedstruct "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	genericcapability "github.com/tsoniclang/gotots/internal/emit/generic/capability"
	genericconcretization "github.com/tsoniclang/gotots/internal/emit/generic/concretization"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	maprepresentation "github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeclStmt,
) (api.StatementEmission, error) {
	declaration, ok := source.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.TYPE || len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	typeNames := make([]*types.TypeName, 0, len(declaration.Specs))
	for _, candidate := range declaration.Specs {
		spec, ok := candidate.(*ast.TypeSpec)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					candidate,
				)
		}
		typeName, ok := context.TypesInfo().DefOf(spec.Name).(*types.TypeName)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					spec.Name,
				)
		}
		if typeName.Name() == "_" {
			continue
		}
		if _, err := context.Names().Declare(typeName); err != nil {
			return api.StatementEmission{}, err
		}
		typeNames = append(typeNames, typeName)
	}

	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, typeName := range typeNames {
		requirements := localTypeRequirements(context, typeName)
		target, handled, err := interfacetype.Emit(
			context.WithRole(api.RoleLocalDeclaration),
			children,
			declaration,
			typeName,
			requirements,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !handled {
			target, handled, err = definedtype.Emit(
				context.WithRole(api.RoleLocalDeclaration),
				children,
				declaration,
				typeName,
				requirements,
			)
		}
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !handled {
			target, err = namedstruct.EmitAssembly(
				context.WithRole(api.RoleLocalDeclaration),
				children,
				declaration,
				typeName,
				requirements,
			)
		}
		if err != nil {
			return api.StatementEmission{}, err
		}
		if target.Disposition() !=
			api.DeclarationDispositionMaterialized {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					source,
				)
		}
		statements = append(statements, target.Declarations()...)
		requests = append(requests, target.Requests()...)
		artifacts := make(
			map[*api.GeneratedArtifact][]api.DeclarationRequirement,
		)
		for _, requirement := range context.LexicalTypeRequirements(typeName) {
			artifact, ok := requirement.LexicalGeneratedArtifact()
			if !ok {
				continue
			}
			if artifact.LexicalAnchor() != typeName {
				return api.StatementEmission{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "local type received an inconsistent generated-artifact requirement",
				}
			}
			artifacts[artifact] = append(
				artifacts[artifact],
				requirement,
			)
		}
		ordered := make([]*api.GeneratedArtifact, 0, len(artifacts))
		for artifact := range artifacts {
			ordered = append(ordered, artifact)
		}
		sort.Slice(ordered, func(left, right int) bool {
			return ordered[left].TargetName() < ordered[right].TargetName()
		})
		for _, artifact := range ordered {
			generated, err := emitLexicalGeneratedArtifact(
				context,
				children,
				source,
				artifact,
				artifacts[artifact],
			)
			if err != nil {
				return api.StatementEmission{},
					api.WrapGeneratedArtifactError(artifact, err)
			}
			statements = append(statements, generated.Declarations()...)
			requests = append(requests, generated.Requests()...)
		}
	}
	return api.NewStatementEmission(statements, requests)
}

func localTypeRequirements(
	context api.Context,
	typeName *types.TypeName,
) []api.DeclarationRequirement {
	var requirements []api.DeclarationRequirement
	for _, requirement := range context.LexicalTypeRequirements(typeName) {
		if _, generated :=
			requirement.LexicalGeneratedArtifact(); !generated {
			requirements = append(requirements, requirement)
		}
	}
	return requirements
}

func emitLexicalGeneratedArtifact(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		structType, ok := artifact.StructType()
		if !ok {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "lexical anonymous-struct artifact has no struct type",
			}
		}
		operations, representationFacets, err :=
			namedstruct.SelectAnonymousRequirements(
				context.Role(),
				artifact,
				requirements,
			)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		return namedstruct.EmitAnonymous(
			context.WithRole(api.RoleLocalDeclaration),
			children,
			structType,
			artifact.TargetName(),
			operations,
			representationFacets,
			false,
		)
	case api.GeneratedArtifactMapSpecialization:
		return emitLexicalMapSpecialization(
			context,
			children,
			source,
			artifact,
			requirements,
		)
	case api.GeneratedArtifactInterfaceAdapter:
		return emitLexicalInterfaceAdapter(
			context,
			children,
			artifact,
			requirements,
		)
	case api.GeneratedArtifactAnonymousInterface:
		return emitLexicalAnonymousInterface(
			context,
			children,
			source,
			artifact,
			requirements,
		)
	case api.GeneratedArtifactInterfaceMethodToken:
		if err := exactLexicalInterfaceRequirement(
			artifact,
			requirements,
		); err != nil {
			return api.DeclarationEmission{}, err
		}
		return api.DirectDeclaration(
			interfacemethodtoken.Build(
				context.Factory(),
				artifact.TargetName(),
				nil,
				nil,
			),
		), nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		sourceType, ok := artifact.InterfaceDynamicType()
		if !ok {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "lexical interface dynamic-type artifact has no Go type",
			}
		}
		if err := exactLexicalInterfaceRequirement(
			artifact,
			requirements,
		); err != nil {
			return api.DeclarationEmission{}, err
		}
		return api.DirectDeclaration(
			interfacedynamictype.Build(
				context.Factory(),
				artifact.TargetName(),
				nil,
				types.Comparable(sourceType),
			),
		), nil
	case api.GeneratedArtifactGenericCapability:
		return emitLexicalGenericCapability(
			context,
			children,
			artifact,
			requirements,
		)
	case api.GeneratedArtifactGenericConcretization:
		deferred, err := exactLexicalGenericConcretizationRequirement(
			artifact,
			requirements,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		statements, requests, err := genericconcretization.Build(
			context.WithRole(api.RoleLocalDeclaration),
			children,
			artifact,
			nil,
			deferred,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		return api.NewDeclarationEmission(statements, requests)
	default:
		anchor := artifact.LexicalAnchor()
		anchorName := "<unknown>"
		position := token.Position{}
		if anchor != nil {
			anchorName = anchor.Name()
			position = context.FileSet().Position(anchor.Pos())
		}
		return api.DeclarationEmission{}, &api.InvariantError{
			Role: context.Role(),
			Reason: fmt.Sprintf(
				"lexical generated-artifact kind %d (%s) for %s at %s is invalid",
				artifact.Kind(),
				artifact.TargetName(),
				anchorName,
				position,
			),
		}
	}
}

func exactLexicalGenericConcretizationRequirement(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	if len(requirements) < 1 || len(requirements) > 2 {
		return false, &api.InvariantError{
			Role:   api.RoleLocalDeclaration,
			Reason: "lexical generic concretization requirements are not exact",
		}
	}
	bound, boundOK := artifact.GenericConcretization()
	base := false
	deferred := false
	for _, requirement := range requirements {
		selected, ok := requirement.GenericConcretization()
		generated, generatedOK := requirement.GeneratedArtifact()
		if !ok || !generatedOK || generated != artifact || !boundOK ||
			selected != bound {
			return false, &api.InvariantError{
				Role:   api.RoleLocalDeclaration,
				Reason: "lexical generic concretization received a foreign request",
			}
		}
		if requirement.DeferredGenericConcretization() {
			if deferred {
				return false, &api.InvariantError{
					Role:   api.RoleLocalDeclaration,
					Reason: "lexical generic concretization has duplicate deferred demand",
				}
			}
			deferred = true
		} else {
			if base {
				return false, &api.InvariantError{
					Role:   api.RoleLocalDeclaration,
					Reason: "lexical generic concretization has duplicate definition demand",
				}
			}
			base = true
		}
	}
	if !base {
		return false, &api.InvariantError{
			Role:   api.RoleLocalDeclaration,
			Reason: "lexical generic concretization lacks its definition demand",
		}
	}
	return deferred, nil
}

func emitLexicalGenericCapability(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	if err := genericcapability.ValidateRequirements(
		context.Role(),
		artifact,
		requirements,
	); err != nil {
		return api.DeclarationEmission{}, err
	}
	facet, err := api.NewGenericCapabilityCallableFacet(artifact)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = context.WithCooperativeCallable(
		facet,
		observation.Cooperative(),
	)
	statement, requests, err := genericcapability.Build(
		context,
		children,
		artifact,
		nil,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		statement,
		api.CombineRequests(
			requests,
			observation.Requests(),
		)...,
	), nil
}

func emitLexicalInterfaceAdapter(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	sourceType, ok := artifact.InterfaceAdapterType()
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical interface adapter has no concrete type",
		}
	}
	contracts, err := interfaceadapter.Contracts(
		artifact,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	statements, requests, err := interfaceadapter.Build(
		context.WithRole(api.RoleLocalDeclaration),
		children,
		artifact.TargetName(),
		sourceType,
		contracts,
		nil,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.NewDeclarationEmission(statements, requests)
}

func emitLexicalAnonymousInterface(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	interfaceType, ok := artifact.InterfaceType()
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical anonymous interface has no interface type",
		}
	}
	if err := exactLexicalInterfaceRequirement(
		artifact,
		requirements,
	); err != nil {
		return api.DeclarationEmission{}, err
	}
	statements, requests, err := interfacetype.Build(
		context.WithRole(api.RoleLocalDeclaration),
		children,
		source,
		artifact.TargetName(),
		interfaceType,
		nil,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.NewDeclarationEmission(statements, requests)
}

func exactLexicalInterfaceRequirement(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) error {
	if len(requirements) != 1 {
		return &api.InvariantError{
			Role:   api.RoleLocalDeclaration,
			Reason: "lexical interface artifact requires one exact request",
		}
	}
	selected, ok := requirements[0].GeneratedArtifact()
	if !ok || selected != artifact {
		return &api.InvariantError{
			Role:   api.RoleLocalDeclaration,
			Reason: "lexical interface artifact received a foreign request",
		}
	}
	return nil
}

func emitLexicalMapSpecialization(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	mapType, ok := artifact.MapType()
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical map-specialization artifact has no map type",
		}
	}
	err := maprepresentation.ValidateRequirements(
		context.Role(),
		artifact,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	keyType, err := children.RepresentedType(
		context.WithRole(api.RoleMapKey),
		source,
		mapType.Key(),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	_, ok = maprepresentation.Source(context, mapType)
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical map specialization has no representation model",
		}
	}
	storageKeyType, err := maprepresentation.EmitStorageKeyType(
		context,
		children,
		source,
		mapType.Key(),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	valueType, err := children.RepresentedType(
		context.WithRole(api.RoleMapValue),
		source,
		mapType.Elem(),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	specialization, err := maprepresentation.BuildSpecialization(
		context.WithRole(api.RoleLocalDeclaration),
		source,
		artifact.TargetName(),
		mapType,
		keyType.Value(),
		storageKeyType.Value(),
		valueType.Value(),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		typescriptclass.Declaration(context.Factory(),
			nil,
			context.Factory().Identifier(artifact.TargetName()),
			nil,
			specialization.HeritageClauses(),
			specialization.Members(),
		),
		api.CombineRequests(
			keyType.Requests(),
			storageKeyType.Requests(),
			valueType.Requests(),
			specialization.Requests(),
		)...,
	), nil
}
