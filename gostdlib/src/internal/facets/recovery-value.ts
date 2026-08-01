import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import {
  Compare,
  type OrderedEquality,
  type OrderedLess,
} from "../portable/cmp/ordered.js";
import { Errno, Signal } from "../../syscall.js";
import { Duration, Ticker, Time } from "../../time.js";

export function CmpCompare<T>(
  less: OrderedLess<T>,
  equal: OrderedEquality<T>,
  left: T,
  right: T,
  _recovery?: GoRecovery,
): number {
  return Compare(less, equal, left, right);
}

export function SyscallErrnoError(
  receiver: Errno,
  _recovery?: GoRecovery,
): gostring {
  return receiver.Error();
}

export function SyscallSignalSignal(
  receiver: Signal,
  _recovery?: GoRecovery,
): void {
  receiver.Signal();
}

export function SyscallSignalString(
  receiver: Signal,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}

export function TimeTickerStop(
  receiver: Ticker | undefined,
  _recovery?: GoRecovery,
): void {
  Ticker.Stop(receiver);
}

export function TimeDurationString(
  receiver: Duration,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}

export function TimeTimeString(
  receiver: Time,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}
