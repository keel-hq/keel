package pubsub

import (
	"testing"

	"github.com/rusenask/keel/util/image"
)

func unsafeImageRef(img string) *image.Reference {
	ref, err := image.Parse(img)
	if err != nil {
		panic(err)
	}
	return ref
}

func Test_isGoogleContainerRegistry(t *testing.T) {
	type args struct {
		registry string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "gcr",
			args: args{registry: unsafeImageRef("gcr.io/v2-namespace/hello-world:1.1").Registry()},
			want: true,
		},
		{
			name: "docker registry",
			args: args{registry: unsafeImageRef("docker.io/v2-namespace/hello-world:1.1").Registry()},
			want: false,
		},
		{
			name: "custom registry",
			args: args{registry: unsafeImageRef("localhost:4000/v2-namespace/hello-world:1.1").Registry()},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGoogleContainerRegistry(tt.args.registry); got != tt.want {
				t.Errorf("isGoogleContainerRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}
