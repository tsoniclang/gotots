// Hand port D5 — Relater.structuredTypeRelatedToWorker (36.7 KB), the
// relation engine's structural core. Go's tagless switches become
// if/else-if chains, the two-result closure returns a tuple, Ternary
// stays a number so `&=` carries over, if-with-initializer statements
// hoist their const, and core.Some/IfElse/OrElse become native
// some/ternary/??. Names mirror the Go source per corpus convention.
structuredTypeRelatedToWorker(source: Type, target: Type, reportErrors: boolean, intersectionState: IntersectionState): Ternary {
  let result: Ternary = TernaryFalse;
  let varianceCheckFailed = false;
  let originalErrorChain: ErrorChain | undefined;
  const saveErrorState = this.getErrorState();
  const relateVariances = (sourceTypeArguments: Type[], targetTypeArguments: Type[], variances: VarianceFlags[], intersectionState: IntersectionState): [Ternary, boolean] => {
    if ((result = this.typeArgumentsRelatedTo(sourceTypeArguments, targetTypeArguments, variances, reportErrors, intersectionState)) !== TernaryFalse) {
      return [result, true];
    }
    if (variances.some((v) => (v & VarianceFlagsAllowsStructuralFallback) !== 0)) {
      // If some type parameter was `Unmeasurable` or `Unreliable`, and we couldn't pass by assuming it was identical, then we
      // have to allow a structural fallback check
      // We elide the variance-based error elaborations, since those might not be too helpful, since we'll potentially
      // be assuming identity of the type parameter.
      originalErrorChain = undefined;
      this.restoreErrorState(saveErrorState);
      return [TernaryFalse, false];
    }
    const allowStructuralFallback = this.c.hasCovariantVoidArgument(targetTypeArguments, variances);
    varianceCheckFailed = !allowStructuralFallback;
    // The type arguments did not relate appropriately, but it may be because we have no variance
    // information (in which case typeArgumentsRelatedTo defaulted to covariance for all type
    // arguments). It might also be the case that the target type has a 'void' type argument for
    // a covariant type parameter that is only used in return positions within the generic type
    // (in which case any type argument is permitted on the source side). In those cases we proceed
    // with a structural comparison. Otherwise, we know for certain the instantiations aren't
    // related and we can return here.
    if (variances.length !== 0 && !allowStructuralFallback) {
      // In some cases generic types that are covariant in regular type checking mode become
      // invariant in --strictFunctionTypes mode because one or more type parameters are used in
      // both co- and contravariant positions. In order to make it easier to diagnose *why* such
      // types are invariant, if any of the type parameters are invariant we reset the reported
      // errors and instead force a structural comparison (which will include elaborations that
      // reveal the reason).
      // We can switch on `reportErrors` here, since varianceCheckFailed guarantees we return `False`,
      // we can return `False` early here to skip calculating the structural error message we don't need.
      if (varianceCheckFailed && !(reportErrors && variances.some((v) => (v & VarianceFlagsVarianceMask) === VarianceFlagsInvariant))) {
        return [TernaryFalse, true];
      }
      // We remember the original error information so we can restore it in case the structural
      // comparison unexpectedly succeeds. This can happen when the structural comparison result
      // is a Ternary.Maybe for example caused by the recursion depth limiter.
      originalErrorChain = this.errorChain;
      this.restoreErrorState(saveErrorState);
    }
    return [TernaryFalse, false];
  };
  if (this.relation === this.c.identityRelation) {
    // We've already checked that source.flags and target.flags are identical
    if ((source.flags & TypeFlagsUnionOrIntersection) !== 0) {
      let result = this.eachTypeRelatedToSomeType(source, target);
      if (result !== TernaryFalse) {
        result &= this.eachTypeRelatedToSomeType(target, source);
      }
      return result;
    } else if ((source.flags & TypeFlagsIndex) !== 0) {
      return this.isRelatedTo(source.Target(), target.Target(), RecursionFlagsBoth, false /*reportErrors*/);
    } else if ((source.flags & TypeFlagsIndexedAccess) !== 0) {
      result = this.isRelatedTo(source.AsIndexedAccessType().objectType, target.AsIndexedAccessType().objectType, RecursionFlagsBoth, false /*reportErrors*/);
      if (result !== TernaryFalse) {
        result &= this.isRelatedTo(source.AsIndexedAccessType().indexType, target.AsIndexedAccessType().indexType, RecursionFlagsBoth, false /*reportErrors*/);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    } else if ((source.flags & TypeFlagsConditional) !== 0) {
      if (source.AsConditionalType().root.isDistributive === target.AsConditionalType().root.isDistributive) {
        result = this.isRelatedTo(source.AsConditionalType().checkType, target.AsConditionalType().checkType, RecursionFlagsBoth, false /*reportErrors*/);
        if (result !== TernaryFalse) {
          result &= this.isRelatedTo(source.AsConditionalType().extendsType, target.AsConditionalType().extendsType, RecursionFlagsBoth, false /*reportErrors*/);
          if (result !== TernaryFalse) {
            result &= this.isRelatedTo(this.c.getTrueTypeFromConditionalType(source), this.c.getTrueTypeFromConditionalType(target), RecursionFlagsBoth, false /*reportErrors*/);
            if (result !== TernaryFalse) {
              result &= this.isRelatedTo(this.c.getFalseTypeFromConditionalType(source), this.c.getFalseTypeFromConditionalType(target), RecursionFlagsBoth, false /*reportErrors*/);
              if (result !== TernaryFalse) {
                return result;
              }
            }
          }
        }
      }
    } else if ((source.flags & TypeFlagsSubstitution) !== 0) {
      result = this.isRelatedTo(source.AsSubstitutionType().baseType, target.AsSubstitutionType().baseType, RecursionFlagsBoth, false /*reportErrors*/);
      if (result !== TernaryFalse) {
        result &= this.isRelatedTo(source.AsSubstitutionType().constraint, target.AsSubstitutionType().constraint, RecursionFlagsBoth, false /*reportErrors*/);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    } else if ((source.flags & TypeFlagsTemplateLiteral) !== 0) {
      const sourceTexts = source.AsTemplateLiteralType().texts;
      const targetTexts = target.AsTemplateLiteralType().texts;
      if (sourceTexts.length === targetTexts.length && sourceTexts.every((t, i) => t === targetTexts[i])) {
        result = TernaryTrue;
        for (const [i, sourceType] of source.AsTemplateLiteralType().types.entries()) {
          const targetType = target.AsTemplateLiteralType().types[i];
          result &= this.isRelatedTo(sourceType, targetType, RecursionFlagsBoth, false /*reportErrors*/);
          if (result === TernaryFalse) {
            return result;
          }
        }
        return result;
      }
    } else if ((source.flags & TypeFlagsStringMapping) !== 0) {
      if (source.AsStringMappingType().Symbol() === target.AsStringMappingType().Symbol()) {
        return this.isRelatedTo(source.AsStringMappingType().target, target.AsStringMappingType().target, RecursionFlagsBoth, false /*reportErrors*/);
      }
    }
    if ((source.flags & TypeFlagsObject) === 0) {
      return TernaryFalse;
    }
  } else if ((source.flags & TypeFlagsUnionOrIntersection) !== 0 || (target.flags & TypeFlagsUnionOrIntersection) !== 0) {
    result = this.unionOrIntersectionRelatedTo(source, target, reportErrors, intersectionState);
    if (result !== TernaryFalse) {
      return result;
    }
    // The ordered decomposition above doesn't handle all cases. Specifically, we also need to handle:
    // Source is instantiable (e.g. source has union or intersection constraint).
    // Source is an object, target is a union (e.g. { a, b: boolean } <=> { a, b: true } | { a, b: false }).
    // Source is an intersection, target is an object (e.g. { a } & { b } <=> { a, b }).
    // Source is an intersection, target is a union (e.g. { a } & { b: boolean } <=> { a, b: true } | { a, b: false }).
    // Source is an intersection, target instantiable (e.g. string & { tag } <=> T["a"] constrained to string & { tag }).
    if (!((source.flags & TypeFlagsInstantiable) !== 0 ||
      (source.flags & TypeFlagsObject) !== 0 && (target.flags & TypeFlagsUnion) !== 0 ||
      (source.flags & TypeFlagsIntersection) !== 0 && (target.flags & (TypeFlagsObject | TypeFlagsUnion | TypeFlagsInstantiable)) !== 0)) {
      return TernaryFalse;
    }
  }
  // We limit alias variance probing to only object and conditional types since their alias behavior
  // is more predictable than other, interned types, which may or may not have an alias depending on
  // the order in which things were checked.
  if ((source.flags & (TypeFlagsObject | TypeFlagsConditional)) !== 0 && source.alias !== undefined && source.alias.typeArguments.length !== 0 &&
    target.alias !== undefined && source.alias.symbol === target.alias.symbol && !(this.c.isMarkerType(source) || this.c.isMarkerType(target))) {
    const variances = this.c.getAliasVariances(source.alias.symbol);
    if (variances.length === 0) {
      return TernaryUnknown;
    }
    const params = this.c.typeAliasLinks.Get(source.alias.symbol).typeParameters;
    const minParams = this.c.getMinTypeArgumentCount(params);
    const nodeIsInJsFile = IsInJSFile(source.alias.symbol.ValueDeclaration);
    const sourceTypes = this.c.fillMissingTypeArguments(source.alias.typeArguments, params, minParams, nodeIsInJsFile);
    const targetTypes = this.c.fillMissingTypeArguments(target.alias.typeArguments, params, minParams, nodeIsInJsFile);
    const [varianceResult, ok] = relateVariances(sourceTypes, targetTypes, variances, intersectionState);
    if (ok) {
      return varianceResult;
    }
  }
  // For a generic type T and a type U that is assignable to T, [...U] is assignable to T, U is assignable to readonly [...T],
  // and U is assignable to [...T] when U is constrained to a mutable array or tuple type.
  if (isSingleElementGenericTupleType(source) && !source.TargetTupleType().readonly) {
    result = this.isRelatedTo(this.c.getTypeArguments(source)[0], target, RecursionFlagsSource, false /*reportErrors*/);
    if (result !== TernaryFalse) {
      return result;
    }
  }
  if (isSingleElementGenericTupleType(target) && (target.TargetTupleType().readonly || this.c.isMutableArrayOrTuple(this.c.getBaseConstraintOrType(source)))) {
    result = this.isRelatedTo(source, this.c.getTypeArguments(target)[0], RecursionFlagsTarget, false /*reportErrors*/);
    if (result !== TernaryFalse) {
      return result;
    }
  }
  if ((target.flags & TypeFlagsTypeParameter) !== 0) {
    // A source type { [P in Q]: X } is related to a target type T if keyof T is related to Q and X is related to T[Q].
    if ((source.objectFlags & ObjectFlagsMapped) !== 0 && source.AsMappedType().declaration.NameType === undefined && this.isRelatedTo(this.c.getIndexType(target), this.c.getConstraintTypeFromMappedType(source), RecursionFlagsBoth, false) !== TernaryFalse) {
      if ((getMappedTypeModifiers(source) & MappedTypeModifiersIncludeOptional) === 0) {
        const templateType = this.c.getTemplateTypeFromMappedType(source);
        const indexedAccessType = this.c.getIndexedAccessType(target, this.c.getTypeParameterFromMappedType(source));
        result = this.isRelatedTo(templateType, indexedAccessType, RecursionFlagsBoth, reportErrors);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
    if (this.relation === this.c.comparableRelation && (source.flags & TypeFlagsTypeParameter) !== 0) {
      // This is a carve-out in comparability to essentially forbid comparing a type parameter with another type parameter
      // unless one extends the other. (Remember: comparability is mostly bidirectional!)
      const constraint = this.c.getConstraintOfTypeParameter(source);
      if (constraint !== undefined && someType(constraint, (c) => (c.flags & TypeFlagsTypeParameter) !== 0)) {
        return this.isRelatedTo(constraint, target, RecursionFlagsSource, false /*reportErrors*/);
      }
      return TernaryFalse;
    }
  } else if ((target.flags & TypeFlagsIndexedAccess) !== 0) {
    if ((source.flags & TypeFlagsIndexedAccess) !== 0) {
      // Relate components directly before falling back to constraint relationships
      // A type S[K] is related to a type T[J] if S is related to T and K is related to J.
      result = this.isRelatedTo(source.AsIndexedAccessType().objectType, target.AsIndexedAccessType().objectType, RecursionFlagsBoth, reportErrors);
      if (result !== TernaryFalse) {
        result &= this.isRelatedTo(source.AsIndexedAccessType().indexType, target.AsIndexedAccessType().indexType, RecursionFlagsBoth, reportErrors);
      }
      if (result !== TernaryFalse) {
        return result;
      }
      if (reportErrors) {
        originalErrorChain = this.errorChain;
      }
    }
    // A type S is related to a type T[K] if S is related to C, where C is the base
    // constraint of T[K] for writing.
    if (this.relation === this.c.assignableRelation || this.relation === this.c.comparableRelation) {
      const objectType = target.AsIndexedAccessType().objectType;
      const indexType = target.AsIndexedAccessType().indexType;
      const baseObjectType = this.c.getBaseConstraintOrType(objectType);
      const baseIndexType = this.c.getBaseConstraintOrType(indexType);
      if (!this.c.isGenericObjectType(baseObjectType) && !this.c.isGenericIndexType(baseIndexType)) {
        const accessFlags = AccessFlagsWriting | (baseObjectType !== objectType ? AccessFlagsNoIndexSignatures : 0);
        const constraint = this.c.getIndexedAccessTypeOrUndefined(baseObjectType, baseIndexType, accessFlags, undefined, undefined);
        if (constraint !== undefined) {
          if (reportErrors && originalErrorChain !== undefined) {
            // create a new chain for the constraint error
            this.restoreErrorState(saveErrorState);
          }
          result = this.isRelatedToEx(source, constraint, RecursionFlagsTarget, reportErrors, undefined /*headMessage*/, intersectionState);
          if (result !== TernaryFalse) {
            return result;
          }
          // prefer the shorter chain of the constraint comparison chain, and the direct comparison chain
          if (reportErrors && originalErrorChain !== undefined && this.errorChain !== undefined) {
            if (chainDepth(originalErrorChain) <= chainDepth(this.errorChain)) {
              this.errorChain = originalErrorChain;
            }
          }
        }
      }
    }
    if (reportErrors) {
      originalErrorChain = undefined;
    }
  } else if ((target.flags & TypeFlagsIndex) !== 0) {
    const targetType = target.AsIndexType().target;
    // A keyof S is related to a keyof T if T is related to S.
    if ((source.flags & TypeFlagsIndex) !== 0) {
      result = this.isRelatedTo(targetType, source.AsIndexType().target, RecursionFlagsBoth, false /*reportErrors*/);
      if (result !== TernaryFalse) {
        return result;
      }
    }
    if (isTupleType(targetType)) {
      // An index type can have a tuple type target when the tuple type contains variadic elements.
      // Check if the source is related to the known keys of the tuple type.
      result = this.isRelatedTo(source, this.c.getKnownKeysOfTupleType(targetType), RecursionFlagsTarget, reportErrors);
      if (result !== TernaryFalse) {
        return result;
      }
    } else {
      // A type S is assignable to keyof T if S is assignable to keyof C, where C is the
      // simplified form of T or, if T doesn't simplify, the constraint of T.
      const constraint = this.c.getSimplifiedTypeOrConstraint(targetType);
      if (constraint !== undefined) {
        // We require Ternary.True here such that circular constraints don't cause
        // false positives. For example, given 'T extends { [K in keyof T]: string }',
        // 'keyof T' has itself as its constraint and produces a Ternary.Maybe when
        // related to other types.
        if (this.isRelatedTo(source, this.c.getIndexTypeEx(constraint, target.AsIndexType().indexFlags | IndexFlagsNoReducibleCheck), RecursionFlagsTarget, reportErrors) === TernaryTrue) {
          return TernaryTrue;
        }
      } else if (this.c.isGenericMappedType(targetType)) {
        // generic mapped types that don't simplify or have a constraint still have a very simple set of keys we can compare against
        // - their nameType or constraintType.
        // In many ways, this comparison is a deferred version of what `getIndexTypeForMappedType` does to actually resolve the keys for _non_-generic types
        const nameType = this.c.getNameTypeFromMappedType(targetType);
        const constraintType = this.c.getConstraintTypeFromMappedType(targetType);
        let targetKeys: Type;
        if (nameType !== undefined && this.c.isMappedTypeWithKeyofConstraintDeclaration(targetType)) {
          // we need to get the apparent mappings and union them with the generic mappings, since some properties may be
          // missing from the `constraintType` which will otherwise be mapped in the object
          const mappedKeys = this.c.getApparentMappedTypeKeys(nameType, targetType);
          // We still need to include the non-apparent (and thus still generic) keys in the target side of the comparison (in case they're in the source side)
          targetKeys = this.c.getUnionType([mappedKeys, nameType]);
        } else if (nameType !== undefined) {
          targetKeys = nameType;
        } else {
          targetKeys = constraintType;
        }
        if (this.isRelatedTo(source, targetKeys, RecursionFlagsTarget, reportErrors) === TernaryTrue) {
          return TernaryTrue;
        }
      }
    }
  } else if ((target.flags & TypeFlagsConditional) !== 0) {
    // If we reach 10 levels of nesting for the same conditional type, assume it is an infinitely expanding recursive
    // conditional type and bail out with a Ternary.Maybe result.
    if (this.c.isDeeplyNestedType(target, this.targetStack, 10)) {
      return TernaryMaybe;
    }
    const c = target.AsConditionalType();
    // We check for a relationship to a conditional type target only when the conditional type has no
    // 'infer' positions, is not distributive or is distributive but doesn't reference the check type
    // parameter in either of the result types, and the source isn't an instantiation of the same
    // conditional type (as happens when computing variance).
    if (c.root.inferTypeParameters === undefined && !this.c.isDistributionDependent(c.root) && !((source.flags & TypeFlagsConditional) !== 0 && source.AsConditionalType().root === c.root)) {
      // Check if the conditional is always true or always false but still deferred for distribution purposes.
      const skipTrue = !this.c.isTypeAssignableTo(this.c.getPermissiveInstantiation(c.checkType), this.c.getPermissiveInstantiation(c.extendsType));
      const skipFalse = !skipTrue && this.c.isTypeAssignableTo(this.c.getRestrictiveInstantiation(c.checkType), this.c.getRestrictiveInstantiation(c.extendsType));
      // TODO: Find a nice way to include potential conditional type breakdowns in error output, if they seem good (they usually don't)
      if (skipTrue) {
        result = TernaryTrue;
      } else {
        result = this.isRelatedToEx(source, this.c.getTrueTypeFromConditionalType(target), RecursionFlagsTarget, false /*reportErrors*/, undefined /*headMessage*/, intersectionState);
      }
      if (result !== TernaryFalse) {
        if (skipFalse) {
          result &= TernaryTrue;
        } else {
          result &= this.isRelatedToEx(source, this.c.getFalseTypeFromConditionalType(target), RecursionFlagsTarget, false /*reportErrors*/, undefined /*headMessage*/, intersectionState);
        }
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
  } else if ((target.flags & TypeFlagsTemplateLiteral) !== 0) {
    if ((source.flags & TypeFlagsTemplateLiteral) !== 0) {
      if (this.relation === this.c.comparableRelation) {
        if (this.c.templateLiteralTypesDefinitelyUnrelated(source.AsTemplateLiteralType(), target.AsTemplateLiteralType())) {
          return TernaryFalse;
        }
        return TernaryTrue;
      }
      // Report unreliable variance for type variables referenced in template literal type placeholders.
      // For example, `foo-${number}` is related to `foo-${string}` even though number isn't related to string.
      this.c.instantiateType(source, this.c.reportUnreliableMapper);
    }
    if (this.c.isTypeMatchedByTemplateLiteralType(source, target.AsTemplateLiteralType(), this.isRelatedToWorker.bind(this))) {
      return TernaryTrue;
    }
  } else if ((target.flags & TypeFlagsStringMapping) !== 0) {
    if ((source.flags & TypeFlagsStringMapping) === 0) {
      if (this.c.isMemberOfStringMapping(source, target)) {
        return TernaryTrue;
      }
    }
  } else if (this.c.isGenericMappedType(target) && this.relation !== this.c.identityRelation) {
    // Check if source type `S` is related to target type `{ [P in Q]: T }` or `{ [P in Q as R]: T}`.
    const keysRemapped = target.AsMappedType().declaration.NameType !== undefined;
    const templateType = this.c.getTemplateTypeFromMappedType(target);
    const modifiers = getMappedTypeModifiers(target);
    if ((modifiers & MappedTypeModifiersExcludeOptional) === 0) {
      // If the mapped type has shape `{ [P in Q]: T[P] }`,
      // source `S` is related to target if `T` = `S`, i.e. `S` is related to `{ [P in Q]: S[P] }`.
      if (!keysRemapped && (templateType.flags & TypeFlagsIndexedAccess) !== 0 && templateType.AsIndexedAccessType().objectType === source && templateType.AsIndexedAccessType().indexType === this.c.getTypeParameterFromMappedType(target)) {
        return TernaryTrue;
      }
      if (!this.c.isGenericMappedType(source)) {
        // If target has shape `{ [P in Q as R]: T}`, then its keys have type `R`.
        // If target has shape `{ [P in Q]: T }`, then its keys have type `Q`.
        let targetKeys: Type;
        if (keysRemapped) {
          targetKeys = this.c.getNameTypeFromMappedType(target);
        } else {
          targetKeys = this.c.getConstraintTypeFromMappedType(target);
        }
        // Type of the keys of source type `S`, i.e. `keyof S`.
        const sourceKeys = this.c.getIndexTypeEx(source, IndexFlagsNoIndexSignatures);
        const includeOptional = (modifiers & MappedTypeModifiersIncludeOptional) !== 0;
        let filteredByApplicability: Type | undefined;
        if (includeOptional) {
          filteredByApplicability = this.c.intersectTypes(targetKeys, sourceKeys);
        }
        // A source type `S` is related to a target type `{ [P in Q]: T }` if `Q` is related to `keyof S` and `S[Q]` is related to `T`.
        // A source type `S` is related to a target type `{ [P in Q as R]: T }` if `R` is related to `keyof S` and `S[R]` is related to `T.
        // A source type `S` is related to a target type `{ [P in Q]?: T }` if some constituent `Q'` of `Q` is related to `keyof S` and `S[Q']` is related to `T`.
        // A source type `S` is related to a target type `{ [P in Q as R]?: T }` if some constituent `R'` of `R` is related to `keyof S` and `S[R']` is related to `T`.
        if (includeOptional && (filteredByApplicability!.flags & TypeFlagsNever) === 0 || !includeOptional && this.isRelatedTo(targetKeys, sourceKeys, RecursionFlagsBoth, false) !== TernaryFalse) {
          const templateType = this.c.getTemplateTypeFromMappedType(target);
          const typeParameter = this.c.getTypeParameterFromMappedType(target);
          // Fastpath: When the template type has the form `Obj[P]` where `P` is the mapped type parameter, directly compare source `S` with `Obj`
          // to avoid creating the (potentially very large) number of new intermediate types made by manufacturing `S[P]`.
          const nonNullComponent = this.c.extractTypesOfKind(templateType, ~TypeFlagsNullable);
          if (!keysRemapped && (nonNullComponent.flags & TypeFlagsIndexedAccess) !== 0 && nonNullComponent.AsIndexedAccessType().indexType === typeParameter) {
            result = this.isRelatedTo(source, nonNullComponent.AsIndexedAccessType().objectType, RecursionFlagsTarget, reportErrors);
            if (result !== TernaryFalse) {
              return result;
            }
          } else {
            // We need to compare the type of a property on the source type `S` to the type of the same property on the target type,
            // so we need to construct an indexing type representing a property, and then use indexing type to index the source type for comparison.
            // If the target type has shape `{ [P in Q]: T }`, then a property of the target has type `P`.
            // If the target type has shape `{ [P in Q]?: T }`, then a property of the target has type `P`,
            // but the property is optional, so we only want to compare properties `P` that are common between `keyof S` and `Q`.
            // If the target type has shape `{ [P in Q as R]: T }`, then a property of the target has type `R`.
            // If the target type has shape `{ [P in Q as R]?: T }`, then a property of the target has type `R`,
            // but the property is optional, so we only want to compare properties `R` that are common between `keyof S` and `R`.
            let indexingType = typeParameter;
            if (keysRemapped) {
              indexingType = filteredByApplicability ?? targetKeys;
            } else if (filteredByApplicability !== undefined) {
              indexingType = this.c.getIntersectionType([filteredByApplicability, typeParameter]);
            }
            const indexedAccessType = this.c.getIndexedAccessType(source, indexingType);
            // Compare `S[indexingType]` to `T`, where `T` is the type of a property of the target type.
            result = this.isRelatedTo(indexedAccessType, templateType, RecursionFlagsBoth, reportErrors);
            if (result !== TernaryFalse) {
              return result;
            }
          }
        }
        originalErrorChain = this.errorChain;
        this.restoreErrorState(saveErrorState);
      }
    }
  }
  if ((source.flags & TypeFlagsTypeVariable) !== 0) {
    // IndexedAccess comparisons are handled above in the `target.flags&TypeFlagsIndexedAccess` branch
    if ((source.flags & TypeFlagsIndexedAccess) === 0 || (target.flags & TypeFlagsIndexedAccess) === 0) {
      let constraint = this.c.getConstraintOfType(source);
      if (constraint === undefined) {
        constraint = this.c.unknownType;
      }
      // hi-speed no-this-instantiation check (less accurate, but avoids costly `this`-instantiation when the constraint will suffice), see #28231 for report on why this is needed
      result = this.isRelatedToEx(constraint, target, RecursionFlagsSource, false /*reportErrors*/, undefined /*headMessage*/, intersectionState);
      if (result !== TernaryFalse) {
        return result;
      }
      const constraintWithThis = this.c.getTypeWithThisArgument(constraint, source, false /*needApparentType*/);
      result = this.isRelatedToEx(constraintWithThis, target, RecursionFlagsSource, reportErrors && constraint !== this.c.unknownType && (target.flags & source.flags & TypeFlagsTypeParameter) === 0, undefined /*headMessage*/, intersectionState);
      if (result !== TernaryFalse) {
        return result;
      }
      if (this.c.isMappedTypeGenericIndexedAccess(source)) {
        // For an indexed access type { [P in K]: E}[X], above we have already explored an instantiation of E with X
        // substituted for P. We also want to explore type { [P in K]: E }[C], where C is the constraint of X.
        const indexConstraint = this.c.getConstraintOfType(source.AsIndexedAccessType().indexType);
        if (indexConstraint !== undefined) {
          result = this.isRelatedTo(this.c.getIndexedAccessType(source.AsIndexedAccessType().objectType, indexConstraint), target, RecursionFlagsSource, reportErrors);
          if (result !== TernaryFalse) {
            return result;
          }
        }
      }
    }
  } else if ((source.flags & TypeFlagsIndex) !== 0) {
    const isDeferredMappedIndex = this.c.shouldDeferIndexType(source.AsIndexType().target, source.AsIndexType().indexFlags) && (source.AsIndexType().target.objectFlags & ObjectFlagsMapped) !== 0;
    result = this.isRelatedTo(this.c.stringNumberSymbolType, target, RecursionFlagsSource, reportErrors && !isDeferredMappedIndex);
    if (result !== TernaryFalse) {
      return result;
    }
    if (isDeferredMappedIndex) {
      const mappedType = source.AsIndexType().target;
      const nameType = this.c.getNameTypeFromMappedType(mappedType);
      // Unlike on the target side, on the source side we do *not* include the generic part of the `nameType`, since that comes from a
      // (potentially anonymous) mapped type local type parameter, so that'd never assign outside the mapped type body, but we still want to
      // allow assignments of index types of identical (or similar enough) mapped types.
      // eg, `keyof {[X in keyof A]: Obj[X]}` should be assignable to `keyof {[Y in keyof A]: Tup[Y]}` because both map over the same set of keys (`keyof A`).
      // Without this source-side breakdown, a `keyof {[X in keyof A]: Obj[X]}` style type won't be assignable to anything except itself, which is much too strict.
      let sourceMappedKeys: Type;
      if (nameType !== undefined && this.c.isMappedTypeWithKeyofConstraintDeclaration(mappedType)) {
        sourceMappedKeys = this.c.getApparentMappedTypeKeys(nameType, mappedType);
      } else if (nameType !== undefined) {
        sourceMappedKeys = nameType;
      } else {
        sourceMappedKeys = this.c.getConstraintTypeFromMappedType(mappedType);
      }
      result = this.isRelatedTo(sourceMappedKeys, target, RecursionFlagsSource, reportErrors);
      if (result !== TernaryFalse) {
        return result;
      }
    }
  } else if ((source.flags & TypeFlagsConditional) !== 0) {
    // If we reach 10 levels of nesting for the same conditional type, assume it is an infinitely expanding recursive
    // conditional type and bail out with a Ternary.Maybe result.
    if (this.c.isDeeplyNestedType(source, this.sourceStack, 10)) {
      return TernaryMaybe;
    }
    if ((target.flags & TypeFlagsConditional) !== 0) {
      // Two conditional types 'T1 extends U1 ? X1 : Y1' and 'T2 extends U2 ? X2 : Y2' are related if
      // one of T1 and T2 is related to the other, U1 and U2 are identical types, X1 is related to X2,
      // and Y1 is related to Y2.
      const sourceParams = source.AsConditionalType().root.inferTypeParameters;
      let sourceExtends = source.AsConditionalType().extendsType;
      let mapper: TypeMapper | undefined;
      if (sourceParams.length !== 0) {
        // If the source has infer type parameters, we instantiate them in the context of the target
        const ctx = this.c.newInferenceContext(sourceParams, undefined /*signature*/, InferenceFlagsNone, this.isRelatedToWorker.bind(this));
        this.c.inferTypes(ctx.inferences, target.AsConditionalType().extendsType, sourceExtends, InferencePriorityNoConstraints | InferencePriorityAlwaysStrict, false);
        sourceExtends = this.c.instantiateType(sourceExtends, ctx.mapper);
        mapper = ctx.mapper;
      }
      if (this.c.isTypeIdenticalTo(sourceExtends, target.AsConditionalType().extendsType) && (this.isRelatedTo(source.AsConditionalType().checkType, target.AsConditionalType().checkType, RecursionFlagsBoth, false) !== 0 || this.isRelatedTo(target.AsConditionalType().checkType, source.AsConditionalType().checkType, RecursionFlagsBoth, false) !== 0)) {
        result = this.isRelatedTo(this.c.instantiateType(this.c.getTrueTypeFromConditionalType(source), mapper), this.c.getTrueTypeFromConditionalType(target), RecursionFlagsBoth, reportErrors);
        if (result !== TernaryFalse) {
          result &= this.isRelatedTo(this.c.getFalseTypeFromConditionalType(source), this.c.getFalseTypeFromConditionalType(target), RecursionFlagsBoth, reportErrors);
        }
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
    // conditionals can be related to one another via normal constraint, as, eg, `A extends B ? O : never` should be assignable to `O`
    // when `O` is a conditional (`never` is trivially assignable to `O`, as is `O`!).
    const defaultConstraint = this.c.getDefaultConstraintOfConditionalType(source);
    if (defaultConstraint !== undefined) {
      result = this.isRelatedTo(defaultConstraint, target, RecursionFlagsSource, reportErrors);
      if (result !== TernaryFalse) {
        return result;
      }
    }
    // conditionals aren't related to one another via distributive constraint as it is much too inaccurate and allows way
    // more assignments than are desirable (since it maps the source check type to its constraint, it loses information).
    if ((target.flags & TypeFlagsConditional) === 0 && this.c.hasNonCircularBaseConstraint(source)) {
      const distributiveConstraint = this.c.getConstraintOfDistributiveConditionalType(source);
      if (distributiveConstraint !== undefined) {
        this.restoreErrorState(saveErrorState);
        result = this.isRelatedTo(distributiveConstraint, target, RecursionFlagsSource, reportErrors);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
  } else if ((source.flags & TypeFlagsTemplateLiteral) !== 0 && (target.flags & TypeFlagsObject) === 0) {
    if ((target.flags & TypeFlagsTemplateLiteral) === 0) {
      const constraint = this.c.getBaseConstraintOfType(source);
      if (constraint !== undefined && constraint !== source) {
        result = this.isRelatedTo(constraint, target, RecursionFlagsSource, reportErrors);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
  } else if ((source.flags & TypeFlagsStringMapping) !== 0) {
    if ((target.flags & TypeFlagsStringMapping) !== 0) {
      if (source.AsStringMappingType().symbol !== target.AsStringMappingType().symbol) {
        return TernaryFalse;
      }
      result = this.isRelatedTo(source.AsStringMappingType().target, target.AsStringMappingType().target, RecursionFlagsBoth, reportErrors);
      if (result !== TernaryFalse) {
        return result;
      }
    } else {
      const constraint = this.c.getBaseConstraintOfType(source);
      if (constraint !== undefined) {
        result = this.isRelatedTo(constraint, target, RecursionFlagsSource, reportErrors);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
  } else {
    // An empty object type is related to any mapped type that includes a '?' modifier.
    if (this.relation !== this.c.subtypeRelation && this.relation !== this.c.strictSubtypeRelation && isPartialMappedType(target) && this.c.isEmptyObjectType(source)) {
      return TernaryTrue;
    }
    if (this.c.isGenericMappedType(target)) {
      if (this.c.isGenericMappedType(source)) {
        result = this.mappedTypeRelatedTo(source, target, reportErrors);
        if (result !== TernaryFalse) {
          return result;
        }
      }
      return TernaryFalse;
    }
    const sourceIsPrimitive = (source.flags & TypeFlagsPrimitive) !== 0;
    if (this.relation !== this.c.identityRelation) {
      source = this.c.getApparentType(source);
    } else if (this.c.isGenericMappedType(source)) {
      return TernaryFalse;
    }
    if ((source.objectFlags & ObjectFlagsReference) !== 0 && (target.objectFlags & ObjectFlagsReference) !== 0 && source.Target() === target.Target() && !isTupleType(source) && !this.c.isMarkerType(source) && !this.c.isMarkerType(target)) {
      // When strictNullChecks is disabled, the element type of the empty array literal is undefinedWideningType,
      // and an empty array literal wouldn't be assignable to a `never[]` without this check.
      if (this.c.isEmptyArrayLiteralType(source)) {
        return TernaryTrue;
      }
      // We have type references to the same generic type, and the type references are not marker
      // type references (which are intended by be compared structurally). Obtain the variance
      // information for the type parameters and relate the type arguments accordingly.
      const variances = this.c.getVariances(source.Target());
      // We return Ternary.Maybe for a recursive invocation of getVariances (signaled by emptyArray). This
      // effectively means we measure variance only from type parameter occurrences that aren't nested in
      // recursive instantiations of the generic type.
      if (variances.length === 0) {
        return TernaryUnknown;
      }
      const [varianceResult, ok] = relateVariances(this.c.getTypeArguments(source), this.c.getTypeArguments(target), variances, intersectionState);
      if (ok) {
        return varianceResult;
      }
    } else if (this.c.isArrayType(target) && (this.c.isReadonlyArrayType(target) && everyType(source, this.c.isArrayOrTupleType.bind(this.c)) || everyType(source, isMutableTupleType))) {
      if (this.relation !== this.c.identityRelation) {
        return this.isRelatedTo(this.c.getIndexTypeOfTypeEx(source, this.c.numberType, this.c.anyType), this.c.getIndexTypeOfTypeEx(target, this.c.numberType, this.c.anyType), RecursionFlagsBoth, reportErrors);
      }
      // By flags alone, we know that the `target` is a readonly array while the source is a normal array or tuple
      // or `target` is an array and source is a tuple - in both cases the types cannot be identical, by construction
      return TernaryFalse;
    } else if (this.c.isGenericTupleType(source) && isTupleType(target) && !this.c.isGenericTupleType(target)) {
      const constraint = this.c.getBaseConstraintOrType(source);
      if (constraint !== source) {
        return this.isRelatedTo(constraint, target, RecursionFlagsSource, reportErrors);
      }
    } else if ((this.relation === this.c.subtypeRelation || this.relation === this.c.strictSubtypeRelation) && this.c.isEmptyObjectType(target) && (target.objectFlags & ObjectFlagsFreshLiteral) !== 0 && !this.c.isEmptyObjectType(source)) {
      return TernaryFalse;
    }
    // Even if relationship doesn't hold for unions, intersections, or generic type references,
    // it may hold in a structural comparison.
    // In a check of the form X = A & B, we will have previously checked if A relates to X or B relates
    // to X. Failing both of those we want to check if the aggregation of A and B's members structurally
    // relates to X. Thus, we include intersection types on the source side here.
    if ((source.flags & (TypeFlagsObject | TypeFlagsIntersection)) !== 0 && (target.flags & TypeFlagsObject) !== 0) {
      // Report structural errors only if we haven't reported any errors yet
      const reportStructuralErrors = reportErrors && this.errorChain === saveErrorState.errorChain && !sourceIsPrimitive;
      result = this.propertiesRelatedTo(source, target, reportStructuralErrors, new Set<string>() /*excludedProperties*/, false /*optionalsOnly*/, intersectionState);
      if (result !== TernaryFalse) {
        result &= this.signaturesRelatedTo(source, target, SignatureKindCall, reportStructuralErrors, intersectionState);
        if (result !== TernaryFalse) {
          result &= this.signaturesRelatedTo(source, target, SignatureKindConstruct, reportStructuralErrors, intersectionState);
          if (result !== TernaryFalse) {
            result &= this.indexSignaturesRelatedTo(source, target, sourceIsPrimitive, reportStructuralErrors, intersectionState);
          }
        }
      }
      if (result !== TernaryFalse) {
        if (!varianceCheckFailed) {
          return result;
        }
        if (originalErrorChain !== undefined) {
          this.errorChain = originalErrorChain;
        } else if (this.errorChain === undefined) {
          this.errorChain = saveErrorState.errorChain;
        }
        // Use variance error (there is no structural one) and return false
      }
    }
    // If S is an object type and T is a discriminated union, S may be related to T if
    // there exists a constituent of T for every combination of the discriminants of S
    // with respect to T. We do not report errors here, as we will use the existing
    // error result from checking each constituent of the union.
    if ((source.flags & (TypeFlagsObject | TypeFlagsIntersection)) !== 0 && (target.flags & TypeFlagsUnion) !== 0) {
      const objectOnlyTarget = this.c.extractTypesOfKind(target, TypeFlagsObject | TypeFlagsIntersection | TypeFlagsSubstitution);
      if ((objectOnlyTarget.flags & TypeFlagsUnion) !== 0) {
        const result = this.typeRelatedToDiscriminatedType(source, objectOnlyTarget);
        if (result !== TernaryFalse) {
          return result;
        }
      }
    }
  }
  return TernaryFalse;
}
