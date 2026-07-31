import type { GoInterfaceValue } from "./interface-value.js";

export declare class GoPanic {
  readonly value: GoInterfaceValue;
  static createRuntime(message: string): GoPanic;
  static raise(value: GoInterfaceValue): never;
  static raiseRuntime(message: string): never;
  static rethrow(failure: object): never;
}

export declare class GoRecovery {
  constructor(pending: GoPanic | undefined);
  take(): GoInterfaceValue | undefined;
  recovered(): boolean;
}
