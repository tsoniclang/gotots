package function_test

import (
	"path/filepath"
	"testing"
)

func TestMultipleResultsExecuteDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadMultipleResultsProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitMultipleResultsProject(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.MultipleResults",
		assembly:  "GoToTSMultipleResults",
		runnerSource: `var pair = GoToTS.MultipleResults.Index.Pair(3);
Console.WriteLine($"{pair.Item1} {pair.Item2.ToString().ToLowerInvariant()}");
var forwarded = GoToTS.MultipleResults.Index.Forward(-2);
Console.WriteLine($"{forwarded.Item1} {forwarded.Item2.ToString().ToLowerInvariant()}");
Console.WriteLine(GoToTS.MultipleResults.Index.Consume(4));
Console.WriteLine(GoToTS.MultipleResults.Index.Consume(-4));
Console.WriteLine(GoToTS.MultipleResults.Index.Reassign(7));
Console.WriteLine(GoToTS.MultipleResults.Index.KeepFirst(9));
Console.WriteLine(GoToTS.MultipleResults.Index.Discard(11));
Console.WriteLine(GoToTS.MultipleResults.Index.AddPair(5));
`,
		requiredTarget: []string{
			"public static (long, bool) Pair(long value)",
			"return (value + (1), value >= (0));",
			"(long, bool) __gotots_results_0 = Pair(value);",
			"long next = __gotots_results_0.Item1;",
			"bool positive = __gotots_results_0.Item2;",
			"return Add(__gotots_results_3.Item1, __gotots_results_3.Item2);",
		},
		forbiddenTarget: []string{
			"out ",
			"dynamic",
			"object[]",
		},
	})
	goOutput := executeMultipleResultsGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}
