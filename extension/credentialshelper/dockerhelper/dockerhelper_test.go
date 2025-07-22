package dockerhelper

import (
	"fmt"
	"testing"

	"github.com/jellydator/ttlcache/v3"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

type testExecutor struct {
	err    error
	output []byte
}

func (e *testExecutor) Run(path, input string) ([]byte, error) {
	return e.output, e.err
}

func TestGetCredentials(t *testing.T) {
	image, err := image.Parse("docker.io/keel-hq/keel:latest")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testCases := []struct {
		name           string
		input          *types.TrackedImage
		executorErr    error
		executorOutput []byte
		expected       *types.Credentials
	}{
		{
			name: "error",
			input: &types.TrackedImage{
				Image: image,
			},
			executorErr: fmt.Errorf("error"),
			expected:    nil,
		},
		{
			name: "success",
			input: &types.TrackedImage{
				Image: image,
			},
			executorOutput: []byte(`{"Username":"testUser","Secret":"testPW"}`),
			expected: &types.Credentials{
				Username: "testUser",
				Password: "testPW",
			},
		},
		{
			name: "invalid json",
			input: &types.TrackedImage{
				Image: image,
			},
			executorOutput: []byte(`invalid json`),
			expected:       nil,
			executorErr:    fmt.Errorf("invalid json"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			helper := &CredentialsHelper{
				executor: &testExecutor{
					err:    tc.executorErr,
					output: tc.executorOutput,
				},
				cache:   ttlcache.New(ttlcache.WithTTL[string, *types.Credentials](defaultCacheTTL)),
				enabled: true,
			}
			creds, err := helper.GetCredentials(tc.input)
			if err != nil {
				if tc.executorErr == nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if creds.Username != tc.expected.Username {
					t.Errorf("expected username %s, got %s", tc.expected.Username, creds.Username)
				}
				if creds.Password != tc.expected.Password {
					t.Errorf("expected password %s, got %s", tc.expected.Password, creds.Password)
				}
			}
		})
	}
}

func TestGetCredentialsFromCache(t *testing.T) {
	image, err := image.Parse("docker.io/keel-hq/keel:latest")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	helper := &CredentialsHelper{
		executor: &testExecutor{
			err:    nil,
			output: []byte(`{"Username":"testUser","Secret":"testPW"}`),
		},
		cache:   ttlcache.New(ttlcache.WithTTL[string, *types.Credentials](defaultCacheTTL)),
		enabled: true,
	}
	creds, err := helper.GetCredentials(&types.TrackedImage{
		Image: image,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds.Password != "testPW" || creds.Username != "testUser" {
		t.Errorf("unexpected credentials: %v", creds)
	}
	helper.executor.(*testExecutor).err = fmt.Errorf("error")
	creds, err = helper.GetCredentials(&types.TrackedImage{
		Image: image,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds.Password != "testPW" || creds.Username != "testUser" {
		t.Errorf("unexpected credentials: %v", creds)
	}
}

func TestGetCredentialsExpiredCache(t *testing.T) {
	image, err := image.Parse("docker.io/keel-hq/keel:latest")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	helper := &CredentialsHelper{
		executor: &testExecutor{
			err:    nil,
			output: []byte(`{"Username":"testUser","Secret":"testPW"}`),
		},
		cache:   ttlcache.New(ttlcache.WithTTL[string, *types.Credentials](defaultCacheTTL)),
		enabled: true,
	}
	creds, err := helper.GetCredentials(&types.TrackedImage{
		Image: image,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds.Password != "testPW" || creds.Username != "testUser" {
		t.Errorf("unexpected credentials: %v", creds)
	}
	helper.cache.DeleteAll()
	creds, err = helper.GetCredentials(&types.TrackedImage{
		Image: image,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds.Password != "testPW" || creds.Username != "testUser" {
		t.Errorf("unexpected credentials: %v", creds)
	}
}
