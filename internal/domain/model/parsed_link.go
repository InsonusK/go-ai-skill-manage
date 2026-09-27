package model

// ParsedLink is what a link parser reads from one link in a file's text --
// the minimal data entity.MakeLink turns into a Link.
type ParsedLink struct {
	// Start and End are the [Start, End) byte range of the link in the
	// file content.
	Start, End int
	// Text, Path, Fragment -- the link's parts:
	//   - markdown [text](path#fragment)
	//   - wikilink [[path#fragment|text]]
	// Path is exactly as written; Fragment keeps its leading "#".
	Text, Path, Fragment string
	// Format is the link syntax, "markdown" or "wikilink".
	Format string
	// Image is true for the "!" image form of the link.
	Image bool
}
