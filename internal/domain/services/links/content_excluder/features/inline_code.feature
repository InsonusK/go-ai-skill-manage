Feature: InlineCode hides inline code spans

 Scenario: Single backticks are hidden with their content
  When I search inline code in
   """
   a `b` c
   """
  Then the excluded spans are
   """
   [[2,5]]
   """

 Scenario: A double-backtick span may hold a single backtick
  When I search inline code in
   """
   ``a`b`` c
   """
  Then the excluded spans are
   """
   [[0,7]]
   """

 Scenario: A longer backtick run does not close a shorter one
  When I search inline code in
   """
   `a`` b`
   """
  Then the excluded spans are
   """
   [[0,7]]
   """

 Scenario: An unmatched backtick hides nothing
  When I search inline code in
   """
   a `b c
   """
  Then the excluded spans are
   """
   []
   """

 Scenario: Inline code may span lines
  When I search inline code in
   """
   `a
   b`
   """
  Then the excluded spans are
   """
   [[0,5]]
   """

 Scenario: Backticks inside a fenced block are not inline code
  When I search inline code in
   """
   ```
   `a` [x](y)
   ```
   `b`
   """
  Then the excluded spans are
   """
   [[19,22]]
   """

 Scenario: Mask blanks every inline code span and keeps newlines and offsets
  When I mask inline code in
   """
   a `b` c
   `d`
   """
  Then the masked content is
   """
   "a     c\n   "
   """
  And the excluded spans are
   """
   [[2,5],[8,11]]
   """
