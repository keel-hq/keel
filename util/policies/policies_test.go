package policies

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/types"
)

func TestGetPolicy(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want types.PolicyType
	}{
		{
			name: "policy all",
			args: args{labels: map[string]string{types.KeelPolicyLabel: "all"}},
			want: types.PolicyTypeAll,
		},
		{
			name: "policy minor",
			args: args{labels: map[string]string{types.KeelPolicyLabel: "minor"}},
			want: types.PolicyTypeMinor,
		},
		{
			name: "legacy policy minor",
			args: args{labels: map[string]string{"keel.observer/policy": "minor"}},
			want: types.PolicyTypeMinor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPolicy(tt.args.labels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
