package poll

import (
	"github.com/rusenask/cron"
	"github.com/rusenask/keel/image"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
)

type Watcher interface {
	Watch(image string) error
	Unwatch(image string) error
	List() ([]types.Repository, error)
}

type RepositoryWatcher struct {
	providers provider.Providers

	cron *cron.Cron
}
