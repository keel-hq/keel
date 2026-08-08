package k8s

import (
	"testing"

	core_v1 "k8s.io/api/core/v1"
)

func TestExtractDigestFromImageID(t *testing.T) {
	tests := []struct {
		name     string
		imageID  string
		expected string
	}{
		{
			name:     "docker-pullable prefix with sha256",
			imageID:  "docker-pullable://registry.example.com/myimage@sha256:abc123def456",
			expected: "sha256:abc123def456",
		},
		{
			name:     "plain format without scheme prefix",
			imageID:  "registry.example.com/myimage@sha256:deadbeef1234",
			expected: "sha256:deadbeef1234",
		},
		{
			name:     "sha512 digest",
			imageID:  "docker-pullable://registry.example.com/myimage@sha512:longdigest9999",
			expected: "sha512:longdigest9999",
		},
		{
			name:     "no @ sign returns empty",
			imageID:  "registry.example.com/myimage:latest",
			expected: "",
		},
		{
			name:     "unrecognised digest algorithm returns empty",
			imageID:  "registry.example.com/myimage@md5:abc123",
			expected: "",
		},
		{
			name:     "empty string returns empty",
			imageID:  "",
			expected: "",
		},
		{
			name:     "only @ with no recognised prefix returns empty",
			imageID:  "registry.example.com/myimage@invaliddigest",
			expected: "",
		},
		{
			name:     "docker hub short name with sha256",
			imageID:  "docker-pullable://docker.io/library/nginx@sha256:cafebabe1234",
			expected: "sha256:cafebabe1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDigestFromImageID(tt.imageID)
			if got != tt.expected {
				t.Errorf("ExtractDigestFromImageID(%q) = %q, want %q", tt.imageID, got, tt.expected)
			}
		})
	}
}

func TestFindPodImageDigest_ContainerMatch(t *testing.T) {
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/myapp:v1.2.3",
						ImageID: "docker-pullable://gcr.io/myproject/myapp@sha256:aaabbbccc111",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "gcr.io/myproject/myapp:v1.2.3")
	expected := "sha256:aaabbbccc111"
	if got != expected {
		t.Errorf("FindPodImageDigest() = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_InitContainerMatch(t *testing.T) {
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/sidecar:v1.0",
						ImageID: "docker-pullable://gcr.io/myproject/sidecar@sha256:sidecardigest",
					},
				},
				InitContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/init-tool:v2.0",
						ImageID: "docker-pullable://gcr.io/myproject/init-tool@sha256:initdigest999",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "gcr.io/myproject/init-tool")
	expected := "sha256:initdigest999"
	if got != expected {
		t.Errorf("FindPodImageDigest() for init container = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_NoMatchingPod(t *testing.T) {
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/other-app:v1.0",
						ImageID: "docker-pullable://gcr.io/myproject/other-app@sha256:otherdigest",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "gcr.io/myproject/myapp")
	if got != "" {
		t.Errorf("FindPodImageDigest() = %q, want empty string for non-matching image", got)
	}
}

func TestFindPodImageDigest_EmptyImageID(t *testing.T) {
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/myapp:v1.0",
						ImageID: "", // pod not yet scheduled / image not pulled
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "gcr.io/myproject/myapp")
	if got != "" {
		t.Errorf("FindPodImageDigest() = %q, want empty string when imageID is empty", got)
	}
}

func TestFindPodImageDigest_EmptyPodList(t *testing.T) {
	got := FindPodImageDigest([]core_v1.Pod{}, "gcr.io/myproject/myapp")
	if got != "" {
		t.Errorf("FindPodImageDigest() = %q, want empty string for empty pod list", got)
	}
}

func TestFindPodImageDigest_MatchInSecondPod(t *testing.T) {
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/other-app:v1.0",
						ImageID: "docker-pullable://gcr.io/myproject/other-app@sha256:otherdigest",
					},
				},
			},
		},
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "gcr.io/myproject/myapp:v2.0",
						ImageID: "docker-pullable://gcr.io/myproject/myapp@sha256:myappdigest222",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "gcr.io/myproject/myapp")
	expected := "sha256:myappdigest222"
	if got != expected {
		t.Errorf("FindPodImageDigest() = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_DockerHubShortName(t *testing.T) {
	// Pod running "nginx:1.21", imageID uses full docker.io path.
	// Caller looks for "nginx" (short name).
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "nginx:1.21",
						ImageID: "docker-pullable://docker.io/library/nginx@sha256:nginxdigest123",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "nginx")
	expected := "sha256:nginxdigest123"
	if got != expected {
		t.Errorf("FindPodImageDigest() with Docker Hub short name = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_DockerHubFullVsShortName(t *testing.T) {
	// Pod's Image field uses "docker.io/library/nginx:1.21", caller uses short "nginx".
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "docker.io/library/nginx:1.21",
						ImageID: "docker-pullable://docker.io/library/nginx@sha256:canonicaldigest",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "nginx")
	expected := "sha256:canonicaldigest"
	if got != expected {
		t.Errorf("FindPodImageDigest() full docker.io vs short name = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_TagStrippedForComparison(t *testing.T) {
	// Target imageRepo includes a tag; matching should work regardless.
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "registry.example.com/app:v1.0",
						ImageID: "docker-pullable://registry.example.com/app@sha256:taggeddigest",
					},
				},
			},
		},
	}

	// Target with tag should still match.
	got := FindPodImageDigest(pods, "registry.example.com/app:v1.0")
	expected := "sha256:taggeddigest"
	if got != expected {
		t.Errorf("FindPodImageDigest() with tagged target = %q, want %q", got, expected)
	}
}

func TestFindPodImageDigest_MultipleContainersInPod(t *testing.T) {
	// Pod has multiple containers; correct one should be matched.
	pods := []core_v1.Pod{
		{
			Status: core_v1.PodStatus{
				ContainerStatuses: []core_v1.ContainerStatus{
					{
						Image:   "registry.example.com/sidecar:v1.0",
						ImageID: "docker-pullable://registry.example.com/sidecar@sha256:sidecarA",
					},
					{
						Image:   "registry.example.com/main-app:v3.0",
						ImageID: "docker-pullable://registry.example.com/main-app@sha256:mainappB",
					},
				},
			},
		},
	}

	got := FindPodImageDigest(pods, "registry.example.com/main-app")
	expected := "sha256:mainappB"
	if got != expected {
		t.Errorf("FindPodImageDigest() multi-container pod = %q, want %q", got, expected)
	}
}
