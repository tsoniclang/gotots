import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";
import { goInterfaceEqual } from "../runtime/interface.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";

type ScannerStep =
  | { readonly kind: "result"; readonly value: bool }
  | { readonly kind: "read"; readonly target: RuntimeSlice<uint8> };

class ScannerState<Failure extends GoInterfaceValue> {
  readonly #buffer: uint8[] = [];
  #done = false;
  #emptyReads = 0;
  #failure: Failure | undefined;
  #pendingFailure: Failure | undefined;
  #token: gostring = "";

  constructor(
    private readonly badReadCount: Failure,
    private readonly tooLong: Failure,
    private readonly eof: Failure,
    private readonly noProgress: Failure,
  ) {}

  Err(): Failure | undefined {
    return this.#failure;
  }

  Text(): gostring {
    return this.#token;
  }

  next(): ScannerStep {
    if (this.#done) {
      return { kind: "result", value: false };
    }
    for (;;) {
      const newline = this.#buffer.indexOf(0x0a);
      if (newline >= 0) {
        this.#token = scanLine(this.#buffer.splice(0, newline + 1), true);
        return { kind: "result", value: true };
      }
      if (this.#pendingFailure !== undefined) {
        if (this.#buffer.length > 0) {
          this.#token = scanLine(this.#buffer.splice(0), false);
          return { kind: "result", value: true };
        }
        this.#done = true;
        if (!goInterfaceEqual(this.#pendingFailure, this.eof)) {
          this.#failure = this.#pendingFailure;
        }
        return { kind: "result", value: false };
      }
      if (this.#buffer.length >= 64 * 1024) {
        this.#failure = this.tooLong;
        this.#done = true;
        return { kind: "result", value: false };
      }
      return { kind: "read", target: scannerReadBuffer() };
    }
  }

  acceptRead(
    target: RuntimeSlice<uint8>,
    count: int64,
    failure: Failure | undefined,
  ): void {
    if (!Number.isInteger(count) || count < 0 || count > target.length) {
      this.#failure = this.badReadCount;
      this.#done = true;
      return;
    }
    for (let index = 0; index < count; index += 1) {
      this.#buffer.push(target.get(index));
    }
    if (failure !== undefined) {
      this.#pendingFailure = failure;
    }
    if (count === 0 && failure === undefined) {
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

export class CanonicalScannerSync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceSync<Failure>,
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

  static Err<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceSync<Failure>>(
    receiver: CanonicalScannerSync<Failure, Source> | undefined,
  ): Failure | undefined {
    return requireScannerSync(receiver).Err();
  }

  static Scan<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceSync<Failure>>(
    receiver: CanonicalScannerSync<Failure, Source> | undefined,
  ): bool {
    return requireScannerSync(receiver).Scan();
  }

  static Text<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceSync<Failure>>(
    receiver: CanonicalScannerSync<Failure, Source> | undefined,
  ): gostring {
    return requireScannerSync(receiver).Text();
  }

  Err(): Failure | undefined {
    return this.#state.Err();
  }

  Scan(): bool {
    for (;;) {
      const step = this.#state.next();
      if (step.kind === "result") {
        return step.value;
      }
      const [count, failure] = requireSource(this.source).Read(step.target);
      this.#state.acceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
  }
}

export function NewScannerCanonicalSync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceSync<Failure>,
>(
  source: Source | undefined,
  badReadCount: Failure | undefined,
  tooLong: Failure | undefined,
  eof: Failure | undefined,
  noProgress: Failure | undefined,
): CanonicalScannerSync<Failure, Source> {
  return new CanonicalScannerSync(
    source,
    requireFailure(badReadCount),
    requireFailure(tooLong),
    requireFailure(eof),
    requireFailure(noProgress),
  );
}

export class CanonicalScannerAsync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceAsync<Failure>,
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

  static Err<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceAsync<Failure>>(
    receiver: CanonicalScannerAsync<Failure, Source> | undefined,
  ): Failure | undefined {
    return requireScannerAsync(receiver).Err();
  }

  static Scan<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceAsync<Failure>>(
    receiver: CanonicalScannerAsync<Failure, Source> | undefined,
  ): Promise<bool> {
    return requireScannerAsync(receiver).Scan();
  }

  static Text<Failure extends GoInterfaceValue, Source extends CanonicalReaderSourceAsync<Failure>>(
    receiver: CanonicalScannerAsync<Failure, Source> | undefined,
  ): gostring {
    return requireScannerAsync(receiver).Text();
  }

  Err(): Failure | undefined {
    return this.#state.Err();
  }

  async Scan(): Promise<bool> {
    for (;;) {
      const step = this.#state.next();
      if (step.kind === "result") {
        return step.value;
      }
      const [count, failure] = await requireSource(this.source).Read(step.target);
      this.#state.acceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
  }
}

export function NewScannerCanonicalAsync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceAsync<Failure>,
>(
  source: Source | undefined,
  badReadCount: Failure | undefined,
  tooLong: Failure | undefined,
  eof: Failure | undefined,
  noProgress: Failure | undefined,
): CanonicalScannerAsync<Failure, Source> {
  return new CanonicalScannerAsync(
    source,
    requireFailure(badReadCount),
    requireFailure(tooLong),
    requireFailure(eof),
    requireFailure(noProgress),
  );
}

function requireFailure<Failure>(failure: Failure | undefined): Failure {
  if (failure === undefined) {
    GoPanic.raiseRuntime("gostdlib provider supplied a nil scanner failure");
  }
  return failure;
}

function requireSource<Source>(source: Source | undefined): Source {
  if (source === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return source;
}

function requireScannerSync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceSync<Failure>,
>(
  receiver: CanonicalScannerSync<Failure, Source> | undefined,
): CanonicalScannerSync<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

function requireScannerAsync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceAsync<Failure>,
>(
  receiver: CanonicalScannerAsync<Failure, Source> | undefined,
): CanonicalScannerAsync<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

function scannerReadBuffer(): RuntimeSlice<uint8> {
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
