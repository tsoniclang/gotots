package command

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/config"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit"
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
	program, err := load.Load(ctx, load.Request{
		Directory:    project.SourceRoot(),
		Pattern:      project.PackagePattern(),
		BuildProfile: project.BuildProfile(),
	})
	if err != nil {
		return Report{}, err
	}
	roots, err := selectRoots(program, project.RootMode())
	if err != nil {
		return Report{}, err
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = project.IntegerRepresentation()
	options.EvaluationOrder = project.EvaluationOrder()
	options.ConcurrencySemantics = project.ConcurrencySemantics()

	standardLibrary, externalProvider, err := certifyProviders(project)
	if err != nil {
		return Report{}, err
	}
	options.StandardLibrary = standardLibrary
	options.ExternalProvider = externalProvider
	sourceImplementations, err := certifySourceImplementations(project, program)
	if err != nil {
		return Report{}, err
	}
	options.SourceImplementations = sourceImplementations

	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		return Report{}, err
	}
	sourceDigest, err := programDigest(program)
	if err != nil {
		return Report{}, err
	}
	evidence := config.EvidenceDigests{Source: sourceDigest}
	if sourceImplementations != nil {
		evidence.SourceImplementations = sourceImplementations.Digest()
	}
	if standardLibrary != nil {
		evidence.StandardLibrary = standardLibrary.ManifestDigest()
	}
	if externalProvider != nil {
		evidence.Externals = externalProvider.ProviderDigest()
	}
	semanticDigest, err := project.SemanticDigest(evidence)
	if err != nil {
		return Report{}, err
	}
	files, err := writeEmission(ctx, project, emission, semanticDigest)
	if err != nil {
		return Report{}, err
	}
	return Report{
		files:          files,
		semanticDigest: semanticDigest,
		output:         project.OutputDirectory(),
	}, nil
}

func certifySourceImplementations(
	project config.Project,
	program *load.Program,
) (*sourceimplementation.Certificate, error) {
	bundles := project.ImplementationBundles()
	if len(bundles) == 0 {
		return nil, nil
	}
	return sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: project.DistributionRoot(),
		Program:        program,
		ContractPaths:  bundles,
		Compilation: sourceimplementation.CompilationDocument{
			Integers:        project.IntegerRepresentation().String(),
			EvaluationOrder: project.EvaluationOrder().String(),
			Concurrency:     project.ConcurrencySemantics().String(),
		},
		ScratchRoot: filepath.Join(
			project.DistributionRoot(),
			".temp",
			"source-implementation-certify",
		),
	})
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
		GoBinary:            "go",
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
		Backend:                     "node",
	})
	if err != nil {
		return nil, nil, err
	}
	return standardLibrary, externalProvider, nil
}
