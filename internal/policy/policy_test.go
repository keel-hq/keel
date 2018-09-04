package policy

import (
	"testing"

	"github.com/keel-hq/keel/types"
)

func Test_getPolicyFromLabels(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			name:  "policy all",
			args:  args{labels: map[string]string{types.KeelPolicyLabel: "all"}},
			want1: true,
			want:  "all",
		},
		{
			name:  "policy minor",
			args:  args{labels: map[string]string{types.KeelPolicyLabel: "minor"}},
			want1: true,
			want:  "minor",
		},
		{
			name:  "legacy policy minor",
			args:  args{labels: map[string]string{"keel.observer/policy": "minor"}},
			want1: true,
			want:  "minor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getPolicyFromLabels(tt.args.labels)
			if got != tt.want {
				t.Errorf("getPolicyFromLabels() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getPolicyFromLabels() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
