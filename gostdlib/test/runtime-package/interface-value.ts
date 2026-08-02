export abstract class GoInterfaceValue {
    abstract readonly $go$type: {
        readonly comparable: boolean;
    };
    abstract readonly $go$methods: ReadonlySet<object>;
    abstract $go$implements(contract: readonly object[]): boolean;
    abstract $go$equal(other: GoInterfaceValue): boolean;
    abstract $go$hash(): number;
    abstract readonly $go$formatString: boolean;
    abstract $go$format(verb: string, flags: string, precision: number | undefined): string;
}
export const GoErrorMethodToken: object = Object.freeze({});
export const GoRuntimeErrorMethodToken: object = Object.freeze({});
export interface GoError extends GoInterfaceValue {
    Error(): string;
}
