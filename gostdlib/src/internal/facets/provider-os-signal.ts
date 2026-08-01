import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import type {
  CancelFunc,
  Context,
} from "../../context.js";
import { signalProcessAsync } from "../node/os/process.js";
import { notifyContextAsync } from "../node/os/signal/notify.js";
import type { Process } from "../../os.js";

export interface CanonicalSignal extends GoInterfaceValue {
  Signal(recovery?: GoRecovery): void;
  String(recovery?: GoRecovery): Promise<gostring>;
}

export async function OsSignalNotifyContextCanonical(
  parent: Context | undefined,
  signals: RuntimeSlice<CanonicalSignal | undefined>,
): Promise<[Context | undefined, NonNullable<CancelFunc>]> {
  return notifyContextAsync(parent, signals);
}

export async function OsProcessSignalCanonical(
  receiver: Process | undefined,
  signal: CanonicalSignal | undefined,
): Promise<GoError | undefined> {
  return signalProcessAsync(receiver, signal);
}
