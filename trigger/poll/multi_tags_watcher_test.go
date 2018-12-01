package poll

import (
	"reflect"
	"strings"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

func TestWatchMultipleTagsWithSemver(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewSemverPolicy(policy.SemverPolicyTypeAll),
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
		tagsToReturn:   []string{"5.0.0"},
	}

	watcher := NewRepositoryWatcher(providers, frc)

	tracked := []*types.TrackedImage{
		mustParse("gcr.io/v2-namespace/hello-world:1.1.1", "@every 10m"),
	}

	err := watcher.Watch(tracked...)
	if err != nil {
		t.Errorf("failed to watch: %s", err)
	}

	if len(watcher.watched) != 1 {
		t.Errorf("expected to find watching 1 entries, found: %d", len(watcher.watched))
	}
}

func TestWatchAllTagsJobWithSemver(t *testing.T) {

	reference, _ := image.Parse("foo/bar:1.1.0")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:  reference,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor),
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 1 {
		tags := []string{}
		for _, s := range fp.submitted {
			tags = append(tags, s.Repository.Tag)
		}
		t.Errorf("expected 1 events, got: %d [%s]", len(fp.submitted), strings.Join(tags, ", "))
	}

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.5.0" {
		t.Errorf("expected event repository tag 1.5.0, but got: %s", submitted.Repository.Tag)
	}

}

func TestWatchAllTagsPrerelease(t *testing.T) {

	referenceB, _ := image.Parse("foo/bar:1.2.0-dev")

	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:  referenceB,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor),
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 1 {
		tags := []string{}
		for _, s := range fp.submitted {
			tags = append(tags, s.Repository.Tag)
		}
		t.Errorf("expected 1 events, got: %d [%s]", len(fp.submitted), strings.Join(tags, ", "))
	}

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.3.0-dev" {
		t.Errorf("expected event repository tag 1.3.0-dev, but got: %s", submitted.Repository.Tag)
	}
}

func TestWatchAllTagsMixed(t *testing.T) {

	referenceA, _ := image.Parse("foo/bar:1.0.0")
	referenceB, _ := image.Parse("foo/bar:1.2.0-dev")

	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:  referenceB,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor),
			},
			&types.TrackedImage{
				Image:  referenceA,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor),
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 2 {
		tags := []string{}
		for _, s := range fp.submitted {
			tags = append(tags, s.Repository.Tag)
		}
		t.Errorf("expected 1 events, got: %d [%s]", len(fp.submitted), strings.Join(tags, ", "))
	}

	submitted := fp.submitted[0]
	submitted2 := fp.submitted[1]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.3.0-dev" {
		t.Errorf("expected event repository tag 1.3.0-dev, but got: %s", submitted.Repository.Tag)
	}

	if submitted2.Repository.Tag != "1.5.0" {
		t.Errorf("expected event repository tag 1.5.0, but got: %s", submitted2.Repository.Tag)
	}
}

func TestWatchAllTagsMixedPolicyAll(t *testing.T) {

	referenceA, _ := image.Parse("foo/bar:1.0.0")
	referenceB, _ := image.Parse("foo/bar:1.6.0-alpha")

	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:  referenceB,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeAll),
			},
			&types.TrackedImage{
				Image:  referenceA,
				Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor),
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 2 {
		tags := []string{}
		for _, s := range fp.submitted {
			tags = append(tags, s.Repository.Tag)
		}
		t.Errorf("expected 1 events, got: %d [%s]", len(fp.submitted), strings.Join(tags, ", "))
	}

	submitted := fp.submitted[0]
	submitted2 := fp.submitted[1]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.8.0-alpha" {
		t.Errorf("expected event repository tag 1.8.0-alpha, but got: %s", submitted.Repository.Tag)
	}

	if submitted2.Repository.Tag != "1.5.0" {
		t.Errorf("expected event repository tag 1.5.0, but got: %s", submitted2.Repository.Tag)
	}
}

func Test_collapse(t *testing.T) {
	type args struct {
		tags []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "single version",
			args: args{tags: []string{"1.0.0"}},
			want: []string{"1.0.0"},
		},
		{
			name: "multi",
			args: args{tags: []string{"1.0.0", "1.4.0"}},
			want: []string{"1.4.0"},
		},
		{
			name: "prerelease",
			args: args{tags: []string{"1.0.0-dev", "1.4.0-dev"}},
			want: []string{"1.4.0-dev"},
		},
		{
			name: "prerelease multi",
			args: args{tags: []string{"1.3.0-bb", "1.0.0-dev", "1.4.0-dev"}},
			want: []string{"1.3.0-bb", "1.4.0-dev"},
		},
		{
			name: "prerelease multi, mixed",
			args: args{tags: []string{"1.2.0", "1.3.0-bb", "1.0.0-dev", "1.4.0-dev"}},
			want: []string{"1.2.0", "1.3.0-bb", "1.4.0-dev"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collapse(tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collapse() = %v, want %v", got, tt.want)
			}
		})
	}
}
