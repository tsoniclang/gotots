package provider_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestLinkedProviderDefinedValuesUseCertifiedRepresentation(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerdefined\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerdefined

import (
	"context"
	"io/fs"
	"iter"
	"syscall"
	"time"
	"unsafe"
)

func Cancel(cancel context.CancelFunc) {
	cancel()
}

func Walk(
	callback fs.WalkDirFunc,
	path string,
	entry fs.DirEntry,
	failure error,
) error {
	return callback(path, entry, failure)
}

func DurationMath(value time.Duration) time.Duration {
	value += time.Second
	if value > time.Minute {
		return -value
	}
	return value
}

func DurationZero() time.Duration {
	var value time.Duration
	return value
}

func DurationSwitch(value time.Duration) bool {
	switch value {
	case 0:
		return true
	default:
		return false
	}
}

func DurationMap(value time.Duration) int {
	return map[time.Duration]int{time.Second: 1}[value]
}

func DurationRange(limit time.Duration) time.Duration {
	var last time.Duration
	for index := range limit {
		last = index
	}
	return last
}

func DurationMaximum(left time.Duration, right time.Duration) time.Duration {
	return max(left, right)
}

func DurationToInt(value time.Duration) int64 {
	return int64(value)
}

func IntToDuration(value int64) time.Duration {
	return time.Duration(value)
}

func negateDuration[T ~int64](value T) T {
	return -value
}

func DurationNegate(value time.Duration) time.Duration {
	return negateDuration(value)
}

func DurationIndex(values []int, index time.Duration) int {
	return values[index]
}

func DurationWindow(values []int, low time.Duration, high time.Duration) []int {
	return values[low:high]
}

func DurationMake(length time.Duration) []int {
	return make([]int, length)
}

func DurationChannel(size time.Duration) chan int {
	return make(chan int, size)
}

type Timed struct {
	Delay time.Duration
}

func DurationStruct(value time.Duration) Timed {
	return Timed{Delay: value}
}

func DurationArray(value time.Duration) [2]time.Duration {
	return [2]time.Duration{value}
}

func DurationSlice(value time.Duration) []time.Duration {
	return []time.Duration{value}
}

func DurationValueMap(value time.Duration) map[string]time.Duration {
	return map[string]time.Duration{"delay": value}
}

func DurationPointer(value time.Duration) *time.Duration {
	return &value
}

func ErrnoPointer(value syscall.Errno) unsafe.Pointer {
	return unsafe.Pointer(value)
}

func PointerErrno(value unsafe.Pointer) syscall.Errno {
	return syscall.Errno(value)
}

func SequenceTotal(sequence iter.Seq[int]) int {
	total := 0
	for value := range sequence {
		total += value
	}
	return total
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	printed := artifacts.printed
	for _, required := range []string{
		"TimeDurationValueOperations.$project",
		"TimeDurationValueOperations.$wrap",
		"case 0n:",
		"GoMapHash.bigint",
		"scalars.int64",
		"let __gotots_range_index_0 = 0n",
		"goIntegerMax",
		"TimeDurationValueOperations.$wrap(-named_time.TimeDurationValueOperations.$project($argument0))",
		"values.get(named_time.TimeDurationValueOperations.$project(index))",
		"values.slice(named_time.TimeDurationValueOperations.$project(low), named_time.TimeDurationValueOperations.$project(high), null)",
		"RuntimeSlice.make<int>(named_time.TimeDurationValueOperations.$project(length), null, 0)",
		"GoChannel.make<int>(named_time.TimeDurationValueOperations.$project(size)",
		"GoArray.literal<time__from_gostdlib.Duration, 2>(2, named_time.TimeDurationValueOperations.$wrap(0n)",
		"GoMap.make<gostring, time__from_gostdlib.Duration>(named_time.TimeDurationValueOperations.$wrap(0n)",
		"GoUnsafePointer.fromInteger(named_syscall.SyscallErrnoValueOperations.$project(value), 0n)",
		"named_syscall.SyscallErrnoValueOperations.$wrap(GoUnsafePointer.toInteger(value, 0n))",
		"globalThis.Number(BigInt.asIntN(64, named_time.TimeDurationValueOperations.$project(value)))",
		"named_time.TimeDurationValueOperations.$wrap(BigInt.asIntN(64, goNumberToBigInt(value)))",
		"IterSeqValueOperations.$project",
		"cancel: (() => Awaitable<void>) | undefined",
		"callback: (($0: gostring, $1:",
		"const __gotots_callee_0 = cancel;",
		"const __gotots_callee_1 = callback;",
		"await __gotots_callee_0();",
		"return await __gotots_callee_1(",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("provider defined-value artifact lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"cancel.$value",
		"callback.$value",
		"sequence.$value",
		"value.$value",
		"context__from_gostdlib.CancelFunc",
		"fs__from_gostdlib.WalkDirFunc",
		"new Duration(",
		"GoMapHash.number(named_time.TimeDurationValueOperations.$project",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("provider defined-value artifact contains %q:\n%s", forbidden, printed)
		}
	}
	if count := strings.Count(
		printed,
		"IterSeqValueOperations.$project",
	); count != 1 {
		t.Fatalf(
			"provider iterator projection count = %d, want one:\n%s",
			count,
			printed,
		)
	}
}

func TestAtomicComparableFacetsMatchGo(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/atomiccomparable\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package atomiccomparable

import (
	"fmt"
	"sync/atomic"
)

func Facts() string {
	var zero atomic.Uint64
	var same atomic.Uint64
	var one atomic.Uint64
	one.Add(1)
	values := map[atomic.Uint64]string{
		zero: "zero",
		one:  "one",
	}
	return fmt.Sprintf(
		"%t %t %s %s",
		zero == same,
		zero == one,
		values[same],
		values[one],
	)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Facts"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"SyncAtomicUint64Operations.$equal",
		"SyncAtomicUint64Operations.$hash",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"atomic comparable artifact lacks %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "atomiccomparable" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("atomic comparable package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Facts"},
		"console.log(await Facts());\n",
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/atomiccomparable"
)

func main() {
	fmt.Println(fixture.Facts())
}
`)
	sourceContext, sourceCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer sourceCancel()
	command := exec.CommandContext(sourceContext, "go", "run", ".")
	command.Dir = runnerDirectory
	sourceOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go atomic comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"atomic comparable differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
}

func TestSyncComparableFacetsMatchGo(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/synccomparable\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package synccomparable

import (
	"fmt"
	"sync"
)

type State struct {
	Mutex sync.Mutex
	Once sync.Once
	ReadWrite sync.RWMutex
	Wait sync.WaitGroup
	Condition sync.Cond
}

func Facts() string {
	var left State
	var right State
	var zero State
	zeroValues := map[State]string{left: "zero"}
	before := fmt.Sprintf("%t %s", left == right, zeroValues[right])

	left.Mutex.Lock()
	left.Mutex.Unlock()
	right.Mutex.Lock()
	right.Mutex.Unlock()
	mutexValues := map[sync.Mutex]string{left.Mutex: "mutex"}

	left.Once.Do(func() {})
	right.Once.Do(func() {})
	onceValues := map[sync.Once]string{left.Once: "once"}

	left.ReadWrite.RLock()
	left.ReadWrite.RUnlock()
	right.ReadWrite.RLock()
	right.ReadWrite.RUnlock()
	readWriteValues := map[sync.RWMutex]string{left.ReadWrite: "rw"}

	left.Wait.Add(1)
	left.Wait.Done()
	right.Wait.Add(1)
	right.Wait.Done()
	waitValues := map[sync.WaitGroup]string{left.Wait: "wait"}

	left.Condition.Signal()
	right.Condition.Signal()
	conditionValues := map[sync.Cond]string{left.Condition: "cond"}

	return fmt.Sprintf(
		"%s | %t %t %s | %t %t %s | %t %t %s | %t %t %s | %t %t %s %s",
		before,
		left.Mutex == zero.Mutex,
		left.Mutex == right.Mutex,
		mutexValues[right.Mutex],
		left.Once == zero.Once,
		left.Once == right.Once,
		onceValues[right.Once],
		left.ReadWrite == zero.ReadWrite,
		left.ReadWrite == right.ReadWrite,
		readWriteValues[right.ReadWrite],
		left.Wait == zero.Wait,
		left.Wait == right.Wait,
		waitValues[right.Wait],
		left.Condition == zero.Condition,
		left.Condition == right.Condition,
		conditionValues[left.Condition],
		conditionValues[right.Condition],
	)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Facts"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, owner := range []string{
		"SyncCondOperations",
		"SyncMutexOperations",
		"SyncOnceOperations",
		"SyncRWMutexOperations",
		"SyncWaitGroupOperations",
	} {
		for _, operation := range []string{"$equal", "$hash"} {
			required := owner + "." + operation
			if !strings.Contains(artifacts.printed, required) {
				t.Fatalf("sync comparable artifact lacks %q", required)
			}
		}
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "synccomparable" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("sync comparable package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Facts"},
		"console.log(await Facts());\n",
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/synccomparable"
)

func main() {
	fmt.Println(fixture.Facts())
}
`)
	sourceContext, sourceCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer sourceCancel()
	command := exec.CommandContext(sourceContext, "go", "run", ".")
	command.Dir = runnerDirectory
	sourceOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go sync comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"sync comparable differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
}
