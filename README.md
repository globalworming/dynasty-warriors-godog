# dynasty-warriors-godog
cucumber and benchmarks, what could go wrong

# everything can be described this way

Actors perceiving something and performing actions is so fundamental, that you can describe a lot of requirements that way. As soon as you have the process codified, you can check some things. 

I am no performance expert in any way, so i naively i would say: when runnning a stress test, I expect some things to hold true regarding throughput or response times or auto scale behavior or... A bunch of things that i do require. And to check that, we build up an environment with certain preconditions. We simulate scenarios, gather evidence, check certain metrics, explore the stage and other scenarios.

Given I set up a specific environment
  And have prepared a performance scenario execution
  And a performance checklist
When I execute the scenario sucessfully
Then I should see that everything on that checklist holds true

# make it relatable

> story: iterative development
> drawback: lack of lower level performance tests

## Running Tests

To run the tests, use the following command from the project root:

go test ./internal/test/benchmarks/... -v

To get cucumber JSON output, set the `BENCHMARK_RESULTS_DIR` environment variable:

BENCHMARK_RESULTS_DIR=/tmp/results go test ./internal/test/benchmarks/... -v
