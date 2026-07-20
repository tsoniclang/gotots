// Hand port B7 — nil-tolerant pointer receiver: the ONE method that
// completes on nil takes the exceptional free-function lowering; every
// non-nil-tolerant sibling stays an ordinary method.
export function Set$Has<T>(s: Set<T> | undefined, key: T): boolean {
  if (s === undefined) {
    return false;
  }
  return s.M.has(key);
}
