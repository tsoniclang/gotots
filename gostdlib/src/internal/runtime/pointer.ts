export interface ProviderPointer<T> {
  value: T;
}

export function providerPointer<T>(value: T): ProviderPointer<T> {
  return { value };
}
