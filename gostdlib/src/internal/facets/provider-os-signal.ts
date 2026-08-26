import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import type { CancelFunc, Context as ProviderContext } from "../../context.js";
import type { Process, Signal as ProviderSignal } from "../../os.js";
import { signalProcess } from "../node/os/process.js";
import { notifyContext } from "../node/os/signal/notify.js";
import type { CanonicalError } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { CanonicalError } from "./provider-io-contract.js";
export type { Context as ProviderContext } from "../../context.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export interface CanonicalSignal extends GoInterfaceValue {
  Signal(recovery?: GoRecovery): void;
  String(recovery?: GoRecovery): gostring;
}

export function OsSignalNotifyContextDirect(
  parent: ProviderContext | undefined,
  signals: RuntimeSlice<ProviderSignal | undefined>,
): [ProviderContext | undefined, NonNullable<CancelFunc>] {
  return notifyContext(parent, signals);
}

export function OsProcessSignalCanonical(
  receiver: Process | undefined,
  signal: CanonicalSignal | undefined,
): CanonicalError | undefined {
  return signalProcess(receiver, signal);
}
