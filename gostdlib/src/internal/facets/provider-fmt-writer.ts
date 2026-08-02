import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64 } from "@gotots/runtime/scalars.js";

import { formatOperands, formatText } from "../portable/fmt/format.js";
import { byteSlice } from "../runtime/slice.js";
import type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export function FprintCanonicalSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, Failure | undefined] {
  return writeSync(writer, formatOperands(arguments_, false));
}

export function FprintfCanonicalSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  writer: Target | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, Failure | undefined] {
  return writeSync(writer, formatText(format, arguments_).text);
}

export function FprintlnCanonicalSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, Failure | undefined] {
  return writeSync(writer, formatOperands(arguments_, true));
}

export async function FprintCanonicalAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return writeAsync(writer, formatOperands(arguments_, false));
}

export async function FprintCanonicalAsyncOrdinaryError<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return FprintCanonicalAsync(writer, arguments_);
}

export async function FprintfCanonicalAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return writeAsync(writer, formatText(format, arguments_).text);
}

export async function FprintfCanonicalAsyncOrdinaryError<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return FprintfCanonicalAsync(writer, format, arguments_);
}

export async function FprintlnCanonicalAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return writeAsync(writer, formatOperands(arguments_, true));
}

export async function FprintlnCanonicalAsyncOrdinaryError<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return FprintlnCanonicalAsync(writer, arguments_);
}

function writeSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  writer: Target | undefined,
  text: string,
): [int64, Failure | undefined] {
  return requireWriter(writer).Write(
    byteSlice(new TextEncoder().encode(text)),
  );
}

async function writeAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  writer: Target | undefined,
  text: string,
): Promise<[int64, Failure | undefined]> {
  return requireWriter(writer).Write(
    byteSlice(new TextEncoder().encode(text)),
  );
}

function requireWriter<Target>(writer: Target | undefined): Target {
  if (writer === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return writer;
}
