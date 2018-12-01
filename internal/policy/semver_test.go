package policy

import (
	"testing"
)

func Test_shouldUpdate(t *testing.T) {
	type args struct {
		spt     SemverPolicyType
		current string
		new     string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "new lower, policy all",
			args: args{
				current: "1.4.5",
				new:     "1.4.3",
				spt:     SemverPolicyTypeAll,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "new minor increase, policy all",
			args: args{
				current: "1.4.5",
				new:     "1.4.6",
				spt:     SemverPolicyTypeAll,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "no increase, policy all",
			args: args{
				current: "1.4.5",
				new:     "1.4.5",
				spt:     SemverPolicyTypeAll,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "minor increase, policy major",
			args: args{
				current: "1.4.5",
				new:     "1.5.5",
				spt:     SemverPolicyTypeMajor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "major increase, policy major",
			args: args{
				current: "1.4.5",
				new:     "2.4.5",
				spt:     SemverPolicyTypeMajor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch increase, policy patch",
			args: args{
				current: "1.4.5",
				new:     "1.4.6",
				spt:     SemverPolicyTypePatch,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch decrease, policy patch",
			args: args{
				current: "1.4.5",
				new:     "1.4.4",
				spt:     SemverPolicyTypePatch,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "patch AND major increase, policy patch",
			args: args{
				current: "1.4.5",
				new:     "2.4.6",
				spt:     SemverPolicyTypePatch,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "patch same, policy patch",
			args: args{
				current: "1.4.5",
				new:     "1.4.5",
				spt:     SemverPolicyTypePatch,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "minor increase, policy minor",
			args: args{
				current: "1.4.5",
				new:     "1.5.5",
				spt:     SemverPolicyTypeMinor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch increase, policy minor",
			args: args{
				current: "1.4.5",
				new:     "1.4.6",
				spt:     SemverPolicyTypeMinor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prerelease patch increase, policy minor, no prerelease",
			args: args{
				current: "1.4.5",
				new:     "1.4.5-xx",
				spt:     SemverPolicyTypeMinor,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "parsed prerelease patch increase, policy minor, no prerelease",
			args: args{
				current: "v1.0.0",
				new:     "v1.0.1-metadata",
				spt:     SemverPolicyTypeMinor,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "parsed prerelease minor increase, policy minor, both have metadata",
			args: args{
				current: "v1.0.0-metadata",
				new:     "v1.0.1-metadata",
				spt:     SemverPolicyTypeMinor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prerelease patch increase, policy minor",
			args: args{
				current: "1.4.5-xx",
				new:     "1.4.6-xx",
				spt:     SemverPolicyTypeMinor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch increase, policy minor, wrong prerelease",
			args: args{
				current: "1.4.5-xx",
				new:     "1.4.6-yy",
				spt:     SemverPolicyTypeMinor,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "number",
			args: args{
				current: "1.4.5",
				new:     "3050",
				spt:     SemverPolicyTypeAll,
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldUpdate(tt.args.spt, tt.args.current, tt.args.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
