Feature: CPU profiling lifecycle
 Scenario: CPU profile is written and closed
  When I record a CPU profile
  Then the profile is a readable gzip stream
 Scenario: Invalid profile destination reports an error
  When I start profiling in a missing directory
  Then profiling error contains "no such file"
