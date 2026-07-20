// Hand port D2 — Program.verifyCompilerOptions (22.7 KB). The
// closure-heavy validation logic ports statement-for-statement: Go
// memoized closures become memoized arrows, variadic args become rest
// parameters, Tristate accessors stay method calls, the paths map
// iterates its entries, and the emit-collision Set is a native Set.
// Receiver methods keep method syntax; local closures capture `this`
// through arrows. Names mirror the Go source per corpus convention.
verifyCompilerOptions(): void {
  const options = this.Options();

  const sourceFile = Memoize((): SourceFile | undefined => {
    const configFile = this.opts.Config.ConfigFile;
    if (configFile === undefined) {
      return undefined;
    }
    return configFile.SourceFile;
  });

  const configFilePath = Memoize((): string => {
    const file = sourceFile();
    if (file !== undefined) {
      return file.FileName();
    }
    return "";
  });

  const getCompilerOptionsPropertySyntax = Memoize((): PropertyAssignment | undefined => {
    return ForEachTsConfigPropArray(sourceFile(), "compilerOptions", Identity);
  });

  const getCompilerOptionsObjectLiteralSyntax = Memoize((): ObjectLiteralExpression | undefined => {
    const compilerOptionsProperty = getCompilerOptionsPropertySyntax();
    if (compilerOptionsProperty !== undefined &&
      compilerOptionsProperty.Initializer !== undefined &&
      IsObjectLiteralExpression(compilerOptionsProperty.Initializer)) {
      return compilerOptionsProperty.Initializer.AsObjectLiteralExpression();
    }
    return undefined;
  });

  const createOptionDiagnosticInObjectLiteralSyntax = (objectLiteral: ObjectLiteralExpression | undefined, onKey: boolean, key1: string, key2: string, message: Message, ...args: unknown[]): Diagnostic | undefined => {
    const diag = ForEachPropertyAssignment(objectLiteral, key1, (property: PropertyAssignment): Diagnostic | undefined => {
      return CreateDiagnosticForNodeInSourceFile(sourceFile(), onKey ? property.Name() : property.Initializer, message, ...args);
    }, key2);
    if (diag !== undefined) {
      this.programDiagnostics.push(diag);
    }
    return diag;
  };

  const createCompilerOptionsDiagnostic = (message: Message, ...args: unknown[]): Diagnostic => {
    const compilerOptionsProperty = getCompilerOptionsPropertySyntax();
    let diag: Diagnostic;
    if (compilerOptionsProperty !== undefined) {
      diag = CreateDiagnosticForNodeInSourceFile(sourceFile(), compilerOptionsProperty.Name(), message, ...args);
    } else {
      diag = NewCompilerDiagnostic(message, ...args);
    }
    this.programDiagnostics.push(diag);
    return diag;
  };

  const createDiagnosticForOption = (onKey: boolean, option1: string, option2: string, message: Message, ...args: unknown[]): Diagnostic => {
    let diag = createOptionDiagnosticInObjectLiteralSyntax(getCompilerOptionsObjectLiteralSyntax(), onKey, option1, option2, message, ...args);
    if (diag === undefined) {
      diag = createCompilerOptionsDiagnostic(message, ...args);
    }
    return diag;
  };

  const createDiagnosticForOptionName = (message: Message, option1: string, option2: string, ...args: unknown[]): void => {
    createDiagnosticForOption(true /*onKey*/, option1, option2, message, option1, option2, ...args);
  };

  const createOptionValueDiagnostic = (option1: string, message: Message, ...args: unknown[]): void => {
    createDiagnosticForOption(false /*onKey*/, option1, "", message, ...args);
  };

  const createRemovedOptionDiagnostic = (name: string, value: string, useInstead: string): void => {
    let message: Message;
    let args: unknown[];
    if (value === "") {
      message = Option_0_has_been_removed_Please_remove_it_from_your_configuration;
      args = [name];
    } else {
      message = Option_0_1_has_been_removed_Please_remove_it_from_your_configuration;
      args = [name, value];
    }

    const diag = createDiagnosticForOption(value === "", name, "", message, ...args);
    if (useInstead !== "") {
      diag.AddMessageChain(NewCompilerDiagnostic(Use_0_instead, useInstead));
    }
  };

  // Removed in TS7

  if (options.BaseUrl !== "") {
    // BaseUrl will have been turned absolute by this point.
    let useInstead = "";
    if (configFilePath() !== "") {
      let relative = GetRelativePathFromFile(configFilePath(), options.BaseUrl, this.comparePathsOptions);
      if (!(relative.startsWith("./") || relative.startsWith("../"))) {
        relative = "./" + relative;
      }
      const suggestion = CombinePaths(relative, "*");
      useInstead = `"paths": {"*": [${JSON.stringify(suggestion)}]}`;
    }
    createRemovedOptionDiagnostic("baseUrl", "", useInstead);
  }

  if (options.OutFile !== "") {
    createRemovedOptionDiagnostic("outFile", "", "");
  }

  if (options.Target === ScriptTargetES5) {
    createRemovedOptionDiagnostic("target", "ES5", "");
  }

  if (options.Module === ModuleKindAMD) {
    createRemovedOptionDiagnostic("module", "AMD", "");
  }
  if (options.Module === ModuleKindSystem) {
    createRemovedOptionDiagnostic("module", "System", "");
  }
  if (options.Module === ModuleKindUMD) {
    createRemovedOptionDiagnostic("module", "UMD", "");
  }

  if (options.ModuleResolution === ModuleResolutionKindClassic) {
    createRemovedOptionDiagnostic("moduleResolution", "Classic", "");
  }

  if (options.AlwaysStrict.IsFalse()) {
    createRemovedOptionDiagnostic("alwaysStrict", "false", "");
  }

  if (options.ESModuleInterop.IsFalse()) {
    createRemovedOptionDiagnostic("esModuleInterop", "false", "");
  }

  if (options.AllowSyntheticDefaultImports.IsFalse()) {
    createRemovedOptionDiagnostic("allowSyntheticDefaultImports", "false", "");
  }

  if (options.ModuleResolution === ModuleResolutionKindNode10) {
    createRemovedOptionDiagnostic("moduleResolution", "node10", "");
  }

  if (!options.DownlevelIteration.IsUnknown()) {
    createRemovedOptionDiagnostic("downlevelIteration", "", "");
  }

  if (options.StrictPropertyInitialization.IsTrue() && !options.GetStrictOptionValue(options.StrictNullChecks)) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1, "strictPropertyInitialization", "strictNullChecks");
  }
  if (options.ExactOptionalPropertyTypes.IsTrue() && !options.GetStrictOptionValue(options.StrictNullChecks)) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1, "exactOptionalPropertyTypes", "strictNullChecks");
  }

  if (options.IsolatedDeclarations.IsTrue()) {
    if (options.GetAllowJS()) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_with_option_1, "allowJs", "isolatedDeclarations");
    }
    if (!options.GetEmitDeclarations()) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1_or_option_2, "isolatedDeclarations", "declaration", "composite");
    }
  }

  if (options.InlineSourceMap.IsTrue()) {
    if (options.SourceMap.IsTrue()) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_with_option_1, "sourceMap", "inlineSourceMap");
    }
    if (options.MapRoot !== "") {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_with_option_1, "mapRoot", "inlineSourceMap");
    }
  }

  if (options.Composite.IsTrue()) {
    if (options.Declaration.IsFalse()) {
      createDiagnosticForOptionName(Composite_projects_may_not_disable_declaration_emit, "declaration", "");
    }
    if (options.Incremental.IsFalse()) {
      createDiagnosticForOptionName(Composite_projects_may_not_disable_incremental_compilation, "declaration", "");
    }
  }

  if (options.TsBuildInfoFile === "" && options.Incremental.IsTrue() && options.ConfigFilePath === "") {
    createCompilerOptionsDiagnostic(Option_incremental_is_only_valid_with_a_known_configuration_file_like_tsconfig_json_or_when_tsBuildInfoFile_is_explicitly_provided);
  }

  this.verifyProjectReferences();

  if (options.Composite.IsTrue()) {
    const rootPaths = new Set<Path>();
    for (const fileName of this.opts.Config.FileNames()) {
      rootPaths.add(this.toPath(fileName));
    }

    for (const file of this.files) {
      if (sourceFileMayBeEmitted(file, this, false) && !rootPaths.has(file.Path())) {
        this.includeProcessor.addProcessingDiagnostic({
          kind: processingDiagnosticKindExplainingFileInclude,
          data: {
            file: file.Path(),
            message: File_0_is_not_listed_within_the_file_list_of_project_1_Projects_must_list_all_files_or_use_an_include_pattern,
            args: [file.FileName(), configFilePath()],
          },
        });
      }
    }
  }

  const forEachOptionPathsSyntax = (callback: (property: PropertyAssignment) => Diagnostic | undefined): Diagnostic | undefined => {
    return ForEachPropertyAssignment(getCompilerOptionsObjectLiteralSyntax(), "paths", callback);
  };

  const createDiagnosticForOptionPaths = (onKey: boolean, key: string, message: Message, ...args: unknown[]): Diagnostic => {
    let diag = forEachOptionPathsSyntax((pathProp: PropertyAssignment): Diagnostic | undefined => {
      if (IsObjectLiteralExpression(pathProp.Initializer)) {
        return createOptionDiagnosticInObjectLiteralSyntax(pathProp.Initializer.AsObjectLiteralExpression(), onKey, key, "", message, ...args);
      }
      return undefined;
    });
    if (diag === undefined) {
      diag = createCompilerOptionsDiagnostic(message, ...args);
    }
    return diag;
  };

  const createDiagnosticForOptionPathKeyValue = (key: string, valueIndex: number, message: Message, ...args: unknown[]): Diagnostic => {
    let diag = forEachOptionPathsSyntax((pathProp: PropertyAssignment): Diagnostic | undefined => {
      if (IsObjectLiteralExpression(pathProp.Initializer)) {
        return ForEachPropertyAssignment(pathProp.Initializer.AsObjectLiteralExpression(), key, (keyProps: PropertyAssignment): Diagnostic | undefined => {
          const initializer = keyProps.Initializer;
          if (IsArrayLiteralExpression(initializer)) {
            const elements = initializer.ElementList();
            if (elements !== undefined && elements.Nodes.length > valueIndex) {
              const diag = CreateDiagnosticForNodeInSourceFile(sourceFile(), elements.Nodes[valueIndex], message, ...args);
              this.programDiagnostics.push(diag);
              return diag;
            }
          }
          return undefined;
        });
      }
      return undefined;
    });
    if (diag === undefined) {
      diag = createCompilerOptionsDiagnostic(message, ...args);
    }
    return diag;
  };

  for (const [key, value] of options.Paths.Entries()) {
    // !!! This code does not handle cases where where the path mappings have the wrong types,
    // as that information is mostly lost during the parsing process.
    if (!hasZeroOrOneAsteriskCharacter(key)) {
      createDiagnosticForOptionPaths(true /*onKey*/, key, Pattern_0_can_have_at_most_one_Asterisk_character, key);
    }
    if (value === undefined) {
      createDiagnosticForOptionPaths(false /*onKey*/, key, Substitutions_for_pattern_0_should_be_an_array, key);
    } else if (value.length === 0) {
      createDiagnosticForOptionPaths(false /*onKey*/, key, Substitutions_for_pattern_0_shouldn_t_be_an_empty_array, key);
    }
    for (const [i, subst] of (value ?? []).entries()) {
      if (!hasZeroOrOneAsteriskCharacter(subst)) {
        createDiagnosticForOptionPathKeyValue(key, i, Substitution_0_in_pattern_1_can_have_at_most_one_Asterisk_character, subst, key);
      }
      if (!PathIsRelative(subst) && !PathIsAbsolute(subst)) {
        createDiagnosticForOptionPathKeyValue(key, i, Non_relative_paths_are_not_allowed_Did_you_forget_a_leading_Slash);
      }
    }
  }

  if (options.SourceMap.IsFalseOrUnknown() && options.InlineSourceMap.IsFalseOrUnknown()) {
    if (options.InlineSources.IsTrue()) {
      createDiagnosticForOptionName(Option_0_can_only_be_used_when_either_option_inlineSourceMap_or_option_sourceMap_is_provided, "inlineSources", "");
    }
    if (options.SourceRoot !== "") {
      createDiagnosticForOptionName(Option_0_can_only_be_used_when_either_option_inlineSourceMap_or_option_sourceMap_is_provided, "sourceRoot", "");
    }
  }

  if (options.MapRoot !== "" && !(options.SourceMap.IsTrue() || options.DeclarationMap.IsTrue())) {
    // Error to specify --mapRoot without --sourcemap
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1_or_option_2, "mapRoot", "sourceMap", "declarationMap");
  }

  if (options.DeclarationDir !== "") {
    if (!options.GetEmitDeclarations()) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1_or_option_2, "declarationDir", "declaration", "composite");
    }
  }

  if (options.DeclarationMap.IsTrue() && !options.GetEmitDeclarations()) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1_or_option_2, "declarationMap", "declaration", "composite");
  }

  if (options.Lib !== undefined && options.NoLib.IsTrue()) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_with_option_1, "lib", "noLib");
  }

  if (options.IsolatedModules.IsTrue() || options.VerbatimModuleSyntax.IsTrue()) {
    if (options.PreserveConstEnums.IsFalse()) {
      createDiagnosticForOptionName(Option_preserveConstEnums_cannot_be_disabled_when_0_is_enabled, options.VerbatimModuleSyntax.IsTrue() ? "verbatimModuleSyntax" : "isolatedModules", "preserveConstEnums");
    }
  }

  if (options.OutDir !== "" ||
    options.RootDir !== "" ||
    options.SourceRoot !== "" ||
    options.MapRoot !== "" ||
    (options.GetEmitDeclarations() && options.DeclarationDir !== "")) {
    // !!! sheetal checkSourceFilesBelongToPath - for root Dir and configFile - explaining why file is in the program
    const dir = this.CommonSourceDirectory();
    if (options.OutDir !== "" && dir === "" && this.files.some((f) => GetRootLength(f.FileName()) > 1)) {
      createDiagnosticForOptionName(Cannot_find_the_common_subdirectory_path_for_the_input_files, "outDir", "");
    }
  }

  if (!options.NoEmit.IsTrue() &&
    !options.Composite.IsTrue() &&
    options.RootDir === "" &&
    options.ConfigFilePath !== "" &&
    (options.OutDir !== "" ||
      (options.GetEmitDeclarations() && options.DeclarationDir !== "") ||
      options.OutFile !== "")) {
    // Check if rootDir inferred changed and issue diagnostic
    const dir = this.CommonSourceDirectory();
    const emittedFiles: string[] = [];
    for (const file of this.files) {
      if (!file.IsDeclarationFile && sourceFileMayBeEmitted(file, this, false)) {
        emittedFiles.push(file.FileName());
      }
    }
    const dir59 = GetComputedCommonSourceDirectory(emittedFiles, this.GetCurrentDirectory(), this.UseCaseSensitiveFileNames());
    if (dir59 !== "" && GetCanonicalFileName(dir, this.UseCaseSensitiveFileNames()) !== GetCanonicalFileName(dir59, this.UseCaseSensitiveFileNames())) {
      // change in layout
      let option1: string;
      if (options.OutFile !== "") {
        option1 = "outFile";
      } else if (options.OutDir !== "") {
        option1 = "outDir";
      } else {
        option1 = "declarationDir";
      }
      let option2 = "";
      if (options.OutFile === "" && options.OutDir !== "") {
        option2 = "declarationDir";
      }
      const diag = createDiagnosticForOption(
        true, /*onKey*/
        option1,
        option2,
        The_common_source_directory_of_0_is_1_The_rootDir_setting_must_be_explicitly_set_to_this_or_another_path_to_adjust_your_output_s_file_layout,
        GetBaseFileName(options.ConfigFilePath),
        GetRelativePathFromFile(options.ConfigFilePath, dir59, this.comparePathsOptions),
      );
      diag.AddMessageChain(NewCompilerDiagnostic(Visit_https_Colon_Slash_Slashaka_ms_Slashts6_for_migration_information));
    }
  }

  if (options.CheckJs.IsTrue() && !options.GetAllowJS()) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1, "checkJs", "allowJs");
  }

  if (options.EmitDeclarationOnly.IsTrue()) {
    if (!options.GetEmitDeclarations()) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1_or_option_2, "emitDeclarationOnly", "declaration", "composite");
    }
  }

  if (options.EmitDecoratorMetadata.IsTrue() && options.ExperimentalDecorators.IsFalseOrUnknown()) {
    createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1, "emitDecoratorMetadata", "experimentalDecorators");
  }

  if (options.JsxFactory !== "") {
    if (options.ReactNamespace !== "") {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_with_option_1, "reactNamespace", "jsxFactory");
    }
    if (options.Jsx === JsxEmitReactJSX || options.Jsx === JsxEmitReactJSXDev) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_when_option_jsx_is_1, "jsxFactory", options.Jsx.String());
    }
    if (ParseIsolatedEntityName(options.JsxFactory) === undefined) {
      createOptionValueDiagnostic("jsxFactory", Invalid_value_for_jsxFactory_0_is_not_a_valid_identifier_or_qualified_name, options.JsxFactory);
    }
  } else if (options.ReactNamespace !== "" && !IsIdentifierText(options.ReactNamespace, LanguageVariantStandard)) {
    createOptionValueDiagnostic("reactNamespace", Invalid_value_for_reactNamespace_0_is_not_a_valid_identifier, options.ReactNamespace);
  }

  if (options.JsxFragmentFactory !== "") {
    if (options.JsxFactory === "") {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_without_specifying_option_1, "jsxFragmentFactory", "jsxFactory");
    }
    if (options.Jsx === JsxEmitReactJSX || options.Jsx === JsxEmitReactJSXDev) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_when_option_jsx_is_1, "jsxFragmentFactory", options.Jsx.String());
    }
    if (ParseIsolatedEntityName(options.JsxFragmentFactory) === undefined) {
      createOptionValueDiagnostic("jsxFragmentFactory", Invalid_value_for_jsxFragmentFactory_0_is_not_a_valid_identifier_or_qualified_name, options.JsxFragmentFactory);
    }
  }

  if (options.ReactNamespace !== "") {
    if (options.Jsx === JsxEmitReactJSX || options.Jsx === JsxEmitReactJSXDev) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_when_option_jsx_is_1, "reactNamespace", options.Jsx.String());
    }
  }

  if (options.JsxImportSource !== "") {
    if (options.Jsx === JsxEmitReact) {
      createDiagnosticForOptionName(Option_0_cannot_be_specified_when_option_jsx_is_1, "jsxImportSource", options.Jsx.String());
    }
  }

  const moduleKind = options.GetEmitModuleKind();

  if (options.AllowImportingTsExtensions.IsTrue() && !(options.NoEmit.IsTrue() || options.EmitDeclarationOnly.IsTrue() || options.RewriteRelativeImportExtensions.IsTrue())) {
    createOptionValueDiagnostic("allowImportingTsExtensions", Option_allowImportingTsExtensions_can_only_be_used_when_one_of_noEmit_emitDeclarationOnly_or_rewriteRelativeImportExtensions_is_set);
  }

  const moduleResolution = options.GetModuleResolutionKind();
  if (options.ResolvePackageJsonExports.IsTrue() && !moduleResolutionSupportsPackageJsonExportsAndImports(moduleResolution)) {
    createDiagnosticForOptionName(Option_0_can_only_be_used_when_moduleResolution_is_set_to_node16_nodenext_or_bundler, "resolvePackageJsonExports", "");
  }
  if (options.ResolvePackageJsonImports.IsTrue() && !moduleResolutionSupportsPackageJsonExportsAndImports(moduleResolution)) {
    createDiagnosticForOptionName(Option_0_can_only_be_used_when_moduleResolution_is_set_to_node16_nodenext_or_bundler, "resolvePackageJsonImports", "");
  }
  if (options.CustomConditions !== undefined && !moduleResolutionSupportsPackageJsonExportsAndImports(moduleResolution)) {
    createDiagnosticForOptionName(Option_0_can_only_be_used_when_moduleResolution_is_set_to_node16_nodenext_or_bundler, "customConditions", "");
  }

  if (moduleResolution === ModuleResolutionKindBundler && !emitModuleKindIsNonNodeESM(moduleKind) && moduleKind !== ModuleKindPreserve && moduleKind !== ModuleKindCommonJS) {
    createOptionValueDiagnostic("moduleResolution", Option_0_can_only_be_used_when_module_is_set_to_preserve_commonjs_or_es2015_or_later, "bundler");
  }

  if (ModuleKindNode16 <= moduleKind && moduleKind <= ModuleKindNodeNext &&
    !(ModuleResolutionKindNode16 <= moduleResolution && moduleResolution <= ModuleResolutionKindNodeNext)) {
    const moduleKindName = moduleKind.String();
    let moduleResolutionName: string;
    const v = ModuleKindToModuleResolutionKind.get(moduleKind);
    if (v !== undefined) {
      moduleResolutionName = v.String();
    } else {
      moduleResolutionName = "Node16";
    }
    createOptionValueDiagnostic("moduleResolution", Option_moduleResolution_must_be_set_to_0_or_left_unspecified_when_option_module_is_set_to_1, moduleResolutionName, moduleKindName);
  } else if (ModuleResolutionKindNode16 <= moduleResolution && moduleResolution <= ModuleResolutionKindNodeNext &&
    !(ModuleKindNode16 <= moduleKind && moduleKind <= ModuleKindNodeNext)) {
    const moduleResolutionName = moduleResolution.String();
    createOptionValueDiagnostic("module", Option_module_must_be_set_to_0_when_option_moduleResolution_is_set_to_1, moduleResolutionName, moduleResolutionName);
  }

  // !!! The below needs filesByName, which is not equivalent to p.filesByPath.

  // If the emit is enabled make sure that every output file is unique and not overwriting any of the input files
  if (!options.NoEmit.IsTrue() && !options.SuppressOutputPathCheck.IsTrue()) {
    const emitFilesSeen = new Set<string>();

    // Verify that all the emit files are unique and don't overwrite input files
    const verifyEmitFilePath = (emitFileName: string): void => {
      if (emitFileName !== "") {
        const emitFilePath = this.toPath(emitFileName);
        // Report error if the output overwrites input file
        if (this.filesByPath.has(emitFilePath)) {
          const diag = NewCompilerDiagnostic(Cannot_write_file_0_because_it_would_overwrite_input_file, emitFileName);
          if (configFilePath() === "") {
            // The program is from either an inferred project or an external project
            diag.AddMessageChain(NewCompilerDiagnostic(Adding_a_tsconfig_json_file_will_help_organize_projects_that_contain_both_TypeScript_and_JavaScript_files_Learn_more_at_https_Colon_Slash_Slashaka_ms_Slashtsconfig));
          }
          this.blockEmittingOfFile(emitFileName, diag);
        }

        let emitFileKey: string;
        if (!this.Host().FS().UseCaseSensitiveFileNames()) {
          emitFileKey = ToFileNameLowerCase(emitFilePath);
        } else {
          emitFileKey = emitFilePath;
        }

        // Report error if multiple files write into same file
        if (emitFilesSeen.has(emitFileKey)) {
          // Already seen the same emit file - report error
          this.blockEmittingOfFile(emitFileName, NewCompilerDiagnostic(Cannot_write_file_0_because_it_would_be_overwritten_by_multiple_input_files, emitFileName));
        } else {
          emitFilesSeen.add(emitFileKey);
        }
      }
    };

    ForEachEmittedFile(this, options, (emitFileNames: OutputPaths, sourceFile: SourceFile): boolean => {
      verifyEmitFilePath(emitFileNames.JsFilePath());
      verifyEmitFilePath(emitFileNames.SourceMapFilePath());
      verifyEmitFilePath(emitFileNames.DeclarationFilePath());
      verifyEmitFilePath(emitFileNames.DeclarationMapPath());
      return false;
    }, this.getSourceFilesToEmit(undefined, false), false);
    verifyEmitFilePath(this.opts.Config.GetBuildInfoFileName());
  }
}
