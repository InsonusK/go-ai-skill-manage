Feature: CLI argument contract
 Global options go before or after the command; a command takes only its
 own options (and, for feedback and mcp, those of the action).

 Scenario: All sync options are parsed without executing integrations
  When I parse argument list
   """
   ["--debug","--color","always","--profile","--profile-output","cpu.prof","--mem-profile-output=mem.prof","sync","-c","custom.yaml","-t","github","-p","https://github.com/org/repo develop","--subpath","skills","--subpath","docs","--target=out","--dry-run","-f","--keep-orphans","--remove-orphans","--add-relations"]
   """
  Then parsed global options are
   """
   {"Help":false,"Version":false,"Debug":true,"Profile":true,"Color":"always","ProfileOutput":"cpu.prof","MemProfileOutput":"mem.prof"}
   """
  And parsed command "sync" options are
   """
   {"Source":{"Config":"custom.yaml","Type":"github","Path":"https://github.com/org/repo develop","Subpaths":["skills","docs"]},
    "Override":{"Target":"out","DryRun":true,"RemoveOrphans":true,"AddRelations":true},"Force":true}
   """

 Scenario: Global options go after the command too
  When I parse argument list
   """
   ["sync","--debug","--color=never"]
   """
  Then parsed global options are
   """
   {"Help":false,"Version":false,"Debug":true,"Profile":false,"Color":"never","ProfileOutput":"ai-skill-manager.prof","MemProfileOutput":"ai-skill-manager.mem.prof"}
   """

 Scenario: --keep-orphans turns removal off; --add-relations=false keeps relations off
  When I parse argument list
   """
   ["sync","--keep-orphans","--add-relations=false"]
   """
  Then parsed command "sync" options are
   """
   {"Source":{"Config":"","Type":"","Path":"","Subpaths":null},
    "Override":{"Target":"","DryRun":false,"RemoveOrphans":false,"AddRelations":false},"Force":false}
   """

 Scenario: validate takes the source options and --add-relations
  When I parse argument list
   """
   ["validate","-c","custom.yaml","--add-relations"]
   """
  Then parsed command "validate" options are
   """
   {"Source":{"Config":"custom.yaml","Type":"","Path":"","Subpaths":null},
    "Override":{"Target":"","DryRun":false,"RemoveOrphans":null,"AddRelations":true}}
   """

 Scenario Outline: Malformed arguments fail parsing
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | [] | command is required: sync, validate, feedback, mcp |
   | ["check"] | unknown command "check": use sync, validate, feedback, mcp |
   | ["sync","extra"] | unexpected argument "extra" |
   | ["sync","validate"] | unexpected argument "validate" |
   | ["sync","--force=invalid"] | --force requires a boolean |
   | ["sync","--type","wrong"] | unknown source type "wrong" |
   | ["sync","--color","bright"] | --color: must be auto, always or never |
   | ["--color","bright"] | --color: must be auto, always or never |
   | ["sync","--color"] | --color requires a value |
   | ["sync","--config="] | --config requires a value |
   | ["sync","--bad"] | unknown flag "--bad" for sync: see aism sync --help |
   | ["-c","x.yaml","sync"] | unknown flag "-c": a command's flags go after the command |

 Scenario Outline: A command takes only its own flags
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | ["validate","--dry-run"] | unknown flag "--dry-run" for validate |
   | ["validate","--target","out"] | unknown flag "--target" for validate |
   | ["validate","-f"] | unknown flag "-f" for validate |
   | ["sync","--skill","g"] | unknown flag "--skill" for sync |
   | ["sync","--name","x"] | unknown flag "--name" for sync |
   | ["feedback","show","id","-t","local"] | unknown flag "-t" for feedback show |
   | ["feedback","send","id","--title","t"] | unknown flag "--title" for feedback send |
   | ["mcp","--name","x"] | unknown flag "--name" for mcp |
   | ["mcp","uninstall","--replace"] | unknown flag "--replace" for mcp uninstall |
   | ["mcp","install","-t","local","-p","x"] | unknown flag "-t" for mcp install |

 Scenario: mcp takes the config
  When I parse argument list
   """
   ["mcp","-c","custom.yaml"]
   """
  Then parsed command "mcp" options are
   """
   {"Action":"","ServerName":"ai-skills","Replace":false,"Source":{"Config":"custom.yaml","Type":"","Path":"","Subpaths":null}}
   """

 Scenario: mcp install takes the server's name and --replace
  When I parse argument list
   """
   ["mcp","install","--name","skills-feedback","--replace"]
   """
  Then parsed command "mcp" options are
   """
   {"Action":"install","ServerName":"skills-feedback","Replace":true,"Source":{"Config":"","Type":"","Path":"","Subpaths":null}}
   """

 Scenario: feedback draft takes the skill, the kind, the title and the body
  When I parse argument list
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","Broken anchor","--body-file","-","-c","custom.yaml"]
   """
  Then parsed command "feedback" options are
   """
   {"Action":"draft","ID":"","Source":{"Config":"custom.yaml","Type":"","Path":"","Subpaths":null},"Skill":"guide","Kind":"bug","Title":"Broken anchor","Body":"","BodyFile":"-"}
   """

 Scenario Outline: feedback <action> takes the draft id
  When I parse argument list
   """
   ["feedback","<action>","2026-09-27-guide"]
   """
  Then parsed command "feedback" options are
   """
   {"Action":"<action>","ID":"2026-09-27-guide","Source":{"Config":"","Type":"","Path":"","Subpaths":null},"Skill":"","Kind":"","Title":"","Body":"","BodyFile":""}
   """
  Examples:
   | action  |
   | show    |
   | send    |
   | decline |

 Scenario Outline: Malformed feedback arguments fail parsing
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | ["feedback"] | feedback needs an action: draft, show, send, decline |
   | ["feedback","post"] | unknown feedback action "post" |
   | ["feedback","send"] | feedback send needs the draft id |
   | ["feedback","show","a","b"] | unexpected argument "b" |
   | ["feedback","draft","x"] | unexpected argument "x" |
   | ["feedback","draft","--skill","g","--title","t","--body","b"] | feedback draft needs --skill, --kind and --title |
   | ["feedback","draft","--skill","g","--kind","bug","--title","t"] | needs either --body or --body-file |
   | ["feedback","draft","--skill","g","--kind","bug","--title","t","--body","b","--body-file","f"] | needs either --body or --body-file |

 Scenario Outline: Malformed mcp arguments fail parsing
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | ["mcp","add"] | unknown mcp action "add": use install, uninstall, or none to serve |
   | ["mcp","install","extra"] | unexpected argument "extra" |

 Scenario Outline: With --help a command's missing arguments don't matter
  When I parse argument list
   """
   <args>
   """
  Then parsed global options are
   """
   {"Help":true,"Version":false,"Debug":false,"Profile":false,"Color":"auto","ProfileOutput":"ai-skill-manager.prof","MemProfileOutput":"ai-skill-manager.mem.prof"}
   """
  Examples:
   | args |
   | ["--help"] |
   | ["feedback","--help"] |
   | ["feedback","draft","-h"] |
   | ["mcp","install","--help"] |
