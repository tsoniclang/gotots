package verification

import "testing"

func TestSchedulerCompletionOwnsDeadlockAndFirstPanic(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		runner string
		want   string
	}{
		{
			name: "nil deadlock",
			runner: `import { GoPanic, GoRuntimePanicValue, GoChannel, GoScheduler } from "./runtime.js";
GoScheduler.run(async () => {
    await GoScheduler.block(GoChannel.receive<number>(undefined));
}).then(
    () => console.log("missing"),
    failure => console.log(
        failure instanceof GoPanic &&
        failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong",
    ),
);
`,
			want: "all goroutines are asleep - deadlock!\n",
		},
		{
			name: "nil send deadlock",
			runner: `import { GoPanic, GoRuntimePanicValue, GoChannel, GoScheduler } from "./runtime.js";
GoScheduler.run(async () => {
    await GoScheduler.block(GoChannel.send<number>(undefined, 1));
}).then(
    () => console.log("missing"),
    failure => console.log(
        failure instanceof GoPanic &&
        failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong",
    ),
);
`,
			want: "all goroutines are asleep - deadlock!\n",
		},
		{
			name: "uncaught goroutine panic",
			runner: `import {
    GoPanic,
    GoRuntimePanicValue,
    GoChannel,
    GoScheduler,
} from "./runtime.js";
GoScheduler.run(async () => {
    GoScheduler.spawn(async () => {
        throw GoPanic.createRuntime("goroutine panic");
    });
    await GoScheduler.block(GoChannel.receive<number>(undefined));
}).then(
    () => console.log("missing"),
    failure => console.log(
        failure instanceof GoPanic &&
        failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong",
    ),
);
`,
			want: "goroutine panic\n",
		},
		{
			name: "first uncaught goroutine panic",
			runner: `import {
    GoPanic,
    GoRuntimePanicValue,
    GoChannel,
    GoScheduler,
} from "./runtime.js";
GoScheduler.run(async () => {
    GoScheduler.spawn(async () => {
        throw GoPanic.createRuntime("first panic");
    });
    GoScheduler.spawn(async () => {
        throw GoPanic.createRuntime("second panic");
    });
    await GoScheduler.block(GoChannel.receive<number>(undefined));
}).then(
    () => console.log("missing"),
    failure => console.log(
        failure instanceof GoPanic &&
        failure.value instanceof GoRuntimePanicValue
            ? failure.value.message
            : "wrong",
    ),
);
`,
			want: "first panic\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			runtimePath, _ := materializeRuntime(t, directory)
			runner := writeRunner(t, directory, "runner.ts", testCase.runner)
			typecheck(t, directory, runtimePath, runner)
			if actual := execute(t, directory, "runner.js"); actual != testCase.want {
				t.Fatalf("scheduler output = %q, want %q", actual, testCase.want)
			}
		})
	}
}

func TestBufferedMainSettlementPrecedesDeadlockCheck(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "buffered-main.ts", `import {
    GoChannel,
    GoScheduler,
} from "./runtime.js";

await GoScheduler.run(async () => {
    const values = GoChannel.make<number>(1, () => 0, value => value);
    await GoScheduler.block(GoChannel.send(values, 7));
    const [value] = await GoScheduler.block(GoChannel.receive(values));
    console.log(value);
});
`)
	typecheck(t, directory, runtimePath, runner)
	if actual := execute(t, directory, "buffered-main.js"); actual != "7\n" {
		t.Fatalf("buffered main output = %q", actual)
	}
}

func TestSchedulerMainReturnAbandonsBlockedGoroutine(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "runner.ts", `import {
    GoChannel,
    GoScheduler,
} from "./runtime.js";

await GoScheduler.run(async () => {
    GoScheduler.spawn(async () => {
        await GoScheduler.block(GoChannel.receive<number>(undefined));
    });
});
console.log("returned");
`)
	typecheck(t, directory, runtimePath, runner)
	if actual := execute(t, directory, "runner.js"); actual != "returned\n" {
		t.Fatalf("main return output = %q", actual)
	}
}
