package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

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
	"reflect"
	"time"
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

func SequenceTotal(sequence iter.Seq[int]) int {
	total := 0
	for value := range sequence {
		total += value
	}
	return total
}

func AssignMapIterator(target *reflect.MapIter, source *reflect.MapIter) {
	*target = *source
}

func RewindMapIterator(value reflect.Value) bool {
	iterator := value.MapRange()
	snapshot := *iterator
	iterator.Next()
	*iterator = snapshot
	return iterator.Next()
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
	options := providerNumberOptions()
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
		"scalars.int64",
		"private static $projectKey($key: time.Duration): scalars.int64",
		"new Map<scalars.int64",
		"let rangeIndex = 0n",
		"goIntegerMax",
		"TimeDurationValueOperations.$wrap(goInt64(-named_time.TimeDurationValueOperations.$project($argument0)))",
		"values.get(named_time.TimeDurationValueOperations.$project(index))",
		"values.slice(named_time.TimeDurationValueOperations.$project(low), named_time.TimeDurationValueOperations.$project(high), null)",
		"RuntimeSlice.make<int>(named_time.TimeDurationValueOperations.$project(length), null, 0)",
		"GoChannel.make<int>(named_time.TimeDurationValueOperations.$project(size)",
		"GoArray.literal<scalars.int64, 2>(2, named_time.TimeDurationValueOperations.$project(named_time.TimeDurationValueOperations.$wrap(0n))",
		"GoMap__from_gotots_runtime.make<gostring, time__from_gostdlib.Duration>(named_time.TimeDurationValueOperations.$wrap(0n)",
		"globalThis.Number(BigInt.asIntN(64, named_time.TimeDurationValueOperations.$project(value)))",
		"named_time.TimeDurationValueOperations.$wrap(BigInt.asIntN(64, goNumberToBigInt(value)))",
		"IterSeqValueOperations.$project",
		"ReflectMapIterOperations.$assign",
		"cancel: (() => void) | undefined",
		"callback: (($0: gostring, $1:",
		"const callee = cancel;",
		"const callee2 = callback;",
		"(callee ?? GoPanic.raiseRuntime(\"call of nil function\"))();",
		"return (callee2 ?? GoPanic.raiseRuntime(\"call of nil function\"))(",
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
		"GoMapHash.bigint",
		"GoMapHash.number(named_time.TimeDurationValueOperations.$project",
		"async ",
		"await ",
		"Promise<",
		"Awaitable<",
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

func TestProviderSequenceUsesExactSynchronousCallable(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providersequence\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providersequence

import "slices"

func Values(yield func(int) bool) {
	yield(1)
}

func Collect() []int {
	return slices.Collect(Values)
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
	options := providerNumberOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Collect"),
		)},
		options,
	)
	if err != nil {
		t.Fatalf("synchronous sequence compile: %v", err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if !strings.Contains(artifacts.printed, "SlicesCollectKernel") {
		t.Fatalf("synchronous sequence kernel is absent:\n%s", artifacts.printed)
	}
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("synchronous sequence output contains %q", forbidden)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
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
	options := providerNumberOptions()
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Facts"},
		"console.log(Facts());\n",
	)
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
	options := providerNumberOptions()
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Facts"},
		"console.log(Facts());\n",
	)
}
