export abstract class GoInterfaceValue {
  abstract readonly $go$type: object;
  abstract readonly $go$methods: ReadonlySet<object>;
  abstract $go$implements(contract: readonly object[]): boolean;
  abstract $go$equal(other: GoInterfaceValue): boolean;
  abstract $go$hash(): number;
}

export interface GoError extends GoInterfaceValue {
  Error(): string;
}

export declare const GoErrorMethodToken: object;
export declare const GoRuntimeErrorMethodToken: object;
export declare const GoError$contract: readonly object[];
export declare function GoError$is(value: GoInterfaceValue | undefined): value is GoError;

export declare class GoBasicError extends GoInterfaceValue implements GoError {
  constructor(message: string);
  readonly $go$type: object;
  readonly $go$methods: ReadonlySet<object>;
  $go$implements(contract: readonly object[]): boolean;
  $go$equal(other: GoInterfaceValue): boolean;
  $go$hash(): number;
  Error(): string;
}
