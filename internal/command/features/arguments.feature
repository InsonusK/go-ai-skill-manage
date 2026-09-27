Feature: CLI argument contract
 Scenario: All options are parsed without executing integrations
  When I parse argument list
   """
   ["--debug","--profile","--profile-output","cpu.prof","sync","-c","custom.yaml","-t","github","-p","https://github.com/org/repo develop","--subpath","skills","--subpath","docs","--target=out","--dry-run","-f","--keep-orphans","--remove-orphans","--add-relations"]
   """
  Then parsed options are
   """
   {"command":"sync","config":"custom.yaml","type":"github","path":"https://github.com/org/repo develop","subpaths":["skills","docs"],"target":"out","dry":true,"force":true,"orphans":true,"relations":true,"debug":true,"profile":true,"profileOutput":"cpu.prof"}
   """
 Scenario: validate takes the same source options
  When I parse argument list
   """
   ["validate","-c","custom.yaml"]
   """
  Then parsed options are
   """
   {"command":"validate","config":"custom.yaml","type":"","path":"","subpaths":null,"target":"","dry":false,"force":false,"orphans":null,"relations":null,"debug":false,"profile":false,"profileOutput":"ai-skill-manager.prof"}
   """
 Scenario Outline: Malformed arguments fail parsing
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | [] | command is required: sync, validate |
   | ["sync","extra"] | unexpected argument |
   | ["sync","validate"] | unexpected argument |
   | ["sync","--force=invalid"] | requires a boolean |
   | ["sync","--type","wrong"] | unknown source type |
   | ["sync","--config="] | requires a value |
