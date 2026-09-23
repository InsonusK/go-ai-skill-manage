package links

import (
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Searcher adapts Extract to entity.LinkSearcher -- the port entity.File
// calls through (via entity.SetDefaultLinkSearcher) to lazily discover its
// own links on first File.Links() call.
type Searcher struct{}

var _ entity.LinkSearcher = Searcher{}

// SearchLinks never fails on its own; Extract is a pure regex scan with no
// way to error. The error return exists only to satisfy entity.LinkSearcher.
func (Searcher) SearchLinks(content string) ([]*model.Link, error) {
	extracted := Extract(content)
	out := make([]*model.Link, len(extracted))
	for i := range extracted {
		out[i] = &extracted[i]
	}
	return out, nil
}
