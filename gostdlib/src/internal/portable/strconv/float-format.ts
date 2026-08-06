import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  float64,
  gostring,
  int64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "../../host-integer.js";

type Decimal = {
  readonly digits: bigint;
  readonly scale: number;
};

type FloatParts = {
  readonly value: number;
  readonly negative: boolean;
  readonly mantissa: bigint;
  readonly exponent2: number;
  readonly mantissaBits: number;
  readonly binaryExponent: number;
};

const bitsBuffer = new ArrayBuffer(8);
const bitsView = new DataView(bitsBuffer);

export function FormatFloat(
  input: float64,
  formatCode: uint8,
  requestedPrecision: int64,
  bitSize: int64,
): gostring {
  const parts = decompose(input, bitSize);
  const value = parts.value;
  if (Number.isNaN(value)) {
    return "NaN";
  }
  if (value === Number.POSITIVE_INFINITY) {
    return "+Inf";
  }
  if (value === Number.NEGATIVE_INFINITY) {
    return "-Inf";
  }

  const format = String.fromCharCode(formatCode);
  const precision = hostInteger(requestedPrecision);
  const sign = parts.negative ? "-" : "";
  if (format === "b") {
    return `${sign}${parts.mantissa}p${signedExponent(parts.exponent2, false)}`;
  }
  if (format === "x" || format === "X") {
    return `${sign}${formatHex(parts, precision, format)}`;
  }
  if (!["e", "E", "f", "g", "G"].includes(format)) {
    return `%${format}`;
  }

  const shortest = precision < 0;
  const decimal = shortest
    ? shortestDecimal(Math.abs(value), bitSize)
    : exactDecimal(parts);
  switch (format) {
  case "e":
  case "E": {
    const digits = shortest ? digitCount(decimal) : precision + 1;
    return sign + renderExponential(roundSignificant(decimal, digits), digits - 1, format);
  }
  case "f": {
    const fractionalDigits = shortest
      ? Math.max(digitCount(decimal) - decimalPoint(decimal), 0)
      : precision;
    return sign + renderFixed(decimal, fractionalDigits);
  }
  case "g":
  case "G": {
    const significantDigits = shortest ? digitCount(decimal) : Math.max(precision, 1);
    const rounded = shortest ? decimal : roundSignificant(decimal, significantDigits);
    const exponent = decimalPoint(rounded) - 1;
    const exponentLimit = shortest ? 6 : significantDigits;
    if (exponent < -4 || exponent >= exponentLimit) {
      const marker = format === "g" ? "e" : "E";
      return sign + renderExponential(rounded, Math.max(digitCount(rounded) - 1, 0), marker);
    }
    return sign + renderMinimalFixed(rounded);
  }
  }
  return `%${format}`;
}

function decompose(input: number, bitSize: int64): FloatParts {
  if (bitSize === 32n) {
    const value = Math.fround(input);
    bitsView.setFloat32(0, value, false);
    return partsFromBits(BigInt(bitsView.getUint32(0, false)), value, 23, 8, 127);
  }
  if (bitSize === 64n) {
    bitsView.setFloat64(0, input, false);
    return partsFromBits(bitsView.getBigUint64(0, false), input, 52, 11, 1023);
  }
  GoPanic.raiseRuntime("strconv: illegal AppendFloat/FormatFloat bitSize");
}

function partsFromBits(
  bits: bigint,
  value: number,
  mantissaBits: number,
  exponentBits: number,
  bias: number,
): FloatParts {
  const fractionMask = (1n << BigInt(mantissaBits)) - 1n;
  const exponentMask = (1n << BigInt(exponentBits)) - 1n;
  const exponentField = Number((bits >> BigInt(mantissaBits)) & exponentMask);
  const negative = ((bits >> BigInt(mantissaBits + exponentBits)) & 1n) !== 0n;
  const fraction = bits & fractionMask;
  const denormal = exponentField === 0;
  const mantissa = denormal ? fraction : fraction | (1n << BigInt(mantissaBits));
  const binaryExponent = (denormal ? 1 : exponentField) - bias;
  return {
    value,
    negative,
    mantissa,
    exponent2: binaryExponent - mantissaBits,
    mantissaBits,
    binaryExponent,
  };
}

function exactDecimal(parts: FloatParts): Decimal {
  if (parts.mantissa === 0n) {
    return { digits: 0n, scale: 0 };
  }
  if (parts.exponent2 >= 0) {
    return normalize({
      digits: parts.mantissa << BigInt(parts.exponent2),
      scale: 0,
    });
  }
  const scale = -parts.exponent2;
  return normalize({
    digits: parts.mantissa * (5n ** BigInt(scale)),
    scale,
  });
}

function shortestDecimal(value: number, bitSize: int64): Decimal {
  if (value === 0) {
    return { digits: 0n, scale: 0 };
  }
  if (bitSize === 64n) {
    return parseDecimal(value.toString());
  }
  for (let precision = 1; precision <= 9; precision += 1) {
    const candidate = value.toPrecision(precision);
    if (Object.is(Math.fround(Number(candidate)), value)) {
      return parseDecimal(candidate);
    }
  }
  return parseDecimal(value.toPrecision(9));
}

function parseDecimal(source: string): Decimal {
  const [coefficient = "0", exponentText = "0"] = source.toLowerCase().split("e");
  const [whole = "0", fraction = ""] = coefficient.split(".");
  let digits = `${whole}${fraction}`;
  let point = whole.length + Number(exponentText);
  while (digits.startsWith("0") && digits.length > 1) {
    digits = digits.slice(1);
    point -= 1;
  }
  while (digits.endsWith("0") && digits.length > 1) {
    digits = digits.slice(0, -1);
  }
  if (/^0+$/.test(digits)) {
    return { digits: 0n, scale: 0 };
  }
  return { digits: BigInt(digits), scale: digits.length - point };
}

function roundSignificant(decimal: Decimal, precision: number): Decimal {
  if (decimal.digits === 0n || digitCount(decimal) <= precision) {
    return decimal;
  }
  const removed = digitCount(decimal) - precision;
  const divisor = 10n ** BigInt(removed);
  return normalize({
    digits: roundDivide(decimal.digits, divisor),
    scale: decimal.scale - removed,
  });
}

function renderExponential(decimal: Decimal, precision: number, marker: string): string {
  const text = decimal.digits === 0n ? "0" : decimal.digits.toString();
  const fraction = `${text.slice(1)}${"0".repeat(precision)}`.slice(0, precision);
  const exponent = decimal.digits === 0n ? 0 : decimalPoint(decimal) - 1;
  return `${text[0] ?? "0"}${precision > 0 ? `.${fraction}` : ""}${marker}${signedExponent(exponent, true)}`;
}

function renderFixed(decimal: Decimal, precision: number): string {
  let scaled: bigint;
  if (precision >= decimal.scale) {
    scaled = decimal.digits * (10n ** BigInt(precision - decimal.scale));
  } else {
    scaled = roundDivide(decimal.digits, 10n ** BigInt(decimal.scale - precision));
  }
  if (precision === 0) {
    return scaled.toString();
  }
  const text = scaled.toString().padStart(precision + 1, "0");
  return `${text.slice(0, -precision)}.${text.slice(-precision)}`;
}

function renderMinimalFixed(decimal: Decimal): string {
  if (decimal.digits === 0n) {
    return "0";
  }
  const text = decimal.digits.toString();
  if (decimal.scale <= 0) {
    return text + "0".repeat(-decimal.scale);
  }
  if (decimal.scale >= text.length) {
    return `0.${"0".repeat(decimal.scale - text.length)}${text}`;
  }
  const split = text.length - decimal.scale;
  return `${text.slice(0, split)}.${text.slice(split)}`;
}

function formatHex(parts: FloatParts, precision: number, format: string): string {
  if (parts.mantissa === 0n) {
    return `0${format}0${precision > 0 ? `.${"0".repeat(precision)}` : ""}${format === "x" ? "p" : "P"}+00`;
  }
  let exponent = parts.binaryExponent;
  let mantissa = parts.mantissa;
  const currentTopBit = bitLength(mantissa) - 1;
  const shift = parts.mantissaBits - currentTopBit;
  if (shift > 0) {
    mantissa <<= BigInt(shift);
    exponent -= shift;
  }

  const naturalDigits = Math.ceil(parts.mantissaBits / 4);
  let fractionText: string;
  if (precision < 0) {
    const fraction = (mantissa - (1n << BigInt(parts.mantissaBits)))
      << BigInt(naturalDigits * 4 - parts.mantissaBits);
    fractionText = fraction.toString(16).padStart(naturalDigits, "0").replace(/0+$/, "");
  } else if (precision * 4 >= parts.mantissaBits) {
    const fraction = (mantissa - (1n << BigInt(parts.mantissaBits)))
      << BigInt(precision * 4 - parts.mantissaBits);
    fractionText = fraction.toString(16).padStart(precision, "0");
  } else {
    const keptBits = precision * 4;
    const removedBits = parts.mantissaBits - keptBits;
    const divisor = 1n << BigInt(removedBits);
    let rounded = roundDivide(mantissa, divisor);
    const leading = 1n << BigInt(keptBits);
    if (rounded >= leading * 2n) {
      rounded >>= 1n;
      exponent += 1;
    }
    fractionText = (rounded - leading).toString(16).padStart(precision, "0");
  }
  if (format === "X") {
    fractionText = fractionText.toUpperCase();
  }
  const exponentMarker = format === "x" ? "p" : "P";
  return `0${format}1${fractionText.length > 0 ? `.${fractionText}` : ""}${exponentMarker}${signedExponent(exponent, true)}`;
}

function normalize(decimal: Decimal): Decimal {
  let digits = decimal.digits;
  let scale = decimal.scale;
  while (digits !== 0n && digits % 10n === 0n) {
    digits /= 10n;
    scale -= 1;
  }
  return digits === 0n ? { digits: 0n, scale: 0 } : { digits, scale };
}

function roundDivide(value: bigint, divisor: bigint): bigint {
  const quotient = value / divisor;
  const remainder = value % divisor;
  const doubled = remainder * 2n;
  return doubled > divisor || (doubled === divisor && quotient % 2n !== 0n)
    ? quotient + 1n
    : quotient;
}

function digitCount(decimal: Decimal): number {
  return decimal.digits === 0n ? 0 : decimal.digits.toString().length;
}

function decimalPoint(decimal: Decimal): number {
  return digitCount(decimal) - decimal.scale;
}

function bitLength(value: bigint): number {
  return value === 0n ? 0 : value.toString(2).length;
}

function signedExponent(exponent: number, padded: boolean): string {
  const magnitude = Math.abs(exponent).toString().padStart(padded ? 2 : 1, "0");
  return `${exponent < 0 ? "-" : "+"}${magnitude}`;
}
