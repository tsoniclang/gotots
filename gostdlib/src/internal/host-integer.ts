function requireInteger(value: number): void {
  if (!Number.isInteger(value)) {
    throw new RangeError("Go integer conversion requires a finite integer");
  }
}

export function hostInteger(value: bigint): number {
  const result = Number(value);
  requireInteger(result);
  if (BigInt(result) !== value) {
    throw new RangeError("Go integer cannot cross the numeric boundary without loss");
  }
  return result;
}

export function integerFromHost(value: number): bigint {
  requireInteger(value);
  return BigInt(value);
}

export function unsignedIntegerFromHost(value: number): bigint {
  requireInteger(value);
  if (value < 0) {
    throw new RangeError("negative host integer cannot become an unsigned Go integer");
  }
  return BigInt(value);
}
