package sourcefact

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const packageStorageSchema = "gotots-go-package-storage-fact-v1"

type PackageStorageMember struct {
	Variable *types.Var
	Name     string
	Origin   DeclarationOrigin
}

func PackageStorageMembers(
	context api.Context,
	assemblyPath string,
	statePath string,
	stateClassName string,
	members []PackageStorageMember,
) (api.StatementEmission, error) {
	if assemblyPath == "" || statePath == "" || stateClassName == "" ||
		len(members) == 0 {
		return api.StatementEmission{}, &Error{
			Reason: "package storage-member facts are incomplete",
		}
	}
	modulePath, err := targetoutput.ModuleSpecifier(assemblyPath, statePath)
	if err != nil {
		return api.StatementEmission{}, err
	}
	stateType, err := api.NewImportRequest(
		context.Factory(),
		api.ImportPhaseType,
		modulePath,
		stateClassName,
		stateClassName,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(stateClassName),
		nil,
	)
	var facts []api.StatementEmission
	for _, member := range members {
		if member.Variable == nil || member.Name == "" || !member.Origin.Valid() {
			return api.StatementEmission{}, &Error{
				Reason: "package storage member is incomplete",
			}
		}
		arguments, argumentErr := SourceVariableMemberArguments(
			context.Factory(),
			member.Variable,
			member.Name,
			member.Origin,
		)
		if argumentErr != nil {
			return api.StatementEmission{}, argumentErr
		}
		fact, factErr := attribute.ApplyMember(
			context,
			target,
			attribute.MemberProperty,
			member.Name,
			api.RuntimeSourceDeclarationFact,
			arguments...,
		)
		if factErr != nil {
			return api.StatementEmission{}, factErr
		}
		facts = append(facts, fact)
	}
	emission, err := combine(facts)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(
		emission.Statements(),
		api.CombineRequests([]api.RootRequest{stateType}, emission.Requests()),
	)
}

func PackageStorage(
	context api.Context,
	targetName string,
	stateName string,
	packagePath string,
	modulePath string,
	moduleVersion string,
	ownerKind string,
	contractKey string,
	outputPath string,
	sourceDigest string,
	fieldCount int,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if targetName == "" || stateName == "" || packagePath == "" ||
		outputPath == "" || sourceDigest == "" || fieldCount <= 0 ||
		!validAuthoredOwner(ownerKind, modulePath, moduleVersion, contractKey) {
		return api.StatementEmission{}, &Error{Reason: "package storage fact is incomplete"}
	}
	target, err := exactDeclarationTarget(
		context.Factory(),
		[]string{targetName},
		artifactTargetType,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factTarget, err := NewTarget(target)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return factTarget.apply(
		context,
		api.RuntimeSourceStorageFact,
		text(context.Factory(), packageStorageSchema),
		text(context.Factory(), "package-state"),
		text(context.Factory(), packagePath),
		text(context.Factory(), modulePath),
		text(context.Factory(), moduleVersion),
		text(context.Factory(), ownerKind),
		text(context.Factory(), contractKey),
		text(context.Factory(), outputPath),
		text(context.Factory(), sourceDigest),
		text(context.Factory(), stateName),
		count(context.Factory(), fieldCount),
	)
}
