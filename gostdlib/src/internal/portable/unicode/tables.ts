import {
  ideographicLatinOffset,
  ideographicRanges16,
  ideographicRanges32,
  ndLatinOffset,
  ndRanges16,
  ndRanges32,
  noLatinOffset,
  noRanges16,
  noRanges32,
  zsLatinOffset,
  zsRanges16,
  zsRanges32,
} from "./classification-data.generated.js";
import { rangeTable } from "./ranges.js";

export const Ideographic = rangeTable(
  ideographicRanges16,
  ideographicRanges32,
  ideographicLatinOffset,
);
export const Nd = rangeTable(ndRanges16, ndRanges32, ndLatinOffset);
export const No = rangeTable(noRanges16, noRanges32, noLatinOffset);
export const Zs = rangeTable(zsRanges16, zsRanges32, zsLatinOffset);
