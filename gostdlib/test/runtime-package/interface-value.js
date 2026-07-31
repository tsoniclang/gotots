const errorMethod = Object.freeze({});

export class GoInterfaceValue {}

export class GoBasicError extends GoInterfaceValue {
  constructor(message) {
    super();
    this.message = message;
    this.$go$type = GoBasicError;
    this.$go$methods = new Set([errorMethod]);
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
