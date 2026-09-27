Feature: Architecture dependency direction
 Scenario: Product domain only depends on domain packages and allowed standard libraries
  When I inspect domain package imports
  Then forbidden domain imports are
   """
   []
   """
