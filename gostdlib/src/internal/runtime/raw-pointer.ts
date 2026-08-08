declare const providerRawPointerBrand: unique symbol;

export interface ProviderRawPointer {
  readonly [providerRawPointerBrand]: void;
}
