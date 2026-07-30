package emit_test

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveSevenGenericMethodAdaptersJoinExactABI(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().
					Lookup("AuditGenericMethodAdapters"),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			assertGenericMethodAdapterShape(t, artifacts.printed)
			sourceModule := sourceModuleForExport(
				t,
				artifacts,
				workingDirectory,
				"AuditGenericMethodAdapters",
			)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditGenericMethodAdapters } from "`+sourceModule+`";

const values = AuditGenericMethodAdapters();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(
				t,
				workingDirectory,
				append(artifacts.paths, runner),
			)
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"AuditGenericMethodAdapters",
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"generic method adapters differ\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func assertGenericMethodAdapterShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function AuditGenericMethodAdapters",
		"$go$binary_equal_",
		"=>",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("generic method artifact lacks %q:\n%s", required, printed)
		}
	}
	if strings.Contains(printed, ".call(") ||
		strings.Contains(printed, ".apply(") ||
		strings.Contains(printed, ".bind(") ||
		strings.Contains(printed, "function ComparableBox_Same") {
		t.Fatalf("generic method adapter uses dynamic callable APIs:\n%s", printed)
	}
	capabilityFirst := regexp.MustCompile(
		`\.Same\(\$goCapability_[0-9a-f]+, `,
	)
	if count := len(capabilityFirst.FindAllString(printed, -1)); count != 2 {
		t.Fatalf(
			"generic method value/expression ABI has %d capability-first calls, want 2:\n%s",
			count,
			printed,
		)
	}
	receiverFirst := regexp.MustCompile(
		`\.Same\((?:__gotots_receiver_|\$argument0), \$goCapability_`,
	)
	if receiverFirst.MatchString(printed) {
		t.Fatalf("generic method adapter emitted receiver before capability:\n%s", printed)
	}
}
