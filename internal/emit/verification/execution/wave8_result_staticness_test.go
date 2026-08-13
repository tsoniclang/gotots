package emit_test

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
			repeatedDefer, err := emit.NewRoot(scope.Lookup("GotoRepeatedDefer"))
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
				[]emit.Root{controlFree, deferOrder, gotoState, repeatedDefer},
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
			deferredFunction := targetFunctionText(
				t,
				controlArtifacts.printed,
				"DeferOrder",
			)
			for _, forbidden := range []string{
				".push(",
				"goDeferPop",
				"while (",
			} {
				if strings.Contains(deferredFunction, forbidden) {
					t.Fatalf(
						"single-entry static defer contains %q:\n%s",
						forbidden,
						deferredFunction,
					)
				}
			}
			if !strings.Contains(deferredFunction, "finally {") ||
				strings.Count(deferredFunction, "__gotots_deferred_") < 2 {
				t.Fatalf(
					"single-entry static defer lacks direct slots:\n%s",
					deferredFunction,
				)
			}
			repeatedFunction := targetFunctionText(
				t,
				controlArtifacts.printed,
				"GotoRepeatedDefer",
			)
			for _, required := range []string{
				".push(",
				"goDeferPop",
				"while (",
			} {
				if !strings.Contains(repeatedFunction, required) {
					t.Fatalf(
						"repeated defer lacks dynamic storage %q:\n%s",
						required,
						repeatedFunction,
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
	abi := "(($0: gostring) => void) | undefined"
	for _, required := range []string{
		"public Call: " + abi,
		"public readonly $value: " + abi,
		"Pointer<" + abi + ">",
		"RuntimeSlice.literal<" + abi + ">",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("signature-owned callable ABI lacks %q", required)
		}
	}
	if strings.Contains(artifacts.printed, "$Value =") {
		t.Fatal("callable carrier exposed a representation-only type parameter")
	}
	if strings.Contains(artifacts.printed, "($0: gostring, $go$recovery") {
		t.Fatalf("source callable carrier exposes recovery authority")
	}
	ordinary := targetFunctionText(t, artifacts.printed, "recoveredTrace")
	ordinaryCallee := regexp.MustCompile(
		`const (__gotots_callee_[0-9]+) = invoke;`,
	).FindStringSubmatch(ordinary)
	if len(ordinaryCallee) != 2 ||
		!strings.Contains(
			ordinary,
			ordinaryCallee[1]+` ?? GoPanic.raiseRuntime("call of nil function"))();`,
		) ||
		strings.Contains(ordinary, ordinaryCallee[1]+"($go$recovery);") {
		t.Fatalf("ordinary invocation did not omit recovery authority:\n%s", ordinary)
	}
	for _, name := range []string{
		"recoverFunctionField",
		"recoverFunctionPointer",
		"recoverFunctionSlice",
	} {
		target := targetFunctionText(t, artifacts.printed, name)
		callee := regexp.MustCompile(
			`const (__gotots_callee_[0-9]+).* = `,
		).FindStringSubmatch(target)
		if len(callee) != 2 ||
			!strings.Contains(target, ".resolve("+callee[1]+")") ||
			!regexp.MustCompile(
				`__gotots_deferred_[0-9]+\(\$go\$recovery, __gotots_argument_[0-9]+\)`,
			).MatchString(target) ||
			strings.Contains(target, callee[1]+"($go$recovery") {
			t.Fatalf(
				"deferred %s invocation did not privately resolve recovery authority:\n%s",
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
		!strings.Contains(
			defined,
			".resolve("+definedCallee[1]+")",
		) ||
		!regexp.MustCompile(
			`__gotots_deferred_[0-9]+\(\$go\$recovery, __gotots_argument_[0-9]+\)`,
		).MatchString(defined) {
		t.Fatalf(
			"defined deferred callable did not consume the private registry ABI:\n%s",
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

func TestWaveEightControlledResultCaptureKeepsExactTupleType(t *testing.T) {
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
		case "DeferBuiltins", "RuntimeFaultIdentity":
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"DeferBuiltins", "RuntimeFaultIdentity"} {
		function := waveEightResultFunction(t, emission, name)
		resultType, ok := function.Type().(tsgo.TupleTypeNode)
		if !ok {
			t.Fatalf("%s result = %T, want tuple", name, function.Type())
		}
		body, ok := function.Body().(tsgo.Block)
		if !ok {
			t.Fatalf("%s body = %T, want block", name, function.Body())
		}
		captures := waveEightResultCaptures(body.Statements())
		if len(captures) != 1 {
			t.Fatalf("%s controlled result captures = %d, want 1", name, len(captures))
		}
		captureType, ok := captures[0].Type().(tsgo.TupleTypeNode)
		if !ok {
			t.Fatalf(
				"%s controlled result capture type = %T, want tuple",
				name,
				captures[0].Type(),
			)
		}
		if _, ok := captures[0].Initializer().(tsgo.ArrayLiteralExpression); !ok {
			t.Fatalf(
				"%s controlled result initializer = %T, want array literal",
				name,
				captures[0].Initializer(),
			)
		}
		got := encodeWaveEightTupleType(t, captureType)
		want := encodeWaveEightTupleType(t, resultType)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s controlled result capture does not preserve its result tuple", name)
		}
	}
}

func waveEightResultFunction(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			identifier, ok := function.Name().(tsgo.Identifier)
			if ok && identifier.Text() == name {
				return function
			}
		}
	}
	t.Fatalf("generated function %s not found", name)
	return nil
}

func waveEightResultCaptures(
	statements []tsgo.Statement,
) []tsgo.VariableDeclaration {
	var result []tsgo.VariableDeclaration
	for _, statement := range statements {
		switch statement := statement.(type) {
		case tsgo.VariableStatement:
			for _, declaration := range statement.DeclarationList().Declarations() {
				name, ok := declaration.Name().(tsgo.Identifier)
				if ok &&
					strings.HasPrefix(name.Text(), "__gotots_results_") &&
					declaration.Initializer() != nil {
					if _, ok := declaration.Initializer().(tsgo.ArrayLiteralExpression); ok {
						result = append(result, declaration)
					}
				}
			}
		case tsgo.Block:
			result = append(result, waveEightResultCaptures(statement.Statements())...)
		case tsgo.LabeledStatement:
			result = append(
				result,
				waveEightResultCaptures([]tsgo.Statement{statement.Statement()})...,
			)
		case tsgo.TryStatement:
			result = append(
				result,
				waveEightResultCaptures(statement.TryBlock().Statements())...,
			)
			if statement.CatchClause() != nil {
				result = append(
					result,
					waveEightResultCaptures(
						statement.CatchClause().Block().Statements(),
					)...,
				)
			}
			if statement.FinallyBlock() != nil {
				result = append(
					result,
					waveEightResultCaptures(statement.FinallyBlock().Statements())...,
				)
			}
		}
	}
	return result
}

func encodeWaveEightTupleType(
	t *testing.T,
	target tsgo.TupleTypeNode,
) []byte {
	t.Helper()
	encoded, err := tsgo.EncodeNode(target)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
