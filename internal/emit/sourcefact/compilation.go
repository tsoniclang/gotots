package sourcefact

import (
	"github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	runtimesourcefact "github.com/tsoniclang/gotots/internal/emit/runtime/sourcefact"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const CompilationSentinelName = "$GoCompilation"
const compilationSchema = "gotots-go-source-compilation-fact-v1"

func Compilation(
	factory tsgo.Factory,
	program *load.Program,
	scalar api.ScalarABI,
	evaluation api.EvaluationOrder,
	standardLibrary *gostdlibcertify.Certificate,
	externalProvider *externalcertify.Certificate,
	packageImplementations *sourceimplementation.Certificate,
	callableImplementations *callableimplementation.Certificate,
) (api.StatementEmission, error) {
	if program == nil {
		return api.StatementEmission{}, &Error{Reason: "compilation fact has no source program"}
	}
	standardLibraryDigest := ""
	if standardLibrary != nil {
		standardLibraryDigest = standardLibrary.ManifestDigest()
	}
	externalProviderDigest := ""
	if externalProvider != nil {
		externalProviderDigest = externalProvider.ManifestDigest()
	}
	packageImplementationDigest := ""
	if packageImplementations != nil {
		packageImplementationDigest = packageImplementations.Digest()
	}
	callableImplementationDigest := ""
	if callableImplementations != nil {
		callableImplementationDigest = callableImplementations.Digest()
	}
	declaration := factory.TypeAliasDeclaration(
		nil,
		factory.Identifier(CompilationSentinelName),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNeverKeyword),
	)
	arguments, err := CompilationArguments(
		factory,
		program.BuildProfile(),
		program.SourceDigest(),
		scalar,
		evaluation,
		standardLibraryDigest,
		externalProviderDigest,
		packageImplementationDigest,
		callableImplementationDigest,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	fact, err := api.RuntimeContract(api.RuntimeSourceCompilationFact)
	if err != nil {
		return api.StatementEmission{}, err
	}
	annotation, err := runtimesourcefact.AnnotationWithArguments(
		factory,
		CompilationSentinelName,
		fact.ExportedName(),
		declaration,
		arguments...,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	attribute, err := tsoniccore.Resolve(tsoniccore.SymbolAttribute)
	if err != nil {
		return api.StatementEmission{}, err
	}
	attributeRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		attribute.Module(),
		attribute.Export(),
		attribute.Export(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	modulePath, err := targetoutput.RuntimeModuleSpecifier(fact.OutputPath())
	if err != nil {
		return api.StatementEmission{}, err
	}
	factRequest, err := api.NewRuntimeImportRequest(
		factory,
		api.ImportPhaseValue,
		modulePath,
		api.RuntimeSourceCompilationFact,
		fact.ExportedName(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := []tsgo.Statement{declaration, annotation}
	requests := []api.RootRequest{attributeRequest, factRequest}
	if packageImplementations != nil {
		implementationFact, implementationErr := api.RuntimeContract(
			api.RuntimeSourceImplementationFact,
		)
		if implementationErr != nil {
			return api.StatementEmission{}, implementationErr
		}
		implementationModule, moduleErr := targetoutput.RuntimeModuleSpecifier(
			implementationFact.OutputPath(),
		)
		if moduleErr != nil {
			return api.StatementEmission{}, moduleErr
		}
		implementationRequest, requestErr := api.NewRuntimeImportRequest(
			factory,
			api.ImportPhaseValue,
			implementationModule,
			api.RuntimeSourceImplementationFact,
			implementationFact.ExportedName(),
		)
		if requestErr != nil {
			return api.StatementEmission{}, requestErr
		}
		requests = append(requests, implementationRequest)
		for _, implementation := range packageImplementations.Implementations() {
			sourcePackage := program.PackageByPath(implementation.PackagePath())
			outputPath, pathErr := targetoutput.PackageAssemblyPath(sourcePackage)
			if pathErr != nil {
				return api.StatementEmission{}, pathErr
			}
			implementationArguments, argumentErr := PackageImplementationArguments(
				factory,
				implementation,
				outputPath,
			)
			if argumentErr != nil {
				return api.StatementEmission{}, argumentErr
			}
			implementationAnnotation, annotationErr := runtimesourcefact.AnnotationWithArguments(
				factory,
				CompilationSentinelName,
				implementationFact.ExportedName(),
				declaration,
				implementationArguments...,
			)
			if annotationErr != nil {
				return api.StatementEmission{}, annotationErr
			}
			statements = append(statements, implementationAnnotation)
		}
	}
	return api.NewStatementEmission(statements, requests)
}

func CompilationArguments(
	factory tsgo.Factory,
	profile environment.BuildProfile,
	sourceDigest string,
	scalar api.ScalarABI,
	evaluation api.EvaluationOrder,
	standardLibraryDigest string,
	externalProviderDigest string,
	packageImplementationDigest string,
	callableImplementationDigest string,
) ([]tsgo.Expression, error) {
	if !profile.Valid() || sourceDigest == "" || !scalar.Valid() || !evaluation.Valid() {
		return nil, &Error{Reason: "compilation fact input is incomplete"}
	}
	byteOrder, err := profile.ByteOrder()
	if err != nil {
		return nil, err
	}
	arguments := []tsgo.Expression{
		text(factory, compilationSchema),
		text(factory, sourceDigest),
		text(factory, profile.ToolchainVersion()),
		text(factory, profile.GOOS()),
		text(factory, profile.GOARCH()),
		truth(factory, profile.CgoEnabled()),
		count(factory, int(scalar.NativeIntegerWidth())),
		text(factory, byteOrderName(byteOrder)),
		text(factory, scalar.IntegerRepresentation().String()),
		text(factory, evaluation.String()),
		text(factory, "go-concurrency"),
		text(factory, "serial-synchronous-execution-envelope"),
		text(factory, standardLibraryDigest),
		text(factory, externalProviderDigest),
		text(factory, packageImplementationDigest),
		text(factory, callableImplementationDigest),
		count(factory, len(profile.Tags())),
	}
	for _, tag := range profile.Tags() {
		arguments = append(arguments, text(factory, tag))
	}
	return arguments, nil
}

func byteOrderName(order environment.ByteOrder) string {
	switch order {
	case environment.ByteOrderLittleEndian:
		return "little-endian"
	case environment.ByteOrderBigEndian:
		return "big-endian"
	default:
		return "invalid"
	}
}
