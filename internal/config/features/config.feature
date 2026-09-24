Feature: Effective synchronization configuration
 Scenario: Defaults for a local source
  Given configuration
   """
   sources:
     - path: skills
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[{"type":"local","path":"/project/skills","subpaths":[],"tags":[],"exclude":[]}],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["link-adapter"]}],"tempDir":"","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":["examples"]}
   """
  And the config log has INFO "settings.validation.exclude_from_checks"
 Scenario: Named targets and global adapters
  Given configuration
   """
   sources: []
   target:
     for_each:
       adapters: [link-adapter]
     default: {}
     claude:
       adapters: [claude-property-adapter, link-adapter]
   settings:
     dry_run: true
     remove_orphans: false
     add_relations: true
     on_conflict: last_wins
     validation:
       exclude_from_checks: []
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["link-adapter"]},{"name":"claude","path":"/project/.claude/skills","adapters":["link-adapter","claude-property-adapter"]}],"tempDir":"","dry":true,"orphans":false,"relations":true,"conflict":"last_wins","exclude":[]}
   """
 Scenario Outline: Temporary directory resolves from the configuration directory
  Given configuration
   """
   sources: []
   settings:
     temp_dir: <temp_dir>
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["link-adapter"]}],"tempDir":"<resolved>","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":["examples"]}
   """
  Examples:
   | temp_dir | resolved |
   | .tmp | /project/.tmp |
   | /var/tmp/aism | /var/tmp/aism |
 Scenario: Unknown adapters fail
  Given configuration
   """
   target:
     custom:
       path: out
       adapters: [unknown]
   """
  When I resolve configuration
  Then the config error contains "unknown adapter"
 Scenario: Invalid settings fail
  Given configuration
   """
   settings:
     dry_run: wrong
   """
  When I resolve configuration
  Then the config error contains "dry_run must be a boolean"
 Scenario: JSON configuration is accepted
  Given configuration
   """
   {"sources":[],"target":"out"}
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[],"targets":[{"name":"default","path":"/project/out","adapters":["link-adapter"]}],"tempDir":"","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":["examples"]}
   """

 Scenario Outline: Configuration rejects invalid public values
  Given configuration
   """
   <config>
   """
  When I resolve configuration
  Then the config error contains "<error>"
  Examples:
   | config | error |
   | [] | config |
   | {sources: wrong} | sources must be a list |
   | {sources: [wrong]} | source must be a mapping |
   | {sources: [{}]} | source path is required |
   | {sources: [{type: unknown, path: input}]} | unknown source type |
   | {sources: [{path: input, tags: [1]}]} | tags entries must be strings |
   | {sources: [{path: input, tags: 42}]} | tags must be a string or list |
   | {settings: {on_conflict: invalid}} | unknown on_conflict |
   | {settings: {temp_dir: 42}} | temp_dir must be a string |
   | {target: {custom: {}}} | requires path |
   | {target: 42} | target must be a mapping |
   | {target: out, settings: {target: other}} | target cannot be defined both |
   | {target: {for_each: wrong}} | for_each must be a mapping |
   | {settings: {validation: wrong}} | validation must be a mapping |
   | {target: {one: {path: out}, two: {path: out/nested}}} | target paths overlap |
   | {target: {one: {path: /}, two: {path: /project/out}}} | target paths overlap |
 Scenario: Legacy settings target and source syntax are preserved
  Given configuration
   """
   sources:
     - type: flat
       path: input
       tags: [!deprecated ""]
       skip_folder: demo
       subpath: part
   settings:
     target:
       for_each:
         adapters: [claude-property-adapter]
     validation:
       rules:
         link:
           skip_folder: null
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[{"type":"local","path":"/project/input","subpaths":["part"],"tags":["!deprecated"],"exclude":["demo"]}],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["claude-property-adapter"]}],"tempDir":"","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":[]}
   """
  And the config log has WARN "key=skip_folder use=exclude_from_checks"
  And the config log has WARN "key=settings.validation.rules.link.skip_folder use=settings.validation.exclude_from_checks"
  And the config log has no INFO

 Scenario Outline: Folders excluded from checks come from settings and from each source
  Given configuration
   """
   <config>
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[{"type":"local","path":"/project/in","subpaths":[],"tags":[],"exclude":<source>}],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["link-adapter"]}],"tempDir":"","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":<global>}
   """
  And the config log has no WARN
  Examples:
   | config | source | global |
   | {sources: [{path: in, exclude_from_checks: [demo]}], settings: {validation: {exclude_from_checks: [examples, docs]}}} | ["demo"] | ["examples","docs"] |
   | {sources: [{path: in, exclude_from_checks: []}], settings: {validation: {exclude_from_checks: []}}} | [] | [] |
   | {sources: [{path: in, exclude_from_checks: null}], settings: {validation: {exclude_from_checks: null}}} | [] | [] |
   | {sources: [{path: in, exclude_from_checks: demo}], settings: {validation: {exclude_from_checks: docs}}} | ["demo"] | ["docs"] |

 Scenario: An empty exclusion setting is not replaced by the default
  Given configuration
   """
   settings:
     validation:
       exclude_from_checks:
   """
  When I resolve configuration
  Then the configuration is
   """
   {"sources":[],"targets":[{"name":"default","path":"/project/.agents/skills","adapters":["link-adapter"]}],"tempDir":"","dry":false,"orphans":true,"relations":false,"conflict":"error","exclude":[]}
   """
  And the config log has no INFO

 Scenario Outline: A new exclusion key and its deprecated name can't be set together
  Given configuration
   """
   <config>
   """
  When I resolve configuration
  Then the config error contains "<error>"
  Examples:
   | config | error |
   | {sources: [{path: in, exclude_from_checks: [a], skip_folder: [b]}]} | exclude_from_checks cannot be defined both with deprecated skip_folder |
   | {settings: {validation: {exclude_from_checks: [a], rules: {link: {skip_folder: [b]}}}}} | settings.validation.exclude_from_checks cannot be defined both |
   | {sources: [{path: in, exclude_from_checks: [1]}]} | exclude_from_checks entries must be strings |
