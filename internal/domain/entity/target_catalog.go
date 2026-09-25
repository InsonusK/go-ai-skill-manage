package entity

// TargetSkillCatalog is the set of skills as they are going to be written
// into a target. It starts as a layer over the loaded source skills (their
// folders as in their repositories), transformers change it, and a
// catalog shared by several targets is cloned once per target: each clone
// is a new layer over it, so what the targets have in common is done once.
//
// Пример: base := NewTargetSkillCatalog(catalog.Skills()); общие
// трансформеры меняют base; claude := base.Clone(), agents := base.Clone()
// -- правки в claude не видны ни в base, ни в agents.
type TargetSkillCatalog struct {
	skills []*TargetSkill
	// applied names the transformers applied to this catalog so far, those
	// applied before it was cloned included.
	applied []string
}

// NewTargetSkillCatalog layers a target skill over each of skills. It reads
// nothing: every value comes from the source skill until changed.
func NewTargetSkillCatalog(skills []*Skill) *TargetSkillCatalog {
	c := &TargetSkillCatalog{}
	for _, s := range skills {
		c.skills = append(c.skills, newTargetSkill(s))
	}
	return c
}

// Skills returns the catalog's skills, in the order of the source skills.
func (c *TargetSkillCatalog) Skills() []*TargetSkill {
	return append([]*TargetSkill(nil), c.skills...)
}

// Clone returns a new catalog layered over c: it sees c as it is, and its
// changes stay in it. Finish changing c before cloning it -- a clone lists
// a skill's files from c on first access and doesn't see files added to c
// afterwards.
func (c *TargetSkillCatalog) Clone() *TargetSkillCatalog {
	out := &TargetSkillCatalog{applied: append([]string(nil), c.applied...)}
	for _, s := range c.skills {
		out.skills = append(out.skills, s.clone())
	}
	return out
}

// Applied names the transformers applied so far, in order.
func (c *TargetSkillCatalog) Applied() []string { return append([]string(nil), c.applied...) }

// AddApplied records that transformer name was applied.
func (c *TargetSkillCatalog) AddApplied(name string) { c.applied = append(c.applied, name) }
