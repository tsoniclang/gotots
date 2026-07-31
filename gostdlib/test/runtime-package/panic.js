import {
  GoErrorMethodToken,
  GoInterfaceValue,
  GoRuntimeErrorMethodToken,
} from "./interface-value.js";

class GoRuntimePanicValue extends GoInterfaceValue {
  constructor(message) {
    super();
    this.message = message;
    this.$go$type = GoRuntimePanicValue;
    this.$go$methods = new Set([GoErrorMethodToken, GoRuntimeErrorMethodToken]);
  }

  $go$implements(contract) {
    return contract.every((token) => this.$go$methods.has(token));
  }

  $go$equal(other) {
    return this === other;
  }

  $go$hash() {
    return 0;
  }

  Error() {
    return this.message;
  }

  RuntimeError() {}
}

export class GoPanic {
  constructor(value) {
    this.value = value;
  }

  static createRuntime(message) {
    return new GoPanic(new GoRuntimePanicValue(message));
  }

  static raise(value) {
    throw new GoPanic(value);
  }

  static raiseRuntime(message) {
    throw GoPanic.createRuntime(message);
  }

  static rethrow(failure) {
    throw failure;
  }
}

export class GoRecovery {
  constructor(pending) {
    this.pending = pending;
  }

  take() {
    const pending = this.pending;
    if (pending === undefined) {
      return undefined;
    }
    this.pending = undefined;
    return pending.value;
  }

  recovered() {
    return this.pending === undefined;
  }
}
