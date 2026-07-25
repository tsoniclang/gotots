// Code generated from schema/tsgo by go generate. DO NOT EDIT.

package tsgo

const (
	SyntaxKindUnknown                                      SyntaxKind = 0
	SyntaxKindEndOfFile                                    SyntaxKind = 1
	SyntaxKindSingleLineCommentTrivia                      SyntaxKind = 2
	SyntaxKindMultiLineCommentTrivia                       SyntaxKind = 3
	SyntaxKindNewLineTrivia                                SyntaxKind = 4
	SyntaxKindWhitespaceTrivia                             SyntaxKind = 5
	SyntaxKindConflictMarkerTrivia                         SyntaxKind = 6
	SyntaxKindNonTextFileMarkerTrivia                      SyntaxKind = 7
	SyntaxKindNumericLiteral                               SyntaxKind = 8
	SyntaxKindBigIntLiteral                                SyntaxKind = 9
	SyntaxKindStringLiteral                                SyntaxKind = 10
	SyntaxKindJsxText                                      SyntaxKind = 11
	SyntaxKindJsxTextAllWhiteSpaces                        SyntaxKind = 12
	SyntaxKindRegularExpressionLiteral                     SyntaxKind = 13
	SyntaxKindNoSubstitutionTemplateLiteral                SyntaxKind = 14
	SyntaxKindTemplateHead                                 SyntaxKind = 15
	SyntaxKindTemplateMiddle                               SyntaxKind = 16
	SyntaxKindTemplateTail                                 SyntaxKind = 17
	SyntaxKindOpenBraceToken                               SyntaxKind = 18
	SyntaxKindCloseBraceToken                              SyntaxKind = 19
	SyntaxKindOpenParenToken                               SyntaxKind = 20
	SyntaxKindCloseParenToken                              SyntaxKind = 21
	SyntaxKindOpenBracketToken                             SyntaxKind = 22
	SyntaxKindCloseBracketToken                            SyntaxKind = 23
	SyntaxKindDotToken                                     SyntaxKind = 24
	SyntaxKindDotDotDotToken                               SyntaxKind = 25
	SyntaxKindSemicolonToken                               SyntaxKind = 26
	SyntaxKindCommaToken                                   SyntaxKind = 27
	SyntaxKindQuestionDotToken                             SyntaxKind = 28
	SyntaxKindLessThanToken                                SyntaxKind = 29
	SyntaxKindLessThanSlashToken                           SyntaxKind = 30
	SyntaxKindGreaterThanToken                             SyntaxKind = 31
	SyntaxKindLessThanEqualsToken                          SyntaxKind = 32
	SyntaxKindGreaterThanEqualsToken                       SyntaxKind = 33
	SyntaxKindEqualsEqualsToken                            SyntaxKind = 34
	SyntaxKindExclamationEqualsToken                       SyntaxKind = 35
	SyntaxKindEqualsEqualsEqualsToken                      SyntaxKind = 36
	SyntaxKindExclamationEqualsEqualsToken                 SyntaxKind = 37
	SyntaxKindEqualsGreaterThanToken                       SyntaxKind = 38
	SyntaxKindPlusToken                                    SyntaxKind = 39
	SyntaxKindMinusToken                                   SyntaxKind = 40
	SyntaxKindAsteriskToken                                SyntaxKind = 41
	SyntaxKindAsteriskAsteriskToken                        SyntaxKind = 42
	SyntaxKindSlashToken                                   SyntaxKind = 43
	SyntaxKindPercentToken                                 SyntaxKind = 44
	SyntaxKindPlusPlusToken                                SyntaxKind = 45
	SyntaxKindMinusMinusToken                              SyntaxKind = 46
	SyntaxKindLessThanLessThanToken                        SyntaxKind = 47
	SyntaxKindGreaterThanGreaterThanToken                  SyntaxKind = 48
	SyntaxKindGreaterThanGreaterThanGreaterThanToken       SyntaxKind = 49
	SyntaxKindAmpersandToken                               SyntaxKind = 50
	SyntaxKindBarToken                                     SyntaxKind = 51
	SyntaxKindCaretToken                                   SyntaxKind = 52
	SyntaxKindExclamationToken                             SyntaxKind = 53
	SyntaxKindTildeToken                                   SyntaxKind = 54
	SyntaxKindAmpersandAmpersandToken                      SyntaxKind = 55
	SyntaxKindBarBarToken                                  SyntaxKind = 56
	SyntaxKindQuestionToken                                SyntaxKind = 57
	SyntaxKindColonToken                                   SyntaxKind = 58
	SyntaxKindAtToken                                      SyntaxKind = 59
	SyntaxKindQuestionQuestionToken                        SyntaxKind = 60
	SyntaxKindBacktickToken                                SyntaxKind = 61
	SyntaxKindHashToken                                    SyntaxKind = 62
	SyntaxKindEqualsToken                                  SyntaxKind = 63
	SyntaxKindPlusEqualsToken                              SyntaxKind = 64
	SyntaxKindMinusEqualsToken                             SyntaxKind = 65
	SyntaxKindAsteriskEqualsToken                          SyntaxKind = 66
	SyntaxKindAsteriskAsteriskEqualsToken                  SyntaxKind = 67
	SyntaxKindSlashEqualsToken                             SyntaxKind = 68
	SyntaxKindPercentEqualsToken                           SyntaxKind = 69
	SyntaxKindLessThanLessThanEqualsToken                  SyntaxKind = 70
	SyntaxKindGreaterThanGreaterThanEqualsToken            SyntaxKind = 71
	SyntaxKindGreaterThanGreaterThanGreaterThanEqualsToken SyntaxKind = 72
	SyntaxKindAmpersandEqualsToken                         SyntaxKind = 73
	SyntaxKindBarEqualsToken                               SyntaxKind = 74
	SyntaxKindBarBarEqualsToken                            SyntaxKind = 75
	SyntaxKindAmpersandAmpersandEqualsToken                SyntaxKind = 76
	SyntaxKindQuestionQuestionEqualsToken                  SyntaxKind = 77
	SyntaxKindCaretEqualsToken                             SyntaxKind = 78
	SyntaxKindIdentifier                                   SyntaxKind = 79
	SyntaxKindPrivateIdentifier                            SyntaxKind = 80
	SyntaxKindJSDocCommentTextToken                        SyntaxKind = 81
	SyntaxKindBreakKeyword                                 SyntaxKind = 82
	SyntaxKindCaseKeyword                                  SyntaxKind = 83
	SyntaxKindCatchKeyword                                 SyntaxKind = 84
	SyntaxKindClassKeyword                                 SyntaxKind = 85
	SyntaxKindConstKeyword                                 SyntaxKind = 86
	SyntaxKindContinueKeyword                              SyntaxKind = 87
	SyntaxKindDebuggerKeyword                              SyntaxKind = 88
	SyntaxKindDefaultKeyword                               SyntaxKind = 89
	SyntaxKindDeleteKeyword                                SyntaxKind = 90
	SyntaxKindDoKeyword                                    SyntaxKind = 91
	SyntaxKindElseKeyword                                  SyntaxKind = 92
	SyntaxKindEnumKeyword                                  SyntaxKind = 93
	SyntaxKindExportKeyword                                SyntaxKind = 94
	SyntaxKindExtendsKeyword                               SyntaxKind = 95
	SyntaxKindFalseKeyword                                 SyntaxKind = 96
	SyntaxKindFinallyKeyword                               SyntaxKind = 97
	SyntaxKindForKeyword                                   SyntaxKind = 98
	SyntaxKindFunctionKeyword                              SyntaxKind = 99
	SyntaxKindIfKeyword                                    SyntaxKind = 100
	SyntaxKindImportKeyword                                SyntaxKind = 101
	SyntaxKindInKeyword                                    SyntaxKind = 102
	SyntaxKindInstanceOfKeyword                            SyntaxKind = 103
	SyntaxKindNewKeyword                                   SyntaxKind = 104
	SyntaxKindNullKeyword                                  SyntaxKind = 105
	SyntaxKindReturnKeyword                                SyntaxKind = 106
	SyntaxKindSuperKeyword                                 SyntaxKind = 107
	SyntaxKindSwitchKeyword                                SyntaxKind = 108
	SyntaxKindThisKeyword                                  SyntaxKind = 109
	SyntaxKindThrowKeyword                                 SyntaxKind = 110
	SyntaxKindTrueKeyword                                  SyntaxKind = 111
	SyntaxKindTryKeyword                                   SyntaxKind = 112
	SyntaxKindTypeOfKeyword                                SyntaxKind = 113
	SyntaxKindVarKeyword                                   SyntaxKind = 114
	SyntaxKindVoidKeyword                                  SyntaxKind = 115
	SyntaxKindWhileKeyword                                 SyntaxKind = 116
	SyntaxKindWithKeyword                                  SyntaxKind = 117
	SyntaxKindImplementsKeyword                            SyntaxKind = 118
	SyntaxKindInterfaceKeyword                             SyntaxKind = 119
	SyntaxKindLetKeyword                                   SyntaxKind = 120
	SyntaxKindPackageKeyword                               SyntaxKind = 121
	SyntaxKindPrivateKeyword                               SyntaxKind = 122
	SyntaxKindProtectedKeyword                             SyntaxKind = 123
	SyntaxKindPublicKeyword                                SyntaxKind = 124
	SyntaxKindStaticKeyword                                SyntaxKind = 125
	SyntaxKindYieldKeyword                                 SyntaxKind = 126
	SyntaxKindAbstractKeyword                              SyntaxKind = 127
	SyntaxKindAccessorKeyword                              SyntaxKind = 128
	SyntaxKindAsKeyword                                    SyntaxKind = 129
	SyntaxKindAssertsKeyword                               SyntaxKind = 130
	SyntaxKindAssertKeyword                                SyntaxKind = 131
	SyntaxKindAnyKeyword                                   SyntaxKind = 132
	SyntaxKindAsyncKeyword                                 SyntaxKind = 133
	SyntaxKindAwaitKeyword                                 SyntaxKind = 134
	SyntaxKindBooleanKeyword                               SyntaxKind = 135
	SyntaxKindConstructorKeyword                           SyntaxKind = 136
	SyntaxKindDeclareKeyword                               SyntaxKind = 137
	SyntaxKindGetKeyword                                   SyntaxKind = 138
	SyntaxKindImmediateKeyword                             SyntaxKind = 139
	SyntaxKindInferKeyword                                 SyntaxKind = 140
	SyntaxKindIntrinsicKeyword                             SyntaxKind = 141
	SyntaxKindIsKeyword                                    SyntaxKind = 142
	SyntaxKindKeyOfKeyword                                 SyntaxKind = 143
	SyntaxKindModuleKeyword                                SyntaxKind = 144
	SyntaxKindNamespaceKeyword                             SyntaxKind = 145
	SyntaxKindNeverKeyword                                 SyntaxKind = 146
	SyntaxKindOutKeyword                                   SyntaxKind = 147
	SyntaxKindReadonlyKeyword                              SyntaxKind = 148
	SyntaxKindRequireKeyword                               SyntaxKind = 149
	SyntaxKindNumberKeyword                                SyntaxKind = 150
	SyntaxKindObjectKeyword                                SyntaxKind = 151
	SyntaxKindSatisfiesKeyword                             SyntaxKind = 152
	SyntaxKindSetKeyword                                   SyntaxKind = 153
	SyntaxKindStringKeyword                                SyntaxKind = 154
	SyntaxKindSymbolKeyword                                SyntaxKind = 155
	SyntaxKindTypeKeyword                                  SyntaxKind = 156
	SyntaxKindUndefinedKeyword                             SyntaxKind = 157
	SyntaxKindUniqueKeyword                                SyntaxKind = 158
	SyntaxKindUnknownKeyword                               SyntaxKind = 159
	SyntaxKindUsingKeyword                                 SyntaxKind = 160
	SyntaxKindFromKeyword                                  SyntaxKind = 161
	SyntaxKindGlobalKeyword                                SyntaxKind = 162
	SyntaxKindBigIntKeyword                                SyntaxKind = 163
	SyntaxKindOverrideKeyword                              SyntaxKind = 164
	SyntaxKindOfKeyword                                    SyntaxKind = 165
	SyntaxKindDeferKeyword                                 SyntaxKind = 166
	SyntaxKindQualifiedName                                SyntaxKind = 167
	SyntaxKindComputedPropertyName                         SyntaxKind = 168
	SyntaxKindTypeParameter                                SyntaxKind = 169
	SyntaxKindParameter                                    SyntaxKind = 170
	SyntaxKindDecorator                                    SyntaxKind = 171
	SyntaxKindPropertySignature                            SyntaxKind = 172
	SyntaxKindPropertyDeclaration                          SyntaxKind = 173
	SyntaxKindMethodSignature                              SyntaxKind = 174
	SyntaxKindMethodDeclaration                            SyntaxKind = 175
	SyntaxKindClassStaticBlockDeclaration                  SyntaxKind = 176
	SyntaxKindConstructor                                  SyntaxKind = 177
	SyntaxKindGetAccessor                                  SyntaxKind = 178
	SyntaxKindSetAccessor                                  SyntaxKind = 179
	SyntaxKindCallSignature                                SyntaxKind = 180
	SyntaxKindConstructSignature                           SyntaxKind = 181
	SyntaxKindIndexSignature                               SyntaxKind = 182
	SyntaxKindTypePredicate                                SyntaxKind = 183
	SyntaxKindTypeReference                                SyntaxKind = 184
	SyntaxKindFunctionType                                 SyntaxKind = 185
	SyntaxKindConstructorType                              SyntaxKind = 186
	SyntaxKindTypeQuery                                    SyntaxKind = 187
	SyntaxKindTypeLiteral                                  SyntaxKind = 188
	SyntaxKindArrayType                                    SyntaxKind = 189
	SyntaxKindTupleType                                    SyntaxKind = 190
	SyntaxKindOptionalType                                 SyntaxKind = 191
	SyntaxKindRestType                                     SyntaxKind = 192
	SyntaxKindUnionType                                    SyntaxKind = 193
	SyntaxKindIntersectionType                             SyntaxKind = 194
	SyntaxKindConditionalType                              SyntaxKind = 195
	SyntaxKindInferType                                    SyntaxKind = 196
	SyntaxKindParenthesizedType                            SyntaxKind = 197
	SyntaxKindThisType                                     SyntaxKind = 198
	SyntaxKindTypeOperator                                 SyntaxKind = 199
	SyntaxKindIndexedAccessType                            SyntaxKind = 200
	SyntaxKindMappedType                                   SyntaxKind = 201
	SyntaxKindLiteralType                                  SyntaxKind = 202
	SyntaxKindNamedTupleMember                             SyntaxKind = 203
	SyntaxKindTemplateLiteralType                          SyntaxKind = 204
	SyntaxKindTemplateLiteralTypeSpan                      SyntaxKind = 205
	SyntaxKindImportType                                   SyntaxKind = 206
	SyntaxKindObjectBindingPattern                         SyntaxKind = 207
	SyntaxKindArrayBindingPattern                          SyntaxKind = 208
	SyntaxKindBindingElement                               SyntaxKind = 209
	SyntaxKindArrayLiteralExpression                       SyntaxKind = 210
	SyntaxKindObjectLiteralExpression                      SyntaxKind = 211
	SyntaxKindPropertyAccessExpression                     SyntaxKind = 212
	SyntaxKindElementAccessExpression                      SyntaxKind = 213
	SyntaxKindCallExpression                               SyntaxKind = 214
	SyntaxKindNewExpression                                SyntaxKind = 215
	SyntaxKindTaggedTemplateExpression                     SyntaxKind = 216
	SyntaxKindTypeAssertionExpression                      SyntaxKind = 217
	SyntaxKindParenthesizedExpression                      SyntaxKind = 218
	SyntaxKindFunctionExpression                           SyntaxKind = 219
	SyntaxKindArrowFunction                                SyntaxKind = 220
	SyntaxKindDeleteExpression                             SyntaxKind = 221
	SyntaxKindTypeOfExpression                             SyntaxKind = 222
	SyntaxKindVoidExpression                               SyntaxKind = 223
	SyntaxKindAwaitExpression                              SyntaxKind = 224
	SyntaxKindPrefixUnaryExpression                        SyntaxKind = 225
	SyntaxKindPostfixUnaryExpression                       SyntaxKind = 226
	SyntaxKindBinaryExpression                             SyntaxKind = 227
	SyntaxKindConditionalExpression                        SyntaxKind = 228
	SyntaxKindTemplateExpression                           SyntaxKind = 229
	SyntaxKindYieldExpression                              SyntaxKind = 230
	SyntaxKindSpreadElement                                SyntaxKind = 231
	SyntaxKindClassExpression                              SyntaxKind = 232
	SyntaxKindOmittedExpression                            SyntaxKind = 233
	SyntaxKindExpressionWithTypeArguments                  SyntaxKind = 234
	SyntaxKindAsExpression                                 SyntaxKind = 235
	SyntaxKindNonNullExpression                            SyntaxKind = 236
	SyntaxKindMetaProperty                                 SyntaxKind = 237
	SyntaxKindSyntheticExpression                          SyntaxKind = 238
	SyntaxKindSatisfiesExpression                          SyntaxKind = 239
	SyntaxKindTemplateSpan                                 SyntaxKind = 240
	SyntaxKindSemicolonClassElement                        SyntaxKind = 241
	SyntaxKindBlock                                        SyntaxKind = 242
	SyntaxKindEmptyStatement                               SyntaxKind = 243
	SyntaxKindVariableStatement                            SyntaxKind = 244
	SyntaxKindExpressionStatement                          SyntaxKind = 245
	SyntaxKindIfStatement                                  SyntaxKind = 246
	SyntaxKindDoStatement                                  SyntaxKind = 247
	SyntaxKindWhileStatement                               SyntaxKind = 248
	SyntaxKindForStatement                                 SyntaxKind = 249
	SyntaxKindForInStatement                               SyntaxKind = 250
	SyntaxKindForOfStatement                               SyntaxKind = 251
	SyntaxKindContinueStatement                            SyntaxKind = 252
	SyntaxKindBreakStatement                               SyntaxKind = 253
	SyntaxKindReturnStatement                              SyntaxKind = 254
	SyntaxKindWithStatement                                SyntaxKind = 255
	SyntaxKindSwitchStatement                              SyntaxKind = 256
	SyntaxKindLabeledStatement                             SyntaxKind = 257
	SyntaxKindThrowStatement                               SyntaxKind = 258
	SyntaxKindTryStatement                                 SyntaxKind = 259
	SyntaxKindDebuggerStatement                            SyntaxKind = 260
	SyntaxKindVariableDeclaration                          SyntaxKind = 261
	SyntaxKindVariableDeclarationList                      SyntaxKind = 262
	SyntaxKindFunctionDeclaration                          SyntaxKind = 263
	SyntaxKindClassDeclaration                             SyntaxKind = 264
	SyntaxKindInterfaceDeclaration                         SyntaxKind = 265
	SyntaxKindTypeAliasDeclaration                         SyntaxKind = 266
	SyntaxKindEnumDeclaration                              SyntaxKind = 267
	SyntaxKindModuleDeclaration                            SyntaxKind = 268
	SyntaxKindModuleBlock                                  SyntaxKind = 269
	SyntaxKindCaseBlock                                    SyntaxKind = 270
	SyntaxKindNamespaceExportDeclaration                   SyntaxKind = 271
	SyntaxKindImportEqualsDeclaration                      SyntaxKind = 272
	SyntaxKindImportDeclaration                            SyntaxKind = 273
	SyntaxKindImportClause                                 SyntaxKind = 274
	SyntaxKindNamespaceImport                              SyntaxKind = 275
	SyntaxKindNamedImports                                 SyntaxKind = 276
	SyntaxKindImportSpecifier                              SyntaxKind = 277
	SyntaxKindExportAssignment                             SyntaxKind = 278
	SyntaxKindExportDeclaration                            SyntaxKind = 279
	SyntaxKindNamedExports                                 SyntaxKind = 280
	SyntaxKindNamespaceExport                              SyntaxKind = 281
	SyntaxKindExportSpecifier                              SyntaxKind = 282
	SyntaxKindMissingDeclaration                           SyntaxKind = 283
	SyntaxKindExternalModuleReference                      SyntaxKind = 284
	SyntaxKindJsxElement                                   SyntaxKind = 285
	SyntaxKindJsxSelfClosingElement                        SyntaxKind = 286
	SyntaxKindJsxOpeningElement                            SyntaxKind = 287
	SyntaxKindJsxClosingElement                            SyntaxKind = 288
	SyntaxKindJsxFragment                                  SyntaxKind = 289
	SyntaxKindJsxOpeningFragment                           SyntaxKind = 290
	SyntaxKindJsxClosingFragment                           SyntaxKind = 291
	SyntaxKindJsxAttribute                                 SyntaxKind = 292
	SyntaxKindJsxAttributes                                SyntaxKind = 293
	SyntaxKindJsxSpreadAttribute                           SyntaxKind = 294
	SyntaxKindJsxExpression                                SyntaxKind = 295
	SyntaxKindJsxNamespacedName                            SyntaxKind = 296
	SyntaxKindCaseClause                                   SyntaxKind = 297
	SyntaxKindDefaultClause                                SyntaxKind = 298
	SyntaxKindHeritageClause                               SyntaxKind = 299
	SyntaxKindCatchClause                                  SyntaxKind = 300
	SyntaxKindImportAttributes                             SyntaxKind = 301
	SyntaxKindImportAttribute                              SyntaxKind = 302
	SyntaxKindPropertyAssignment                           SyntaxKind = 303
	SyntaxKindShorthandPropertyAssignment                  SyntaxKind = 304
	SyntaxKindSpreadAssignment                             SyntaxKind = 305
	SyntaxKindEnumMember                                   SyntaxKind = 306
	SyntaxKindSourceFile                                   SyntaxKind = 307
	SyntaxKindJSDocTypeExpression                          SyntaxKind = 308
	SyntaxKindJSDocNameReference                           SyntaxKind = 309
	SyntaxKindJSDocAllType                                 SyntaxKind = 310
	SyntaxKindJSDocNullableType                            SyntaxKind = 311
	SyntaxKindJSDocNonNullableType                         SyntaxKind = 312
	SyntaxKindJSDocOptionalType                            SyntaxKind = 313
	SyntaxKindJSDocVariadicType                            SyntaxKind = 314
	SyntaxKindJSDoc                                        SyntaxKind = 315
	SyntaxKindJSDocText                                    SyntaxKind = 316
	SyntaxKindJSDocTypeLiteral                             SyntaxKind = 317
	SyntaxKindJSDocSignature                               SyntaxKind = 318
	SyntaxKindJSDocLink                                    SyntaxKind = 319
	SyntaxKindJSDocLinkCode                                SyntaxKind = 320
	SyntaxKindJSDocLinkPlain                               SyntaxKind = 321
	SyntaxKindJSDocUnknownTag                              SyntaxKind = 322
	SyntaxKindJSDocAugmentsTag                             SyntaxKind = 323
	SyntaxKindJSDocImplementsTag                           SyntaxKind = 324
	SyntaxKindJSDocDeprecatedTag                           SyntaxKind = 325
	SyntaxKindJSDocPublicTag                               SyntaxKind = 326
	SyntaxKindJSDocPrivateTag                              SyntaxKind = 327
	SyntaxKindJSDocProtectedTag                            SyntaxKind = 328
	SyntaxKindJSDocReadonlyTag                             SyntaxKind = 329
	SyntaxKindJSDocOverrideTag                             SyntaxKind = 330
	SyntaxKindJSDocCallbackTag                             SyntaxKind = 331
	SyntaxKindJSDocOverloadTag                             SyntaxKind = 332
	SyntaxKindJSDocParameterTag                            SyntaxKind = 333
	SyntaxKindJSDocReturnTag                               SyntaxKind = 334
	SyntaxKindJSDocThisTag                                 SyntaxKind = 335
	SyntaxKindJSDocTypeTag                                 SyntaxKind = 336
	SyntaxKindJSDocTemplateTag                             SyntaxKind = 337
	SyntaxKindJSDocTypedefTag                              SyntaxKind = 338
	SyntaxKindJSDocSeeTag                                  SyntaxKind = 339
	SyntaxKindJSDocPropertyTag                             SyntaxKind = 340
	SyntaxKindJSDocThrowsTag                               SyntaxKind = 341
	SyntaxKindJSDocSatisfiesTag                            SyntaxKind = 342
	SyntaxKindJSDocImportTag                               SyntaxKind = 343
	SyntaxKindSyntaxList                                   SyntaxKind = 344
	SyntaxKindJSTypeAliasDeclaration                       SyntaxKind = 345
	SyntaxKindJSImportDeclaration                          SyntaxKind = 346
	SyntaxKindNotEmittedStatement                          SyntaxKind = 347
	SyntaxKindPartiallyEmittedExpression                   SyntaxKind = 348
	SyntaxKindSyntheticReferenceExpression                 SyntaxKind = 349
	SyntaxKindNotEmittedTypeElement                        SyntaxKind = 350
	SyntaxKindCount                                        SyntaxKind = 351
	SyntaxKindFirstAssignment                              SyntaxKind = 63
	SyntaxKindLastAssignment                               SyntaxKind = 78
	SyntaxKindFirstCompoundAssignment                      SyntaxKind = 64
	SyntaxKindLastCompoundAssignment                       SyntaxKind = 78
	SyntaxKindFirstReservedWord                            SyntaxKind = 82
	SyntaxKindLastReservedWord                             SyntaxKind = 117
	SyntaxKindFirstKeyword                                 SyntaxKind = 82
	SyntaxKindLastKeyword                                  SyntaxKind = 166
	SyntaxKindFirstFutureReservedWord                      SyntaxKind = 118
	SyntaxKindLastFutureReservedWord                       SyntaxKind = 126
	SyntaxKindFirstTypeNode                                SyntaxKind = 183
	SyntaxKindLastTypeNode                                 SyntaxKind = 206
	SyntaxKindFirstPunctuation                             SyntaxKind = 18
	SyntaxKindLastPunctuation                              SyntaxKind = 78
	SyntaxKindFirstToken                                   SyntaxKind = 0
	SyntaxKindLastToken                                    SyntaxKind = 166
	SyntaxKindFirstLiteralToken                            SyntaxKind = 8
	SyntaxKindLastLiteralToken                             SyntaxKind = 14
	SyntaxKindFirstTemplateToken                           SyntaxKind = 14
	SyntaxKindLastTemplateToken                            SyntaxKind = 17
	SyntaxKindFirstBinaryOperator                          SyntaxKind = 29
	SyntaxKindLastBinaryOperator                           SyntaxKind = 78
	SyntaxKindFirstStatement                               SyntaxKind = 244
	SyntaxKindLastStatement                                SyntaxKind = 260
	SyntaxKindFirstNode                                    SyntaxKind = 167
	SyntaxKindFirstJSDocNode                               SyntaxKind = 308
	SyntaxKindLastJSDocNode                                SyntaxKind = 343
	SyntaxKindFirstJSDocTagNode                            SyntaxKind = 322
	SyntaxKindLastJSDocTagNode                             SyntaxKind = 343
	SyntaxKindFirstContextualKeyword                       SyntaxKind = 127
	SyntaxKindLastContextualKeyword                        SyntaxKind = 166
	SyntaxKindLastUnaryOperator                            SyntaxKind = 54
	SyntaxKindFirstTriviaToken                             SyntaxKind = 2
	SyntaxKindLastTriviaToken                              SyntaxKind = 6
)

const (
	NodeFlagsNone                               NodeFlags = 0
	NodeFlagsLet                                NodeFlags = 1
	NodeFlagsConst                              NodeFlags = 2
	NodeFlagsUsing                              NodeFlags = 4
	NodeFlagsReparsed                           NodeFlags = 8
	NodeFlagsSynthesized                        NodeFlags = 16
	NodeFlagsOptionalChain                      NodeFlags = 32
	NodeFlagsExportContext                      NodeFlags = 64
	NodeFlagsContainsThis                       NodeFlags = 128
	NodeFlagsHasImplicitReturn                  NodeFlags = 256
	NodeFlagsHasExplicitReturn                  NodeFlags = 512
	NodeFlagsDisallowInContext                  NodeFlags = 1024
	NodeFlagsYieldContext                       NodeFlags = 2048
	NodeFlagsDecoratorContext                   NodeFlags = 4096
	NodeFlagsAwaitContext                       NodeFlags = 8192
	NodeFlagsDisallowConditionalTypesContext    NodeFlags = 16384
	NodeFlagsThisNodeHasError                   NodeFlags = 32768
	NodeFlagsJavaScriptFile                     NodeFlags = 65536
	NodeFlagsThisNodeOrAnySubNodesHasError      NodeFlags = 131072
	NodeFlagsHasAsyncFunctions                  NodeFlags = 262144
	NodeFlagsPossiblyContainsDynamicImport      NodeFlags = 524288
	NodeFlagsPossiblyContainsImportMeta         NodeFlags = 1048576
	NodeFlagsHasJSDoc                           NodeFlags = 2097152
	NodeFlagsJSDoc                              NodeFlags = 4194304
	NodeFlagsAmbient                            NodeFlags = 8388608
	NodeFlagsInWithStatement                    NodeFlags = 16777216
	NodeFlagsJsonFile                           NodeFlags = 33554432
	NodeFlagsPossiblyContainsDeprecatedTag      NodeFlags = 67108864
	NodeFlagsUnreachable                        NodeFlags = 134217728
	NodeFlagsReparserTransformedLiteral         NodeFlags = 268435456
	NodeFlagsBlockScoped                        NodeFlags = 7
	NodeFlagsConstant                           NodeFlags = 6
	NodeFlagsAwaitUsing                         NodeFlags = 6
	NodeFlagsReachabilityCheckFlags             NodeFlags = 768
	NodeFlagsReachabilityAndEmitFlags           NodeFlags = 262912
	NodeFlagsContextFlags                       NodeFlags = 25263104
	NodeFlagsTypeExcludesFlags                  NodeFlags = 10240
	NodeFlagsPermanentlySetIncrementalFlags     NodeFlags = 1572864
	NodeFlagsIdentifierHasExtendedUnicodeEscape NodeFlags = 128
	NodeFlagsIdentifierIsInJSDocNamespace       NodeFlags = 262144
	NodeFlagsNestedNamespace                    NodeFlags = 32
)

const (
	TokenFlagsNone                           TokenFlags = 0
	TokenFlagsPrecedingLineBreak             TokenFlags = 1
	TokenFlagsPrecedingJSDocComment          TokenFlags = 2
	TokenFlagsUnterminated                   TokenFlags = 4
	TokenFlagsExtendedUnicodeEscape          TokenFlags = 8
	TokenFlagsScientific                     TokenFlags = 16
	TokenFlagsOctal                          TokenFlags = 32
	TokenFlagsHexSpecifier                   TokenFlags = 64
	TokenFlagsBinarySpecifier                TokenFlags = 128
	TokenFlagsOctalSpecifier                 TokenFlags = 256
	TokenFlagsContainsSeparator              TokenFlags = 512
	TokenFlagsUnicodeEscape                  TokenFlags = 1024
	TokenFlagsContainsInvalidEscape          TokenFlags = 2048
	TokenFlagsHexEscape                      TokenFlags = 4096
	TokenFlagsContainsLeadingZero            TokenFlags = 8192
	TokenFlagsContainsInvalidSeparator       TokenFlags = 16384
	TokenFlagsPrecedingJSDocLeadingAsterisks TokenFlags = 32768
	TokenFlagsSingleQuote                    TokenFlags = 65536
	TokenFlagsPrecedingJSDocWithDeprecated   TokenFlags = 131072
	TokenFlagsPrecedingJSDocWithSeeOrLink    TokenFlags = 262144
	TokenFlagsBinaryOrOctalSpecifier         TokenFlags = 384
	TokenFlagsWithSpecifier                  TokenFlags = 448
	TokenFlagsStringLiteralFlags             TokenFlags = 72716
	TokenFlagsNumericLiteralFlags            TokenFlags = 25584
	TokenFlagsTemplateLiteralLikeFlags       TokenFlags = 7180
	TokenFlagsRegularExpressionLiteralFlags  TokenFlags = 4
	TokenFlagsIsInvalid                      TokenFlags = 26656
)

const (
	LanguageVariantStandard LanguageVariant = 0
	LanguageVariantJSX      LanguageVariant = 1
)

const (
	ScriptKindUnknown  ScriptKind = 0
	ScriptKindJS       ScriptKind = 1
	ScriptKindJSX      ScriptKind = 2
	ScriptKindTS       ScriptKind = 3
	ScriptKindTSX      ScriptKind = 4
	ScriptKindExternal ScriptKind = 5
	ScriptKindJSON     ScriptKind = 6
	ScriptKindDeferred ScriptKind = 7
)

type AdditiveOperator SyntaxKind

const (
	AdditiveOperatorPlusToken  AdditiveOperator = 39
	AdditiveOperatorMinusToken AdditiveOperator = 40
)

type AdditiveOperatorOrHigher SyntaxKind

const (
	AdditiveOperatorOrHigherAsteriskAsteriskToken AdditiveOperatorOrHigher = 42
	AdditiveOperatorOrHigherAsteriskToken         AdditiveOperatorOrHigher = 41
	AdditiveOperatorOrHigherSlashToken            AdditiveOperatorOrHigher = 43
	AdditiveOperatorOrHigherPercentToken          AdditiveOperatorOrHigher = 44
	AdditiveOperatorOrHigherPlusToken             AdditiveOperatorOrHigher = 39
	AdditiveOperatorOrHigherMinusToken            AdditiveOperatorOrHigher = 40
)

type AssignmentOperator SyntaxKind

const (
	AssignmentOperatorEqualsToken                                  AssignmentOperator = 63
	AssignmentOperatorPlusEqualsToken                              AssignmentOperator = 64
	AssignmentOperatorMinusEqualsToken                             AssignmentOperator = 65
	AssignmentOperatorAsteriskAsteriskEqualsToken                  AssignmentOperator = 67
	AssignmentOperatorAsteriskEqualsToken                          AssignmentOperator = 66
	AssignmentOperatorSlashEqualsToken                             AssignmentOperator = 68
	AssignmentOperatorPercentEqualsToken                           AssignmentOperator = 69
	AssignmentOperatorAmpersandEqualsToken                         AssignmentOperator = 73
	AssignmentOperatorBarEqualsToken                               AssignmentOperator = 74
	AssignmentOperatorCaretEqualsToken                             AssignmentOperator = 78
	AssignmentOperatorLessThanLessThanEqualsToken                  AssignmentOperator = 70
	AssignmentOperatorGreaterThanGreaterThanGreaterThanEqualsToken AssignmentOperator = 72
	AssignmentOperatorGreaterThanGreaterThanEqualsToken            AssignmentOperator = 71
	AssignmentOperatorBarBarEqualsToken                            AssignmentOperator = 75
	AssignmentOperatorAmpersandAmpersandEqualsToken                AssignmentOperator = 76
	AssignmentOperatorQuestionQuestionEqualsToken                  AssignmentOperator = 77
)

type AssignmentOperatorOrHigher SyntaxKind

const (
	AssignmentOperatorOrHigherQuestionQuestionToken                        AssignmentOperatorOrHigher = 60
	AssignmentOperatorOrHigherAsteriskAsteriskToken                        AssignmentOperatorOrHigher = 42
	AssignmentOperatorOrHigherAsteriskToken                                AssignmentOperatorOrHigher = 41
	AssignmentOperatorOrHigherSlashToken                                   AssignmentOperatorOrHigher = 43
	AssignmentOperatorOrHigherPercentToken                                 AssignmentOperatorOrHigher = 44
	AssignmentOperatorOrHigherPlusToken                                    AssignmentOperatorOrHigher = 39
	AssignmentOperatorOrHigherMinusToken                                   AssignmentOperatorOrHigher = 40
	AssignmentOperatorOrHigherLessThanLessThanToken                        AssignmentOperatorOrHigher = 47
	AssignmentOperatorOrHigherGreaterThanGreaterThanToken                  AssignmentOperatorOrHigher = 48
	AssignmentOperatorOrHigherGreaterThanGreaterThanGreaterThanToken       AssignmentOperatorOrHigher = 49
	AssignmentOperatorOrHigherLessThanToken                                AssignmentOperatorOrHigher = 29
	AssignmentOperatorOrHigherLessThanEqualsToken                          AssignmentOperatorOrHigher = 32
	AssignmentOperatorOrHigherGreaterThanToken                             AssignmentOperatorOrHigher = 31
	AssignmentOperatorOrHigherGreaterThanEqualsToken                       AssignmentOperatorOrHigher = 33
	AssignmentOperatorOrHigherInstanceOfKeyword                            AssignmentOperatorOrHigher = 103
	AssignmentOperatorOrHigherInKeyword                                    AssignmentOperatorOrHigher = 102
	AssignmentOperatorOrHigherEqualsEqualsToken                            AssignmentOperatorOrHigher = 34
	AssignmentOperatorOrHigherEqualsEqualsEqualsToken                      AssignmentOperatorOrHigher = 36
	AssignmentOperatorOrHigherExclamationEqualsEqualsToken                 AssignmentOperatorOrHigher = 37
	AssignmentOperatorOrHigherExclamationEqualsToken                       AssignmentOperatorOrHigher = 35
	AssignmentOperatorOrHigherAmpersandToken                               AssignmentOperatorOrHigher = 50
	AssignmentOperatorOrHigherBarToken                                     AssignmentOperatorOrHigher = 51
	AssignmentOperatorOrHigherCaretToken                                   AssignmentOperatorOrHigher = 52
	AssignmentOperatorOrHigherAmpersandAmpersandToken                      AssignmentOperatorOrHigher = 55
	AssignmentOperatorOrHigherBarBarToken                                  AssignmentOperatorOrHigher = 56
	AssignmentOperatorOrHigherEqualsToken                                  AssignmentOperatorOrHigher = 63
	AssignmentOperatorOrHigherPlusEqualsToken                              AssignmentOperatorOrHigher = 64
	AssignmentOperatorOrHigherMinusEqualsToken                             AssignmentOperatorOrHigher = 65
	AssignmentOperatorOrHigherAsteriskAsteriskEqualsToken                  AssignmentOperatorOrHigher = 67
	AssignmentOperatorOrHigherAsteriskEqualsToken                          AssignmentOperatorOrHigher = 66
	AssignmentOperatorOrHigherSlashEqualsToken                             AssignmentOperatorOrHigher = 68
	AssignmentOperatorOrHigherPercentEqualsToken                           AssignmentOperatorOrHigher = 69
	AssignmentOperatorOrHigherAmpersandEqualsToken                         AssignmentOperatorOrHigher = 73
	AssignmentOperatorOrHigherBarEqualsToken                               AssignmentOperatorOrHigher = 74
	AssignmentOperatorOrHigherCaretEqualsToken                             AssignmentOperatorOrHigher = 78
	AssignmentOperatorOrHigherLessThanLessThanEqualsToken                  AssignmentOperatorOrHigher = 70
	AssignmentOperatorOrHigherGreaterThanGreaterThanGreaterThanEqualsToken AssignmentOperatorOrHigher = 72
	AssignmentOperatorOrHigherGreaterThanGreaterThanEqualsToken            AssignmentOperatorOrHigher = 71
	AssignmentOperatorOrHigherBarBarEqualsToken                            AssignmentOperatorOrHigher = 75
	AssignmentOperatorOrHigherAmpersandAmpersandEqualsToken                AssignmentOperatorOrHigher = 76
	AssignmentOperatorOrHigherQuestionQuestionEqualsToken                  AssignmentOperatorOrHigher = 77
)

type BinaryOperator SyntaxKind

const (
	BinaryOperatorQuestionQuestionToken                        BinaryOperator = 60
	BinaryOperatorAsteriskAsteriskToken                        BinaryOperator = 42
	BinaryOperatorAsteriskToken                                BinaryOperator = 41
	BinaryOperatorSlashToken                                   BinaryOperator = 43
	BinaryOperatorPercentToken                                 BinaryOperator = 44
	BinaryOperatorPlusToken                                    BinaryOperator = 39
	BinaryOperatorMinusToken                                   BinaryOperator = 40
	BinaryOperatorLessThanLessThanToken                        BinaryOperator = 47
	BinaryOperatorGreaterThanGreaterThanToken                  BinaryOperator = 48
	BinaryOperatorGreaterThanGreaterThanGreaterThanToken       BinaryOperator = 49
	BinaryOperatorLessThanToken                                BinaryOperator = 29
	BinaryOperatorLessThanEqualsToken                          BinaryOperator = 32
	BinaryOperatorGreaterThanToken                             BinaryOperator = 31
	BinaryOperatorGreaterThanEqualsToken                       BinaryOperator = 33
	BinaryOperatorInstanceOfKeyword                            BinaryOperator = 103
	BinaryOperatorInKeyword                                    BinaryOperator = 102
	BinaryOperatorEqualsEqualsToken                            BinaryOperator = 34
	BinaryOperatorEqualsEqualsEqualsToken                      BinaryOperator = 36
	BinaryOperatorExclamationEqualsEqualsToken                 BinaryOperator = 37
	BinaryOperatorExclamationEqualsToken                       BinaryOperator = 35
	BinaryOperatorAmpersandToken                               BinaryOperator = 50
	BinaryOperatorBarToken                                     BinaryOperator = 51
	BinaryOperatorCaretToken                                   BinaryOperator = 52
	BinaryOperatorAmpersandAmpersandToken                      BinaryOperator = 55
	BinaryOperatorBarBarToken                                  BinaryOperator = 56
	BinaryOperatorEqualsToken                                  BinaryOperator = 63
	BinaryOperatorPlusEqualsToken                              BinaryOperator = 64
	BinaryOperatorMinusEqualsToken                             BinaryOperator = 65
	BinaryOperatorAsteriskAsteriskEqualsToken                  BinaryOperator = 67
	BinaryOperatorAsteriskEqualsToken                          BinaryOperator = 66
	BinaryOperatorSlashEqualsToken                             BinaryOperator = 68
	BinaryOperatorPercentEqualsToken                           BinaryOperator = 69
	BinaryOperatorAmpersandEqualsToken                         BinaryOperator = 73
	BinaryOperatorBarEqualsToken                               BinaryOperator = 74
	BinaryOperatorCaretEqualsToken                             BinaryOperator = 78
	BinaryOperatorLessThanLessThanEqualsToken                  BinaryOperator = 70
	BinaryOperatorGreaterThanGreaterThanGreaterThanEqualsToken BinaryOperator = 72
	BinaryOperatorGreaterThanGreaterThanEqualsToken            BinaryOperator = 71
	BinaryOperatorBarBarEqualsToken                            BinaryOperator = 75
	BinaryOperatorAmpersandAmpersandEqualsToken                BinaryOperator = 76
	BinaryOperatorQuestionQuestionEqualsToken                  BinaryOperator = 77
	BinaryOperatorCommaToken                                   BinaryOperator = 27
)

type BitwiseOperator SyntaxKind

const (
	BitwiseOperatorAmpersandToken BitwiseOperator = 50
	BitwiseOperatorBarToken       BitwiseOperator = 51
	BitwiseOperatorCaretToken     BitwiseOperator = 52
)

type BitwiseOperatorOrHigher SyntaxKind

const (
	BitwiseOperatorOrHigherAsteriskAsteriskToken                  BitwiseOperatorOrHigher = 42
	BitwiseOperatorOrHigherAsteriskToken                          BitwiseOperatorOrHigher = 41
	BitwiseOperatorOrHigherSlashToken                             BitwiseOperatorOrHigher = 43
	BitwiseOperatorOrHigherPercentToken                           BitwiseOperatorOrHigher = 44
	BitwiseOperatorOrHigherPlusToken                              BitwiseOperatorOrHigher = 39
	BitwiseOperatorOrHigherMinusToken                             BitwiseOperatorOrHigher = 40
	BitwiseOperatorOrHigherLessThanLessThanToken                  BitwiseOperatorOrHigher = 47
	BitwiseOperatorOrHigherGreaterThanGreaterThanToken            BitwiseOperatorOrHigher = 48
	BitwiseOperatorOrHigherGreaterThanGreaterThanGreaterThanToken BitwiseOperatorOrHigher = 49
	BitwiseOperatorOrHigherLessThanToken                          BitwiseOperatorOrHigher = 29
	BitwiseOperatorOrHigherLessThanEqualsToken                    BitwiseOperatorOrHigher = 32
	BitwiseOperatorOrHigherGreaterThanToken                       BitwiseOperatorOrHigher = 31
	BitwiseOperatorOrHigherGreaterThanEqualsToken                 BitwiseOperatorOrHigher = 33
	BitwiseOperatorOrHigherInstanceOfKeyword                      BitwiseOperatorOrHigher = 103
	BitwiseOperatorOrHigherInKeyword                              BitwiseOperatorOrHigher = 102
	BitwiseOperatorOrHigherEqualsEqualsToken                      BitwiseOperatorOrHigher = 34
	BitwiseOperatorOrHigherEqualsEqualsEqualsToken                BitwiseOperatorOrHigher = 36
	BitwiseOperatorOrHigherExclamationEqualsEqualsToken           BitwiseOperatorOrHigher = 37
	BitwiseOperatorOrHigherExclamationEqualsToken                 BitwiseOperatorOrHigher = 35
	BitwiseOperatorOrHigherAmpersandToken                         BitwiseOperatorOrHigher = 50
	BitwiseOperatorOrHigherBarToken                               BitwiseOperatorOrHigher = 51
	BitwiseOperatorOrHigherCaretToken                             BitwiseOperatorOrHigher = 52
)

type CompoundAssignmentOperator SyntaxKind

const (
	CompoundAssignmentOperatorPlusEqualsToken                              CompoundAssignmentOperator = 64
	CompoundAssignmentOperatorMinusEqualsToken                             CompoundAssignmentOperator = 65
	CompoundAssignmentOperatorAsteriskAsteriskEqualsToken                  CompoundAssignmentOperator = 67
	CompoundAssignmentOperatorAsteriskEqualsToken                          CompoundAssignmentOperator = 66
	CompoundAssignmentOperatorSlashEqualsToken                             CompoundAssignmentOperator = 68
	CompoundAssignmentOperatorPercentEqualsToken                           CompoundAssignmentOperator = 69
	CompoundAssignmentOperatorAmpersandEqualsToken                         CompoundAssignmentOperator = 73
	CompoundAssignmentOperatorBarEqualsToken                               CompoundAssignmentOperator = 74
	CompoundAssignmentOperatorCaretEqualsToken                             CompoundAssignmentOperator = 78
	CompoundAssignmentOperatorLessThanLessThanEqualsToken                  CompoundAssignmentOperator = 70
	CompoundAssignmentOperatorGreaterThanGreaterThanGreaterThanEqualsToken CompoundAssignmentOperator = 72
	CompoundAssignmentOperatorGreaterThanGreaterThanEqualsToken            CompoundAssignmentOperator = 71
	CompoundAssignmentOperatorBarBarEqualsToken                            CompoundAssignmentOperator = 75
	CompoundAssignmentOperatorAmpersandAmpersandEqualsToken                CompoundAssignmentOperator = 76
	CompoundAssignmentOperatorQuestionQuestionEqualsToken                  CompoundAssignmentOperator = 77
)

type EqualityOperator SyntaxKind

const (
	EqualityOperatorEqualsEqualsToken            EqualityOperator = 34
	EqualityOperatorEqualsEqualsEqualsToken      EqualityOperator = 36
	EqualityOperatorExclamationEqualsEqualsToken EqualityOperator = 37
	EqualityOperatorExclamationEqualsToken       EqualityOperator = 35
)

type EqualityOperatorOrHigher SyntaxKind

const (
	EqualityOperatorOrHigherAsteriskAsteriskToken                  EqualityOperatorOrHigher = 42
	EqualityOperatorOrHigherAsteriskToken                          EqualityOperatorOrHigher = 41
	EqualityOperatorOrHigherSlashToken                             EqualityOperatorOrHigher = 43
	EqualityOperatorOrHigherPercentToken                           EqualityOperatorOrHigher = 44
	EqualityOperatorOrHigherPlusToken                              EqualityOperatorOrHigher = 39
	EqualityOperatorOrHigherMinusToken                             EqualityOperatorOrHigher = 40
	EqualityOperatorOrHigherLessThanLessThanToken                  EqualityOperatorOrHigher = 47
	EqualityOperatorOrHigherGreaterThanGreaterThanToken            EqualityOperatorOrHigher = 48
	EqualityOperatorOrHigherGreaterThanGreaterThanGreaterThanToken EqualityOperatorOrHigher = 49
	EqualityOperatorOrHigherLessThanToken                          EqualityOperatorOrHigher = 29
	EqualityOperatorOrHigherLessThanEqualsToken                    EqualityOperatorOrHigher = 32
	EqualityOperatorOrHigherGreaterThanToken                       EqualityOperatorOrHigher = 31
	EqualityOperatorOrHigherGreaterThanEqualsToken                 EqualityOperatorOrHigher = 33
	EqualityOperatorOrHigherInstanceOfKeyword                      EqualityOperatorOrHigher = 103
	EqualityOperatorOrHigherInKeyword                              EqualityOperatorOrHigher = 102
	EqualityOperatorOrHigherEqualsEqualsToken                      EqualityOperatorOrHigher = 34
	EqualityOperatorOrHigherEqualsEqualsEqualsToken                EqualityOperatorOrHigher = 36
	EqualityOperatorOrHigherExclamationEqualsEqualsToken           EqualityOperatorOrHigher = 37
	EqualityOperatorOrHigherExclamationEqualsToken                 EqualityOperatorOrHigher = 35
)

type ExponentiationOperator SyntaxKind

const (
	ExponentiationOperatorAsteriskAsteriskToken ExponentiationOperator = 42
)

type ImportPhaseModifierSyntaxKind SyntaxKind

const (
	ImportPhaseModifierSyntaxKindTypeKeyword  ImportPhaseModifierSyntaxKind = 156
	ImportPhaseModifierSyntaxKindDeferKeyword ImportPhaseModifierSyntaxKind = 166
)

type JSDocNodeSyntaxKind SyntaxKind

const (
	JSDocNodeSyntaxKindJSDocTypeExpression  JSDocNodeSyntaxKind = 308
	JSDocNodeSyntaxKindJSDocNameReference   JSDocNodeSyntaxKind = 309
	JSDocNodeSyntaxKindJSDocAllType         JSDocNodeSyntaxKind = 310
	JSDocNodeSyntaxKindJSDocNullableType    JSDocNodeSyntaxKind = 311
	JSDocNodeSyntaxKindJSDocNonNullableType JSDocNodeSyntaxKind = 312
	JSDocNodeSyntaxKindJSDocOptionalType    JSDocNodeSyntaxKind = 313
	JSDocNodeSyntaxKindJSDocVariadicType    JSDocNodeSyntaxKind = 314
	JSDocNodeSyntaxKindJSDoc                JSDocNodeSyntaxKind = 315
	JSDocNodeSyntaxKindJSDocText            JSDocNodeSyntaxKind = 316
	JSDocNodeSyntaxKindJSDocTypeLiteral     JSDocNodeSyntaxKind = 317
	JSDocNodeSyntaxKindJSDocSignature       JSDocNodeSyntaxKind = 318
	JSDocNodeSyntaxKindJSDocLink            JSDocNodeSyntaxKind = 319
	JSDocNodeSyntaxKindJSDocLinkCode        JSDocNodeSyntaxKind = 320
	JSDocNodeSyntaxKindJSDocLinkPlain       JSDocNodeSyntaxKind = 321
	JSDocNodeSyntaxKindJSDocUnknownTag      JSDocNodeSyntaxKind = 322
	JSDocNodeSyntaxKindJSDocAugmentsTag     JSDocNodeSyntaxKind = 323
	JSDocNodeSyntaxKindJSDocImplementsTag   JSDocNodeSyntaxKind = 324
	JSDocNodeSyntaxKindJSDocDeprecatedTag   JSDocNodeSyntaxKind = 325
	JSDocNodeSyntaxKindJSDocPublicTag       JSDocNodeSyntaxKind = 326
	JSDocNodeSyntaxKindJSDocPrivateTag      JSDocNodeSyntaxKind = 327
	JSDocNodeSyntaxKindJSDocProtectedTag    JSDocNodeSyntaxKind = 328
	JSDocNodeSyntaxKindJSDocReadonlyTag     JSDocNodeSyntaxKind = 329
	JSDocNodeSyntaxKindJSDocOverrideTag     JSDocNodeSyntaxKind = 330
	JSDocNodeSyntaxKindJSDocCallbackTag     JSDocNodeSyntaxKind = 331
	JSDocNodeSyntaxKindJSDocOverloadTag     JSDocNodeSyntaxKind = 332
	JSDocNodeSyntaxKindJSDocParameterTag    JSDocNodeSyntaxKind = 333
	JSDocNodeSyntaxKindJSDocReturnTag       JSDocNodeSyntaxKind = 334
	JSDocNodeSyntaxKindJSDocThisTag         JSDocNodeSyntaxKind = 335
	JSDocNodeSyntaxKindJSDocTypeTag         JSDocNodeSyntaxKind = 336
	JSDocNodeSyntaxKindJSDocTemplateTag     JSDocNodeSyntaxKind = 337
	JSDocNodeSyntaxKindJSDocTypedefTag      JSDocNodeSyntaxKind = 338
	JSDocNodeSyntaxKindJSDocSeeTag          JSDocNodeSyntaxKind = 339
	JSDocNodeSyntaxKindJSDocPropertyTag     JSDocNodeSyntaxKind = 340
	JSDocNodeSyntaxKindJSDocThrowsTag       JSDocNodeSyntaxKind = 341
	JSDocNodeSyntaxKindJSDocSatisfiesTag    JSDocNodeSyntaxKind = 342
	JSDocNodeSyntaxKindJSDocImportTag       JSDocNodeSyntaxKind = 343
)

type JsxTokenSyntaxKind SyntaxKind

const (
	JsxTokenSyntaxKindLessThanSlashToken    JsxTokenSyntaxKind = 30
	JsxTokenSyntaxKindEndOfFile             JsxTokenSyntaxKind = 1
	JsxTokenSyntaxKindConflictMarkerTrivia  JsxTokenSyntaxKind = 6
	JsxTokenSyntaxKindJsxText               JsxTokenSyntaxKind = 11
	JsxTokenSyntaxKindJsxTextAllWhiteSpaces JsxTokenSyntaxKind = 12
	JsxTokenSyntaxKindOpenBraceToken        JsxTokenSyntaxKind = 18
	JsxTokenSyntaxKindLessThanToken         JsxTokenSyntaxKind = 29
)

type KeywordExpressionSyntaxKind SyntaxKind

const (
	KeywordExpressionSyntaxKindNullKeyword   KeywordExpressionSyntaxKind = 105
	KeywordExpressionSyntaxKindTrueKeyword   KeywordExpressionSyntaxKind = 111
	KeywordExpressionSyntaxKindFalseKeyword  KeywordExpressionSyntaxKind = 96
	KeywordExpressionSyntaxKindThisKeyword   KeywordExpressionSyntaxKind = 109
	KeywordExpressionSyntaxKindSuperKeyword  KeywordExpressionSyntaxKind = 107
	KeywordExpressionSyntaxKindImportKeyword KeywordExpressionSyntaxKind = 101
)

type KeywordSyntaxKind SyntaxKind

const (
	KeywordSyntaxKindBreakKeyword       KeywordSyntaxKind = 82
	KeywordSyntaxKindCaseKeyword        KeywordSyntaxKind = 83
	KeywordSyntaxKindCatchKeyword       KeywordSyntaxKind = 84
	KeywordSyntaxKindClassKeyword       KeywordSyntaxKind = 85
	KeywordSyntaxKindConstKeyword       KeywordSyntaxKind = 86
	KeywordSyntaxKindContinueKeyword    KeywordSyntaxKind = 87
	KeywordSyntaxKindDebuggerKeyword    KeywordSyntaxKind = 88
	KeywordSyntaxKindDefaultKeyword     KeywordSyntaxKind = 89
	KeywordSyntaxKindDeleteKeyword      KeywordSyntaxKind = 90
	KeywordSyntaxKindDoKeyword          KeywordSyntaxKind = 91
	KeywordSyntaxKindElseKeyword        KeywordSyntaxKind = 92
	KeywordSyntaxKindEnumKeyword        KeywordSyntaxKind = 93
	KeywordSyntaxKindExportKeyword      KeywordSyntaxKind = 94
	KeywordSyntaxKindExtendsKeyword     KeywordSyntaxKind = 95
	KeywordSyntaxKindFalseKeyword       KeywordSyntaxKind = 96
	KeywordSyntaxKindFinallyKeyword     KeywordSyntaxKind = 97
	KeywordSyntaxKindForKeyword         KeywordSyntaxKind = 98
	KeywordSyntaxKindFunctionKeyword    KeywordSyntaxKind = 99
	KeywordSyntaxKindIfKeyword          KeywordSyntaxKind = 100
	KeywordSyntaxKindImportKeyword      KeywordSyntaxKind = 101
	KeywordSyntaxKindInKeyword          KeywordSyntaxKind = 102
	KeywordSyntaxKindInstanceOfKeyword  KeywordSyntaxKind = 103
	KeywordSyntaxKindNewKeyword         KeywordSyntaxKind = 104
	KeywordSyntaxKindNullKeyword        KeywordSyntaxKind = 105
	KeywordSyntaxKindReturnKeyword      KeywordSyntaxKind = 106
	KeywordSyntaxKindSuperKeyword       KeywordSyntaxKind = 107
	KeywordSyntaxKindSwitchKeyword      KeywordSyntaxKind = 108
	KeywordSyntaxKindThisKeyword        KeywordSyntaxKind = 109
	KeywordSyntaxKindThrowKeyword       KeywordSyntaxKind = 110
	KeywordSyntaxKindTrueKeyword        KeywordSyntaxKind = 111
	KeywordSyntaxKindTryKeyword         KeywordSyntaxKind = 112
	KeywordSyntaxKindTypeOfKeyword      KeywordSyntaxKind = 113
	KeywordSyntaxKindVarKeyword         KeywordSyntaxKind = 114
	KeywordSyntaxKindVoidKeyword        KeywordSyntaxKind = 115
	KeywordSyntaxKindWhileKeyword       KeywordSyntaxKind = 116
	KeywordSyntaxKindWithKeyword        KeywordSyntaxKind = 117
	KeywordSyntaxKindImplementsKeyword  KeywordSyntaxKind = 118
	KeywordSyntaxKindInterfaceKeyword   KeywordSyntaxKind = 119
	KeywordSyntaxKindLetKeyword         KeywordSyntaxKind = 120
	KeywordSyntaxKindPackageKeyword     KeywordSyntaxKind = 121
	KeywordSyntaxKindPrivateKeyword     KeywordSyntaxKind = 122
	KeywordSyntaxKindProtectedKeyword   KeywordSyntaxKind = 123
	KeywordSyntaxKindPublicKeyword      KeywordSyntaxKind = 124
	KeywordSyntaxKindStaticKeyword      KeywordSyntaxKind = 125
	KeywordSyntaxKindYieldKeyword       KeywordSyntaxKind = 126
	KeywordSyntaxKindAbstractKeyword    KeywordSyntaxKind = 127
	KeywordSyntaxKindAccessorKeyword    KeywordSyntaxKind = 128
	KeywordSyntaxKindAsKeyword          KeywordSyntaxKind = 129
	KeywordSyntaxKindAssertsKeyword     KeywordSyntaxKind = 130
	KeywordSyntaxKindAssertKeyword      KeywordSyntaxKind = 131
	KeywordSyntaxKindAnyKeyword         KeywordSyntaxKind = 132
	KeywordSyntaxKindAsyncKeyword       KeywordSyntaxKind = 133
	KeywordSyntaxKindAwaitKeyword       KeywordSyntaxKind = 134
	KeywordSyntaxKindBooleanKeyword     KeywordSyntaxKind = 135
	KeywordSyntaxKindConstructorKeyword KeywordSyntaxKind = 136
	KeywordSyntaxKindDeclareKeyword     KeywordSyntaxKind = 137
	KeywordSyntaxKindGetKeyword         KeywordSyntaxKind = 138
	KeywordSyntaxKindImmediateKeyword   KeywordSyntaxKind = 139
	KeywordSyntaxKindInferKeyword       KeywordSyntaxKind = 140
	KeywordSyntaxKindIntrinsicKeyword   KeywordSyntaxKind = 141
	KeywordSyntaxKindIsKeyword          KeywordSyntaxKind = 142
	KeywordSyntaxKindKeyOfKeyword       KeywordSyntaxKind = 143
	KeywordSyntaxKindModuleKeyword      KeywordSyntaxKind = 144
	KeywordSyntaxKindNamespaceKeyword   KeywordSyntaxKind = 145
	KeywordSyntaxKindNeverKeyword       KeywordSyntaxKind = 146
	KeywordSyntaxKindOutKeyword         KeywordSyntaxKind = 147
	KeywordSyntaxKindReadonlyKeyword    KeywordSyntaxKind = 148
	KeywordSyntaxKindRequireKeyword     KeywordSyntaxKind = 149
	KeywordSyntaxKindNumberKeyword      KeywordSyntaxKind = 150
	KeywordSyntaxKindObjectKeyword      KeywordSyntaxKind = 151
	KeywordSyntaxKindSatisfiesKeyword   KeywordSyntaxKind = 152
	KeywordSyntaxKindSetKeyword         KeywordSyntaxKind = 153
	KeywordSyntaxKindStringKeyword      KeywordSyntaxKind = 154
	KeywordSyntaxKindSymbolKeyword      KeywordSyntaxKind = 155
	KeywordSyntaxKindTypeKeyword        KeywordSyntaxKind = 156
	KeywordSyntaxKindUndefinedKeyword   KeywordSyntaxKind = 157
	KeywordSyntaxKindUniqueKeyword      KeywordSyntaxKind = 158
	KeywordSyntaxKindUnknownKeyword     KeywordSyntaxKind = 159
	KeywordSyntaxKindUsingKeyword       KeywordSyntaxKind = 160
	KeywordSyntaxKindFromKeyword        KeywordSyntaxKind = 161
	KeywordSyntaxKindGlobalKeyword      KeywordSyntaxKind = 162
	KeywordSyntaxKindBigIntKeyword      KeywordSyntaxKind = 163
	KeywordSyntaxKindOverrideKeyword    KeywordSyntaxKind = 164
	KeywordSyntaxKindOfKeyword          KeywordSyntaxKind = 165
	KeywordSyntaxKindDeferKeyword       KeywordSyntaxKind = 166
)

type KeywordTypeSyntaxKind SyntaxKind

const (
	KeywordTypeSyntaxKindAnyKeyword       KeywordTypeSyntaxKind = 132
	KeywordTypeSyntaxKindBigIntKeyword    KeywordTypeSyntaxKind = 163
	KeywordTypeSyntaxKindBooleanKeyword   KeywordTypeSyntaxKind = 135
	KeywordTypeSyntaxKindIntrinsicKeyword KeywordTypeSyntaxKind = 141
	KeywordTypeSyntaxKindNeverKeyword     KeywordTypeSyntaxKind = 146
	KeywordTypeSyntaxKindNumberKeyword    KeywordTypeSyntaxKind = 150
	KeywordTypeSyntaxKindObjectKeyword    KeywordTypeSyntaxKind = 151
	KeywordTypeSyntaxKindStringKeyword    KeywordTypeSyntaxKind = 154
	KeywordTypeSyntaxKindSymbolKeyword    KeywordTypeSyntaxKind = 155
	KeywordTypeSyntaxKindUndefinedKeyword KeywordTypeSyntaxKind = 157
	KeywordTypeSyntaxKindUnknownKeyword   KeywordTypeSyntaxKind = 159
	KeywordTypeSyntaxKindVoidKeyword      KeywordTypeSyntaxKind = 115
)

type LiteralSyntaxKind SyntaxKind

const (
	LiteralSyntaxKindNumericLiteral                LiteralSyntaxKind = 8
	LiteralSyntaxKindBigIntLiteral                 LiteralSyntaxKind = 9
	LiteralSyntaxKindStringLiteral                 LiteralSyntaxKind = 10
	LiteralSyntaxKindJsxText                       LiteralSyntaxKind = 11
	LiteralSyntaxKindJsxTextAllWhiteSpaces         LiteralSyntaxKind = 12
	LiteralSyntaxKindRegularExpressionLiteral      LiteralSyntaxKind = 13
	LiteralSyntaxKindNoSubstitutionTemplateLiteral LiteralSyntaxKind = 14
)

type LogicalOperator SyntaxKind

const (
	LogicalOperatorAmpersandAmpersandToken LogicalOperator = 55
	LogicalOperatorBarBarToken             LogicalOperator = 56
)

type LogicalOperatorOrHigher SyntaxKind

const (
	LogicalOperatorOrHigherAsteriskAsteriskToken                  LogicalOperatorOrHigher = 42
	LogicalOperatorOrHigherAsteriskToken                          LogicalOperatorOrHigher = 41
	LogicalOperatorOrHigherSlashToken                             LogicalOperatorOrHigher = 43
	LogicalOperatorOrHigherPercentToken                           LogicalOperatorOrHigher = 44
	LogicalOperatorOrHigherPlusToken                              LogicalOperatorOrHigher = 39
	LogicalOperatorOrHigherMinusToken                             LogicalOperatorOrHigher = 40
	LogicalOperatorOrHigherLessThanLessThanToken                  LogicalOperatorOrHigher = 47
	LogicalOperatorOrHigherGreaterThanGreaterThanToken            LogicalOperatorOrHigher = 48
	LogicalOperatorOrHigherGreaterThanGreaterThanGreaterThanToken LogicalOperatorOrHigher = 49
	LogicalOperatorOrHigherLessThanToken                          LogicalOperatorOrHigher = 29
	LogicalOperatorOrHigherLessThanEqualsToken                    LogicalOperatorOrHigher = 32
	LogicalOperatorOrHigherGreaterThanToken                       LogicalOperatorOrHigher = 31
	LogicalOperatorOrHigherGreaterThanEqualsToken                 LogicalOperatorOrHigher = 33
	LogicalOperatorOrHigherInstanceOfKeyword                      LogicalOperatorOrHigher = 103
	LogicalOperatorOrHigherInKeyword                              LogicalOperatorOrHigher = 102
	LogicalOperatorOrHigherEqualsEqualsToken                      LogicalOperatorOrHigher = 34
	LogicalOperatorOrHigherEqualsEqualsEqualsToken                LogicalOperatorOrHigher = 36
	LogicalOperatorOrHigherExclamationEqualsEqualsToken           LogicalOperatorOrHigher = 37
	LogicalOperatorOrHigherExclamationEqualsToken                 LogicalOperatorOrHigher = 35
	LogicalOperatorOrHigherAmpersandToken                         LogicalOperatorOrHigher = 50
	LogicalOperatorOrHigherBarToken                               LogicalOperatorOrHigher = 51
	LogicalOperatorOrHigherCaretToken                             LogicalOperatorOrHigher = 52
	LogicalOperatorOrHigherAmpersandAmpersandToken                LogicalOperatorOrHigher = 55
	LogicalOperatorOrHigherBarBarToken                            LogicalOperatorOrHigher = 56
)

type LogicalOrCoalescingAssignmentOperator SyntaxKind

const (
	LogicalOrCoalescingAssignmentOperatorAmpersandAmpersandEqualsToken LogicalOrCoalescingAssignmentOperator = 76
	LogicalOrCoalescingAssignmentOperatorBarBarEqualsToken             LogicalOrCoalescingAssignmentOperator = 75
	LogicalOrCoalescingAssignmentOperatorQuestionQuestionEqualsToken   LogicalOrCoalescingAssignmentOperator = 77
)

type ModifierSyntaxKind SyntaxKind

const (
	ModifierSyntaxKindAbstractKeyword  ModifierSyntaxKind = 127
	ModifierSyntaxKindAccessorKeyword  ModifierSyntaxKind = 128
	ModifierSyntaxKindAsyncKeyword     ModifierSyntaxKind = 133
	ModifierSyntaxKindConstKeyword     ModifierSyntaxKind = 86
	ModifierSyntaxKindDeclareKeyword   ModifierSyntaxKind = 137
	ModifierSyntaxKindDefaultKeyword   ModifierSyntaxKind = 89
	ModifierSyntaxKindExportKeyword    ModifierSyntaxKind = 94
	ModifierSyntaxKindInKeyword        ModifierSyntaxKind = 102
	ModifierSyntaxKindPrivateKeyword   ModifierSyntaxKind = 122
	ModifierSyntaxKindProtectedKeyword ModifierSyntaxKind = 123
	ModifierSyntaxKindPublicKeyword    ModifierSyntaxKind = 124
	ModifierSyntaxKindReadonlyKeyword  ModifierSyntaxKind = 148
	ModifierSyntaxKindOutKeyword       ModifierSyntaxKind = 147
	ModifierSyntaxKindOverrideKeyword  ModifierSyntaxKind = 164
	ModifierSyntaxKindStaticKeyword    ModifierSyntaxKind = 125
)

type MultiplicativeOperator SyntaxKind

const (
	MultiplicativeOperatorAsteriskToken MultiplicativeOperator = 41
	MultiplicativeOperatorSlashToken    MultiplicativeOperator = 43
	MultiplicativeOperatorPercentToken  MultiplicativeOperator = 44
)

type MultiplicativeOperatorOrHigher SyntaxKind

const (
	MultiplicativeOperatorOrHigherAsteriskAsteriskToken MultiplicativeOperatorOrHigher = 42
	MultiplicativeOperatorOrHigherAsteriskToken         MultiplicativeOperatorOrHigher = 41
	MultiplicativeOperatorOrHigherSlashToken            MultiplicativeOperatorOrHigher = 43
	MultiplicativeOperatorOrHigherPercentToken          MultiplicativeOperatorOrHigher = 44
)

type PostfixUnaryOperator SyntaxKind

const (
	PostfixUnaryOperatorPlusPlusToken   PostfixUnaryOperator = 45
	PostfixUnaryOperatorMinusMinusToken PostfixUnaryOperator = 46
)

type PrefixUnaryOperator SyntaxKind

const (
	PrefixUnaryOperatorPlusToken        PrefixUnaryOperator = 39
	PrefixUnaryOperatorMinusToken       PrefixUnaryOperator = 40
	PrefixUnaryOperatorTildeToken       PrefixUnaryOperator = 54
	PrefixUnaryOperatorExclamationToken PrefixUnaryOperator = 53
	PrefixUnaryOperatorPlusPlusToken    PrefixUnaryOperator = 45
	PrefixUnaryOperatorMinusMinusToken  PrefixUnaryOperator = 46
)

type PseudoLiteralSyntaxKind SyntaxKind

const (
	PseudoLiteralSyntaxKindTemplateHead   PseudoLiteralSyntaxKind = 15
	PseudoLiteralSyntaxKindTemplateMiddle PseudoLiteralSyntaxKind = 16
	PseudoLiteralSyntaxKindTemplateTail   PseudoLiteralSyntaxKind = 17
)

type PunctuationSyntaxKind SyntaxKind

const (
	PunctuationSyntaxKindOpenBraceToken                               PunctuationSyntaxKind = 18
	PunctuationSyntaxKindCloseBraceToken                              PunctuationSyntaxKind = 19
	PunctuationSyntaxKindOpenParenToken                               PunctuationSyntaxKind = 20
	PunctuationSyntaxKindCloseParenToken                              PunctuationSyntaxKind = 21
	PunctuationSyntaxKindOpenBracketToken                             PunctuationSyntaxKind = 22
	PunctuationSyntaxKindCloseBracketToken                            PunctuationSyntaxKind = 23
	PunctuationSyntaxKindDotToken                                     PunctuationSyntaxKind = 24
	PunctuationSyntaxKindDotDotDotToken                               PunctuationSyntaxKind = 25
	PunctuationSyntaxKindSemicolonToken                               PunctuationSyntaxKind = 26
	PunctuationSyntaxKindCommaToken                                   PunctuationSyntaxKind = 27
	PunctuationSyntaxKindQuestionDotToken                             PunctuationSyntaxKind = 28
	PunctuationSyntaxKindLessThanToken                                PunctuationSyntaxKind = 29
	PunctuationSyntaxKindLessThanSlashToken                           PunctuationSyntaxKind = 30
	PunctuationSyntaxKindGreaterThanToken                             PunctuationSyntaxKind = 31
	PunctuationSyntaxKindLessThanEqualsToken                          PunctuationSyntaxKind = 32
	PunctuationSyntaxKindGreaterThanEqualsToken                       PunctuationSyntaxKind = 33
	PunctuationSyntaxKindEqualsEqualsToken                            PunctuationSyntaxKind = 34
	PunctuationSyntaxKindExclamationEqualsToken                       PunctuationSyntaxKind = 35
	PunctuationSyntaxKindEqualsEqualsEqualsToken                      PunctuationSyntaxKind = 36
	PunctuationSyntaxKindExclamationEqualsEqualsToken                 PunctuationSyntaxKind = 37
	PunctuationSyntaxKindEqualsGreaterThanToken                       PunctuationSyntaxKind = 38
	PunctuationSyntaxKindPlusToken                                    PunctuationSyntaxKind = 39
	PunctuationSyntaxKindMinusToken                                   PunctuationSyntaxKind = 40
	PunctuationSyntaxKindAsteriskToken                                PunctuationSyntaxKind = 41
	PunctuationSyntaxKindAsteriskAsteriskToken                        PunctuationSyntaxKind = 42
	PunctuationSyntaxKindSlashToken                                   PunctuationSyntaxKind = 43
	PunctuationSyntaxKindPercentToken                                 PunctuationSyntaxKind = 44
	PunctuationSyntaxKindPlusPlusToken                                PunctuationSyntaxKind = 45
	PunctuationSyntaxKindMinusMinusToken                              PunctuationSyntaxKind = 46
	PunctuationSyntaxKindLessThanLessThanToken                        PunctuationSyntaxKind = 47
	PunctuationSyntaxKindGreaterThanGreaterThanToken                  PunctuationSyntaxKind = 48
	PunctuationSyntaxKindGreaterThanGreaterThanGreaterThanToken       PunctuationSyntaxKind = 49
	PunctuationSyntaxKindAmpersandToken                               PunctuationSyntaxKind = 50
	PunctuationSyntaxKindBarToken                                     PunctuationSyntaxKind = 51
	PunctuationSyntaxKindCaretToken                                   PunctuationSyntaxKind = 52
	PunctuationSyntaxKindExclamationToken                             PunctuationSyntaxKind = 53
	PunctuationSyntaxKindTildeToken                                   PunctuationSyntaxKind = 54
	PunctuationSyntaxKindAmpersandAmpersandToken                      PunctuationSyntaxKind = 55
	PunctuationSyntaxKindBarBarToken                                  PunctuationSyntaxKind = 56
	PunctuationSyntaxKindQuestionToken                                PunctuationSyntaxKind = 57
	PunctuationSyntaxKindColonToken                                   PunctuationSyntaxKind = 58
	PunctuationSyntaxKindAtToken                                      PunctuationSyntaxKind = 59
	PunctuationSyntaxKindQuestionQuestionToken                        PunctuationSyntaxKind = 60
	PunctuationSyntaxKindBacktickToken                                PunctuationSyntaxKind = 61
	PunctuationSyntaxKindHashToken                                    PunctuationSyntaxKind = 62
	PunctuationSyntaxKindEqualsToken                                  PunctuationSyntaxKind = 63
	PunctuationSyntaxKindPlusEqualsToken                              PunctuationSyntaxKind = 64
	PunctuationSyntaxKindMinusEqualsToken                             PunctuationSyntaxKind = 65
	PunctuationSyntaxKindAsteriskEqualsToken                          PunctuationSyntaxKind = 66
	PunctuationSyntaxKindAsteriskAsteriskEqualsToken                  PunctuationSyntaxKind = 67
	PunctuationSyntaxKindSlashEqualsToken                             PunctuationSyntaxKind = 68
	PunctuationSyntaxKindPercentEqualsToken                           PunctuationSyntaxKind = 69
	PunctuationSyntaxKindLessThanLessThanEqualsToken                  PunctuationSyntaxKind = 70
	PunctuationSyntaxKindGreaterThanGreaterThanEqualsToken            PunctuationSyntaxKind = 71
	PunctuationSyntaxKindGreaterThanGreaterThanGreaterThanEqualsToken PunctuationSyntaxKind = 72
	PunctuationSyntaxKindAmpersandEqualsToken                         PunctuationSyntaxKind = 73
	PunctuationSyntaxKindBarEqualsToken                               PunctuationSyntaxKind = 74
	PunctuationSyntaxKindBarBarEqualsToken                            PunctuationSyntaxKind = 75
	PunctuationSyntaxKindAmpersandAmpersandEqualsToken                PunctuationSyntaxKind = 76
	PunctuationSyntaxKindQuestionQuestionEqualsToken                  PunctuationSyntaxKind = 77
	PunctuationSyntaxKindCaretEqualsToken                             PunctuationSyntaxKind = 78
)

type RelationalOperator SyntaxKind

const (
	RelationalOperatorLessThanToken          RelationalOperator = 29
	RelationalOperatorLessThanEqualsToken    RelationalOperator = 32
	RelationalOperatorGreaterThanToken       RelationalOperator = 31
	RelationalOperatorGreaterThanEqualsToken RelationalOperator = 33
	RelationalOperatorInstanceOfKeyword      RelationalOperator = 103
	RelationalOperatorInKeyword              RelationalOperator = 102
)

type RelationalOperatorOrHigher SyntaxKind

const (
	RelationalOperatorOrHigherAsteriskAsteriskToken                  RelationalOperatorOrHigher = 42
	RelationalOperatorOrHigherAsteriskToken                          RelationalOperatorOrHigher = 41
	RelationalOperatorOrHigherSlashToken                             RelationalOperatorOrHigher = 43
	RelationalOperatorOrHigherPercentToken                           RelationalOperatorOrHigher = 44
	RelationalOperatorOrHigherPlusToken                              RelationalOperatorOrHigher = 39
	RelationalOperatorOrHigherMinusToken                             RelationalOperatorOrHigher = 40
	RelationalOperatorOrHigherLessThanLessThanToken                  RelationalOperatorOrHigher = 47
	RelationalOperatorOrHigherGreaterThanGreaterThanToken            RelationalOperatorOrHigher = 48
	RelationalOperatorOrHigherGreaterThanGreaterThanGreaterThanToken RelationalOperatorOrHigher = 49
	RelationalOperatorOrHigherLessThanToken                          RelationalOperatorOrHigher = 29
	RelationalOperatorOrHigherLessThanEqualsToken                    RelationalOperatorOrHigher = 32
	RelationalOperatorOrHigherGreaterThanToken                       RelationalOperatorOrHigher = 31
	RelationalOperatorOrHigherGreaterThanEqualsToken                 RelationalOperatorOrHigher = 33
	RelationalOperatorOrHigherInstanceOfKeyword                      RelationalOperatorOrHigher = 103
	RelationalOperatorOrHigherInKeyword                              RelationalOperatorOrHigher = 102
)

type ShiftOperator SyntaxKind

const (
	ShiftOperatorLessThanLessThanToken                  ShiftOperator = 47
	ShiftOperatorGreaterThanGreaterThanToken            ShiftOperator = 48
	ShiftOperatorGreaterThanGreaterThanGreaterThanToken ShiftOperator = 49
)

type ShiftOperatorOrHigher SyntaxKind

const (
	ShiftOperatorOrHigherAsteriskAsteriskToken                  ShiftOperatorOrHigher = 42
	ShiftOperatorOrHigherAsteriskToken                          ShiftOperatorOrHigher = 41
	ShiftOperatorOrHigherSlashToken                             ShiftOperatorOrHigher = 43
	ShiftOperatorOrHigherPercentToken                           ShiftOperatorOrHigher = 44
	ShiftOperatorOrHigherPlusToken                              ShiftOperatorOrHigher = 39
	ShiftOperatorOrHigherMinusToken                             ShiftOperatorOrHigher = 40
	ShiftOperatorOrHigherLessThanLessThanToken                  ShiftOperatorOrHigher = 47
	ShiftOperatorOrHigherGreaterThanGreaterThanToken            ShiftOperatorOrHigher = 48
	ShiftOperatorOrHigherGreaterThanGreaterThanGreaterThanToken ShiftOperatorOrHigher = 49
)

type TokenSyntaxKind SyntaxKind

const (
	TokenSyntaxKindUnknown                                      TokenSyntaxKind = 0
	TokenSyntaxKindEndOfFile                                    TokenSyntaxKind = 1
	TokenSyntaxKindSingleLineCommentTrivia                      TokenSyntaxKind = 2
	TokenSyntaxKindMultiLineCommentTrivia                       TokenSyntaxKind = 3
	TokenSyntaxKindNewLineTrivia                                TokenSyntaxKind = 4
	TokenSyntaxKindWhitespaceTrivia                             TokenSyntaxKind = 5
	TokenSyntaxKindConflictMarkerTrivia                         TokenSyntaxKind = 6
	TokenSyntaxKindNonTextFileMarkerTrivia                      TokenSyntaxKind = 7
	TokenSyntaxKindNumericLiteral                               TokenSyntaxKind = 8
	TokenSyntaxKindBigIntLiteral                                TokenSyntaxKind = 9
	TokenSyntaxKindStringLiteral                                TokenSyntaxKind = 10
	TokenSyntaxKindJsxText                                      TokenSyntaxKind = 11
	TokenSyntaxKindJsxTextAllWhiteSpaces                        TokenSyntaxKind = 12
	TokenSyntaxKindRegularExpressionLiteral                     TokenSyntaxKind = 13
	TokenSyntaxKindNoSubstitutionTemplateLiteral                TokenSyntaxKind = 14
	TokenSyntaxKindTemplateHead                                 TokenSyntaxKind = 15
	TokenSyntaxKindTemplateMiddle                               TokenSyntaxKind = 16
	TokenSyntaxKindTemplateTail                                 TokenSyntaxKind = 17
	TokenSyntaxKindOpenBraceToken                               TokenSyntaxKind = 18
	TokenSyntaxKindCloseBraceToken                              TokenSyntaxKind = 19
	TokenSyntaxKindOpenParenToken                               TokenSyntaxKind = 20
	TokenSyntaxKindCloseParenToken                              TokenSyntaxKind = 21
	TokenSyntaxKindOpenBracketToken                             TokenSyntaxKind = 22
	TokenSyntaxKindCloseBracketToken                            TokenSyntaxKind = 23
	TokenSyntaxKindDotToken                                     TokenSyntaxKind = 24
	TokenSyntaxKindDotDotDotToken                               TokenSyntaxKind = 25
	TokenSyntaxKindSemicolonToken                               TokenSyntaxKind = 26
	TokenSyntaxKindCommaToken                                   TokenSyntaxKind = 27
	TokenSyntaxKindQuestionDotToken                             TokenSyntaxKind = 28
	TokenSyntaxKindLessThanToken                                TokenSyntaxKind = 29
	TokenSyntaxKindLessThanSlashToken                           TokenSyntaxKind = 30
	TokenSyntaxKindGreaterThanToken                             TokenSyntaxKind = 31
	TokenSyntaxKindLessThanEqualsToken                          TokenSyntaxKind = 32
	TokenSyntaxKindGreaterThanEqualsToken                       TokenSyntaxKind = 33
	TokenSyntaxKindEqualsEqualsToken                            TokenSyntaxKind = 34
	TokenSyntaxKindExclamationEqualsToken                       TokenSyntaxKind = 35
	TokenSyntaxKindEqualsEqualsEqualsToken                      TokenSyntaxKind = 36
	TokenSyntaxKindExclamationEqualsEqualsToken                 TokenSyntaxKind = 37
	TokenSyntaxKindEqualsGreaterThanToken                       TokenSyntaxKind = 38
	TokenSyntaxKindPlusToken                                    TokenSyntaxKind = 39
	TokenSyntaxKindMinusToken                                   TokenSyntaxKind = 40
	TokenSyntaxKindAsteriskToken                                TokenSyntaxKind = 41
	TokenSyntaxKindAsteriskAsteriskToken                        TokenSyntaxKind = 42
	TokenSyntaxKindSlashToken                                   TokenSyntaxKind = 43
	TokenSyntaxKindPercentToken                                 TokenSyntaxKind = 44
	TokenSyntaxKindPlusPlusToken                                TokenSyntaxKind = 45
	TokenSyntaxKindMinusMinusToken                              TokenSyntaxKind = 46
	TokenSyntaxKindLessThanLessThanToken                        TokenSyntaxKind = 47
	TokenSyntaxKindGreaterThanGreaterThanToken                  TokenSyntaxKind = 48
	TokenSyntaxKindGreaterThanGreaterThanGreaterThanToken       TokenSyntaxKind = 49
	TokenSyntaxKindAmpersandToken                               TokenSyntaxKind = 50
	TokenSyntaxKindBarToken                                     TokenSyntaxKind = 51
	TokenSyntaxKindCaretToken                                   TokenSyntaxKind = 52
	TokenSyntaxKindExclamationToken                             TokenSyntaxKind = 53
	TokenSyntaxKindTildeToken                                   TokenSyntaxKind = 54
	TokenSyntaxKindAmpersandAmpersandToken                      TokenSyntaxKind = 55
	TokenSyntaxKindBarBarToken                                  TokenSyntaxKind = 56
	TokenSyntaxKindQuestionToken                                TokenSyntaxKind = 57
	TokenSyntaxKindColonToken                                   TokenSyntaxKind = 58
	TokenSyntaxKindAtToken                                      TokenSyntaxKind = 59
	TokenSyntaxKindQuestionQuestionToken                        TokenSyntaxKind = 60
	TokenSyntaxKindBacktickToken                                TokenSyntaxKind = 61
	TokenSyntaxKindHashToken                                    TokenSyntaxKind = 62
	TokenSyntaxKindEqualsToken                                  TokenSyntaxKind = 63
	TokenSyntaxKindPlusEqualsToken                              TokenSyntaxKind = 64
	TokenSyntaxKindMinusEqualsToken                             TokenSyntaxKind = 65
	TokenSyntaxKindAsteriskEqualsToken                          TokenSyntaxKind = 66
	TokenSyntaxKindAsteriskAsteriskEqualsToken                  TokenSyntaxKind = 67
	TokenSyntaxKindSlashEqualsToken                             TokenSyntaxKind = 68
	TokenSyntaxKindPercentEqualsToken                           TokenSyntaxKind = 69
	TokenSyntaxKindLessThanLessThanEqualsToken                  TokenSyntaxKind = 70
	TokenSyntaxKindGreaterThanGreaterThanEqualsToken            TokenSyntaxKind = 71
	TokenSyntaxKindGreaterThanGreaterThanGreaterThanEqualsToken TokenSyntaxKind = 72
	TokenSyntaxKindAmpersandEqualsToken                         TokenSyntaxKind = 73
	TokenSyntaxKindBarEqualsToken                               TokenSyntaxKind = 74
	TokenSyntaxKindBarBarEqualsToken                            TokenSyntaxKind = 75
	TokenSyntaxKindAmpersandAmpersandEqualsToken                TokenSyntaxKind = 76
	TokenSyntaxKindQuestionQuestionEqualsToken                  TokenSyntaxKind = 77
	TokenSyntaxKindCaretEqualsToken                             TokenSyntaxKind = 78
	TokenSyntaxKindIdentifier                                   TokenSyntaxKind = 79
	TokenSyntaxKindPrivateIdentifier                            TokenSyntaxKind = 80
	TokenSyntaxKindJSDocCommentTextToken                        TokenSyntaxKind = 81
	TokenSyntaxKindBreakKeyword                                 TokenSyntaxKind = 82
	TokenSyntaxKindCaseKeyword                                  TokenSyntaxKind = 83
	TokenSyntaxKindCatchKeyword                                 TokenSyntaxKind = 84
	TokenSyntaxKindClassKeyword                                 TokenSyntaxKind = 85
	TokenSyntaxKindConstKeyword                                 TokenSyntaxKind = 86
	TokenSyntaxKindContinueKeyword                              TokenSyntaxKind = 87
	TokenSyntaxKindDebuggerKeyword                              TokenSyntaxKind = 88
	TokenSyntaxKindDefaultKeyword                               TokenSyntaxKind = 89
	TokenSyntaxKindDeleteKeyword                                TokenSyntaxKind = 90
	TokenSyntaxKindDoKeyword                                    TokenSyntaxKind = 91
	TokenSyntaxKindElseKeyword                                  TokenSyntaxKind = 92
	TokenSyntaxKindEnumKeyword                                  TokenSyntaxKind = 93
	TokenSyntaxKindExportKeyword                                TokenSyntaxKind = 94
	TokenSyntaxKindExtendsKeyword                               TokenSyntaxKind = 95
	TokenSyntaxKindFalseKeyword                                 TokenSyntaxKind = 96
	TokenSyntaxKindFinallyKeyword                               TokenSyntaxKind = 97
	TokenSyntaxKindForKeyword                                   TokenSyntaxKind = 98
	TokenSyntaxKindFunctionKeyword                              TokenSyntaxKind = 99
	TokenSyntaxKindIfKeyword                                    TokenSyntaxKind = 100
	TokenSyntaxKindImportKeyword                                TokenSyntaxKind = 101
	TokenSyntaxKindInKeyword                                    TokenSyntaxKind = 102
	TokenSyntaxKindInstanceOfKeyword                            TokenSyntaxKind = 103
	TokenSyntaxKindNewKeyword                                   TokenSyntaxKind = 104
	TokenSyntaxKindNullKeyword                                  TokenSyntaxKind = 105
	TokenSyntaxKindReturnKeyword                                TokenSyntaxKind = 106
	TokenSyntaxKindSuperKeyword                                 TokenSyntaxKind = 107
	TokenSyntaxKindSwitchKeyword                                TokenSyntaxKind = 108
	TokenSyntaxKindThisKeyword                                  TokenSyntaxKind = 109
	TokenSyntaxKindThrowKeyword                                 TokenSyntaxKind = 110
	TokenSyntaxKindTrueKeyword                                  TokenSyntaxKind = 111
	TokenSyntaxKindTryKeyword                                   TokenSyntaxKind = 112
	TokenSyntaxKindTypeOfKeyword                                TokenSyntaxKind = 113
	TokenSyntaxKindVarKeyword                                   TokenSyntaxKind = 114
	TokenSyntaxKindVoidKeyword                                  TokenSyntaxKind = 115
	TokenSyntaxKindWhileKeyword                                 TokenSyntaxKind = 116
	TokenSyntaxKindWithKeyword                                  TokenSyntaxKind = 117
	TokenSyntaxKindImplementsKeyword                            TokenSyntaxKind = 118
	TokenSyntaxKindInterfaceKeyword                             TokenSyntaxKind = 119
	TokenSyntaxKindLetKeyword                                   TokenSyntaxKind = 120
	TokenSyntaxKindPackageKeyword                               TokenSyntaxKind = 121
	TokenSyntaxKindPrivateKeyword                               TokenSyntaxKind = 122
	TokenSyntaxKindProtectedKeyword                             TokenSyntaxKind = 123
	TokenSyntaxKindPublicKeyword                                TokenSyntaxKind = 124
	TokenSyntaxKindStaticKeyword                                TokenSyntaxKind = 125
	TokenSyntaxKindYieldKeyword                                 TokenSyntaxKind = 126
	TokenSyntaxKindAbstractKeyword                              TokenSyntaxKind = 127
	TokenSyntaxKindAccessorKeyword                              TokenSyntaxKind = 128
	TokenSyntaxKindAsKeyword                                    TokenSyntaxKind = 129
	TokenSyntaxKindAssertsKeyword                               TokenSyntaxKind = 130
	TokenSyntaxKindAssertKeyword                                TokenSyntaxKind = 131
	TokenSyntaxKindAnyKeyword                                   TokenSyntaxKind = 132
	TokenSyntaxKindAsyncKeyword                                 TokenSyntaxKind = 133
	TokenSyntaxKindAwaitKeyword                                 TokenSyntaxKind = 134
	TokenSyntaxKindBooleanKeyword                               TokenSyntaxKind = 135
	TokenSyntaxKindConstructorKeyword                           TokenSyntaxKind = 136
	TokenSyntaxKindDeclareKeyword                               TokenSyntaxKind = 137
	TokenSyntaxKindGetKeyword                                   TokenSyntaxKind = 138
	TokenSyntaxKindImmediateKeyword                             TokenSyntaxKind = 139
	TokenSyntaxKindInferKeyword                                 TokenSyntaxKind = 140
	TokenSyntaxKindIntrinsicKeyword                             TokenSyntaxKind = 141
	TokenSyntaxKindIsKeyword                                    TokenSyntaxKind = 142
	TokenSyntaxKindKeyOfKeyword                                 TokenSyntaxKind = 143
	TokenSyntaxKindModuleKeyword                                TokenSyntaxKind = 144
	TokenSyntaxKindNamespaceKeyword                             TokenSyntaxKind = 145
	TokenSyntaxKindNeverKeyword                                 TokenSyntaxKind = 146
	TokenSyntaxKindOutKeyword                                   TokenSyntaxKind = 147
	TokenSyntaxKindReadonlyKeyword                              TokenSyntaxKind = 148
	TokenSyntaxKindRequireKeyword                               TokenSyntaxKind = 149
	TokenSyntaxKindNumberKeyword                                TokenSyntaxKind = 150
	TokenSyntaxKindObjectKeyword                                TokenSyntaxKind = 151
	TokenSyntaxKindSatisfiesKeyword                             TokenSyntaxKind = 152
	TokenSyntaxKindSetKeyword                                   TokenSyntaxKind = 153
	TokenSyntaxKindStringKeyword                                TokenSyntaxKind = 154
	TokenSyntaxKindSymbolKeyword                                TokenSyntaxKind = 155
	TokenSyntaxKindTypeKeyword                                  TokenSyntaxKind = 156
	TokenSyntaxKindUndefinedKeyword                             TokenSyntaxKind = 157
	TokenSyntaxKindUniqueKeyword                                TokenSyntaxKind = 158
	TokenSyntaxKindUnknownKeyword                               TokenSyntaxKind = 159
	TokenSyntaxKindUsingKeyword                                 TokenSyntaxKind = 160
	TokenSyntaxKindFromKeyword                                  TokenSyntaxKind = 161
	TokenSyntaxKindGlobalKeyword                                TokenSyntaxKind = 162
	TokenSyntaxKindBigIntKeyword                                TokenSyntaxKind = 163
	TokenSyntaxKindOverrideKeyword                              TokenSyntaxKind = 164
	TokenSyntaxKindOfKeyword                                    TokenSyntaxKind = 165
	TokenSyntaxKindDeferKeyword                                 TokenSyntaxKind = 166
)

type TriviaSyntaxKind SyntaxKind

const (
	TriviaSyntaxKindSingleLineCommentTrivia TriviaSyntaxKind = 2
	TriviaSyntaxKindMultiLineCommentTrivia  TriviaSyntaxKind = 3
	TriviaSyntaxKindNewLineTrivia           TriviaSyntaxKind = 4
	TriviaSyntaxKindWhitespaceTrivia        TriviaSyntaxKind = 5
	TriviaSyntaxKindConflictMarkerTrivia    TriviaSyntaxKind = 6
)

func RequiresBindingIdentifierEscape(text string) bool {
	switch text {
	case
		"break",
		"case",
		"catch",
		"class",
		"const",
		"continue",
		"debugger",
		"default",
		"delete",
		"do",
		"else",
		"enum",
		"export",
		"extends",
		"false",
		"finally",
		"for",
		"function",
		"if",
		"import",
		"in",
		"instanceof",
		"new",
		"null",
		"return",
		"super",
		"switch",
		"this",
		"throw",
		"true",
		"try",
		"typeof",
		"var",
		"void",
		"while",
		"with",
		"implements",
		"interface",
		"let",
		"package",
		"private",
		"protected",
		"public",
		"static",
		"yield",
		"abstract",
		"accessor",
		"as",
		"asserts",
		"assert",
		"any",
		"async",
		"await",
		"boolean",
		"constructor",
		"declare",
		"get",
		"immediate",
		"infer",
		"intrinsic",
		"is",
		"keyof",
		"module",
		"namespace",
		"never",
		"out",
		"readonly",
		"require",
		"number",
		"object",
		"satisfies",
		"set",
		"string",
		"symbol",
		"type",
		"undefined",
		"unique",
		"unknown",
		"using",
		"from",
		"global",
		"bigint",
		"override",
		"of",
		"defer",
		"arguments",
		"eval":
		return true
	default:
		return false
	}
}
