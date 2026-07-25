package function_test

import (
	"path/filepath"
	"testing"
)

func TestExpressionSwitchExecutesDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadExpressionSwitchProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitExpressionSwitch(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.ExpressionSwitch",
		assembly:  "GoToTSExpressionSwitch",
		runnerSource: `Console.WriteLine(GoToTS.ExpressionSwitch.Index.Classify(0));
Console.WriteLine(GoToTS.ExpressionSwitch.Index.Classify(1));
Console.WriteLine(GoToTS.ExpressionSwitch.Index.Classify(2));
Console.WriteLine(GoToTS.ExpressionSwitch.Index.Classify(9));
`,
		requiredTarget: []string{
			"public static long Classify(long value)",
			"switch (current)",
			"case 0:",
			"case 1:",
			"case 2:",
			"default:",
		},
	})
	goOutput := executeExpressionSwitchGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
