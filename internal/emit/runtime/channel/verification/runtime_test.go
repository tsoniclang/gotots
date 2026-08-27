package verification

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestChannelRuntimePrintsStrictSynchronousSurface(t *testing.T) {
	directory := t.TempDir()
	runtimePath, printed := materializeRuntime(t, directory)
	for _, forbidden := range []string{
		": any",
		": unknown",
		".call(",
		".apply(",
		".bind(",
		"async ",
		"await ",
		"Promise",
		"GoScheduler",
		"subscribe",
		"receivers",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("channel runtime contains %q:\n%s", forbidden, printed)
		}
	}
	for _, required := range []string{
		"export class GoChannel<T>",
		"Math.floor(Math.random() * ready.length)",
		"this.bufferHead >= 64 && this.bufferHead * 2 >= this.buffer.length",
		"this.bufferHead in this.buffer",
		"serial channel send would block",
		"serial channel receive would block",
		"serial select would block",
		"$observeClose(observer: () => void): () => void",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("channel runtime lacks %q:\n%s", required, printed)
		}
	}
	if len(printed) > 18_000 {
		t.Fatalf("channel runtime = %d bytes, want <= 18000", len(printed))
	}
	if strings.Count(printed, "Math.floor(Math.random() * ready.length)") != 1 {
		t.Fatalf("ready selection evaluates its random index more than once:\n%s", printed)
	}

	runner := writeRunner(t, directory, "ordinary.ts", `import {
    GoChannel,
    goSelect,
} from "./runtime.js";

const output: string[] = [];
const buffered = GoChannel.make<number>(1, () => 0, value => value);
GoChannel.send(buffered, 7);
const [bufferedValue, bufferedOK] = GoChannel.receive(buffered);
output.push(String(bufferedValue) + ":" + String(bufferedOK));

const closed = GoChannel.make<number>(2, () => 0, value => value);
let observedCloses = 0;
closed.$observeClose(() => { observedCloses++; });
GoChannel.send(closed, 9);
GoChannel.close(closed);
closed.$observeClose(() => { observedCloses++; });
const [drained, drainedOK] = GoChannel.receive(closed);
const [zero, zeroOK] = GoChannel.receive(closed);
output.push(
    String(drained) + ":" + String(drainedOK) + ":" +
    String(zero) + ":" + String(zeroOK) + ":" + String(observedCloses),
);

const unobserved = GoChannel.make<number>(1, () => 0, value => value);
let removedObserverRan = false;
const unobserve = unobserved.$observeClose(() => {
    removedObserverRan = true;
});
unobserve();
GoChannel.close(unobserved);
output.push(String(removedObserverRan));

const left = GoChannel.make<number>(1, () => 0, value => value);
const right = GoChannel.make<number>(1, () => 0, value => value);
GoChannel.send(left, 10);
GoChannel.send(right, 20);
Math.random = () => 0.99;
let selectedValue = 0;
const selected = goSelect([
    GoChannel.$selectReceive(left, value => { selectedValue = value; }),
    GoChannel.$selectReceive(right, value => { selectedValue = value; }),
]);
output.push(String(selected) + ":" + String(selectedValue));

const compacted = GoChannel.make<number>(1, () => 0, value => value);
for (let index = 0; index < 512; index++) {
    GoChannel.send(compacted, index);
    GoChannel.receive(compacted);
}
output.push("bounded");
console.log(output.join(","));
`)
	typecheck(t, directory, runtimePath, runner)
	if actual := execute(t, directory, "ordinary.js"); actual !=
		"7:true,9:true:0:false:2,false,1:20,bounded\n" {
		t.Fatalf("ordinary channel output = %q", actual)
	}
}

func TestChannelLifecycleAndBlockingBoundaries(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "runner.ts", `import {
    GoChannel,
    GoPanic,
    GoRuntimePanicValue,
    goSelect,
    goSelectReady,
} from "./runtime.js";

function panicValue(operation: () => void): string {
    try {
        operation();
        return "missing";
    } catch (failure) {
        return failure instanceof GoPanic &&
            failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong";
    }
}

const output: string[] = [];
output.push(panicValue(() => GoChannel.close<number>(undefined)));

const closed = GoChannel.make<number>(0, () => 0, value => value);
GoChannel.close(closed);
output.push(panicValue(() => GoChannel.close(closed)));
output.push(panicValue(() => GoChannel.send(closed, 1)));
output.push(panicValue(() => {
    goSelect([GoChannel.$selectSend(closed, 1)]);
}));

const blocking = GoChannel.make<number>(0, () => 0, value => value);
output.push(panicValue(() => GoChannel.send(blocking, 1)));
output.push(panicValue(() => { GoChannel.receive(blocking); }));

let copies = 0;
const copied = GoChannel.make<{ value: number }>(
    1,
    () => ({ value: 0 }),
    value => {
        copies++;
        return { value: value.value };
    },
);
const original = { value: 3 };
GoChannel.send(copied, original);
original.value = 9;
const [copiedValue] = GoChannel.receive(copied);
output.push(String(copies) + ":" + String(copiedValue.value));

copies = 0;
const selectedCopy = GoChannel.make<{ value: number }>(
    1,
    () => ({ value: 0 }),
    value => {
        copies++;
        return { value: value.value };
    },
);
goSelect([GoChannel.$selectSend(selectedCopy, { value: 4 })]);
const [selectedCopyValue] = GoChannel.receive(selectedCopy);
output.push(String(copies) + ":" + String(selectedCopyValue.value));

for (const capacity of [
    -1,
    1.5,
    Number.MAX_SAFE_INTEGER + 1,
    9007199254740992n,
]) {
    output.push(panicValue(() => {
        GoChannel.make<number>(capacity, () => 0, value => value);
    }));
}

const validBigInt = GoChannel.make<number>(2n, () => 0, value => value);
output.push(
    String(GoChannel.capacity(validBigInt)) + ":" +
    String(validBigInt === validBigInt),
);

const ready = GoChannel.make<number>(1, () => 0, value => value);
GoChannel.send(ready, 7);
let commits = 0;
let selectedValue = 0;
const readySelected = goSelectReady([
    GoChannel.$selectReceive<number>(undefined, () => { commits++; }),
    GoChannel.$selectReceive(ready, value => {
        commits++;
        selectedValue = value;
    }),
]);
output.push(
    String(readySelected) + ":" + String(commits) + ":" +
    String(selectedValue),
);
output.push(panicValue(() => {
    goSelect([
        GoChannel.$selectReceive<number>(undefined, () => {}),
    ]);
}));
console.log(output.join(","));
`)
	typecheck(t, directory, runtimePath, runner)
	const expected = "close of nil channel,close of closed channel," +
		"send on closed channel,send on closed channel," +
		"serial channel send would block,serial channel receive would block," +
		"1:3,1:4," +
		"makechan: size out of range,makechan: size out of range," +
		"makechan: size out of range,makechan: size out of range," +
		"2:true,1:1:7,serial select would block\n"
	if actual := execute(t, directory, "runner.js"); actual != expected {
		t.Fatalf("channel edge output = %q, want %q", actual, expected)
	}
}

func materializeRuntime(t *testing.T, directory string) (string, string) {
	t.Helper()
	factory := tsgo.NewFactory()
	interfaceDefinitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModuleInterfaceValue,
		[]api.RuntimeSymbol{
			api.RuntimeInterfaceValue,
			api.RuntimeErrorMethodToken,
			api.RuntimeRuntimeErrorToken,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	panicDefinitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModulePanic,
		[]api.RuntimeSymbol{
			api.RuntimePanicValue,
			api.RuntimePanic,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	channelDefinitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModuleChannel,
		[]api.RuntimeSymbol{
			api.RuntimeReceiveChannel,
			api.RuntimeSendChannel,
			api.RuntimeSelectCase,
			api.RuntimeChannel,
			api.RuntimeSelectAttempt,
			api.RuntimeSelectReady,
			api.RuntimeSelect,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var statements []tsgo.Statement
	for _, definitions := range [][]runtimeemission.Definition{
		interfaceDefinitions,
		panicDefinitions,
		channelDefinitions,
	} {
		for _, definition := range definitions {
			statements = append(statements, definition.Statement())
		}
	}
	sourcePath, err := tsgo.NewPath("runtime.ts")
	if err != nil {
		t.Fatal(err)
	}
	source := factory.SourceFile(
		statements,
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        sourcePath,
			Path:            sourcePath,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
	client, err := tsgo.StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	printed, err := client.PrintNode(source, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(directory, "runtime.ts")
	writeFile(t, runtimePath, printed)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	return runtimePath, printed
}

func writeRunner(
	t *testing.T,
	directory string,
	name string,
	content string,
) string {
	t.Helper()
	path := filepath.Join(directory, name)
	writeFile(t, path, content)
	return path
}

func typecheck(t *testing.T, directory string, paths ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(directory, "out"),
	}
	arguments = append(arguments, paths...)
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		directory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func execute(t *testing.T, directory, runner string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"node",
		filepath.Join(directory, "out", runner),
	)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		panic("resolve channel verification repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."),
	)
}
