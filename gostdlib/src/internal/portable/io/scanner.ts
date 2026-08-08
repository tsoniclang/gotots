import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger } from "../../host-integer.js";
import { goInterfaceEqual } from "../../runtime/interface.js";

type ScannerStep =
  | { readonly kind: "result"; readonly value: bool }
  | { readonly kind: "read"; readonly target: RuntimeSlice<uint8> };

export class ScannerState<Failure extends GoInterfaceValue> {
  #buffer: uint8[];
  #done = false;
  #emptyReads = 0;
  #end = 0;
  #failure: Failure | undefined;
  #pendingFailure: Failure | undefined;
  #start = 0;
  #token: gostring = "";

  constructor(
    private readonly badReadCount: Failure,
    private readonly tooLong: Failure,
    private readonly eof: Failure,
    private readonly noProgress: Failure,
    buffer: uint8[] = [],
  ) {
    this.#buffer = buffer;
  }

  copy(): ScannerState<Failure> {
    const target = new ScannerState(
      this.badReadCount,
      this.tooLong,
      this.eof,
      this.noProgress,
      this.#buffer,
    );
    target.#done = this.#done;
    target.#emptyReads = this.#emptyReads;
    target.#end = this.#end;
    target.#failure = this.#failure;
    target.#pendingFailure = this.#pendingFailure;
    target.#start = this.#start;
    target.#token = this.#token;
    return target;
  }

  Err(): Failure | undefined {
    return this.#failure;
  }

  Text(): gostring {
    return this.#token;
  }

  Next(): ScannerStep {
    if (this.#done) {
      return { kind: "result", value: false };
    }
    for (;;) {
      const newline = this.#buffer.indexOf(0x0a, this.#start);
      if (newline >= this.#start && newline < this.#end) {
        this.#token = scanLine(this.#buffer.slice(this.#start, newline + 1), true);
        this.#start = newline + 1;
        return { kind: "result", value: true };
      }
      if (this.#pendingFailure !== undefined) {
        if (this.#start < this.#end) {
          this.#token = scanLine(this.#buffer.slice(this.#start, this.#end), false);
          this.#start = this.#end;
          return { kind: "result", value: true };
        }
        this.#done = true;
        if (!goInterfaceEqual(this.#pendingFailure, this.eof)) {
          this.#failure = this.#pendingFailure;
        }
        return { kind: "result", value: false };
      }
      if (this.#end - this.#start >= 64 * 1024) {
        this.#failure = this.tooLong;
        this.#done = true;
        return { kind: "result", value: false };
      }
      return { kind: "read", target: scannerReadBuffer() };
    }
  }

  AcceptRead(
    target: RuntimeSlice<uint8>,
    count: int,
    failure: Failure | undefined,
  ): void {
    if (count < 0n || count > BigInt(target.length)) {
      this.#failure = this.badReadCount;
      this.#done = true;
      return;
    }
    const hostCount = hostInteger(count);
    for (let index = 0; index < hostCount; index += 1) {
      this.#buffer[this.#end + index] = target.get(index);
    }
    this.#end += hostCount;
    if (failure !== undefined) {
      this.#pendingFailure = failure;
    }
    if (count === 0n && failure === undefined) {
      this.#emptyReads += 1;
      if (this.#emptyReads > 100) {
        this.#failure = this.noProgress;
        this.#done = true;
      }
    } else {
      this.#emptyReads = 0;
    }
  }
}

interface ProviderReader<Failure extends GoInterfaceValue> {
  Read(target: RuntimeSlice<uint8>): [int, Failure | undefined];
}

export class ProviderScanner<
  Failure extends GoInterfaceValue,
  Source extends ProviderReader<Failure>,
> {
  readonly #state: ScannerState<Failure>;

  constructor(
    private readonly source: Source | undefined,
    badReadCount: Failure,
    tooLong: Failure,
    eof: Failure,
    noProgress: Failure,
  ) {
    this.#state = new ScannerState(badReadCount, tooLong, eof, noProgress);
  }

  Err(): Failure | undefined {
    return this.#state.Err();
  }

  Scan(): bool {
    for (;;) {
      const step = this.#state.Next();
      if (step.kind === "result") {
        return step.value;
      }
      const [count, failure] = requireSource(this.source).Read(step.target);
      this.#state.AcceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
  }
}

export function scannerReadBuffer(): RuntimeSlice<uint8> {
  return RuntimeSlice.make<uint8>(4096, 4096, 0);
}

function scanLine(source: readonly uint8[], terminated: boolean): gostring {
  let end = source.length - (terminated ? 1 : 0);
  if (end > 0 && source[end - 1] === 0x0d) {
    end -= 1;
  }
  let result = "";
  for (let index = 0; index < end; index += 1) {
    result += String.fromCharCode(source[index] ?? 0);
  }
  return result;
}

function requireSource<Source>(source: Source | undefined): Source {
  if (source === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return source;
}
