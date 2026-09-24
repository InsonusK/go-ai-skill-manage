package model

type Link struct {
	// Start and End are the positions of the link in the file content.
	Start, End int
	// Raw - link text as it appears in the file
	// Format - link format, either "markdown" or "wikilink"
	// Text, Path Fragment - link content.
	// - makrdown [label](path#fragment)
	// - wikilink [[path#fragment|label]]
	Raw, Text, Path, Fragment, Format string
	// Image - true if the link is an image link (starts with !)
	Image bool
	// External - true if Path is a URL (http://, https://, mailto:, ftp://,
	// file://) rather than a path inside the repository
	External bool
	// TODO return Target to link
	// Target - the resolved target of the link, if any
	//Target File
}
