Feature: CPU and heap profiling lifecycle
 Scenario: The CPU and the heap profiles are written and closed
  When I record a CPU profile
  Then the CPU profile is a readable gzip stream
  And the heap profile is a readable gzip stream
 Scenario: Invalid profile destination reports an error
  When I start profiling in a missing directory
  Then profiling error contains "no such file"
