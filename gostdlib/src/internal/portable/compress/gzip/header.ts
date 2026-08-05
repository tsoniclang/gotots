import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import type { Reader } from "../../../../io.js";
import { state as ioState } from "../../../../io.js";
import { byteSlice } from "../../../runtime/slice.js";
import { ProviderError } from "../../../runtime/error.js";
import { goInterfaceEqual } from "../../../runtime/interface.js";
import { unexpectedEOF } from "../../io/read.js";

export interface GzipHeader {
  readonly comment: string;
  readonly extra: RuntimeSlice<uint8>;
  readonly modificationTimeSeconds: number;
  readonly name: string;
  readonly operatingSystem: uint8;
}

export type GzipSourceStep<Result, Failure> =
  | {
    readonly kind: "done";
    readonly result: Result;
  }
  | {
    readonly kind: "read";
    readonly destination: RuntimeSlice<uint8>;
    readonly resume: (
      count: int,
      failure: Failure | undefined,
    ) => GzipSourceStep<Result, Failure>;
  };

type ExactRead<Failure> =
  | { readonly kind: "values"; readonly values: readonly number[] }
  | { readonly kind: "failure"; readonly failure: Failure }
  | { readonly kind: "read" };

type HeaderPhase =
  | "fixed"
  | "extra-length"
  | "extra"
  | "name"
  | "comment"
  | "checksum"
  | "done";

export class GzipSourceState<Failure extends GoInterfaceValue> {
  private readonly consumed: number[] = [];
  private readonly headerBytes: number[] = [];
  private readonly pending: number[] = [];
  private terminalFailure: Failure | undefined;
  private phase: HeaderPhase = "fixed";
  private flags = 0;
  private extraLength = 0;
  private extra = RuntimeSlice.nil<uint8>();
  private readonly name: number[] = [];
  private readonly comment: number[] = [];
  private modificationTimeSeconds = 0;
  private operatingSystem: uint8 = 0;
  private emptyReads = 0;

  constructor(
    private readonly eof: Failure,
    private readonly unexpectedEOF: Failure,
    private readonly noProgress: Failure,
    private readonly invalidHeader: () => Failure,
  ) {}

  beginHeader(): GzipSourceStep<
    [GzipHeader | undefined, Failure | undefined],
    Failure
  > {
    return this.advanceHeader();
  }

  beginDrain(): GzipSourceStep<
    [RuntimeSlice<uint8>, Failure | undefined],
    Failure
  > {
    this.pending.length = 0;
    return this.advanceDrain();
  }

  private advanceHeader(): GzipSourceStep<
    [GzipHeader | undefined, Failure | undefined],
    Failure
  > {
    for (;;) {
      switch (this.phase) {
        case "fixed": {
          const fixed = this.takeExact(10);
          if (fixed.kind !== "values") {
            return this.resolveHeaderRead(fixed);
          }
          if (
            fixed.values[0] !== 0x1f
            || fixed.values[1] !== 0x8b
            || fixed.values[2] !== 8
            || ((fixed.values[3] ?? 0) & 0xe0) !== 0
          ) {
            return done([undefined, undefined]);
          }
          this.flags = fixed.values[3] ?? 0;
          this.modificationTimeSeconds = (
            (fixed.values[4] ?? 0)
            + (fixed.values[5] ?? 0) * 0x100
            + (fixed.values[6] ?? 0) * 0x1_0000
            + (fixed.values[7] ?? 0) * 0x1_000_000
          ) >>> 0;
          this.operatingSystem = fixed.values[9] ?? 0;
          this.phase = (this.flags & 0x04) === 0 ? "name" : "extra-length";
          break;
        }
        case "extra-length": {
          const length = this.takeExact(2);
          if (length.kind !== "values") {
            return this.resolveHeaderRead(length);
          }
          this.extraLength =
            (length.values[0] ?? 0) | ((length.values[1] ?? 0) << 8);
          this.phase = "extra";
          break;
        }
        case "extra": {
          const extra = this.takeExact(this.extraLength);
          if (extra.kind !== "values") {
            return this.resolveHeaderRead(extra);
          }
          this.extra = byteSlice(extra.values);
          this.phase = "name";
          break;
        }
        case "name": {
          if ((this.flags & 0x08) === 0) {
            this.phase = "comment";
            break;
          }
          const next = this.takeExact(1);
          if (next.kind !== "values") {
            return this.resolveHeaderRead(next);
          }
          const value = next.values[0] ?? 0;
          if (value === 0) {
            this.phase = "comment";
            break;
          }
          if (this.name.length >= 511) {
            return done([undefined, this.invalidHeader()]);
          }
          this.name.push(value);
          break;
        }
        case "comment": {
          if ((this.flags & 0x10) === 0) {
            this.phase = "checksum";
            break;
          }
          const next = this.takeExact(1);
          if (next.kind !== "values") {
            return this.resolveHeaderRead(next);
          }
          const value = next.values[0] ?? 0;
          if (value === 0) {
            this.phase = "checksum";
            break;
          }
          if (this.comment.length >= 511) {
            return done([undefined, this.invalidHeader()]);
          }
          this.comment.push(value);
          break;
        }
        case "checksum": {
          if ((this.flags & 0x02) !== 0) {
            const checksum = this.takeExact(2);
            if (checksum.kind !== "values") {
              return this.resolveHeaderRead(checksum);
            }
            const expected =
              (checksum.values[0] ?? 0) | ((checksum.values[1] ?? 0) << 8);
            const actual = crc32(this.headerBytes.slice(0, -2)) & 0xffff;
            if (expected !== actual) {
              return done([undefined, this.invalidHeader()]);
            }
          }
          this.phase = "done";
          break;
        }
        case "done":
          return done([{
            comment: String.fromCharCode(...this.comment),
            extra: this.extra,
            modificationTimeSeconds: this.modificationTimeSeconds,
            name: String.fromCharCode(...this.name),
            operatingSystem: this.operatingSystem,
          }, undefined]);
      }
    }
  }

  private resolveHeaderRead(
    result: Exclude<ExactRead<Failure>, { readonly kind: "values" }>,
  ): GzipSourceStep<
    [GzipHeader | undefined, Failure | undefined],
    Failure
  > {
    if (result.kind === "failure") {
      return done([undefined, result.failure]);
    }
    return this.read(() => this.advanceHeader());
  }

  private advanceDrain(): GzipSourceStep<
    [RuntimeSlice<uint8>, Failure | undefined],
    Failure
  > {
    if (this.terminalFailure !== undefined) {
      return done([byteSlice(this.consumed), this.terminalFailure]);
    }
    return this.read(() => this.advanceDrain());
  }

  private takeExact(count: number): ExactRead<Failure> {
    if (this.pending.length >= count) {
      const values = this.pending.splice(0, count);
      this.headerBytes.push(...values);
      return { kind: "values", values };
    }
    if (this.terminalFailure === undefined) {
      return { kind: "read" };
    }
    return {
      kind: "failure",
      failure: this.pending.length > 0 && goInterfaceEqual(
        this.terminalFailure,
        this.eof,
      )
        ? this.unexpectedEOF
        : this.terminalFailure,
    };
  }

  private read<Result>(
    resume: () => GzipSourceStep<Result, Failure>,
  ): GzipSourceStep<Result, Failure> {
    const destination = RuntimeSlice.make<uint8>(4096, 4096, 0);
    return {
      kind: "read",
      destination,
      resume: (count, failure) => {
        if (count === 0n && failure === undefined) {
          this.emptyReads += 1;
          if (this.emptyReads >= 100) {
            this.terminalFailure = this.noProgress;
          }
        } else {
          this.emptyReads = 0;
        }
        if (failure !== undefined) {
          this.terminalFailure = failure;
        }
        for (let index = 0; index < count; index += 1) {
          const value = destination.get(index);
          this.pending.push(value);
          this.consumed.push(value);
        }
        return resume();
      },
    };
  }
}

export function runGzipSourceSync<Result, Failure>(
  initial: GzipSourceStep<Result, Failure>,
  readSource: (
    destination: RuntimeSlice<uint8>,
  ) => [int, Failure | undefined],
): Result {
  let current = initial;
  while (current.kind === "read") {
    const [count, failure] = readSource(current.destination);
    current = current.resume(count, failure);
  }
  return current.result;
}

export async function runGzipSourceAsync<Result, Failure>(
  initial: GzipSourceStep<Result, Failure>,
  readSource: (
    destination: RuntimeSlice<uint8>,
  ) => Awaitable<[int, Failure | undefined]>,
): Promise<Result> {
  let current = initial;
  while (current.kind === "read") {
    const [count, failure] = await readSource(current.destination);
    current = current.resume(count, failure);
  }
  return current.result;
}

export class GzipSource {
  private readonly state: GzipSourceState<GoError>;

  constructor(private readonly source: Reader) {
    this.state = new GzipSourceState(
      ioState.EOF,
      unexpectedEOF,
      ioState.ErrNoProgress,
      () => new ProviderError("gzip: invalid header"),
    );
  }

  ReadHeader(): [GzipHeader | undefined, GoError | undefined] {
    return runGzipSourceSync(
      this.state.beginHeader(),
      (destination) => this.source.Read(destination),
    );
  }

  Drain(): [RuntimeSlice<uint8>, GoError | undefined] {
    return runGzipSourceSync(
      this.state.beginDrain(),
      (destination) => this.source.Read(destination),
    );
  }
}

function done<Result, Failure>(
  result: Result,
): GzipSourceStep<Result, Failure> {
  return { kind: "done", result };
}

function crc32(values: readonly number[]): number {
  let checksum = 0xffff_ffff;
  for (const value of values) {
    checksum ^= value;
    for (let bit = 0; bit < 8; bit += 1) {
      checksum = (checksum >>> 1) ^ ((checksum & 1) === 0 ? 0 : 0xedb8_8320);
    }
  }
  return (checksum ^ 0xffff_ffff) >>> 0;
}
