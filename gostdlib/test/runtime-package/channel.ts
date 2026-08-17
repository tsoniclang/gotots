export interface GoSelectCase {
    ready(): boolean;
    commit(): boolean | object;
    subscribe(claim: (failure: object | undefined) => boolean): () => void;
}
export interface GoReceiveChannel<T> {
    $length(): number;
    $capacity(): number;
    receive(): Promise<[
        T,
        boolean
    ]>;
    $selectReceive(accept: (value: T, ok: boolean) => void): GoSelectCase;
}
