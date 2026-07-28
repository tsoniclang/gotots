package localtype

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/declaration/definedtype"
	namedstruct "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
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
		for _, requirement := range context.LexicalAnonymousStructs(typeName) {
			artifact, _, ok := requirement.AnonymousStruct()
			if !ok || artifact.LexicalAnchor() != typeName {
				return api.StatementEmission{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "local type received an inconsistent anonymous-struct requirement",
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
			operations, err := lexicalAnonymousStructOperations(
				artifact,
				artifacts[artifact],
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			target, err := namedstruct.EmitAnonymous(
				context.WithRole(api.RoleLocalDeclaration),
				children,
				artifact.SourceType(),
				artifact.TargetName(),
				operations,
				false,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, target.Declarations()...)
			requests = append(requests, target.Requests()...)
		}
	}
	return api.NewStatementEmission(statements, requests)
}

func localTypeRequirements(
	context api.Context,
	typeName *types.TypeName,
) []api.DeclarationRequirement {
	var requirements []api.DeclarationRequirement
	for _, requirement := range context.LexicalAnonymousStructs(typeName) {
		if requirement.Kind() !=
			api.DeclarationRequirementAnonymousStruct {
			requirements = append(requirements, requirement)
		}
	}
	return requirements
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
		default:
			return nil, &api.InvariantError{
				Role:   api.RoleLocalDeclaration,
				Reason: "lexical anonymous-struct demand is invalid",
			}
		}
	}
	return operations, nil
}
