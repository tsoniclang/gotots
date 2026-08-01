import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import { ProviderError } from "../../runtime/error.js";
import type { Reader, Writer } from "../../../io.js";

export const unexpectedEOF: GoError = new ProviderError("unexpected EOF");
export const shortWrite: GoError = new ProviderError("short write");
export const shortBuffer: GoError = new ProviderError("short buffer");
export const noProgress: GoError = new ProviderError("multiple Read calls return no data or error");

export function readFull(
  reader: Reader,
  destination: RuntimeSlice<uint8>,
  eof: GoError,
): [int64, GoError | undefined] {
  let total = 0;
  let emptyReads = 0;

  while (total < destination.length) {
    const target = total === 0
      ? destination
      : destination.slice(total, destination.length, null);
    const [count, failure] = reader.Read(target);
    if (count < 0 || count > destination.length - total) {
      return [total, new ProviderError("invalid read result")];
    }
    total += count;
    if (total === destination.length) {
      return [total, undefined];
    }
    if (failure !== undefined) {
      if (failure === eof && total > 0) {
        return [total, unexpectedEOF];
      }
      return [total, failure];
    }
    if (count === 0) {
      emptyReads += 1;
      if (emptyReads >= 100) {
        return [total, noProgress];
      }
    } else {
      emptyReads = 0;
    }
  }

  return [total, undefined];
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
