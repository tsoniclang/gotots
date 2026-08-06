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

func TestChannelAndSchedulerRuntimePrintStrictAndExecute(t *testing.T) {
	directory := t.TempDir()
	runtimePath, printed := materializeRuntime(t, directory)
	for _, forbidden := range []string{
		": any",
		": unknown",
		".call(",
		".apply(",
		".bind(",
		"switch (",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("channel runtime contains %q:\n%s", forbidden, printed)
		}
	}
	for _, required := range []string{
		"export class GoChannel<T>",
		"export class GoScheduler",
		"const settledOperation: Promise<T> = operation.then",
		"GoScheduler.check();",
		"Math.floor(Math.random() * ready.length)",
		"const order: number[] = [];",
		"Math.random() * (remaining + 1)",
		"this.bufferHead >= 64 && this.bufferHead * 2 >= this.buffer.length",
		"GoDenseIndex.get(this.buffer, this.bufferHead)",
		"this.senders.add(offer)",
		"this.receivers.add(receive)",
		"senders.delete(offer)",
		"receivers.delete(receive)",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("channel runtime lacks %q:\n%s", required, printed)
		}
	}
	if len(printed) > 24_000 {
		t.Fatalf("channel runtime = %d bytes, want <= 24000", len(printed))
	}
	if strings.Contains(printed, "nextListener") ||
		strings.Contains(printed, "this.buffer[this.bufferHead]") ||
		strings.Contains(printed, "Map<number, () => void>") ||
		strings.Contains(printed, "private listeners") ||
		strings.Contains(printed, "selectSenders") ||
		strings.Contains(printed, "selectReceivers") ||
		strings.Contains(printed, "sendValues") ||
		strings.Contains(printed, "sendResolvers") ||
		strings.Contains(printed, "sendHead") ||
		strings.Contains(printed, "receiveHead") {
		t.Fatalf("channel listeners retain historical identity state:\n%s", printed)
	}
	sendStart := strings.Index(printed, "private subscribeSelectSend")
	receiveStart := strings.Index(printed, "private subscribeSelectReceive")
	selectOffset := -1
	if receiveStart >= 0 {
		selectOffset = strings.Index(
			printed[receiveStart:],
			"\n    $selectSend(value:",
		)
	}
	selectStart := receiveStart + selectOffset
	if sendStart < 0 ||
		receiveStart <= sendStart ||
		selectOffset < 0 ||
		selectStart <= receiveStart ||
		!strings.Contains(
			printed[sendStart:receiveStart],
			"this.senders.add(offer)",
		) ||
		!strings.Contains(
			printed[sendStart:receiveStart],
			"this.senders.delete(offer)",
		) ||
		!strings.Contains(
			printed[receiveStart:selectStart],
			"this.receivers.add(receive)",
		) ||
		!strings.Contains(
			printed[receiveStart:selectStart],
			"this.receivers.delete(receive)",
		) {
		t.Fatalf("select cancellation is not owned by its live offer:\n%s", printed)
	}
	settlement := strings.Index(
		printed,
		"const settledOperation: Promise<T> = operation.then",
	)
	if settlement < 0 {
		t.Fatal("scheduler settlement attachment is absent")
	}
	deadlockCheck := strings.Index(
		printed[settlement:],
		"GoScheduler.check();",
	)
	if deadlockCheck < 0 {
		t.Fatal("scheduler does not attach settlement before deadlock checking")
	}

	runner := writeRunner(t, directory, "ordinary.ts", `import {
    GoChannel,
    GoScheduler,
    goSelect,
} from "./runtime.js";

const output: string[] = [];
await GoScheduler.run(async () => {
    const buffered = GoChannel.make<number>(1, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(buffered, 7));
    const [bufferedValue, bufferedOK] =
        await GoScheduler.block(GoChannel.receive(buffered));
    output.push(String(bufferedValue) + ":" + String(bufferedOK));

    const unbuffered = GoChannel.make<number>(0, () => 0, value => value);
    GoScheduler.spawn(async () => {
        await GoScheduler.block(GoChannel.send(unbuffered, 1));
        await GoScheduler.block(GoChannel.send(unbuffered, 2));
    });
    const [first] = await GoScheduler.block(GoChannel.receive(unbuffered));
    const [second] = await GoScheduler.block(GoChannel.receive(unbuffered));
    output.push(String(first) + ":" + String(second));

    const closed = GoChannel.make<number>(2, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(closed, 9));
    GoChannel.close(closed);
    const [drained, drainedOK] =
        await GoScheduler.block(GoChannel.receive(closed));
    const [zero, zeroOK] =
        await GoScheduler.block(GoChannel.receive(closed));
    output.push(
        String(drained) + ":" + String(drainedOK) + ":" +
        String(zero) + ":" + String(zeroOK),
    );

    const left = GoChannel.make<number>(1, () => 0, value => value);
    const right = GoChannel.make<number>(1, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(left, 10));
    await GoScheduler.block(GoChannel.send(right, 20));
    Math.random = () => 0.99;
    let selectedValue = 0;
    const selected = await GoScheduler.block(goSelect([
        GoChannel.$selectReceive(left, value => { selectedValue = value; }),
        GoChannel.$selectReceive(right, value => { selectedValue = value; }),
    ]));
    output.push(String(selected) + ":" + String(selectedValue));

    const compacted = GoChannel.make<number>(1, () => 0, value => value);
    for (let index = 0; index < 512; index++) {
        await GoScheduler.block(GoChannel.send(compacted, index));
        await GoScheduler.block(GoChannel.receive(compacted));
    }
    output.push("bounded");
});
console.log(output.join(","));
`)
	typecheck(t, directory, runtimePath, runner)
	if actual := execute(t, directory, "ordinary.js"); actual !=
		"7:true,1:2,9:true:0:false,1:20,bounded\n" {
		t.Fatalf("ordinary channel output = %q", actual)
	}
}

func TestChannelLifecycleAndSelectEdgeProperties(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "runner.ts", `import {
    GoChannel,
    GoPanic,
    GoRuntimePanicValue,
    GoScheduler,
    goSelect,
    goSelectReady,
} from "./runtime.js";

async function panicValue(
    operation: () => Promise<void> | void,
): Promise<string> {
    try {
        await operation();
        return "missing";
    } catch (failure) {
        return failure instanceof GoPanic &&
            failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong";
    }
}

const output: string[] = [];
await GoScheduler.run(async () => {
    output.push(await panicValue(() => GoChannel.close<number>(undefined)));

    const closed = GoChannel.make<number>(0, () => 0, value => value);
    GoChannel.close(closed);
    output.push(await panicValue(() => GoChannel.close(closed)));
    output.push(await panicValue(async () => {
        await GoScheduler.block(GoChannel.send(closed, 1));
    }));
    output.push(await panicValue(async () => {
        await GoScheduler.block(goSelect([
            GoChannel.$selectSend(closed, 1),
        ]));
    }));

    const selectedClose =
        GoChannel.make<number>(0, () => 0, value => value);
    const pendingSelectedSend = goSelect([
        GoChannel.$selectSend(selectedClose, 2),
    ]);
    let selectedCloseThrew = false;
    try {
        GoChannel.close(selectedClose);
    } catch {
        selectedCloseThrew = true;
    }
    output.push(String(selectedCloseThrew));
    output.push(await panicValue(async () => {
        await GoScheduler.block(pendingSelectedSend);
    }));

    const blockedSender = GoChannel.make<number>(0, () => 0, value => value);
    const pendingSend = GoChannel.send(blockedSender, 2);
    const observedSend = panicValue(async () => {
        await pendingSend;
    });
    GoChannel.close(blockedSender);
    output.push(await observedSend);

    const blockedReceiver = GoChannel.make<number>(0, () => 0, value => value);
    const pendingReceive = GoChannel.receive(blockedReceiver);
    GoChannel.close(blockedReceiver);
    const [zero, ok] = await pendingReceive;
    output.push(String(zero) + ":" + String(ok));

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
    await GoScheduler.block(GoChannel.send(copied, original));
    original.value = 9;
    const [copiedValue] =
        await GoScheduler.block(GoChannel.receive(copied));
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
    const selectedCase =
        GoChannel.$selectSend(selectedCopy, { value: 4 });
    await GoScheduler.block(goSelect([selectedCase]));
    const [selectedCopyValue] =
        await GoScheduler.block(GoChannel.receive(selectedCopy));
    output.push(String(copies) + ":" + String(selectedCopyValue.value));

    for (const capacity of [
        -1,
        1.5,
        Number.MAX_SAFE_INTEGER + 1,
        9007199254740992n,
    ]) {
        output.push(await panicValue(() => {
            GoChannel.make<number>(capacity, () => 0, value => value);
        }));
    }

    const validBigInt =
        GoChannel.make<number>(2n, () => 0, value => value);
    output.push(
        String(GoChannel.capacity(validBigInt)) + ":" +
        String(validBigInt === validBigInt),
    );

    const ready = GoChannel.make<number>(1, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(ready, 7));
    let commits = 0;
    let selectedValue = 0;
    const readySelected = goSelectReady([
        GoChannel.$selectReceive<number>(
            undefined,
            () => { commits++; },
        ),
        GoChannel.$selectReceive(ready, value => {
            commits++;
            selectedValue = value;
        }),
    ]);
    output.push(
        String(readySelected) + ":" +
        String(commits) + ":" +
        String(selectedValue),
    );

    const blocking = GoChannel.make<number>(0, () => 0, value => value);
    GoScheduler.spawn(async () => {
        await GoScheduler.block(GoChannel.send(blocking, 8));
    });
    let blockedValue = 0;
    const blockingSelected = await GoScheduler.block(goSelect([
        GoChannel.$selectReceive(blocking, value => {
            blockedValue = value;
        }),
    ]));
    output.push(String(blockingSelected) + ":" + String(blockedValue));

    const rendezvous =
        GoChannel.make<number>(0, () => 0, value => value);
    let rendezvousValue = 0;
    const selectedSend = goSelect([
        GoChannel.$selectSend(rendezvous, 12),
    ]);
    const selectedReceive = goSelect([
        GoChannel.$selectReceive(rendezvous, value => {
            rendezvousValue = value;
        }),
    ]);
    const rendezvousCases = await Promise.all([
        GoScheduler.block(selectedSend),
        GoScheduler.block(selectedReceive),
    ]);
    output.push(
        rendezvousCases.join(":") + ":" + String(rendezvousValue),
    );

    const canceled = GoChannel.make<number>(0, () => 0, value => value);
    const winner = GoChannel.make<number>(1, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(winner, 9));
    const canceledSelected = await GoScheduler.block(goSelect([
        GoChannel.$selectReceive(canceled, () => {}),
        GoChannel.$selectReceive(winner, () => {}),
    ]));
    let historicalCommit = false;
    const historicalSend = GoChannel.send(canceled, 10).then(() => {
        historicalCommit = true;
    });
    await Promise.resolve();
    output.push(String(canceledSelected) + ":" + String(historicalCommit));
    const [canceledValue] =
        await GoScheduler.block(GoChannel.receive(canceled));
    await historicalSend;
    output.push(String(canceledValue));
});
console.log(output.join(","));
`)
	typecheck(t, directory, runtimePath, runner)
	const expected = "close of nil channel,close of closed channel," +
		"send on closed channel,send on closed channel," +
		"false,send on closed channel,send on closed channel," +
		"0:false,1:3,1:4," +
		"makechan: size out of range,makechan: size out of range," +
		"makechan: size out of range,makechan: size out of range," +
		"2:true,1:1:7,0:8,0:0:12,1:false,10\n"
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
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	denseDefinitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModuleDenseIndex,
		[]api.RuntimeSymbol{api.RuntimeDenseIndex},
		api.ConcurrencySemanticsDisabled,
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
			api.RuntimeScheduler,
		},
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	var statements []tsgo.Statement
	for _, definitions := range [][]runtimeemission.Definition{
		interfaceDefinitions,
		panicDefinitions,
		denseDefinitions,
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
