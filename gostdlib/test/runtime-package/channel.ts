export interface GoSelectCase {
    ready(): boolean;
    commit(): boolean | object;
}
export interface GoReceiveChannel<T> {
    $length(): number;
    $capacity(): number;
    receive(): [
        T,
        boolean
    ];
    $selectReceive(accept: (value: T, ok: boolean) => void): GoSelectCase;
    $observeClose(observer: () => void): () => void;
}
