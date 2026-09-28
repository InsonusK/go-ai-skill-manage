Feature: Command line: mcp install and uninstall set up the MCP server in Claude Code's .mcp.json

 Background:
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n"}
   """

 Scenario: install writes the server next to the config, run by the name found in PATH
  Given the tool runs as "aism" found in PATH
  When I run arguments "mcp install"
  Then exit code is "0"
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"ai-skills":{"command":"aism","args":["mcp"]}}}
   """
  And stdout contains "Added MCP server ai-skills to "
  And stdout contains "Commit .mcp.json. Claude Code asks to approve the server ai-skills the first time you run claude in this project."

 Scenario: The other servers and keys of .mcp.json stay
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    ".mcp.json":"{\"mcpServers\":{\"other\":{\"command\":\"npx\",\"args\":[\"-y\",\"x\"],\"env\":{\"K\":\"${K:-}\"}}},\"extra\":true}"}
   """
  When I run arguments "mcp install"
  Then exit code is "0"
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"other":{"command":"npx","args":["-y","x"],"env":{"K":"${K:-}"}},"ai-skills":{"command":"aism","args":["mcp"]}},"extra":true}
   """

 Scenario: A config other than ai-skills.yaml is passed to the server; the file lies next to it
  Given CLI project
   """
   {"sub/cfg.yaml":"sources:\n  - path: skills\ntarget: out\n"}
   """
  And the tool runs as "ai-skill-manager" found in PATH
  When I run arguments "mcp install -c sub/cfg.yaml --name skills-feedback"
  Then exit code is "0"
  And project JSON file "sub/.mcp.json" is
   """
   {"mcpServers":{"skills-feedback":{"command":"ai-skill-manager","args":["mcp","-c","cfg.yaml"]}}}
   """

 Scenario: A tool not in PATH is written by its absolute path, with a warning
  Given the tool isn't in PATH and lies at "/home/me/bin/aism"
  When I run arguments "mcp install"
  Then exit code is "0"
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"ai-skills":{"command":"/home/me/bin/aism","args":["mcp"]}}}
   """
  And stdout contains "Warning: /home/me/bin/aism is an absolute path of this machine"

 Scenario: install again changes nothing
  Given I run arguments "mcp install"
  When I run arguments "mcp install"
  Then exit code is "0"
  And stdout contains "MCP server ai-skills is already in "

 Scenario: Another server under the name is kept unless --replace
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    ".mcp.json":"{\"mcpServers\":{\"ai-skills\":{\"command\":\"something-else\"}}}"}
   """
  When I run arguments "mcp install"
  Then exit code is "1"
  And console contains "already has another MCP server ai-skills: {\"command\":\"something-else\"}\nwould write: {\"command\":\"aism\",\"args\":[\"mcp\"]}\nrun with --replace to overwrite it, or pick another --name"
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"ai-skills":{"command":"something-else"}}}
   """
  When I run arguments "mcp install --replace"
  Then exit code is "0"
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"ai-skills":{"command":"aism","args":["mcp"]}}}
   """

 Scenario: A .mcp.json that isn't valid is left for the user to fix
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",".mcp.json":"{\"mcpServers\": [}"}
   """
  When I run arguments "mcp install"
  Then exit code is "1"
  And console contains "isn't a JSON object"
  And project file ".mcp.json" contains "{\"mcpServers\": [}"

 Scenario: uninstall removes only this server
  Given CLI project
   """
   {"ai-skills.yaml":"sources:\n  - path: skills\ntarget: out\n",
    ".mcp.json":"{\"mcpServers\":{\"other\":{\"command\":\"x\"},\"ai-skills\":{\"command\":\"aism\",\"args\":[\"mcp\"]}}}"}
   """
  When I run arguments "mcp uninstall"
  Then exit code is "0"
  And stdout contains "Removed MCP server ai-skills from "
  And project JSON file ".mcp.json" is
   """
   {"mcpServers":{"other":{"command":"x"}}}
   """
  When I run arguments "mcp uninstall"
  Then exit code is "0"
  And stdout contains "No MCP server ai-skills in "

 Scenario: uninstall without .mcp.json has nothing to do
  When I run arguments "mcp uninstall --name skills-feedback"
  Then exit code is "0"
  And stdout contains "No MCP server skills-feedback in "
  And project path ".mcp.json" exists "false"
