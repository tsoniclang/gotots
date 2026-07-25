package function_test

import (
	"path/filepath"
	"testing"
)

func TestLoopControlExecutesDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadLoopControlProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitLoopControl(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.LoopControl",
		assembly:  "GoToTSLoopControl",
		runnerSource: `Console.WriteLine(GoToTS.LoopControl.Index.Sum(0));
Console.WriteLine(GoToTS.LoopControl.Index.Sum(1));
Console.WriteLine(GoToTS.LoopControl.Index.Sum(5));
Console.WriteLine(GoToTS.LoopControl.Index.Sum(10));
`,
		requiredTarget: []string{
			"public static long Sum(long limit)",
			"for (long current = 0; current < limit; current++)",
			"if (current == 2)",
			"continue;",
			"if (total > (10))",
			"break;",
		},
	})
	goOutput := executeLoopControlGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
