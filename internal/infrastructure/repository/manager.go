package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// SourceManager caches acquired repositories by SourceKey (type+path+tree),
// dispatching to the right underlying provider by SourceSpec.Type, so the
// same source referenced by multiple SourceSpecs (different Subpaths/Tags)
// is fetched only once. It owns the whole acquisition lifecycle: callers
// close every cached Repository via a single Close call instead of tracking
// per-source cleanup funcs themselves.
type SourceManager struct {
	providers map[string]interfaces.SourceProvider
	repos     map[model.SourceKey]*model.Repository
	order     []model.SourceKey
}

// NewSourceManager returns a SourceManager dispatching by SourceSpec.Type to
// the given providers (e.g. "local", "github").
func NewSourceManager(providers map[string]interfaces.SourceProvider) *SourceManager {
	return &SourceManager{providers: providers, repos: map[model.SourceKey]*model.Repository{}}
}

// Acquire returns the cached Repository for s's identity, fetching it via
// the registered provider for s.Type only on the first call for that
// identity.
func (m *SourceManager) Acquire(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, error) {
	key := s.Key()
	if repo, ok := m.repos[key]; ok {
		return repo, nil
	}
	provider, ok := m.providers[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown source type %q", s.Type)
	}
	repo, err := provider.Acquire(ctx, s, options)
	if err != nil {
		return nil, err
	}
	m.repos[key] = repo
	m.order = append(m.order, key)
	return repo, nil
}

// Close closes every acquired Repository, most recently acquired first,
// joining any errors.
func (m *SourceManager) Close() error {
	var err error
	for i := len(m.order) - 1; i >= 0; i-- {
		err = errors.Join(err, m.repos[m.order[i]].Close())
	}
	return err
}
