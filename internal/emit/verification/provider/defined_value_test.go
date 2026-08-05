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
		"context__from_gostdlib.CancelFunc",
		"fs__from_gostdlib.WalkDirFunc",
		"const __gotots_callee_0 = cancel;",
		"const __gotots_callee_1 = callback;",
		"await __gotots_callee_0();",
		".$from(await __gotots_callee_1(",
		".$to(__gotots_argument_1)",
		".$to(__gotots_argument_2)",
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
