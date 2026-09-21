package interfaces

import "github.com/InsonusK/go-ai-skill-manage/internal/domain/model"

// DocumentCodec converts frontmatter bytes to/from model.Document --
// implemented by infrastructure/document.Codec.
type DocumentCodec interface {
	Decode([]byte) (model.Document, error)
	Encode(model.Document) ([]byte, error)
}
