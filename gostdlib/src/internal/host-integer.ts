function requireSafeInteger(value: number): void {
  if (!Number.isSafeInteger(value)) {
    throw new RangeError("Go integer cannot cross the JavaScript host boundary exactly");
  }
}

export function hostInteger(value: bigint): number {
  const result = Number(value);
  requireSafeInteger(result);
  if (BigInt(result) !== value) {
    throw new RangeError("Go integer cannot cross the JavaScript host boundary exactly");
  }
  return result;
}

export function integerFromHost(value: number): bigint {
  requireSafeInteger(value);
  return BigInt(value);
}

export function unsignedIntegerFromHost(value: number): bigint {
  requireSafeInteger(value);
  if (value < 0) {
    throw new RangeError("negative host integer cannot become an unsigned Go integer");
  }
  return BigInt(value);
}
