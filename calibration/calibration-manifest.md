# Calibration Manifest (rendered)

| ID | Verdict | Identity | Span | Go bytes | Baseline bytes | Source sha256 (12) |
|---|---|---|---|---:|---:|---|
| A1 | ordinary | github.com/microsoft/typescript-go/internal/scanner::method::regExpParser.getSpellingSuggestionForUnicodePropertyValue | internal/scanner/regexp.go[30323:30609) | 286 | 1006 | 1dddf9fef11a |
| A2 | ordinary | github.com/microsoft/typescript-go/internal/core::method::Tristate.IsTrue | internal/core/tristate.go[271:326) | 55 | 86 | 9056bb5488ff |
| A3 | ordinary | github.com/microsoft/typescript-go/internal/core::func::Identity | internal/core/core.go[15243:15284) | 41 | 175 | bdc7acbb68cc |
| A4 | ordinary | github.com/microsoft/typescript-go/internal/core::func::IfElse | internal/core/core.go[7163:7478) | 315 | 259 | cec017e5b6da |
| A5 | ordinary | github.com/microsoft/typescript-go/internal/core::method::TextRange.Len | internal/core/text.go[421:480) | 59 | 127 | 8de02b9c76f2 |
| B1 | exception | github.com/microsoft/typescript-go/internal/checker::func::hashWrite32 | internal/checker/checker.go[816336:816510) | 174 | 673 | 81ac35fd316c |
| B2 | exception | github.com/microsoft/typescript-go/internal/vfs/osvfs::method::osFS.ReadFile | internal/vfs/osvfs/os.go[2271:2402) | 131 | 556 | 4eba651864cb |
| B3 | exception | github.com/microsoft/typescript-go/internal/core::func::Same | internal/core/core.go[3458:3583) | 125 | 372 | da146fdfeb16 |
| B4 | exception | github.com/microsoft/typescript-go/internal/scanner::method::Scanner.scanString | internal/scanner/scanner.go[44707:46042) | 1335 | 3730 | 268789d2dc08 |
| B5 | exception | github.com/microsoft/typescript-go/internal/printer::method::NameGenerator.getScope | internal/printer/namegenerator.go[2591:2755) | 164 | 578 | 0afa79218e1d |
| B6 | exception | github.com/microsoft/typescript-go/internal/collections::method::OrderedMap.Set | internal/collections/ordered_map.go[1283:1502) | 219 | 2171 | 15e5523286d8 |
| B7 | exception | github.com/microsoft/typescript-go/internal/collections::method::Set.Has | internal/collections/set.go[299:398) | 99 | 837 | 621e0839f3d1 |
| B8 | exception | github.com/microsoft/typescript-go/internal/sourcemap::method::Generator.Base64DataURL | internal/sourcemap/generator.go[11248:11598) | 350 | 3592 | f089cbed2cc3 |
| B9 | exception | neutral/value-receiver-copy | calibration/neutral/value_receiver_copy.go[174:257) | 83 | 0 | a96ec35a8c80 |
| B10 | manual-required | github.com/microsoft/typescript-go/internal/checker::method::tracedTypeAdapter.Display | internal/checker/tracer.go[7836:8606) | 770 | 147 | 62ab02ae850a |
| B11 | exception | github.com/microsoft/typescript-go/internal/core::func::Map | internal/core/core.go[1436:1626) | 190 | 932 | ba690cea4f9e |
| B12 | exception | github.com/microsoft/typescript-go/internal/core::func::Memoize | internal/core/core.go[7001:7161) | 160 | 476 | e27c5e42f809 |
| B13 | exception | github.com/microsoft/typescript-go/internal/parser::func::init@internal/parser/jsdoc.go:13:6 | internal/parser/jsdoc.go[271:331) | 60 | 83 | 96bfb6b01644 |
| B14 | exception | neutral/typed-nil-interface | calibration/neutral/typed_nil_interface.go[294:481) | 187 | 0 | 61ce6571baf2 |
| B15 | exception | neutral/method-value-evaluation | calibration/neutral/method_value_evaluation.go[195:391) | 196 | 0 | e6baf87eb7fc |
| C1 | specialized | github.com/microsoft/typescript-go/internal/collections::method::OrderedMap.Set | internal/collections/ordered_map.go[1283:1502) | 219 | 2171 | 15e5523286d8 |
| C2 | specialized | github.com/microsoft/typescript-go/internal/core::func::OrElse | internal/core/core.go[7480:7833) | 353 | 270 | 5b445f4c8480 |
| C3 | specialized | github.com/microsoft/typescript-go/internal/core::func::Map | internal/core/core.go[1436:1626) | 190 | 932 | ba690cea4f9e |
| D1 | ordinary | github.com/microsoft/typescript-go/internal/diagnostics::func::keyToMessage | internal/diagnostics/diagnostics_generated.go[580490:898597) | 318107 | 380623 | 6236021c16cc |
| D2 | ordinary | github.com/microsoft/typescript-go/internal/compiler::method::Program.verifyCompilerOptions | internal/compiler/program.go[29383:52133) | 22750 | 75606 | 8780c2769ed5 |
| D3 | exception | github.com/microsoft/typescript-go/internal/debug::func::AssertNever | internal/debug/debug.go[474:898) | 424 | 59612 | 3395c4b6f054 |
| D4 | ordinary | github.com/microsoft/typescript-go/internal/checker::func::NewChecker | internal/checker/checker.go[45739:62451) | 16712 | 58207 | 37d382becbfb |
| D5 | ordinary | github.com/microsoft/typescript-go/internal/checker::method::Relater.structuredTypeRelatedToWorker | internal/checker/relater.go[143594:180331) | 36737 | 48798 | 69cd06be944f |
