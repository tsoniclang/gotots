export { Duration } from "./internal/portable/time/duration.js";
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

export const Millisecond = new Duration(1_000_000);
export const Minute = new Duration(60_000_000_000);
