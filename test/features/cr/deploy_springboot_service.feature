@cr
@springboot
Feature: CR: Deploy spring boot service with persistence

  Background:
    Given Namespace is created

  Scenario: CR deploy jbpm-springboot-example service
    Given Kogito Operator is deployed
    
    When Deploy spring boot example service "jbpm-springboot-example"

    Then Kogito application "jbpm-springboot-example" has 1 pods running within 10 minutes
    And HTTP GET request on service "jbpm-springboot-example" with path "orders" is successful within 2 minutes

#####

  # Disabled because of https://issues.redhat.com/browse/KOGITO-948
  @disabled
  @persistence
  Scenario: CR deploy jbpm-springboot-example service with persistence
    Given Kogito Operator is deployed with dependencies
    And Deploy spring boot example service "jbpm-springboot-example" with persistence enabled
    And Kogito application "jbpm-springboot-example" has 1 pods running within 10 minutes
    And HTTP GET request on service "jbpm-springboot-example" with path "orders" is successful within 3 minutes
    And HTTP POST request on service "jbpm-springboot-example" with path "orders" and body:
      """json
      {
        "approver" : "john", 
        "order" : {
          "orderNumber" : "12345", 
          "shipped" : false
        }
      }
      """
    And HTTP GET request on service "jbpm-springboot-example" with path "orders" should return an array of size 1 within 1 minutes
    
    When Scale Kogito application "jbpm-springboot-example" to 0 pods within 2 minutes
    And Scale Kogito application "jbpm-springboot-example" to 1 pods within 2 minutes
    
    Then HTTP GET request on service "jbpm-springboot-example" with path "orders" should return an array of size 1 within 2 minutes