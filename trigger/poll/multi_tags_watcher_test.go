package poll

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/credentialshelper"
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
			{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

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

type runTestCase struct {
	currentTag  string
	expectedTag string
	bumpPolicy  policy.Policy
}

// Helper function to factorize code
func testRunHelper(testCases []runTestCase, availableTags []string, t *testing.T) {
	var testImages []*types.TrackedImage
	for _, testCase := range testCases {
		reference, _ := image.Parse("foo/bar:" + testCase.currentTag)
		testImages = append(testImages, &types.TrackedImage{
			Image:  reference,
			Policy: testCase.bumpPolicy,
		})
	}
	fp := &fakeProvider{
		images: testImages,
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: availableTags,
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// Compute number of expected events (version bump expected)
	var nbEvents = 0
	for _, testCase := range testCases {
		if testCase.currentTag != testCase.expectedTag {
			nbEvents++
		}
	}
	// checking whether new job was submitted
	if len(fp.submitted) != nbEvents {
		tags := []string{}
		for _, s := range fp.submitted {
			tags = append(tags, s.Repository.Tag)
		}
		t.Errorf("expected "+strconv.Itoa(nbEvents)+" events, got: %d [%s]", len(fp.submitted), strings.Join(tags, ", "))
	} else {
		for i, testCase := range testCases {
			submitted := fp.submitted[i]

			if submitted.Repository.Name != "index.docker.io/foo/bar" {
				t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
			}

			if submitted.Repository.Tag != testCase.expectedTag {
				t.Errorf("expected event repository tag "+testCase.expectedTag+", but got: %s", submitted.Repository.Tag)
			}
		}
	}
}

func TestWatchAllTagsJobWith2pointSemver(t *testing.T) {
	availableTags := []string{"1.3", "2.5", "2.7", "3.8"}
	testRunHelper([]runTestCase{{"1.3", "3.8", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"2.5", "3.8", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"2.5", "2.7", policy.NewSemverPolicy(policy.SemverPolicyTypeMinor, false)}}, availableTags, t)
}

func TestWatchAllTagsJobWithSemver(t *testing.T) {
	availableTags := []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"}
	testCases := []runTestCase{{"1.1.0", "1.5.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)}}
	testRunHelper(testCases, availableTags, t)
}

func TestWatchAllTagsPrerelease(t *testing.T) {
	availableTags := []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"}
	testCases := []runTestCase{{"1.2.0-dev", "1.3.0-dev", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)}}
	testRunHelper(testCases, availableTags, t)
}

// Full Semver, including pre-releases
func TestWatchAllTagsFullSemver(t *testing.T) {
	availableTags := []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"}
	testCases := []runTestCase{{"1.2.0-dev", "1.8.0-alpha", policy.NewSemverPolicy(policy.SemverPolicyTypeMinor, false)}}
	testRunHelper(testCases, availableTags, t)

	// Test simulating linuxserver tagging strategy
	availableTags = []string{"v0.1.2-ls1", "v0.1.2-ls2", "v0.1.3-ls1", "v0.1.3-ls2", "v0.2.0-ls2", "v0.2.0-ls3"}
	testCases = []runTestCase{{"v0.1.0-ls1", "v0.2.0-ls3", policy.NewSemverPolicy(policy.SemverPolicyTypeMinor, false)}}
	testRunHelper(testCases, availableTags, t)

}

func TestWatchAllTagsHiddenMinorWith2PointVersions(t *testing.T) {
	availableTags := []string{"1.3", "1.5", "2.0", "1.2.1"}
	testRunHelper([]runTestCase{{"1.2", "1.2.1", policy.NewSemverPolicy(policy.SemverPolicyTypePatch, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"1.2", "1.5", policy.NewSemverPolicy(policy.SemverPolicyTypeMinor, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"1.2", "2.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, false)}}, availableTags, t)
}

// Bug #490: new major version "hiding" minor one
func TestWatchAllTagsHiddenMinor(t *testing.T) {
	availableTags := []string{"1.3.0", "1.5.0", "2.0.0", "1.2.1"}
	testRunHelper([]runTestCase{{"1.2.0", "1.2.1", policy.NewSemverPolicy(policy.SemverPolicyTypePatch, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"1.2.0", "1.5.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMinor, false)}}, availableTags, t)
	testRunHelper([]runTestCase{{"1.2.0", "2.0.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, false)}}, availableTags, t)
}

func TestWatchAllTagsMixed(t *testing.T) {
	availableTags := []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"}
	testCases := []runTestCase{
		{"1.0.0", "1.5.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)},
		{"1.2.0-dev", "1.3.0-dev", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)}}
	testRunHelper(testCases, availableTags, t)
}

func TestWatchAllTagsMixedPolicyAll(t *testing.T) {
	availableTags := []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"}
	testCases := []runTestCase{
		{"1.0.0", "1.5.0", policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)},
		{"1.6.0-alpha", "1.8.0-alpha", policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true)}}
	testRunHelper(testCases, availableTags, t)
}

func Test_semverSort(t *testing.T) {
	tags := []string{"1.3.0", "aa1.0.0", "zzz", "1.3.0-dev", "1.5.0", "2.0.0-alpha", "1.3.0-dev1", "1.8.0-alpha", "1.3.1-dev", "123", "1.2.3-rc.1.2+meta"}
	expectedTags := []string{"2.0.0-alpha", "1.8.0-alpha", "1.5.0", "1.3.1-dev", "1.3.0", "1.3.0-dev1", "1.3.0-dev", "1.2.3-rc.1.2+meta"}
	expectedVersions := make([]*semver.Version, len(expectedTags))
	for i, tag := range expectedTags {
		v, _ := semver.NewVersion(tag)
		expectedVersions[i] = v
	}
	sortedTags := semverSort(tags)

	if !reflect.DeepEqual(sortedTags, expectedVersions) {
		t.Errorf("Invalid sorted tags; expected: %s; got: %s", expectedVersions, sortedTags)
	}
}

type testingCredsHelper struct {
	err         error              // err to return
	credentials *types.Credentials // creds to return
}

func (h *testingCredsHelper) IsEnabled() bool {
	return true
}

func (h *testingCredsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	return h.credentials, h.err
}

func TestWatchMultipleTagsWithCredentialsHelper(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	t.Run("TestError", func(t *testing.T) {
		mockHelper := &testingCredsHelper{
			err: errors.New("doesn't work"),
		}
		credentialshelper.RegisterCredentialsHelper("mock", mockHelper)
		defer credentialshelper.UnregisterCredentialsHelper("mock")

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
		assert.Equal(t, "", frc.opts.Username)
		assert.Equal(t, "", frc.opts.Password)
	})

	t.Run("TestOK", func(t *testing.T) {
		mockHelper := &testingCredsHelper{
			err: nil,
			credentials: &types.Credentials{
				Username: "user",
				Password: "pass",
			},
		}
		credentialshelper.RegisterCredentialsHelper("mock", mockHelper)
		defer credentialshelper.UnregisterCredentialsHelper("mock")

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
		assert.Equal(t, "user", frc.opts.Username)
		assert.Equal(t, "pass", frc.opts.Password)
	})

}
