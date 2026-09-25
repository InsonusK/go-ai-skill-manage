package entity

// TargetFile is a file of a TargetSkill: a layer over the file below it --
// the source File, or the base catalog's TargetFile -- holding only what a
// transformer changed. A file added in a target (e.g. the managed-state
// marker) has no source file.
type TargetFile struct {
	origin *File
	parent *TargetFile

	path *string
	// content is this layer's content; nil means the layer below's.
	content []byte
}

func newTargetFile(origin *File) *TargetFile { return &TargetFile{origin: origin} }

// clone returns a new layer over f.
func (f *TargetFile) clone() *TargetFile { return &TargetFile{origin: f.origin, parent: f} }

// Origin is the source file this one comes from, nil for a file added in
// a target.
func (f *TargetFile) Origin() *File { return f.origin }

// Path is the file's path from its skill's folder.
func (f *TargetFile) Path() string {
	var fromOrigin string
	if f.origin != nil {
		fromOrigin = f.origin.path
	}
	return layered(f.path, f.parent, (*TargetFile).Path, fromOrigin)
}

// Content returns the file's content, reading the source file only when
// no layer changed it. The returned bytes may be shared with the layers
// below: change them only through SetContent.
func (f *TargetFile) Content() ([]byte, error) {
	switch {
	case f.content != nil:
		return f.content, nil
	case f.parent != nil:
		return f.parent.Content()
	case f.origin != nil:
		return f.origin.Content()
	}
	return []byte{}, nil
}

func (f *TargetFile) SetPath(p string) { f.path = &p }

func (f *TargetFile) SetContent(content []byte) {
	if content == nil {
		content = []byte{}
	}
	f.content = content
}
