package secrets

import "testing"

func Test_registryMatches(t *testing.T) {
	type args struct {
		imageRegistry  string
		secretRegistry string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "matches",
			args: args{imageRegistry: "docker.io", secretRegistry: "docker.io"},
			want: true,
		},
		{
			name: "doesnt match",
			args: args{imageRegistry: "docker.io", secretRegistry: "index.docker.io"},
			want: false,
		},
		{
			name: "matches, secret with port",
			args: args{imageRegistry: "docker.io", secretRegistry: "docker.io:443"},
			want: true,
		},
		{
			name: "matches, image with port",
			args: args{imageRegistry: "docker.io:443", secretRegistry: "docker.io"},
			want: true,
		},
		{
			name: "matches, image with scheme",
			args: args{imageRegistry: "https://docker.io", secretRegistry: "docker.io"},
			want: true,
		},
		{
			name: "matches, secret with scheme",
			args: args{imageRegistry: "docker.io", secretRegistry: "https://docker.io"},
			want: true,
		},
		{
			name: "matches, both with scheme",
			args: args{imageRegistry: "https://docker.io", secretRegistry: "https://docker.io"},
			want: true,
		},
		{
			name: "matches, both with scheme and port",
			args: args{imageRegistry: "https://docker.io:443", secretRegistry: "https://docker.io:443"},
			want: true,
		},
		{
			name: "matches, both with scheme and port and a URL path in the secret",
			args: args{imageRegistry: "https://docker.io:443", secretRegistry: "https://docker.io:443/v1"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registryMatches(tt.args.imageRegistry, tt.args.secretRegistry); got != tt.want {
				t.Errorf("registryMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
