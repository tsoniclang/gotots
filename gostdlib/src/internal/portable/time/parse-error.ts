import {
  GoErrorMethodToken,
  type GoError,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { ProviderInterfaceValue } from "../io/value.js";
import { timeQuote } from "./quote.js";

const parseErrorType = Object.freeze({ comparable: true });

export class ParseError extends ProviderInterfaceValue implements GoError {
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  constructor(
    public Layout: gostring,
    public Value: gostring,
    public LayoutElem: gostring,
    public ValueElem: gostring,
    public Message: gostring,
  ) {
    super(parseErrorType);
  }

  static Error(receiver: ParseError | undefined): gostring {
    if (receiver === undefined) {
      return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    return receiver.Error();
  }

  Error(): gostring {
    if (this.Message !== "") {
      return `parsing time ${timeQuote(this.Value)}${this.Message}`;
    }
    return `parsing time ${timeQuote(this.Value)} as ${timeQuote(this.Layout)}`
      + `: cannot parse ${timeQuote(this.ValueElem)}`
      + ` as ${timeQuote(this.LayoutElem)}`;
  }

  override $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    if (verb === "T") {
      return "*time.ParseError";
    }
    const message = this.Error();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}
