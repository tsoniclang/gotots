import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import { ProviderError } from "../../runtime/error.js";
import { byteSlice, sliceValues } from "../../runtime/slice.js";

const standardAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
const hexadecimalAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUV";
const standardPadding = 0x3d;

let createEncoding: (alphabet: gostring) => Encoding;

export class Encoding {
  readonly #alphabet: gostring;
  readonly #decode = new Map<uint8, uint8>();

  private constructor(alphabet: gostring) {
    this.#alphabet = alphabet;
    for (let index = 0; index < alphabet.length; index += 1) {
      this.#decode.set(alphabet.charCodeAt(index), index);
    }
  }

  static {
    createEncoding = (alphabet: gostring): Encoding => new Encoding(alphabet);
  }

  static AppendDecode(
    receiver: Encoding | undefined,
    destination: RuntimeSlice<uint8>,
    source: RuntimeSlice<uint8>,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    const result = requireEncoding(receiver).#decodeBytes(sliceValues(source));
    return [destination.append(0, result.output), result.failure];
  }

  static AppendEncode(
    receiver: Encoding | undefined,
    destination: RuntimeSlice<uint8>,
    source: RuntimeSlice<uint8>,
  ): RuntimeSlice<uint8> {
    const encoded = requireEncoding(receiver).#encodeBytes(sliceValues(source));
    return destination.append(0, encoded);
  }

  static EncodedLen(receiver: Encoding | undefined, length: int64): int64 {
    requireEncoding(receiver);
    return Math.floor((length + 4) / 5) * 8;
  }

  #encodeBytes(source: readonly uint8[]): uint8[] {
    const output: uint8[] = [];
    for (let index = 0; index < source.length; index += 5) {
      const remaining = Math.min(5, source.length - index);
      const first = source[index] ?? 0;
      const second = source[index + 1] ?? 0;
      const third = source[index + 2] ?? 0;
      const fourth = source[index + 3] ?? 0;
      const fifth = source[index + 4] ?? 0;
      const digits = [
        first >> 3,
        ((first & 0x07) << 2) | (second >> 6),
        (second >> 1) & 0x1f,
        ((second & 0x01) << 4) | (third >> 4),
        ((third & 0x0f) << 1) | (fourth >> 7),
        (fourth >> 2) & 0x1f,
        ((fourth & 0x03) << 3) | (fifth >> 5),
        fifth & 0x1f,
      ];
      const digitCount = [0, 2, 4, 5, 7, 8][remaining] ?? 0;
      for (let digit = 0; digit < digitCount; digit += 1) {
        output.push(this.#alphabet.charCodeAt(digits[digit] ?? 0));
      }
      for (let digit = digitCount; digit < 8; digit += 1) {
        output.push(standardPadding);
      }
    }
    return output;
  }

  #decodeBytes(source: readonly uint8[]): DecodeResult {
    const encoded = source.filter(
      (value): boolean => value !== 0x0a && value !== 0x0d,
    );
    const output: uint8[] = [];
    let sourceIndex = 0;
    let ended = false;
    while (sourceIndex < encoded.length && !ended) {
      const digits = [0, 0, 0, 0, 0, 0, 0, 0];
      let digitCount = 8;
      for (let digit = 0; digit < 8; digit += 1) {
        if (sourceIndex === encoded.length) {
          return {
            output,
            failure: corruptInput(sourceIndex - digit),
          };
        }
        const input = encoded[sourceIndex] ?? 0;
        sourceIndex += 1;
        const remaining = encoded.length - sourceIndex;
        if (input === standardPadding && digit >= 2 && remaining < 8) {
          if (remaining + digit < 7) {
            return { output, failure: corruptInput(encoded.length) };
          }
          for (let padding = 0; padding < 7 - digit; padding += 1) {
            if (
              remaining > padding &&
              encoded[sourceIndex + padding] !== standardPadding
            ) {
              return {
                output,
                failure: corruptInput(sourceIndex + padding - 1),
              };
            }
          }
          digitCount = digit;
          ended = true;
          if (digitCount === 3 || digitCount === 6) {
            return { output, failure: corruptInput(sourceIndex - 1) };
          }
          break;
        }
        const decoded = this.#decode.get(input);
        if (decoded === undefined) {
          return { output, failure: corruptInput(sourceIndex - 1) };
        }
        digits[digit] = decoded;
      }

      if (digitCount >= 2) {
        output.push(
          (((digits[0] ?? 0) << 3) | ((digits[1] ?? 0) >> 2)) & 0xff,
        );
      }
      if (digitCount >= 4) {
        output.push(
          (
            ((digits[1] ?? 0) << 6) |
            ((digits[2] ?? 0) << 1) |
            ((digits[3] ?? 0) >> 4)
          ) & 0xff,
        );
      }
      if (digitCount >= 5) {
        output.push(
          (((digits[3] ?? 0) << 4) | ((digits[4] ?? 0) >> 1)) & 0xff,
        );
      }
      if (digitCount >= 7) {
        output.push(
          (
            ((digits[4] ?? 0) << 7) |
            ((digits[5] ?? 0) << 2) |
            ((digits[6] ?? 0) >> 3)
          ) & 0xff,
        );
      }
      if (digitCount >= 8) {
        output.push(
          (((digits[6] ?? 0) << 5) | (digits[7] ?? 0)) & 0xff,
        );
      }
    }
    return { output, failure: undefined };
  }
}

type DecodeResult = {
  readonly output: uint8[];
  readonly failure: ProviderError | undefined;
};

function requireEncoding(encoding: Encoding | undefined): Encoding {
  if (encoding === undefined) {
    GoPanic.raiseRuntime("nil *base32.Encoding");
  }
  return encoding;
}

function corruptInput(offset: number): ProviderError {
  return new ProviderError(`illegal base32 data at input byte ${offset}`);
}

export function standardEncoding(): Encoding {
  return createEncoding(standardAlphabet);
}

export function hexadecimalEncoding(): Encoding {
  return createEncoding(hexadecimalAlphabet);
}
