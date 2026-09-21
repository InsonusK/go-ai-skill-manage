package sourcing

import (
	"context"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Manager caches acquired repositories by SourceKey (type+path+tree),
// dispatching to the right underlying provider by SourceSpec.Type, so the
// same source referenced by multiple SourceSpecs (different Subpaths/Tags)
// is fetched only once. It owns the whole acquisition lifecycle: callers
// close every cached Repository via a single Close call instead of tracking
// per-source cleanup funcs themselves. Manager only ever calls the
// interfaces.SourceProvider port it was given -- the actual filesystem/Git/
// HTTP work happens in whichever infrastructure adapters are injected.
// Manager itself satisfies interfaces.SourceCache.
type Manager struct {
	providers map[string]interfaces.SourceProvider
	repos     map[model.SourceKey]*model.Repository
	order     []model.SourceKey
}

// NewManager returns a Manager dispatching by SourceSpec.Type to the given
// providers (e.g. "local", "github").
func NewManager(providers map[string]interfaces.SourceProvider) *Manager {
	return &Manager{providers: providers, repos: map[model.SourceKey]*model.Repository{}}
}

// GetOrAdd returns the cached Repository for s's identity, fetching it via
// the registered provider for s.Type only on the first call for that
// identity.
func (m *Manager) GetOrAdd(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, error) {
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

// Lookup returns the already-acquired Repository with the given ID, without
// fetching anything new -- for callers that only know a Repository.ID (e.g.
// from a Link.Target) and need the *Repository it came from, not a fresh
// acquisition by SourceKey. A linear scan is fine here: realistic source
// counts are small, and IDs are unique by construction.
func (m *Manager) Lookup(ctx context.Context, id string) (*model.Repository, bool) {
	for _, repo := range m.repos {
		if repo.ID == id {
			return repo, true
		}
	}
	return nil, false
}

// Close closes every acquired Repository, most recently acquired first,
// joining any errors. It always attempts every close, even when ctx is
// already canceled -- shutdown cleanup must not be skipped because the
// signal that triggered it also canceled the context.
func (m *Manager) Close(ctx context.Context) error {
	var err error
	for i := len(m.order) - 1; i >= 0; i-- {
		err = errors.Join(err, m.repos[m.order[i]].Close())
	}
	return err
}
