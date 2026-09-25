Feature: A target skill catalog is a layer over the source skills

 # A skill is shown as its fields plus every file (the marker file
 # included) as path from the skill's folder -> content.

 Background:
  Given source skills "a/guide/SKILL.md,h.skill/h.skill.md,f.skill.md" in
   """
   {"a/guide/SKILL.md":"---\nname: guide\ndescription: source\n---\n","a/guide/docs/x.md":"x",
    "h.skill/h.skill.md":"---\nname: h\n---\n","h.skill/y.md":"y",
    "f.skill.md":"---\nname: f\n---\n"}
   """

 Scenario: A new catalog reads nothing and shows every skill as its source
  When I remember how many files were opened
  And I make the base target catalog
  Then no more files were opened
  And in the "base" catalog "guide" is
   """
   {"Name":"guide","MainFilePath":"a/guide/SKILL.md","SkillDirPath":"a/guide","Format":"agent-dir","Description":"source",
    "Files":{"SKILL.md":"---\nname: guide\ndescription: source\n---\n","docs/x.md":"x"}}
   """
  And in the "base" catalog "h" is
   """
   {"Name":"h","MainFilePath":"h.skill/h.skill.md","SkillDirPath":"h.skill","Format":"human-dir","Description":null,
    "Files":{"h.skill.md":"---\nname: h\n---\n","y.md":"y"}}
   """
  And in the "base" catalog "f" is
   """
   {"Name":"f","MainFilePath":"f.skill.md","SkillDirPath":"","Format":"flat","Description":null,
    "Files":{"f.skill.md":"---\nname: f\n---\n"}}
   """

 Scenario: Changes stay in the target skill, the source skill is untouched
  When I make the base target catalog
  And in the "base" catalog I set the name of "f" to "flat"
  And in the "base" catalog I set the skill dir path of "f" to "flat"
  And in the "base" catalog I set the main file path of "f" to "flat/SKILL.md"
  And in the "base" catalog I set the format of "f" to "agent-dir"
  And in the "base" catalog I set the path of file "f.skill.md" of "f" to "SKILL.md"
  And in the "base" catalog I set the content of file "SKILL.md" of "f" to "changed"
  And in the "base" catalog I set the description of "f" to "new"
  Then in the "base" catalog "f" is
   """
   {"Name":"flat","MainFilePath":"flat/SKILL.md","SkillDirPath":"flat","Format":"agent-dir","Description":"new",
    "Files":{"SKILL.md":"changed"}}
   """
  And the source skill "f" is
   """
   {"Name":"f","MainFilePath":"f.skill.md","SkillDirPath":"","Format":"flat","Description":null,
    "Files":{"f.skill.md":"---\nname: f\n---\n"}}
   """

 Scenario: The document a target skill returns is a copy until it is set back
  When I make the base target catalog
  And in the "base" catalog I change the description of "guide" to "lost" without setting the document
  Then in the "base" catalog "guide" is
   """
   {"Name":"guide","MainFilePath":"a/guide/SKILL.md","SkillDirPath":"a/guide","Format":"agent-dir","Description":"source",
    "Files":{"SKILL.md":"---\nname: guide\ndescription: source\n---\n","docs/x.md":"x"}}
   """

 Scenario: A clone sees the base as it is, and its own changes stay in it
  When I make the base target catalog
  And in the "base" catalog I set the skill dir path of "guide" to "guide"
  And in the "base" catalog I set the content of file "docs/x.md" of "guide" to "base x"
  And I clone the base target catalog as "claude"
  And I clone the base target catalog as "agents"
  And in the "claude" catalog I set the description of "guide" to "claude"
  And in the "claude" catalog I set the content of file "SKILL.md" of "guide" to "claude main"
  And in the "agents" catalog I set the content of file "docs/x.md" of "guide" to "agents x"
  Then in the "claude" catalog "guide" is
   """
   {"Name":"guide","MainFilePath":"a/guide/SKILL.md","SkillDirPath":"guide","Format":"agent-dir","Description":"claude",
    "Files":{"SKILL.md":"claude main","docs/x.md":"base x"}}
   """
  And in the "agents" catalog "guide" is
   """
   {"Name":"guide","MainFilePath":"a/guide/SKILL.md","SkillDirPath":"guide","Format":"agent-dir","Description":"source",
    "Files":{"SKILL.md":"---\nname: guide\ndescription: source\n---\n","docs/x.md":"agents x"}}
   """
  And in the "base" catalog "guide" is
   """
   {"Name":"guide","MainFilePath":"a/guide/SKILL.md","SkillDirPath":"guide","Format":"agent-dir","Description":"source",
    "Files":{"SKILL.md":"---\nname: guide\ndescription: source\n---\n","docs/x.md":"base x"}}
   """

 Scenario: A file added in a target has no source file and stays in its layer
  When I make the base target catalog
  And in the "base" catalog I add file "base.txt" with "b" to "h"
  And I clone the base target catalog as "claude"
  And in the "claude" catalog I add file ".ai-skills-managed" with "{}" to "h"
  Then in the "claude" catalog "h" is
   """
   {"Name":"h","MainFilePath":"h.skill/h.skill.md","SkillDirPath":"h.skill","Format":"human-dir","Description":null,
    "Files":{"h.skill.md":"---\nname: h\n---\n","y.md":"y","base.txt":"b",".ai-skills-managed":"{}"}}
   """
  And in the "claude" catalog file ".ai-skills-managed" of "h" has no source file
  And in the "base" catalog "h" is
   """
   {"Name":"h","MainFilePath":"h.skill/h.skill.md","SkillDirPath":"h.skill","Format":"human-dir","Description":null,
    "Files":{"h.skill.md":"---\nname: h\n---\n","y.md":"y","base.txt":"b"}}
   """
  And the source skill "h" is
   """
   {"Name":"h","MainFilePath":"h.skill/h.skill.md","SkillDirPath":"h.skill","Format":"human-dir","Description":null,
    "Files":{"h.skill.md":"---\nname: h\n---\n","y.md":"y"}}
   """

 Scenario: A file is changed when its layer or one below changed its content
  When I make the base target catalog
  And in the "base" catalog I set the content of file "docs/x.md" of "guide" to "base x"
  And I clone the base target catalog as "claude"
  And in the "claude" catalog I set the content of file "y.md" of "h" to "claude y"
  Then in the "claude" catalog file "docs/x.md" of "guide" is changed
  And in the "claude" catalog file "y.md" of "h" is changed
  And in the "base" catalog file "y.md" of "h" is unchanged
  And in the "claude" catalog file "SKILL.md" of "guide" is unchanged

 Scenario: A clone starts with the transformers applied to the base and records its own apart
  When I make the base target catalog
  And the "base" catalog records "flat" as applied
  And I clone the base target catalog as "claude"
  And the "claude" catalog records "claude-when-to-use" as applied
  Then the "claude" catalog has applied "flat,claude-when-to-use"
  And the "base" catalog has applied "flat"
