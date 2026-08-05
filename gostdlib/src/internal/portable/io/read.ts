import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import { ProviderError } from "../../runtime/error.js";
import { goInterfaceEqual } from "../../runtime/interface.js";
import type { Writer } from "../../../io.js";

export const unexpectedEOF: GoError = new ProviderError("unexpected EOF");
export const shortWrite: GoError = new ProviderError("short write");
export const shortBuffer: GoError = new ProviderError("short buffer");
export const noProgress: GoError = new ProviderError("multiple Read calls return no data or error");

export function readFullSync<Failure extends GoInterfaceValue>(
  read: (destination: RuntimeSlice<uint8>) => [int, Failure | undefined],
  destination: RuntimeSlice<uint8>,
  eof: Failure,
  unexpected: Failure,
): [int, Failure | undefined] {
  let total = 0;
  let failure: Failure | undefined;
  while (total < destination.length && failure === undefined) {
    const target = total === 0
      ? destination
      : destination.slice(total, destination.length, null);
    const result = read(target);
    const count = hostInteger(result[0]);
    failure = result[1];
    total += count;
  }
  if (total >= destination.length) {
    return [integerFromHost(total), undefined];
  }
  return total > 0 && goInterfaceEqual(failure, eof)
    ? [integerFromHost(total), unexpected]
    : [integerFromHost(total), failure];
}

export async function readFullAsync<Failure extends GoInterfaceValue>(
  read: (
    destination: RuntimeSlice<uint8>,
  ) => Awaitable<[int, Failure | undefined]>,
  destination: RuntimeSlice<uint8>,
  eof: Failure,
  unexpected: Failure,
): Promise<[int, Failure | undefined]> {
  let total = 0;
  let failure: Failure | undefined;
  while (total < destination.length && failure === undefined) {
    const target = total === 0
      ? destination
      : destination.slice(total, destination.length, null);
    const result = await read(target);
    const count = hostInteger(result[0]);
    failure = result[1];
    total += count;
  }
  if (total >= destination.length) {
    return [integerFromHost(total), undefined];
  }
  return total > 0 && goInterfaceEqual(failure, eof)
    ? [integerFromHost(total), unexpected]
    : [integerFromHost(total), failure];
}

export function writeAll(
  writer: Writer,
  source: RuntimeSlice<uint8>,
): [int, GoError | undefined] {
  let total = 0;
  while (total < source.length) {
    const remaining = source.slice(total, source.length, null);
    const [goCount, failure] = writer.Write(remaining);
    const count = hostInteger(goCount);
    if (count < 0 || count > remaining.length) {
      return [integerFromHost(total), new ProviderError("invalid write result")];
    }
    total += count;
    if (failure !== undefined) {
      return [integerFromHost(total), failure];
    }
    if (count === 0) {
      return [integerFromHost(total), shortWrite];
    }
  }
  return [integerFromHost(total), undefined];
}
