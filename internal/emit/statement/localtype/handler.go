package localtype

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/declaration/definedtype"
	namedstruct "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
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
		typeName, ok := context.TypesInfo().Defs[spec.Name].(*types.TypeName)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					spec.Name,
				)
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
		target, handled, err := definedtype.Emit(
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
		for _, requirement := range context.LexicalGeneratedArtifacts(typeName) {
			artifact, ok := requirement.GeneratedArtifact()
			if !ok || artifact.LexicalAnchor() != typeName {
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
				return api.StatementEmission{}, err
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
	for _, requirement := range context.LexicalGeneratedArtifacts(typeName) {
		if _, generated := requirement.GeneratedArtifact(); !generated {
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
		operations, err := lexicalAnonymousStructOperations(
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
	default:
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical generated-artifact kind is invalid",
		}
	}
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
	capabilities, err := maprepresentation.CapabilitiesFromRequirements(
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
		valueType.Value(),
		capabilities,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			nil,
			context.Factory().Identifier(artifact.TargetName()),
			nil,
			nil,
			specialization.Members(),
		),
		api.CombineRequests(
			keyType.Requests(),
			valueType.Requests(),
			specialization.Requests(),
		)...,
	), nil
}

func lexicalAnonymousStructOperations(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) ([]api.NamedStructOperation, error) {
	var operations []api.NamedStructOperation
	for _, requirement := range requirements {
		selected, demand, ok := requirement.AnonymousStruct()
		if !ok || selected != artifact {
			return nil, &api.InvariantError{
				Role:   api.RoleLocalDeclaration,
				Reason: "lexical anonymous struct received a foreign requirement",
			}
		}
		switch demand {
		case api.AnonymousStructDemandDefinition:
		case api.AnonymousStructDemandZero:
			operations = append(operations, api.NamedStructOperationZero)
		case api.AnonymousStructDemandCopy:
			operations = append(operations, api.NamedStructOperationCopy)
		case api.AnonymousStructDemandEqual:
			operations = append(operations, api.NamedStructOperationEqual)
		case api.AnonymousStructDemandHash:
			operations = append(operations, api.NamedStructOperationHash)
		default:
			return nil, &api.InvariantError{
				Role:   api.RoleLocalDeclaration,
				Reason: "lexical anonymous-struct demand is invalid",
			}
		}
	}
	return operations, nil
}
