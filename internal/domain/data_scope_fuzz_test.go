package domain

import (
	"reflect"
	"strings"
	"testing"
)

func FuzzEffectiveDataScopes(f *testing.F) {
	f.Add([]byte("crm\x00customers\x00public\x00accounts\x00id"), []byte("crm\x00customers\x00public\x00accounts\x00id"))
	f.Add([]byte("crm\x00customers\x00\x00\x00\x00us-east"), []byte("crm\x00customers\x00\x00\x00\x00eu-west"))
	f.Add([]byte("crm\x00customers"), []byte("crm\x00customers\x00public\x00accounts"))
	f.Add([]byte("\x00\x00\x00\x00\x00\x00\x00tenant-a"), []byte("crm\x00\x00\x00\x00\x00\x00tenant-a"))

	f.Fuzz(func(t *testing.T, parentRaw []byte, childRaw []byte) {
		if len(parentRaw) > 4096 || len(childRaw) > 4096 {
			t.Skip()
		}

		parent := fuzzDataScope(parentRaw)
		child := fuzzDataScope(childRaw)
		want, wantOK := fuzzMergeDataScopeOracle(parent, child)
		got, ok := EffectiveDataScopes([]DataScope{parent}, []DataScope{child})

		if ok != wantOK {
			t.Fatalf("accepted = %v, want %v; parent=%#v child=%#v got=%#v", ok, wantOK, parent, child, got)
		}
		if !wantOK {
			if got != nil {
				t.Fatalf("rejected scopes must not return an effective boundary: %#v", got)
			}
			return
		}
		if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
			t.Fatalf("effective scope = %#v, want %#v", got, want)
		}
	})
}

func fuzzDataScope(raw []byte) DataScope {
	parts := strings.SplitN(string(raw), "\x00", 10)
	parts = append(parts, make([]string, 10-len(parts))...)
	return DataScope{
		DataDomain:     parts[0],
		Dataset:        parts[1],
		Schema:         parts[2],
		Table:          parts[3],
		Field:          parts[4],
		Classification: parts[5],
		Region:         parts[6],
		TenantFilter:   parts[7],
		MaskingPolicy:  parts[8],
		RowFilter:      parts[9],
	}
}

func fuzzMergeDataScopeOracle(parent DataScope, child DataScope) (DataScope, bool) {
	parentValues := []string{parent.DataDomain, parent.Dataset, parent.Schema, parent.Table, parent.Field, parent.Classification, parent.Region, parent.TenantFilter, parent.MaskingPolicy, parent.RowFilter}
	childValues := []string{child.DataDomain, child.Dataset, child.Schema, child.Table, child.Field, child.Classification, child.Region, child.TenantFilter, child.MaskingPolicy, child.RowFilter}
	merged := make([]string, len(parentValues))
	for index := range parentValues {
		if parentValues[index] != "" && childValues[index] != "" && parentValues[index] != childValues[index] {
			return DataScope{}, false
		}
		merged[index] = childValues[index]
		if parentValues[index] != "" {
			merged[index] = parentValues[index]
		}
	}
	return DataScope{
		DataDomain:     merged[0],
		Dataset:        merged[1],
		Schema:         merged[2],
		Table:          merged[3],
		Field:          merged[4],
		Classification: merged[5],
		Region:         merged[6],
		TenantFilter:   merged[7],
		MaskingPolicy:  merged[8],
		RowFilter:      merged[9],
	}, true
}
