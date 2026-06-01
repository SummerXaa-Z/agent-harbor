package domain

// CloneDataScopes returns a detached copy of data scopes.
func CloneDataScopes(value []DataScope) []DataScope {
	if value == nil {
		return nil
	}
	return append([]DataScope(nil), value...)
}

// EffectiveDataScopes returns the effective child boundary when child is equal to
// or narrower than parent. A scope slice is treated as OR alternatives.
func EffectiveDataScopes(parent []DataScope, child []DataScope) ([]DataScope, bool) {
	if len(parent) == 0 {
		return CloneDataScopes(child), true
	}
	if len(child) == 0 {
		return CloneDataScopes(parent), true
	}

	effective := make([]DataScope, 0, len(child))
	for _, childScope := range child {
		matched := false
		for _, parentScope := range parent {
			scope, ok := mergeDataScope(parentScope, childScope)
			if ok {
				effective = append(effective, scope)
				matched = true
				break
			}
		}
		if !matched {
			return nil, false
		}
	}
	return effective, true
}

func mergeDataScope(parent DataScope, child DataScope) (DataScope, bool) {
	var merged DataScope
	var ok bool
	if merged.DataDomain, ok = mergeDataScopeValue(parent.DataDomain, child.DataDomain); !ok {
		return DataScope{}, false
	}
	if merged.Dataset, ok = mergeDataScopeValue(parent.Dataset, child.Dataset); !ok {
		return DataScope{}, false
	}
	if merged.Schema, ok = mergeDataScopeValue(parent.Schema, child.Schema); !ok {
		return DataScope{}, false
	}
	if merged.Table, ok = mergeDataScopeValue(parent.Table, child.Table); !ok {
		return DataScope{}, false
	}
	if merged.Field, ok = mergeDataScopeValue(parent.Field, child.Field); !ok {
		return DataScope{}, false
	}
	if merged.Classification, ok = mergeDataScopeValue(parent.Classification, child.Classification); !ok {
		return DataScope{}, false
	}
	if merged.Region, ok = mergeDataScopeValue(parent.Region, child.Region); !ok {
		return DataScope{}, false
	}
	if merged.TenantFilter, ok = mergeDataScopeValue(parent.TenantFilter, child.TenantFilter); !ok {
		return DataScope{}, false
	}
	if merged.MaskingPolicy, ok = mergeDataScopeValue(parent.MaskingPolicy, child.MaskingPolicy); !ok {
		return DataScope{}, false
	}
	if merged.RowFilter, ok = mergeDataScopeValue(parent.RowFilter, child.RowFilter); !ok {
		return DataScope{}, false
	}
	return merged, true
}

func mergeDataScopeValue(parent string, child string) (string, bool) {
	if parent == "" {
		return child, true
	}
	if child == "" || child == parent {
		return parent, true
	}
	return "", false
}
