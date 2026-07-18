// The Go panic carrier module source.
package abi

const gopanicSource = `// The Go panic carrier: distinguishes translated Go panics from host
// errors and carries the exact Go runtime-error message.
export class GoPanic extends Error {
  // value retains the exact typed panic payload through deferred
  // execution and recovery; format lazily produces the printed message
  // at the point the runtime would finally print it (after all defers).
  readonly value: unknown;
  private readonly format: () => string;
  private computed: string | undefined;
  constructor(message: string, value?: unknown, format?: () => string) {
    super(message);
    this.name = "GoPanic";
    this.value = value === undefined ? message : value;
    this.format = format ?? (() => message);
  }
  get goMessage(): string {
    if (this.computed === undefined) {
      this.computed = this.format();
    }
    return this.computed;
  }
}

export function goPanicDivide(): never {
  throw new GoPanic("runtime error: integer divide by zero");
}

export function goPanicShift(): never {
  throw new GoPanic("runtime error: negative shift amount");
}

export function goPanicNil(): never {
  throw new GoPanic("runtime error: invalid memory address or nil pointer dereference");
}

export function goPanicNilMapWrite(): never {
  throw new GoPanic("assignment to entry in nil map");
}
`
