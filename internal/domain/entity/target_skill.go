package entity

import "fmt"

// TargetSkill is a skill as it is going to be written into a target. It is
// a layer over the skill below it -- the source Skill, or, in a catalog
// cloned for one target, the base catalog's TargetSkill -- and holds only
// what a transformer changed; everything else is read through the layer
// below. Nothing of the source is copied or read until asked for.
//
// The skill's document (frontmatter and body) is not a field of its own:
// it is always its marker file's content, so the two can't disagree.
//
// Пример: базовый слой переложил скил "guide" из "a/guide" в "guide"
// (SetSkillDirPath); слой для .claude/skills поверх него поменял только
// frontmatter (SetDocument) -- SkillDirPath() у него "guide" (из базового
// слоя), Name() -- "guide" (из исходного скила).
type TargetSkill struct {
	origin *Skill
	parent *TargetSkill

	name, mainFilePath, skillDirPath *string
	format                           *SkillFormat

	// mainFile and files are this layer's files, built on first access from
	// the layer below; nil until then.
	mainFile *TargetFile
	files    []*TargetFile
	listed   bool
}

func newTargetSkill(origin *Skill) *TargetSkill { return &TargetSkill{origin: origin} }

// clone returns a new layer over s.
func (s *TargetSkill) clone() *TargetSkill { return &TargetSkill{origin: s.origin, parent: s} }

// Origin is the source skill this one comes from.
func (s *TargetSkill) Origin() *Skill { return s.origin }

func (s *TargetSkill) Name() string {
	return layered(s.name, s.parent, (*TargetSkill).Name, s.origin.Name)
}

// MainFilePath is the path of the skill's marker file, as Skill's: from the
// repository folder before the skill is laid out in a target, from the
// target folder after.
func (s *TargetSkill) MainFilePath() string {
	return layered(s.mainFilePath, s.parent, (*TargetSkill).MainFilePath, s.origin.MainFilePath)
}

// SkillDirPath is the skill's folder, as Skill's ("" for a flat skill).
func (s *TargetSkill) SkillDirPath() string {
	return layered(s.skillDirPath, s.parent, (*TargetSkill).SkillDirPath, s.origin.SkillDirPath)
}

func (s *TargetSkill) Format() SkillFormat {
	return layered(s.format, s.parent, (*TargetSkill).Format, s.origin.Format)
}

// Document parses the skill's marker file as it is now into its
// frontmatter and body. Changing the result changes nothing until
// SetDocument.
func (s *TargetSkill) Document() (SkillDocument, error) {
	content, err := s.MainFile().Content()
	if err != nil {
		return SkillDocument{}, err
	}
	return MakeSkillDocument(content)
}

// SetDocument writes doc into the skill's marker file. The frontmatter is
// encoded anew: its keys come out sorted, YAML comments are lost.
func (s *TargetSkill) SetDocument(doc SkillDocument) error {
	content, err := doc.Encode()
	if err != nil {
		return fmt.Errorf("skill %s: %w", s.Name(), err)
	}
	s.MainFile().SetContent(content)
	return nil
}

func (s *TargetSkill) SetName(name string)          { s.name = &name }
func (s *TargetSkill) SetMainFilePath(p string)     { s.mainFilePath = &p }
func (s *TargetSkill) SetSkillDirPath(p string)     { s.skillDirPath = &p }
func (s *TargetSkill) SetFormat(format SkillFormat) { s.format = &format }

// MainFile is the skill's marker file in this layer.
func (s *TargetSkill) MainFile() *TargetFile {
	if s.mainFile == nil {
		if s.parent != nil {
			s.mainFile = s.parent.MainFile().clone()
		} else {
			s.mainFile = newTargetFile(s.origin.MainFile)
		}
	}
	return s.mainFile
}

// Files lists the skill's other files in this layer, the ones added here
// included. The first call lists the layer below (for the bottom layer:
// the source skill's files, see Skill.FilesByPath); later changes of the
// layer below are not seen.
func (s *TargetSkill) Files() ([]*TargetFile, error) {
	if !s.listed {
		if s.parent != nil {
			below, err := s.parent.Files()
			if err != nil {
				return nil, err
			}
			for _, f := range below {
				s.files = append(s.files, f.clone())
			}
		} else {
			below, err := s.origin.FilesByPath("")
			if err != nil {
				return nil, err
			}
			for _, f := range below {
				s.files = append(s.files, newTargetFile(f))
			}
		}
		s.listed = true
	}
	return s.files, nil
}

// AddFile adds to this layer a file the source skill doesn't have, at
// path p from the skill's folder.
func (s *TargetSkill) AddFile(p string, content []byte) (*TargetFile, error) {
	if _, err := s.Files(); err != nil {
		return nil, err
	}
	f := &TargetFile{path: &p, content: content}
	s.files = append(s.files, f)
	return f, nil
}

// layered returns this layer's value if it has one, otherwise the value of
// the layer below: parent's, or the source's for the bottom layer.
func layered[T any, L any](own *T, parent *L, fromParent func(*L) T, fromOrigin T) T {
	if own != nil {
		return *own
	}
	if parent != nil {
		return fromParent(parent)
	}
	return fromOrigin
}
