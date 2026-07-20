// Hand port B3 — slice backing-store identity exception: &s1[0] == &s2[0]
// is a slot-identity observation, expressed once via the slice carrier.
export function Same<T>(s1: GoSlice<T>, s2: GoSlice<T>): boolean {
  if (len(s1) === len(s2)) {
    return len(s1) === 0 || sameSlot(s1, 0, s2, 0);
  }
  return false;
}
