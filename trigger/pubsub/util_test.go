package pubsub

import (
	"testing"
)

func Test_extractContainerRegistryURI(t *testing.T) {
	type args struct {
		imageName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "gcr.io/v2-namespace/hello-world:1.1",
			args: args{imageName: "gcr.io/v2-namespace/hello-world:1.1"},
			want: "gcr.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractContainerRegistryURI(tt.args.imageName); got != tt.want {
				t.Errorf("extractContainerRegistryURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
