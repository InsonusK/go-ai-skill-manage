Feature: Select skills using tag expressions
 Scenario Outline: Operator and hierarchical matching
  Given tags "<tags>"
  When I evaluate "<expression>"
  Then the match is "<match>" and error is "<error>"
  Examples:
   | tags | expression | match | error |
   | go,cli | go & cli | true | |
   | go | !cli & go | true | |
   | cli | go & cli | false | |
   | cli | go \| cli & !old | true | |
   | old | !(go \| cli) | true | |
   | b/c | a/b/c | true | |
   | a/b/c | b | false | |
   | lang | lang/* | true | |
   | lang/go | lang/* | true | |
   | lang/go/deep | lang/* | false | |
   | lang/go/deep | lang/** | true | |
   | lang/go/beta | lang/*/beta | true | |
   | lang/go/deep/beta | lang/*/beta | false | |
   | lang/beta | lang/**/beta | true | |
   | x/beta | **/beta | true | |
   | x | /* | false | |
   | x/y | /* | true | |
   | x | /** | true | |
   | | * | false | |
   | x | ** | true | |
   | go | (go | false | invalid tag expression |
   | go | go cli | false | invalid tag expression |
   | go | go & | false | invalid tag expression |
   | go | | false | invalid tag expression |
