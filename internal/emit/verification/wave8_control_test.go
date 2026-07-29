package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveEightDeferAndPanicCompileThroughPublicPipeline(t *testing.T) {
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
			testWaveEightDeferAndPanic(t, testCase.options)
		})
	}
}

func testWaveEightDeferAndPanic(t *testing.T, options emit.Options) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := roots[:0]
	for _, root := range roots {
		if !strings.HasPrefix(root.Object().Name(), "Goto") {
			selected = append(selected, root)
		}
	}
	roots = selected
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"class GoPanic",
		"class GoRecovery",
		"try {",
		"finally {",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("Wave 8 artifacts lack %q:\n%s", required, artifacts.printed)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
    DeferEvaluationAndCopy,
    DeferBuiltins,
    DeferOrder,
    DeferPointerReceiver,
    DeferReceiverCopy,
    DeferVariadicAndMultiple,
    NamedResultMutation,
    NilDeferredTiming,
    NilInterfaceDeferredTiming,
    NilValueReceiverTiming,
    OrdinaryReturnUnwind,
    PanicNilContracts,
    PanicNilIdentity,
    RecoverContinuesUnwind,
    RecoverDirectness,
    RecoverOnNormalReturn,
    RecoverOutsideDefer,
    RecoverPanic,
    RecoveryCallableForms,
    ReplacementPanic,
    RuntimeFaultIdentity,
} from "`+artifacts.sourceModule+`";

const output: string[] = [];
const deferred = DeferOrder();
output.push(deferred[0], String(deferred[1]));
output.push(RecoverPanic());
const directness = RecoverDirectness();
output.push(String(directness[0]), String(directness[1]));
output.push(String(PanicNilIdentity()));
const panicNil = PanicNilContracts();
output.push(
    String(panicNil[0]),
    String(panicNil[1]),
    String(panicNil[2]),
    String(panicNil[3]),
    panicNil[4],
);
const runtimeFault = RuntimeFaultIdentity();
output.push(
    String(runtimeFault[0]),
    String(runtimeFault[1]),
    String(runtimeFault[2]),
    String(runtimeFault[3]),
    runtimeFault[4],
);
output.push(
    DeferEvaluationAndCopy(),
    DeferReceiverCopy(),
    DeferPointerReceiver(),
);
output.push(DeferVariadicAndMultiple());
const builtins = DeferBuiltins();
output.push(
    String(builtins[0]),
    String(builtins[1]),
    String(builtins[2]),
);
output.push(RecoveryCallableForms());
output.push(String(RecoverOnNormalReturn()));
output.push(String(RecoverOutsideDefer()));
output.push(RecoverContinuesUnwind(), ReplacementPanic());
output.push(String(NamedResultMutation()), OrdinaryReturnUnwind());
const nilDeferred = NilDeferredTiming();
output.push(String(nilDeferred[0]), String(nilDeferred[1]));
const nilInterface = NilInterfaceDeferredTiming();
output.push(String(nilInterface[0]), String(nilInterface[1]));
const nilReceiver = NilValueReceiverTiming();
output.push(String(nilReceiver[0]), String(nilReceiver[1]));
console.log(output.join(" "));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	paths := append(artifacts.paths, runner)
	waveThreeTypecheck(t, workingDirectory, paths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeWaveEightGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"Wave 8 output differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func TestWaveEightDirectGotoCompilesThroughPublicPipeline(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := roots[:0]
	for _, root := range roots {
		switch root.Object().Name() {
		case "GotoDirectMultiple",
			"GotoForward",
			"GotoLabeledLoop",
			"GotoLoop",
			"GotoSameLabelA",
			"GotoSameLabelB",
			"GotoSwitchFallthrough",
			"GotoSwitchClause",
			"GotoSwitchClauseLoop",
			"GotoTypeSwitchClauseAudit":
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if strings.Contains(artifacts.printed, "__gotots_goto_state_") {
		t.Fatalf(
			"direct goto fixtures used the state machine:\n%s",
			artifacts.printed,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
    GotoForward,
    GotoDirectMultiple,
    GotoLabeledLoop,
    GotoLoop,
	    GotoSameLabelA,
	    GotoSameLabelB,
	    GotoSwitchFallthrough,
	    GotoSwitchClause,
	    GotoSwitchClauseLoop,
	    GotoTypeSwitchClauseAudit,
} from "`+artifacts.sourceModule+`";

console.log(
    String(GotoForward(-4)),
    String(GotoForward(4)),
    String(GotoDirectMultiple(5)),
    String(GotoLabeledLoop(5)),
    String(GotoLoop(5)),
    String(GotoSameLabelA(5)),
    String(GotoSameLabelB(5)),
	    String(GotoSwitchClause(1)),
	    String(GotoSwitchClause(4)),
	    String(GotoSwitchClauseLoop(2)),
	    String(GotoSwitchFallthrough(1)),
	    String(GotoSwitchFallthrough(2)),
	    String(GotoTypeSwitchClauseAudit()),
);
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
	goOutput := executeWaveEightGotoGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"Wave 8 goto output differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func TestWaveEightStateGotoCompilesThroughPublicPipeline(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := roots[:0]
	names := map[string]bool{
		"FallthroughControl":            true,
		"GotoDeferRange":                true,
		"GotoDeferRangeAudit":           true,
		"GotoFromRange":                 true,
		"GotoFromRangeAudit":            true,
		"GotoNestedState":               true,
		"GotoState":                     true,
		"GotoStateAddress":              true,
		"GotoStateDeclarations":         true,
		"GotoStateFreshAddress":         true,
		"GotoStateStatic":               true,
		"GotoStateLoopControl":          true,
		"GotoRepeatedDefer":             true,
		"GotoRangeStateContinueAudit":   true,
		"GotoSwitchStateBreak":          true,
		"GotoSwitchStateFallthrough":    true,
		"GotoTypeSwitchStateBreakAudit": true,
		"GotoVoid":                      true,
		"GotoVoidAudit":                 true,
		"GotoWithDefer":                 true,
		"LabeledControl":                true,
	}
	for _, root := range roots {
		if names[root.Object().Name()] {
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(artifacts.printed, "__gotots_goto_state_") {
		t.Fatalf("non-structural goto lacks a state machine:\n%s", artifacts.printed)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
    FallthroughControl,
    GotoDeferRangeAudit,
    GotoFromRangeAudit,
    GotoNestedState,
    GotoRepeatedDefer,
    GotoState,
    GotoStateAddress,
    GotoStateDeclarations,
    GotoStateFreshAddress,
    GotoStateLoopControl,
    GotoStateStatic,
    GotoRangeStateContinueAudit,
    GotoSwitchStateBreak,
    GotoSwitchStateFallthrough,
    GotoTypeSwitchStateBreakAudit,
    GotoVoidAudit,
    GotoWithDefer,
    LabeledControl,
} from "`+artifacts.sourceModule+`";

console.log(
    String(GotoState(5)),
    String(GotoStateAddress(5)),
    String(GotoStateDeclarations(2)),
    String(GotoStateFreshAddress()),
    String(GotoStateLoopControl()),
    String(GotoStateStatic(2)),
    String(GotoRangeStateContinueAudit()),
    String(GotoSwitchStateBreak(1)),
    String(GotoSwitchStateFallthrough(1)),
    String(GotoSwitchStateFallthrough(2)),
    String(GotoTypeSwitchStateBreakAudit()),
    String(GotoNestedState(5)),
    String(GotoWithDefer(5)),
    String(GotoRepeatedDefer(3)),
    String(GotoVoidAudit()),
    String(GotoDeferRangeAudit()),
    String(GotoFromRangeAudit()),
    String(LabeledControl(5)),
    String(FallthroughControl(1)),
    String(FallthroughControl(2)),
);
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
	goOutput := executeWaveEightStateGotoGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"Wave 8 state-goto output differs\nTypeScript:\n%s\nGo:\n%s",
			targetOutput,
			goOutput,
		)
	}
}

func executeWaveEightGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveEightControlDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave8")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave8control v0.0.0

replace example.com/wave8control => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave8control"
)

func main() {
	trace, value := values.DeferOrder()
	fmt.Print(trace, " ", value, " ")
	fmt.Print(values.RecoverPanic(), " ")
	direct, indirect := values.RecoverDirectness()
	fmt.Print(direct, " ", indirect, " ")
	fmt.Print(values.PanicNilIdentity(), " ")
	nilError, nilContract, nilRuntime, nilIdentity, nilMessage :=
		values.PanicNilContracts()
	fmt.Print(
		nilError, " ",
		nilContract, " ",
		nilRuntime, " ",
		nilIdentity, " ",
		nilMessage, " ",
	)
	asError, asContract, asRuntime, asPanicNil, message :=
		values.RuntimeFaultIdentity()
	fmt.Print(
		asError, " ",
		asContract, " ",
		asRuntime, " ",
		asPanicNil, " ",
		message, " ",
	)
	fmt.Print(values.DeferEvaluationAndCopy(), " ")
	fmt.Print(values.DeferReceiverCopy(), " ")
	fmt.Print(values.DeferPointerReceiver(), " ")
	fmt.Print(values.DeferVariadicAndMultiple(), " ")
	mapCleared, copied, sliceCleared := values.DeferBuiltins()
	fmt.Print(mapCleared, " ", copied, " ", sliceCleared, " ")
	fmt.Print(values.RecoveryCallableForms(), " ")
	fmt.Print(values.RecoverOnNormalReturn(), " ")
	fmt.Print(values.RecoverOutsideDefer(), " ")
	fmt.Print(values.RecoverContinuesUnwind(), " ")
	fmt.Print(values.ReplacementPanic(), " ")
	fmt.Print(values.NamedResultMutation(), " ")
	fmt.Print(values.OrdinaryReturnUnwind(), " ")
	nilReached, nilRecovered := values.NilDeferredTiming()
	fmt.Print(nilReached, " ", nilRecovered, " ")
	interfaceReached, interfaceRecovered := values.NilInterfaceDeferredTiming()
	fmt.Print(interfaceReached, " ", interfaceRecovered, " ")
	receiverReached, receiverRecovered := values.NilValueReceiverTiming()
	fmt.Println(receiverReached, receiverRecovered)
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func executeWaveEightGotoGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveEightControlDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave8-goto")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave8control v0.0.0

replace example.com/wave8control => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave8control"
)

func main() {
	fmt.Println(
		values.GotoForward(-4),
		values.GotoForward(4),
		values.GotoDirectMultiple(5),
		values.GotoLabeledLoop(5),
		values.GotoLoop(5),
		values.GotoSameLabelA(5),
		values.GotoSameLabelB(5),
			values.GotoSwitchClause(1),
			values.GotoSwitchClause(4),
			values.GotoSwitchClauseLoop(2),
			values.GotoSwitchFallthrough(1),
			values.GotoSwitchFallthrough(2),
			values.GotoTypeSwitchClauseAudit(),
	)
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func executeWaveEightStateGotoGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveEightControlDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave8-state-goto")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave8control v0.0.0

replace example.com/wave8control => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave8control"
)

func main() {
	fmt.Println(
		values.GotoState(5),
		values.GotoStateAddress(5),
		values.GotoStateDeclarations(2),
		values.GotoStateFreshAddress(),
		values.GotoStateLoopControl(),
		values.GotoStateStatic(2),
		values.GotoRangeStateContinueAudit(),
		values.GotoSwitchStateBreak(1),
		values.GotoSwitchStateFallthrough(1),
		values.GotoSwitchStateFallthrough(2),
		values.GotoTypeSwitchStateBreakAudit(),
		values.GotoNestedState(5),
		values.GotoWithDefer(5),
		values.GotoRepeatedDefer(3),
		values.GotoVoidAudit(),
		values.GotoDeferRangeAudit(),
		values.GotoFromRange([]int{1, 2, -1, 8}),
		values.LabeledControl(5),
		values.FallthroughControl(1),
		values.FallthroughControl(2),
	)
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func waveEightControlDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"control",
		"wave8",
	)
}
