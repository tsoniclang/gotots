import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int } from "@gotots/gostdlib/internal/scalars.js";

import { formatOperands, formatText } from "../portable/fmt/format.js";
import { byteSlice } from "../runtime/slice.js";
import type { ProviderWriterInterface } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { ProviderWriterInterface } from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export function FprintDirect<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, Failure | undefined] {
  return write(writer, formatOperands(arguments_, false));
}

export function FprintfDirect<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  writer: Target | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, Failure | undefined] {
  return write(writer, formatText(format, arguments_).text);
}

export function FprintlnDirect<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  writer: Target | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, Failure | undefined] {
  return write(writer, formatOperands(arguments_, true));
}

function write<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  writer: Target | undefined,
  text: string,
): [int, Failure | undefined] {
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
