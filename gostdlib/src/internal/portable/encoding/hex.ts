import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import { QuoteRune } from "../strconv/quote.js";
import { IsPrint } from "../unicode/properties.js";
import { ProviderError } from "../../runtime/error.js";

const digits = "0123456789abcdef";
const errLength = new ProviderError("encoding/hex: odd length hex string");

export function EncodedLen(length: int64): int64 {
  return length * 2;
}

export function AppendEncode(
  destination: RuntimeSlice<uint8>,
  source: RuntimeSlice<uint8>,
): RuntimeSlice<uint8> {
  const encoded: uint8[] = [];
  for (let index = 0; index < source.length; index += 1) {
    const value = source.get(index);
    encoded.push(
      digits.charCodeAt(Math.floor(value / 16)),
      digits.charCodeAt(value % 16),
    );
  }
  return destination.append(0, encoded);
}

export function AppendDecode(
  destination: RuntimeSlice<uint8>,
  source: RuntimeSlice<uint8>,
): [RuntimeSlice<uint8>, GoError | undefined] {
  const decoded: uint8[] = [];
  let index = 0;
  while (index + 1 < source.length) {
    const firstByte = source.get(index);
    const secondByte = source.get(index + 1);
    const first = decodeDigit(firstByte);
    if (first === undefined) {
      return [destination.append(0, decoded), invalidByte(firstByte)];
    }
    const second = decodeDigit(secondByte);
    if (second === undefined) {
      return [destination.append(0, decoded), invalidByte(secondByte)];
    }
    decoded.push((first << 4) | second);
    index += 2;
  }
  if (index < source.length) {
    const finalByte = source.get(index);
    if (decodeDigit(finalByte) === undefined) {
      return [destination.append(0, decoded), invalidByte(finalByte)];
    }
    return [destination.append(0, decoded), errLength];
  }
  return [destination.append(0, decoded), undefined];
}

export function EncodeToString(source: RuntimeSlice<uint8>): gostring {
  let result = "";
  for (let index = 0; index < source.length; index += 1) {
    const byte = source.get(index);
    result += digits.charAt(Math.floor(byte / 16)) + digits.charAt(byte % 16);
  }
  return result;
}

function decodeDigit(value: uint8): number | undefined {
  if (value >= 0x30 && value <= 0x39) {
    return value - 0x30;
  }
  if (value >= 0x61 && value <= 0x66) {
    return value - 0x61 + 10;
  }
  if (value >= 0x41 && value <= 0x46) {
    return value - 0x41 + 10;
  }
  return undefined;
}

function invalidByte(value: uint8): ProviderError {
  const codePoint = `U+${value.toString(16).toUpperCase().padStart(4, "0")}`;
  const suffix = IsPrint(value) ? ` ${QuoteRune(value)}` : "";
  return new ProviderError(`encoding/hex: invalid byte: ${codePoint}${suffix}`);
}
