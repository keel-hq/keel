package policy

import (
	"reflect"
	"testing"
)

func TestForcePolicyShouldUpdate(t *testing.T) {
	tests := []struct {
		name       string
		matchTag   bool
		current    string
		new        string
		wantUpdate bool
	}{
		{
			name:       "force, same tag is an update (mutable tag re-push)",
			matchTag:   false,
			current:    "latest",
			new:        "latest",
			wantUpdate: true,
		},
		{
			name:       "force, non semver tags have no ordering",
			matchTag:   false,
			current:    "latest",
			new:        "main",
			wantUpdate: true,
		},
		{
			name:       "force, non semver current, semver new",
			matchTag:   false,
			current:    "latest",
			new:        "1.2.3",
			wantUpdate: true,
		},
		{
			name:       "force, semver new is higher",
			matchTag:   false,
			current:    "7.9.1-1-ubi8",
			new:        "8.0.0",
			wantUpdate: true,
		},
		{
			name:       "force, semver new is lower, issue 823 cp-kafka 7.9.1-1-ubi8 -> 3.0.0",
			matchTag:   false,
			current:    "7.9.1-1-ubi8",
			new:        "3.0.0",
			wantUpdate: false,
		},
		{
			name:       "force, semver new is lower, issue 823 8.1.0 -> 8.0.0",
			matchTag:   false,
			current:    "8.1.0",
			new:        "8.0.0",
			wantUpdate: false,
		},
		{
			name:       "force, same semver tag re-push is an update",
			matchTag:   false,
			current:    "8.1.0",
			new:        "8.1.0",
			wantUpdate: true,
		},
		{
			name:       "force, patch bump",
			matchTag:   false,
			current:    "1.4.5",
			new:        "1.4.6",
			wantUpdate: true,
		},
		{
			name:       "force, same semver with different prerelease is lower",
			matchTag:   false,
			current:    "1.2.3-4",
			new:        "1.2.3-3",
			wantUpdate: false,
		},
		{
			name:       "force, same semver with different prerelease is higher",
			matchTag:   false,
			current:    "1.2.3-3",
			new:        "1.2.3-4",
			wantUpdate: true,
		},
		{
			name:       "force, prerelease to release is higher",
			matchTag:   false,
			current:    "7.9.1-1-ubi8",
			new:        "7.9.1",
			wantUpdate: true,
		},
		{
			name:       "force tag match, same tag is an update",
			matchTag:   true,
			current:    "latest",
			new:        "latest",
			wantUpdate: true,
		},
		{
			name:       "force tag match, different tag is not an update",
			matchTag:   true,
			current:    "latest",
			new:        "main",
			wantUpdate: false,
		},
		{
			name:       "force tag match, different higher semver tag is not an update",
			matchTag:   true,
			current:    "1.2.3",
			new:        "1.2.4",
			wantUpdate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := NewForcePolicy(tt.matchTag)
			got, err := fp.ShouldUpdate(tt.current, tt.new)
			if err != nil {
				t.Errorf("ShouldUpdate() unexpected error = %v", err)
				return
			}
			if got != tt.wantUpdate {
				t.Errorf("ShouldUpdate(%q, %q) = %v, want %v", tt.current, tt.new, got, tt.wantUpdate)
			}
		})
	}
}

func TestForcePolicyFilter(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "mixed tags, semver descending first, registry order otherwise (issue 823)",
			tags: []string{"3.0.0", "3.0.1", "7.9.1-1-ubi8", "8.0.0", "latest", "8.3.1"},
			want: []string{"8.3.1", "8.0.0", "7.9.1-1-ubi8", "3.0.1", "3.0.0", "latest"},
		},
		{
			name: "non semver tags keep their relative order",
			tags: []string{"latest", "main", "beta"},
			want: []string{"latest", "main", "beta"},
		},
		{
			name: "semver tags only, descending",
			tags: []string{"1.3", "2.5", "2.7", "3.8"},
			want: []string{"3.8", "2.7", "2.5", "1.3"},
		},
		{
			name: "v prefix is preserved",
			tags: []string{"v1.0.0", "v1.2.0", "v1.1.0"},
			want: []string{"v1.2.0", "v1.1.0", "v1.0.0"},
		},
		{
			name: "empty tags",
			tags: []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := NewForcePolicy(false)
			got := fp.Filter(tt.tags)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}
