declare const rawPointerIdentity: unique symbol;
export interface RawPointer {
  readonly [rawPointerIdentity]: true;
}
