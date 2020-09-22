package scheduler

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	indexpkg "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/index"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
)

type Scheduler struct {
	store                       store.Store
	gitserverClient             gitserver.Client
	batchSize                   int
	minimumTimeSinceLastEnqueue time.Duration
	minimumSearchCount          int
	minimumSearchRatio          float64
	minimumPreciseCount         int
	metrics                     SchedulerMetrics
}

var _ goroutine.Handler = &Scheduler{}

func NewScheduler(
	store store.Store,
	gitserverClient gitserver.Client,
	interval time.Duration,
	batchSize int,
	minimumTimeSinceLastEnqueue time.Duration,
	minimumSearchCount int,
	minimumSearchRatio float64,
	minimumPreciseCount int,
	metrics SchedulerMetrics,
) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), interval, &Scheduler{
		store:                       store,
		gitserverClient:             gitserverClient,
		batchSize:                   batchSize,
		minimumTimeSinceLastEnqueue: minimumTimeSinceLastEnqueue,
		minimumSearchCount:          minimumSearchCount,
		minimumSearchRatio:          minimumSearchRatio,
		minimumPreciseCount:         minimumPreciseCount,
		metrics:                     metrics,
	})
}

func (s *Scheduler) Handle(ctx context.Context) error {
	indexableRepositories, err := s.store.IndexableRepositories(ctx, store.IndexableRepositoryQueryOptions{
		Limit:                       s.batchSize,
		MinimumTimeSinceLastEnqueue: s.minimumTimeSinceLastEnqueue,
		MinimumSearchCount:          s.minimumSearchCount,
		MinimumPreciseCount:         s.minimumPreciseCount,
		MinimumSearchRatio:          s.minimumSearchRatio,
	})
	if err != nil {
		return errors.Wrap(err, "store.IndexableRepositories")
	}

	for _, indexableRepository := range indexableRepositories {
		if err := s.queueIndex(ctx, indexableRepository); err != nil {
			if isRepoNotExist(err) {
				continue
			}

			return err
		}
	}

	return nil
}

func (s *Scheduler) HandleError(err error) {
	s.metrics.Errors.Inc()
	log15.Error("Failed to update indexable repositories", "err", err)
}

func (s *Scheduler) queueIndex(ctx context.Context, indexableRepository store.IndexableRepository) (err error) {
	commit, err := s.gitserverClient.Head(ctx, s.store, indexableRepository.RepositoryID)
	if err != nil {
		return errors.Wrap(err, "gitserver.Head")
	}

	isQueued, err := s.store.IsQueued(ctx, indexableRepository.RepositoryID, commit) // TODO - expand this check?
	if err != nil {
		return errors.Wrap(err, "store.IsQueued")
	}
	if isQueued {
		return nil
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return errors.Wrap(err, "store.Transact")
	}
	defer func() {
		err = tx.Done(err)
	}()

	isConfigured, err := s.gitserverClient.FileExists(ctx, s.store, indexableRepository.RepositoryID, commit, "sourcegraph.yaml")
	if err != nil {
		return errors.Wrap(err, "gitserver.FileExists")
	}

	var indexes []store.Index

	if isConfigured {
		content, err := s.gitserverClient.Text(ctx, s.store, indexableRepository.RepositoryID, commit, "sourcegraph.yaml")
		if err != nil {
			return errors.Wrap(err, "gitserver.Text")
		}

		configuration, err := indexpkg.UnmarshalYAML(content)
		if err != nil {
			return errors.Wrap(err, "index.UnmarshalYAML")
		}

		for _, indexJob := range configuration.IndexJobs {
			indexes = append(indexes, store.Index{
				State:           "queued",
				Commit:          commit,
				RepositoryID:    indexableRepository.RepositoryID,
				Root:            indexJob.Root,
				InstallImage:    indexJob.Install.Image,
				InstallCommands: indexJob.Install.Commands,
				Indexer:         indexJob.Index.Indexer,
				Arguments:       indexJob.Index.Arguments,
			})
		}
	} else {
		index := store.Index{
			State:        "queued",
			Commit:       commit,
			RepositoryID: indexableRepository.RepositoryID,
		}

		isGo, err := s.gitserverClient.FileExists(ctx, s.store, indexableRepository.RepositoryID, commit, "go.mod")
		if err != nil {
			return errors.Wrap(err, "gitserver.FileExists")
		}

		if isGo {
			index.Root = "" // TODO
			index.InstallImage = ""
			index.InstallCommands = []string{}
			index.Indexer = "sourcegraph/lsif-go:latest"
			index.Arguments = []string{"lsif-go", "--no-animation"}
		} else {
			index.Root = "" // TODO
			index.InstallImage = "circleci/node:12"
			index.InstallCommands = []string{"yarn", "install", "--frozen-lockfile", "--non-interactive"}
			index.Indexer = "sourcegraph/lsif-node:latest"
			index.Arguments = []string{"lsif-tsc", "-p", "."}
		}

		indexes = append(indexes, index)
	}

	for _, index := range indexes {
		id, err := tx.InsertIndex(ctx, index)
		if err != nil {
			return errors.Wrap(err, "store.QueueIndex")
		}

		log15.Info(
			"Enqueued index",
			"id", id,
			"repository_id", indexableRepository.RepositoryID,
			"commit", commit,
		)

	}

	now := time.Now().UTC()
	update := store.UpdateableIndexableRepository{
		RepositoryID:        indexableRepository.RepositoryID,
		LastIndexEnqueuedAt: &now,
	}

	if err := tx.UpdateIndexableRepository(ctx, update, now); err != nil {
		return errors.Wrap(err, "store.UpdateIndexableRepository")
	}

	return nil
}

func isRepoNotExist(err error) bool {
	for err != nil {
		if vcs.IsRepoNotExist(err) {
			return true
		}

		err = errors.Unwrap(err)
	}

	return false
}
