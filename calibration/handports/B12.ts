// Hand port B12 — closure capture plus generic zero: the zero of T is
// demanded by `var value T`, supplied once by the instantiation.
export function Memoize<T>(create: (() => T) | undefined, zero: T): () => T {
  let value = zero;
  return () => {
    if (create !== undefined) {
      value = create();
      create = undefined;
    }
    return value;
  };
}
