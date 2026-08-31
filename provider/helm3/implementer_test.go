package helm3

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type fakeReleaseQuerier struct {
	releases map[string][]*release.Release
	err      error
	queries  []map[string]string
}

func (q *fakeReleaseQuerier) Query(labels map[string]string) ([]*release.Release, error) {
	q.queries = append(q.queries, labels)
	if q.err != nil {
		return nil, q.err
	}
	results := q.releases[labels["status"]]
	if len(results) == 0 {
		return nil, driver.ErrReleaseNotFound
	}
	return results, nil
}

func helmRelease(namespace, name string, version int, status release.Status) *release.Release {
	return &release.Release{
		Namespace: namespace,
		Name:      name,
		Version:   version,
		Info:      &release.Info{Status: status},
	}
}

func TestListCurrentReleasesFiltersAtStorageLayerAndPreservesLatestSemantics(t *testing.T) {
	querier := &fakeReleaseQuerier{releases: map[string][]*release.Release{
		release.StatusDeployed.String(): {
			helmRelease("apps", "current", 2, release.StatusDeployed),
			helmRelease("apps", "upgrading", 3, release.StatusDeployed),
			helmRelease("other", "current", 4, release.StatusDeployed),
		},
		release.StatusFailed.String(): {
			helmRelease("apps", "failed", 5, release.StatusFailed),
			helmRelease("apps", "current", 1, release.StatusFailed),
		},
		release.StatusPendingUpgrade.String(): {
			helmRelease("apps", "upgrading", 4, release.StatusPendingUpgrade),
		},
	}}

	results, err := listCurrentReleases(querier)
	require.NoError(t, err)
	require.Equal(t, []*release.Release{
		helmRelease("apps", "current", 2, release.StatusDeployed),
		helmRelease("other", "current", 4, release.StatusDeployed),
		helmRelease("apps", "failed", 5, release.StatusFailed),
	}, results)

	require.Len(t, querier.queries, len(currentReleaseStatuses))
	for _, labels := range querier.queries {
		require.Equal(t, "helm", labels["owner"])
		require.NotEqual(t, release.StatusSuperseded.String(), labels["status"])
	}
}

func TestListCurrentReleasesReturnsStorageErrors(t *testing.T) {
	expected := errors.New("storage unavailable")
	_, err := listCurrentReleases(&fakeReleaseQuerier{err: expected})
	require.ErrorIs(t, err, expected)
}

func TestImplementerList(t *testing.T) {
	t.Skip()

	imp := NewHelm3Implementer()
	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(releases) == 0 {
		t.Errorf("why no releases? ")
	}

}
