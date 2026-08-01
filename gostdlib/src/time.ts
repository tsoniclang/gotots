export { Duration } from "./internal/portable/time/duration.js";
export { ParseError } from "./internal/portable/time/parse-error.js";
export { ParseDuration } from "./internal/portable/time/parse-duration.js";
export { Parse } from "./internal/portable/time/parse-time.js";
export {
  Now,
  Since,
  Time,
  Unix,
  UnixMilli,
  Until,
} from "./internal/portable/time/time.js";
export {
  After,
  AfterFunc,
  NewTicker,
  NewTimer,
  Ticker,
  Timer,
} from "./internal/portable/time/timer.js";

import { Duration } from "./internal/portable/time/duration.js";

export const Nanosecond = new Duration(1);
export const Microsecond = new Duration(1_000);
export const Millisecond = new Duration(1_000_000);
export const Second = new Duration(1_000_000_000);
export const Minute = new Duration(60_000_000_000);
export const Hour = new Duration(3_600_000_000_000);
