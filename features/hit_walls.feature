Feature: Hit Walls
  As a player
  I want to interact with environment boundaries
  So that I can test the performance of collision detection and response

  Scenario: Player character repeatedly hits a wall
    Given the player is moving at high speed
    When the player hits a wall 200 times
    Then the average impact processing time should be less than 5 milliseconds
    And all hit wall operations should complete without error

  Scenario: Player character hits a wall once
    Given the player is moving at low speed
    When the player hits a wall 1 time
    Then the average impact processing time should be less than 5 milliseconds
    And all hit wall operations should complete without error
