export abstract class GoInterfaceValue {
    abstract readonly $go$type: object;
    abstract readonly $go$methods: ReadonlySet<object>;
    abstract $go$implements(contract: readonly object[]): boolean;
    abstract $go$equal(other: GoInterfaceValue): boolean;
    abstract $go$hash(): number;
}
export const GoErrorMethodToken: object = Object.freeze({});
export const GoRuntimeErrorMethodToken: object = Object.freeze({});
export interface GoError extends GoInterfaceValue {
    Error(): string;
}
