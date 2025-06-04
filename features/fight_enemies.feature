Feature: Fight Enemies
  As a player
  I want to fight enemies
  So that I can test the performance of combat mechanics

  Scenario: Player fights a specific number of enemies
    Given the player has a level of 10
    When the player fights 100 enemies
    Then the average time per enemy defeated should be less than 10 milliseconds
    And all fight operations should complete without error

  Scenario: Player fights many enemies
    Given the player has a level of 50
    When the player fights 1000 enemies
    Then the average time per enemy defeated should be less than 15 milliseconds
    And all fight operations should complete without error
