// Hand port C2 — specialization decision: comparable generic. Parametric
// TS cannot express `value != *new(T)` for every comparable binding, so
// the leaf instantiations specialize; each is emitted once.
export function OrElse$string(value: string, defaultValue: string): string {
  if (value !== "") {
    return value;
  }
  return defaultValue;
}
export function OrElse$Point(value: Point, defaultValue: Point): Point {
  if (!(value.X === 0n && value.Y === 0n)) {
    return value;
  }
  return defaultValue;
}
