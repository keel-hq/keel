package version

import (
	"reflect"
	"testing"

	"github.com/rusenask/keel/types"
)

func TestGetVersionFromImageName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Version
		wantErr bool
	}{
		{
			name:    "image",
			args:    args{name: "karolis/webhook-demo:1.4.5"},
			want:    &types.Version{Major: 1, Minor: 4, Patch: 5},
			wantErr: false,
		},
		{
			name:    "image latest",
			args:    args{name: "karolis/webhook-demo:latest"},
			wantErr: true,
		},
		{
			name:    "image webhookrelay",
			args:    args{name: "gcr.io/webhookrelay/webhookrelay:0.1.14"},
			want:    &types.Version{Major: 0, Minor: 1, Patch: 14},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetVersionFromImageName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVersionFromImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVersionFromImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUpdate(t *testing.T) {
	type args struct {
		current *types.Version
		new     *types.Version
		policy  types.PolicyType
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
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 4, Patch: 3},
				policy:  types.PolicyTypeAll,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "new minor increase, policy all",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 4, Patch: 6},
				policy:  types.PolicyTypeAll,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "no increase, policy all",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 4, Patch: 5},
				policy:  types.PolicyTypeAll,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "minor increase, policy major",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 5, Patch: 5},
				policy:  types.PolicyTypeMajor,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "major increase, policy major",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 2, Minor: 4, Patch: 5},
				policy:  types.PolicyTypeMajor,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch increase, policy patch",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 4, Patch: 6},
				policy:  types.PolicyTypePatch,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "patch AND major increase, policy patch",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 2, Minor: 4, Patch: 6},
				policy:  types.PolicyTypePatch,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "patch same, policy patch",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 4, Patch: 5},
				policy:  types.PolicyTypePatch,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "minor increase, policy minor",
			args: args{
				current: &types.Version{Major: 1, Minor: 4, Patch: 5},
				new:     &types.Version{Major: 1, Minor: 5, Patch: 5},
				policy:  types.PolicyTypeMinor,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ShouldUpdate(tt.args.current, tt.args.new, tt.args.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShouldUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ShouldUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
