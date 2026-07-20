// Hand port B11 — nil-vs-empty-slice exception: undefined is the nil
// slice; an allocated empty result is distinct. Otherwise ordinary.
export function Map<T, U>(slice: readonly T[] | undefined, f: (value: T) => U): U[] | undefined {
  if (slice === undefined) {
    return undefined;
  }
  const result = new Array<U>(slice.length);
  for (let i = 0; i < slice.length; i++) {
    result[i] = f(slice[i]);
  }
  return result;
}
