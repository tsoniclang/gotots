package verification

import "testing"

func TestDirectAndSelectWaitersShareFIFOQueues(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "mixed-queues.ts", `import {
    GoChannel,
    GoScheduler,
    goSelect,
} from "./runtime.js";

await GoScheduler.run(async () => {
    const output: string[] = [];

    const receives = GoChannel.make<number>(0, () => 0, value => value);
    let selectedReceiveValue = 0;
    const selectedReceive = goSelect([
        GoChannel.$selectReceive(receives, value => {
            selectedReceiveValue = value;
        }),
    ]);
    const directReceive = GoChannel.receive(receives);
    await GoScheduler.block(GoChannel.send(receives, 1));
    await GoScheduler.block(GoChannel.send(receives, 2));
    const selectedReceiveCase =
        await GoScheduler.block(selectedReceive);
    const [directReceiveValue] =
        await GoScheduler.block(directReceive);
    output.push(
        String(selectedReceiveCase) + ":" +
        String(selectedReceiveValue) + ":" +
        String(directReceiveValue),
    );

    const sends = GoChannel.make<number>(0, () => 0, value => value);
    const directSend = GoChannel.send(sends, 3);
    const selectedSend = goSelect([
        GoChannel.$selectSend(sends, 4),
    ]);
    const [firstSendValue] =
        await GoScheduler.block(GoChannel.receive(sends));
    const [secondSendValue] =
        await GoScheduler.block(GoChannel.receive(sends));
    await GoScheduler.block(directSend);
    const selectedSendCase = await GoScheduler.block(selectedSend);
    output.push(
        String(firstSendValue) + ":" +
        String(secondSendValue) + ":" +
        String(selectedSendCase),
    );

    console.log(output.join(","));
});
`)
	typecheck(t, directory, runtimePath, runner)
	const expected = "0:1:2,3:4:0\n"
	if actual := execute(t, directory, "mixed-queues.js"); actual != expected {
		t.Fatalf("mixed waiter FIFO = %q, want %q", actual, expected)
	}
}

func TestBlockedSameChannelSelectRegistrationIsFair(t *testing.T) {
	directory := t.TempDir()
	runtimePath, _ := materializeRuntime(t, directory)
	runner := writeRunner(t, directory, "blocked-select-fair.ts", `import {
    GoChannel,
    GoScheduler,
    goSelect,
} from "./runtime.js";

await GoScheduler.run(async () => {
    const values = GoChannel.make<number>(0, () => 0, value => value);
    let selectedValue = 0;
    Math.random = () => 0;
    const selected = goSelect([
        GoChannel.$selectReceive(values, value => {
            selectedValue = value + 10;
        }),
        GoChannel.$selectReceive(values, value => {
            selectedValue = value + 20;
        }),
    ]);
    await GoScheduler.block(GoChannel.send(values, 5));
    const selectedCase = await GoScheduler.block(selected);
    console.log(String(selectedCase) + ":" + String(selectedValue));
});
`)
	typecheck(t, directory, runtimePath, runner)
	const expected = "1:25\n"
	if actual := execute(t, directory, "blocked-select-fair.js"); actual != expected {
		t.Fatalf("blocked select fairness = %q, want %q", actual, expected)
	}
}
