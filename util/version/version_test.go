package version

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/types"
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
			want:    MustParse("1.4.5"),
			wantErr: false,
		},
		{
			name:    "semver with v prefix",
			args:    args{name: "gcr.io/stemnapp/alpine-api:v0.0.824"},
			want:    MustParse("v0.0.824"),
			wantErr: false,
		},
		{
			name:    "image latest",
			args:    args{name: "karolis/webhook-demo:latest"},
			wantErr: true,
		},
		{
			name:    "image no tag",
			args:    args{name: "karolis/webhook-demo"},
			wantErr: true,
		},
		{
			name:    "image webhookrelay",
			args:    args{name: "gcr.io/webhookrelay/webhookrelay:0.1.14"},
			want:    MustParse("0.1.14"),
			wantErr: false,
		},
		{
			name:    "non semver, missing minor and patch",
			args:    args{name: "index.docker.io/application:42"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "image webhookrelay with rc",
			args:    args{name: "gcr.io/webhookrelay/webhookrelay:0.9.0-rc5"},
			want:    MustParse("0.9.0-rc5"),
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

func TestGetVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Version
		wantErr bool
	}{
		{
			name: "normal version",
			args: args{version: "1.2.3"},
			want: &types.Version{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Original: "1.2.3",
			},
			wantErr: false,
		},
		{
			name: "legacy semver version",
			args: args{version: "v1.2.3"},
			want: &types.Version{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Original: "v1.2.3",
			},
			wantErr: false,
		},
		{
			name:    "not semver",
			args:    args{version: "23"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "not semver, long number",
			args:    args{version: "1234567"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetVersion(tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowest(t *testing.T) {
	type args struct {
		tags []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{tags: []string{}},
			want: "",
		},
		{
			name: "three semvers",
			args: args{tags: []string{"5.0.0", "1.0.0", "3.0.0"}},
			want: "1.0.0",
		},
		{
			name: "rc candidates",
			args: args{tags: []string{"0.15.1", "0.9.0-rc5", "0.14.0"}},
			// rc will be skipped altogether
			want: "0.14.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Lowest(tt.args.tags); got != tt.want {
				t.Errorf("Lowest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAvailable(t *testing.T) {
	type args struct {
		current         string
		tags            []string
		matchPreRelease bool
	}
	tests := []struct {
		name             string
		args             args
		wantNewVersion   string
		wantNewAvailable bool
		wantErr          bool
	}{
		{
			name: "test with pre-release",
			args: args{
				current: "0.15.0",
				tags:    []string{"0.15.0", "0.9.0-rc5"},
			},
			wantNewVersion:   "",
			wantNewAvailable: false,
			wantErr:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewVersion, gotNewAvailable, err := NewAvailable(tt.args.current, tt.args.tags, tt.args.matchPreRelease)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotNewVersion != tt.wantNewVersion {
				t.Errorf("NewAvailable() gotNewVersion = %v, want %v", gotNewVersion, tt.wantNewVersion)
			}
			if gotNewAvailable != tt.wantNewAvailable {
				t.Errorf("NewAvailable() gotNewAvailable = %v, want %v", gotNewAvailable, tt.wantNewAvailable)
			}
		})
	}
}
