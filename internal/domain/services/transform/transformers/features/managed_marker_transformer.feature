Feature: ManagedMarkerTransformer marks every skill folder as written by this tool

 # The target is listed as "{skill dir}/{file path}" -> content.

 Scenario: Every skill gets a marker naming its source, its place there and the transformers before it
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\n","a/guide/docs/x.md":"x","f.skill.md":"---\nname: f\n---\n"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I run the transformers "flat,claude-when-to-use,managed-marker"
  Then the target holds
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n","guide/docs/x.md":"x",
    "guide/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"a/guide\",\n  \"transformers\": [\n    \"flat\",\n    \"claude-when-to-use\"\n  ],\n  \"version\": \"go-2\"\n}\n",
    "f/SKILL.md":"---\nname: f\n---\n",
    "f/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"f.skill.md\",\n  \"transformers\": [\n    \"flat\",\n    \"claude-when-to-use\"\n  ],\n  \"version\": \"go-2\"\n}\n"}
   """

 Scenario: A marker the source skill already has is replaced, not doubled
  Given a repository "repo" holding
   """
   {"a/guide/SKILL.md":"---\nname: guide\n---\n","a/guide/.ai-skills-managed":"{\"source\":\"old\"}"}
   """
  And skills at "." of "repo" are loaded
  When I make the target catalog
  And I run the transformers "flat,managed-marker"
  Then the target holds
   """
   {"guide/SKILL.md":"---\nname: guide\n---\n",
    "guide/.ai-skills-managed":"{\n  \"source\": \"local:repo\",\n  \"skill_path\": \"a/guide\",\n  \"transformers\": [\n    \"flat\"\n  ],\n  \"version\": \"go-2\"\n}\n"}
   """
