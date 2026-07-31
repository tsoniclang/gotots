import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import { WithCancel } from "../../../../context.js";
import type {
  CancelFunc,
  Context,
} from "../../../../context.js";
import type { Signal } from "../../../../os.js";
import { sliceValues } from "../../../runtime/slice.js";
import { nodeSignal } from "../signal.js";

const catchableSignals: readonly NodeJS.Signals[] = [
  "SIGHUP",
  "SIGINT",
  "SIGQUIT",
  "SIGILL",
  "SIGTRAP",
  "SIGABRT",
  "SIGBUS",
  "SIGFPE",
  "SIGUSR1",
  "SIGSEGV",
  "SIGUSR2",
  "SIGPIPE",
  "SIGALRM",
  "SIGTERM",
  "SIGSTKFLT",
  "SIGCHLD",
  "SIGCONT",
  "SIGTSTP",
  "SIGTTIN",
  "SIGTTOU",
  "SIGURG",
  "SIGXCPU",
  "SIGXFSZ",
  "SIGVTALRM",
  "SIGPROF",
  "SIGWINCH",
  "SIGIO",
  "SIGPWR",
  "SIGSYS",
];

export function notifyContext(
  parent: Context | undefined,
  signals: RuntimeSlice<Signal | undefined>,
): [Context | undefined, NonNullable<CancelFunc>] {
  const [context, cancel] = WithCancel(parent);
  const selected = selectedSignals(signals);
  let stopped = false;

  const stop = async (): Promise<void> => {
    if (stopped) {
      return;
    }
    stopped = true;
    for (const signal of selected) {
      process.removeListener(signal, onSignal);
    }
    await cancel();
  };
  const onSignal = (): void => {
    void stop();
  };

  for (const signal of selected) {
    process.once(signal, onSignal);
  }
  const done = context?.Done();
  if (done !== undefined) {
    void done.receive().then(stop);
  }
  return [context, stop];
}

function selectedSignals(
  signals: RuntimeSlice<Signal | undefined>,
): NodeJS.Signals[] {
  const selected: NodeJS.Signals[] = [];
  const values = sliceValues(signals);
  if (values.length === 0) {
    return [...catchableSignals];
  }
  for (const signal of values) {
    if (signal === undefined) {
      continue;
    }
    const name = nodeSignal(signal.String());
    if (name !== undefined && !selected.includes(name)) {
      selected.push(name);
    }
  }
  return selected;
}
