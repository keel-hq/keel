package poll

import (
	"github.com/rusenask/cron"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
)

type Watcher interface {
	Watch(repoName ) error
	Remove(repository )
	List() ([]types.Repository, error)
}

type RepositoryWatcher struct {
	providers provider.Providers

	cron *cron.Cron
}
func ()