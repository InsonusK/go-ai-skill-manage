Feature: A pipeline runs its transformers in order and records each applied one

 Scenario: Every transformer runs in order and is recorded after the ones applied before
  Given transformers "flat,claude,marker"
  And the catalog was already transformed by "base"
  When I make a pipeline of them
  And I run the pipeline
  Then the pipeline succeeds
  And the transformers ran in order "flat,claude,marker"
  And the catalog has applied "base,flat,claude,marker"

 Scenario: The first failing transformer stops the pipeline and isn't recorded
  Given transformers "flat,claude!,marker"
  When I make a pipeline of them
  And I run the pipeline
  Then the pipeline fails with "transformer claude: broken"
  And the transformers ran in order "flat,claude"
  And the catalog has applied "flat"

 Scenario: A canceled run applies nothing
  Given transformers "flat,marker"
  And the run is canceled
  When I make a pipeline of them
  And I run the pipeline
  Then the pipeline fails with "context canceled"
  And the transformers ran in order ""

 Scenario: A transformer given twice is refused
  Given transformers "flat,marker,flat"
  When I make a pipeline of them
  Then the pipeline fails with "is given twice"
