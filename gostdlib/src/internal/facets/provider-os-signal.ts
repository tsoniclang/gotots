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
import {
  notifyContextAsync,
  selectedSignals,
  selectedSignalsAsync,
  startNotification,
} from "../node/os/signal/notify.js";
import type { Process, Signal } from "../../os.js";
import {
  ContextWithCancelCanonicalSync,
} from "./provider-context.js";
import type { CanonicalContextSync } from "./provider-context.js";

export type { CanonicalContextSync } from "./provider-context.js";
export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
} from "./provider-io-contract.js";

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

export function OsSignalNotifyContextCanonicalContext<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  signals: RuntimeSlice<Signal | undefined>,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): [
  CanonicalContextSync<Failure>,
  (_recovery?: GoRecovery) => Promise<void>,
] {
  const selected = selectedSignals(signals);
  const [context, cancel] = ContextWithCancelCanonicalSync<Failure, Parent>(
    parent,
    canceled,
    contextContract,
  );
  return startNotification(context, cancel, selected);
}

export async function OsSignalNotifyContextCanonicalContextSignal<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  signals: RuntimeSlice<CanonicalSignal | undefined>,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): Promise<[
  CanonicalContextSync<Failure>,
  (_recovery?: GoRecovery) => Promise<void>,
]> {
  const selected = await selectedSignalsAsync(signals);
  const [context, cancel] = ContextWithCancelCanonicalSync<Failure, Parent>(
    parent,
    canceled,
    contextContract,
  );
  return startNotification(context, cancel, selected);
}

export async function OsProcessSignalCanonical(
  receiver: Process | undefined,
  signal: CanonicalSignal | undefined,
): Promise<GoError | undefined> {
  return signalProcessAsync(receiver, signal);
}
