package localtype

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	namedstruct "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	canonicalsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func localTypeSourceFacts(
	context api.Context,
	typeName *types.TypeName,
	source *ast.TypeSpec,
	declarations []tsgo.Statement,
	requirements []api.DeclarationRequirement,
) (api.StatementEmission, error) {
	origin, err := canonicalsourcefact.AuthoredOrigin(context, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	origin, err = canonicalsourcefact.TypeDeclarationOrigin(context, source, origin)
	if err != nil {
		return api.StatementEmission{}, err
	}
	declaration, err := canonicalsourcefact.LocalTypeDeclaration(
		context,
		typeName,
		origin,
		declarations,
	)
	if err != nil || typeName.IsAlias() {
		return declaration, err
	}
	memberOrigins, err := canonicalsourcefact.TypeMemberOrigins(
		context,
		typeName,
		source,
		origin,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	members, err := canonicalsourcefact.LocalTypeMembers(
		context,
		typeName,
		memberOrigins,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operations, err := canonicalsourcefact.LocalStructOperations(
		context,
		typeName,
		requirements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return combineSourceFacts(declaration, members, operations)
}

func lexicalArtifactSourceFacts(
	context api.Context,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
	declarations []tsgo.Statement,
) (api.StatementEmission, error) {
	artifactFact, err := canonicalsourcefact.GeneratedArtifact(
		context,
		artifact,
		declarations,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	facts := []api.StatementEmission{artifactFact}
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		operations, _, selectErr := namedstruct.SelectAnonymousRequirements(
			context.Role(),
			artifact,
			requirements,
		)
		if selectErr != nil {
			return api.StatementEmission{}, selectErr
		}
		operationFacts, factErr := canonicalsourcefact.AnonymousStructOperations(
			context,
			artifact,
			operations,
		)
		if factErr != nil {
			return api.StatementEmission{}, factErr
		}
		facts = append(facts, operationFacts)
	case api.GeneratedArtifactInterfaceAdapter:
		implementationFacts, factErr := canonicalsourcefact.InterfaceImplementations(
			context,
			artifact,
			requirements,
			declarations,
		)
		if factErr != nil {
			return api.StatementEmission{}, factErr
		}
		facts = append(facts, implementationFacts)
	}
	return combineSourceFacts(facts...)
}

func combineSourceFacts(facts ...api.StatementEmission) (api.StatementEmission, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, fact := range facts {
		statements = append(statements, fact.Statements()...)
		requests = append(requests, fact.Requests()...)
	}
	return api.NewStatementEmission(statements, requests)
}
