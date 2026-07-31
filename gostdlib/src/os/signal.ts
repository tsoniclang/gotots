import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  CancelFunc,
  Context,
} from "../context.js";
import type { Signal } from "../os.js";
import { notifyContext } from "../internal/node/os/signal/notify.js";

export function NotifyContext(
  parent: Context | undefined,
  signals: RuntimeSlice<Signal | undefined>,
): [Context | undefined, NonNullable<CancelFunc>] {
  return notifyContext(parent, signals);
}
