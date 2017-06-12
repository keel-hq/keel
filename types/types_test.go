package types

import "testing"

func TestParsePolicy(t *testing.T) {
	type args struct {
		policy string
	}
	tests := []struct {
		name string
		args args
		want PolicyType
	}{
		{
			name: "all",
			args: args{policy: "all"},
			want: PolicyTypeAll,
		},
		{
			name: "minor",
			args: args{policy: "minor"},
			want: PolicyTypeMinor,
		},
		{
			name: "major",
			args: args{policy: "major"},
			want: PolicyTypeMajor,
		},
		{
			name: "patch",
			args: args{policy: "patch"},
			want: PolicyTypePatch,
		},
		{
			name: "random",
			args: args{policy: "rand"},
			want: PolicyTypeUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsePolicy(tt.args.policy); got != tt.want {
				t.Errorf("ParsePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
