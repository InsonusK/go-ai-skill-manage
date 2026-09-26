package sourcing

import (
	"context"
	"errors"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Manager caches acquired repositories by SourceKey (type+path+tree),
// dispatching to the right underlying provider by key.Type, so the same
// source referenced by multiple SourceSpecs (different Subpaths/Tags,
// which don't affect identity) is fetched only once. It owns the whole
// acquisition lifecycle: callers
// close every cached Repository via a single Close call instead of tracking
// per-source cleanup funcs themselves. Manager only ever calls the
// interfaces.SourceProvider port it was given -- the actual filesystem/Git/
// HTTP work happens in whichever infrastructure adapters are injected.
// Manager itself satisfies interfaces.SourceCache.
type Manager struct {
	providers map[string]interfaces.SourceProvider
	repos     map[model.SourceKey]*entity.Repository
	order     []model.SourceKey
	tempDir   string
}

// NewManager returns a Manager dispatching by SourceKey.Type to the given
// providers (e.g. "local", "github"), using tempDir for every acquisition's
// AcquisitionOptions.TempDir -- callers don't thread that through every
// GetOrAdd call themselves.
func NewManager(providers map[string]interfaces.SourceProvider, tempDir string) *Manager {
	return &Manager{providers: providers, repos: map[model.SourceKey]*entity.Repository{}, tempDir: tempDir}
}

// Get returns the cached Repository for key, fetching it via the
// registered provider for key.Type only on the first call for that
// identity.
func (m *Manager) Get(ctx context.Context, key model.SourceKey) (*entity.Repository, error) {
	if repo, ok := m.repos[key]; ok {
		return repo, nil
	}
	provider, ok := m.providers[key.Type]
	if !ok {
		return nil, fmt.Errorf("unknown source type %q", key.Type)
	}
	repo, err := provider.Acquire(ctx, key, model.AcquisitionOptions{TempDir: m.tempDir})
	if err != nil {
		return nil, err
	}
	m.repos[key] = repo
	m.order = append(m.order, key)
	return repo, nil
}

// LookupKey returns the already-acquired Repository of key, without
// fetching anything new.
func (m *Manager) LookupKey(ctx context.Context, key model.SourceKey) (*entity.Repository, bool) {
	repo, ok := m.repos[key]
	return repo, ok
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
