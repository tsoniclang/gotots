import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  Awaitable,
  gostring,
  int,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";
import { integerFromHost } from "../../host-integer.js";

import type { WriteCloser, Writer } from "../../../io.js";
import { ProviderInterfaceValue } from "../io/value.js";
import { ProviderError } from "../../runtime/error.js";
import { byteSlice, sliceValues } from "../../runtime/slice.js";

const standardAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
const urlAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

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

  static AppendDecode(
    receiver: Encoding | undefined,
    destination: RuntimeSlice<uint8>,
    source: RuntimeSlice<uint8>,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    const encoding = requireEncoding(receiver);
    const [decoded, failure] = encoding.#decodeString(
      String.fromCharCode(...sliceValues(source)),
    );
    return [destination.append(0, sliceValues(decoded)), failure];
  }

  static AppendEncode(
    receiver: Encoding | undefined,
    destination: RuntimeSlice<uint8>,
    source: RuntimeSlice<uint8>,
  ): RuntimeSlice<uint8> {
    const encoded = requireEncoding(receiver).#encodeBytes(sliceValues(source));
    return destination.append(0, byteCodes(encoded));
  }

  static EncodeToString(
    receiver: Encoding | undefined,
    source: RuntimeSlice<uint8>,
  ): gostring {
    return requireEncoding(receiver).#encodeBytes(sliceValues(source));
  }

  static EncodedLen(receiver: Encoding | undefined, length: int): int {
    requireEncoding(receiver);
    return ((length + 2n) / 3n) * 4n;
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

export type Base64EncoderStep<Result, Failure> =
  | {
    readonly kind: "done";
    readonly result: Result;
  }
  | {
    readonly kind: "write";
    readonly output: RuntimeSlice<uint8>;
    readonly resume: (
      failure: Failure | undefined,
    ) => Base64EncoderStep<Result, Failure>;
  };

export class Base64EncoderState<Failure> {
  readonly #pending: uint8[] = [];
  #failure: Failure | undefined;

  constructor(private readonly encoding: Encoding | undefined) {}

  beginWrite(
    source: RuntimeSlice<uint8>,
  ): Base64EncoderStep<[int, Failure | undefined], Failure> {
    if (this.#failure !== undefined) {
      return done([0n, this.#failure]);
    }
    const values = sliceValues(source);
    let consumed = 0;
    if (this.#pending.length > 0) {
      while (consumed < values.length && this.#pending.length < 3) {
        this.#pending.push(values[consumed] ?? 0);
        consumed += 1;
      }
      if (this.#pending.length < 3) {
        return done([integerFromHost(consumed), undefined]);
      }
      const output = this.#encode(this.#pending);
      return write(output, (failure) => {
        if (failure !== undefined) {
          return this.#failWrite(consumed, failure);
        }
        this.#pending.length = 0;
        return this.#writeInterior(values, consumed);
      });
    }
    return this.#writeInterior(values, consumed);
  }

  beginClose(): Base64EncoderStep<Failure | undefined, Failure> {
    if (this.#failure !== undefined || this.#pending.length === 0) {
      return done(this.#failure);
    }
    const output = this.#encode(this.#pending);
    return write(output, (failure) => {
      this.#failure = failure;
      this.#pending.length = 0;
      return done(failure);
    });
  }

  #writeInterior(
    values: readonly uint8[],
    consumed: number,
  ): Base64EncoderStep<[int, Failure | undefined], Failure> {
    const completeLength = Math.min(
      Math.floor((values.length - consumed) / 3) * 3,
      768,
    );
    if (completeLength === 0) {
      return this.#finishWrite(values, consumed);
    }
    const output = this.#encode(
      values.slice(consumed, consumed + completeLength),
    );
    return write(output, (failure) => {
      if (failure !== undefined) {
        return this.#failWrite(consumed, failure);
      }
      return this.#writeInterior(values, consumed + completeLength);
    });
  }

  #finishWrite(
    values: readonly uint8[],
    consumed: number,
  ): Base64EncoderStep<[int, Failure | undefined], Failure> {
    while (consumed < values.length) {
      this.#pending.push(values[consumed] ?? 0);
      consumed += 1;
    }
    return done([integerFromHost(consumed), undefined]);
  }

  #failWrite(
    consumed: number,
    failure: Failure,
  ): Base64EncoderStep<[int, Failure | undefined], Failure> {
    this.#failure = failure;
    return done([integerFromHost(consumed), failure]);
  }

  #encode(source: readonly uint8[]): RuntimeSlice<uint8> {
    return byteSlice(byteCodes(Encoding.EncodeToString(
      this.encoding,
      byteSlice(source),
    )));
  }
}

export function runBase64EncoderSync<Result, Failure>(
  initial: Base64EncoderStep<Result, Failure>,
  writeOutput: (
    output: RuntimeSlice<uint8>,
  ) => [int, Failure | undefined],
): Result {
  let current = initial;
  while (current.kind === "write") {
    const result = writeOutput(current.output);
    current = current.resume(result[1]);
  }
  return current.result;
}

export async function runBase64EncoderAsync<Result, Failure>(
  initial: Base64EncoderStep<Result, Failure>,
  writeOutput: (
    output: RuntimeSlice<uint8>,
  ) => Awaitable<[int, Failure | undefined]>,
): Promise<Result> {
  let current = initial;
  while (current.kind === "write") {
    const result = await writeOutput(current.output);
    current = current.resume(result[1]);
  }
  return current.result;
}

class Base64Encoder extends ProviderInterfaceValue implements WriteCloser {
  static readonly comparable = true;
  readonly #state: Base64EncoderState<GoError>;

  constructor(
    encoding: Encoding | undefined,
    private readonly writer: Writer | undefined,
  ) {
    super(Base64Encoder);
    this.#state = new Base64EncoderState(encoding);
  }

  Write(source: RuntimeSlice<uint8>): [int, GoError | undefined] {
    return runBase64EncoderSync(
      this.#state.beginWrite(source),
      (output) => requireWriter(this.writer).Write(output),
    );
  }

  Close(): GoError | undefined {
    return runBase64EncoderSync(
      this.#state.beginClose(),
      (output) => requireWriter(this.writer).Write(output),
    );
  }
}

function done<Result, Failure>(
  result: Result,
): Base64EncoderStep<Result, Failure> {
  return { kind: "done", result };
}

function write<Result, Failure>(
  output: RuntimeSlice<uint8>,
  resume: (
    failure: Failure | undefined,
  ) => Base64EncoderStep<Result, Failure>,
): Base64EncoderStep<Result, Failure> {
  return { kind: "write", output, resume };
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

export function urlEncoding(): Encoding {
  return createEncoding(urlAlphabet, 0x3d);
}
