import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  float64,
  int,
  int64,
  uint64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { FormatFloat } from "./float-format.js";
import { FormatInt, FormatUint } from "./integer.js";

export function AppendBool(target: RuntimeSlice<uint8>, value: bool): RuntimeSlice<uint8> {
  return appendASCII(target, value ? "true" : "false");
}

export function AppendFloat(
  target: RuntimeSlice<uint8>,
  value: float64,
  format: uint8,
  precision: int,
  bitSize: int,
): RuntimeSlice<uint8> {
  return appendASCII(target, FormatFloat(value, format, precision, bitSize));
}

export function AppendInt(
  target: RuntimeSlice<uint8>,
  value: int64,
  base: int,
): RuntimeSlice<uint8> {
  return appendASCII(target, FormatInt(value, base));
}

export function AppendUint(
  target: RuntimeSlice<uint8>,
  value: uint64,
  base: int,
): RuntimeSlice<uint8> {
  return appendASCII(target, FormatUint(value, base));
}

function appendASCII(target: RuntimeSlice<uint8>, value: string): RuntimeSlice<uint8> {
  return target.append(0, [...value].map((character) => character.charCodeAt(0)));
}
