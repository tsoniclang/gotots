import { ErrRange } from "./internal/portable/strconv/number-error.js";

export { ParseFloat } from "./internal/portable/strconv/float.js";
export {
  Atoi,
  FormatInt,
  FormatUint,
  Itoa,
  ParseInt,
  ParseUint,
} from "./internal/portable/strconv/integer.js";

export const state = {
  ErrRange,
};
