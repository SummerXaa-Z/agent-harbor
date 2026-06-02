package domain

import (
	"reflect"
	"testing"
)

func TestEffectiveDataScopes(t *testing.T) {
	parent := []DataScope{{
		DataDomain:   "crm",
		Dataset:      "customers",
		TenantFilter: "tenant_id = 'tenant-a'",
		Region:       "us-east",
	}}

	tests := []struct {
		name   string
		parent []DataScope
		child  []DataScope
		want   []DataScope
		ok     bool
	}{
		{
			name:   "equal scope is accepted",
			parent: parent,
			child:  parent,
			want:   parent,
			ok:     true,
		},
		{
			name:   "empty child inherits parent",
			parent: parent,
			child:  nil,
			want:   parent,
			ok:     true,
		},
		{
			name: "empty parent accepts child",
			child: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
			}},
			want: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
			}},
			ok: true,
		},
		{
			name: "child fills empty parent field",
			parent: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
			}},
			child: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
				Table:      "accounts",
			}},
			want: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
				Table:      "accounts",
			}},
			ok: true,
		},
		{
			name: "child cannot change fixed parent field",
			parent: []DataScope{{
				DataDomain: "crm",
				Region:     "us-east",
			}},
			child: []DataScope{{
				DataDomain: "crm",
				Region:     "eu-west",
			}},
			ok: false,
		},
		{
			name: "one parent alternative may contain the child",
			parent: []DataScope{
				{DataDomain: "finance", Dataset: "invoices"},
				{DataDomain: "crm", Dataset: "customers"},
			},
			child: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
				Table:      "accounts",
			}},
			want: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
				Table:      "accounts",
			}},
			ok: true,
		},
		{
			name: "unmatched child alternative rejects list",
			parent: []DataScope{{
				DataDomain: "crm",
				Dataset:    "customers",
			}},
			child: []DataScope{
				{DataDomain: "crm", Dataset: "customers", Table: "accounts"},
				{DataDomain: "finance", Dataset: "invoices"},
			},
			ok: false,
		},
		{
			name: "child inherits fixed parent fields",
			parent: []DataScope{{
				DataDomain:   "crm",
				TenantFilter: "tenant_id = 'tenant-a'",
			}},
			child: []DataScope{{
				DataDomain: "crm",
				Table:      "accounts",
			}},
			want: []DataScope{{
				DataDomain:   "crm",
				Table:        "accounts",
				TenantFilter: "tenant_id = 'tenant-a'",
			}},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := EffectiveDataScopes(tt.parent, tt.child)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v, got %#v", ok, tt.ok, got)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("scopes = %#v, want %#v", got, tt.want)
			}
		})
	}
}
