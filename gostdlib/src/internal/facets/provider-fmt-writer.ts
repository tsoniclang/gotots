import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64 } from "@gotots/runtime/scalars.js";

import { formatOperands, formatText } from "../portable/fmt/format.js";
import { byteSlice } from "../runtime/slice.js";
import type { CanonicalWriter } from "./provider-io-contract.js";

export type {
  CanonicalError,
  CanonicalWriter,
} from "./provider-io-contract.js";

export async function FprintCanonical<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return write(writer, formatOperands(arguments_, false));
}

export async function FprintfCanonical<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  writer: Target | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return write(writer, formatText(format, arguments_).text);
}

export async function FprintlnCanonical<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): Promise<[int64, Failure | undefined]> {
  return write(writer, formatOperands(arguments_, true));
}

async function write<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
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
