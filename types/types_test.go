package types

import (
	"testing"
)

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

func TestVersion_String(t *testing.T) {
	type fields struct {
		Major      int64
		Minor      int64
		Patch      int64
		PreRelease string
		Metadata   string
		Prefix     string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "semver with v",
			fields: fields{
				Major:  1,
				Minor:  1,
				Patch:  0,
				Prefix: "v",
			},
			want: "v1.1.0",
		},
		{
			name: "semver standard",
			fields: fields{
				Major: 1,
				Minor: 1,
				Patch: 5,
			},
			want: "1.1.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Version{
				Major:      tt.fields.Major,
				Minor:      tt.fields.Minor,
				Patch:      tt.fields.Patch,
				PreRelease: tt.fields.PreRelease,
				Metadata:   tt.fields.Metadata,
				Prefix:     tt.fields.Prefix,
			}
			if got := v.String(); got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
