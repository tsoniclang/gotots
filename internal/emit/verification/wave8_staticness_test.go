package emit_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveEightControlDemandDoesNotRewriteOrdinaryFunction(t *testing.T) {
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
				Directory: waveEightControlDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			scope := program.Roots()[0].Types().Scope()
			controlFree, err := emit.NewRoot(scope.Lookup("ControlFree"))
			if err != nil {
				t.Fatal(err)
			}
			deferOrder, err := emit.NewRoot(scope.Lookup("DeferOrder"))
			if err != nil {
				t.Fatal(err)
			}
			gotoState, err := emit.NewRoot(scope.Lookup("GotoState"))
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := emit.CompileWithOptions(
				program,
				[]emit.Root{controlFree},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			withControl, err := emit.CompileWithOptions(
				program,
				[]emit.Root{controlFree, deferOrder, gotoState},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			ordinaryArtifacts := materializeArtifacts(
				t,
				ordinary,
				t.TempDir(),
			)
			controlArtifacts := materializeArtifacts(
				t,
				withControl,
				t.TempDir(),
			)
			ordinaryFunction := targetFunctionText(
				t,
				ordinaryArtifacts.printed,
				"ControlFree",
			)
			controlFunction := targetFunctionText(
				t,
				controlArtifacts.printed,
				"ControlFree",
			)
			if ordinaryFunction != controlFunction {
				t.Fatalf(
					"control demand rewrote ControlFree\nordinary:\n%s\nwith control:\n%s",
					ordinaryFunction,
					controlFunction,
				)
			}
			for _, forbidden := range []string{
				"$go$recovery",
				"__gotots_defers_",
				"__gotots_goto_state_",
				"try {",
				"finally {",
			} {
				if strings.Contains(ordinaryFunction, forbidden) {
					t.Fatalf(
						"ordinary ControlFree contains %q:\n%s",
						forbidden,
						ordinaryFunction,
					)
				}
			}
		})
	}
}

func TestWaveEightCallableABIIsSignatureOwnedAcrossCarriers(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	abi := "(($0: gostring, $go$recovery?: GoRecovery) => void) | undefined"
	for _, required := range []string{
		"public Call: " + abi,
		"public readonly $value: " + abi,
		"GoPointer<" + abi + ", " + abi + ">",
		"RuntimeSlice.literal<" + abi + ">",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("signature-owned callable ABI lacks %q", required)
		}
	}
	ordinary := targetFunctionText(t, artifacts.printed, "recoveredTrace")
	ordinaryCallee := regexp.MustCompile(
		`const (__gotots_callee_[0-9]+) = invoke;`,
	).FindStringSubmatch(ordinary)
	if len(ordinaryCallee) != 2 ||
		!strings.Contains(ordinary, ordinaryCallee[1]+"();") ||
		strings.Contains(ordinary, ordinaryCallee[1]+"($go$recovery);") {
		t.Fatalf("ordinary invocation did not omit recovery authority:\n%s", ordinary)
	}
	deferredCall := regexp.MustCompile(
		`__gotots_callee_[0-9]+\(__gotots_argument_[0-9]+, \$go\$recovery\);`,
	)
	for _, name := range []string{
		"recoverFunctionField",
		"recoverFunctionPointer",
		"recoverFunctionSlice",
	} {
		target := targetFunctionText(t, artifacts.printed, name)
		if !deferredCall.MatchString(target) {
			t.Fatalf(
				"deferred %s invocation did not supply recovery authority:\n%s",
				name,
				target,
			)
		}
	}
	defined := targetFunctionText(
		t,
		artifacts.printed,
		"recoverDefinedFunction",
	)
	definedCallee := regexp.MustCompile(
		`const (__gotots_callee_[0-9]+): .* = selected\.\$value;`,
	).FindStringSubmatch(defined)
	if len(definedCallee) != 2 ||
		!regexp.MustCompile(
			regexp.QuoteMeta(definedCallee[1])+
				`\(__gotots_argument_[0-9]+, \$go\$recovery\);`,
		).MatchString(defined) {
		t.Fatalf(
			"defined deferred callable did not consume the signature ABI:\n%s",
			defined,
		)
	}
}

func TestWaveEightNestedGotoDispatchPreservesSourceControlTargets(
	t *testing.T,
) {
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
	required := map[string]bool{
		"GotoRangeStateContinueAudit":   true,
		"GotoStateLoopControl":          true,
		"GotoSwitchStateBreak":          true,
		"GotoSwitchStateFallthrough":    true,
		"GotoTypeSwitchClauseAudit":     true,
		"GotoTypeSwitchStateBreakAudit": true,
	}
	selected := roots[:0]
	for _, root := range roots {
		if required[root.Object().Name()] {
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	assertWaveEightControlTarget(
		t,
		targetFunctionText(t, artifacts.printed, "GotoStateLoopControl"),
		"for",
		"continue",
		"break",
	)
	assertWaveEightControlTarget(
		t,
		targetFunctionText(t, artifacts.printed, "GotoRangeStateContinue"),
		"for",
		"continue",
	)
	assertWaveEightControlTarget(
		t,
		targetFunctionText(t, artifacts.printed, "GotoSwitchStateBreak"),
		"switch",
		"break",
	)
	assertWaveEightControlTarget(
		t,
		targetFunctionText(t, artifacts.printed, "GotoTypeSwitchStateBreak"),
		"switch",
		"break",
	)
	fallthroughTarget := targetFunctionText(
		t,
		artifacts.printed,
		"GotoSwitchStateFallthrough",
	)
	if !strings.Contains(fallthroughTarget, "__gotots_goto_state_") ||
		!strings.Contains(fallthroughTarget, "case 2:") {
		t.Fatalf(
			"switch fallthrough did not follow its clause-local state:\n%s",
			fallthroughTarget,
		)
	}
	directTypeSwitch := targetFunctionText(
		t,
		artifacts.printed,
		"GotoTypeSwitchClause",
	)
	if strings.Contains(directTypeSwitch, "__gotots_goto_state_") {
		t.Fatalf(
			"direct type-switch goto selected a state machine:\n%s",
			directTypeSwitch,
		)
	}
}

func TestWaveEightStateTransitionsOnlyFollowFallthrough(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("GotoState"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	target := targetFunctionText(t, artifacts.printed, "GotoState")
	for description, pattern := range map[string]string{
		"goto": `continue __gotots_goto_dispatch_[0-9]+;\s+` +
			`__gotots_goto_state_[0-9]+ = [0-9]+;\s+` +
			`continue __gotots_goto_dispatch_[0-9]+;`,
		"return": `return [^;]+;\s+` +
			`__gotots_goto_state_[0-9]+ = [0-9]+;`,
	} {
		if regexp.MustCompile(pattern).MatchString(target) {
			t.Fatalf(
				"state transition follows terminal %s:\n%s",
				description,
				target,
			)
		}
	}
}

func assertWaveEightControlTarget(
	t *testing.T,
	target string,
	construct string,
	branches ...string,
) {
	t.Helper()
	pattern := regexp.MustCompile(
		`(__gotots_control_target_[0-9]+): ` + construct,
	)
	match := pattern.FindStringSubmatch(target)
	if len(match) != 2 {
		t.Fatalf("nested goto lacks a source %s target:\n%s", construct, target)
	}
	for _, branch := range branches {
		if !strings.Contains(target, branch+" "+match[1]+";") {
			t.Fatalf(
				"source %s %s was captured by generated dispatch:\n%s",
				construct,
				branch,
				target,
			)
		}
	}
}
