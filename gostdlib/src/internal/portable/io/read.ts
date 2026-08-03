import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int64, uint8 } from "@gotots/runtime/scalars.js";

import { ProviderError } from "../../runtime/error.js";
import { goInterfaceEqual } from "../../runtime/interface.js";
import type { Writer } from "../../../io.js";

export const unexpectedEOF: GoError = new ProviderError("unexpected EOF");
export const shortWrite: GoError = new ProviderError("short write");
export const shortBuffer: GoError = new ProviderError("short buffer");
export const noProgress: GoError = new ProviderError("multiple Read calls return no data or error");

export function readFullSync<Failure extends GoInterfaceValue>(
  read: (destination: RuntimeSlice<uint8>) => [int64, Failure | undefined],
  destination: RuntimeSlice<uint8>,
  eof: Failure,
  unexpected: Failure,
): [int64, Failure | undefined] {
  let total = 0;
  let failure: Failure | undefined;
  while (total < destination.length && failure === undefined) {
    const target = total === 0
      ? destination
      : destination.slice(total, destination.length, null);
    const result = read(target);
    const count = result[0];
    failure = result[1];
    total += count;
  }
  if (total >= destination.length) {
    return [total, undefined];
  }
  return total > 0 && goInterfaceEqual(failure, eof)
    ? [total, unexpected]
    : [total, failure];
}

export async function readFullAsync<Failure extends GoInterfaceValue>(
  read: (
    destination: RuntimeSlice<uint8>,
  ) => Awaitable<[int64, Failure | undefined]>,
  destination: RuntimeSlice<uint8>,
  eof: Failure,
  unexpected: Failure,
): Promise<[int64, Failure | undefined]> {
  let total = 0;
  let failure: Failure | undefined;
  while (total < destination.length && failure === undefined) {
    const target = total === 0
      ? destination
      : destination.slice(total, destination.length, null);
    const result = await read(target);
    const count = result[0];
    failure = result[1];
    total += count;
  }
  if (total >= destination.length) {
    return [total, undefined];
  }
  return total > 0 && goInterfaceEqual(failure, eof)
    ? [total, unexpected]
    : [total, failure];
}

export function writeAll(
  writer: Writer,
  source: RuntimeSlice<uint8>,
): [int64, GoError | undefined] {
  let total = 0;
  while (total < source.length) {
    const remaining = source.slice(total, source.length, null);
    const [count, failure] = writer.Write(remaining);
    if (count < 0 || count > remaining.length) {
      return [total, new ProviderError("invalid write result")];
    }
    total += count;
    if (failure !== undefined) {
      return [total, failure];
    }
    if (count === 0) {
      return [total, shortWrite];
    }
  }
  return [total, undefined];
}
