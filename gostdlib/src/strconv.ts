import { ErrRange, ErrSyntax } from "./internal/portable/strconv/number-error.js";

export {
  AppendBool,
  AppendFloat,
  AppendInt,
  AppendUint,
} from "./internal/portable/strconv/append.js";
export { ParseFloat } from "./internal/portable/strconv/float.js";
export {
  Atoi,
  FormatInt,
  FormatUint,
  Itoa,
  ParseInt,
  ParseUint,
} from "./internal/portable/strconv/integer.js";
export {
  AppendQuote,
  Quote,
  QuoteRune,
  Unquote,
} from "./internal/portable/strconv/quote.js";

export const state = {
  ErrRange,
  ErrSyntax,
};
