package function_test

import (
	"path/filepath"
	"testing"
)

func TestIntegerConstantsExecuteDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadIntegerConstantsProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitIntegerConstants(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.IntegerConstants",
		assembly:  "GoToTSIntegerConstants",
		runnerSource: `Console.WriteLine(GoToTS.IntegerConstants.Index.Small());
Console.WriteLine(GoToTS.IntegerConstants.Index.BeyondSafe());
Console.WriteLine(GoToTS.IntegerConstants.Index.Maximum());
Console.WriteLine(GoToTS.IntegerConstants.Index.Minimum());
`,
		requiredTarget: []string{
			"return 42;",
			"return (2097152) * (4294967296) + (1);",
			"return (2147483647) * (4294967296) + (4294967295);",
			"return ((0) - (2147483647) - (1)) * (4294967296);",
		},
		forbiddenTarget: []string{"9223372036854776000"},
	})
	goOutput := executeIntegerConstantsGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
