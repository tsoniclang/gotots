import type { int32 } from "@gotots/runtime/scalars.js";

import {
  Range16,
  Range32,
  RangeTable,
  Is,
  rangeTable,
} from "./internal/portable/unicode/ranges.js";

export { ToLower, ToUpper } from "./internal/portable/unicode/case.js";
export {
  IsDigit,
  IsLower,
  IsSpace,
  IsUpper,
} from "./internal/portable/unicode/properties.js";
export { Is, Range16, Range32, RangeTable };

export const MaxASCII: int32 = 0x7f;

export const state: {
  Zs: RangeTable | undefined;
} = {
  Zs: rangeTable(
    [
      [0x0020, 0x00a0, 128],
      [0x1680, 0x2000, 2432],
      [0x2001, 0x200a, 1],
      [0x202f, 0x205f, 48],
      [0x3000, 0x3000, 1],
    ],
    [],
    1,
  ),
};
