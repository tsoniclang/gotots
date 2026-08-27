export {
  BinarySearch as SlicesBinarySearchKernel,
  BinarySearchFunc as SlicesBinarySearchFuncKernel,
  Compare as SlicesCompareKernel,
  CompareFunc as SlicesCompareFuncKernel,
  Contains as SlicesContainsKernel,
  ContainsFunc as SlicesContainsFuncKernel,
  Equal as SlicesEqualKernel,
  EqualFunc as SlicesEqualFuncKernel,
  Index as SlicesIndexKernel,
  IndexFunc as SlicesIndexFuncKernel,
} from "../portable/slices/read.js";
export {
  Clip as SlicesClipKernel,
  Clone as SlicesCloneKernel,
  Compact as SlicesCompactKernel,
  CompactFunc as SlicesCompactFuncKernel,
  Concat as SlicesConcatKernel,
  Delete as SlicesDeleteKernel,
  DeleteFunc as SlicesDeleteFuncKernel,
  Grow as SlicesGrowKernel,
  Repeat as SlicesRepeatKernel,
  Reverse as SlicesReverseKernel,
} from "../portable/slices/transform.js";
export {
  Insert as SlicesInsertKernel,
  Replace as SlicesReplaceKernel,
} from "../portable/slices/splice.js";
export {
  Sort as SlicesSortKernel,
  SortFunc as SlicesSortFuncKernel,
  SortStableFunc as SlicesSortStableFuncKernel,
} from "../portable/slices/sort.js";
export {
  AppendSeq as SlicesAppendSeqKernel,
  Collect as SlicesCollectKernel,
  Sorted as SlicesSortedKernel,
  SortedFunc as SlicesSortedFuncKernel,
  Values as SlicesValuesKernel,
} from "../portable/slices/sequence.js";
