package function_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

func executeCallableNilGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(callableNilProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/callablenil v0.0.0

replace example.com/callablenil => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/callablenil"
)

func nilCallResult() (result string) {
	defer func() {
		if recover() != nil {
			result = fmt.Sprintf("panic:%d", values.TraceValue())
		}
	}()
	values.NilCallOrder()
	return "no-panic"
}

func nilVoidCallResult() (result string) {
	defer func() {
		if recover() != nil {
			result = fmt.Sprintf("panic:%d", values.TraceValue())
		}
	}()
	values.NilVoidCallOrder()
	return "no-panic"
}

func main() {
	defined := values.DefinedFromRaw(values.Increment)
	other := values.OtherFromDefined(defined)
	fmt.Println(values.IsNilRaw(nil))
	fmt.Println(values.IsNilDefined(nil))
	fmt.Println(values.IsNonNilAlias(defined))
	fmt.Println(values.LocalNil())
	fmt.Println(values.IsNilDefined(values.NilResult()))
	fmt.Println(values.IsNilDefined(values.ConvertNilRaw()))
	fmt.Println(values.IsNilRaw(values.ConvertNilDefined()))
	fmt.Println(values.IsNilDefined(values.TransformFromOther(nil)))
	fmt.Println(values.ApplyDefined(defined, 4))
	fmt.Println(values.ApplyDefined(values.TransformFromOther(other), 5))
	fmt.Println(values.ApplyRawAlias(values.Increment, 6))
	fmt.Println(values.ApplyDefined(values.Offset(3), 7))
	fmt.Println(values.PassRawToDefined(11))
	fmt.Println(values.ApplyDefined(values.ReturnRawAsDefined(), 12))
	fmt.Println(values.ReturnDefinedAsRaw()(13))
	fmt.Println(values.AssignDefinedToRaw(defined, 14))
	fmt.Println(values.LocalVarCall(defined, 17))
	fmt.Println(values.LocalImplicitDefined(18))
	fmt.Println(values.ApplyDefined(values.ImplicitOffset(4), 15))
	fmt.Println(values.LocalDefined(10))
	fmt.Println(values.NewRawIsNil())
	fmt.Println(values.NewDefinedIsNil())
	fmt.Println(values.StoreThroughPointer(8))
	fmt.Println(values.PackageIsNil())
	values.SetPackage(defined)
	fmt.Println(values.CallPackage(9))
	fmt.Println(values.ShortCircuit(false))
	fmt.Println(values.ShortCircuit(true))
	fmt.Println(values.Conditional(true))
	fmt.Println(values.Conditional(false))
	values.ResetTrace()
	fmt.Println(nilCallResult())
	values.ResetTrace()
	fmt.Println(nilVoidCallResult())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
