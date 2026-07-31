import { GoErrorMethodToken, GoInterfaceValue, GoRuntimeErrorMethodToken } from "./interface-value.js";
export class GoPanic {
    private constructor(public readonly value: GoInterfaceValue) {
    }
    static createRuntime(message: string): GoPanic {
        return new GoPanic(new GoRuntimePanicValue(message));
    }
    static raise(value: GoInterfaceValue): never {
        throw new GoPanic(value);
    }
    static raiseRuntime(message: string): never {
        throw new GoPanic(new GoRuntimePanicValue(message));
    }
    static rethrow(failure: object): never {
        throw failure;
    }
}
export class GoRuntimePanicValue implements GoInterfaceValue {
    constructor(public readonly message: string) {
    }
    readonly $go$type: object = GoRuntimePanicValue;
    readonly $go$methods: ReadonlySet<object> = new Set<object>([GoErrorMethodToken, GoRuntimeErrorMethodToken]);
    $go$implements(contract: readonly object[]): boolean {
        return contract.every((token: object): boolean => this.$go$methods.has(token));
    }
    $go$equal(other: GoInterfaceValue): boolean {
        return this === other;
    }
    $go$hash(): number {
        return 0;
    }
    Error(): string {
        return this.message;
    }
    RuntimeError(): void {
    }
}
