Feature: CLI argument contract
 Scenario: All options are parsed without executing integrations
  When I parse argument list
   """
   ["--debug","--profile","--profile-output","cpu.prof","sync","-c","custom.yaml","-t","github","-p","https://github.com/org/repo develop","--subpath","skills","--subpath","docs","--target=out","--dry-run","-f","--keep-orphans","--remove-orphans","--add-relations"]
   """
  Then parsed options are
   """
   {"config":"custom.yaml","type":"github","path":"https://github.com/org/repo develop","subpaths":["skills","docs"],"target":"out","dry":true,"force":true,"orphans":true,"relations":true,"debug":true,"profile":true,"profileOutput":"cpu.prof"}
   """
 Scenario Outline: Malformed arguments fail parsing
  When I parse argument list
   """
   <args>
   """
  Then argument error contains "<error>"
  Examples:
   | args | error |
   | [] | command is required |
   | ["sync","extra"] | unexpected argument |
   | ["sync","--force=invalid"] | requires a boolean |
   | ["sync","--type","wrong"] | unknown source type |
   | ["sync","--config="] | requires a value |
