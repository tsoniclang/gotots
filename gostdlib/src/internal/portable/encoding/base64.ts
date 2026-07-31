import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import type { WriteCloser, Writer } from "../../../io.js";
import { ProviderInterfaceValue } from "../io/value.js";
import { ProviderError } from "../../runtime/error.js";
import { byteSlice, sliceValues } from "../../runtime/slice.js";

const standardAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

let createEncoding: (alphabet: gostring, padding: number) => Encoding;

export class Encoding {
  readonly #alphabet: gostring;
  readonly #padding: number;
  readonly #decode = new Map<number, number>();

  private constructor(
    alphabet: gostring,
    padding: number,
  ) {
    this.#alphabet = alphabet;
    this.#padding = padding;
    for (let index = 0; index < alphabet.length; index += 1) {
      this.#decode.set(alphabet.charCodeAt(index), index);
    }
  }

  static {
    createEncoding = (alphabet: gostring, padding: number): Encoding =>
      new Encoding(alphabet, padding);
  }

  static DecodeString(
    receiver: Encoding | undefined,
    source: gostring,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    return requireEncoding(receiver).#decodeString(source);
  }

  static EncodeToString(
    receiver: Encoding | undefined,
    source: RuntimeSlice<uint8>,
  ): gostring {
    return requireEncoding(receiver).#encodeBytes(sliceValues(source));
  }

  static EncodedLen(receiver: Encoding | undefined, length: int64): int64 {
    requireEncoding(receiver);
    return Math.floor((length + 2) / 3) * 4;
  }

  #encodeBytes(source: readonly uint8[]): gostring {
    let result = "";
    for (let index = 0; index < source.length; index += 3) {
      const first = source[index] ?? 0;
      const second = source[index + 1] ?? 0;
      const third = source[index + 2] ?? 0;
      const remaining = source.length - index;
      const value = (first << 16) | (second << 8) | third;
      result += this.#alphabet[(value >> 18) & 0x3f];
      result += this.#alphabet[(value >> 12) & 0x3f];
      result += remaining > 1
        ? this.#alphabet[(value >> 6) & 0x3f]
        : String.fromCharCode(this.#padding);
      result += remaining > 2
        ? this.#alphabet[value & 0x3f]
        : String.fromCharCode(this.#padding);
    }
    return result;
  }

  #decodeString(source: gostring): [RuntimeSlice<uint8>, GoError | undefined] {
    const output: uint8[] = [];
    let sourceIndex = 0;
    while (sourceIndex < source.length) {
      const quantum = this.#decodeQuantum(source, sourceIndex);
      output.push(...quantum.output);
      sourceIndex = quantum.next;
      if (quantum.failure !== undefined) {
        return [byteSlice(output), quantum.failure];
      }
    }
    return [byteSlice(output), undefined];
  }

  #decodeQuantum(source: gostring, start: number): DecodedQuantum {
    const decoded = [0, 0, 0, 0];
    let decodedLength = 4;
    let sourceIndex = start;
    let trailingFailure: ProviderError | undefined;

    for (let digitIndex = 0; digitIndex < decoded.length; digitIndex += 1) {
      if (sourceIndex === source.length) {
        if (digitIndex === 0) {
          return { next: sourceIndex, output: [], failure: undefined };
        }
        if (digitIndex === 1 || this.#padding >= 0) {
          return {
            next: sourceIndex,
            output: [],
            failure: corruptInput(sourceIndex - digitIndex),
          };
        }
        decodedLength = digitIndex;
        break;
      }

      const input = source.charCodeAt(sourceIndex);
      sourceIndex += 1;
      const value = this.#decode.get(input);
      if (value !== undefined) {
        decoded[digitIndex] = value;
        continue;
      }
      if (input === 0x0a || input === 0x0d) {
        digitIndex -= 1;
        continue;
      }
      if (input !== this.#padding) {
        return {
          next: sourceIndex,
          output: [],
          failure: corruptInput(sourceIndex - 1),
        };
      }
      if (digitIndex < 2) {
        return {
          next: sourceIndex,
          output: [],
          failure: corruptInput(sourceIndex - 1),
        };
      }
      if (digitIndex === 2) {
        sourceIndex = skipNewlines(source, sourceIndex);
        if (sourceIndex === source.length) {
          return {
            next: sourceIndex,
            output: [],
            failure: corruptInput(source.length),
          };
        }
        if (source.charCodeAt(sourceIndex) !== this.#padding) {
          return {
            next: sourceIndex,
            output: [],
            failure: corruptInput(sourceIndex - 1),
          };
        }
        sourceIndex += 1;
      }
      sourceIndex = skipNewlines(source, sourceIndex);
      if (sourceIndex < source.length) {
        trailingFailure = corruptInput(sourceIndex);
      }
      decodedLength = digitIndex;
      break;
    }

    const value =
      ((decoded[0] ?? 0) << 18) |
      ((decoded[1] ?? 0) << 12) |
      ((decoded[2] ?? 0) << 6) |
      (decoded[3] ?? 0);
    const output: uint8[] = [];
    if (decodedLength >= 2) {
      output.push((value >> 16) & 0xff);
    }
    if (decodedLength >= 3) {
      output.push((value >> 8) & 0xff);
    }
    if (decodedLength >= 4) {
      output.push(value & 0xff);
    }
    return { next: sourceIndex, output, failure: trailingFailure };
  }
}

type DecodedQuantum = {
  readonly next: number;
  readonly output: readonly uint8[];
  readonly failure: ProviderError | undefined;
};

export function NewEncoder(
  encoding: Encoding | undefined,
  writer: Writer | undefined,
): WriteCloser {
  return new Base64Encoder(encoding, writer);
}

class Base64Encoder extends ProviderInterfaceValue implements WriteCloser {
  private readonly pending: uint8[] = [];
  private failure: GoError | undefined;

  constructor(
    private readonly encoding: Encoding | undefined,
    private readonly writer: Writer | undefined,
  ) {
    super(Base64Encoder);
  }

  Write(source: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (this.failure !== undefined) {
      return [0, this.failure];
    }
    const values = sliceValues(source);
    let consumed = 0;
    if (this.pending.length > 0) {
      while (consumed < values.length && this.pending.length < 3) {
        this.pending.push(values[consumed] ?? 0);
        consumed += 1;
      }
      if (this.pending.length < 3) {
        return [consumed, undefined];
      }
      const encoded = Encoding.EncodeToString(
        this.encoding,
        byteSlice(this.pending),
      );
      const [, error] = requireWriter(this.writer).Write(byteSlice(byteCodes(encoded)));
      if (error !== undefined) {
        this.failure = error;
        return [consumed, error];
      }
      this.pending.length = 0;
    }

    const completeLength = Math.floor((values.length - consumed) / 3) * 3;
    if (completeLength > 0) {
      const complete = values.slice(consumed, consumed + completeLength);
      const encoded = Encoding.EncodeToString(this.encoding, byteSlice(complete));
      const [, error] = requireWriter(this.writer).Write(byteSlice(byteCodes(encoded)));
      if (error !== undefined) {
        this.failure = error;
        return [consumed, error];
      }
      consumed += completeLength;
    }
    while (consumed < values.length) {
      this.pending.push(values[consumed] ?? 0);
      consumed += 1;
    }
    return [consumed, undefined];
  }

  Close(): GoError | undefined {
    if (this.failure !== undefined) {
      return this.failure;
    }
    if (this.pending.length > 0) {
      const encoded = Encoding.EncodeToString(
        this.encoding,
        byteSlice(this.pending),
      );
      const [, error] = requireWriter(this.writer).Write(byteSlice(byteCodes(encoded)));
      this.failure = error;
      this.pending.length = 0;
    }
    return this.failure;
  }
}

function requireEncoding(encoding: Encoding | undefined): Encoding {
  if (encoding === undefined) {
    GoPanic.raiseRuntime("nil *base64.Encoding");
  }
  return encoding;
}

function requireWriter(writer: Writer | undefined): Writer {
  if (writer === undefined) {
    GoPanic.raiseRuntime("nil io.Writer");
  }
  return writer;
}

function corruptInput(offset: number): ProviderError {
  return new ProviderError(`illegal base64 data at input byte ${offset}`);
}

function byteCodes(value: gostring): uint8[] {
  const result: uint8[] = [];
  for (let index = 0; index < value.length; index += 1) {
    result.push(value.charCodeAt(index));
  }
  return result;
}

function skipNewlines(source: gostring, start: number): number {
  let index = start;
  while (
    index < source.length &&
    (source.charCodeAt(index) === 0x0a || source.charCodeAt(index) === 0x0d)
  ) {
    index += 1;
  }
  return index;
}

export function standardEncoding(): Encoding {
  return createEncoding(standardAlphabet, 0x3d);
}
