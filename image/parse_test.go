package image

import (
	"reflect"
	"testing"
)

func TestShortParseWithTag(t *testing.T) {

	reference, err := Parse("foo/bar:1.1")
	if err != nil {
		t.Errorf("error while parsing tag: %s", err)
	}

	if reference.Tag() != "1.1" {
		t.Errorf("unexpected tag: %s", reference.Tag())
	}

	if reference.Registry() != DefaultHostname {
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
				Repository: "docker.io/foo/bar",
				Registry:   DefaultHostname,
				ShortName:  "foo/bar",
				Tag:        "1.1",
			},
			wantErr: false,
		},
		{
			name: "localhost.localdomain/foo/bar:1.1",
			args: args{remote: "localhost.localdomain/foo/bar:1.1"},
			want: &Repository{
				Name:       "foo/bar:1.1",
				Repository: "localhost.localdomain/foo/bar",
				Registry:   "localhost.localdomain",
				ShortName:  "foo/bar",
				Tag:        "1.1",
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
