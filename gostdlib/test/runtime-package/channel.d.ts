export interface GoReceiveChannel<T> {
  receive(): Promise<[T, boolean]>;
}

export interface GoSendChannel<T> {
  send(value: T): Promise<void>;
  close(): void;
}

export declare class GoChannel<T> implements GoReceiveChannel<T>, GoSendChannel<T> {
  static make<T>(capacity: number | bigint, zero: () => T, copy: (value: T) => T): GoChannel<T>;
  static send<T>(channel: GoSendChannel<T> | undefined, value: T): Promise<void>;
  static receive<T>(channel: GoReceiveChannel<T> | undefined): Promise<[T, boolean]>;
  static close<T>(channel: GoSendChannel<T> | undefined): void;
  send(value: T): Promise<void>;
  receive(): Promise<[T, boolean]>;
  close(): void;
}
