package image

import (
	"reflect"
	"testing"
)

func TestShortParseWithTag(t *testing.T) {

	reference, err := Parse("foo/bar:1.1")
	if err != nil {
		t.Fatalf("error while parsing tag: %s", err)
	}

	if reference.Remote() != DefaultRegistryHostname+"/foo/bar:1.1" {
		t.Errorf("unexpected remote: %s", reference.Remote())
	}

	if reference.Tag() != "1.1" {
		t.Errorf("unexpected tag: %s", reference.Tag())
	}

	if reference.Registry() != DefaultRegistryHostname {
		t.Errorf("unexpected registry: %s", reference.Registry())
	}

	if reference.ShortName() != "foo/bar" {
		t.Errorf("unexpected name: %s", reference.ShortName())
	}

	if reference.Name() != "foo/bar:1.1" {
		t.Errorf("unexpected name: %s", reference.Name())
	}

}

func TestParseRepo(t *testing.T) {
	type args struct {
		remote string
	}
	tests := []struct {
		name    string
		args    args
		want    *Repository
		wantErr bool
	}{
		{
			name: "foo/bar:1.1",
			args: args{remote: "foo/bar:1.1"},
			want: &Repository{
				Name:       "foo/bar:1.1",
				Repository: "index.docker.io/foo/bar",
				Remote:     "index.docker.io/foo/bar:1.1",
				Registry:   DefaultRegistryHostname,
				ShortName:  "foo/bar",
				Tag:        "1.1",
				Scheme:     "https",
			},
			wantErr: false,
		},
		{
			name: "localhost.localdomain/foo/bar:1.1",
			args: args{remote: "localhost.localdomain/foo/bar:1.1"},
			want: &Repository{
				Name:       "foo/bar:1.1",
				Repository: "localhost.localdomain/foo/bar",
				Remote:     "localhost.localdomain/foo/bar:1.1",
				Registry:   "localhost.localdomain",
				ShortName:  "foo/bar",
				Tag:        "1.1",
				Scheme:     "https",
			},
			wantErr: false,
		},
		{
			name: "https://httphost.sh/foo/bar:1.1",
			args: args{remote: "https://httphost.sh/foo/bar:1.1"},
			want: &Repository{
				Name:       "foo/bar:1.1",
				Repository: "httphost.sh/foo/bar",
				Remote:     "httphost.sh/foo/bar:1.1",
				Registry:   "httphost.sh",
				ShortName:  "foo/bar",
				Tag:        "1.1",
				Scheme:     "https",
			},
			wantErr: false,
		},
		{
			name: "localhost.localdomain/foo/bar (no tag)",
			args: args{remote: "localhost.localdomain/foo/bar"},
			want: &Repository{
				Name:       "foo/bar:latest",
				Repository: "localhost.localdomain/foo/bar",
				Remote:     "localhost.localdomain/foo/bar:latest",
				Registry:   "localhost.localdomain",
				ShortName:  "foo/bar",
				Tag:        "latest",
				Scheme:     "https",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepo(tt.args.remote)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
