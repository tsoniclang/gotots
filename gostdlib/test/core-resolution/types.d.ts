export type bool = boolean;
export type int8 = number;
export type int16 = number;
export type int32 = number;
export type uint8 = number;
export type uint16 = number;
export type uint32 = number;
export type float32 = number;
export type float64 = number;

declare const rawPointerIdentity: unique symbol;
export interface RawPointer {
  readonly [rawPointerIdentity]: true;
}
