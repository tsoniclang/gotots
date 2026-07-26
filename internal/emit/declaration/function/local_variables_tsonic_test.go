package function_test

import (
	"path/filepath"
	"testing"
)

func TestLocalVariablesExecuteDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadLocalVariablesProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitLocalVariables(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.LocalVariables",
		assembly:  "GoToTSLocalVariables",
		runnerSource: `Console.WriteLine(GoToTS.LocalVariables.Index.Compute(0));
Console.WriteLine(GoToTS.LocalVariables.Index.Compute(5));
Console.WriteLine(GoToTS.LocalVariables.Index.Compute(20));
Console.WriteLine(GoToTS.LocalVariables.Index.LateOuter(0));
Console.WriteLine(GoToTS.LocalVariables.Index.LateOuter(5));
`,
		requiredTarget: []string{
			"public static long Compute(long input)",
			"long @base = input",
			"long base__shadow_1 = @base",
			"long left = base__shadow_1",
			"long right = base__shadow_1",
			"long __u3c0_ = left + right",
			"long __gotots_assign_0 = right",
			"long __gotots_assign_1 = left",
			"public static long LateOuter(long input)",
			"long value__shadow_1 = input + (1)",
			"long value = input + (2)",
			"long __go_class = value + (3)",
			"long __go_arguments = __go_class + (4)",
		},
	})
	goOutput := executeLocalVariablesGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
