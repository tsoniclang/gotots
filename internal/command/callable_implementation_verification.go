package command

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/config"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyCallableImplementationContracts(
	project config.Project,
	plan printPlan,
	outputDirectory string,
) (printPlan, error) {
	prepared, err := prepareCallableImplementations(project)
	if err != nil {
		return printPlan{}, err
	}
	if prepared == nil {
		if plan.hasCallableImplementation {
			return printPlan{}, commandError(
				"verify callable implementation",
				"unconfigured callable evidence survived compilation",
			)
		}
		return plan, nil
	}
	if !plan.hasCallableImplementation {
		return printPlan{}, commandError(
			"verify callable implementation",
			"configured callable evidence is absent from compilation",
		)
	}
	if err := exactJoinPreparedCallablePlan(prepared, plan.callableImplementation); err != nil {
		return printPlan{}, err
	}
	existing := make(map[string]struct{}, len(plan.files))
	for _, file := range plan.files {
		existing[file.outputPath] = struct{}{}
	}
	modules := make(
		[]callableimplementation.StagedModule,
		len(plan.callableImplementation.modules),
	)
	for index, module := range plan.callableImplementation.modules {
		if _, collision := existing[module.outputPath]; collision {
			return printPlan{}, commandError(
				"verify callable implementation",
				fmt.Sprintf("output %q collides with a generated target", module.outputPath),
			)
		}
		var err error
		certificationSources := make(
			[]callableimplementation.CertificationSource,
			len(module.certificationSources),
		)
		for sourceIndex, source := range module.certificationSources {
			certificationSources[sourceIndex], err =
				callableimplementation.NewCertificationSource(
					source.sourcePath,
					source.sourceDigest,
				)
			if err != nil {
				return printPlan{}, err
			}
		}
		modules[index], err = callableimplementation.NewStagedModule(
			module.sourcePath,
			module.outputPath,
			module.sourceDigest,
			module.exports,
			certificationSources,
		)
		if err != nil {
			return printPlan{}, err
		}
	}
	generated, err := stagedCallableImplementationTargets(plan.files)
	if err != nil {
		return printPlan{}, err
	}
	callables := make(
		[]callableimplementation.StagedCallable,
		len(plan.callableImplementation.targets),
	)
	for index, target := range plan.callableImplementation.targets {
		variant, variantErr := callableImplementationVariant(target.variant)
		if variantErr != nil {
			return printPlan{}, variantErr
		}
		generatedTarget, targetErr := stagedCallableImplementationTarget(
			target,
			variant,
		)
		if targetErr != nil {
			return printPlan{}, targetErr
		}
		callables[index], targetErr = callableimplementation.NewStagedCallable(
			target.sourceIdentity,
			target.sourceSignature,
			target.sourceBodyDigest,
			variant,
			target.implementationOutput,
			target.implementationExport,
			generatedTarget,
		)
		if targetErr != nil {
			return printPlan{}, targetErr
		}
	}
	verified, err := callableimplementation.VerifyStagedGeneratedContracts(
		callableimplementation.StagedVerificationConfig{
			RepositoryRoot: project.DistributionRoot(),
			ScratchRoot: filepath.Join(
				outputDirectory,
				callableImplementationVerificationDirectoryName,
			),
			TSGoTool:  project.TSGoTool(),
			Generated: generated,
			Modules:   modules,
			Callables: callables,
		},
	)
	if err != nil {
		return printPlan{}, err
	}
	if len(verified) != len(modules) {
		return printPlan{}, commandError(
			"verify callable implementation",
			"verified module denominator differs",
		)
	}
	sort.Slice(verified, func(left, right int) bool {
		return verified[left].OutputPath() < verified[right].OutputPath()
	})
	for _, module := range verified {
		payload, encodeErr := tsgo.EncodeSourceFile(module.SourceFile())
		if encodeErr != nil {
			return printPlan{}, commandError(
				"encode callable implementation AST",
				fmt.Sprintf("%s: %v", module.OutputPath(), encodeErr),
			)
		}
		protocolPath := filepath.Join(
			plan.protocolDirectory,
			fmt.Sprintf("%06d.ast", len(plan.files)),
		)
		if err := writeExclusive(protocolPath, payload); err != nil {
			return printPlan{}, commandError(
				"stage callable implementation AST",
				err.Error(),
			)
		}
		plan.files = append(plan.files, printPlanFile{
			outputPath:   module.OutputPath(),
			protocolPath: protocolPath,
			protocolHash: sha256.Sum256(payload),
		})
	}
	return plan, nil
}

func exactJoinPreparedCallablePlan(
	prepared *callableimplementation.Prepared,
	plan callableImplementationPrintPlan,
) error {
	expectedModules := prepared.Modules()
	if len(expectedModules) != len(plan.modules) {
		return commandError(
			"join callable implementation handoff",
			"module denominator differs",
		)
	}
	actualModules := make(map[string]callableImplementationModule, len(plan.modules))
	for _, module := range plan.modules {
		actualModules[module.outputPath] = module
	}
	expectedTargets := make(map[string]callableImplementationTarget)
	for _, expected := range expectedModules {
		actual, ok := actualModules[expected.OutputPath()]
		claims := expected.CallableClaims()
		exports := make([]string, len(claims))
		for index, claim := range claims {
			exports[index] = claim.Export
			expectedTargets[claim.SourceIdentity] = callableImplementationTarget{
				sourceIdentity: claim.SourceIdentity, sourceSignature: claim.SourceSignature,
				sourceBodyDigest: claim.SourceBodyDigest,
				variant:          string(claim.Variant), implementationOutput: expected.OutputPath(),
				implementationExport: claim.Export,
			}
		}
		sort.Strings(exports)
		if !ok || actual.sourcePath != expected.SourcePath() ||
			actual.sourceDigest != expected.SourceDigest() ||
			!slices.Equal(actual.exports, exports) ||
			!sameCallableCertificationSources(
				actual.certificationSources,
				expected.CertificationSources(),
			) {
			return commandError(
				"join callable implementation handoff",
				fmt.Sprintf("module %q differs from prepared evidence", expected.OutputPath()),
			)
		}
		delete(actualModules, expected.OutputPath())
	}
	if len(actualModules) != 0 || len(expectedTargets) != len(plan.targets) {
		return commandError(
			"join callable implementation handoff",
			"callable denominator differs",
		)
	}
	for _, actual := range plan.targets {
		expected, ok := expectedTargets[actual.sourceIdentity]
		if !ok || actual.sourceSignature != expected.sourceSignature ||
			actual.sourceBodyDigest != expected.sourceBodyDigest ||
			actual.variant != expected.variant ||
			actual.implementationOutput != expected.implementationOutput ||
			actual.implementationExport != expected.implementationExport {
			return commandError(
				"join callable implementation handoff",
				fmt.Sprintf("callable %q differs from prepared evidence", actual.sourceIdentity),
			)
		}
		delete(expectedTargets, actual.sourceIdentity)
	}
	if len(expectedTargets) != 0 {
		return commandError(
			"join callable implementation handoff",
			"prepared callable is absent from compilation",
		)
	}
	return nil
}

func sameCallableCertificationSources(
	actual []callableImplementationCertificationSource,
	expected []callableimplementation.CertificationSource,
) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index, source := range expected {
		if actual[index].sourcePath != source.SourcePath() ||
			actual[index].sourceDigest != source.SourceDigest() {
			return false
		}
	}
	return true
}

func stagedCallableImplementationTargets(
	files []printPlanFile,
) ([]callableimplementation.StagedTarget, error) {
	result := make([]callableimplementation.StagedTarget, len(files))
	for index, file := range files {
		var err error
		result[index], err = callableimplementation.NewStagedTarget(
			file.outputPath,
			file.protocolPath,
			file.protocolHash,
		)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func callableImplementationVariant(
	selected string,
) (callableimplementation.Variant, error) {
	switch selected {
	case string(callableimplementation.VariantSource):
		return callableimplementation.VariantSource, nil
	case string(callableimplementation.VariantKernel):
		return callableimplementation.VariantKernel, nil
	default:
		return callableimplementation.VariantInvalid, commandError(
			"verify callable implementation",
			"variant is invalid",
		)
	}
}

func stagedCallableImplementationTarget(
	target callableImplementationTarget,
	variant callableimplementation.Variant,
) (callableimplementation.GeneratedTarget, error) {
	switch target.kind {
	case callableImplementationTargetModuleFunction:
		return callableimplementation.NewGeneratedModuleTarget(
			target.sourceIdentity,
			variant,
			target.generatedOutput,
			target.generatedExport,
		)
	case callableImplementationTargetStaticMethod:
		return callableimplementation.NewGeneratedStaticMethodTarget(
			target.sourceIdentity,
			variant,
			target.generatedOutput,
			target.className,
			target.memberName,
		)
	default:
		return callableimplementation.GeneratedTarget{}, commandError(
			"verify callable implementation",
			"generated target kind is invalid",
		)
	}
}
