// Package discovery recognizes skill layouts, resolves names and tags, and
// selects the skills a source contributes. A Detector stays lean: it reads
// only a skill's own main file and records nested files' paths; LoadFiles
// reads nested-file content later, for skills that survive selection.
package discovery
