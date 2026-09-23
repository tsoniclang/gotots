import type {
  bool as TsonicBool,
  int8 as TsonicInt8,
  int16 as TsonicInt16,
  int32 as TsonicInt32,
  int64 as TsonicInt64,
  uint8 as TsonicUint8,
  uint16 as TsonicUint16,
  uint32 as TsonicUint32,
  uint64 as TsonicUint64,
  float32 as TsonicFloat32,
  float64 as TsonicFloat64,
} from "@tsonic/core/types.js";
import type { GoString } from "@gotots/runtime/string-value.js";

export type bool = TsonicBool;
export type int8 = TsonicInt8;
export type int16 = TsonicInt16;
export type int32 = TsonicInt32;
export type int64 = TsonicInt64;
export type uint8 = TsonicUint8;
export type uint16 = TsonicUint16;
export type uint32 = TsonicUint32;
export type uint64 = TsonicUint64;
export type gostring = GoString;
export type float32 = TsonicFloat32;
export type float64 = TsonicFloat64;
export type int = bigint;
export type uint = bigint;
export type uintptr = bigint;
