Feature: Compute changes independently of filesystem writes
 Scenario Outline: Managed state determines actions
  Given planning input with state "<state>" and force "<force>"
  When I plan a sync
  Then plan actions are "<actions>"
  And planned main text is
   """
   ---
   name: a
   ---
   [B](out/b/SKILL.md)
   """
  Examples:
   | state | force | actions |
   | missing | false | a:create,b:create |
   | current | false | a:skip,b:skip |
   | current | true | a:update,b:update |
   | old | false | a:update,b:update |
   | orphan | false | a:create,b:create,old:remove |
   | unmanaged | false | a:create,b:create |
 Scenario: External files are named deterministically
  Given a planning tree
   """
   {"a.skill.md":"---\nname: a\n---\n[x](one/image.png) [y](two/image.png)","one/image.png":"first","two/image.png":"second"}
   """
  When I plan a sync
  Then shared files are
   """
   {"files/image.png":"first","files/image_1.png":"second"}
   """
