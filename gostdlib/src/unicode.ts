import type { int32 } from "@gotots/gostdlib/internal/scalars.js";

import {
  Range16,
  Range32,
  RangeTable,
  In,
  Is,
} from "./internal/portable/unicode/ranges.js";
import {
  Ideographic,
  Nd,
  No,
  Zs,
} from "./internal/portable/unicode/tables.js";

export {
  SimpleFold,
  ToLower,
  ToUpper,
} from "./internal/portable/unicode/case.js";
export {
  IsDigit,
  IsLetter,
  IsLower,
  IsNumber,
  IsPrint,
  IsSpace,
  IsUpper,
} from "./internal/portable/unicode/properties.js";
export { In, Is, Range16, Range32, RangeTable };

export const MaxASCII: int32 = 0x7f;

export const state: {
  Ideographic: RangeTable | undefined;
  Nd: RangeTable | undefined;
  No: RangeTable | undefined;
  Zs: RangeTable | undefined;
} = {
  Ideographic,
  Nd,
  No,
  Zs,
};
