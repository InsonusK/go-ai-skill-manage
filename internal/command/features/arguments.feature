Feature: CLI argument contract
 Scenario: All options are parsed without executing integrations
  When I parse argument list
   """
   ["--debug","--profile","--profile-output","cpu.prof","--mem-profile-output=mem.prof","sync","-c","custom.yaml","-t","github","-p","https://github.com/org/repo develop","--subpath","skills","--subpath","docs","--target=out","--dry-run","-f","--keep-orphans","--remove-orphans","--add-relations"]
   """
  Then parsed options are
   """
   {"command":"sync","config":"custom.yaml","type":"github","path":"https://github.com/org/repo develop","subpaths":["skills","docs"],"target":"out","dry":true,"force":true,"orphans":true,"relations":true,"debug":true,"profile":true,"profileOutput":"cpu.prof","memProfileOutput":"mem.prof"}
   """
 Scenario: validate takes the same source options
  When I parse argument list
   """
   ["validate","-c","custom.yaml"]
   """
  Then parsed options are
   """
   {"command":"validate","config":"custom.yaml","type":"","path":"","subpaths":null,"target":"","dry":false,"force":false,"orphans":null,"relations":null,"debug":false,"profile":false,"profileOutput":"ai-skill-manager.prof","memProfileOutput":"ai-skill-manager.mem.prof"}
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
   | ["sync","extra"] | unexpected argument |
   | ["sync","validate"] | unexpected argument |
   | ["sync","--force=invalid"] | requires a boolean |
   | ["sync","--type","wrong"] | unknown source type |
   | ["sync","--config="] | requires a value |

 Scenario: mcp takes the config
  When I parse argument list
   """
   ["mcp","-c","custom.yaml"]
   """
  Then parsed options are
   """
   {"command":"mcp","config":"custom.yaml","type":"","path":"","subpaths":null,"target":"","dry":false,"force":false,"orphans":null,"relations":null,"debug":false,"profile":false,"profileOutput":"ai-skill-manager.prof","memProfileOutput":"ai-skill-manager.mem.prof"}
   """
 Scenario: feedback draft takes the skill, the kind, the title and the body
  When I parse argument list
   """
   ["feedback","draft","--skill","guide","--kind","bug","--title","Broken anchor","--body-file","-","-c","custom.yaml"]
   """
  Then parsed feedback options are
   """
   {"Action":"draft","ID":"","Skill":"guide","Kind":"bug","Title":"Broken anchor","Body":"","BodyFile":"-"}
   """
 Scenario Outline: feedback <action> takes the draft id
  When I parse argument list
   """
   ["feedback","<action>","2026-09-27-guide"]
   """
  Then parsed feedback options are
   """
   {"Action":"<action>","ID":"2026-09-27-guide","Skill":"","Kind":"","Title":"","Body":"","BodyFile":""}
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
   | ["feedback","send","id","--title","t"] | are for feedback draft only |
   | ["sync","--skill","g"] | are for feedback draft only |
 Scenario Outline: mcp takes install or uninstall; --name and --replace belong to them
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | ["mcp","add"] | unknown mcp action "add": use install, uninstall, or none to serve |
   | ["mcp","--name","x"] | --name is for mcp install and mcp uninstall only |
   | ["sync","--name","x"] | --name is for mcp install and mcp uninstall only |
   | ["mcp","uninstall","--replace"] | --replace is for mcp install only |
   | ["mcp","install","-t","local","-p","x"] | mcp works with a config file (-c), not --type/--path |
   | ["mcp","install","extra"] | unexpected argument "extra" |
