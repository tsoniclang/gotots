import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, gostring } from "@gotots/runtime/scalars.js";

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
  startNotificationAsync,
} from "../node/os/signal/notify.js";
import type { Process, Signal } from "../../os.js";
import {
  ContextWithCancelCanonicalSync,
} from "./provider-context.js";
import type { CanonicalContextSync } from "./provider-context.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";

export type { CanonicalContextSync } from "./provider-context.js";
export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
} from "./provider-io-contract.js";

export interface CanonicalSignal extends GoInterfaceValue {
  Signal(recovery?: GoRecovery): Awaitable<void>;
  String(recovery?: GoRecovery): Awaitable<gostring>;
}

export async function OsSignalNotifyContextCanonical(
  parent: Context | undefined,
  signals: RuntimeSlice<CanonicalSignal | undefined>,
): Promise<[Context | undefined, NonNullable<CancelFunc>]> {
  return notifyContextAsync(parent, signals);
}

export async function OsSignalNotifyContextCanonicalContext<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContextSync<Failure>,
>(
  parent: Parent | undefined,
  signals: RuntimeSlice<Signal | undefined>,
  canceled: Failure | undefined,
  contextContract: readonly object[],
): Promise<[
  CanonicalContextSync<Failure>,
  NonNullable<CancelFunc>,
]> {
  const selected = selectedSignals(signals);
  const [context, cancel] = await ContextWithCancelCanonicalSync<Failure, Parent>(
    parent,
    canceled,
    contextContract,
  );
  return startNotificationAsync(context, cancel, selected);
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
  NonNullable<CancelFunc>,
]> {
  const selected = await selectedSignalsAsync(signals);
  const [context, cancel] = await ContextWithCancelCanonicalSync<Failure, Parent>(
    parent,
    canceled,
    contextContract,
  );
  return startNotificationAsync(context, cancel, selected);
}

export async function OsProcessSignalCanonical(
  receiver: Process | undefined,
  signal: CanonicalSignal | undefined,
): Promise<CanonicalErrorAsync | undefined> {
  return signalProcessAsync(receiver, signal);
}
