package benchmarks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	// "strings" // Not used directly in this snippet, but good to keep if TestBenchmark evolves
	"testing"

	"dynasty-warriors-godog/gameengine"

	"github.com/cucumber/godog"
	// "github.com/stretchr/testify/assert" // Using basic error messages for now
)

// GodogsCtxKey is a type for context keys.
type GodogsCtxKey string

const (
	// GodogsCtxBenchmarkResultKey is the context key for storing testing.BenchmarkResult.
	GodogsCtxBenchmarkResultKey GodogsCtxKey = "benchmarkResult"
	// GodogsCtxErrorKey is the context key for storing errors from benchmarked operations.
	GodogsCtxErrorKey GodogsCtxKey = "benchmarkError"
	// GodogsCtxPlayerLevelKey is the context key for player level.
	GodogsCtxPlayerLevelKey GodogsCtxKey = "playerLevel"
	// GodogsCtxAreaKey is the context key for area name.
	GodogsCtxAreaKey GodogsCtxKey = "areaName"
    // GodogsCtxTargetCountKey is the context key for things like number of enemies, guards etc.
    GodogsCtxTargetCountKey GodogsCtxKey = "targetCount"
)

// TestAndBenchCommon provides common logging and naming for tests and benchmarks.
type TestAndBenchCommon struct {
	name string
	logf func(format string, args ...interface{})
	// t    *testing.T // No longer storing *testing.T directly if using godog.TestingT
	godogT godog.TestingT // Store godog.TestingT
}

func NewTestAndBenchCommon(gt godog.TestingT) TestAndBenchCommon {
	return TestAndBenchCommon{
		name:   gt.Name(), // Use Name() from godog.TestingT
		logf:   gt.Logf,   // Use Logf from godog.TestingT
		godogT: gt,
	}
}

func (c TestAndBenchCommon) Name() string {
	return c.name
}

func (c TestAndBenchCommon) Logf(format string, args ...interface{}) {
	c.logf(format, args...)
}

// makeErrorChannel creates a buffered channel for errors.
func makeErrorChannel(bufferSize int) chan error {
	return make(chan error, bufferSize)
}

// trackBenchmarkError sends an error to the channel if it's not nil.
func trackBenchmarkError(b *testing.B, err error, errorChannel chan error) {
	if err != nil {
		select {
		case errorChannel <- err:
		default:
			b.Logf("Error channel full, dropping error: %v", err)
		}
	}
}

// RunAndReport executes a benchmark function, captures its result and errors.
func RunAndReport(
	benchmarkFunc func(b *testing.B) error,
	errorChannel chan error,
	c TestAndBenchCommon,
	ctx context.Context,
) (context.Context, testing.BenchmarkResult, []error) {
	
	var overallErr error
	benchmarkResult := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := benchmarkFunc(b) 
			if err != nil && overallErr == nil {
				overallErr = err
			}
		}
		b.StopTimer()
	})

	close(errorChannel)
	var backgroundErrors []error
	for e := range errorChannel {
		backgroundErrors = append(backgroundErrors, e)
	}

	if overallErr != nil {
		backgroundErrors = append([]error{fmt.Errorf("benchmark function structure error: %w", overallErr)}, backgroundErrors...)
	}
	
	c.Logf("%s Result	%s,%v,%v\n", c.Name(), c.Name(), benchmarkResult.N, benchmarkResult.NsPerOp())
	
	ctx = context.WithValue(ctx, GodogsCtxBenchmarkResultKey, benchmarkResult)
	if len(backgroundErrors) > 0 {
		ctx = context.WithValue(ctx, GodogsCtxErrorKey, backgroundErrors) 
	}
	return ctx, benchmarkResult, backgroundErrors
}

// createResultFile is a helper.
func createResultFile(t *testing.T, path string) *os.File {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("failed to create results directory: %v", err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create result file: %v", err)
	}
	return file
}

// RunCucumberSuite runs the Godog test suite.
func RunCucumberSuite(t *testing.T, featureFilePath string, scenarioInitializer func(ctx *godog.ScenarioContext)) {
	godogOptions := &godog.Options{
		Format:         "pretty",
		Paths:          []string{featureFilePath},
		TestingT:       t,
		Strict:         true,
		StopOnFailure:  true,
		DefaultContext: context.Background(), 
	}

	resultsDir, useCucumberJsonOutput := os.LookupEnv("BENCHMARK_RESULTS_DIR")
	if useCucumberJsonOutput {
		godogOptions.Format = "cucumber"
		// Ensure resultsDir exists
		if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
			if err := os.MkdirAll(resultsDir, 0755); err != nil {
				t.Fatalf("Failed to create BENCHMARK_RESULTS_DIR at %s: %v", resultsDir, err)
			}
		}
		cucumberOutputFile := filepath.Join(resultsDir, "cucumber.json")
		out := createResultFile(t, cucumberOutputFile) // t is passed to createResultFile
		godogOptions.Output = out
		defer out.Close() 
	}

	suite := godog.TestSuite{
		ScenarioInitializer: scenarioInitializer, 
		Options:             godogOptions,
	}

	status := suite.Run()
	if status != 0 {
		t.Errorf("non-zero status (%d) returned from Godog suite run", status)
	}
}

// TestBenchmark is the main entry point for running the Godog features.
func TestBenchmark(t *testing.T) {
	// Path to the features directory, relative to the project root.
	// This assumes tests are run from the project root, or CWD is project root.
	// If CWD is the package dir (internal/test/benchmarks), this needs to be ../../../features
	featuresPath := "../../../features" // Adjusted path

	featuresDir, err := filepath.Abs(featuresPath)
	if err != nil {
		// Fallback for safety, or if Abs fails to give a sensible path under test conditions
		wd, _ := os.Getwd()
		t.Logf("Original featuresPath: %s, CWD: %s", featuresPath, wd)
		// This path might be more robust if tests are always run from project root
		// or if the test binary's CWD is consistently the project root.
		// However, `go test` often changes CWD to the package dir.
		// Let's try to construct it relative to where this test file is.
		// This file is in internal/test/benchmarks. Project root is three levels up.
		featuresDir = filepath.Join(wd, featuresPath) // Re-evaluate based on CWD if Abs was confusing
		featuresDir, err = filepath.Abs(featuresDir) // Try Abs again on the potentially more explicit path
		if err != nil {
			t.Fatalf("Failed to get absolute path for features directory (featuresPath: %s, wd: %s): %v", featuresPath, wd, err)
		}
	}
	t.Logf("Attempting to use features directory: %s", featuresDir)

	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		// For debugging path issues:
		wd, _ := os.Getwd()
		t.Logf("CWD: %s, Resolved featuresDir: %s, Original relative path: %s", wd, featuresDir, featuresPath)
		t.Fatalf("Features directory does not exist at '%s'. Ensure it's created at the project root and the path is correct relative to the test execution context.", featuresDir)
	}
	
	t.Logf("Running Godog suites from resolved path: %s", featuresDir)
	RunCucumberSuite(t, featuresDir, InitializeScenario)
}

// Context helper functions
func getBenchmarkResultFromCtx(ctx context.Context) (testing.BenchmarkResult, error) {
	val := ctx.Value(GodogsCtxBenchmarkResultKey)
	if val == nil {
		return testing.BenchmarkResult{}, fmt.Errorf("benchmarkResult not found in context")
	}
	res, ok := val.(testing.BenchmarkResult)
	if !ok {
		return testing.BenchmarkResult{}, fmt.Errorf("benchmarkResult in context is not of type testing.BenchmarkResult")
	}
	return res, nil
}

func getErrorFromCtx(ctx context.Context) []error {
	val := ctx.Value(GodogsCtxErrorKey)
	if val == nil {
		return nil
	}
	errs, ok := val.([]error)
	if !ok {
		if singleErr, okSingle := val.(error); okSingle {
			return []error{singleErr}
		}
		return []error{fmt.Errorf("errors in context are not of type []error or error: %T", val)}
	}
	return errs
}

func getIntFromCtx(ctx context.Context, key GodogsCtxKey) (int, error) {
    val := ctx.Value(key)
    if val == nil {
        return 0, fmt.Errorf("value for key '%s' not found in context", key)
    }
    i, ok := val.(int)
    if !ok {
        return 0, fmt.Errorf("value for key '%s' in context is not of type int: %T", key, val)
    }
    return i, nil
}

func getStringFromCtx(ctx context.Context, key GodogsCtxKey) (string, error) {
    val := ctx.Value(key)
    if val == nil {
        return "", fmt.Errorf("value for key '%s' not found in context", key)
    }
    s, ok := val.(string)
    if !ok {
        return "", fmt.Errorf("value for key '%s' in context is not of type string: %T", key, val)
    }
    return s, nil
}

// Step Definition Functions

func playerHasLevel(ctx context.Context, level int) (context.Context, error) {
	return context.WithValue(ctx, GodogsCtxPlayerLevelKey, level), nil
}

func playerIsInArea(ctx context.Context, areaName string) (context.Context, error) {
	return context.WithValue(ctx, GodogsCtxAreaKey, areaName), nil
}

func playerIsMovingAtSpeed(ctx context.Context, speed string) (context.Context, error) {
	return context.WithValue(ctx, "playerSpeed", speed), nil
}

// Modified aggregate function
func aggregate(ctx context.Context, benchmarkResult testing.BenchmarkResult, backgroundErrors []error, targetCount int) (context.Context, error) {
	ctx = context.WithValue(ctx, GodogsCtxBenchmarkResultKey, benchmarkResult)
	ctx = context.WithValue(ctx, GodogsCtxTargetCountKey, targetCount)

	if len(backgroundErrors) > 0 {
		ctx = context.WithValue(ctx, GodogsCtxErrorKey, backgroundErrors)
		return ctx, backgroundErrors[0] 
	}
	return ctx, nil
}

// Modified WhenXXX functions
func playerFightsEnemiesMod(ctx context.Context, numEnemies int, gt godog.TestingT) (context.Context, error) {
	if gt == nil { return ctx, fmt.Errorf("godog.TestingT not found in context for playerFightsEnemiesMod") }
	c := NewTestAndBenchCommon(gt)
	playerLevel, err := getIntFromCtx(ctx, GodogsCtxPlayerLevelKey)
	if err != nil {
		return ctx, fmt.Errorf("player level not set: %w", err)
	}
	if numEnemies <= 0 { 
		return ctx, fmt.Errorf("number of enemies must be positive, got %d", numEnemies)
	}

	errorChannel := makeErrorChannel(numEnemies + 10) // Buffer based on count

	updatedCtx, br, bgErrs := RunAndReport(func(b *testing.B) error {
		// The gameengine.FightEnemies is expected to use the passed 'b' for its N iterations.
		gameEngineErr := gameengine.FightEnemies(b, numEnemies, playerLevel)
		// This error is from the *entire* FightEnemies operation.
		// If FightEnemies has errors per sub-op, it should use trackBenchmarkError.
		if gameEngineErr != nil {
		    trackBenchmarkError(b, gameEngineErr, errorChannel) 
			// return gameEngineErr // This indicates a failure of the benchmarked unit itself
		}
		return nil // Assume errors within b.N are handled by trackBenchmarkError
	}, errorChannel, c, ctx)
	return aggregate(updatedCtx, br, bgErrs, numEnemies)
}

func guardsSpawnMod(ctx context.Context, numGuards int, gt godog.TestingT) (context.Context, error) {
	if gt == nil { return ctx, fmt.Errorf("godog.TestingT not found in context for guardsSpawnMod") }
	c := NewTestAndBenchCommon(gt)
	if numGuards <= 0 {
		return ctx, fmt.Errorf("number of guards must be positive, got %d", numGuards)
	}
	errorChannel := makeErrorChannel(numGuards + 10)

	updatedCtx, br, bgErrs := RunAndReport(func(b *testing.B) error {
		gameEngineErr := gameengine.SpawnGuards(b, numGuards)
		if gameEngineErr != nil {
			trackBenchmarkError(b, gameEngineErr, errorChannel)
			// return gameEngineErr
		}
		return nil
	}, errorChannel, c, ctx)
	return aggregate(updatedCtx, br, bgErrs, numGuards)
}

func playerHitsWallMod(ctx context.Context, numHits int, gt godog.TestingT) (context.Context, error) {
	if gt == nil { return ctx, fmt.Errorf("godog.TestingT not found in context for playerHitsWallMod") }
	c := NewTestAndBenchCommon(gt)
	if numHits <= 0 {
		return ctx, fmt.Errorf("number of hits must be positive, got %d", numHits)
	}
	errorChannel := makeErrorChannel(numHits + 10)

	updatedCtx, br, bgErrs := RunAndReport(func(b *testing.B) error {
		gameEngineErr := gameengine.HitWall(b, numHits)
		if gameEngineErr != nil {
			trackBenchmarkError(b, gameEngineErr, errorChannel)
			// return gameEngineErr
		}
		return nil
	}, errorChannel, c, ctx)
	return aggregate(updatedCtx, br, bgErrs, numHits)
}

// Then step definitions
func averageTimePerEnemyDefeatedShouldBeLessThan(ctx context.Context, expectedMsPerItem int) error {
	benchmarkResult, err := getBenchmarkResultFromCtx(ctx)
	if err != nil {
		return err
	}
	targetCount, err := getIntFromCtx(ctx, GodogsCtxTargetCountKey)
	if err != nil {
		return fmt.Errorf("target count (numEnemies) not found in context for calculation: %w", err)
	}
	if targetCount == 0 {
		return fmt.Errorf("target count (numEnemies) is zero, cannot calculate per-item performance")
	}

	observedNsPerOp := benchmarkResult.NsPerOp() 
	observedNsPerSingleItem := observedNsPerOp / int64(targetCount)
	observedMsPerSingleItem := float64(observedNsPerSingleItem) / 1e6

	expectedMaxNsPerItem := int64(expectedMsPerItem) * 1e6

	fmt.Printf("  Benchmark Metric: Average Time Per Enemy Defeated\n")
	fmt.Printf("    Target Count in Operation: %d enemies\n", targetCount)
	fmt.Printf("    Total NsPerOp (for group): %d ns\n", observedNsPerOp)
	fmt.Printf("    Observed NsPerItem: %d ns (%.4f ms)\n", observedNsPerSingleItem, observedMsPerSingleItem)
	fmt.Printf("    Expected Max NsPerItem: %d ns (%d ms)\n", expectedMaxNsPerItem, expectedMsPerItem)
	// fmt.Printf("    Full Benchmark Result: %s\n", benchmarkResult.String())


	if observedNsPerSingleItem > expectedMaxNsPerItem {
		return fmt.Errorf("expected average time per enemy to be less than %d ms (%.0fns), but was %.4f ms (%.0fns)",
			expectedMsPerItem, float64(expectedMaxNsPerItem), observedMsPerSingleItem, float64(observedNsPerSingleItem))
	}
	return nil
}

func playerReactsToAllGuardsWithinSeconds(ctx context.Context, expectedSeconds float64) error {
	benchmarkResult, err := getBenchmarkResultFromCtx(ctx)
	if err != nil {
		return err
	}
	observedNsPerOp := benchmarkResult.NsPerOp()
	observedOpSeconds := float64(observedNsPerOp) / 1e9
	
	targetCount, _ := getIntFromCtx(ctx, GodogsCtxTargetCountKey) // For logging

	fmt.Printf("  Benchmark Metric: Total Guard Spawning & Reaction Time\n")
	fmt.Printf("    Target Count in Operation: %d guards\n", targetCount)
	fmt.Printf("    Observed NsPerOp (total for operation): %d ns (%.4f s)\n", observedNsPerOp, observedOpSeconds)
	fmt.Printf("    Expected Max Operation Time: %.2f s\n", expectedSeconds)
	// fmt.Printf("    Full Benchmark Result: %s\n", benchmarkResult.String())

	if observedOpSeconds > expectedSeconds {
		return fmt.Errorf("expected reaction to all guards (spawning operation) to be within %.2f seconds, but was %.4f seconds",
			expectedSeconds, observedOpSeconds)
	}
	return nil
}

func averageImpactProcessingTimeShouldBeLessThan(ctx context.Context, expectedMsPerItem int) error {
	benchmarkResult, err := getBenchmarkResultFromCtx(ctx)
	if err != nil {
		return err
	}
	targetCount, err := getIntFromCtx(ctx, GodogsCtxTargetCountKey)
	if err != nil {
		return fmt.Errorf("target count (numHits) not found in context for calculation: %w", err)
	}
	if targetCount == 0 {
		return fmt.Errorf("target count (numHits) is zero, cannot calculate per-item performance")
	}

	observedNsPerOp := benchmarkResult.NsPerOp()
	observedNsPerSingleItem := observedNsPerOp / int64(targetCount)
	observedMsPerSingleItem := float64(observedNsPerSingleItem) / 1e6

	expectedMaxNsPerItem := int64(expectedMsPerItem) * 1e6

	fmt.Printf("  Benchmark Metric: Average Impact Processing Time Per Hit\n")
	fmt.Printf("    Target Count in Operation: %d hits\n", targetCount)
	fmt.Printf("    Total NsPerOp (for group): %d ns\n", observedNsPerOp)
	fmt.Printf("    Observed NsPerItem: %d ns (%.4f ms)\n", observedNsPerSingleItem, observedMsPerSingleItem)
	fmt.Printf("    Expected Max NsPerItem: %d ns (%d ms)\n", expectedMaxNsPerItem, expectedMsPerItem)
	// fmt.Printf("    Full Benchmark Result: %s\n", benchmarkResult.String())

	if observedNsPerSingleItem > expectedMaxNsPerItem {
		return fmt.Errorf("expected average impact processing time per hit to be less than %d ms (%.0fns), but was %.4f ms (%.0fns)",
			expectedMsPerItem, float64(expectedMaxNsPerItem), observedMsPerSingleItem, float64(observedNsPerSingleItem))
	}
	return nil
}

func allOperationsShouldCompleteWithoutError(ctx context.Context, operationType string) error {
	errorsInCtx := getErrorFromCtx(ctx) 
	if len(errorsInCtx) > 0 {
		fmt.Printf("Found %d background error(s) for '%s' operations:\n", len(errorsInCtx), operationType)
		for i, e := range errorsInCtx {
			fmt.Printf("    Error %d: %v\n", i+1, e)
		}
		return fmt.Errorf("expected all '%s' operations to complete without error, but found %d errors. First error: %w", operationType, len(errorsInCtx), errorsInCtx[0])
	}
	return nil
}

// InitializeScenario binds step definitions.
func InitializeScenario(scenarioCtx *godog.ScenarioContext) {
	// Given steps
	scenarioCtx.Step(`^the player has a level of (\d+)$`, playerHasLevel)
	scenarioCtx.Step(`^the player is in the '([^']*)' area$`, playerIsInArea)
	scenarioCtx.Step(`^the player is moving at (\w+) speed$`, playerIsMovingAtSpeed)

	// When steps
	scenarioCtx.Step(`^the player fights (\d+) enemies$`, func(sCtx context.Context, count int) (context.Context, error) {
		godogT := godog.T(sCtx) // This retrieves godog.TestingT from the context
		if godogT == nil { return sCtx, fmt.Errorf("godog.T(sCtx) returned nil for 'player fights enemies' step") }
		// Removed duplicated declaration of godogT
		return playerFightsEnemiesMod(sCtx, count, godogT)
	})
	scenarioCtx.Step(`^(\d+) guards spawn around the player$`, func(sCtx context.Context, count int) (context.Context, error) {
		godogT := godog.T(sCtx)
		if godogT == nil { return sCtx, fmt.Errorf("godog.T(sCtx) returned nil for 'guards spawn around' step") }
		return guardsSpawnMod(sCtx, count, godogT)
	})
	scenarioCtx.Step(`^(\d+) guard spawns near the player$`, func(sCtx context.Context, count int) (context.Context, error) {
		godogT := godog.T(sCtx)
		if godogT == nil { return sCtx, fmt.Errorf("godog.T(sCtx) returned nil for 'guard spawns near' step") }
		return guardsSpawnMod(sCtx, count, godogT)
	})
	scenarioCtx.Step(`^the player hits a wall (\d+) time(s)?$`, func(sCtx context.Context, count int) (context.Context, error) {
		godogT := godog.T(sCtx)
		if godogT == nil { return sCtx, fmt.Errorf("godog.T(sCtx) returned nil for 'player hits wall' step") }
		return playerHitsWallMod(sCtx, count, godogT)
	})

	// Then steps
	scenarioCtx.Step(`^the average time per enemy defeated should be less than (\d+) milliseconds$`, averageTimePerEnemyDefeatedShouldBeLessThan)
	scenarioCtx.Step(`^the player reacts to all guards within (\d+) seconds$`, func(sCtx context.Context, seconds int) error {
		return playerReactsToAllGuardsWithinSeconds(sCtx, float64(seconds))
	})
	scenarioCtx.Step(`^the average impact processing time should be less than (\d+) milliseconds$`, averageImpactProcessingTimeShouldBeLessThan)
	scenarioCtx.Step(`^all (fight|guard spawning|hit wall) operations should complete without error$`, allOperationsShouldCompleteWithoutError)
}
