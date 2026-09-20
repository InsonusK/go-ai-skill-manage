Feature: YAML frontmatter
 Scenario: Round trip preserves properties and body
  Given a document
   """
   ---
   name: sample
   tags: [go, cli]
   custom: Привет
   ---
   # Body
   Some content.
   """
  When I round trip the document
  Then the document result is
   """
   {"name":"sample","tags":["go","cli"],"custom":"Привет","body":"# Body\nSome content."}
   """
 Scenario: Documents without frontmatter remain text
  Given a document
   """
   # Text
   """
  When I round trip the document
  Then the document result is
   """
   {"body":"# Text"}
   """
 Scenario: Malformed frontmatter is rejected
  Given a document
   """
   ---
   tags: [
   ---
   """
  When I round trip the document
  Then the document error contains "frontmatter"
