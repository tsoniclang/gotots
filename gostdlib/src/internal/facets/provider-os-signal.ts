import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, gostring } from "@gotots/runtime/scalars.js";

import type { CancelFunc } from "../../context.js";
import type { Process } from "../../os.js";
import { signalProcessAsync } from "../node/os/process.js";
import {
  selectedSignalsAsync,
  startNotificationAsync,
} from "../node/os/signal/notify.js";
import {
  ContextWithCancelCanonical,
  type CanonicalContext,
} from "./provider-context.js";
import type { CanonicalError } from "./provider-io-contract.js";
import type { InterfaceContract } from "./provider-support.js";

export type { CanonicalContext } from "./provider-context.js";
export type { CanonicalError } from "./provider-io-contract.js";

export interface CanonicalSignal extends GoInterfaceValue {
  Signal(recovery?: GoRecovery): Awaitable<void>;
  String(recovery?: GoRecovery): Awaitable<gostring>;
}

export async function OsSignalNotifyContextCanonical<
  Failure extends GoInterfaceValue,
  Parent extends CanonicalContext<Failure>,
>(
  parent: Parent | undefined,
  signals: RuntimeSlice<CanonicalSignal | undefined>,
  canceled: Failure | undefined,
  contextContract: InterfaceContract,
): Promise<[
  CanonicalContext<Failure>,
  NonNullable<CancelFunc>,
]> {
  const selected = await selectedSignalsAsync(signals);
  const [context, cancel] = await ContextWithCancelCanonical<Failure, Parent>(
    parent,
    canceled,
    contextContract,
  );
  return startNotificationAsync(context, cancel, selected);
}

export async function OsProcessSignalCanonical(
  receiver: Process | undefined,
  signal: CanonicalSignal | undefined,
): Promise<CanonicalError | undefined> {
  return signalProcessAsync(receiver, signal);
}
