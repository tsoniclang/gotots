// Hand port A4 — parametric generic control flow (both branches already
// evaluated at call entry, exactly as in Go).
export function IfElse<T>(b: boolean, whenTrue: T, whenFalse: T): T {
  if (b) {
    return whenTrue;
  }
  return whenFalse;
}
