package command

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
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
	if !plan.hasCallableImplementation {
		return plan, nil
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
		modules[index], err = callableimplementation.NewStagedModule(
			module.sourcePath,
			module.outputPath,
			module.sourceDigest,
			module.exports,
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
