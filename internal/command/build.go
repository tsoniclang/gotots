package command

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/config"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	"github.com/tsoniclang/gotots/internal/load"
)

type Report struct {
	files          int
	semanticDigest string
	output         string
}

func (r Report) Summary() string {
	return fmt.Sprintf(
		"generated_files=%d semantic_digest=%s output=%s",
		r.files,
		r.semanticDigest,
		r.output,
	)
}

func Build(ctx context.Context, project config.Project) (Report, error) {
	var semanticDigest string
	files, err := writeOutputTransaction(
		project.OutputDirectory(),
		func(outputDirectory string) (int, error) {
			compiled, digest, prepareErr := prepareBuildInWorker(
				ctx,
				project,
				outputDirectory,
			)
			if prepareErr != nil {
				return 0, prepareErr
			}
			semanticDigest = digest
			verified, verifyErr := verifyImplementationContracts(
				project,
				compiled,
				outputDirectory,
			)
			if verifyErr != nil {
				return 0, verifyErr
			}
			written, writeErr := writePrintPlanTo(
				project,
				verified,
				semanticDigest,
				outputDirectory,
			)
			if writeErr != nil {
				return 0, writeErr
			}
			if verifyErr := project.GoTool().VerifyComplete(); verifyErr != nil {
				return 0, verifyErr
			}
			return written, nil
		},
	)
	if err != nil {
		return Report{}, err
	}
	return Report{
		files:          files,
		semanticDigest: semanticDigest,
		output:         project.OutputDirectory(),
	}, nil
}

func prepareBuild(
	ctx context.Context,
	project config.Project,
	outputDirectory string,
) (printPlan, string, error) {
	certificationSources, err := implementationcontract.LoadCertificationSources(
		project.ImplementationCertificationSources(),
	)
	if err != nil {
		return printPlan{}, "", err
	}
	preparedImplementations, err := prepareSourceImplementations(
		project,
		certificationSources,
	)
	if err != nil {
		return printPlan{}, "", err
	}
	preparedCallables, err := prepareCallableImplementations(
		project,
		certificationSources,
	)
	if err != nil {
		return printPlan{}, "", err
	}
	standardLibrary, externalProvider, err := certifyProviders(project)
	if err != nil {
		return printPlan{}, "", err
	}
	program, err := load.Load(ctx, load.Request{
		Directory:    project.SourceRoot(),
		Pattern:      project.PackagePattern(),
		BuildProfile: project.BuildProfile(),
		GoTool:       project.GoTool(),
	})
	if err != nil {
		return printPlan{}, "", err
	}
	roots, err := selectRoots(program, project.RootMode())
	if err != nil {
		return printPlan{}, "", err
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = project.IntegerRepresentation()
	options.EvaluationOrder = project.EvaluationOrder()

	options.StandardLibrary = standardLibrary
	options.ExternalProvider = externalProvider
	sourceImplementations, err := joinSourceImplementations(
		preparedImplementations,
		program,
	)
	if err != nil {
		return printPlan{}, "", err
	}
	options.SourceImplementations = sourceImplementations
	callableImplementations, err := joinCallableImplementations(
		preparedCallables,
		program,
	)
	if err != nil {
		return printPlan{}, "", err
	}
	options.CallableImplementations = callableImplementations

	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		return printPlan{}, "", err
	}
	evidence := config.EvidenceDigests{Source: program.SourceDigest()}
	if sourceImplementations != nil {
		evidence.PackageImplementations = sourceImplementations.Digest()
	}
	if callableImplementations != nil {
		evidence.CallableImplementations = callableImplementations.Digest()
	}
	if standardLibrary != nil {
		evidence.StandardLibrary = standardLibrary.ManifestDigest()
	}
	if externalProvider != nil {
		evidence.Externals = externalProvider.ProviderDigest()
	}
	semanticDigest, err := project.SemanticDigest(evidence)
	if err != nil {
		return printPlan{}, "", err
	}
	plan, err := stagePrintPlan(outputDirectory, emission)
	if err != nil {
		return printPlan{}, "", err
	}
	return plan, semanticDigest, nil
}

func prepareCallableImplementations(
	project config.Project,
	certificationSources []implementationcontract.CertificationSource,
) (*callableimplementation.Prepared, error) {
	contracts := project.CallableImplementations()
	if len(contracts) == 0 {
		return nil, nil
	}
	return callableimplementation.PrepareAll(callableimplementation.Config{
		ContractPaths:        contracts,
		CertificationSources: certificationSources,
		BuildProfile:         project.BuildProfile(),
		Compilation: callableimplementation.CompilationDocument{
			Integers:        project.IntegerRepresentation().String(),
			EvaluationOrder: project.EvaluationOrder().String(),
		},
	})
}

func prepareSourceImplementations(
	project config.Project,
	certificationSources []implementationcontract.CertificationSource,
) (*sourceimplementation.Prepared, error) {
	contracts := project.PackageImplementations()
	if len(contracts) == 0 {
		return nil, nil
	}
	return sourceimplementation.PrepareAll(sourceimplementation.Config{
		RepositoryRoot:       project.DistributionRoot(),
		ContractPaths:        contracts,
		CertificationSources: certificationSources,
		BuildProfile:         project.BuildProfile(),
		Compilation: sourceimplementation.CompilationDocument{
			Integers:        project.IntegerRepresentation().String(),
			EvaluationOrder: project.EvaluationOrder().String(),
		},
		ScratchRoot: filepath.Join(
			project.DistributionRoot(),
			".temp",
			"source-implementation-certify",
		),
		TSGoTool: project.TSGoTool(),
	})
}

func joinSourceImplementations(
	prepared *sourceimplementation.Prepared,
	program *load.Program,
) (*sourceimplementation.Certificate, error) {
	if prepared == nil {
		return nil, nil
	}
	return prepared.Join(program)
}

func joinCallableImplementations(
	prepared *callableimplementation.Prepared,
	program *load.Program,
) (*callableimplementation.Certificate, error) {
	if prepared == nil {
		return nil, nil
	}
	return prepared.Join(program)
}

func certifyProviders(
	project config.Project,
) (*gostdlibcertify.Certificate, *externalcertify.Certificate, error) {
	if !project.StandardLibraryEnabled() && !project.ExternalsEnabled() {
		return nil, nil, nil
	}
	root := project.DistributionRoot()
	standardLibrary, err := gostdlibcertify.Verify(gostdlibcertify.Config{
		RepositoryRoot:      root,
		ProviderRoot:        filepath.Join(root, "gostdlib"),
		ManifestPath:        filepath.Join(root, "gostdlib", "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(root, "gostdlib", "contract", "modules.json"),
		FacetMapPath:        filepath.Join(root, "gostdlib", "contract", "facets.json"),
		RuntimeContractPath: filepath.Join(root, "gostdlib", "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(root, "gostdlib", "tsconfig.json"),
		ScratchDirectory:    filepath.Join(root, ".temp", "gostdlib-certify"),
		GoTool:              project.GoTool(),
		TSGoTool:            project.TSGoTool(),
		BuildProfile:        project.BuildProfile(),
		Backend:             "node",
		MinimumGoVersion:    project.BuildProfile().ToolchainVersion(),
		MaximumGoVersion:    project.BuildProfile().ToolchainVersion(),
	})
	if err != nil {
		return nil, nil, err
	}
	if !project.ExternalsEnabled() {
		return standardLibrary, nil, nil
	}
	externalProvider, err := externalcertify.Verify(externalcertify.Config{
		RepositoryRoot:              root,
		ProviderRoot:                filepath.Join(root, "externals"),
		ManifestPath:                filepath.Join(root, "externals", "contract", "manifest.json"),
		BindingMapPath:              filepath.Join(root, "externals", "contract", "bindings.json"),
		TSConfigPath:                filepath.Join(root, "externals", "tsconfig.json"),
		StandardLibraryManifestPath: filepath.Join(root, "gostdlib", "contract", "manifest.json"),
		StandardLibraryRuntimePath:  filepath.Join(root, "gostdlib", "contract", "runtime.json"),
		BuildProfile:                project.BuildProfile(),
		GoTool:                      project.GoTool(),
		TSGoTool:                    project.TSGoTool(),
		Backend:                     "node",
	})
	if err != nil {
		return nil, nil, err
	}
	return standardLibrary, externalProvider, nil
}
