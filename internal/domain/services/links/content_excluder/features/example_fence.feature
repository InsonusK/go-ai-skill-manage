Feature: ExampleFence hides fenced code blocks tagged "example"

 Scenario: A backtick example block is hidden with its fence lines
  When I search example fences in
   """
   a
   ```example
   [s](b)
   ```
   c
   """
  Then the excluded spans are
   """
   [[2,24]]
   """

 Scenario: A tilde example block is hidden
  When I search example fences in
   """
   a
   ~~~example
   [s](b)
   ~~~
   c
   """
  Then the excluded spans are
   """
   [[2,24]]
   """

 Scenario: Untagged and other-tagged blocks are not hidden
  When I search example fences in
   """
   ```
   [a](b)
   ```
   ```go
   [c](d)
   ```
   """
  Then the excluded spans are
   """
   []
   """

 Scenario: An unclosed example block is hidden up to the end of content
  When I search example fences in
   """
   a
   ```example
   [s](b)
   """
  Then the excluded spans are
   """
   [[2,19]]
   """

 Scenario: A fence indented by four spaces is not a fence
  When I search example fences in
   """
       ```example
   [s](b)
       ```
   """
  Then the excluded spans are
   """
   []
   """

 Scenario: A shorter fence line does not close a longer opening fence
  When I search example fences in
   """
   ````example
   ```
   [s](b)
   ````
   c
   """
  Then the excluded spans are
   """
   [[0,28]]
   """

 Scenario: A fence line with an info string does not close the block
  When I search example fences in
   """
   ```example
   ```text
   [s](b)
   ```
   c
   """
  Then the excluded spans are
   """
   [[0,30]]
   """

 Scenario: Mask blanks the example block and keeps newlines and offsets
  When I mask example fences in
   """
   a
   ```example
   [s](b)
   ```
   c
   """
  Then the masked content is
   """
   "a\n          \n      \n   \nc"
   """
  And the excluded spans are
   """
   [[2,24]]
   """
