package gameengine

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// SimulateWork simulates some CPU-bound work.
func SimulateWork(duration time.Duration) {
	// This is a placeholder. In a real scenario, this would be actual game logic.
	// For now, we'll just sleep to simulate work.
	// A more CPU-intensive task could be:
	// start := time.Now()
	// for time.Since(start) < duration {
	//	 _ = math.Sqrt(rand.Float64())
	// }
	time.Sleep(duration)
}

// FightEnemies simulates the player fighting a number of enemies.
// The function is designed to be called within a benchmark loop (b.N iterations).
func FightEnemies(b *testing.B, numEnemiesPerIteration int, playerLevel int) error {
	if numEnemiesPerIteration <= 0 {
		return fmt.Errorf("numEnemiesPerIteration must be positive, got %d", numEnemiesPerIteration)
	}
	// Simulate complexity based on playerLevel. Higher level = faster processing (less time per enemy).
	// This is an arbitrary calculation for demonstration.
	workPerEnemy := time.Microsecond * 100 / time.Duration(playerLevel)
	
	totalEnemiesInBenchmark := b.N * numEnemiesPerIteration

	if b.N == 1 && numEnemiesPerIteration > 1 { // Special handling if benchmark runs once but for many items
		// This logic helps if a Gherkin step says "fights 100 enemies" and we want that to be one benchmark op
		// but still simulate work for 100 enemies.
		// However, standard benchmark practice is b.N operations.
		// For this placeholder, we'll assume b.N is the number of times the "fight N enemies" action is performed.
	}

	// Simulate fighting each enemy for one benchmark iteration (b.N is 1 for this inner loop)
	for i := 0; i < numEnemiesPerIteration; i++ {
		SimulateWork(workPerEnemy) 
	}
	// In a real game, you might return an error if something went wrong during the fight.
	if rand.Intn(1000) == 7 { // Simulate a rare random error
		return fmt.Errorf("a mystical force interrupted the battle after %d enemies in one iteration", numEnemiesPerIteration)
	}
	b.Logf("Simulated fighting %d enemies (player level %d). Total in benchmark: %d", numEnemiesPerIteration, playerLevel, totalEnemiesInBenchmark)
	return nil
}

// SpawnGuards simulates spawning a number of guards.
// The function is designed to be called within a benchmark loop (b.N iterations).
func SpawnGuards(b *testing.B, numGuardsPerIteration int) error {
	if numGuardsPerIteration <= 0 {
		return fmt.Errorf("numGuardsPerIteration must be positive, got %d", numGuardsPerIteration)
	}
	// Simulate work for spawning each guard.
	workPerGuard := time.Microsecond * 50
	for i := 0; i < numGuardsPerIteration; i++ {
		SimulateWork(workPerGuard)
	}
	// In a real game, this might involve AI initialization, pathfinding calculations, etc.
	if rand.Intn(1000) == 7 { // Simulate a rare random error
		return fmt.Errorf("a magical anomaly prevented %d guards from spawning correctly in one iteration", numGuardsPerIteration)
	}
	b.Logf("Simulated spawning %d guards. Total in benchmark: %d", numGuardsPerIteration, b.N*numGuardsPerIteration)
	return nil
}

// HitWall simulates the player character hitting a wall.
// The function is designed to be called within a benchmark loop (b.N iterations).
func HitWall(b *testing.B, numHitsPerIteration int) error {
	if numHitsPerIteration <= 0 {
		return fmt.Errorf("numHitsPerIteration must be positive, got %d", numHitsPerIteration)
	}
	// Simulate work for processing a wall hit (collision detection, physics response).
	workPerHit := time.Microsecond * 20
	for i := 0; i < numHitsPerIteration; i++ {
		SimulateWork(workPerHit)
	}
	if rand.Intn(1000) == 7 { // Simulate a rare random error
		return fmt.Errorf("the wall phased out of existence during collision for %d hits in one iteration", numHitsPerIteration)
	}
	b.Logf("Simulated hitting a wall %d times. Total in benchmark: %d", numHitsPerIteration, b.N*numHitsPerIteration)
	return nil
}
