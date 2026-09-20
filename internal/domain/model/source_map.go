package model

// SourceMap indexes every repository acquired for one synchronization run by
// its source key (Repository.ID). SyncService.Run builds it once in the
// acquire loop, right after each source's Acquire succeeds, and it stays
// read-only for the rest of the run.
type SourceMap struct {
	repos map[string]*Repository
	order []string
}

// NewSourceMap returns an empty SourceMap.
func NewSourceMap() *SourceMap { return &SourceMap{repos: map[string]*Repository{}} }

// Put registers repo under its own ID, preserving first-seen order.
func (m *SourceMap) Put(repo *Repository) {
	if _, ok := m.repos[repo.ID]; !ok {
		m.order = append(m.order, repo.ID)
	}
	m.repos[repo.ID] = repo
}

// Get returns the repository registered under id, or nil if none was.
func (m *SourceMap) Get(id string) *Repository { return m.repos[id] }

// Repositories returns every registered repository in acquisition order.
func (m *SourceMap) Repositories() []*Repository {
	out := make([]*Repository, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.repos[id])
	}
	return out
}
