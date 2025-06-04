Feature: Spawn Guards
  As a game master
  I want to spawn guards
  So that I can test the performance of entity spawning and player reaction AI

  Scenario: Multiple guards spawn and player reacts
    Given the player is in the 'market_square' area
    When 50 guards spawn around the player
    Then the player reacts to all guards within 5 seconds
    And all guard spawning operations should complete without error

  Scenario: A single guard spawns
    Given the player is in the 'castle_gate' area
    When 1 guard spawns near the player
    Then the player reacts to all guards within 1 second
    And all guard spawning operations should complete without error
