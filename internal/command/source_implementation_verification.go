package command

import (
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/config"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
)

func verifySourceImplementationContracts(
	project config.Project,
	compiled compiledPrintPlan,
	outputDirectory string,
) (verifiedPrintPlan, error) {
	plan := compiled.plan
	if err := plan.validate(outputDirectory); err != nil {
		return verifiedPrintPlan{}, err
	}
	if !plan.hasSourceImplementation {
		return verifiedPrintPlan{plan: plan}, nil
	}
	generated, err := stagedSourceImplementationTargets(
		plan.sourceImplementation.generated,
	)
	if err != nil {
		return verifiedPrintPlan{}, err
	}
	installed, err := stagedSourceImplementationTargets(plan.files)
	if err != nil {
		return verifiedPrintPlan{}, err
	}
	packages := make(
		[]sourceimplementation.ContractPackage,
		len(plan.sourceImplementation.packages),
	)
	for index, selected := range plan.sourceImplementation.packages {
		packages[index], err = sourceimplementation.NewContractPackage(
			selected.packagePath,
			selected.assemblyPath,
			selected.exports,
		)
		if err != nil {
			return verifiedPrintPlan{}, err
		}
	}
	if err := sourceimplementation.VerifyStagedGeneratedContracts(
		sourceimplementation.StagedVerificationConfig{
			RepositoryRoot: project.DistributionRoot(),
			ScratchRoot: filepath.Join(
				outputDirectory,
				sourceImplementationVerificationDirectoryName,
			),
			TSGoTool:  project.TSGoTool(),
			Generated: generated,
			Installed: installed,
			Packages:  packages,
		},
	); err != nil {
		return verifiedPrintPlan{}, err
	}
	return verifiedPrintPlan{plan: plan}, nil
}

func stagedSourceImplementationTargets(
	files []printPlanFile,
) ([]sourceimplementation.StagedTarget, error) {
	result := make([]sourceimplementation.StagedTarget, len(files))
	for index, file := range files {
		var err error
		result[index], err = sourceimplementation.NewStagedTarget(
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
