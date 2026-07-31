export const GoErrorMethodToken = Object.freeze({});
export const GoRuntimeErrorMethodToken = Object.freeze({});
export const GoError$contract = Object.freeze([GoErrorMethodToken]);

export class GoInterfaceValue {}

export function GoError$is(value) {
  return value !== undefined && value.$go$implements(GoError$contract);
}

export class GoBasicError extends GoInterfaceValue {
  constructor(message) {
    super();
    this.message = message;
    this.$go$type = GoBasicError;
    this.$go$methods = new Set([GoErrorMethodToken]);
  }

  $go$implements(contract) {
    return contract.every((method) => this.$go$methods.has(method));
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
}
