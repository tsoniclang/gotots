import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";
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
  const selected = selectedSignals(signals);
  const [context, cancel] = WithCancel(parent);
  return startNotification(context, cancel, selected);
}

interface NotificationContext {
  Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
}

export function startNotification<Value extends NotificationContext>(
  context: Value,
  cancel: NonNullable<CancelFunc>,
  selected: readonly NodeJS.Signals[],
): [Value, NonNullable<CancelFunc>] {
  return startNotificationWithDone(context, cancel, selected, context.Done());
}

function startNotificationWithDone<Value>(
  context: Value,
  cancel: NonNullable<CancelFunc>,
  selected: readonly NodeJS.Signals[],
  done: GoReceiveChannel<GoEmptyStruct> | undefined,
): [Value, NonNullable<CancelFunc>] {
  let stopped = false;
  let unsubscribeDone = (): void => undefined;

  const stop = (): void => {
    if (stopped) {
      return;
    }
    stopped = true;
    unsubscribeDone();
    for (const signal of selected) {
      process.removeListener(signal, onSignal);
    }
    cancel();
  };
  const onSignal = (): void => stop();

  for (const signal of selected) {
    process.once(signal, onSignal);
  }
  if (done !== undefined) {
    const receive = done.$selectReceive(stop);
    if (receive.ready()) {
      receive.commit();
    } else {
      unsubscribeDone = done.$observeClose(stop);
    }
  }
  return [context, stop];
}

export function selectedSignals<Value extends { String(): string }>(
  signals: RuntimeSlice<Value | undefined>,
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
