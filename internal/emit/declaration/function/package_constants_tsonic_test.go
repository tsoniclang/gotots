package function_test

import (
	"path/filepath"
	"testing"
)

func TestPackageConstantsExecuteDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadPackageConstantsProject(t)
	workingDirectory := t.TempDir()
	targetFiles := emitPackageConstantsProject(t, loaded, workingDirectory)
	printed := make(map[string]string, len(targetFiles))
	for name, targetFile := range targetFiles {
		printed[name+".ts"] = printTargetFile(t, targetFile, workingDirectory)
	}
	targetOutput := executeFilesThroughTsonic(
		t,
		printed,
		"use.ts",
		tsonicProof{
			namespace: "GoToTS.PackageConstants",
			assembly:  "GoToTSPackageConstants",
			runnerSource: `Console.WriteLine(GoToTS.PackageConstants.Use.AddBase(2));
Console.WriteLine(GoToTS.PackageConstants.Use.IsEnabled() ? "true" : "false");
`,
			requiredTarget: []string{
				"public static readonly long Base;",
				"public static readonly bool Enabled;",
				"Base = 40;",
				"Enabled = true;",
				"return Constants.Base + value;",
				"return Constants.Enabled;",
			},
			forbiddenTarget: []string{
				"dynamic",
				"object Base",
				"object Enabled",
			},
		},
	)
	goOutput := runPackageConstantsGo(t, filepath.Join(workingDirectory, "go"))
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
