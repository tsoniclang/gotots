import type { GoError } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, uint8 } from "@gotots/runtime/scalars.js";

import {
  Compare,
  type OrderedEquality,
  type OrderedLess,
} from "../portable/cmp/ordered.js";
import { Errno, Signal } from "../../syscall.js";
import { Duration, ParseError, Ticker, Time } from "../../time.js";

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

export function TimeParseErrorError(
  receiver: ParseError | undefined,
  _recovery?: GoRecovery,
): gostring {
  return ParseError.Error(receiver);
}

export function TimeTimeAppendText(
  receiver: Time,
  target: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): [RuntimeSlice<uint8>, GoError | undefined] {
  return receiver.AppendText(target);
}

export function TimeTimeIsZero(
  receiver: Time,
  _recovery?: GoRecovery,
): boolean {
  return receiver.IsZero();
}

export function TimeTimeMarshalJSON(
  receiver: Time,
  _recovery?: GoRecovery,
): [RuntimeSlice<uint8>, GoError | undefined] {
  return receiver.MarshalJSON();
}

export function TimeTimeMarshalText(
  receiver: Time,
  _recovery?: GoRecovery,
): [RuntimeSlice<uint8>, GoError | undefined] {
  return receiver.MarshalText();
}

export function TimeTimeString(
  receiver: Time,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}
