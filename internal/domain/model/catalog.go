package model

// Catalog holds the skills selected from all sources for one synchronization
// run, plus the name-collision policy applied while it grows. Catalog is a
// passive value: growing it (name-collision resolution) and querying skill
// ownership of a path are discovery's responsibility (see the discovery
// package's Add/Owner functions), so model keeps no outward dependency.
type Catalog struct {
	Skills   []*Skill
	Conflict string
}
